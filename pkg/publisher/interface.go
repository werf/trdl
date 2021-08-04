package publisher

import (
	"context"
	"io"

	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/vault-plugin-secrets-trdl/pkg/config"
)

type Interface interface {
	GetRepository(ctx context.Context, storage logical.Storage, options RepositoryOptions) (RepositoryInterface, error)
	RotateRepositoryKeys(ctx context.Context, storage logical.Storage, repository RepositoryInterface) error
	UpdateTimestamps(ctx context.Context, storage logical.Storage, repository RepositoryInterface) error

	StageReleaseTarget(ctx context.Context, repository RepositoryInterface, releaseName, path string, data io.Reader) error
	StageChannelsConfig(ctx context.Context, repository RepositoryInterface, trdlChannelsConfig *config.TrdlChannels) error
	StageInMemoryFiles(ctx context.Context, repository RepositoryInterface, files []*InMemoryFile) error
}

type RepositoryInterface interface {
	Init() error
	SetPrivKeys(privKeys TufRepoPrivKeys) error
	GetPrivKeys() TufRepoPrivKeys
	GenPrivKeys() error

	RotatePrivKeys(ctx context.Context) (bool, TufRepoPrivKeys, error)
	UpdateTimestamps(ctx context.Context) error

	StageTarget(ctx context.Context, pathInsideTargets string, data io.Reader) error
	CommitStaged(ctx context.Context) error
}
