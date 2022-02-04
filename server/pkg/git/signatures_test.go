package git

import (
	_ "embed"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/werf/trdl/server/pkg/testutil"
)

var (
	//go:embed _fixtures/pgp_keys/developer_public.pgp
	publicPGPKeyDataDeveloper []byte

	//go:embed _fixtures/pgp_keys/tl_public.pgp
	publicPGPKeyDataTL []byte

	//go:embed _fixtures/pgp_keys/pm_public.pgp
	publicPGPKeyDataPM []byte
)

var _ = Describe("VerifyTagSignatures and VerifyCommitSignatures", func() {

	const (
		pgpSigningKeyDeveloper = "74E1259029B147CB4033E8B80D4C9C140E8A1030"
		pgpSigningKeyTL        = "2BA55FD8158034EEBE92AA9ED9D79B63AFC30C7A"
		pgpSigningKeyPM        = "C353F279F552B3EF16DAE0A64354E51BF178F735"

		branchName   = "main"
		tagName      = "v1.0.0"
		msgNewCommit = "New commit"
		msgNewTag    = "New version"

		kindNameLightweightTag = "lightweight tag"
		kindNameAnnotatedTag   = "annotated tag"
		kindNameCommit         = "commit"
	)

	addSignatureToGitNotes := func(pgpSigningKey, ref string) {
		testutil.RunSucceedCommand(
			testDir,
			"git",
			"signatures", "add", "--key", pgpSigningKey, ref,
		)
	}

	addTagSignatureToGitNotes := func(pgpSigningKey string) {
		addSignatureToGitNotes(pgpSigningKey, tagName)
	}

	addCommitSignatureToGitNotes := func(pgpSigningKey string) {
		addSignatureToGitNotes(pgpSigningKey, branchName)
	}

	addLightweightTagWithoutSignature := func() {
		testutil.RunSucceedCommand(
			testDir,
			"git",
			"-c", "tag.gpgsign=false",
			"tag", tagName,
		)
	}

	addLightweightTagWithRegularSignature := func(pgpSigningKey string) {
		testutil.RunSucceedCommand(
			testDir,
			"git",
			"-c", "commit.gpgsign=true", "-c", "user.signingkey="+pgpSigningKey,
			"commit", "--allow-empty", "-m", msgNewCommit,
		)

		testutil.RunSucceedCommand(
			testDir,
			"git",
			"-c", "tag.gpgsign=false",
			"tag", tagName,
		)
	}

	addAnnotatedTagWithoutSignature := func() {
		testutil.RunSucceedCommand(
			testDir,
			"git",
			"-c", "tag.gpgsign=false",
			"tag", tagName, "-m", msgNewTag,
		)
	}

	addAnnotatedTagWithRegularSignature := func(pgpSigningKey string) {
		testutil.RunSucceedCommand(
			testDir,
			"git",
			"-c", "tag.gpgsign=true", "-c", "user.signingkey="+pgpSigningKey,
			"tag", tagName, "-m", msgNewTag,
		)
	}

	addCommitWithoutSignature := func() {
		testutil.RunSucceedCommand(
			testDir,
			"git",
			"-c", "commit.gpgsign=false",
			"commit", "--allow-empty", "-m", msgNewCommit,
		)
	}

	addCommitWithRegularSignature := func(pgpSigningKey string) {
		testutil.RunSucceedCommand(
			testDir,
			"git",
			"-c", "commit.gpgsign=true", "-c", "user.signingkey="+pgpSigningKey,
			"commit", "--allow-empty", "-m", msgNewCommit,
		)
	}

	type tableEntry struct {
		trustedPGPPublicKeys               []string
		requiredNumberOfVerifiedSignatures int
		expectedErrMsg                     string
	}

	tableItBodyTagFunc := func(entry tableEntry) {
		repo, err := CloneInMemory(testDir, CloneOptions{})
		Ω(err).ShouldNot(HaveOccurred())

		err = VerifyTagSignatures(
			repo,
			tagName,
			entry.trustedPGPPublicKeys,
			entry.requiredNumberOfVerifiedSignatures,
		)

		if entry.expectedErrMsg != "" {
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(BeEquivalentTo(entry.expectedErrMsg))
		} else {
			Ω(err).ShouldNot(HaveOccurred())
		}
	}

	tableItBodyCommitFunc := func(entry tableEntry) {
		repo, err := CloneInMemory(testDir, CloneOptions{})
		Ω(err).ShouldNot(HaveOccurred())

		head, err := repo.Head()
		Ω(err).ShouldNot(HaveOccurred())

		headCommit := head.Hash()
		err = VerifyCommitSignatures(
			repo,
			headCommit.String(),
			entry.trustedPGPPublicKeys,
			entry.requiredNumberOfVerifiedSignatures,
		)

		if entry.expectedErrMsg != "" {
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(BeEquivalentTo(entry.expectedErrMsg))
		} else {
			Ω(err).ShouldNot(HaveOccurred())
		}
	}

	tableItBodyFuncByKind := map[string]func(e tableEntry){
		kindNameLightweightTag: tableItBodyTagFunc,
		kindNameAnnotatedTag:   tableItBodyTagFunc,
		kindNameCommit:         tableItBodyCommitFunc,
	}

	var addWithoutSignatureByKind = map[string]func(){
		kindNameLightweightTag: addLightweightTagWithoutSignature,
		kindNameAnnotatedTag:   addAnnotatedTagWithoutSignature,
		kindNameCommit:         addCommitWithoutSignature,
	}

	var addWithRegularSignatureByKind = map[string]func(pgpSigningKey string){
		kindNameLightweightTag: addLightweightTagWithRegularSignature,
		kindNameAnnotatedTag:   addAnnotatedTagWithRegularSignature,
		kindNameCommit:         addCommitWithRegularSignature,
	}

	var addSignatureToGitNotesByKind = map[string]func(pgpSigningKey string){
		kindNameLightweightTag: addTagSignatureToGitNotes,
		kindNameAnnotatedTag:   addTagSignatureToGitNotes,
		kindNameCommit:         addCommitSignatureToGitNotes,
	}

	for _, kind := range []string{kindNameLightweightTag, kindNameAnnotatedTag, kindNameCommit} {
		kind := kind

		BeforeEach(func() {
			testutil.RunSucceedCommand(
				testDir,
				"git",
				"-c", "init.defaultBranch="+branchName,
				"init",
			)

			testutil.RunSucceedCommand(
				testDir,
				"git",
				"commit", "--allow-empty", "-m", "Initial commit",
			)
		})

		Context(kind+" not signed", func() {
			BeforeEach(func() {
				addWithoutSignatureByKind[kind]()
			})

			DescribeTable("perform signature verification", tableItBodyFuncByKind[kind],
				Entry("without trustedPGPPublicKeys and requiredNumberOfVerifiedSignatures", tableEntry{
					trustedPGPPublicKeys:               []string{},
					requiredNumberOfVerifiedSignatures: 0,
				}),
				Entry("with trustedPGPPublicKeys and without requiredNumberOfVerifiedSignatures", tableEntry{
					trustedPGPPublicKeys:               []string{string(publicPGPKeyDataDeveloper)},
					requiredNumberOfVerifiedSignatures: 0,
				}),
				Entry("without trustedPGPPublicKeys and with requiredNumberOfVerifiedSignatures", tableEntry{
					trustedPGPPublicKeys:               []string{},
					requiredNumberOfVerifiedSignatures: 1,
					expectedErrMsg:                     NewNotEnoughVerifiedPGPSignaturesError(1).Error(),
				}),
				Entry("with trustedPGPPublicKeys and requiredNumberOfVerifiedSignatures", tableEntry{
					trustedPGPPublicKeys:               []string{string(publicPGPKeyDataDeveloper)},
					requiredNumberOfVerifiedSignatures: 1,
					expectedErrMsg:                     NewNotEnoughVerifiedPGPSignaturesError(1).Error(),
				}),
			)
		})

		Context(kind+" with regular signature", func() {
			BeforeEach(func() {
				addWithRegularSignatureByKind[kind](pgpSigningKeyDeveloper)
			})

			DescribeTable("perform signature verification", tableItBodyFuncByKind[kind],
				Entry("without trustedPGPPublicKeys and requiredNumberOfVerifiedSignatures", tableEntry{
					trustedPGPPublicKeys:               []string{},
					requiredNumberOfVerifiedSignatures: 0,
				}),
				Entry("with trustedPGPPublicKeys and without requiredNumberOfVerifiedSignatures", tableEntry{
					trustedPGPPublicKeys:               []string{string(publicPGPKeyDataDeveloper)},
					requiredNumberOfVerifiedSignatures: 0,
				}),
				Entry("without trustedPGPPublicKeys and with requiredNumberOfVerifiedSignatures", tableEntry{
					trustedPGPPublicKeys:               []string{},
					requiredNumberOfVerifiedSignatures: 1,
					expectedErrMsg:                     NewNotEnoughVerifiedPGPSignaturesError(1).Error(),
				}),
				Entry("with trustedPGPPublicKeys (1 key) and requiredNumberOfVerifiedSignatures (1)", tableEntry{
					trustedPGPPublicKeys:               []string{string(publicPGPKeyDataDeveloper)},
					requiredNumberOfVerifiedSignatures: 1,
				}),
				Entry("with trustedPGPPublicKeys (1 key) and requiredNumberOfVerifiedSignatures (2)", tableEntry{
					trustedPGPPublicKeys:               []string{string(publicPGPKeyDataDeveloper)},
					requiredNumberOfVerifiedSignatures: 2,
					expectedErrMsg:                     NewNotEnoughVerifiedPGPSignaturesError(1).Error(),
				}),
			)
		})

		Context(kind+" with signature in git notes", func() {
			BeforeEach(func() {
				addWithoutSignatureByKind[kind]()
				addSignatureToGitNotesByKind[kind](pgpSigningKeyDeveloper)
			})

			DescribeTable("perform signature verification", tableItBodyFuncByKind[kind],
				Entry("without trustedPGPPublicKeys and requiredNumberOfVerifiedSignatures", tableEntry{
					trustedPGPPublicKeys:               []string{},
					requiredNumberOfVerifiedSignatures: 0,
				}),
				Entry("with trustedPGPPublicKeys and without requiredNumberOfVerifiedSignatures", tableEntry{
					trustedPGPPublicKeys:               []string{string(publicPGPKeyDataDeveloper)},
					requiredNumberOfVerifiedSignatures: 0,
				}),
				Entry("without trustedPGPPublicKeys and with requiredNumberOfVerifiedSignatures", tableEntry{
					trustedPGPPublicKeys:               []string{},
					requiredNumberOfVerifiedSignatures: 1,
					expectedErrMsg:                     NewNotEnoughVerifiedPGPSignaturesError(1).Error(),
				}),
				Entry("with trustedPGPPublicKeys (1 key) and requiredNumberOfVerifiedSignatures (1)", tableEntry{
					trustedPGPPublicKeys:               []string{string(publicPGPKeyDataDeveloper)},
					requiredNumberOfVerifiedSignatures: 1,
				}),
				Entry("with trustedPGPPublicKeys (1 key) and requiredNumberOfVerifiedSignatures (2)", tableEntry{
					trustedPGPPublicKeys:               []string{string(publicPGPKeyDataDeveloper)},
					requiredNumberOfVerifiedSignatures: 2,
					expectedErrMsg:                     NewNotEnoughVerifiedPGPSignaturesError(1).Error(),
				}),
			)
		})

		Context(kind+" with two signatures in git notes", func() {
			BeforeEach(func() {
				addWithoutSignatureByKind[kind]()
				addSignatureToGitNotesByKind[kind](pgpSigningKeyDeveloper)
				addSignatureToGitNotesByKind[kind](pgpSigningKeyTL)
			})

			DescribeTable("perform signature verification", tableItBodyFuncByKind[kind],
				Entry("with more trustedPGPPublicKeys then requiredNumberOfVerifiedSignatures", tableEntry{
					trustedPGPPublicKeys:               []string{string(publicPGPKeyDataDeveloper), string(publicPGPKeyDataTL)},
					requiredNumberOfVerifiedSignatures: 1,
				}),
				Entry("with the same amount trustedPGPPublicKeys as requiredNumberOfVerifiedSignatures", tableEntry{
					trustedPGPPublicKeys:               []string{string(publicPGPKeyDataDeveloper), string(publicPGPKeyDataTL)},
					requiredNumberOfVerifiedSignatures: 2,
				}),
				Entry("with less trustedPGPPublicKeys then requiredNumberOfVerifiedSignatures", tableEntry{
					trustedPGPPublicKeys:               []string{string(publicPGPKeyDataDeveloper), string(publicPGPKeyDataTL)},
					requiredNumberOfVerifiedSignatures: 3,
					expectedErrMsg:                     NewNotEnoughVerifiedPGPSignaturesError(1).Error(),
				}),
			)
		})

		Context(kind+" with regular signature and two signatures in git notes", func() {
			BeforeEach(func() {
				addWithRegularSignatureByKind[kind](pgpSigningKeyDeveloper)
				addSignatureToGitNotesByKind[kind](pgpSigningKeyTL)
				addSignatureToGitNotesByKind[kind](pgpSigningKeyPM)
			})

			DescribeTable("perform signature verification", tableItBodyFuncByKind[kind],
				Entry("with more trustedPGPPublicKeys then requiredNumberOfVerifiedSignatures", tableEntry{
					trustedPGPPublicKeys:               []string{string(publicPGPKeyDataDeveloper), string(publicPGPKeyDataTL), string(publicPGPKeyDataPM)},
					requiredNumberOfVerifiedSignatures: 1,
				}),
				Entry("with the same amount trustedPGPPublicKeys as requiredNumberOfVerifiedSignatures", tableEntry{
					trustedPGPPublicKeys:               []string{string(publicPGPKeyDataDeveloper), string(publicPGPKeyDataTL), string(publicPGPKeyDataPM)},
					requiredNumberOfVerifiedSignatures: 3,
				}),
				Entry("with less trustedPGPPublicKeys then requiredNumberOfVerifiedSignatures", tableEntry{
					trustedPGPPublicKeys:               []string{string(publicPGPKeyDataDeveloper), string(publicPGPKeyDataTL), string(publicPGPKeyDataPM)},
					requiredNumberOfVerifiedSignatures: 4,
					expectedErrMsg:                     NewNotEnoughVerifiedPGPSignaturesError(1).Error(),
				}),
			)
		})

		Context(kind+" with three signatures by one signer", func() {
			BeforeEach(func() {
				addWithRegularSignatureByKind[kind](pgpSigningKeyDeveloper)
				addSignatureToGitNotesByKind[kind](pgpSigningKeyDeveloper)
				addSignatureToGitNotesByKind[kind](pgpSigningKeyDeveloper)
			})

			DescribeTable("perform signature verification", tableItBodyFuncByKind[kind],
				Entry("with trustedPGPPublicKeys (1) and requiredNumberOfVerifiedSignatures (1)", tableEntry{
					trustedPGPPublicKeys:               []string{string(publicPGPKeyDataDeveloper)},
					requiredNumberOfVerifiedSignatures: 1,
				}),
				Entry("with the same amount trustedPGPPublicKeys as requiredNumberOfVerifiedSignatures", tableEntry{
					trustedPGPPublicKeys:               []string{string(publicPGPKeyDataDeveloper)},
					requiredNumberOfVerifiedSignatures: 3,
					expectedErrMsg:                     NewNotEnoughVerifiedPGPSignaturesError(2).Error(),
				}),
			)
		})
	}
})
