package flow

import (
	_ "embed"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
)

type testOptions struct {
	projectName string

	repo       string
	secondRepo string
	branchName string

	group   string
	channel string

	tag1 string
	tag2 string
	tag3 string

	version1 string
	version2 string
	version3 string

	pgpKeys map[string]string

	buildSecrets map[string]string
}

var _ = Describe("trdl flow test", Label("e2e", "trdl", "flow"), func() {
	DescribeTable("should perform all steps",
		func(testOpts testOptions) {
			By("initializing git repo")
			{
				importGPGKeys(testOpts.pgpKeys)
				for _, v := range testOpts.pgpKeys {
					SuiteData.GPGKeys = append(SuiteData.GPGKeys, v)
				}
				initGitRepo(SuiteData.TestDir, "main")
			}
			By("setup minio and vault plugin")
			{
				dockerComposeUp(SuiteData.TestDir)
				setupVaultPlugin(SuiteData.TestDir)
			}
			By("configure server")
			{
				serverInitProject(SuiteData.TestDir, testOpts.projectName)
				serverConfigureProject(SuiteData.TestDir, serverConfigureOptions{
					ProjectName:        testOpts.projectName,
					RepoURL:            "/test_dir",
					TrdlChannelsBranch: testOpts.branchName,
					RequiredNumberOfVerifiedSignaturesOnCommit: 3,
					S3Endpoint:        "http://minio:9000",
					S3Region:          "ru-central1",
					S3AccessKeyID:     "minioadmin",
					S3SecretAccessKey: "minioadmin",
					S3BucketName:      "repo",
				})
				serverReadProjectConfig(SuiteData.TestDir, testOpts.projectName)
				serverAddGPGKeys(SuiteData.TestDir, testOpts.projectName, testOpts.pgpKeys)
				serverAddBuildSecrets(SuiteData.TestDir, testOpts.projectName, testOpts.buildSecrets)
			}
			By(fmt.Sprintf("[server] Releasing tag %q ...", testOpts.tag1))
			{
				By(fmt.Sprintf("[server] Creating tag tag %q", testOpts.tag1))
				gitTag(SuiteData.TestDir, testOpts.tag1, testOpts.pgpKeys["developer"])

				By(fmt.Sprintf("[server] Signing tag %q", testOpts.tag1))
				quorumSignTag(SuiteData.TestDir, testOpts.pgpKeys["tl"], testOpts.pgpKeys["pm"], testOpts.tag1)

				By(fmt.Sprintf("[server] Releasing tag %q", testOpts.tag1))
				serverRelease(testOpts.projectName, testOpts.tag1)
			}
			By("[client] Adding repo ...")
			{
				clientAdd(SuiteData.TestDir, testOpts.repo, 1, SuiteData.TrdlBinPath)
			}
			By("[server] Publishing channels ...")
			{
				_ = gitAddTrdlChannelsConfiguration(
					SuiteData.TestDir,
					testOpts.pgpKeys["developer"],
					TrdlChannelsConfiguration{
						Group:   testOpts.group,
						Channel: testOpts.channel,
						Version: testOpts.version1,
					})
				quorumSignCommit(SuiteData.TestDir, testOpts.pgpKeys["tl"], testOpts.pgpKeys["pm"], testOpts.branchName)
				serverPublish(testOpts.projectName)
			}
			By("[client] Using channel release ...")
			{
				clientUse(
					SuiteData.TrdlBinPath,
					SuiteData.TmpDir,
					testOpts.repo, TrdlChannelsConfiguration{
						Group:   testOpts.group,
						Channel: testOpts.channel,
						Version: testOpts.version1,
					})
			}

			By(fmt.Sprintf("[server] Releasing tag %q ...", testOpts.tag2))
			{
				currentTag := testOpts.tag2
				curentVersion := testOpts.version2
				By(fmt.Sprintf("[server] Creating tag tag %q", currentTag))
				gitTag(SuiteData.TestDir, currentTag, testOpts.pgpKeys["developer"])

				By(fmt.Sprintf("[server] Signing tag %q", currentTag))
				quorumSignTag(SuiteData.TestDir, testOpts.pgpKeys["tl"], testOpts.pgpKeys["pm"], currentTag)

				By(fmt.Sprintf("[server] Releasing tag %q", currentTag))
				serverRelease(testOpts.projectName, currentTag)

				By("[server] Publishing channels ...")
				_ = gitAddTrdlChannelsConfiguration(
					SuiteData.TestDir,
					testOpts.pgpKeys["developer"],
					TrdlChannelsConfiguration{
						Group:   testOpts.group,
						Channel: testOpts.channel,
						Version: curentVersion,
					})
				quorumSignCommit(SuiteData.TestDir, testOpts.pgpKeys["tl"], testOpts.pgpKeys["pm"], testOpts.branchName)
				serverPublish(testOpts.projectName)
			}

			By("[client] Using new channel release ...")
			{
				clientUse(
					SuiteData.TrdlBinPath,
					SuiteData.TmpDir,
					testOpts.repo, TrdlChannelsConfiguration{
						Group:   testOpts.group,
						Channel: testOpts.channel,
						Version: testOpts.version1,
					})
			}

			By("[client] Wait for the background update to be completed ...")
			{
				time.Sleep(time.Second * 5)
			}

			By("[client] Using new channel release ...")
			{
				clientUse(
					SuiteData.TrdlBinPath,
					SuiteData.TmpDir,
					testOpts.repo, TrdlChannelsConfiguration{
						Group:   testOpts.group,
						Channel: testOpts.channel,
						Version: testOpts.version2,
					})
			}
			By("[client] Getting channel release when no updates available ...")
			{
				clientUpdate(SuiteData.TrdlBinPath, testOpts.repo, TrdlChannelsConfiguration{
					Group:   testOpts.group,
					Channel: testOpts.channel,
					Version: testOpts.version2,
				},
				)
			}

			By(fmt.Sprintf("[server] Releasing tag %q ...", testOpts.tag3))
			{
				currentTag := testOpts.tag3
				curentVersion := testOpts.version3
				By(fmt.Sprintf("[server] Creating tag tag %q", currentTag))
				gitTag(SuiteData.TestDir, currentTag, testOpts.pgpKeys["developer"])

				By(fmt.Sprintf("[server] Signing tag %q", currentTag))
				quorumSignTag(SuiteData.TestDir, testOpts.pgpKeys["tl"], testOpts.pgpKeys["pm"], currentTag)

				By(fmt.Sprintf("[server] Releasing tag %q", currentTag))
				serverRelease(testOpts.projectName, currentTag)

				By("[server] Publishing channels ...")
				_ = gitAddTrdlChannelsConfiguration(
					SuiteData.TestDir,
					testOpts.pgpKeys["developer"],
					TrdlChannelsConfiguration{
						Group:   testOpts.group,
						Channel: testOpts.channel,
						Version: curentVersion,
					})
				quorumSignCommit(SuiteData.TestDir, testOpts.pgpKeys["tl"], testOpts.pgpKeys["pm"], testOpts.branchName)
				serverPublish(testOpts.projectName)
			}

			By("[client] Getting channel release when update available ...")
			{
				clientUpdate(SuiteData.TrdlBinPath, testOpts.repo, TrdlChannelsConfiguration{
					Group:   testOpts.group,
					Channel: testOpts.channel,
					Version: testOpts.version3,
				},
				)
			}
		},
		Entry("standart test", testOptions{
			projectName: "test1",
			repo:        "test",
			secondRepo:  "test2",
			group:       "1",
			channel:     "alpha",
			tag1:        "v1.0.1",
			tag2:        "v1.0.2",
			tag3:        "v1.0.3",
			version1:    "1.0.1",
			version2:    "1.0.2",
			version3:    "1.0.3",
			branchName:  "main",

			pgpKeys: map[string]string{
				"developer": "74E1259029B147CB4033E8B80D4C9C140E8A1030",
				"tl":        "2BA55FD8158034EEBE92AA9ED9D79B63AFC30C7A",
				"pm":        "C353F279F552B3EF16DAE0A64354E51BF178F735",
			},

			buildSecrets: map[string]string{
				"secretId0-test": "secretData",
				"secretId1-test": "secretData",
			},
		}),
	)
})
