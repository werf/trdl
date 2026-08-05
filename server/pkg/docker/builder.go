package docker

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/djherbis/nio/v3"
	"github.com/samber/lo"

	"github.com/werf/logboek"
	"github.com/werf/trdl/server/pkg/mac_signing"
	"github.com/werf/trdl/server/pkg/secrets"
)

const (
	buildxDriverEnv = "TRDL_BUILDX_DRIVER"
	// Every TRDL_BUILDX_DRIVER_OPTS_<SUFFIX> variable carries one `--driver-opt`,
	// e.g. TRDL_BUILDX_DRIVER_OPTS_NAMESPACE="namespace=trdl-build", or several
	// when TRDL_BUILDX_DRIVER_OPTS_SEPARATOR is set.
	buildxDriverOptsEnvPrefix    = "TRDL_BUILDX_DRIVER_OPTS_"
	buildxDriverOptsSeparatorEnv = "TRDL_BUILDX_DRIVER_OPTS_SEPARATOR"

	buildxDriverConfigurationSource = "the buildx_driver plugin configuration"

	defaultBuildxDriver = "docker-container"
)

// supportedBuildxDrivers are the drivers trdl has verified. The build streams a
// tarball to stdout (`-o - -`), which the default "docker" driver cannot export,
// so an unsupported driver fails closed here with a clear error rather than
// later with an opaque build failure.
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
	BuildxDriver          string
	BuildxDriverOpts      []string
	Secrets               []secrets.Secret
	MacSigningCredentials *mac_signing.Credentials
	Logger                Logger
}

func NewBuilder(ctx context.Context, opts *NewBuilderOpts) (*Builder, error) {
	builderName := fmt.Sprintf("trdl-builder-%s", opts.BuildId)

	builderArgs, err := buildxCreateArgs(builderName, opts.BuildxDriver, opts.BuildxDriverOpts)
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

func buildxCreateArgs(builderName, configuredDriver string, configuredDriverOpts []string) ([]string, error) {
	driver, driverSource := resolveBuildxDriver(configuredDriver)
	if err := ValidateBuildxDriver(driver); err != nil {
		return nil, fmt.Errorf("buildx driver from %s: %w", driverSource, err)
	}

	args := []string{
		"buildx",
		"create",
		"--name", builderName,
		"--driver=" + driver,
	}
	for _, opt := range resolveBuildxDriverOpts(configuredDriverOpts) {
		args = append(args, "--driver-opt="+opt)
	}

	return args, nil
}

// ValidateBuildxDriver accepts an empty driver, meaning the setting is not in
// use. The build streams a tarball to stdout (`-o - -`), which the default
// "docker" driver cannot export, so an unsupported driver is rejected here
// instead of failing opaquely mid-build.
func ValidateBuildxDriver(driver string) error {
	driver = strings.TrimSpace(driver)
	if driver == "" || lo.Contains(supportedBuildxDrivers, driver) {
		return nil
	}

	return fmt.Errorf("unsupported driver %q (supported: %s)", driver, strings.Join(supportedBuildxDrivers, ", "))
}

// resolveBuildxDriver returns the driver to create the builder with and the
// name of the setting it came from, so that a rejection points at the knob the
// operator has to fix.
func resolveBuildxDriver(configuredDriver string) (string, string) {
	if driver := strings.TrimSpace(configuredDriver); driver != "" {
		return driver, buildxDriverConfigurationSource
	}
	if driver := strings.TrimSpace(os.Getenv(buildxDriverEnv)); driver != "" {
		return driver, buildxDriverEnv
	}

	return defaultBuildxDriver, "the default"
}

func resolveBuildxDriverOpts(configuredDriverOpts []string) []string {
	var opts []string
	for _, configuredOpt := range configuredDriverOpts {
		opts = append(opts, parseDriverOpts(configuredOpt, "")...)
	}
	if len(opts) > 0 {
		return opts
	}

	return driverOptsFromEnv()
}

func driverOptsFromEnv() []string {
	var names []string
	for _, keyValue := range os.Environ() {
		name, _, _ := strings.Cut(keyValue, "=")
		if strings.HasPrefix(name, buildxDriverOptsEnvPrefix) && name != buildxDriverOptsSeparatorEnv {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	separator := os.Getenv(buildxDriverOptsSeparatorEnv)

	var opts []string
	for _, name := range names {
		opts = append(opts, parseDriverOpts(os.Getenv(name), separator)...)
	}

	return opts
}

// An empty separator passes the value through untouched, so comma-valued opts
// such as nodeselector/tolerations survive without escaping.
func parseDriverOpts(raw, separator string) []string {
	parts := []string{raw}
	if separator != "" {
		parts = strings.Split(raw, separator)
	}

	var opts []string
	for _, part := range parts {
		if v := strings.TrimSpace(part); v != "" {
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
