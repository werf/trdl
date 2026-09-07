package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/docker/cli/cli/config"
	bkclient "github.com/moby/buildkit/client"
	// Blank imports register the docker-container:// and kube-pod:// transports
	// in the BuildKit client; unix:// and tcp:// are handled natively.
	_ "github.com/moby/buildkit/client/connhelper/dockercontainer"
	_ "github.com/moby/buildkit/client/connhelper/kubepod"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/moby/buildkit/session/secrets/secretsprovider"
	"github.com/moby/buildkit/session/upload/uploadprovider"
	"github.com/moby/buildkit/util/progress/progressui"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"

	"github.com/werf/logboek"
	"github.com/werf/trdl/server/pkg/mac_signing"
	"github.com/werf/trdl/server/pkg/secrets"
)

const buildkitdAddressEnv = "TRDL_BUILDKITD_ADDRESS"

var supportedBuildkitdAddressSchemes = []string{"unix", "tcp", "docker-container", "kube-pod"}

func resolveBuildkitdAddress(ctx context.Context, configuredAddress string) (string, error) {
	address := strings.TrimSpace(configuredAddress)
	if address == "" {
		address = strings.TrimSpace(os.Getenv(buildkitdAddressEnv))
	}
	if err := ValidateBuildkitdAddress(ctx, address); err != nil {
		return "", err
	}
	return address, nil
}

func ValidateBuildkitdAddress(ctx context.Context, address string) error {
	address = strings.TrimSpace(address)
	if address == "" {
		return nil
	}

	scheme, endpoint, found := strings.Cut(address, "://")
	if !found || !lo.Contains(supportedBuildkitdAddressSchemes, scheme) {
		return fmt.Errorf("unsupported buildkitd address %q: expected scheme to be one of %s", address, strings.Join(supportedBuildkitdAddressSchemes, ", "))
	}
	if strings.TrimSpace(endpoint) == "" {
		return fmt.Errorf("invalid buildkitd address %q: no endpoint after the %s:// scheme", address, scheme)
	}

	return nil
}

func buildkitSecretsData(buildSecrets []secrets.Secret, macSigningCredentials *mac_signing.Credentials) map[string][]byte {
	data := map[string][]byte{}
	for _, s := range buildSecrets {
		data[s.Id] = s.Data
	}

	if macSigningCredentials != nil {
		identityName := mac_signing.MacSigningCertificateName
		data[identityName+"_cert"] = []byte(macSigningCredentials.Certificate)
		data[identityName+"_password"] = []byte(macSigningCredentials.Password)
		data[identityName+"_notary_key_id"] = []byte(macSigningCredentials.NotaryKeyID)
		data[identityName+"_notary_key"] = []byte(macSigningCredentials.NotaryKey)
		data[identityName+"_notary_issuer"] = []byte(macSigningCredentials.NotaryIssuer)
	}

	return data
}

// The docker CLI path gets registry credentials from the docker config through
// buildx; the BuildKit client has to attach the same provider itself, otherwise
// image-resolve-mode=pull can only reach public registries. The returned
// function removes the token seed directory once the build is over.
func buildkitSessionAttachables(ctx context.Context, contextUploader *uploadprovider.Uploader, secretsData map[string][]byte) ([]session.Attachable, func()) {
	dockerConfigDirMu.Lock()
	defer dockerConfigDirMu.Unlock()

	dockerConfig := config.LoadDefaultConfigFile(logboek.Context(ctx).OutStream())
	seedDir, restoreDockerConfigDir := useWritableDockerConfigDirForTokenSeeds(ctx)
	defer restoreDockerConfigDir()

	attachables := []session.Attachable{
		contextUploader,
		secretsprovider.FromMap(secretsData),
		authprovider.NewDockerAuthProvider(authprovider.DockerAuthProviderConfig{
			AuthConfigProvider: authprovider.LoadAuthConfig(dockerConfig),
		}),
	}

	return attachables, func() {
		if seedDir == "" {
			return
		}
		if err := os.RemoveAll(seedDir); err != nil {
			logboek.Context(ctx).Default().LogLn(fmt.Sprintf("Unable to remove the BuildKit registry token seed dir %q: %s", seedDir, err))
		}
	}
}

// dockerConfigDirMu serializes the config.Dir() redirect below; docker/cli
// keeps that directory in an unsynchronised package global.
var dockerConfigDirMu sync.Mutex

