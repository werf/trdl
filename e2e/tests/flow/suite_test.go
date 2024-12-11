package flow

import (
	"encoding/json"
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

var (
	tmpDir      string
	testDir     string
	trdlHomeDir string
	trdlBinPath string
	stubs       *gostub.Stubs
)

type SyncBeforeSuiteFirstFuncResult struct {
	ComputedPathToTrdl string
}

var _ = SynchronizedBeforeSuite(func() []byte {
	pathToTrdl := testutil.ComputeTrdlBinPath()

	result := SyncBeforeSuiteFirstFuncResult{
		ComputedPathToTrdl: pathToTrdl,
	}

	serializedResult, err := json.Marshal(result)
	Ω(err).ShouldNot(HaveOccurred())

	return serializedResult
}, func(firstFuncResultSerialized []byte) {
	var firstFuncResult SyncBeforeSuiteFirstFuncResult
	Expect(json.Unmarshal(firstFuncResultSerialized, &firstFuncResult)).To(Succeed())

	trdlBinPath = firstFuncResult.ComputedPathToTrdl
})

var _ = BeforeEach(func() {
	stubs = gostub.New()
	tmpDir = testutil.GetTempDir()

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
