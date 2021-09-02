package flow

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/prashantv/gostub"

	"github.com/werf/trdl/e2e/util"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Flow Suite")
}

var (
	tmpDir      string
	testDir     string
	trdlHomeDir string
	trdlBinPath string
	stubs       *gostub.Stubs
)

var _ = SynchronizedBeforeSuite(util.ComputeTrdlBinPath, func(computedPathToWerf []byte) {
	trdlBinPath = string(computedPathToWerf)
})

var _ = BeforeEach(func() {
	stubs = gostub.New()
	tmpDir = util.GetTempDir()

	testDir = filepath.Join(tmpDir, "project")
	Ω(os.Mkdir(testDir, os.ModePerm))

	trdlHomeDir = filepath.Join(tmpDir, ".trdl")
	trdlHomeDir = tmpDir
	stubs.SetEnv("TRDL_HOME_DIR", trdlHomeDir)
	stubs.SetEnv("TRDL_NO_SELF_UPDATE", "1")
})

var _ = AfterEach(func() {
	err := os.RemoveAll(tmpDir)
	Ω(err).ShouldNot(HaveOccurred())
})
