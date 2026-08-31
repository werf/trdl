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
// Registered only when the guard job asks for it, and NOT skipped at runtime:
// the suite's AfterEach runs `server:dev:cleanup`, whose first line deletes the
// plugin binary that BeforeSuite builds once for the whole suite. A spec that
// exists and skips still runs that AfterEach, which leaves the next spec unable
// to enable the plugin at all. So this suite tolerates exactly one live spec,
// and the label filter in the job is what keeps it to one.
// guardNamespace is the namespace the job passes through
// buildkitd_driver_opts. It must differ from the kubeconfig's namespace, or the
// assertion below would hold even with the options dropped.
const guardNamespace = "trdl-forwarding-guard"

func init() {
	if os.Getenv("TRDL_TEST_BUILDKITD_DRIVER") != "kubernetes" {
		return
	}

	Describe("kubernetes buildkitd driver forwarding", Label("e2e", "trdl", "buildkitd-driver-forwarding"), func() {
		It("fails the release with the driver's own error instead of building through the docker CLI", func() {
			projectName := "buildkitd-driver-forwarding"
			// The same fixture identities the flow test uses; these are key
			// fingerprints, not addresses.
			pgpKeys := map[string]string{
				"developer": "74E1259029B147CB4033E8B80D4C9C140E8A1030",
				"tl":        "2BA55FD8158034EEBE92AA9ED9D79B63AFC30C7A",
				"pm":        "C353F279F552B3EF16DAE0A64354E51BF178F735",
			}

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
					TrdlChannelsBranch: "main",
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

				// Two things have to be true, and the namespace is what makes the
				// second one observable. The error has to come from the driver at
				// all — the docker CLI path would have succeeded here — and it has
				// to name the namespace the OPTIONS carried, not the one the
				// kubeconfig names. Dropping buildkitd_driver_opts anywhere along
				// the way leaves the kubeconfig's namespace in this message.
				Expect(output).Should(ContainSubstring("builder pod " + guardNamespace + "/"))
			}
		})
	})
}
