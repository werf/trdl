package docker

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/werf/trdl/server/pkg/secrets"
)

// driverOptsFromEnv reads the ambient environment, so an operator-set
// TRDL_BUILDX_DRIVER_OPTS_* would otherwise leak into every expectation.
func clearDriverOptsEnv(t *testing.T) {
	for _, keyValue := range os.Environ() {
		if name, _, _ := strings.Cut(keyValue, "="); strings.HasPrefix(name, buildxDriverOptsEnvPrefix) {
			t.Setenv(name, "")
		}
	}
}

func TestBuildxCreateArgs_DefaultDriverUnchanged(t *testing.T) {
	clearDriverOptsEnv(t)
	t.Setenv(buildxDriverEnv, "")

	args, err := buildxCreateArgs(context.Background(), "trdl-builder-42", "", nil)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"buildx", "create",
		"--name", "trdl-builder-42",
		"--driver=docker-container",
	}, args)
}

func TestBuildxCreateArgs_KubernetesDriverWithOpts(t *testing.T) {
	clearDriverOptsEnv(t)
	t.Setenv(buildxDriverEnv, "kubernetes")
	t.Setenv(buildxDriverOptsEnvPrefix+"NAMESPACE", "namespace=trdl-build")
	t.Setenv(buildxDriverOptsEnvPrefix+"ROOTLESS", "rootless=true")

	args, err := buildxCreateArgs(context.Background(), "trdl-builder-42", "", nil)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"buildx", "create",
		"--name", "trdl-builder-42",
		"--driver=kubernetes",
		"--driver-opt=namespace=trdl-build",
		"--driver-opt=rootless=true",
	}, args)
}

func TestBuildxCreateArgs_DefaultDriverWithOpts(t *testing.T) {
	clearDriverOptsEnv(t)
	t.Setenv(buildxDriverEnv, "")
	t.Setenv(buildxDriverOptsEnvPrefix+"IMAGE", "image=moby/buildkit:v0.12.0")
	t.Setenv(buildxDriverOptsEnvPrefix+"NETWORK", "network=host")

	args, err := buildxCreateArgs(context.Background(), "trdl-builder-42", "", nil)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"buildx", "create",
		"--name", "trdl-builder-42",
		"--driver=docker-container",
		"--driver-opt=image=moby/buildkit:v0.12.0",
		"--driver-opt=network=host",
	}, args)
}

func TestBuildxCreateArgs_DriverValueTrimmed(t *testing.T) {
	clearDriverOptsEnv(t)
	t.Setenv(buildxDriverEnv, "  kubernetes  ")

	args, err := buildxCreateArgs(context.Background(), "trdl-builder-42", "", nil)

	require.NoError(t, err)
	assert.Contains(t, args, "--driver=kubernetes")
}

func TestBuildxCreateArgs_UnsupportedDriverRejected(t *testing.T) {
	clearDriverOptsEnv(t)
	t.Setenv(buildxDriverEnv, "docker")

	args, err := buildxCreateArgs(context.Background(), "trdl-builder-42", "", nil)

	require.Error(t, err)
	assert.Nil(t, args)
	assert.Contains(t, err.Error(), `"docker"`)
	assert.Contains(t, err.Error(), buildxDriverEnv)
}

func TestBuildxCreateArgs_CommaValuePassedThroughWithoutSeparator(t *testing.T) {
	clearDriverOptsEnv(t)
	t.Setenv(buildxDriverEnv, "kubernetes")
	t.Setenv(buildxDriverOptsEnvPrefix+"NODESELECTOR", "nodeselector=disktype=ssd,zone=a")

	args, err := buildxCreateArgs(context.Background(), "trdl-builder-42", "", nil)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"buildx", "create",
		"--name", "trdl-builder-42",
		"--driver=kubernetes",
		"--driver-opt=nodeselector=disktype=ssd,zone=a",
	}, args)
}

