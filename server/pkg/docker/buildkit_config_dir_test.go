package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/docker/cli/cli/config"
	"github.com/hashicorp/go-hclog"
	bkclient "github.com/moby/buildkit/client"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth"
	"github.com/stretchr/testify/require"

	"github.com/werf/logboek"
)

func TestBuildkitSessionAttachables_KeepsWritableConfigDir(t *testing.T) {
	isolateTokenSeedTest(t)

	dir := filepath.Join(t.TempDir(), ".docker")
	config.SetDir(dir)
	before := tokenSeedDirs(t)

	attachables, cleanup := buildkitSessionAttachables(context.Background(), nil, nil)
	t.Cleanup(cleanup)
	require.NoError(t, tokenAuthority(attachables))
	cleanup()

	require.Equal(t, dir, config.Dir())
	require.FileExists(t, filepath.Join(dir, ".token_seed"))
	require.Equal(t, before, tokenSeedDirs(t))
}

func TestBuildkitSessionAttachables_SeedsGoToPrivateDirWhenConfigDirUnwritable(t *testing.T) {
	isolateTokenSeedTest(t)

	unwritable := unwritableDockerConfigDir(t)
	config.SetDir(unwritable)
	before := tokenSeedDirs(t)

	attachables, cleanup := buildkitSessionAttachables(context.Background(), nil, nil)
	t.Cleanup(cleanup)
	require.NoError(t, tokenAuthority(attachables))

	require.Equal(t, unwritable, config.Dir())
	require.NoFileExists(t, filepath.Join(unwritable, ".token_seed"))

	created := newTokenSeedDirs(before, tokenSeedDirs(t))
	require.Len(t, created, 1)
	info, err := os.Stat(created[0])
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o700), info.Mode().Perm())
	require.FileExists(t, filepath.Join(created[0], ".token_seed"))

	cleanup()
	require.NoDirExists(t, created[0])
}

func TestBuildkitSessionAttachables_EveryBuildGetsItsOwnSeedDir(t *testing.T) {
	isolateTokenSeedTest(t)

	config.SetDir(unwritableDockerConfigDir(t))
	before := tokenSeedDirs(t)

	const builds = 8
	var wg sync.WaitGroup
	errs := make([]error, builds)
	cleanups := make([]func(), builds)
	for i := range builds {
		wg.Add(1)
		go func() {
			defer wg.Done()
			attachables, cleanup := buildkitSessionAttachables(context.Background(), nil, nil)
			errs[i] = tokenAuthority(attachables)
			cleanups[i] = cleanup
		}()
	}
	wg.Wait()
	for _, cleanup := range cleanups {
		t.Cleanup(cleanup)
	}

	for _, err := range errs {
		require.NoError(t, err)
	}
	require.Len(t, newTokenSeedDirs(before, tokenSeedDirs(t)), builds)

	for _, cleanup := range cleanups {
		cleanup()
	}
	require.Equal(t, before, tokenSeedDirs(t))
}

func TestBuildWithBuildkitClient_RemovesTokenSeedsAfterTheBuild(t *testing.T) {
	isolateTokenSeedTest(t)

	config.SetDir(unwritableDockerConfigDir(t))
	before := tokenSeedDirs(t)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())

	ctx := logboek.NewContext(context.Background(), logboek.DefaultLogger())
	bkClient, err := bkclient.New(ctx, "tcp://"+address)
	require.NoError(t, err)
	defer bkClient.Close()

	err = buildWithBuildkitClient(ctx, bkClient, ".trdl/Dockerfile", nil, io.NopCloser(bytes.NewReader(nil)), nopWriteCloser{io.Discard}, hclog.NewNullLogger())
	require.Error(t, err)
	require.Equal(t, before, tokenSeedDirs(t))
}

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

// isolateTokenSeedTest keeps the test off the developer's docker config,
// credential helpers and shared temp dir, and restores config.Dir() afterwards.
func isolateTokenSeedTest(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	t.Setenv("PATH", "")
	t.Setenv("DOCKER_AUTH_CONFIG", "")
	original := config.Dir()
	t.Cleanup(func() { config.SetDir(original) })
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

func unwritableDockerConfigDir(t *testing.T) string {
	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	require.NoError(t, os.WriteFile(blocker, nil, 0o644))
	return filepath.Join(blocker, ".docker")
}

func tokenSeedDirs(t *testing.T) map[string]bool {
	dirs := map[string]bool{}
	for _, base := range []string{"/dev/shm", os.TempDir()} {
		matches, err := filepath.Glob(filepath.Join(base, "trdl-docker-config-*"))
		require.NoError(t, err)
		for _, match := range matches {
			dirs[match] = true
		}
	}
	return dirs
}

func newTokenSeedDirs(before, after map[string]bool) []string {
	var created []string
	for dir := range after {
		if !before[dir] {
			created = append(created, dir)
		}
	}
	return created
}
