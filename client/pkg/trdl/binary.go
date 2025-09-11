package trdl

import (
	"fmt"
	"os"
	"path/filepath"
)

func GetTrdlBinaryPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("unable to determine trdl binary path: %w", err)
	}

	realExe, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("unable to resolve symlinks for %s: %w", exe, err)
	}

	return realExe, nil
}
