package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/hashicorp/go-hclog"
	"github.com/theupdateframework/go-tuf"

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

	repository := NewRepository(s3fs, tufStore, tufRepo, logger)

	if err := tufStore.PrivKeys.SetupStoreSigners(tufStore); err != nil {
		return nil, fmt.Errorf("unable to set private keys into tuf store: %w", err)
	}

	return repository, nil
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

	repository.logger.Debug(fmt.Sprintf("-- S3Repository.SetPrivKeys BEFORE AddPrivateKeyWithExpires: %#v\n", privKeys))

	if err := privKeys.SetupStoreSigners(repository.TufStore); err != nil {
		return fmt.Errorf("unable to set private keys into tuf store: %w", err)
	}

	if err := privKeys.SetupTufRepoSigners(repository.TufRepo); err != nil {
		return fmt.Errorf("unable to set private keys into tuf repo: %w", err)
	}

	repository.logger.Debug(fmt.Sprintf("-- S3Repository.SetPrivKeys AFTER AddPrivateKeyWithExpires: %#v\n", privKeys))

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
	if err := repository.TufRepo.Snapshot(); err != nil {
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

type RootMeta struct {
	Signed Signed `json:"signed"`
}
type Signed struct {
	Keys  map[string]Key            `json:"keys"`
	Roles map[string]RoleDefinition `json:"roles"`
}

type Key struct {
	KeyVal KeyVal `json:"keyval"`
}

type KeyVal struct {
	Public string `json:"public"`
}

type RoleDefinition struct {
	KeyIDs []string `json:"keyids"`
}

func (repository *S3Repository) GetRolePublicKeysFromS3Meta(file, role string) ([]string, error) {
	meta, err := repository.TufRepo.GetMeta()
	if err != nil {
		return nil, fmt.Errorf("error getting metadata from TUF repo: %w", err)
	}

	var rootMeta RootMeta
	if err := json.Unmarshal(meta[file], &rootMeta); err != nil {
		return nil, fmt.Errorf("error unmarshalling %s: %w", file, err)
	}

	rootRole, ok := rootMeta.Signed.Roles[role]
	if !ok {
		return nil, nil
	}

	publicKeys := make([]string, 0, len(rootRole.KeyIDs))
	for _, keyID := range rootRole.KeyIDs {
		key, ok := rootMeta.Signed.Keys[keyID]
		if !ok {
			return nil, fmt.Errorf("key %q not found in keys section", keyID)
		}
		publicKeys = append(publicKeys, key.KeyVal.Public)
	}

	return publicKeys, nil
}
