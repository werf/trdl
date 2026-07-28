package docker

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/djherbis/nio/v3"
	"github.com/samber/lo"

	"github.com/werf/logboek"
	"github.com/werf/trdl/server/pkg/mac_signing"
	"github.com/werf/trdl/server/pkg/secrets"
)

const (
	buildxDriverEnv = "TRDL_BUILDX_DRIVER"
	// One `--driver-opt` per line, e.g. "namespace=trdl-build\nrootless=true".
	buildxDriverOptsEnv = "TRDL_BUILDX_DRIVER_OPTS"

	defaultBuildxDriver = "docker-container"
)

// supportedBuildxDrivers are the drivers trdl will create a builder for. The
// build streams a tarball to stdout (`-o - -`), which the default "docker"
// driver cannot export; only full-BuildKit drivers are accepted, so an
// unsupported driver fails closed here with a clear error rather than later with
// an opaque build failure.
var supportedBuildxDrivers = []string{"docker-container", "kubernetes"}

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
	MacSigningCredentials *mac_signing.Credentials
	Logger                Logger
}

func NewBuilder(ctx context.Context, opts *NewBuilderOpts) (*Builder, error) {
	builderName := fmt.Sprintf("trdl-builder-%s", opts.BuildId)

	builderArgs, err := buildxCreateArgs(builderName)
	if err != nil {
		return nil, fmt.Errorf("unable to construct buildx create args: %w", err)
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

func buildxCreateArgs(builderName string) ([]string, error) {
	driver := strings.TrimSpace(os.Getenv(buildxDriverEnv))
	if driver == "" {
		driver = defaultBuildxDriver
	}
	if !lo.Contains(supportedBuildxDrivers, driver) {
		return nil, fmt.Errorf("unsupported buildx driver %q from %s (supported: %s)", driver, buildxDriverEnv, strings.Join(supportedBuildxDrivers, ", "))
	}

	args := []string{
		"buildx",
		"create",
		"--name", builderName,
		"--driver=" + driver,
	}
	for _, opt := range parseDriverOpts(os.Getenv(buildxDriverOptsEnv)) {
		args = append(args, "--driver-opt="+opt)
	}

	return args, nil
}

// Newline separation keeps commas intact for CSV-valued opts such as
// nodeselector/tolerations.
func parseDriverOpts(raw string) []string {
	var opts []string
	for _, line := range strings.Split(raw, "\n") {
		if v := strings.TrimSpace(line); v != "" {
			opts = append(opts, v)
		}
	}

	return opts
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

			if recommendation := getRecommendation(line); recommendation != "" {
				logger.Info("Recommendation: " + recommendation)
			}
		}
		if err := scanner.Err(); err != nil {
			logger.Error("error reading stderr", "err", err)
		}
	}()

	return pw
}

func setCliArgs(builder, serviceDockerfilePathInContext string, secrets []secrets.Secret, macSigningCredentials *mac_signing.Credentials) ([]string, error) {
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
