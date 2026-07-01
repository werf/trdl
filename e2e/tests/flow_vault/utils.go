package flow

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/deckhouse/delivery-kit-sdk/pkg/signature/elf/inhouse"
	"github.com/deckhouse/delivery-kit-sdk/test/pkg/cert_utils"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"

	clientUtil "github.com/werf/trdl/client/pkg/util"
	"github.com/werf/trdl/server/pkg/publisher"
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

func quorumSignCommit(testDir, pgpSigningKeyTL, pgpSigningKeyPM, branchName string) {
	if runtime.GOOS == "darwin" {
		err := os.Setenv("GIT_EDITOR", `vim -c ":normal iNew version" -c ":wq"`)
		Expect(err).ShouldNot(HaveOccurred())
	}
	testutil.RunSucceedCommand(
		testDir,
		"git",
		"signatures", "add", "--key", pgpSigningKeyTL, branchName,
	)

	testutil.RunSucceedCommand(
		testDir,
		"git",
		"signatures", "add", "--key", pgpSigningKeyPM, branchName,
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
	InitialLastPublishedGitCommit              string
	RequiredNumberOfVerifiedSignaturesOnCommit int
	S3Endpoint                                 string
	S3Region                                   string
	S3AccessKeyID                              string
	S3SecretAccessKey                          string
	S3BucketName                               string
}

func serverConfigureProject(testDir string, opts serverConfigureOptions) {
	lastPubCommit := func() string {
		if opts.InitialLastPublishedGitCommit != "" {
			return fmt.Sprintf("initial_last_published_git_commit=%s", opts.InitialLastPublishedGitCommit)
		}
		return ""
	}()
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
		lastPubCommit,
	)
}

// serverConfigureELFSigning configures ELF signing for the project through the
// real Vault plugin and returns the root CA reference used to verify signatures.
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

// verifyELFSigning downloads the published linux/amd64 ELF artifact from MinIO
// and validates its embedded signature against the generated root CA.
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

func serverAddBuildSecrets(testDir, projectName string, secrets map[string]string) {
	for id, data := range secrets {
		testutil.RunSucceedCommand(
			testDir,
			"vault", "write",
			vaultAddress,
			fmt.Sprintf("%s/configure/build/secrets", projectName),
			fmt.Sprintf("id=%s", id),
			fmt.Sprintf("data=%s", data),
		)
	}
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

func serverRelease(bin, projectName, tagName string) {
	testutil.RunSucceedCommand(
		"",
		bin,
		"release", projectName, tagName,
		"--token", "root",
		"--max-attempts", "1",
	)
}

func serverPublish(bin, projectName string) {
	testutil.RunSucceedCommand(
		"",
		bin,
		"publish", projectName,
		"--token", "root",
		"--max-attempts", "1",
	)
}

func clientAdd(testDir, repo string, rootVersion int, trdlBinPath string) {
	resp, err := http.Get(fmt.Sprintf("%s/%s/%d.root.json", minioAddress, repo, rootVersion))
	Expect(err).ShouldNot(HaveOccurred())
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	Expect(err).ShouldNot(HaveOccurred())
	rootRoleSha512 := clientUtil.Sha512Checksum(data)

	testutil.RunSucceedCommand(
		testDir,
		trdlBinPath,
		"add", repo, fmt.Sprintf("%s/%s", minioAddress, repo), fmt.Sprintf("%d", rootVersion), rootRoleSha512,
	)
}

type TrdlChannelsConfiguration struct {
	Group   string
	Channel string
	Version string
}

func gitAddTrdlChannelsConfiguration(testDir, pgpSigningKeyDeveloper string, channelCfg TrdlChannelsConfiguration) string {
	type configurationGroupChannel struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	}

	type configurationGroup struct {
		Name     string                      `yaml:"name"`
		Channels []configurationGroupChannel `yaml:"channels"`
	}

	type configuration struct {
		Groups []configurationGroup `yaml:"groups"`
	}

	conf := configuration{
		Groups: []configurationGroup{
			{
				Name: channelCfg.Group,
				Channels: []configurationGroupChannel{
					{
						Name:    channelCfg.Channel,
						Version: channelCfg.Version,
					},
				},
			},
		},
	}

	data, err := yaml.Marshal(conf)
	Expect(err).ShouldNot(HaveOccurred())

	err = os.WriteFile(filepath.Join(testDir, "trdl_channels.yaml"), data, 0o755)
	Expect(err).ShouldNot(HaveOccurred())

	testutil.RunSucceedCommand(
		testDir,
		"git",
		"add", "trdl_channels.yaml",
	)

	testutil.RunSucceedCommand(
		testDir,
		"git",
		"-c", "commit.gpgsign=true",
		"-c", "user.signingkey="+pgpSigningKeyDeveloper,
		"commit", "-m", "Update trdl_channels.yaml",
	)

	return testutil.GetHeadCommit(testDir)
}

func clientUse(trdlBinPath, tmpDir, repo string, channelCfg TrdlChannelsConfiguration) {
	var shellCommandName string
	var shellCommandArgsFunc func(testScriptPath string) []string
	var scriptFormat string
	var expectedOutput string
	if runtime.GOOS == "windows" {
		shellCommandName = "powershell.exe"
		shellCommandArgsFunc = func(testScriptPath string) []string {
			return []string{"-command", testScriptPath}
		}
		scriptFormat = `
$TRDL_USE_SCRIPT_PATH = %[1]s
. $TRDL_USE_SCRIPT_PATH.Trim()
script.bat
`
		expectedOutput = "v" + channelCfg.Version + "\r\n"
	} else {
		shellCommandName = "sh"
		shellCommandArgsFunc = func(testScriptPath string) []string {
			return []string{"-c", testScriptPath}
		}
		scriptFormat = `
. $(%[1]s)
script.sh
`
		expectedOutput = "v" + channelCfg.Version + "\n"
	}

	shellCommandPath, err := exec.LookPath(shellCommandName)
	Expect(err).ShouldNot(HaveOccurred())

	trdlUseCommand := strings.Join(append(
		[]string{trdlBinPath},
		testutil.TrdlBinArgs("use", repo, channelCfg.Group, channelCfg.Channel)...,
	), " ")

	scriptPath := filepath.Join(tmpDir, "script.ps1")
	err = os.WriteFile(scriptPath, []byte(fmt.Sprintf(scriptFormat, trdlUseCommand)), 0o755)
	Expect(err).ShouldNot(HaveOccurred())

	shellCommandArgs := shellCommandArgsFunc(scriptPath)
	output := testutil.SucceedCommandOutputString(
		"",
		shellCommandPath,
		shellCommandArgs...,
	)
	Expect(output).Should(Equal(expectedOutput))
}

func clientUpdate(trdlBinPath, repo string, channelCfg TrdlChannelsConfiguration) {
	testutil.RunSucceedCommand(
		"",
		trdlBinPath,
		testutil.TrdlBinArgs("update", repo, channelCfg.Group, channelCfg.Channel)...,
	)

	output := testutil.SucceedCommandOutputString(
		"",
		trdlBinPath,
		testutil.TrdlBinArgs("bin-path", repo, channelCfg.Group, channelCfg.Channel)...,
	)

	pathParts := publisher.SplitFilepath(strings.TrimSpace(output))
	Expect(pathParts[len(pathParts)-3]).Should(Equal(channelCfg.Version))
}
