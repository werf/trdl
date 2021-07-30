package flow

import (
	"bytes"
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

	"github.com/werf/trdl/e2e/util"
	"github.com/werf/trdl/pkg/trdl"
)

type useEntry struct {
	shell                string
	shellCommandName     string
	shellCommandArgsFunc func(testScriptPath string) []string
	testScriptFormat     string
	testScriptBasename   string
	expectedOutput       string
}

var _ = Describe("Use", func() {
	BeforeEach(func() {
		util.RunSucceedCommand(
			"",
			trdlBinPath,
			"add", testRepoName, validRepoUrl, validRootVersion, validRootSHA512,
		)
	})

	// check
	DescribeTable("should print only shell script", func(shell string) {
		useAsFileOutput := util.SucceedCommandOutputString(
			"",
			trdlBinPath,
			"use", testRepoName, validGroup, "--as-file",
		)
		scriptPath := strings.TrimSpace(useAsFileOutput)

		output := util.SucceedCommandOutputString(
			"",
			trdlBinPath,
			"use", testRepoName, validGroup,
		)

		scriptData, err := ioutil.ReadFile(scriptPath)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(bytes.Equal(scriptData, []byte(output))).Should(BeTrue())
	},
		Entry(trdl.ShellCmd, trdl.ShellCmd),
		Entry(trdl.ShellPowerShell, trdl.ShellPowerShell),
		Entry(trdl.ShellUnix, trdl.ShellUnix),
	)

	DescribeTable("should source script and run channel release binary", func(entry useEntry) {
		switch entry.shell {
		case trdl.ShellCmd, trdl.ShellPowerShell:
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
			util.TrdlBinArgs("use", testRepoName, validGroup, "--as-file", "--shell", entry.shell)...,
		), " ")

		testScriptPath := filepath.Join(tmpDir, entry.testScriptBasename)
		err = ioutil.WriteFile(testScriptPath, []byte(fmt.Sprintf(entry.testScriptFormat, trdlUseCommand)), os.ModePerm)
		Ω(err).ShouldNot(HaveOccurred())

		shellCommandArgs := entry.shellCommandArgsFunc(testScriptPath)

		By("Updating in foreground ...")
		output := util.SucceedCommandOutputString(
			"",
			shellCommandPath,
			shellCommandArgs...,
		)
		Ω(output).Should(Equal(entry.expectedOutput))

		By("Updating in background ...") // TODO: change channel release on this step
		output = util.SucceedCommandOutputString(
			"",
			shellCommandPath,
			shellCommandArgs...,
		)
		Ω(output).Should(Equal(entry.expectedOutput))

		// Wait for the update in background is done on windows to prevent error: ...\.locks\repositories\test\tuf\ea3bd73e2b506e00527232b3ed743c066da83a8e3066f62a71e75eb9b4aa1db6: The process cannot access the file because it is being used by another process".
		if runtime.GOOS == "windows" {
			time.Sleep(time.Millisecond * 500)
		}
	},
		Entry(trdl.ShellCmd, useEntry{
			shell:            trdl.ShellCmd,
			shellCommandName: "cmd.exe",
			shellCommandArgsFunc: func(testScriptPath string) []string {
				return []string{"/C", fmt.Sprintf("CALL %s", testScriptPath)}
			},
			testScriptFormat: `
@echo off
FOR /F "tokens=*" %%%%g IN ('%[1]s') do (SET TRDL_USE_SCRIPT_PATH=%%%%g)
%%TRDL_USE_SCRIPT_PATH%% && script.bat
`,
			testScriptBasename: "test_script.bat",
			expectedOutput:     "\"v0.0.1\"\r\n",
		}),
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
