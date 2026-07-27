//go:build linux && amd64 && cgo

package elf_signing

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/deckhouse/delivery-kit-sdk/pkg/signature/elf/inhouse"
	"github.com/deckhouse/delivery-kit-sdk/test/pkg/cert_utils"
	. "github.com/onsi/gomega"

	"github.com/werf/trdl/server/pkg/testutil"
)

var trdlRepositoryDirectory string

const (
	vaultAddress = "-address=http://localhost:8200"
	minioAddress = "http://localhost:9000"
)

func init() {
	var err error
	trdlRepositoryDirectory, err = filepath.Abs("../../../")
	if err != nil {
		panic(err)
	}
}

func BuildTrdlServerBin() {
	testutil.RunSucceedCommand(
		trdlRepositoryDirectory,
		"task",
		"--yes",
		"server:build:test:with-coverage",
	)
}

func ComputeTrdlVaultClientPath() string {
	testutil.RunSucceedCommand(
		trdlRepositoryDirectory,
		"task",
		"--yes",
		"release:build:test:with-coverage",
	)
	p, _ := filepath.Abs(filepath.Join(trdlRepositoryDirectory, "bin/trdl-vault/trdl-vault"))
	return p
}

func importGPGKeys(keys map[string]string) {
	for user := range keys {
		testutil.RunSucceedCommand(
			testutil.FixturePath("pgp_keys"),
			"gpg",
			"--import",
			fmt.Sprintf("%s_private.pgp", user),
		)
	}
}

func removeGPGKeys(keys []string) {
	for _, keyId := range keys {
		testutil.RunSucceedCommand(
			testutil.FixturePath("pgp_keys"),
			"gpg",
			"--batch", "--yes", "--delete-secret-and-public-key",
			keyId,
		)
	}
}

func initGitRepo(testDir, branchName string) {
	testutil.CopyIn(testutil.FixturePath("complete_cycle"), testDir)

	testutil.RunSucceedCommand(
		testDir,
		"git",
		"-c", "init.defaultBranch="+branchName,
		"init",
	)

	testutil.RunSucceedCommand(
		testDir,
		"touch", "testfile",
	)

	testutil.RunSucceedCommand(
		testDir,
		"git",
		"add", "-A",
	)

	testutil.RunSucceedCommand(
		testDir,
		"git",
		"commit", "-m", "Initial commit",
	)
}

func gitTag(testDir, tag, pgpSigningKeyDeveloper string) {
	testutil.RunSucceedCommand(
		testDir,
		"git",
		"-c", "tag.gpgsign=true",
		"-c", "user.signingkey="+pgpSigningKeyDeveloper,
		"tag", tag, "-m", "New version",
	)
}

func quorumSignTag(testDir, pgpSigningKeyTL, pgpSigningKeyPM, tag string) {
	if runtime.GOOS == "darwin" {
		err := os.Setenv("GIT_EDITOR", `vim -c ":normal iNew version" -c ":wq"`)
		Expect(err).ShouldNot(HaveOccurred())
	}
	testutil.RunSucceedCommand(
		testDir,
		"git",
		"signatures", "add", "--key", pgpSigningKeyTL, tag,
	)

	testutil.RunSucceedCommand(
		testDir,
		"git",
		"signatures", "add", "--key", pgpSigningKeyPM, tag,
	)
}

func setupMinio(bucketName string) {
	testutil.RunSucceedCommand(
		trdlRepositoryDirectory,
		"task",
		"--yes",
		"server:setup-minio",
		fmt.Sprintf("project_name=%s", bucketName),
	)
}

func setupVault(testDir string) {
	testutil.RunSucceedCommand(
		trdlRepositoryDirectory,
		"task",
		"--yes",
		"server:setup-vault-local",
		fmt.Sprintf("test_dir=%s", testDir),
	)
}

func serverInitProject(testDir, projectName string) {
	testutil.RunSucceedCommand(
		testDir,
		"vault", "secrets", "enable",
		vaultAddress,
		fmt.Sprintf("-path=%s", projectName), "vault-plugin-secrets-trdl",
	)
}

