package client

import (
	"path/filepath"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/werf/trdl/server/pkg/testutil"
)

var _ = Describe("Version pinning", func() {
	When("repo added", func() {
		BeforeEach(func() {
			testutil.RunSucceedCommand(
				"",
				trdlBinPath,
				"add", "-d", testRepoName, validRepoUrl, validRootVersion, validRootSHA512,
			)
		})

		DescribeTable("should update to a version matching the selector",
			func(selector, expectedRelease string) {
				testutil.RunSucceedCommand(
					"",
					trdlBinPath,
					"update", testRepoName, selector,
				)

				output := testutil.SucceedCommandOutputString(
					"",
					trdlBinPath,
					"bin-path", testRepoName, selector,
				)
				Expect(output).Should(Equal(releaseBinDir(expectedRelease) + "\n"))
			},
			Entry("exact version", "v0.0.1", "v0.0.1"),
			Entry("exact latest version", "v0.0.2", "v0.0.2"),
			Entry("range >= resolves to the greatest match", ">=0.0.1", "v0.0.2"),
			Entry("range < excludes the greatest", "<0.0.2", "v0.0.1"),
			Entry("tilde constraint", "~0.0.1", "v0.0.2"),
			Entry("caret constraint", "^0.0.1", "v0.0.1"),
		)

		When("updated to a pinned version", func() {
			BeforeEach(func() {
				testutil.RunSucceedCommand(
					"",
					trdlBinPath,
					"update", testRepoName, "v0.0.1",
				)
			})

			It("bin-path", func() {
				output := testutil.SucceedCommandOutputString(
					"",
					trdlBinPath,
					"bin-path", testRepoName, "v0.0.1",
				)
				Expect(output).Should(Equal(releaseBinDir("v0.0.1") + "\n"))
			})

			It("dir-path", func() {
				output := testutil.SucceedCommandOutputString(
					"",
					trdlBinPath,
					"dir-path", testRepoName, "v0.0.1",
				)
				Expect(output).Should(Equal(releaseDir("v0.0.1") + "\n"))
			})

			It("exec", func() {
				args := []string{"exec", testRepoName, "v0.0.1"}
				if runtime.GOOS == "windows" {
					args = append(args, "script.bat")
				}

				output := testutil.SucceedCommandOutputString(
					"",
					trdlBinPath,
					args...,
				)

				if runtime.GOOS == "windows" {
					Expect(output).Should(Equal("\"v0.0.1\"\r\n"))
				} else {
					Expect(output).Should(Equal("v0.0.1\n"))
				}
			})
		})

		It("should reject an invalid version selector on exec", func() {
			_, err := testutil.RunCommandWithOptions(
				"",
				trdlBinPath,
				[]string{"exec", testRepoName, "v#bad"},
				testutil.RunCommandOptions{ShouldSucceed: false},
			)
			Expect(err).Should(HaveOccurred())
		})

		It("should fail to exec a version that has not been pinned locally", func() {
			output, err := testutil.RunCommandWithOptions(
				"",
				trdlBinPath,
				[]string{"exec", testRepoName, "v0.0.2"},
				testutil.RunCommandOptions{ShouldSucceed: false},
			)
			Expect(err).Should(HaveOccurred())
			Expect(string(output)).Should(ContainSubstring("not found locally"))
		})

		It("should preserve a pinned version across an autoclean update", func() {
			testutil.RunSucceedCommand(
				"",
				trdlBinPath,
				"update", testRepoName, "v0.0.1",
			)

			testutil.RunSucceedCommand(
				"",
				trdlBinPath,
				"update", testRepoName, "v0.0.2", "--autoclean",
			)

			By("the older pinned version is still resolvable")
			output := testutil.SucceedCommandOutputString(
				"",
				trdlBinPath,
				"bin-path", testRepoName, "v0.0.1",
			)
			Expect(output).Should(Equal(releaseBinDir("v0.0.1") + "\n"))
		})
	})
})

func releaseDir(release string) string {
	osArch := "any-any"
	if runtime.GOOS == "windows" {
		osArch = "windows-any"
	}

	return filepath.Join(trdlHomeDir, "repositories/test/releases", release, osArch)
}

func releaseBinDir(release string) string {
	return filepath.Join(releaseDir(release), "bin")
}
