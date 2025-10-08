package trdl

import (
	"fmt"
	"os"
	"path/filepath"
)

var (
	trdlBinaryPath    string
	trdlBinaryPathErr error
)

func init() {
	exe, err := os.Executable()
	if err != nil {
		trdlBinaryPathErr = fmt.Errorf("unable to determine trdl binary path: %w", err)
		return
	}

	realExe, err := filepath.EvalSymlinks(exe)
	if err != nil {
		trdlBinaryPath = exe
		return
	}

	trdlBinaryPath = realExe
}

func GetTrdlBinaryPath() (string, error) {
	return trdlBinaryPath, trdlBinaryPathErr
}
