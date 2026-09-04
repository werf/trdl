package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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
// image-resolve-mode=pull can only reach public registries.
func buildkitSessionAttachables(ctx context.Context, contextUploader *uploadprovider.Uploader, secretsData map[string][]byte) []session.Attachable {
	dockerConfig := config.LoadDefaultConfigFile(logboek.Context(ctx).OutStream())
	dockerConfigDirForTokenSeeds(ctx)

	return []session.Attachable{
		contextUploader,
		secretsprovider.FromMap(secretsData),
		authprovider.NewDockerAuthProvider(authprovider.DockerAuthProviderConfig{
			AuthConfigProvider: authprovider.LoadAuthConfig(dockerConfig),
		}),
	}
}

// The auth provider keeps registry token seeds under config.Dir() and creates
// that directory on the first token request, so a process without a writable
// home (a builtin backend on a read-only root) fails every pull, anonymous ones
// included. The config file has already been read from the default location by
// the time this runs; only the seeds move.
func dockerConfigDirForTokenSeeds(ctx context.Context) string {
	dir := config.Dir()
	if err := os.MkdirAll(dir, 0o755); err == nil {
		return dir
	}

	fallback := filepath.Join(os.TempDir(), "trdl-docker-config")
	config.SetDir(fallback)

	msg := fmt.Sprintf("Docker config dir %q is not writable, keeping BuildKit registry token seeds in %q", dir, fallback)
	logboek.Context(ctx).Default().LogLn(msg)
	return fallback
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
	solveOpt := bkclient.SolveOpt{
		Frontend:      "dockerfile.v0",
		FrontendAttrs: buildkitFrontendAttrs(dockerfilePath, contextUploader.Add(contextReader)),
		Session:       buildkitSessionAttachables(ctx, contextUploader, secretsData),
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
