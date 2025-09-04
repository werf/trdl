package trdl

import (
	"fmt"
	"os"
)

func GetTrdlBinaryPath() (string, error) {
	trdlBinaryPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("unable to determine trdl binary path: %w", err)
	}
	return trdlBinaryPath, nil
}
