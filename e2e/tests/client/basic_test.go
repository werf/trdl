package client

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/werf/trdl/client/pkg/trdl"
	"github.com/werf/trdl/server/pkg/testutil"
)

var _ = Describe("Basic", func() {
	It("add", func() {
		testutil.RunSucceedCommand(
			"",
			trdlBinPath, "--debug",
			"add", testRepoName, validRepoUrl, validRootVersion, validRootSHA512,
		)

		AssertRepoFieldsInListOutput([]string{testRepoName, validRepoUrl, trdl.DefaultChannel})
	})

	When("repo added", func() {
		BeforeEach(func() {
			testutil.RunSucceedCommand(
				"",
				trdlBinPath, "--debug",
				"add", testRepoName, validRepoUrl, validRootVersion, validRootSHA512,
			)
		})

		It("list", func() {
			output := testutil.SucceedCommandOutputString(
				"",
				trdlBinPath,
				"list",
			)

			Ω(output).Should(ContainSubstring(fmt.Sprintf("%s  %s  %s", testRepoName, validRepoUrl, trdl.DefaultChannel)))
		})

		It("set-default-channel", func() {
			testutil.RunSucceedCommand(
				"",
				trdlBinPath,
				"set-default-channel", testRepoName, "beta",
			)

			AssertRepoFieldsInListOutput([]string{testRepoName, validRepoUrl, "beta"})
		})

		It("update", func() {
			testutil.RunSucceedCommand(
				"",
				trdlBinPath,
				"update", testRepoName, validGroup,
			)
		})

		It("self-update", func() {
			trdlBinPathToOverride := filepath.Join(tmpDir, filepath.Base(trdlBinPath))
			testutil.CopyIn(trdlBinPath, trdlBinPathToOverride)

			By("check dev trdl version")
			{
				output := testutil.SucceedCommandOutputString(
					"",
					trdlBinPathToOverride,
					"version",
				)
				version := strings.TrimSpace(output)
				Ω(version).Should(Equal(trdlBinVersion))
			}

			By("run update command")
			{
				stubs.SetEnv("TRDL_NO_SELF_UPDATE", "0")
				testutil.RunSucceedCommand(
					"",
					trdlBinPathToOverride,
					"update", testRepoName, validGroup,
				)
			}

			By("check changed trdl version")
			{
				output := testutil.SucceedCommandOutputString(
					"",
					trdlBinPathToOverride,
					"version",
				)
				version := strings.TrimSpace(output)
				Ω(version).ShouldNot(Equal(trdlBinVersion))
			}
		})

		When("stable channel updated", func() {
			BeforeEach(func() {
				testutil.RunSucceedCommand(
					"",
					trdlBinPath,
					"update", testRepoName, validGroup,
				)
			})

			It("bin-path", func() {
				output := testutil.SucceedCommandOutputString(
					"",
					trdlBinPath,
					"bin-path", testRepoName, validGroup,
				)

				if runtime.GOOS == "windows" {
					Ω(output).Should(Equal(filepath.Join(trdlHomeDir, "repositories/test/releases/v0.0.1/windows-any/bin") + "\n"))
				} else {
					Ω(output).Should(Equal(filepath.Join(trdlHomeDir, "repositories/test/releases/v0.0.1/any-any/bin") + "\n"))
				}
			})

			It("dir-path", func() {
				output := testutil.SucceedCommandOutputString(
					"",
					trdlBinPath,
					"dir-path", testRepoName, validGroup,
				)

				if runtime.GOOS == "windows" {
					Ω(output).Should(Equal(filepath.Join(trdlHomeDir, "repositories/test/releases/v0.0.1/windows-any") + "\n"))
				} else {
					Ω(output).Should(Equal(filepath.Join(trdlHomeDir, "repositories/test/releases/v0.0.1/any-any") + "\n"))
				}
			})

			It("exec", func() {
				args := []string{"exec", testRepoName, validGroup}
				if runtime.GOOS == "windows" {
					args = append(args, "script.bat")
				}

				output := testutil.SucceedCommandOutputString(
					"",
					trdlBinPath,
					args...,
				)

				if runtime.GOOS == "windows" {
					Ω(output).Should(Equal("\"v0.0.1\"\r\n"))
				} else {
					Ω(output).Should(Equal("v0.0.1\n"))
				}
			})
		})
	})
})

func AssertRepoFieldsInListOutput(expectedFields []string) {
	expectedOutput := testutil.SucceedCommandOutputString(
		"",
		trdlBinPath,
		"list",
	)

	lines := strings.Split(expectedOutput, "\n")
	Ω(len(lines)).Should(Equal(3))
	Ω(strings.Fields(lines[1])).Should(Equal(expectedFields))
}
