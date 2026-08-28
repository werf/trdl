package flow

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// The two lines carrying buildkitd_driver and buildkitd_driver_opts from the
// stored configuration into the release build have nothing else covering them:
// delete them and every other suite stays green, because the build falls back to
// the docker CLI, which works on the runner.
//
// This guard needs no cluster. The configuration names the kubernetes driver,
// which cannot come up here, while the environment names a docker-container
// buildx driver that can. The release therefore has to fail, and it has to fail
// with the driver's own error: a build that succeeds means the configured driver
// never reached it.
var _ = Describe("kubernetes buildkitd driver forwarding", Label("e2e", "trdl", "buildkitd-driver-forwarding"), func() {
	BeforeEach(skipUnlessGuardingTheBuildkitdDriver)

	It("fails the release with the driver's own error instead of building through the docker CLI", func() {
		projectName := "buildkitd-driver-forwarding"
		pgpKeys := map[string]string{"developer": "developer@trdl.dev", "tl": "tl@trdl.dev", "pm": "pm@trdl.dev"}

		By("initializing git repo")
		{
			importGPGKeys(pgpKeys)
			for _, v := range pgpKeys {
				SuiteData.GPGKeys = append(SuiteData.GPGKeys, v)
			}
			initGitRepo(SuiteData.TestDir, "main")
		}

		By("setup minio and vault")
		{
			setupMinio(projectName)
			setupVault(SuiteData.TestDir)
		}

		By("configure server")
		{
			serverInitProject(SuiteData.TestDir, projectName)
			serverConfigureProject(SuiteData.TestDir, serverConfigureOptions{
				ProjectName:        projectName,
				RepoURL:            SuiteData.TestDir,
				TrdlChannelsBranch: "trdl",
				RequiredNumberOfVerifiedSignaturesOnCommit: 3,
				S3Endpoint:        "http://localhost:9000",
				S3Region:          "ru-central1",
				S3AccessKeyID:     "minioadmin",
				S3SecretAccessKey: "minioadmin",
				S3BucketName:      projectName,
			})
			serverAddGPGKeys(SuiteData.TestDir, projectName, pgpKeys)
		}

		By("releasing a signed tag")
		{
			gitTag(SuiteData.TestDir, "v1.0.0", pgpKeys["developer"])
			quorumSignTag(SuiteData.TestDir, pgpKeys["tl"], pgpKeys["pm"], "v1.0.0")

			output := serverReleaseExpectingFailure(SuiteData.TrdlVaultClientBinPath, projectName, "v1.0.0")

			// The driver reports the pod it could not bring up. Reaching the docker
			// CLI instead would either succeed or fail with a buildx error, and
			// neither mentions a builder pod.
			Expect(output).Should(ContainSubstring("builder pod"))
		}
	})
})

// The default suites run without a label filter, so these specs have to stand
// aside there: they assert a release FAILS, which is the opposite of what every
// other job expects. The guard job sets the variable, and its own step checks
// that specs actually ran, so the skip cannot quietly empty the job.
func skipUnlessGuardingTheBuildkitdDriver() {
	if os.Getenv("TRDL_TEST_BUILDKITD_DRIVER") != "kubernetes" {
		Skip("TRDL_TEST_BUILDKITD_DRIVER does not name the kubernetes driver")
	}
}
