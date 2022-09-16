package client

import (
	"os"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prashantv/gostub"

	"github.com/werf/trdl/server/pkg/testutil"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Client Suite")
}

var (
	testRepoName = "test"

	validRepoUrl     string
	validRootVersion = "0"
	validRootSHA512  = "951ca9cd3e55162a2e990a9d291d51684e2bf4e7537003cf40649f7612ac9db7f9a73ff8ceb05ac14eba32c706b30874cf21a98d01a02293149d0cbbdb1e4f99"
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
	tufRepoUP()
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

func tufRepoUP() {
	fixturesDir := testutil.FixturePath("tuf_repo")
	testutil.RunSucceedCommand(
		"",
		"docker-compose",
		"--project-directory", fixturesDir,
		"up", "--detach",
	)

	output := testutil.SucceedCommandOutputString(
		"",
		"docker-compose",
		"--project-directory", fixturesDir,
		"port", "server", "8080",
	)
	validRepoUrl = "http://" + strings.TrimSpace(output)
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
