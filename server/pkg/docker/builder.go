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

	buildkitdDriverKubernetes = "kubernetes"
)

// supportedBuildxDrivers are the drivers trdl has verified. The build streams a
// tarball to stdout (`-o - -`), which the default "docker" driver cannot export,
// so an unsupported driver fails closed here with a clear error rather than
// later with an opaque build failure.
var supportedBuildxDrivers = []string{"docker-container", "kubernetes"}

// supportedBuildkitdDrivers are the ways trdl provisions a buildkitd of its own,
// needing no docker binary. Unset means it provisions none: the build either
// connects to buildkitd_address or goes through the docker CLI.
var supportedBuildkitdDrivers = []string{buildkitdDriverKubernetes}

type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

type Builder struct {
	builderName       string
	buildArgs         []string
	buildkitdAddress  string
	kubernetesBuilder *kubernetesBuilder
	dockerfilePath    string
	secretsData       map[string][]byte
	logger            Logger
}

type NewBuilderOpts struct {
	BuildId                 string
	DockerfilePathInContext string
	BuildkitdAddress        string
	BuildxDriver            string
	BuildxDriverOpts        []string
	BuildkitdDriver         string
	BuildkitdDriverOpts     []string
	Secrets                 []secrets.Secret
	MacSigningCredentials   *mac_signing.Credentials
	Logger                  Logger
}

func NewBuilder(ctx context.Context, opts *NewBuilderOpts) (*Builder, error) {
	buildkitdAddress, err := resolveBuildkitdAddress(ctx, opts.BuildkitdAddress)
	if err != nil {
		return nil, err
	}
	if buildkitdAddress != "" {
		// configure rejects these settings written together, but the address can
		// also come from the environment, and then they are unreachable rather
		// than rejected. Blank values mean "not set", as they do everywhere the
		// settings are resolved.
		if unused := unusedBuilderSettings(opts); len(unused) > 0 {
			msg := fmt.Sprintf("Building against buildkitd at %q, the configured %s settings are not used", buildkitdAddress, strings.Join(unused, " and "))
			logboek.Context(ctx).Default().LogLn(msg)
			opts.Logger.Info(msg)
		}

		return &Builder{
			buildkitdAddress: buildkitdAddress,
			dockerfilePath:   opts.DockerfilePathInContext,
			secretsData:      buildkitSecretsData(opts.Secrets, opts.MacSigningCredentials),
			logger:           opts.Logger,
		}, nil
	}

	builderName := fmt.Sprintf("trdl-builder-%s", opts.BuildId)

	if strings.TrimSpace(opts.BuildkitdDriver) != "" {
		kubernetesBuilder, err := newKubernetesBuilder(ctx, builderName, opts.BuildkitdDriver, opts.BuildkitdDriverOpts, opts.Logger)
		if err != nil {
			return nil, err
		}

		return &Builder{
			builderName:       builderName,
			kubernetesBuilder: kubernetesBuilder,
			dockerfilePath:    opts.DockerfilePathInContext,
			secretsData:       buildkitSecretsData(opts.Secrets, opts.MacSigningCredentials),
			logger:            opts.Logger,
		}, nil
	}

	builderArgs, err := buildxCreateArgs(ctx, builderName, opts.BuildxDriver, opts.BuildxDriverOpts)
	if err != nil {
		return nil, fmt.Errorf("unable to construct buildx create args: %w", err)
	}

	if err := runDockerCmd(ctx, builderArgs); err != nil {
		return nil, fmt.Errorf("builder setup failed: %w", err)
	}

	args, err := setCliArgs(builderName, opts.DockerfilePathInContext, opts.Secrets, opts.MacSigningCredentials)
	if err != nil {
		return nil, fmt.Errorf("unable to set cli args: %w", err)
	}

	return &Builder{
		builderName: builderName,
		buildArgs:   args,
		logger:      opts.Logger,
	}, nil
}

// unusedBuilderSettings names the settings a buildkitd address makes unreachable,
// so the build log says which knob is being ignored rather than leaving it to be
// discovered from a builder that never appears.
func unusedBuilderSettings(opts *NewBuilderOpts) []string {
	var unused []string
	if strings.TrimSpace(opts.BuildxDriver) != "" || len(trimDriverOpts(opts.BuildxDriverOpts)) > 0 {
		unused = append(unused, "buildx driver")
	}
	if strings.TrimSpace(opts.BuildkitdDriver) != "" || len(trimDriverOpts(opts.BuildkitdDriverOpts)) > 0 {
		unused = append(unused, "buildkitd driver")
	}

	return unused
}

