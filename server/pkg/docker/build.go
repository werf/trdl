package docker

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path"
	"regexp"

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

	logboek.Context(ctx).Default().LogF("Building docker image with artifacts\n")
	logger.Debug("Building docker image with artifacts")

	args, err := setCliArgs(serviceDockerfilePathInContext, secrets)
	if err != nil {
		return fmt.Errorf("unable to set cli args: %w", err), nil
	}

	if err := RunCliBuild(ctx, contextReader, opts.TarWriter, args...); err != nil {
		return fmt.Errorf("can't build artifacts: %w", err), nil
	}

	logboek.Context(ctx).Default().LogF("Build is successful\n")

	cleanupFunc := func() error {
		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			return fmt.Errorf("unable to create docker client: %w", err)
		}
		return RemoveImagesByLabels(ctx, cli, serviceLabels)
	}

	return nil, cleanupFunc
}

func RunCliBuild(ctx context.Context, contextReader *nio.PipeReader, tarWriter *nio.PipeWriter, args ...string) error {
	finalArgs := append([]string{"buildx", "build"}, args...)
	cmd := exec.CommandContext(ctx, "docker", finalArgs...)
	var stdErr bytes.Buffer
	cmd.Stdout = tarWriter
	cmd.Stdin = contextReader
	cmd.Stderr = &stdErr

	if err := cmd.Run(); err != nil {
		if len := stdErr.Len(); len > 0 {
			errSection, parseErr := extractRelevantLogs(&stdErr)
			if parseErr == nil {
				return fmt.Errorf("build failed: %s %w", errSection.String(), err)
			}
		}
		return fmt.Errorf("build failed: %w", err)
	}

	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("unable to close tar writer: %w", err)
	}

	return nil
}

// this needs for parsing buildx logs due to the fact that buildx writes all to stderr
// it will look for the first error section of the logs that contains the error
// or if not found just lines starts with "error:" or "ERROR:"
func extractRelevantLogs(stderr *bytes.Buffer) (bytes.Buffer, error) {
	scanner := bufio.NewScanner(stderr)

	var errSection bytes.Buffer
	var foundSection bool
	reSectionStart := regexp.MustCompile(`^------$`)
	reError := regexp.MustCompile(`(?i)^error:`)

	for scanner.Scan() {
		line := scanner.Text()

		if reSectionStart.MatchString(line) {
			if foundSection {
				break
			}
			foundSection = true
		}

		if foundSection || reError.MatchString(line) {
			errSection.WriteString(line + "\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return errSection, fmt.Errorf("error reading stderr: %w", err)
	}

	return errSection, nil
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
