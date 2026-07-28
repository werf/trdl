//go:build ai_tests
// +build ai_tests

package mac_signing

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/werf/trdl/server"
	tasksManagerTestutil "github.com/werf/trdl/server/pkg/tasks_manager/testutil"
	"github.com/werf/trdl/server/pkg/testutil"
)

// The suite proves trdl's mac-signing plumbing end to end without Apple
// credentials. It builds _fixtures/quill_stub and serves it from a throwaway
// registry; that quill validates the five QUILL_* env vars and appends a marker
// with the received cert and notary key id to the artifact. Asserting the marker
// in the published artifact proves the Vault-stored credential values travelled
// through the buildkit secret mounts into the signer stage, the Mach-O detection
// loop ran, and the "signed" artifact was re-exported through the final scratch
// stage.
var _ = Describe("Mac signing", func() {
	var storage logical.Storage
	var backend *server.Backend
	var minioAddress string

	certificate := base64.StdEncoding.EncodeToString([]byte("dummy-mac-signing-certificate"))
	notaryKey := base64.StdEncoding.EncodeToString([]byte("dummy-notary-key"))

	const (
		tag     = "v1.0.1"
		version = "1.0.1"

		password     = "dummy-password"
		notaryKeyID  = "dummy-notary-key-id"
		notaryIssuer = "dummy-notary-issuer"

		machOMagic = "\317\372\355\376\007\000\000\001"

		pgpSigningKeyDeveloper = "74E1259029B147CB4033E8B80D4C9C140E8A1030"
		pgpSigningKeyTL        = "2BA55FD8158034EEBE92AA9ED9D79B63AFC30C7A"
		pgpSigningKeyPM        = "C353F279F552B3EF16DAE0A64354E51BF178F735"
	)

	handleRequest := func(path string, data map[string]interface{}) *logical.Response {
		req := &logical.Request{Storage: storage}
		req.Path = path
		req.Operation = logical.CreateOperation
		req.Data = data
		resp, err := backend.HandleRequest(context.Background(), req)
		Expect(err).ShouldNot(HaveOccurred())
		return resp
	}

	BeforeEach(func() {
		testutil.RunSucceedCommand(
			testutil.FixturePath("pgp_keys"),
			"gpg",
			"--import",
			"developer_private.pgp",
			"tl_private.pgp",
			"pm_private.pgp",
		)

		var err error
		backend, err = server.NewBackend(hclog.L())
		Expect(err).ShouldNot(HaveOccurred())
		storage = &logical.InmemStorage{}

		config := logical.TestBackendConfig()
		config.StorageView = storage
		Expect(backend.Setup(context.Background(), config)).To(Succeed())

		testutil.CopyIn(testutil.FixturePath("complete_cycle"), testDir)
		testutil.RunSucceedCommand(testDir, "git", "-c", "init.defaultBranch=main", "init")
		testutil.RunSucceedCommand(testDir, "git", "add", "-A")
		testutil.RunSucceedCommand(testDir, "git", "commit", "-m", "Initial commit")

		testutil.RunSucceedCommand(testDir, "docker", "compose", "up", "--detach")
		testutil.RunSucceedCommand(testDir, "docker", "compose", "run", "mc", "mb", "main/repo")
		testutil.RunSucceedCommand(testDir, "docker", "compose", "run", "mc", "policy", "set", "download", "main/repo")

		minioAddress = "http://127.0.0.1:" + composePort("minio", "9000")

		// The builder runs buildkit in its own container and cannot see images from
		// the local docker daemon, so the stub is served over a registry. Only a
		// loopback address is treated as insecure by default, and only network=host
		// lets the builder reach it.
		quillImage := "127.0.0.1:" + composePort("registry", "5000") + "/quill-stub:latest"
		testutil.RunSucceedCommand(testutil.FixturePath("quill_stub"), "docker", "build", "--tag", quillImage, ".")
		testutil.RunSucceedCommand(testutil.FixturePath("quill_stub"), "docker", "push", quillImage)

		setEnv("TRDL_QUILL_IMAGE", quillImage)
		setEnv("TRDL_BUILDX_DRIVER_OPTS_NETWORK", "network=host")
	})

	AfterEach(func() {
		testutil.RunSucceedCommand(testDir, "docker", "compose", "down")
	})

	It("signs Mach-O release artifacts through the quill stage", func() {
		By("[server] Configuring ...")
		handleRequest("configure", map[string]interface{}{
			"git_repo_url":                                     testDir,
			"git_trdl_channels_branch":                         "main",
			"initial_last_published_git_commit":                "",
			"required_number_of_verified_signatures_on_commit": 3,
			"s3_endpoint":                                      minioAddress,
			"s3_region":                                        "ru-central1",
			"s3_access_key_id":                                 "minioadmin",
			"s3_secret_access_key":                             "minioadmin",
			"s3_bucket_name":                                   "repo",
		})

		for _, user := range []string{"developer", "tl", "pm"} {
			data, err := os.ReadFile(testutil.FixturePath("pgp_keys", fmt.Sprintf("%s_public.pgp", user)))
			Expect(err).ShouldNot(HaveOccurred())
			handleRequest("configure/trusted_pgp_public_key", map[string]interface{}{
				"name":       user,
				"public_key": string(data),
			})
		}

		By("[server] Configuring mac signing credentials ...")
		handleRequest("configure/build/mac_signing_identity", map[string]interface{}{
			"data":          certificate,
			"password":      password,
			"notary_key_id": notaryKeyID,
			"notary_key":    notaryKey,
			"notary_issuer": notaryIssuer,
		})

		By("[git] Tagging and quorum-signing " + tag + " ...")
		testutil.RunSucceedCommand(
			testDir,
			"git",
			"-c", "tag.gpgsign=true",
			"-c", "user.signingkey="+pgpSigningKeyDeveloper,
			"tag", tag, "-m", "New version",
		)
		for _, key := range []string{pgpSigningKeyTL, pgpSigningKeyPM} {
			testutil.RunSucceedCommand(testDir, "git", "signatures", "add", "--key", key, tag)
			testutil.RunSucceedCommand(testDir, "git", "signatures", "add", "--key", key, "main")
		}

		By("[server] Releasing " + tag + " ...")
		resp := handleRequest("release", map[string]interface{}{"git_tag": tag})
		Expect(resp).ShouldNot(BeNil())
		taskUUID, ok := resp.Data["task_uuid"].(string)
		Expect(ok).Should(BeTrue(), fmt.Sprintf("%+v", resp.Data))
		tasksManagerTestutil.WaitForTaskSuccess(GinkgoWriter, GinkgoT(), context.Background(), backend, storage, taskUUID)

		By("[minio] Verifying the published artifact was signed by the quill stage ...")
		artifactURL := fmt.Sprintf("%s/repo/targets/releases/%s/any-any/bin/macho-stub", minioAddress, version)
		httpResp, err := http.Get(artifactURL)
		Expect(err).ShouldNot(HaveOccurred())
		defer func() { _ = httpResp.Body.Close() }()
		Expect(httpResp.StatusCode).Should(Equal(http.StatusOK))

		artifact, err := io.ReadAll(httpResp.Body)
		Expect(err).ShouldNot(HaveOccurred())

		Expect(string(artifact)).Should(HavePrefix(machOMagic),
			"the Mach-O stub payload must survive the signer stage")
		Expect(string(artifact)).Should(HaveSuffix(fmt.Sprintf("TRDL-E2E-STUB-SIGNED:%s:%s", certificate, notaryKeyID)),
			"the credential values stored in Vault must reach quill through the buildkit secret mounts")
	})
})