func TestBuildxCreateArgs_CustomOptsSeparator(t *testing.T) {
	clearDriverOptsEnv(t)
	t.Setenv(buildxDriverEnv, "kubernetes")
	t.Setenv(buildxDriverOptsSeparatorEnv, ";")
	t.Setenv(buildxDriverOptsEnvPrefix+"KUBE", "namespace=trdl-build;nodeselector=disktype=ssd,zone=a")

	args, err := buildxCreateArgs(context.Background(), "trdl-builder-42", "", nil)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"buildx", "create",
		"--name", "trdl-builder-42",
		"--driver=kubernetes",
		"--driver-opt=namespace=trdl-build",
		"--driver-opt=nodeselector=disktype=ssd,zone=a",
	}, args)
}

func TestBuildxCreateArgs_OptsOrderedByVariableName(t *testing.T) {
	clearDriverOptsEnv(t)
	t.Setenv(buildxDriverEnv, "kubernetes")
	t.Setenv(buildxDriverOptsEnvPrefix+"A1", "second=2")
	t.Setenv(buildxDriverOptsEnvPrefix+"A", "first=1")
	t.Setenv(buildxDriverOptsEnvPrefix+"EMPTY", "  ")

	args, err := buildxCreateArgs(context.Background(), "trdl-builder-42", "", nil)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"buildx", "create",
		"--name", "trdl-builder-42",
		"--driver=kubernetes",
		"--driver-opt=first=1",
		"--driver-opt=second=2",
	}, args)
}

func TestSetCliArgs_ExecPathUnchanged(t *testing.T) {
	t.Setenv("TEST_TRDL_SECRET", "")

	args, err := setCliArgs("trdl-builder-42", ".trdl/Dockerfile", []secrets.Secret{{Id: "TEST_TRDL_SECRET", Data: []byte("value")}}, nil)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"--file", ".trdl/Dockerfile",
		"--pull",
		"--no-cache",
		"--builder", "trdl-builder-42",
		"--secret", "id=TEST_TRDL_SECRET",
		"-o", "-", "-",
	}, args)
}

type discardLogger struct{}

func (discardLogger) Info(string, ...interface{})  {}
func (discardLogger) Error(string, ...interface{}) {}

func TestNewBuilder_BuildkitdAddressSkipsBuildxProvisioning(t *testing.T) {
	t.Setenv(buildkitdAddressEnv, "")

	builder, err := NewBuilder(context.Background(), &NewBuilderOpts{
		BuildId:                 "42",
		DockerfilePathInContext: ".trdl/Dockerfile",
		BuildkitdAddress:        "tcp://buildkitd:1234",
		Logger:                  discardLogger{},
	})

	require.NoError(t, err)
	assert.Equal(t, "tcp://buildkitd:1234", builder.buildkitdAddress)
	assert.Equal(t, ".trdl/Dockerfile", builder.dockerfilePath)
	assert.Empty(t, builder.buildArgs)
	assert.Empty(t, builder.builderName)
}

func TestParseDriverOpts(t *testing.T) {
	assert.Nil(t, parseDriverOpts("", ""))
	assert.Nil(t, parseDriverOpts("  ", ""))
	assert.Equal(t,
		[]string{"nodeselector=disktype=ssd,zone=a"},
		parseDriverOpts(" nodeselector=disktype=ssd,zone=a ", ""),
	)
	assert.Nil(t, parseDriverOpts(",  ,", ","))
	assert.Equal(t,
		[]string{"namespace=trdl-build", "rootless=true"},
		parseDriverOpts("  namespace=trdl-build ,, rootless=true ,", ","),
	)
}

func TestBuildxCreateArgs_ConfigurationOverridesEnv(t *testing.T) {
	clearDriverOptsEnv(t)
	t.Setenv(buildxDriverEnv, "docker-container")
	t.Setenv(buildxDriverOptsEnvPrefix+"IMAGE", "image=moby/buildkit:v0.12.0")

	args, err := buildxCreateArgs(context.Background(), "trdl-builder-42", "kubernetes", []string{"namespace=trdl-build", "rootless=true"})

	require.NoError(t, err)
	assert.Equal(t, []string{
		"buildx", "create",
		"--name", "trdl-builder-42",
		"--driver=kubernetes",
		"--driver-opt=namespace=trdl-build",
		"--driver-opt=rootless=true",
	}, args)
}

