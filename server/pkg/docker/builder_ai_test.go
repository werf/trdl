//go:build ai_tests

package docker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAI_BuildxCreateArgs_ConfigurationOverridesEnv(t *testing.T) {
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

func TestAI_BuildxCreateArgs_EnvUsedWhenConfigurationEmpty(t *testing.T) {
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

func TestAI_BuildxCreateArgs_ConfiguredDriverKeepsEnvOpts(t *testing.T) {
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

func TestAI_BuildxCreateArgs_ConfiguredOptsPassedThroughAndTrimmed(t *testing.T) {
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

func TestAI_BuildxCreateArgs_UnsupportedConfiguredDriverRejected(t *testing.T) {
	clearDriverOptsEnv(t)
	t.Setenv(buildxDriverEnv, "kubernetes")

	args, err := buildxCreateArgs(context.Background(), "trdl-builder-42", "docker", nil)

	require.Error(t, err)
	assert.Nil(t, args)
	assert.Contains(t, err.Error(), `"docker"`)
	assert.Contains(t, err.Error(), buildxDriverConfigurationSource)
	assert.NotContains(t, err.Error(), buildxDriverEnv)
}

func TestAI_NewBuilder_UsesConfiguredDriver(t *testing.T) {
	clearDriverOptsEnv(t)
	t.Setenv(buildxDriverEnv, "kubernetes")

	builder, err := NewBuilder(context.Background(), &NewBuilderOpts{
		BuildId:      "42",
		BuildxDriver: "docker",
	})

	require.Error(t, err)
	assert.Nil(t, builder)
	assert.Contains(t, err.Error(), buildxDriverConfigurationSource)
}

func TestAI_ValidateBuildxDriver(t *testing.T) {
	assert.NoError(t, ValidateBuildxDriver(context.Background(), ""))
	assert.NoError(t, ValidateBuildxDriver(context.Background(), "  "))
	assert.NoError(t, ValidateBuildxDriver(context.Background(), "  kubernetes  "))
	assert.NoError(t, ValidateBuildxDriver(context.Background(), "docker-container"))
	assert.Error(t, ValidateBuildxDriver(context.Background(), "docker"))
	assert.Error(t, ValidateBuildxDriver(context.Background(), "remote"))
}
