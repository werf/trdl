package flow

import (
	"context"
	_ "embed"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"

	clientUtil "github.com/werf/trdl/client/pkg/util"
	"github.com/werf/trdl/server"
	"github.com/werf/trdl/server/pkg/publisher"
	tasksManagerTestutil "github.com/werf/trdl/server/pkg/tasks_manager/testutil"
	"github.com/werf/trdl/server/pkg/testutil"
	"github.com/werf/trdl/server/pkg/util"
)

var _ = Describe("Complete cycle", func() {
	var storage logical.Storage
	var backend *server.Backend
	var minioAddress string
	var minioRepoAddress string
	var systemClock *util.FixedClock

	const (
		repo       = "test"
		secondRepo = "test2"
		group      = "1"
		channel    = "alpha"
		tag1       = "v1.0.1"
		tag2       = "v1.0.2"
		tag3       = "v1.0.3"
		version1   = "1.0.1"
		version2   = "1.0.2"
		version3   = "1.0.3"

		branchName = "main"

		pgpSigningKeyDeveloper = "74E1259029B147CB4033E8B80D4C9C140E8A1030"
		pgpSigningKeyTL        = "2BA55FD8158034EEBE92AA9ED9D79B63AFC30C7A"
		pgpSigningKeyPM        = "C353F279F552B3EF16DAE0A64354E51BF178F735"
	)

	serverInitVariables := func() {
		systemClock = util.NewFixedClock(time.Now())
		server.SystemClock = systemClock

		var err error
		backend, err = server.NewBackend(hclog.L())
		Ω(err).ShouldNot(HaveOccurred())
		storage = &logical.InmemStorage{}

		config := logical.TestBackendConfig()
		config.StorageView = storage
		err = backend.Setup(context.Background(), config)
		Ω(err).ShouldNot(HaveOccurred())
	}

	gpgImportKeys := func() {
		testutil.RunSucceedCommand(
			testutil.FixturePath("pgp_keys"),
			"gpg",
			"--import",
			"developer_private.pgp",
			"tl_private.pgp",
			"pm_private.pgp",
		)
	}

	gitInitRepo := func() {
		testutil.CopyIn(testutil.FixturePath("complete_cycle"), testDir)

		testutil.RunSucceedCommand(
			testDir,
			"git",
			"-c", "init.defaultBranch="+branchName,
			"init",
		)

		testutil.RunSucceedCommand(
			testDir,
			"git",
			"add", "-A",
		)

		testutil.RunSucceedCommand(
			testDir,
			"git",
			"commit", "-m", "Initial commit",
		)
	}

	gitTag := func(tag string) {
		testutil.RunSucceedCommand(
			testDir,
			"git",
			"-c", "tag.gpgsign=true",
			"-c", "user.signingkey="+pgpSigningKeyDeveloper,
			"tag", tag, "-m", "New version",
		)
	}

	quorumSignTag := func(tag string) {
		testutil.RunSucceedCommand(
			testDir,
			"git",
			"signatures", "add", "--key", pgpSigningKeyTL, tag,
		)

		testutil.RunSucceedCommand(
			testDir,
			"git",
			"signatures", "add", "--key", pgpSigningKeyPM, tag,
		)
	}

	quorumSignCommit := func(_ string) {
		testutil.RunSucceedCommand(
			testDir,
			"git",
			"signatures", "add", "--key", pgpSigningKeyTL, branchName,
		)

		testutil.RunSucceedCommand(
			testDir,
			"git",
			"signatures", "add", "--key", pgpSigningKeyPM, branchName,
		)
	}

	composeUpMinio := func() {
		testutil.RunSucceedCommand(
			testDir,
			"docker",
			"compose",
			"up", "--detach",
		)
	}

	composeAddMinioRepo := func() {
		testutil.RunSucceedCommand(
			testDir,
			"docker",
			"compose",
			"run", "mc", "mb", "main/repo",
		)

		testutil.RunSucceedCommand(
			testDir,
			"docker",
			"compose",
			"run", "mc", "policy", "set", "download", "main/repo",
		)

		output := testutil.SucceedCommandOutputString(
			testDir,
			"docker",
			"compose",
			"port", "minio", "9000",
		)
		minioAddress = "http://" + strings.TrimSpace(output)
		minioRepoAddress = minioAddress + "/repo"
	}

	composeDownMinio := func() {
		testutil.RunSucceedCommand(
			testDir,
			"docker",
			"compose",
			"down",
		)
	}

	serverConfigure := func() {
		req := &logical.Request{Storage: storage}
		req.Path = "configure"
		req.Operation = logical.CreateOperation
		req.Data = map[string]interface{}{
			"git_repo_url":                                     testDir,
			"git_trdl_channels_branch":                         branchName,
			"initial_last_published_git_commit":                "",
			"required_number_of_verified_signatures_on_commit": 3,
			"s3_endpoint":                                      minioAddress,
			"s3_region":                                        "ru-central1",
			"s3_access_key_id":                                 "minioadmin",
			"s3_secret_access_key":                             "minioadmin",
			"s3_bucket_name":                                   "repo",
		}
		resp, err := backend.HandleRequest(context.Background(), req)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(resp).Should(BeNil())

		for _, user := range []string{"developer", "tl", "pm"} {
			fileName := fmt.Sprintf("%s_public.pgp", user)
			filePath := testutil.FixturePath("pgp_keys", fileName)
			data, err := ioutil.ReadFile(filePath)
			Ω(err).ShouldNot(HaveOccurred())

			req = &logical.Request{Storage: storage}
			req.Path = "configure/trusted_pgp_public_key"
			req.Operation = logical.CreateOperation
			req.Data = map[string]interface{}{
				"name":       user,
				"public_key": string(data),
			}

			resp, err = backend.HandleRequest(context.Background(), req)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(resp).Should(BeNil())
		}
	}

	serverRelease := func(tagName string) {
		req := &logical.Request{Storage: storage}
		req.Path = "release"
		req.Operation = logical.CreateOperation
		req.Data = map[string]interface{}{"git_tag": tagName}
		resp, err := backend.HandleRequest(context.Background(), req)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(resp).ShouldNot(BeNil())

		val, ok := resp.Data["task_uuid"]
		Ω(ok).Should(BeTrue(), fmt.Sprintf("%+v", resp.Data))
		taskUUID := val.(string)

		tasksManagerTestutil.WaitForTaskSuccess(GinkgoWriter, GinkgoT(), context.Background(), backend, storage, taskUUID)
	}

	clientAdd := func(repo string, rootVersion int) {
		resp, err := http.Get(minioRepoAddress + fmt.Sprintf("/%d.root.json", rootVersion))
		Ω(err).ShouldNot(HaveOccurred())
		defer func() { _ = resp.Body.Close() }()

		data, err := ioutil.ReadAll(resp.Body)
		Ω(err).ShouldNot(HaveOccurred())
		rootRoleSha512 := clientUtil.Sha512Checksum(data)

		testutil.RunSucceedCommand(
			testDir,
			trdlBinPath,
			"add", repo, minioRepoAddress, fmt.Sprintf("%d", rootVersion), rootRoleSha512,
		)
	}

	serverPublish := func() {
		req := &logical.Request{Storage: storage}
		req.Path = "publish"
		req.Operation = logical.CreateOperation
		req.Data = map[string]interface{}{}
		resp, err := backend.HandleRequest(context.Background(), req)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(resp).ShouldNot(BeNil())

		val, ok := resp.Data["task_uuid"]
		Ω(ok).Should(BeTrue(), fmt.Sprintf("%+v", resp.Data))
		taskUUID := val.(string)

		tasksManagerTestutil.WaitForTaskSuccess(GinkgoWriter, GinkgoT(), context.Background(), backend, storage, taskUUID)
	}

	clientUpdate := func(repo, group, channel, expectedVersion string) {
		testutil.RunSucceedCommand(
			"",
			trdlBinPath,
			testutil.TrdlBinArgs("update", repo, group, channel)...,
		)

		output := testutil.SucceedCommandOutputString(
			"",
			trdlBinPath,
			testutil.TrdlBinArgs("bin-path", repo, group, channel)...,
		)

		pathParts := publisher.SplitFilepath(strings.TrimSpace(output))
		Ω(pathParts[len(pathParts)-3]).Should(Equal(expectedVersion))
	}

	clientUse := func(group, channel, expectedVersion string) {
		var shellCommandName string
		var shellCommandArgsFunc func(testScriptPath string) []string
		var scriptFormat string
		var expectedOutput string
		if runtime.GOOS == "windows" {
			shellCommandName = "powershell.exe"
			shellCommandArgsFunc = func(testScriptPath string) []string {
				return []string{"-command", testScriptPath}
			}
			scriptFormat = `
$TRDL_USE_SCRIPT_PATH = %[1]s
. $TRDL_USE_SCRIPT_PATH.Trim()
script.bat
`
			expectedOutput = fmt.Sprintf("v" + expectedVersion + "\r\n")
		} else {
			shellCommandName = "sh"
			shellCommandArgsFunc = func(testScriptPath string) []string {
				return []string{"-c", testScriptPath}
			}
			scriptFormat = `
. $(%[1]s)
script.sh
`
			expectedOutput = fmt.Sprintf("v" + expectedVersion + "\n")
		}

		shellCommandPath, err := exec.LookPath(shellCommandName)
		Ω(err).ShouldNot(HaveOccurred())

		trdlUseCommand := strings.Join(append(
			[]string{trdlBinPath},
			testutil.TrdlBinArgs("use", repo, group, channel)...,
		), " ")

		scriptPath := filepath.Join(tmpDir, "script.ps1")
		err = ioutil.WriteFile(scriptPath, []byte(fmt.Sprintf(scriptFormat, trdlUseCommand)), 0o755)
		Ω(err).ShouldNot(HaveOccurred())

		shellCommandArgs := shellCommandArgsFunc(scriptPath)
		output := testutil.SucceedCommandOutputString(
			"",
			shellCommandPath,
			shellCommandArgs...,
		)
		Ω(output).Should(Equal(expectedOutput))
	}

	BeforeEach(func() {
		gpgImportKeys()
		serverInitVariables()
		gitInitRepo()
		composeUpMinio()
		composeAddMinioRepo()
	})

	AfterEach(func() {
		composeDownMinio()
	})

	gitAddTrdlChannelsConfiguration := func(group, channel, version string) string {
		type configurationGroupChannel struct {
			Name    string `yaml:"name"`
			Version string `yaml:"version"`
		}

		type configurationGroup struct {
			Name     string                      `yaml:"name"`
			Channels []configurationGroupChannel `yaml:"channels"`
		}

		type configuration struct {
			Groups []configurationGroup `yaml:"groups"`
		}

		conf := configuration{
			Groups: []configurationGroup{
				{
					Name: group,
					Channels: []configurationGroupChannel{
						{
							Name:    channel,
							Version: version,
						},
					},
				},
			},
		}

		data, err := yaml.Marshal(conf)
		Ω(err).ShouldNot(HaveOccurred())

		err = ioutil.WriteFile(filepath.Join(testDir, "trdl_channels.yaml"), data, 0o755)
		Ω(err).ShouldNot(HaveOccurred())

		testutil.RunSucceedCommand(
			testDir,
			"git",
			"add", "trdl_channels.yaml",
		)

		testutil.RunSucceedCommand(
			testDir,
			"git",
			"-c", "commit.gpgsign=true",
			"-c", "user.signingkey="+pgpSigningKeyDeveloper,
			"commit", "-m", "Update trdl_channels.yaml",
		)

		return testutil.GetHeadCommit(testDir)
	}

	rotateTUFRoles := func() {
		// After 1 year all TUF roles will be rotated: root.json, targets.json, snapshot.json and timestamp.json
		systemClock.NowTime = systemClock.NowTime.AddDate(1, 0, 0)

		req := &logical.Request{Storage: storage}
		Expect(backend.Periodic(context.Background(), req)).To(Succeed())

		// Wait for the background task to be completed
		time.Sleep(time.Millisecond * 500)
	}

	It("should perform all steps", func() {
		By("[server] Configuring ...")
		serverConfigure()

		By(fmt.Sprintf("[server] Releasing tag %q ...", tag1))
		{
			gitTag(tag1)
			quorumSignTag(tag1)
			serverRelease(tag1)
		}

		By("[client] Adding repo ...")
		clientAdd(repo, 1)

		By("[server] Publishing channels ...")
		{
			commit := gitAddTrdlChannelsConfiguration(group, channel, version1)
			quorumSignCommit(commit)
			serverPublish()
		}

		By("[client] Using channel release ...")
		clientUse(group, channel, version1)

		By(fmt.Sprintf("[server] Releasing tag %q ...", tag2))
		{
			gitTag(tag2)
			quorumSignTag(tag2)
			serverRelease(tag2)
			commit := gitAddTrdlChannelsConfiguration(group, channel, version2)
			quorumSignCommit(commit)
			serverPublish()
		}

		By("[client] Using new channel release ...")
		clientUse(group, channel, version1)

		// Wait for the background update to be completed
		time.Sleep(time.Millisecond * 500)
		clientUse(group, channel, version2)

		By(fmt.Sprintf("[server] Rotating TUF roles expiration ..."))
		rotateTUFRoles()

		By("[client-1] Getting channel release when no updates available ...")
		clientUpdate(repo, group, channel, version2)

		By(fmt.Sprintf("[server] Rotating TUF roles expiration ..."))
		rotateTUFRoles()

		By("[client-1] Getting channel release when no updates available ...")
		clientUpdate(repo, group, channel, version2)

		By("[client-2] Adding repo ...")
		clientAdd(secondRepo, 3)

		By("[client-2] Getting channel release when update available ...")
		clientUpdate(secondRepo, group, channel, version2)

		By(fmt.Sprintf("[server] Rotating TUF roles expiration ..."))
		rotateTUFRoles()

		By(fmt.Sprintf("[server] Releasing tag %q ...", tag3))
		{
			gitTag(tag3)
			quorumSignTag(tag3)
			serverRelease(tag3)
			commit := gitAddTrdlChannelsConfiguration(group, channel, version3)
			quorumSignCommit(commit)
			serverPublish()
		}

		By("[client-1] Getting channel release when update available ...")
		clientUpdate(repo, group, channel, version3)

		By("[client-2] Getting channel release when update available ...")
		clientUpdate(secondRepo, group, channel, version3)
	})
})
