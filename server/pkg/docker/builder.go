package docker

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/djherbis/nio/v3"

	"github.com/werf/logboek"
	"github.com/werf/trdl/server/pkg/mac_signing"
	"github.com/werf/trdl/server/pkg/secrets"
)

type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

type Builder struct {
	builderName string
	buildArgs   []string
	logger      Logger
}

type NewBuilderOpts struct {
	BuildId               string
	ContextPath           string
	Secrets               []secrets.Secret
	MacSigningCredentials *mac_signing.MacSigningCredentials
	Logger                Logger
}

func NewBuilder(ctx context.Context, opts *NewBuilderOpts) (*Builder, error) {
	builderName := fmt.Sprintf("trdl-builder-%s", opts.BuildId)
	builderArgs := []string{
		"buildx",
		"create",
		"--name", builderName,
		"--driver=docker-container",
	}

	if err := runDockerCmd(ctx, builderArgs); err != nil {
		return nil, fmt.Errorf("builder setup failed: %w", err)
	}

	args, err := setCliArgs(builderName, opts.ContextPath, opts.Secrets, opts.MacSigningCredentials)
	if err != nil {
		return nil, fmt.Errorf("unable to set cli args: %w", err)
	}

	return &Builder{
		builderName: builderName,
		buildArgs:   args,
		logger:      opts.Logger,
	}, nil
}

func (b *Builder) Build(ctx context.Context, contextReader *nio.PipeReader, tarWriter *nio.PipeWriter) error {
	finalArgs := append([]string{"buildx", "build"}, b.buildArgs...)
	cmd := exec.CommandContext(ctx, "docker", finalArgs...)

	cmd.Stdout = tarWriter
	cmd.Stdin = contextReader

	multiWriter := io.MultiWriter(logboek.Context(ctx).OutStream(), logWriter(b.logger))
	cmd.Stderr = multiWriter

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("unable to close tar writer: %w", err)
	}
	return nil
}

func (b *Builder) Remove(ctx context.Context) error {
	if err := runDockerCmd(ctx, []string{"buildx", "rm", b.builderName}); err != nil {
		return fmt.Errorf("unable to cleanup: %w", err)
	}
	return nil
}

func logWriter(logger Logger) *io.PipeWriter {
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

func setCliArgs(builder, serviceDockerfilePathInContext string, secrets []secrets.Secret, macSigningCredentials *mac_signing.MacSigningCredentials) ([]string, error) {
	args := []string{
		"--file", serviceDockerfilePathInContext,
		"--pull",
		"--no-cache",
		"--builder", builder,
	}

	if len(secrets) > 0 {
		if err := SetTempEnvVars(secrets); err != nil {
			return nil, fmt.Errorf("unable to set secrets")
		}
		args = append(args, GetSecretsCommandMounts(secrets)...)
	}

	if macSigningCredentials != nil {
		if err := SetMacSigningTempEnvVars(macSigningCredentials); err != nil {
			return nil, fmt.Errorf("unable to set mac signing credentials")
		}
		args = append(args, GetMacSigningCommandMounts(macSigningCredentials)...)
	}

	args = append(args, "-o", "-", "-")
	return args, nil
}

func runDockerCmd(ctx context.Context, args []string) error {
	cmd := exec.CommandContext(ctx, "docker", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker command failed: %s %w", stderr.String(), err)
	}
	return nil
}
