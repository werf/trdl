package util

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"
)

func Exec(path string, args []string) error {
	if runtime.GOOS == "windows" {
		cmd := exec.Command(path, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Run(); err != nil {
			return err
		}

		os.Exit(0)
	}

	args = append([]string{path}, args...)
	err := syscall.Exec(path, args, os.Environ())
	if err != nil {
		return fmt.Errorf("unable to exec path %q: %s", path, err)
	}

	return nil
}
