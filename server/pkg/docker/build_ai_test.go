//go:build ai_tests

package docker

import (
	"context"
	"io"
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

// The error proves that the configured driver traveled from the release options
// all the way into the buildx arguments, beating the environment on the way.
// The build is expected to stop at driver validation, before any docker
// invocation, but nothing in the production code guarantees that, so an empty
// PATH keeps a regression from provisioning a real builder and the artifacts
// pipe is drained so it cannot deadlock the test binary either.
func TestAI_BuildReleaseArtifacts_ForwardsConfiguredDriver(t *testing.T) {
	clearDriverOptsEnv(t)
	t.Setenv(buildxDriverEnv, "kubernetes")
	t.Setenv("PATH", t.TempDir())

	gitRepo, err := git.Init(memory.NewStorage(), memfs.New())
	require.NoError(t, err)

	tarReader, tarWriter := nio.Pipe(buffer.New(1024))
	go func() { _, _ = io.Copy(io.Discard, tarReader) }()

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