func cleanupEnvironment() {
	testutil.RunSucceedCommand(
		trdlRepositoryDirectory,
		"task",
		"--yes",
		"server:dev:cleanup",
	)
}

type serverConfigureOptions struct {
	ProjectName                                string
	RepoURL                                    string
	TrdlChannelsBranch                         string
	RequiredNumberOfVerifiedSignaturesOnCommit int
	S3Endpoint                                 string
	S3Region                                   string
	S3AccessKeyID                              string
	S3SecretAccessKey                          string
	S3BucketName                               string
}

func serverConfigureProject(testDir string, opts serverConfigureOptions) {
	testutil.RunSucceedCommand(
		testDir,
		"vault", "write",
		vaultAddress,
		fmt.Sprintf("%s/configure", opts.ProjectName),
		fmt.Sprintf("git_repo_url=%s", opts.RepoURL),
		fmt.Sprintf("git_trdl_channels_branch=%s", opts.TrdlChannelsBranch),
		fmt.Sprintf("required_number_of_verified_signatures_on_commit=%d", opts.RequiredNumberOfVerifiedSignaturesOnCommit),
		fmt.Sprintf("s3_endpoint=%s", opts.S3Endpoint),
		fmt.Sprintf("s3_region=%s", opts.S3Region),
		fmt.Sprintf("s3_access_key_id=%s", opts.S3AccessKeyID),
		fmt.Sprintf("s3_secret_access_key=%s", opts.S3SecretAccessKey),
		fmt.Sprintf("s3_bucket_name=%s", opts.S3BucketName),
	)
}

func serverReadProjectConfig(testDir, projectName string) {
	testutil.RunSucceedCommand(
		testDir,
		"vault", "read",
		vaultAddress,
		fmt.Sprintf("%s/configure", projectName),
	)
}

func serverAddGPGKeys(testDir, projectName string, keys map[string]string) {
	for user := range keys {
		fileName := fmt.Sprintf("%s_public.pgp", user)
		filePath := testutil.FixturePath("pgp_keys", fileName)
		data, err := os.ReadFile(filePath)
		Expect(err).ShouldNot(HaveOccurred())

		testutil.RunSucceedCommand(
			testDir,
			"vault", "write",
			vaultAddress,
			fmt.Sprintf("%s/configure/trusted_pgp_public_key", projectName),
			fmt.Sprintf("name=%s", user),
			fmt.Sprintf("public_key=%s", string(data)),
		)
	}
}

func serverConfigureELFSigning(testDir, projectName string) string {
	certs := cert_utils.GenerateCertificatesWithOptions(cert_utils.GenerateCertificatesOptions{
		UseBase64Encoding: true,
	})

	testutil.RunSucceedCommand(
		testDir,
		"vault", "write",
		vaultAddress,
		fmt.Sprintf("%s/configure/delivery_kit_elf_signing", projectName),
		fmt.Sprintf("key=%s", certs.PrivRef),
		fmt.Sprintf("certificate=%s", certs.LeafRef),
		fmt.Sprintf("intermediates=%s", certs.IntermediatesRef),
	)

	return certs.RootRef
}

func serverRelease(bin, projectName, tagName string) {
	testutil.RunSucceedCommand(
		"",
		bin,
		"release", projectName, tagName,
		"--token", "root",
		"--max-attempts", "1",
	)
}

func verifyELFSigning(tmpDir, projectName, rootCARef, version string) {
	artifactURL := fmt.Sprintf("%s/%s/targets/releases/%s/linux-amd64/bin/tool", minioAddress, projectName, version)
	resp, err := http.Get(artifactURL)
	Expect(err).ShouldNot(HaveOccurred())
	defer func() { _ = resp.Body.Close() }()
	Expect(resp.StatusCode).Should(Equal(http.StatusOK))

	elfPath := filepath.Join(tmpDir, fmt.Sprintf("signed-tool-%s", version))
	out, err := os.Create(elfPath)
	Expect(err).ShouldNot(HaveOccurred())
	_, err = io.Copy(out, resp.Body)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(out.Close()).ShouldNot(HaveOccurred())

	Expect(inhouse.Verify(context.Background(), []string{rootCARef}, elfPath)).ShouldNot(HaveOccurred())
}
