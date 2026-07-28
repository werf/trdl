package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildxCreateArgs_DefaultDriverUnchanged(t *testing.T) {
	t.Setenv(buildxDriverEnv, "")
	t.Setenv(buildxDriverOptsEnv, "")

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
	t.Setenv(buildxDriverOptsEnv, "namespace=trdl-build\nrootless=true")

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
	t.Setenv(buildxDriverOptsEnv, "image=moby/buildkit:v0.12.0\nnetwork=host")

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
	t.Setenv(buildxDriverOptsEnv, "")

	args, err := buildxCreateArgs("trdl-builder-42")

	require.NoError(t, err)
	assert.Contains(t, args, "--driver=kubernetes")
}

func TestBuildxCreateArgs_UnsupportedDriverRejected(t *testing.T) {
	t.Setenv(buildxDriverEnv, "docker")
	t.Setenv(buildxDriverOptsEnv, "")

	args, err := buildxCreateArgs("trdl-builder-42")

	require.Error(t, err)
	assert.Nil(t, args)
	assert.Contains(t, err.Error(), `"docker"`)
	assert.Contains(t, err.Error(), buildxDriverEnv)
}

func TestParseDriverOpts(t *testing.T) {
	assert.Nil(t, parseDriverOpts(""))
	assert.Nil(t, parseDriverOpts("\n  \n"))
	assert.Equal(t,
		[]string{"namespace=trdl-build", "nodeselector=disktype=ssd,zone=a"},
		parseDriverOpts("  namespace=trdl-build \n\n nodeselector=disktype=ssd,zone=a \n"),
	)
}