func buildxCreateArgs(ctx context.Context, builderName, configuredDriver string, configuredDriverOpts []string) ([]string, error) {
	driver, driverSource := resolveBuildxDriver(configuredDriver)
	if err := ValidateBuildxDriver(ctx, driver); err != nil {
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
func ValidateBuildxDriver(ctx context.Context, driver string) error {
	driver = strings.TrimSpace(driver)
	if driver == "" || lo.Contains(supportedBuildxDrivers, driver) {
		return nil
	}

	return fmt.Errorf("unsupported driver %q (supported: %s)", driver, strings.Join(supportedBuildxDrivers, ", "))
}

// ValidateBuildkitdDriver accepts an empty driver, meaning trdl provisions no
// buildkitd of its own.
func ValidateBuildkitdDriver(ctx context.Context, driver string) error {
	driver = strings.TrimSpace(driver)
	if driver == "" || lo.Contains(supportedBuildkitdDrivers, driver) {
		return nil
	}

	return fmt.Errorf("unsupported buildkitd driver %q (supported: %s)", driver, strings.Join(supportedBuildkitdDrivers, ", "))
}

// ValidateBuildkitdDriverOpts rejects an option the driver cannot honor while
// the configuration is being written, rather than at release time. The options
// have no environment counterpart, so this check is reachable for every value
// that can ever arrive.
func ValidateBuildkitdDriverOpts(ctx context.Context, driver string, driverOpts []string) error {
	driverOpts = trimDriverOpts(driverOpts)
	if strings.TrimSpace(driver) == "" {
		if len(driverOpts) > 0 {
			return fmt.Errorf("buildkitd driver options are set without a buildkitd driver")
		}

		return nil
	}

	if _, err := parseKubernetesDriverOpts(driverOpts); err != nil {
		return err
	}

	return nil
}

func trimDriverOpts(driverOpts []string) []string {
	return lo.FilterMap(driverOpts, func(driverOpt string, _ int) (string, bool) {
		trimmed := strings.TrimSpace(driverOpt)

		return trimmed, trimmed != ""
	})
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

// An empty configured list means the setting is not in use, not "run with no
// options": `configure` cannot tell an omitted field from an explicitly empty
// one, so both fall back to the environment.
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
	// A build that fails before draining the context leaves the goroutine filling
	// it blocked on write forever; closing the reader here fails those writes.
	defer contextReader.Close()

	if b.buildkitdAddress != "" {
		return buildWithBuildkit(ctx, b.buildkitdAddress, b.dockerfilePath, b.secretsData, contextReader, tarWriter, b.logger)
	}

	if b.kubernetesBuilder != nil {
		bkClient, err := b.kubernetesBuilder.client(ctx)
		if err != nil {
			return err
		}
		defer bkClient.Close()

		return buildWithBuildkitClient(ctx, bkClient, b.dockerfilePath, b.secretsData, contextReader, tarWriter, b.logger)
	}

	finalArgs := append([]string{"buildx", "build"}, b.buildArgs...)
	cmd := exec.CommandContext(ctx, "docker", finalArgs...)

	cmd.Stdout = tarWriter
	cmd.Stdin = contextReader

	logPipe, waitForLogs := logWriter(b.logger)
	defer waitForLogs()

	cmd.Stderr = io.MultiWriter(logboek.Context(ctx).OutStream(), logPipe)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("unable to close tar writer: %w", err)
	}
	return nil
}

func (b *Builder) Remove(ctx context.Context) error {
	if b.buildkitdAddress != "" {
		return nil
	}

	if b.kubernetesBuilder != nil {
		return b.kubernetesBuilder.remove(ctx)
	}

	if err := runDockerCmd(ctx, []string{"buildx", "rm", b.builderName}); err != nil {
		return fmt.Errorf("unable to cleanup: %w", err)
	}
	return nil
}

const maxLogLineSize = 1024 * 1024

// The returned wait function closes the writer and returns once every buffered
// line has reached the logger, so the tail of a build log is not lost when the
// build finishes.
func logWriter(logger Logger) (*io.PipeWriter, func()) {
	pr, pw := io.Pipe()
	done := make(chan struct{})
	go func() {
		defer close(done)
		// The build writes into this pipe, so the moment line parsing stops the
		// build itself blocks on the next write. Whatever happens above, keep
		// draining until the writer is closed.
		defer func() { _, _ = io.Copy(io.Discard, pr) }()

		scanner := bufio.NewScanner(pr)
		scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), maxLogLineSize)
		for scanner.Scan() {
			line := scanner.Text()
			logger.Info(line)

			if recommendation := getRecommendation(line); recommendation != "" {
				logger.Info("Recommendation: " + recommendation)
			}
		}
		if err := scanner.Err(); err != nil {
			logger.Error("unable to read build output, the rest of it is not logged", "err", err)
		}
	}()

	return pw, func() {
		pw.Close()
		<-done
	}
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
