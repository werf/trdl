//go:build ai_tests

package docker

import (
	"context"
	"testing"

	"github.com/djherbis/buffer"
	"github.com/djherbis/nio/v3"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The build stops at builder creation, before any docker invocation, so the
// error proves that the configured driver traveled from the release options
// all the way into the buildx arguments.
func TestAI_BuildReleaseArtifacts_ForwardsConfiguredDriver(t *testing.T) {
	clearDriverOptsEnv(t)
	t.Setenv(buildxDriverEnv, "kubernetes")

	gitRepo, err := git.Init(memory.NewStorage(), memfs.New())
	require.NoError(t, err)

	_, tarWriter := nio.Pipe(buffer.New(1024))

	err = BuildReleaseArtifacts(context.Background(), BuildReleaseArtifactsOpts{
		FromImage:    "alpine",
		RunCommands:  []string{"true"},
		GitRepo:      gitRepo,
		TarWriter:    tarWriter,
		Storage:      &logical.InmemStorage{},
		BuildxDriver: "docker",
	}, hclog.NewNullLogger())

	require.Error(t, err)
	assert.Contains(t, err.Error(), buildxDriverConfigurationSource)
	assert.Contains(t, err.Error(), `"docker"`)
}
