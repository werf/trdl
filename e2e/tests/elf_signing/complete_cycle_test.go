//go:build linux && amd64 && cgo

package elf_signing

import (
	_ "embed"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
)

type testOptions struct {
	projectName string
	branchName  string
	tag1        string
	version1    string
	pgpKeys     map[string]string
}

var _ = Describe("trdl ELF signing test", Label("e2e", "trdl", "elf_signing"), func() {
	DescribeTable("should sign and verify a published ELF artifact",
		func(testOpts testOptions) {
			var elfSigningRootCARef string
			By("initializing git repo")
			{
				importGPGKeys(testOpts.pgpKeys)
				for _, v := range testOpts.pgpKeys {
					SuiteData.GPGKeys = append(SuiteData.GPGKeys, v)
				}
				initGitRepo(SuiteData.TestDir, testOpts.branchName)
			}
			By("setup minio and vault")
			{
				setupMinio(testOpts.projectName)
				setupVault(SuiteData.TestDir)
			}
			By("configure server")
			{
				serverInitProject(SuiteData.TestDir, testOpts.projectName)
				serverConfigureProject(SuiteData.TestDir, serverConfigureOptions{
					ProjectName:        testOpts.projectName,
					RepoURL:            SuiteData.TestDir,
					TrdlChannelsBranch: testOpts.branchName,
					RequiredNumberOfVerifiedSignaturesOnCommit: 3,
					S3Endpoint:        "http://localhost:9000",
					S3Region:          "ru-central1",
					S3AccessKeyID:     "minioadmin",
					S3SecretAccessKey: "minioadmin",
					S3BucketName:      testOpts.projectName,
				})
				serverReadProjectConfig(SuiteData.TestDir, testOpts.projectName)
				serverAddGPGKeys(SuiteData.TestDir, testOpts.projectName, testOpts.pgpKeys)
				elfSigningRootCARef = serverConfigureELFSigning(SuiteData.TestDir, testOpts.projectName)
			}
			By(fmt.Sprintf("[server] Releasing tag %q ...", testOpts.tag1))
			{
				By(fmt.Sprintf("[server] Creating tag %q", testOpts.tag1))
				gitTag(SuiteData.TestDir, testOpts.tag1, testOpts.pgpKeys["developer"])

				By(fmt.Sprintf("[server] Signing tag %q", testOpts.tag1))
				quorumSignTag(SuiteData.TestDir, testOpts.pgpKeys["tl"], testOpts.pgpKeys["pm"], testOpts.tag1)

				By(fmt.Sprintf("[server] Releasing tag %q", testOpts.tag1))
				serverRelease(SuiteData.TrdlVaultClientBinPath, testOpts.projectName, testOpts.tag1)
			}
			By("[server] Verifying published ELF binary signature ...")
			{
				verifyELFSigning(SuiteData.TmpDir, testOpts.projectName, elfSigningRootCARef, testOpts.version1)
			}
		},
		Entry("standart test", testOptions{
			projectName: "test1",
			branchName:  "main",
			tag1:        "v1.0.1",
			version1:    "1.0.1",

			pgpKeys: map[string]string{
				"developer": "74E1259029B147CB4033E8B80D4C9C140E8A1030",
				"tl":        "2BA55FD8158034EEBE92AA9ED9D79B63AFC30C7A",
				"pm":        "C353F279F552B3EF16DAE0A64354E51BF178F735",
			},
		}),
	)
})
