package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/hashicorp/go-hclog"
	"github.com/theupdateframework/go-tuf"
	"github.com/theupdateframework/go-tuf/data"
	"github.com/theupdateframework/go-tuf/sign"

	"github.com/werf/trdl/server/pkg/util"
)

type S3Options struct {
	AwsConfig  *aws.Config
	BucketName string
}

type TufRepoOptions struct {
	PrivKeys TufRepoPrivKeys
}

func NewRepositoryWithOptions(s3Options S3Options, tufRepoOptions TufRepoOptions, logger hclog.Logger) (*S3Repository, error) {
	s3fs := NewS3Filesystem(s3Options.AwsConfig, s3Options.BucketName, logger)
	tufStore := NewNonAtomicTufStore(tufRepoOptions.PrivKeys, s3fs, logger)
	tufRepo, err := tuf.NewRepo(tufStore)
	if err != nil {
		return nil, fmt.Errorf("error initializing tuf repo: %w", err)
	}

	return NewRepository(s3fs, tufStore, tufRepo, logger), nil
}

type S3Repository struct {
	S3Filesystem *S3Filesystem
	TufStore     *NonAtomicTufStore
	TufRepo      *tuf.Repo

	logger hclog.Logger
}

func NewRepository(s3Filesystem *S3Filesystem, tufStore *NonAtomicTufStore, tufRepo *tuf.Repo, logger hclog.Logger) *S3Repository {
	return &S3Repository{
		S3Filesystem: s3Filesystem,
		TufStore:     tufStore,
		TufRepo:      tufRepo,
		logger:       logger,
	}
}

func (repository *S3Repository) SetPrivKeys(privKeys TufRepoPrivKeys) error {
	repository.logger.Debug("-- S3Repository.SetPrivKeys")

	repository.TufStore.PrivKeys = privKeys
	repository.logger.Debug(fmt.Sprintf("-- S3Repository.SetPrivKeys BEFORE AddPrivateKeyWithExpires: %#v\n", repository.TufStore.PrivKeys))

	for _, desc := range []struct {
		role string
		key  *sign.PrivateKey
	}{
		{"root", privKeys.Root},
		{"targets", privKeys.Targets},
		{"snapshot", privKeys.Snapshot},
		{"timestamp", privKeys.Timestamp},
	} {
		if err := repository.TufRepo.AddPrivateKeyWithExpires(desc.role, desc.key, data.DefaultExpires("root")); err != nil {
			return fmt.Errorf("unable to add tuf repository private key for role %s: %w", desc.role, err)
		}
	}

	repository.logger.Debug(fmt.Sprintf("-- S3Repository.SetPrivKeys AFTER AddPrivateKeyWithExpires: %#v\n", repository.TufStore.PrivKeys))

	return nil
}

func (repository *S3Repository) GetPrivKeys() TufRepoPrivKeys {
	return repository.TufStore.PrivKeys
}

func (repository *S3Repository) GenPrivKeys() error {
	if _, err := repository.TufRepo.GenKey("root"); err != nil {
		return fmt.Errorf("error generating tuf repository root key: %w", err)
	}

	if _, err := repository.TufRepo.GenKey("targets"); err != nil {
		return fmt.Errorf("error generating tuf repository targets key: %w", err)
	}

	if _, err := repository.TufRepo.GenKey("snapshot"); err != nil {
		return fmt.Errorf("error generating tuf repository snapshot key: %w", err)
	}

	if _, err := repository.TufRepo.GenKey("timestamp"); err != nil {
		return fmt.Errorf("error generating tuf repository timestamp key: %w", err)
	}

	return nil
}

func (repository *S3Repository) RotatePrivKeys(ctx context.Context) (bool, TufRepoPrivKeys, error) {
	// TODO: Check priv keys expiration and generate new keys when necessary.

	return false, TufRepoPrivKeys{}, nil
}

func (repository *S3Repository) Init() error {
	err := repository.TufRepo.Init(false)

	if err == tuf.ErrInitNotAllowed {
		repository.logger.Info("Tuf repository already initialized: skip initialization")
	} else if err != nil {
		return fmt.Errorf("unable to init tuf repository: %w", err)
	}

	return nil
}

func (repository *S3Repository) StageTarget(ctx context.Context, pathInsideTargets string, data io.Reader) error {
	if err := repository.TufStore.StageTargetFile(ctx, pathInsideTargets, data); err != nil {
		return fmt.Errorf("unable to add staged file %q: %w", pathInsideTargets, err)
	}

	if err := repository.TufRepo.AddTarget(pathInsideTargets, json.RawMessage("")); err != nil {
		return fmt.Errorf("unable to register target file %q in the tuf repo: %w", pathInsideTargets, err)
	}

	return nil
}

func (repository *S3Repository) UpdateTimestamps(_ context.Context, systemClock util.Clock) error {
	return NewTufRepoRotator(repository.TufRepo).Rotate(repository.logger, systemClock.Now())
}

func (repository *S3Repository) CommitStaged(_ context.Context) error {
	if err := repository.TufRepo.Snapshot(tuf.CompressionTypeNone); err != nil {
		return fmt.Errorf("tuf repo snapshot failed: %w", err)
	}
	if err := repository.TufRepo.Timestamp(); err != nil {
		return fmt.Errorf("tuf repo timestamp failed: %w", err)
	}
	if err := repository.TufRepo.Commit(); err != nil {
		return fmt.Errorf("unable to commit staged changes into the repo: %w", err)
	}
	return nil
}

func (repository *S3Repository) GetTargets(ctx context.Context) ([]string, error) {
	targetsMeta, err := repository.TufRepo.Targets()
	if err != nil {
		return nil, fmt.Errorf("unable to get TUF-repo targets metadata: %w", err)
	}

	var res []string
	for path := range targetsMeta {
		res = append(res, path)
	}
	return res, nil
}
