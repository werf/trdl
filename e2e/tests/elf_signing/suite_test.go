//go:build linux && amd64 && cgo

package elf_signing

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
	RunSpecs(t, "ELF Signing Suite")
}

var SuiteData = struct {
	TrdlVaultClientBinPath string

	TmpDir  string
	TestDir string

	Stubs *gostub.Stubs

	GPGKeys []string
}{}

var (
	_ = BeforeSuite(func() {
		BuildTrdlServerBin()
		SuiteData.TrdlVaultClientBinPath = ComputeTrdlVaultClientPath()
	})

	_ = BeforeEach(func() {
		SuiteData.Stubs = gostub.New()
		SuiteData.TmpDir = testutil.GetTempDir()

		SuiteData.TestDir = filepath.Join(SuiteData.TmpDir, "project")
		Expect(os.Mkdir(SuiteData.TestDir, os.ModePerm)).Should(Succeed())
	})

	_ = AfterEach(func() {
		cleanupEnvironment()
		removeGPGKeys(SuiteData.GPGKeys)
		err := os.RemoveAll(SuiteData.TmpDir)
		Expect(err).ShouldNot(HaveOccurred())
	})
)
