package flow

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"

	clientUtil "github.com/werf/trdl/client/pkg/util"
	"github.com/werf/trdl/e2e/util"
	"github.com/werf/trdl/server"
	"github.com/werf/trdl/server/pkg/tasks_manager/testutil"
)

var _ = Describe("Complete cycle", func() {
	var storage logical.Storage
	var backend logical.Backend
	var minioAddress string
	var minioRepoAddress string

	const (
		repo     = "test"
		group    = "1"
		channel  = "alpha"
		tag1     = "v1.0.1"
		tag2     = "v1.0.2"
		version1 = "1.0.1"
		version2 = "1.0.2"
	)

	serverInitVariables := func() {
		var err error
		backend, err = server.NewBackend()
		Ω(err).ShouldNot(HaveOccurred())
		storage = &logical.InmemStorage{}

		config := logical.TestBackendConfig()
		config.StorageView = storage
		err = backend.Setup(context.Background(), config)
		Ω(err).ShouldNot(HaveOccurred())
	}

	gitInitRepo := func() {
		util.CopyIn(util.FixturePath("complete_cycle"), testDir)

		util.RunSucceedCommand(
			testDir,
			"git",
			"init",
		)

		util.RunSucceedCommand(
			testDir,
			"git",
			"add", "-A",
		)

		util.RunSucceedCommand(
			testDir,
			"git",
			"commit", "-m", "Initial commit",
		)

		util.RunSucceedCommand(
			testDir,
			"git",
			"checkout", "-b", "production",
		)
	}

	gitTag := func(tag string) {
		util.RunSucceedCommand(
			testDir,
			"git",
			"tag", tag,
		)
	}

	composeUpMinio := func() {
		util.RunSucceedCommand(
			testDir,
			"docker-compose",
			"up", "--detach",
		)
	}

	composeAddMinioRepo := func() {
		util.RunSucceedCommand(
			testDir,
			"docker-compose",
			"run", "mc", "mb", "main/repo",
		)

		util.RunSucceedCommand(
			testDir,
			"docker-compose",
			"run", "mc", "policy", "set", "download", "main/repo",
		)

		output := util.SucceedCommandOutputString(
			testDir,
			"docker-compose",
			"port", "minio", "9000",
		)
		minioAddress = "http://" + strings.TrimSpace(output)
		minioRepoAddress = minioAddress + "/repo"
	}

	composeDownMinio := func() {
		util.RunSucceedCommand(
			testDir,
			"docker-compose",
			"down",
		)
	}

	serverConfigure := func() {
		req := &logical.Request{Storage: storage}
		req.Path = "configure"
		req.Operation = logical.CreateOperation
		req.Data = map[string]interface{}{
			"git_repo_url":                                     testDir,
			"git_trdl_channels_branch":                         "production",
			"initial_last_published_git_commit":                "",
			"required_number_of_verified_signatures_on_commit": 0,
			"s3_endpoint":                                      minioAddress,
			"s3_region":                                        "ru-central1",
			"s3_access_key_id":                                 "minioadmin",
			"s3_secret_access_key":                             "minioadmin",
			"s3_bucket_name":                                   "repo",
		}
		resp, err := backend.HandleRequest(context.Background(), req)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(resp).Should(BeNil())
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

		testutil.WaitForTaskSuccess(GinkgoWriter, GinkgoT(), context.Background(), backend, storage, taskUUID)
	}

	clientAdd := func() {
		resp, err := http.Get(minioRepoAddress + "/1.root.json")
		Ω(err).ShouldNot(HaveOccurred())
		defer func() { _ = resp.Body.Close() }()

		data, err := ioutil.ReadAll(resp.Body)
		Ω(err).ShouldNot(HaveOccurred())
		rootRoleSha512 := clientUtil.Sha512Checksum(data)

		util.RunSucceedCommand(
			testDir,
			trdlBinPath,
			"add", repo, minioRepoAddress, "1", rootRoleSha512,
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

		testutil.WaitForTaskSuccess(GinkgoWriter, GinkgoT(), context.Background(), backend, storage, taskUUID)
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
			util.TrdlBinArgs("use", repo, group, channel)...,
		), " ")

		scriptPath := filepath.Join(tmpDir, "script.ps1")
		err = ioutil.WriteFile(scriptPath, []byte(fmt.Sprintf(scriptFormat, trdlUseCommand)), 0o755)
		Ω(err).ShouldNot(HaveOccurred())

		shellCommandArgs := shellCommandArgsFunc(scriptPath)
		output := util.SucceedCommandOutputString(
			"",
			shellCommandPath,
			shellCommandArgs...,
		)
		Ω(output).Should(Equal(expectedOutput))
	}

	BeforeEach(func() {
		serverInitVariables()
		gitInitRepo()
		composeUpMinio()
		composeAddMinioRepo()
	})

	AfterEach(func() {
		composeDownMinio()
	})

	gitAddTrdlChannelsConfiguration := func(group, channel, version string) {
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

		util.RunSucceedCommand(
			testDir,
			"git",
			"add", "trdl_channels.yaml",
		)

		util.RunSucceedCommand(
			testDir,
			"git",
			"commit", "-m", "Update trdl_channels.yaml",
		)
	}

	It("should perform all steps", func() {
		By("[server] Configuring ...")
		serverConfigure()

		By(fmt.Sprintf("[server] Releasing tag %q ...", tag1))
		{
			gitTag(tag1)
			serverRelease(tag1)
		}

		By("[client] Adding repo ...")
		clientAdd()

		By("[server] Publishing channels ...")
		{
			gitAddTrdlChannelsConfiguration(group, channel, version1)
			serverPublish()
		}

		By("[client] Using channel release ...")
		clientUse(group, channel, version1)

		By(fmt.Sprintf("[server] Releasing tag %q ...", tag2))
		{
			gitTag(tag2)
			serverRelease(tag2)
			gitAddTrdlChannelsConfiguration(group, channel, version2)
			serverPublish()
		}

		By("[client] Using new channel release ...")
		clientUse(group, channel, version1)

		// Wait for the background update to be completed
		time.Sleep(time.Millisecond * 500)
		clientUse(group, channel, version2)
	})
})
