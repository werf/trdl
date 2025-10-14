package client

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prashantv/gostub"

	"github.com/werf/trdl/client/pkg/util"
	"github.com/werf/trdl/server/pkg/testutil"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Client Suite")
}

var (
	testRepoName = "test"

	validRepoUrl     string
	validRootSHA512  string
	validRootVersion = "0"
	validGroup       = "0"
)

var (
	tmpDir         string
	trdlHomeDir    string
	trdlBinPath    string
	trdlBinVersion string
	stubs          *gostub.Stubs
)

type SyncBeforeSuiteFirstFuncResult struct {
	ValidRepoURL       string
	ValidRootSHA512    string
	ComputedPathToTrdl string
}

var _ = SynchronizedBeforeSuite(func() []byte {
	fixturesDir := testutil.FixturePath("tuf_repo")

	testutil.RunSucceedCommand(
		"",
		"docker",
		"compose",
		"--project-directory", fixturesDir,
		"up", "--detach", "--build",
	)

	output := testutil.SucceedCommandOutputString(
		"",
		"docker",
		"compose",
		"--project-directory", fixturesDir,
		"port", "server", "8080",
	)
	repoUrl := "http://" + strings.TrimSpace(output)

	var rootSHA512 string
	{
		rootJsonURI := repoUrl + "/root.json"
		resp, err := http.Get(rootJsonURI)
		Expect(err).ShouldNot(HaveOccurred())
		defer resp.Body.Close()

		data, err := ioutil.ReadAll(resp.Body)
		Expect(err).ShouldNot(HaveOccurred())

		rootSHA512 = util.Sha512Checksum(data)
	}

	pathToTrdl := testutil.ComputeTrdlBinPath()

	result := SyncBeforeSuiteFirstFuncResult{
		ValidRepoURL:       repoUrl,
		ValidRootSHA512:    rootSHA512,
		ComputedPathToTrdl: pathToTrdl,
	}

	serializedResult, err := json.Marshal(result)
	Expect(err).ShouldNot(HaveOccurred())

	return serializedResult
}, func(firstFuncResultSerialized []byte) {
	var firstFuncResult SyncBeforeSuiteFirstFuncResult
	Expect(json.Unmarshal(firstFuncResultSerialized, &firstFuncResult)).To(Succeed())

	validRepoUrl = firstFuncResult.ValidRepoURL
	validRootSHA512 = firstFuncResult.ValidRootSHA512
	trdlBinPath = firstFuncResult.ComputedPathToTrdl

	output := testutil.SucceedCommandOutputString(
		"",
		trdlBinPath,
		"version",
	)
	version := strings.TrimSpace(output)
	trdlBinVersion = version
})

var _ = SynchronizedAfterSuite(
	func() {},
	func() {
		fixturesDir := testutil.FixturePath("tuf_repo")

		testutil.RunSucceedCommand(
			"",
			"docker",
			"compose",
			"--project-directory", fixturesDir,
			"down",
		)
	},
)

var _ = BeforeEach(func() {
	stubs = gostub.New()
	tmpDir = testutil.GetTempDir()
	trdlHomeDir = tmpDir
	stubs.SetEnv("TRDL_HOME_DIR", trdlHomeDir)
	stubs.SetEnv("TRDL_NO_SELF_UPDATE", "1")

	// check that the tested binary is not overwritten during self-update in previous tests by mistake
	{
		output := testutil.SucceedCommandOutputString(
			"",
			trdlBinPath,
			"version",
		)
		version := strings.TrimSpace(output)
		Expect(version).Should(Equal(trdlBinVersion))
	}
})

var _ = AfterEach(func() {
	err := os.RemoveAll(tmpDir)
	Expect(err).ShouldNot(HaveOccurred())
})
