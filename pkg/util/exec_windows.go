package util

import (
	"fmt"
	"os"
	"os/exec"
)

func Exec(path string, args []string) error {
	cmd := exec.Command(path, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("unable to run command: %q", err)
	}

	os.Exit(0)

	return nil
}
