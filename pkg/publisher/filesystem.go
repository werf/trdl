package publisher

import (
	"context"
	"io"
)

type Filesystem interface {
	IsFileExist(ctx context.Context, path string) (bool, error)

	ReadFile(ctx context.Context, path string, writer io.WriterAt) error
	ReadFileStream(ctx context.Context, path string, writer io.Writer) error
	ReadFileBytes(ctx context.Context, path string) ([]byte, error)

	WriteFileBytes(ctx context.Context, path string, data []byte) error
	WriteFileStream(ctx context.Context, path string, reader io.Reader) error
}
