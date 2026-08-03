package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	bkclient "github.com/moby/buildkit/client"
	// Blank imports register the docker-container:// and kube-pod:// transports
	// in the BuildKit client; unix:// and tcp:// are handled natively.
	_ "github.com/moby/buildkit/client/connhelper/dockercontainer"
	_ "github.com/moby/buildkit/client/connhelper/kubepod"
	"github.com/moby/buildkit/session"
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

func resolveBuildkitdAddress(configuredAddress string) (string, error) {
	address := strings.TrimSpace(configuredAddress)
	if address == "" {
		address = strings.TrimSpace(os.Getenv(buildkitdAddressEnv))
	}
	if err := ValidateBuildkitdAddress(address); err != nil {
		return "", err
	}
	return address, nil
}

func ValidateBuildkitdAddress(address string) error {
	address = strings.TrimSpace(address)
	if address == "" {
		return nil
	}

	scheme, _, found := strings.Cut(address, "://")
	if !found || !lo.Contains(supportedBuildkitdAddressSchemes, scheme) {
		return fmt.Errorf("unsupported buildkitd address %q: expected scheme to be one of %s", address, strings.Join(supportedBuildkitdAddressSchemes, ", "))
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
		if macSigningCredentials.Password != "" {
			data[identityName+"_password"] = []byte(macSigningCredentials.Password)
		}
		data[identityName+"_notary_key_id"] = []byte(macSigningCredentials.NotaryKeyID)
		data[identityName+"_notary_key"] = []byte(macSigningCredentials.NotaryKey)
		data[identityName+"_notary_issuer"] = []byte(macSigningCredentials.NotaryIssuer)
	}

	return data
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

	contextUploader := uploadprovider.New()
	solveOpt := bkclient.SolveOpt{
		Frontend:      "dockerfile.v0",
		FrontendAttrs: buildkitFrontendAttrs(dockerfilePath, contextUploader.Add(contextReader)),
		Session:       []session.Attachable{contextUploader, secretsprovider.FromMap(secretsData)},
		Exports: []bkclient.ExportEntry{
			{
				Type: bkclient.ExporterTar,
				Output: func(map[string]string) (io.WriteCloser, error) {
					return tarWriter, nil
				},
			},
		},
	}

	progressWriter := logWriter(logger)
	defer progressWriter.Close()

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

	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("unable to close tar writer: %w", err)
	}
	return nil
}
