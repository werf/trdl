package docker

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/docker/cli/cli/config"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth"
	"github.com/stretchr/testify/require"
)

func TestBuildkitSessionAttachables_KeepsWritableConfigDir(t *testing.T) {
	restoreDockerConfigDir(t)

	dir := filepath.Join(t.TempDir(), ".docker")
	config.SetDir(dir)

	err := tokenAuthority(buildkitSessionAttachables(context.Background(), nil, nil))

	require.NoError(t, err)
	require.Equal(t, dir, config.Dir())
	require.FileExists(t, filepath.Join(dir, ".token_seed"))
}

func TestBuildkitSessionAttachables_SeedsGoToPrivateTempDirWhenConfigDirUnwritable(t *testing.T) {
	restoreDockerConfigDir(t)

	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	require.NoError(t, os.WriteFile(blocker, nil, 0o644))
	unwritable := filepath.Join(blocker, ".docker")
	config.SetDir(unwritable)

	err := tokenAuthority(buildkitSessionAttachables(context.Background(), nil, nil))

	require.NoError(t, err)
	require.Equal(t, unwritable, config.Dir())
	require.NoFileExists(t, filepath.Join(unwritable, ".token_seed"))

	seedDir := tokenSeedDir
	require.True(t, strings.HasPrefix(filepath.Base(seedDir), "trdl-docker-config-"), seedDir)
	require.Contains(t, []string{"/dev/shm", filepath.Clean(os.TempDir())}, filepath.Dir(seedDir))
	info, err := os.Stat(seedDir)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o700), info.Mode().Perm())
	require.FileExists(t, filepath.Join(seedDir, ".token_seed"))
}

func TestBuildkitSessionAttachables_ConcurrentBuildsDoNotShareTheRedirect(t *testing.T) {
	restoreDockerConfigDir(t)

	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	require.NoError(t, os.WriteFile(blocker, nil, 0o644))
	unwritable := filepath.Join(blocker, ".docker")
	config.SetDir(unwritable)

	var wg sync.WaitGroup
	errs := make([]error, 8)
	for i := range errs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = tokenAuthority(buildkitSessionAttachables(context.Background(), nil, nil))
		}()
	}
	wg.Wait()

	for _, err := range errs {
		require.NoError(t, err)
	}
	require.Equal(t, unwritable, config.Dir())
}

// tokenAuthority drives the auth provider the way buildkitd does on a bearer
// challenge, which is the call that creates the token seed directory.
func tokenAuthority(attachables []session.Attachable) error {
	if len(attachables) != 3 {
		return fmt.Errorf("expected 3 attachables, got %d", len(attachables))
	}
	authServer, ok := attachables[2].(auth.AuthServer)
	if !ok {
		return fmt.Errorf("the third attachable %T is not the auth provider", attachables[2])
	}

	_, err := authServer.GetTokenAuthority(context.Background(), &auth.GetTokenAuthorityRequest{
		Host: "registry.invalid",
		Salt: bytes.Repeat([]byte{7}, 32),
	})
	return err
}

func restoreDockerConfigDir(t *testing.T) {
	original := config.Dir()
	t.Cleanup(func() { config.SetDir(original) })
}
