package client

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/werf/trdl/client/pkg/trdl"
	"github.com/werf/trdl/server/pkg/testutil"
)

type useEntry struct {
	shell                string
	shellCommandName     string
	shellCommandArgsFunc func(testScriptPath string) []string
	testScriptFormat     string
	testScriptBasename   string
	expectedOutput       string
}

var _ = XDescribe("Use", func() {
	BeforeEach(func() {
		testutil.RunSucceedCommand(
			"",
			trdlBinPath,
			"add", testRepoName, validRepoUrl, validRootVersion, validRootSHA512,
		)
	})

	DescribeTable("should source script and run channel release binary", func(entry useEntry) {
		switch entry.shell {
		case trdl.ShellPowerShell:
			if runtime.GOOS != "windows" {
				Skip("windows test")
			}
		case trdl.ShellUnix:
			if runtime.GOOS == "windows" {
				Skip("unix test")
			}
		}

		shellCommandPath, err := exec.LookPath(entry.shellCommandName)
		Ω(err).ShouldNot(HaveOccurred())

		trdlUseCommand := strings.Join(append(
			[]string{trdlBinPath},
			testutil.TrdlBinArgs("use", testRepoName, validGroup, "--shell", entry.shell)...,
		), " ")

		testScriptPath := filepath.Join(tmpDir, entry.testScriptBasename)
		err = ioutil.WriteFile(testScriptPath, []byte(fmt.Sprintf(entry.testScriptFormat, trdlUseCommand)), os.ModePerm)
		Ω(err).ShouldNot(HaveOccurred())

		shellCommandArgs := entry.shellCommandArgsFunc(testScriptPath)

		By("Updating in foreground ...")
		output := testutil.SucceedCommandOutputString(
			"",
			shellCommandPath,
			shellCommandArgs...,
		)
		Ω(output).Should(Equal(entry.expectedOutput))

		By("Updating in background ...")
		output = testutil.SucceedCommandOutputString(
			"",
			shellCommandPath,
			shellCommandArgs...,
		)
		Ω(output).Should(Equal(entry.expectedOutput))

		// Wait for the update in background is done on windows to prevent error: ...\.locks\repositories\test\tuf\ea3bd73e2b506e00527232b3ed743c066da83a8e3066f62a71e75eb9b4aa1db6: The process cannot access the file because it is being used by another process".
		// Wait for the update in background is done on unix to prevent error: unlinkat /tmp/trdl-e2e-tests-932831490: directory not empty occurred
		time.Sleep(time.Millisecond * 500)
	},
		Entry(trdl.ShellPowerShell, useEntry{
			shell:            trdl.ShellPowerShell,
			shellCommandName: "powershell.exe",
			shellCommandArgsFunc: func(testScriptPath string) []string {
				return []string{"-command", testScriptPath}
			},
			testScriptFormat: `
$TRDL_USE_SCRIPT_PATH = %[1]s
. $TRDL_USE_SCRIPT_PATH.Trim()
script.bat
`,
			testScriptBasename: "test_script.ps1",
			expectedOutput:     "\"v0.0.1\"\r\n",
		}),
		Entry("sh", useEntry{
			shell:            trdl.ShellUnix,
			shellCommandName: "sh",
			shellCommandArgsFunc: func(testScriptPath string) []string {
				return []string{"-c", testScriptPath}
			},
			testScriptFormat: `
. $(%[1]s)
script.sh
`,
			testScriptBasename: "test_script.sh",
			expectedOutput:     "v0.0.1\n",
		}),
		Entry("bash", useEntry{
			shell:            trdl.ShellUnix,
			shellCommandName: "bash",
			shellCommandArgsFunc: func(testScriptPath string) []string {
				return []string{"-c", testScriptPath}
			},
			testScriptFormat: `
. $(%[1]s)
script.sh
`,
			testScriptBasename: "test_script.sh",
			expectedOutput:     "v0.0.1\n",
		}),
	)
})
