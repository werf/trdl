package publisher

import (
	"context"
	"io"

	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/trdl/server/pkg/config"
	"github.com/werf/trdl/server/pkg/util"
)

type Interface interface {
	GetRepository(ctx context.Context, storage logical.Storage, options RepositoryOptions) (RepositoryInterface, error)
	RotateRepositoryKeys(ctx context.Context, storage logical.Storage, repository RepositoryInterface, systemClock util.Clock) error
	UpdateTimestamps(ctx context.Context, storage logical.Storage, repository RepositoryInterface, systemClock util.Clock) error
	StageReleaseTarget(ctx context.Context, repository RepositoryInterface, releaseName, path string, data io.Reader) error
	StageChannelsConfig(ctx context.Context, repository RepositoryInterface, trdlChannelsConfig *config.TrdlChannels) error
	StageInMemoryFiles(ctx context.Context, repository RepositoryInterface, files []*InMemoryFile) error
	GetExistingReleases(ctx context.Context, repository RepositoryInterface) ([]string, error)
}

type RepositoryInterface interface {
	Init() error
	SetPrivKeys(privKeys TufRepoPrivKeys) error
	GetPrivKeys() TufRepoPrivKeys
	GenPrivKeys() error
	RotatePrivKeys(ctx context.Context) (bool, TufRepoPrivKeys, error)
	UpdateTimestamps(ctx context.Context, systemClock util.Clock) error
	StageTarget(ctx context.Context, pathInsideTargets string, data io.Reader) error
	CommitStaged(ctx context.Context) error
	GetTargets(ctx context.Context) ([]string, error)
}
