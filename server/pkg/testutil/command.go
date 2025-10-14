package testutil

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func RunSucceedCommand(dir, command string, args ...string) {
	_, _ = RunCommandWithOptions(dir, command, args, RunCommandOptions{ShouldSucceed: true})
}

func SucceedCommandOutputString(dir, command string, args ...string) string {
	res, _ := RunCommandWithOptions(dir, command, args, RunCommandOptions{ShouldSucceed: true})
	return string(res)
}

type RunCommandOptions struct {
	ExtraEnv      []string
	ToStdin       string
	ShouldSucceed bool
}

func RunCommandWithOptions(dir, command string, args []string, options RunCommandOptions) ([]byte, error) {
	if isTrdlTestBinaryPath(command) {
		args = TrdlBinArgs(args...)
	}

	cmd := exec.Command(command, args...)

	if dir != "" {
		cmd.Dir = dir
	}

	if len(options.ExtraEnv) != 0 {
		cmd.Env = append(os.Environ(), options.ExtraEnv...)
	}

	if options.ToStdin != "" {
		var b bytes.Buffer
		b.Write([]byte(options.ToStdin))
		cmd.Stdin = &b
	}

	res, err := cmd.CombinedOutput()
	_, _ = GinkgoWriter.Write(res)

	if options.ShouldSucceed {
		errorDesc := fmt.Sprintf("%[2]s %[3]s (dir: %[1]s)", dir, command, strings.Join(args, " "))
		Expect(err).ShouldNot(HaveOccurred(), errorDesc)
	}

	return res, err
}
