package publisher

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
)

type InMemoryFile struct {
	Name string
	Data []byte
}

func StageReleaseFile(ctx context.Context, repository *S3Repository, releaseName, path string, data io.Reader) error {
	return repository.PublishTarget(ctx, filepath.Join(releaseName, path), data)
}

func StageInMemoryFiles(ctx context.Context, repository *S3Repository, files []*InMemoryFile) error {
	for _, file := range files {
		if err := repository.PublishTarget(ctx, file.Name, bytes.NewReader(file.Data)); err != nil {
			return fmt.Errorf("error publishing %q: %s", file.Name, err)
		}
	}
	return nil
}
