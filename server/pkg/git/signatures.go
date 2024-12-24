package git

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/hashicorp/go-hclog"

	"github.com/werf/trdl/server/pkg/pgp"
)

type NotEnoughVerifiedPGPSignaturesError struct {
	Number int
}

func (r *NotEnoughVerifiedPGPSignaturesError) Error() string {
	return fmt.Sprintf("not enough verified PGP signatures: %d verified signature(s) required", r.Number)
}

func NewNotEnoughVerifiedPGPSignaturesError(number int) error {
	return &NotEnoughVerifiedPGPSignaturesError{Number: number}
}

func VerifyTagSignatures(repo *git.Repository, tagName string, trustedPGPPublicKeys []string, requiredNumberOfVerifiedSignatures int, logger hclog.Logger) error {
	tr, err := repo.Tag(tagName)
	if err != nil {
		return fmt.Errorf("unable to get tag: %w", err)
	}

	to, err := repo.TagObject(tr.Hash())
	if err != nil {
		if err == plumbing.ErrObjectNotFound { // lightweight tag
			revHash, err := repo.ResolveRevision(plumbing.Revision(tr.Hash().String()))
			if err != nil {
				return fmt.Errorf("resolve revision %s failed: %w", tr.Hash(), err)
			}

			return VerifyCommitSignatures(repo, revHash.String(), trustedPGPPublicKeys, requiredNumberOfVerifiedSignatures, logger)
		}

		return fmt.Errorf("unable to get tag object: %w", err)
	}

	if to.PGPSignature != "" {
		encoded := &plumbing.MemoryObject{}
		if err := to.EncodeWithoutSignature(encoded); err != nil {
			return fmt.Errorf("unable to encode tag object: %w", err)
		}

		trustedPGPPublicKeys, requiredNumberOfVerifiedSignatures, err = pgp.VerifyPGPSignatures([]string{to.PGPSignature}, func() (io.Reader, error) { return encoded.Reader() }, trustedPGPPublicKeys, requiredNumberOfVerifiedSignatures, logger)
		if err != nil {
			return err
		}
	}

	if requiredNumberOfVerifiedSignatures == 0 {
		return nil
	}

	return verifyObjectSignatures(repo, to.Hash.String(), trustedPGPPublicKeys, requiredNumberOfVerifiedSignatures, logger)
}

func VerifyCommitSignatures(repo *git.Repository, commit string, trustedPGPPublicKeys []string, requiredNumberOfVerifiedSignatures int, logger hclog.Logger) error {
	co, err := repo.CommitObject(plumbing.NewHash(commit))
	if err != nil {
		return fmt.Errorf("unable to get commit %q: %w", commit, err)
	}

	if co.PGPSignature != "" {
		encoded := &plumbing.MemoryObject{}
		if err := co.EncodeWithoutSignature(encoded); err != nil {
			return err
		}

		trustedPGPPublicKeys, requiredNumberOfVerifiedSignatures, err = pgp.VerifyPGPSignatures([]string{co.PGPSignature}, func() (io.Reader, error) { return encoded.Reader() }, trustedPGPPublicKeys, requiredNumberOfVerifiedSignatures, logger)
		if err != nil {
			return err
		}
	}

	if requiredNumberOfVerifiedSignatures == 0 {
		return nil
	}

	return verifyObjectSignatures(repo, commit, trustedPGPPublicKeys, requiredNumberOfVerifiedSignatures, logger)
}

func verifyObjectSignatures(repo *git.Repository, objectID string, trustedPGPPublicKeys []string, requiredNumberOfVerifiedSignatures int, logger hclog.Logger) error {
	signatures, err := objectSignaturesFromNotes(repo, objectID)
	if err != nil {
		if errors.Is(err, plumbing.ErrObjectNotFound) {
			logger.Debug(fmt.Sprintf("[DEBUG-SIGNATURES] git object not found (%s): exiting", objectID))
			return NewNotEnoughVerifiedPGPSignaturesError(requiredNumberOfVerifiedSignatures)
		}

		return err
	}

	if logger != nil {
		logger.Debug(fmt.Sprintf("[DEBUG-SIGNATURES] verifyObjectSignatures objectSignaturesFromNotes >%v<", signatures))
	}

	if len(signatures) == 0 {
		if logger != nil {
			logger.Debug("[DEBUG-SIGNATURES] no signatures: exiting")
		}
		return NewNotEnoughVerifiedPGPSignaturesError(requiredNumberOfVerifiedSignatures)
	}

	_, requiredNumberOfVerifiedSignatures, err = pgp.VerifyPGPSignatures(signatures, func() (io.Reader, error) { return strings.NewReader(objectID), nil }, trustedPGPPublicKeys, requiredNumberOfVerifiedSignatures, logger)
	if err != nil {
		return err
	}

	if requiredNumberOfVerifiedSignatures != 0 {
		if logger != nil {
			logger.Debug("[DEBUG-SIGNATURES] required number of verified signatures not met: exiting")
		}
		return NewNotEnoughVerifiedPGPSignaturesError(requiredNumberOfVerifiedSignatures)
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

		return nil, fmt.Errorf("unable to check existence of reference %q: %w", notesReferenceName, err)
	}

	refHeadCommit := ref.Hash()

	// for annotated tags it needs to get the tag object and then its target object.
	obj, err := repo.Object(plumbing.AnyObject, refHeadCommit)
	if err != nil {
		return nil, fmt.Errorf("unable to get object %q: %w", refHeadCommit, err)
	}

	if tagObj, ok := obj.(*object.Tag); ok {
		refHeadCommit = tagObj.Target
	}

	refCommitObj, err := repo.CommitObject(refHeadCommit)
	if err != nil {
		return nil, fmt.Errorf("unable to get objectID %q: %w", refHeadCommit, err)
	}

	tree, err := refCommitObj.Tree()
	if err != nil {
		return nil, fmt.Errorf("unable to get objectID %q tree: %w", refHeadCommit, err)
	}

	var file *object.File

FindObjectFile:
	for _, path := range objectFanoutPaths(objectID) {
		var err error
		file, err = tree.File(path)
		switch {
		case err == object.ErrFileNotFound:
			continue
		case err != nil:
			return nil, fmt.Errorf("unable to get objectID %q tree file %s: %w", refHeadCommit, path, err)
		default:
			break FindObjectFile
		}
	}

	if file == nil {
		return nil, nil
	}

	r, err := file.Reader()
	if err != nil {
		return nil, fmt.Errorf("unable to get objectID %q tree file %s reader: %w", refHeadCommit, objectID, err)
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