// The auth provider never hands registry credentials to buildkitd: on a bearer
// challenge the daemon asks the client for an ed25519 public key and the client
// fetches tokens itself. The key is derived from the daemon's salt and a
// per-host client seed, 16 random bytes generated locally and never sent
// anywhere, which the provider persists as <config.Dir()>/.token_seed. Nothing
// in that file is a token or a credential. Persisting it runs
// os.MkdirAll(config.Dir()) on every bearer challenge, anonymous pulls
// included, and that is the one step upstream does not tolerate on a read-only
// filesystem, so a process without a writable home (a builtin backend on a
// read-only root) fails every pull. NewDockerAuthProvider captures config.Dir()
// when it is constructed, which is why the redirect only has to hold across
// that call. The directory is private to one build, so mounts sharing a
// process share nothing through it, and lives in memory-backed /dev/shm where
// that exists so the seed never reaches a disk.
func useWritableDockerConfigDirForTokenSeeds(ctx context.Context) (string, func()) {
	dir := config.Dir()
	if err := os.MkdirAll(dir, 0o755); err == nil {
		return "", func() {}
	}

	var seedDir string
	var err error
	for _, base := range []string{"/dev/shm", os.TempDir()} {
		if seedDir, err = os.MkdirTemp(base, "trdl-docker-config-"); err == nil {
			break
		}
	}
	if err != nil {
		msg := fmt.Sprintf("Docker config dir %q is not writable and no private dir for BuildKit registry token seeds could be created: %s", dir, err)
		logboek.Context(ctx).Default().LogLn(msg)
		return "", func() {}
	}

	msg := fmt.Sprintf("Docker config dir %q is not writable, keeping BuildKit registry token seeds in %q for this build", dir, seedDir)
	logboek.Context(ctx).Default().LogLn(msg)
	config.SetDir(seedDir)
	return seedDir, func() { config.SetDir(dir) }
}

func buildkitFrontendAttrs(dockerfilePath, contextStreamURL string) map[string]string {
	return map[string]string{
		"filename":           dockerfilePath,
		"context":            contextStreamURL,
		"no-cache":           "",
		"image-resolve-mode": "pull",
	}
}

func buildWithBuildkit(ctx context.Context, address, dockerfilePath string, secretsData map[string][]byte, contextReader io.ReadCloser, tarWriter io.WriteCloser, logger Logger) error {
	bkClient, err := bkclient.New(ctx, address)
	if err != nil {
		return fmt.Errorf("unable to connect to buildkitd at %q: %w", address, err)
	}
	defer bkClient.Close()

	return buildWithBuildkitClient(ctx, bkClient, dockerfilePath, secretsData, contextReader, tarWriter, logger)
}

func buildWithBuildkitClient(ctx context.Context, bkClient *bkclient.Client, dockerfilePath string, secretsData map[string][]byte, contextReader io.ReadCloser, tarWriter io.WriteCloser, logger Logger) error {
	contextUploader := uploadprovider.New()
	attachables, removeTokenSeeds := buildkitSessionAttachables(ctx, contextUploader, secretsData)
	defer removeTokenSeeds()
	solveOpt := bkclient.SolveOpt{
		Frontend:      "dockerfile.v0",
		FrontendAttrs: buildkitFrontendAttrs(dockerfilePath, contextUploader.Add(contextReader)),
		Session:       attachables,
		Exports: []bkclient.ExportEntry{
			{
				Type: bkclient.ExporterTar,
				Output: func(map[string]string) (io.WriteCloser, error) {
					return tarWriter, nil
				},
			},
		},
	}

	progressWriter, waitForLogs := logWriter(logger)
	defer waitForLogs()

	display, err := progressui.NewDisplay(io.MultiWriter(logboek.Context(ctx).OutStream(), progressWriter), progressui.PlainMode)
	if err != nil {
		return fmt.Errorf("unable to create build progress display: %w", err)
	}

	statusCh := make(chan *bkclient.SolveStatus)
	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		if _, err := bkClient.Solve(egCtx, nil, solveOpt, statusCh); err != nil {
			return fmt.Errorf("build failed: %w", err)
		}
		return nil
	})
	eg.Go(func() error {
		// Solve closes statusCh on return; an uncancelable context lets the
		// display drain it so the build error is reported, not a context one.
		if _, err := display.UpdateFrom(context.WithoutCancel(ctx), statusCh); err != nil {
			return fmt.Errorf("unable to display build progress: %w", err)
		}
		return nil
	})
	if err := eg.Wait(); err != nil {
		return err
	}

	// The tar exporter closes the writer it was handed as soon as the export
	// stream ends, so this only guarantees EOF for the reader if it did not.
	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("unable to close tar writer: %w", err)
	}
	return nil
}
