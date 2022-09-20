package client

import (
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
	validRootVersion string
	validGroup       = "0"
)

var (
	tmpDir         string
	trdlHomeDir    string
	trdlBinPath    string
	trdlBinVersion string
	stubs          *gostub.Stubs
)

var _ = SynchronizedBeforeSuite(func() []byte {
	initTufRepo()
	return testutil.ComputeTrdlBinPath()
}, func(computedPathToTrdl []byte) {
	trdlBinPath = string(computedPathToTrdl)

	output := testutil.SucceedCommandOutputString(
		"",
		trdlBinPath,
		"version",
	)
	version := strings.TrimSpace(output)
	trdlBinVersion = version
})

func initTufRepo() {
	fixturesDir := testutil.FixturePath("tuf_repo")
	testutil.RunSucceedCommand(
		"",
		"docker-compose",
		"--project-directory", fixturesDir,
		"up", "--detach", "--build",
	)

	output := testutil.SucceedCommandOutputString(
		"",
		"docker-compose",
		"--project-directory", fixturesDir,
		"port", "server", "8080",
	)
	validRepoUrl = "http://" + strings.TrimSpace(output)

	// Get root.json SHA sum.
	{
		rootJsonURI := validRepoUrl + "/root.json"
		resp, err := http.Get(rootJsonURI)
		Ω(err).ShouldNot(HaveOccurred())
		defer resp.Body.Close()

		data, err := ioutil.ReadAll(resp.Body)
		Ω(err).ShouldNot(HaveOccurred())

		validRootVersion = "0"
		validRootSHA512 = util.Sha512Checksum(data)
	}
}

var _ = AfterSuite(func() {
	fixturesDir := testutil.FixturePath("tuf_repo")

	testutil.RunSucceedCommand(
		"",
		"docker-compose",
		"--project-directory", fixturesDir,
		"down",
	)
})

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
		Ω(version).Should(Equal(trdlBinVersion))
	}
})

var _ = AfterEach(func() {
	err := os.RemoveAll(tmpDir)
	Ω(err).ShouldNot(HaveOccurred())
})
