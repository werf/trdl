package testutil

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

func GetTempDir() string {
	dir, err := ioutil.TempDir("", "trdl-e2e-tests-")
	Expect(err).ShouldNot(HaveOccurred())

	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		dir, err = filepath.EvalSymlinks(dir)
		Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("eval symlinks of path %s failed: %s", dir, err))
	}

	return dir
}

func ComputeTrdlBinPath() string {
	binPath := os.Getenv("TRDL_TEST_BINARY_PATH")
	if binPath == "" {
		var err error
		binPath, err = gexec.Build("github.com/werf/trdl/client/cmd/trdl")
		Expect(err).ShouldNot(HaveOccurred())
	}

	return binPath
}

func TrdlBinArgs(userArgs ...string) []string {
	var args []string
	if os.Getenv("TRDL_TEST_BINARY_PATH") != "" && os.Getenv("TRDL_TEST_COVERAGE_DIR") != "" {
		coverageFilePath := filepath.Join(
			os.Getenv("TRDL_TEST_COVERAGE_DIR"),
			fmt.Sprintf("%s-%s.out", strconv.FormatInt(time.Now().UTC().UnixNano(), 10), GetRandomString(10)),
		)
		args = append(args, fmt.Sprintf("-test.coverprofile=%s", coverageFilePath))
	}

	args = append(args, userArgs...)

	return args
}

func isTrdlTestBinaryPath(path string) bool {
	werfTestBinaryPath := os.Getenv("TRDL_TEST_BINARY_PATH")
	return werfTestBinaryPath != "" && werfTestBinaryPath == path
}

func FixturePath(paths ...string) string {
	absFixturesPath, err := filepath.Abs("_fixtures")
	Expect(err).ShouldNot(HaveOccurred())
	pathsToJoin := append([]string{absFixturesPath}, paths...)
	return filepath.Join(pathsToJoin...)
}

func MeetsRequirementTools(requiredSuiteTools []string) bool {
	hasRequirements := true
	for _, tool := range requiredSuiteTools {
		_, err := exec.LookPath(tool)
		if err != nil {
			fmt.Printf("You must have %s installed on your PATH\n", tool)
			hasRequirements = false
		}
	}

	return hasRequirements
}
