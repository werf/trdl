//go:build ai_tests
// +build ai_tests

package mac_signing

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/werf/trdl/server/pkg/testutil"
)

func TestAI_MacSigning(t *testing.T) {
	testutil.MeetsRequirementTools([]string{"docker", "git", "git-signatures", "gpg"})
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mac Signing Suite")
}

var (
	tmpDir  string
	testDir string
)

var _ = BeforeEach(func() {
	tmpDir = testutil.GetTempDir()

	testDir = filepath.Join(tmpDir, "project")
	Expect(os.Mkdir(testDir, os.ModePerm)).To(Succeed())
})

var _ = AfterEach(func() {
	Expect(os.RemoveAll(tmpDir)).To(Succeed())
})
