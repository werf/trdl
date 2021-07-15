package git

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/werf/vault-plugin-secrets-trdl/pkg/pgp"
)

func VerifyTagSignatures(repo *git.Repository, tagName string, pgpKeys []string, requiredNumberOfVerifiedSignatures int) error {
	tr, err := repo.Tag(tagName)
	if err != nil {
		return fmt.Errorf("unable to get tag: %s", err)
	}

	to, err := repo.TagObject(tr.Hash())
	if err != nil {
		if err == plumbing.ErrObjectNotFound { // lightweight tag
			revHash, err := repo.ResolveRevision(plumbing.Revision(tr.Hash().String()))
			if err != nil {
				return fmt.Errorf("resolve revision %s failed: %s", tr.Hash(), err)
			}

			return VerifyCommitSignatures(repo, revHash.String(), pgpKeys, requiredNumberOfVerifiedSignatures)
		}

		return fmt.Errorf("unable to get tag object: %s", err)
	}

	if to.PGPSignature != "" {
		encoded := &plumbing.MemoryObject{}
		if err := to.EncodeWithoutSignature(encoded); err != nil {
			return fmt.Errorf("unable to encode tag object: %s", err)
		}

		pgpKeys, requiredNumberOfVerifiedSignatures, err = pgp.VerifyPGPSignatures([]string{to.PGPSignature}, func() (io.Reader, error) { return encoded.Reader() }, pgpKeys, requiredNumberOfVerifiedSignatures)
		if err != nil {
			return err
		}
	}

	if requiredNumberOfVerifiedSignatures == 0 {
		return nil
	}

	return verifyObjectSignatures(repo, to.Hash.String(), pgpKeys, requiredNumberOfVerifiedSignatures)
}

func VerifyCommitSignatures(repo *git.Repository, commit string, pgpKeys []string, requiredNumberOfVerifiedSignatures int) error {
	co, err := repo.CommitObject(plumbing.NewHash(commit))
	if err != nil {
		return fmt.Errorf("unable to get commit %q: %s", commit, err)
	}

	if co.PGPSignature != "" {
		encoded := &plumbing.MemoryObject{}
		if err := co.EncodeWithoutSignature(encoded); err != nil {
			return err
		}

		pgpKeys, requiredNumberOfVerifiedSignatures, err = pgp.VerifyPGPSignatures([]string{co.PGPSignature}, func() (io.Reader, error) { return encoded.Reader() }, pgpKeys, requiredNumberOfVerifiedSignatures)
		if err != nil {
			return err
		}
	}

	if requiredNumberOfVerifiedSignatures == 0 {
		return nil
	}

	return verifyObjectSignatures(repo, commit, pgpKeys, requiredNumberOfVerifiedSignatures)
}

func verifyObjectSignatures(repo *git.Repository, objectID string, pgpKeys []string, requiredNumberOfVerifiedSignatures int) error {
	signatures, err := objectSignaturesFromNotes(repo, objectID)
	if err != nil {
		return err
	}

	if len(signatures) == 0 {
		return fmt.Errorf("not enough pgp signatures")
	}

	pgpKeys, requiredNumberOfVerifiedSignatures, err = pgp.VerifyPGPSignatures(signatures, func() (io.Reader, error) { return strings.NewReader(objectID), nil }, pgpKeys, requiredNumberOfVerifiedSignatures)
	if err != nil {
		return err
	}

	if requiredNumberOfVerifiedSignatures != 0 {
		return fmt.Errorf("not enough verified pgp signatures: %d verified signature(s) required", requiredNumberOfVerifiedSignatures)
	}

	return nil
}

const notesReferenceName = "refs/tags/latest-signature"

func objectSignaturesFromNotes(repo *git.Repository, objectID string) ([]string, error) {
	ref, err := repo.Reference(notesReferenceName, true)
	if err != nil {
		if err == plumbing.ErrReferenceNotFound {
			return nil, nil
		}

		return nil, fmt.Errorf("unable to check existance of reference %q: %s", notesReferenceName, err)
	}

	refHeadCommit := ref.Hash()
	refCommitObj, err := repo.CommitObject(refHeadCommit)
	if err != nil {
		return nil, fmt.Errorf("unable to get objectID %q: %s", refHeadCommit, err)
	}

	tree, err := refCommitObj.Tree()
	if err != nil {
		return nil, fmt.Errorf("unable to get objectID %q tree: %s", refHeadCommit, err)
	}

	file, err := tree.File(objectID)
	if err != nil {
		if err == object.ErrFileNotFound {
			return nil, nil
		}

		return nil, fmt.Errorf("unable to get objectID %q tree file %s: %s", refHeadCommit, objectID, err)
	}

	r, err := file.Reader()
	if err != nil {
		return nil, fmt.Errorf("unable to get objectID %q tree file %s reader: %s", refHeadCommit, objectID, err)
	}

	var signatures []string
	s := bufio.NewScanner(r)
	for s.Scan() {
		line := s.Text()
		if line != "" {
			signatures = append(signatures, fmt.Sprintf(`-----BEGIN PGP SIGNATURE-----

%s
-----END PGP SIGNATURE-----`, base64LineToMultiline(line)))
		}
	}

	return signatures, nil
}

func base64LineToMultiline(base64Line string) string {
	var lines []string
	lineRunes := []rune(base64Line)
	for len(lineRunes) != 0 {
		var chunk []rune
		if len(lineRunes) >= 76 {
			chunk, lineRunes = lineRunes[:76], lineRunes[76:]
		} else {
			chunk, lineRunes = lineRunes, []rune{}
		}

		lines = append(lines, string(chunk))
	}

	return strings.Join(lines, "\n")
}
