package docker

import (
	"archive/tar"
	"context"
	"fmt"
	"path"
	"time"

	"github.com/djherbis/buffer"
	"github.com/djherbis/nio/v3"
	"github.com/go-git/go-git/v5"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"
	uuid "github.com/satori/go.uuid"

	"github.com/werf/logboek"
	trdlGit "github.com/werf/trdl/server/pkg/git"
	"github.com/werf/trdl/server/pkg/mac_signing"
	"github.com/werf/trdl/server/pkg/secrets"
)

const (
	defaultTimeOut = 30
)

type BuildReleaseArtifactsOpts struct {
	FromImage   string
	RunCommands []string
	GitRepo     *git.Repository
	TarWriter   *nio.PipeWriter
	Storage     logical.Storage
}

func BuildReleaseArtifacts(ctx context.Context, opts BuildReleaseArtifactsOpts, logger hclog.Logger) error {
	serviceDirInContext := ".trdl"
	serviceDockerfilePathInContext := path.Join(serviceDirInContext, "Dockerfile")
	buildId := uuid.NewV4().String()
	serviceLabels := map[string]string{
		"vault-trdl-release-uuid": buildId,
	}

	contextBuf := buffer.New(64 * 1024 * 1024)
	contextReader, contextWriter := nio.Pipe(contextBuf)

	secrets, err := secrets.GetSecrets(ctx, opts.Storage)
	if err != nil {
		return fmt.Errorf("unable to get build secrets: %w", err)
	}
	credentials, err := mac_signing.GetCredentials(ctx, opts.Storage)
	if err != nil {
		fmt.Printf("Warning: unable to get mac signing credentials: %v\n", err)
		fmt.Println("Continue without mac signing...")
		return nil
	}

	go func() {
		if err := func() error {
			tw := tar.NewWriter(contextWriter)

			logboek.Context(ctx).Default().LogF("Adding git worktree files to the build context\n")
			logger.Debug("Adding git worktree files to the build context")

			if err := trdlGit.AddWorktreeFilesToTar(tw, opts.GitRepo); err != nil {
				return fmt.Errorf("unable to add git worktree files to tar: %w", err)
			}

			dockerfileOpts := DockerfileOpts{
				Labels:                serviceLabels,
				Secrets:               secrets,
				MacSigningCredentials: credentials,
			}
			if err := GenerateAndAddDockerfileToTar(tw, serviceDockerfilePathInContext, opts.FromImage, opts.RunCommands, dockerfileOpts); err != nil {
				return fmt.Errorf("unable to add service dockerfile to tar: %w", err)
			}

			if err := tw.Close(); err != nil {
				return fmt.Errorf("unable to close tar writer: %w", err)
			}

			return nil
		}(); err != nil {
			if closeErr := contextWriter.CloseWithError(err); closeErr != nil {
				panic(closeErr)
			}
			return
		}

		if err := contextWriter.Close(); err != nil {
			panic(err)
		}
	}()

	logboek.Context(ctx).Default().LogLn("Building docker image with artifacts")
	logger.Info("Building docker image with artifacts")

	builder, err := NewBuilder(ctx, &NewBuilderOpts{
		BuildId:               buildId,
		ContextPath:           serviceDockerfilePathInContext,
		Secrets:               secrets,
		MacSigningCredentials: credentials,
		Logger:                logger,
	})
	if err != nil {
		return fmt.Errorf("unable to create docker builder: %w", err)
	}
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), defaultTimeOut*time.Second)
		defer cancel()
		if err := builder.Remove(cleanupCtx); err != nil {
			errStr := fmt.Sprintf("Unable to remove builder `%s`: %s", builder.builderName, err.Error())
			logboek.Context(ctx).Default().LogLn(errStr)
			logger.Info(errStr)
		}
	}()

	if err := builder.Build(ctx, contextReader, opts.TarWriter); err != nil {
		return fmt.Errorf("can't build artifacts: %w", err)
	}

	logboek.Context(ctx).Default().LogLn("Build is successful")
	logger.Info("Build is successful")

	return nil
}