func TestBuildxCreateArgs_EnvUsedWhenConfigurationEmpty(t *testing.T) {
	clearDriverOptsEnv(t)
	t.Setenv(buildxDriverEnv, "kubernetes")
	t.Setenv(buildxDriverOptsEnvPrefix+"NAMESPACE", "namespace=trdl-build")

	args, err := buildxCreateArgs(context.Background(), "trdl-builder-42", "", nil)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"buildx", "create",
		"--name", "trdl-builder-42",
		"--driver=kubernetes",
		"--driver-opt=namespace=trdl-build",
	}, args)
}

func TestBuildxCreateArgs_ConfiguredDriverKeepsEnvOpts(t *testing.T) {
	clearDriverOptsEnv(t)
	t.Setenv(buildxDriverEnv, "")
	t.Setenv(buildxDriverOptsEnvPrefix+"NAMESPACE", "namespace=trdl-build")

	args, err := buildxCreateArgs(context.Background(), "trdl-builder-42", "kubernetes", nil)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"buildx", "create",
		"--name", "trdl-builder-42",
		"--driver=kubernetes",
		"--driver-opt=namespace=trdl-build",
	}, args)
}

func TestBuildxCreateArgs_ConfiguredOptsPassedThroughAndTrimmed(t *testing.T) {
	clearDriverOptsEnv(t)
	t.Setenv(buildxDriverEnv, "")

	args, err := buildxCreateArgs(context.Background(), "trdl-builder-42", "  kubernetes  ", []string{"  nodeselector=disktype=ssd,zone=a  ", "   "})

	require.NoError(t, err)
	assert.Equal(t, []string{
		"buildx", "create",
		"--name", "trdl-builder-42",
		"--driver=kubernetes",
		"--driver-opt=nodeselector=disktype=ssd,zone=a",
	}, args)
}

func TestBuildxCreateArgs_UnsupportedConfiguredDriverRejected(t *testing.T) {
	clearDriverOptsEnv(t)
	t.Setenv(buildxDriverEnv, "kubernetes")

	args, err := buildxCreateArgs(context.Background(), "trdl-builder-42", "docker", nil)

	require.Error(t, err)
	assert.Nil(t, args)
	assert.Contains(t, err.Error(), `"docker"`)
	assert.Contains(t, err.Error(), buildxDriverConfigurationSource)
	assert.NotContains(t, err.Error(), buildxDriverEnv)
}

func TestNewBuilder_UsesConfiguredDriver(t *testing.T) {
	clearDriverOptsEnv(t)
	t.Setenv(buildxDriverEnv, "kubernetes")
	// Validation is expected to reject the driver before docker is invoked; an
	// empty PATH makes sure a regression fails the test instead of provisioning
	// a real builder on the machine running it.
	t.Setenv("PATH", t.TempDir())

	builder, err := NewBuilder(context.Background(), &NewBuilderOpts{
		BuildId:      "42",
		BuildxDriver: "docker",
	})

	require.Error(t, err)
	assert.Nil(t, builder)
	assert.Contains(t, err.Error(), buildxDriverConfigurationSource)
}

func TestValidateBuildxDriver(t *testing.T) {
	assert.NoError(t, ValidateBuildxDriver(context.Background(), ""))
	assert.NoError(t, ValidateBuildxDriver(context.Background(), "  "))
	assert.NoError(t, ValidateBuildxDriver(context.Background(), "  kubernetes  "))
	assert.NoError(t, ValidateBuildxDriver(context.Background(), "docker-container"))
	assert.Error(t, ValidateBuildxDriver(context.Background(), "docker"))
	assert.Error(t, ValidateBuildxDriver(context.Background(), "remote"))
}
