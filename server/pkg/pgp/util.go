package pgp

import (
	"io"
	"strings"

	"golang.org/x/crypto/openpgp"
)

func VerifyPGPSignatures(pgpSignatures []string, signedReaderFunc func() (io.Reader, error), pgpKeys []string, requiredNumberOfVerifiedSignatures int) ([]string, int, error) {
	for _, pgpSignature := range pgpSignatures {
		i := 0
		l := len(pgpKeys)
		for i < l {
			keyring, err := openpgp.ReadArmoredKeyRing(strings.NewReader(pgpKeys[i]))
			if err != nil {
				return nil, 0, err
			}

			signedReader, err := signedReaderFunc()
			if err != nil {
				return nil, 0, err
			}

			if _, err = openpgp.CheckArmoredDetachedSignature(keyring, signedReader, strings.NewReader(pgpSignature)); err != nil {
				i++
				continue
			}

			requiredNumberOfVerifiedSignatures--
			if requiredNumberOfVerifiedSignatures == 0 {
				return nil, 0, nil
			}

			pgpKeys = append(append([]string{}, pgpKeys[:i]...), pgpKeys[i+1:]...)
			break
		}
	}

	return pgpKeys, requiredNumberOfVerifiedSignatures, nil
}
