package git

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/werf/trdl/server/pkg/testutil"
)

func Test(t *testing.T) {
	testutil.MeetsRequirementTools([]string{"git", "git-signatures", "gpg"})
	RegisterFailHandler(Fail)
	RunSpecs(t, "Git Suite")
}

var (
	tmpDir  string
	testDir string
)

var _ = SynchronizedBeforeSuite(func() []byte {
	testutil.RunSucceedCommand(
		testutil.FixturePath("pgp_keys"), // TODO: move to testutil
		"gpg",
		"--import",
		"developer_private.pgp",
		"tl_private.pgp",
		"pm_private.pgp",
	)

	return nil
}, func(_ []byte) {})

var _ = BeforeEach(func() {
	tmpDir = testutil.GetTempDir()
	testDir = filepath.Join(tmpDir, "project")
	Ω(os.Mkdir(testDir, os.ModePerm))
})

var _ = AfterEach(func() {
	err := os.RemoveAll(tmpDir)
	Ω(err).ShouldNot(HaveOccurred())
})
