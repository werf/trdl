package docker

import (
	"archive/tar"
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"path"

	"github.com/djherbis/buffer"
	"github.com/djherbis/nio/v3"
	"github.com/docker/docker/client"
	"github.com/go-git/go-git/v5"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"
	uuid "github.com/satori/go.uuid"

	"github.com/werf/logboek"
	trdlGit "github.com/werf/trdl/server/pkg/git"
	"github.com/werf/trdl/server/pkg/secrets"
)

type BuildReleaseArtifactsOpts struct {
	FromImage   string
	RunCommands []string
	GitRepo     *git.Repository
	TarWriter   *nio.PipeWriter
	Storage     logical.Storage
}

func BuildReleaseArtifacts(ctx context.Context, opts BuildReleaseArtifactsOpts, logger hclog.Logger) (error, func() error) {
	serviceDirInContext := ".trdl"
	serviceDockerfilePathInContext := path.Join(serviceDirInContext, "Dockerfile")
	serviceLabels := map[string]string{
		"vault-trdl-release-uuid": uuid.NewV4().String(),
	}

	contextBuf := buffer.New(64 * 1024 * 1024)
	contextReader, contextWriter := nio.Pipe(contextBuf)

	secrets, err := secrets.GetSecrets(ctx, opts.Storage)
	if err != nil {
		return fmt.Errorf("unable to get build secrets: %w", err), nil
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
				Labels:  serviceLabels,
				Secrets: secrets,
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

	args, err := setCliArgs(serviceDockerfilePathInContext, secrets)
	if err != nil {
		return fmt.Errorf("unable to set cli args: %w", err), nil
	}

	if err := RunCliBuild(ctx, logger, contextReader, opts.TarWriter, args...); err != nil {
		return fmt.Errorf("can't build artifacts: %w", err), nil
	}

	logboek.Context(ctx).Default().LogLn("Build is successful")
	logger.Info("Build is successful")

	cleanupFunc := func() error {
		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			return fmt.Errorf("unable to create docker client: %w", err)
		}
		return RemoveImagesByLabels(ctx, cli, serviceLabels)
	}

	return nil, cleanupFunc
}

func RunCliBuild(ctx context.Context, logger hclog.Logger, contextReader *nio.PipeReader, tarWriter *nio.PipeWriter, args ...string) error {
	finalArgs := append([]string{"buildx", "build"}, args...)
	cmd := exec.CommandContext(ctx, "docker", finalArgs...)

	cmd.Stdout = tarWriter
	cmd.Stdin = contextReader

	multiWriter := io.MultiWriter(logboek.Context(ctx).OutStream(), logWriter(logger))
	cmd.Stderr = multiWriter

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("unable to close tar writer: %w", err)
	}

	return nil
}

func logWriter(logger hclog.Logger) *io.PipeWriter {
	pr, pw := io.Pipe()
	go func() {
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			line := scanner.Text()
			logger.Info(line)
		}
		if err := scanner.Err(); err != nil {
			logger.Error("error reading stderr", "err", err)
		}
	}()

	return pw
}

func setCliArgs(serviceDockerfilePathInContext string, secrets []secrets.Secret) ([]string, error) {
	args := []string{
		"--file", serviceDockerfilePathInContext,
		"--pull",
		"--no-cache",
	}

	if len(secrets) > 0 {
		if err := SetTempEnvVars(secrets); err != nil {
			return nil, fmt.Errorf("unable to set secrets")
		}
		args = append(args, GetSecretsCommandMounts(secrets)...)
	}

	args = append(args, "-o", "-", "-")
	return args, nil
}
