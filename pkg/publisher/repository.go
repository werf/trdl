package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/theupdateframework/go-tuf"
)

type S3Options struct {
	AwsConfig  *aws.Config
	BucketName string
}

type TufRepoOptions struct {
	PrivKeys TufRepoPrivKeys
}

func NewRepositoryWithOptions(s3Options S3Options, tufRepoOptions TufRepoOptions) (*S3Repository, error) {
	s3fs := NewS3Filesystem(s3Options.AwsConfig, s3Options.BucketName)
	tufStore := NewNonAtomicTufStore(tufRepoOptions.PrivKeys, s3fs)
	tufRepo, err := tuf.NewRepo(tufStore)
	if err != nil {
		return nil, fmt.Errorf("error initializing tuf repo: %s", err)
	}

	return NewRepository(s3fs, tufStore, tufRepo), nil
}

type S3Repository struct {
	S3Filesystem *S3Filesystem
	TufStore     *NonAtomicTufStore
	TufRepo      *tuf.Repo
}

func NewRepository(s3Filesystem *S3Filesystem, tufStore *NonAtomicTufStore, tufRepo *tuf.Repo) *S3Repository {
	return &S3Repository{
		S3Filesystem: s3Filesystem,
		TufStore:     tufStore,
		TufRepo:      tufRepo,
	}
}

func (repository *S3Repository) PublishTarget(ctx context.Context, pathInsideTargets string, data io.Reader) error {
	if err := repository.TufStore.StageTargetFile(ctx, pathInsideTargets, data); err != nil {
		return fmt.Errorf("unable to add staged file %q: %s", pathInsideTargets, err)
	}

	if err := repository.TufRepo.AddTarget(pathInsideTargets, json.RawMessage("")); err != nil {
		return fmt.Errorf("unable to register target file %q in the tuf repo: %s", pathInsideTargets, err)
	}

	return nil
}

func (repository *S3Repository) Commit(ctx context.Context) error {
	if err := repository.TufRepo.Snapshot(tuf.CompressionTypeNone); err != nil {
		return fmt.Errorf("tuf repo snapshot failed: %s", err)
	}

	if err := repository.TufRepo.Timestamp(); err != nil {
		return fmt.Errorf("tuf repo timestamp failed: %s", err)
	}

	if err := repository.TufRepo.Commit(); err != nil {
		return fmt.Errorf("unable to commit staged changes into the repo: %s", err)
	}

	return nil
}
