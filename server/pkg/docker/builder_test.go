package docker

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	args, err := buildxCreateArgs("trdl-builder-42", "", nil)

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

	args, err := buildxCreateArgs("trdl-builder-42", "", nil)

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

	args, err := buildxCreateArgs("trdl-builder-42", "", nil)

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

	args, err := buildxCreateArgs("trdl-builder-42", "", nil)

	require.NoError(t, err)
	assert.Contains(t, args, "--driver=kubernetes")
}

func TestBuildxCreateArgs_UnsupportedDriverRejected(t *testing.T) {
	clearDriverOptsEnv(t)
	t.Setenv(buildxDriverEnv, "docker")

	args, err := buildxCreateArgs("trdl-builder-42", "", nil)

	require.Error(t, err)
	assert.Nil(t, args)
	assert.Contains(t, err.Error(), `"docker"`)
	assert.Contains(t, err.Error(), buildxDriverEnv)
}

func TestBuildxCreateArgs_CommaValuePassedThroughWithoutSeparator(t *testing.T) {
	clearDriverOptsEnv(t)
	t.Setenv(buildxDriverEnv, "kubernetes")
	t.Setenv(buildxDriverOptsEnvPrefix+"NODESELECTOR", "nodeselector=disktype=ssd,zone=a")

	args, err := buildxCreateArgs("trdl-builder-42", "", nil)

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

	args, err := buildxCreateArgs("trdl-builder-42", "", nil)

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

	args, err := buildxCreateArgs("trdl-builder-42", "", nil)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"buildx", "create",
		"--name", "trdl-builder-42",
		"--driver=kubernetes",
		"--driver-opt=first=1",
		"--driver-opt=second=2",
	}, args)
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
