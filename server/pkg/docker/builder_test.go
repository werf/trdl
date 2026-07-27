package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildxCreateArgs_DefaultDriverUnchanged(t *testing.T) {
	// With no env set the invocation must stay byte-for-byte the historical one.
	t.Setenv(buildxDriverEnv, "")
	t.Setenv(buildxDriverOptsEnv, "")

	args, err := buildxCreateArgs("trdl-builder-42")

	assert.NoError(t, err)
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

	assert.NoError(t, err)
	assert.Equal(t, []string{
		"buildx", "create",
		"--name", "trdl-builder-42",
		"--driver=kubernetes",
		"--driver-opt=namespace=trdl-build",
		"--driver-opt=rootless=true",
	}, args)
}

func TestBuildxCreateArgs_UnsupportedDriverRejected(t *testing.T) {
	// The default "docker" driver cannot export the stdout tarball, so it must
	// be rejected up front rather than failing opaquely during the build.
	t.Setenv(buildxDriverEnv, "docker")
	t.Setenv(buildxDriverOptsEnv, "")

	_, err := buildxCreateArgs("trdl-builder-42")

	assert.Error(t, err)
}

func TestParseDriverOpts(t *testing.T) {
	assert.Nil(t, parseDriverOpts(""))
	assert.Nil(t, parseDriverOpts("\n  \n"))
	assert.Equal(t,
		[]string{"namespace=trdl-build", "nodeselector=disktype=ssd,zone=a"},
		parseDriverOpts("  namespace=trdl-build \n\n nodeselector=disktype=ssd,zone=a \n"),
	)
}
