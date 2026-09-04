package docker

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/cli/cli/config"
	"github.com/stretchr/testify/require"
)

func TestDockerConfigDirForTokenSeeds_KeepsWritableDir(t *testing.T) {
	restoreDockerConfigDir(t)

	dir := filepath.Join(t.TempDir(), ".docker")
	config.SetDir(dir)

	require.Equal(t, dir, dockerConfigDirForTokenSeeds(context.Background()))
	require.Equal(t, dir, config.Dir())
	require.DirExists(t, dir)
}

func TestDockerConfigDirForTokenSeeds_FallsBackToTempDirWhenUnwritable(t *testing.T) {
	restoreDockerConfigDir(t)

	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	require.NoError(t, os.WriteFile(blocker, nil, 0o644))
	config.SetDir(filepath.Join(blocker, ".docker"))

	fallback := filepath.Join(os.TempDir(), "trdl-docker-config")
	require.Equal(t, fallback, dockerConfigDirForTokenSeeds(context.Background()))
	require.Equal(t, fallback, config.Dir())
	require.NoError(t, os.MkdirAll(config.Dir(), 0o755))
}

func TestBuildkitSessionAttachables_MovesTokenSeedsOffUnwritableConfigDir(t *testing.T) {
	restoreDockerConfigDir(t)

	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	require.NoError(t, os.WriteFile(blocker, nil, 0o644))
	config.SetDir(filepath.Join(blocker, ".docker"))

	attachables := buildkitSessionAttachables(context.Background(), nil, nil)

	require.Len(t, attachables, 3)
	require.Equal(t, filepath.Join(os.TempDir(), "trdl-docker-config"), config.Dir())
}

func restoreDockerConfigDir(t *testing.T) {
	original := config.Dir()
	t.Cleanup(func() { config.SetDir(original) })
}
