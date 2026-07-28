package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildxCreateArgs_DefaultDriverUnchanged(t *testing.T) {
	t.Setenv(buildxDriverEnv, "")

	args, err := buildxCreateArgs("trdl-builder-42")

	require.NoError(t, err)
	assert.Equal(t, []string{
		"buildx", "create",
		"--name", "trdl-builder-42",
		"--driver=docker-container",
	}, args)
}

func TestBuildxCreateArgs_KubernetesDriverWithOpts(t *testing.T) {
	t.Setenv(buildxDriverEnv, "kubernetes")
	t.Setenv(buildxDriverOptsEnvPrefix+"KUBE", "namespace=trdl-build,rootless=true")

	args, err := buildxCreateArgs("trdl-builder-42")

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
	t.Setenv(buildxDriverEnv, "")
	t.Setenv(buildxDriverOptsEnvPrefix+"CONTAINER", "image=moby/buildkit:v0.12.0,network=host")

	args, err := buildxCreateArgs("trdl-builder-42")

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
	t.Setenv(buildxDriverEnv, "  kubernetes  ")

	args, err := buildxCreateArgs("trdl-builder-42")

	require.NoError(t, err)
	assert.Contains(t, args, "--driver=kubernetes")
}

func TestBuildxCreateArgs_UnsupportedDriverRejected(t *testing.T) {
	t.Setenv(buildxDriverEnv, "docker")

	args, err := buildxCreateArgs("trdl-builder-42")

	require.Error(t, err)
	assert.Nil(t, args)
	assert.Contains(t, err.Error(), `"docker"`)
	assert.Contains(t, err.Error(), buildxDriverEnv)
}

func TestBuildxCreateArgs_CustomOptsSeparator(t *testing.T) {
	// CSV-valued opts require a non-comma separator.
	t.Setenv(buildxDriverEnv, "kubernetes")
	t.Setenv(buildxDriverOptsSeparatorEnv, ";")
	t.Setenv(buildxDriverOptsEnvPrefix+"KUBE", "namespace=trdl-build;nodeselector=disktype=ssd,zone=a")

	args, err := buildxCreateArgs("trdl-builder-42")

	require.NoError(t, err)
	assert.Equal(t, []string{
		"buildx", "create",
		"--name", "trdl-builder-42",
		"--driver=kubernetes",
		"--driver-opt=namespace=trdl-build",
		"--driver-opt=nodeselector=disktype=ssd,zone=a",
	}, args)
}

func TestBuildxCreateArgs_MultipleOptVars(t *testing.T) {
	t.Setenv(buildxDriverEnv, "kubernetes")
	t.Setenv(buildxDriverOptsSeparatorEnv, "")
	t.Setenv(buildxDriverOptsEnvPrefix+"ROOTLESS", "rootless=true")
	t.Setenv(buildxDriverOptsEnvPrefix+"NAMESPACE", " namespace=trdl-build ")
	t.Setenv(buildxDriverOptsEnvPrefix+"EMPTY", "  ")

	args, err := buildxCreateArgs("trdl-builder-42")

	require.NoError(t, err)
	assert.Equal(t, []string{
		"buildx", "create",
		"--name", "trdl-builder-42",
		"--driver=kubernetes",
		"--driver-opt=namespace=trdl-build",
		"--driver-opt=rootless=true",
	}, args)
}

func TestParseDriverOpts(t *testing.T) {
	assert.Nil(t, parseDriverOpts("", ","))
	assert.Nil(t, parseDriverOpts(",  ,", ","))
	assert.Equal(t,
		[]string{"namespace=trdl-build", "rootless=true"},
		parseDriverOpts("  namespace=trdl-build ,, rootless=true ,", ","),
	)
	assert.Equal(t,
		[]string{"namespace=trdl-build", "nodeselector=disktype=ssd,zone=a"},
		parseDriverOpts("namespace=trdl-build\nnodeselector=disktype=ssd,zone=a", "\n"),
	)
}
