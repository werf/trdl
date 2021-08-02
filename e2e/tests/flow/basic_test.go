package flow

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/werf/trdl/e2e/util"
	"github.com/werf/trdl/pkg/trdl"
)

var _ = Describe("Basic", func() {
	It("add", func() {
		util.RunSucceedCommand(
			"",
			trdlBinPath,
			"add", testRepoName, validRepoUrl, validRootVersion, validRootSHA512,
		)

		AssertRepoFieldsInListOutput([]string{testRepoName, validRepoUrl, trdl.DefaultChannel})
	})

	When("repo added", func() {
		BeforeEach(func() {
			util.RunSucceedCommand(
				"",
				trdlBinPath,
				"add", testRepoName, validRepoUrl, validRootVersion, validRootSHA512,
			)
		})

		It("list", func() {
			output := util.SucceedCommandOutputString(
				"",
				trdlBinPath,
				"list",
			)

			Ω(output).Should(Equal(fmt.Sprintf(`Name  URL                                 Default Channel  
%s  %s  %s           
`, testRepoName, validRepoUrl, trdl.DefaultChannel)))
		})

		It("set-default-channel", func() {
			util.RunSucceedCommand(
				"",
				trdlBinPath,
				"set-default-channel", testRepoName, "beta",
			)

			AssertRepoFieldsInListOutput([]string{testRepoName, validRepoUrl, "beta"})
		})

		It("update", func() {
			util.RunSucceedCommand(
				"",
				trdlBinPath,
				"update", testRepoName, "v0",
			)
		})

		When("stable channel updated", func() {
			BeforeEach(func() {
				util.RunSucceedCommand(
					"",
					trdlBinPath,
					"add", testRepoName, validRepoUrl, validRootVersion, validRootSHA512,
				)

				util.RunSucceedCommand(
					"",
					trdlBinPath,
					"update", testRepoName, "v0",
				)
			})

			It("bin-path", func() {
				output := util.SucceedCommandOutputString(
					"",
					trdlBinPath,
					"bin-path", testRepoName, "v0",
				)

				if runtime.GOOS == "windows" {
					Ω(output).Should(Equal(filepath.Join(trdlHomeDir, "repositories/test/releases/v0.0.1/windows-any/bin") + "\n"))
				} else {
					Ω(output).Should(Equal(filepath.Join(trdlHomeDir, "repositories/test/releases/v0.0.1/any-any/bin") + "\n"))
				}
			})

			It("dir-path", func() {
				output := util.SucceedCommandOutputString(
					"",
					trdlBinPath,
					"dir-path", testRepoName, "v0",
				)

				if runtime.GOOS == "windows" {
					Ω(output).Should(Equal(filepath.Join(trdlHomeDir, "repositories/test/releases/v0.0.1/windows-any") + "\n"))
				} else {
					Ω(output).Should(Equal(filepath.Join(trdlHomeDir, "repositories/test/releases/v0.0.1/any-any") + "\n"))
				}
			})

			It("exec", func() {
				output := util.SucceedCommandOutputString(
					"",
					trdlBinPath,
					"exec", testRepoName, "v0",
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
	expectedOutput := util.SucceedCommandOutputString(
		"",
		trdlBinPath,
		"list",
	)

	lines := strings.Split(expectedOutput, "\n")
	Ω(len(lines)).Should(Equal(3))
	Ω(strings.Fields(lines[1])).Should(Equal(expectedFields))
}