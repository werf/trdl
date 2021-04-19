package publisher

import (
	"fmt"
	"io"
)

type SnapshotFilesIterator interface {
	Next() (string, []byte, error)
}

func ForEachSnapshotFile(files SnapshotFilesIterator, f func(string, []byte) error) error {
	for {
		path, data, err := files.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("error getting next snapshot file: %s", err)
		}

		err = f(path, data)

		if err != nil {
			return err
		}
	}

	return nil
}

type InMemoryFile struct {
	Name string
	Data []byte
}

type InMemorySnapshotFiles struct {
	Files []InMemoryFile
}

func NewInMemorySnapshotFiles() *InMemorySnapshotFiles {
	return &InMemorySnapshotFiles{}
}

func (sfiles *InMemorySnapshotFiles) Iterator() SnapshotFilesIterator {
	return &InMemorySnapshotFilesIterator{SnapshotFiles: sfiles}
}

type InMemorySnapshotFilesIterator struct {
	SnapshotFiles *InMemorySnapshotFiles
	pos           int
}

func (iter *InMemorySnapshotFilesIterator) Next() (string, []byte, error) {
	if iter.pos < len(iter.SnapshotFiles.Files) {
		f := iter.SnapshotFiles.Files[iter.pos]
		iter.pos++
		return f.Name, f.Data, nil
	}

	return "", nil, io.EOF
}

func (sfiles *InMemorySnapshotFiles) TarWriter() io.Writer {
	return &TarSnapshotWriter{SnapshotFiles: sfiles}
}

type TarSnapshotWriter struct {
	SnapshotFiles *InMemorySnapshotFiles
}

func (w *TarSnapshotWriter) Write(p []byte) (n int, err error) {
	// TODO: parse input data as tar archive, create FileDesc structs for each file
	return 0, nil
}
