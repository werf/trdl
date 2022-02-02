package client

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/prashantv/gostub"

	"github.com/werf/trdl/server/pkg/testutil"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Client Suite")
}

const (
	testRepoName = "test"

	validRepoUrl     = "http://localhost:9000/test-project"
	validRootVersion = "4"
	validRootSHA512  = "67afb6eb389e2ec89017ff19f94caf1c9a78d79565553d155d93d0525b28f86f6ffb6a96f3c20c4b062b7e4b2498f20050d31fd057998ec01ea625a84a93ec7e"
	validGroup       = "v0"
)

var (
	tmpDir      string
	trdlHomeDir string
	trdlBinPath string
	stubs       *gostub.Stubs
)

var _ = SynchronizedBeforeSuite(testutil.ComputeTrdlBinPath, func(computedPathToWerf []byte) {
	trdlBinPath = string(computedPathToWerf)
})

var _ = BeforeEach(func() {
	stubs = gostub.New()
	tmpDir = testutil.GetTempDir()
	trdlHomeDir = tmpDir
	stubs.SetEnv("TRDL_HOME_DIR", trdlHomeDir)
	stubs.SetEnv("TRDL_NO_SELF_UPDATE", "1")
})

var _ = AfterEach(func() {
	err := os.RemoveAll(tmpDir)
	Ω(err).ShouldNot(HaveOccurred())
})
