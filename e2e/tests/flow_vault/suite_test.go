package flow

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prashantv/gostub"

	"github.com/werf/trdl/server/pkg/testutil"
)

func Test(t *testing.T) {
	testutil.MeetsRequirementTools([]string{"docker", "git", "git-signatures", "gpg"})
	RegisterFailHandler(Fail)
	RunSpecs(t, "Flow Suite")
}

var SuiteData = struct {
	TrdlBinPath            string
	TrdlHomeDir            string
	TrdlVaultClientBinPath string

	TmpDir  string
	TestDir string

	Stubs *gostub.Stubs

	GPGKeys []string
}{}

var (
	_ = BeforeSuite(func() {
		BuildTrdlServerBin()
		SuiteData.TrdlBinPath = testutil.ComputeTrdlBinPath()
		SuiteData.TrdlVaultClientBinPath = ComputeTrdlVaultClientPath()
	})

	_ = BeforeEach(func() {
		SuiteData.Stubs = gostub.New()
		SuiteData.TmpDir = testutil.GetTempDir()

		SuiteData.TestDir = filepath.Join(SuiteData.TmpDir, "project")
		Ω(os.Mkdir(SuiteData.TestDir, os.ModePerm))

		SuiteData.TrdlHomeDir = filepath.Join(SuiteData.TmpDir, ".trdl")
		SuiteData.TrdlHomeDir = SuiteData.TmpDir
		SuiteData.Stubs.SetEnv("TRDL_HOME_DIR", SuiteData.TrdlHomeDir)
		SuiteData.Stubs.SetEnv("TRDL_NO_SELF_UPDATE", "1")
	})

	_ = AfterEach(func() {
		removeGPGKeys(SuiteData.GPGKeys)
		err := os.RemoveAll(SuiteData.TmpDir)
		Ω(err).ShouldNot(HaveOccurred())
		dockerComposeDown(SuiteData.TestDir)
	})
)
