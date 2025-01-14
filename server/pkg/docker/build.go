package docker

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/djherbis/buffer"
	"github.com/djherbis/nio/v3"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/docker/client"
	"github.com/go-git/go-git/v5"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"
	uuid "github.com/satori/go.uuid"
	"github.com/spf13/cobra"
	"github.com/werf/logboek"
	trdlGit "github.com/werf/trdl/server/pkg/git"
	"github.com/werf/trdl/server/pkg/secrets"
)

var (
	defaultCLI command.Cli
)

const (
	ctxDockerCliKey = "docker_cli"
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

	logboek.Context(ctx).Default().LogF("Building docker image with artifacts\n")
	logger.Debug("Building docker image with artifacts")

	args, err := setCliArgs(serviceDockerfilePathInContext, secrets)
	if err != nil {
		return fmt.Errorf("unable to set cli args: %w", err), nil
	}

	cli, err := newDockerCli(defaultCliOptions(ctx))
	if err != nil {
		return fmt.Errorf("error creating docker cli: %w", err), nil
	}

	if err := CliBuild(cli, contextReader, opts.TarWriter, args...); err != nil {
		return err, nil
	}

	cleanupFunc := func() error {
		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			return fmt.Errorf("unable to create docker client: %w", err)
		}
		return RemoveImagesByLabels(ctx, cli, serviceLabels)
	}

	return nil, cleanupFunc
}

func CliBuild(c command.Cli, contextReader *nio.PipeReader, tarWriter *nio.PipeWriter, args ...string) error {
	var finalArgs []string
	var cmd *cobra.Command

	c.SetIn(streams.NewIn(contextReader))

	cmd = NewBuildxCommand(c)
	finalArgs = append([]string{"build"}, args...)

	cmd = prepareCliCmd(cmd, finalArgs...)

	reader, writer, err := os.Pipe()
	if err != nil {
		return err
	}

	origStdOut := os.Stdout
	os.Stdout = writer
	defer func() { os.Stdout = origStdOut }()

	go func() {
		defer reader.Close()
		if _, err := io.Copy(tarWriter, reader); err != nil {
			if closeErr := tarWriter.CloseWithError(err); closeErr != nil {
				panic(closeErr)
			}
			return
		}
	}()

	err = cmd.Execute()
	if err != nil {
		return fmt.Errorf("failed to execute build: %w", err)
	}
	return nil
}
