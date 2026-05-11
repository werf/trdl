package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func IsDirExist(path string) (bool, error) {
	fileInfo, err := os.Lstat(path)
	if err != nil {
		if isNotExistErr(err) {
			return false, nil
		}

		return false, err
	}

	return fileInfo.IsDir(), nil
}

func IsRegularFileExist(path string) (bool, error) {
	fileInfo, err := os.Lstat(path)
	if err != nil {
		if isNotExistErr(err) {
			return false, nil
		}

		return false, err
	}

	return fileInfo.Mode().IsRegular(), nil
}

// AtomicWriteFile writes data to dst atomically by first writing to a temporary
// file in tmpDir, then renaming. This is safe for concurrent use across processes.
func AtomicWriteFile(dst string, data []byte, perm os.FileMode, tmpDir string) error {
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return fmt.Errorf("create tmp dir %q: %w", tmpDir, err)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create dst dir %q: %w", filepath.Dir(dst), err)
	}

	f, err := os.CreateTemp(tmpDir, filepath.Base(dst)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := f.Name()
	defer os.Remove(tmpPath)

	if _, err := f.Write(data); err != nil {
		f.Close()
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := f.Chmod(perm); err != nil {
		f.Close()
		return fmt.Errorf("chmod temp file: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, dst); err != nil {
		return fmt.Errorf("rename temp file to %q: %w", dst, err)
	}

	return nil
}

func isNotExistErr(err error) bool {
	return os.IsNotExist(err) || IsNotDirectoryErr(err)
}

func IsNotDirectoryErr(err error) bool {
	return strings.HasSuffix(err.Error(), "not a directory")
}
