//go:build darwin || linux
// +build darwin linux

package util

import (
	"fmt"
	"os"
	"syscall"
)

func Exec(path string, args []string) error {
	args = append([]string{path}, args...)
	err := syscall.Exec(path, args, os.Environ())
	if err != nil {
		return fmt.Errorf("unable to exec path %q: %s", path, err)
	}

	return nil
}
