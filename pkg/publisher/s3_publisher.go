package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/theupdateframework/go-tuf"
)

type S3Options struct {
	AwsConfig  *aws.Config
	BucketName string
}

type TufRepoOptions struct { // KEYS!
	PrivKeys TufRepoPrivKeys
}

func PublishSnapshotIntoS3(ctx context.Context, files SnapshotFilesIterator, tufRepoOptions TufRepoOptions, s3Options S3Options) error {
	s3fs := NewS3Filesystem(s3Options.AwsConfig, s3Options.BucketName)
	tufStore := NewNonAtomicTufStore(tufRepoOptions.PrivKeys, s3fs)
	tufRepo, err := tuf.NewRepo(tufStore)
	if err != nil {
		return fmt.Errorf("error initializing tuf repo: %s", err)
	}

	// TODO: detect empty tuf repo, perform proper initialization of root.json and other's roles metadata files
	if err := tufRepo.Init(false); err != nil {
		return fmt.Errorf("unable to init repo: %s", err)
	}

	for {
		pathInsideTargets, data, err := files.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("error iterating files: %s", err)
		}

		path := filepath.Join("targets", pathInsideTargets)

		if err := tufStore.AddStagedFile(path, data); err != nil {
			return fmt.Errorf("unable to add staged file %q: %s", path, err)
		}

		if err := tufRepo.AddTarget(path, json.RawMessage("")); err != nil {
			return fmt.Errorf("unable to register target file %q in the tuf repo: %s", path, err)
		}
	}

	if err := tufRepo.Snapshot(tuf.CompressionTypeNone); err != nil {
		return fmt.Errorf("tuf repo snapshot failed: %s", err)
	}

	if err := tufRepo.Timestamp(); err != nil {
		return fmt.Errorf("tuf repo timestamp failed: %s", err)
	}

	if err := tufRepo.Commit(); err != nil {
		return fmt.Errorf("unable to commit staged changes into the repo: %s", err)
	}

	return nil
}
