package pgp

import (
	"fmt"
	"io"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/hashicorp/go-hclog"
)

func VerifyPGPSignatures(pgpSignatures []string, signedReaderFunc func() (io.Reader, error), pgpKeys []string, requiredNumberOfVerifiedSignatures int, logger hclog.Logger) ([]string, int, error) {
	if requiredNumberOfVerifiedSignatures == 0 {
		return pgpKeys, 0, nil
	}

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

			if _, err = openpgp.CheckArmoredDetachedSignature(keyring, signedReader, strings.NewReader(pgpSignature), nil); err != nil {
				if logger != nil {
					logger.Debug(fmt.Sprintf("[DEBUG-SIGNATURES] VerifyPGPSignatures -- will skip pgpKey due to error: %s\n>%v<", err, pgpKeys[i]))
				}
				i++
				continue
			}

			requiredNumberOfVerifiedSignatures--
			if requiredNumberOfVerifiedSignatures == 0 {
				return pgpKeys, 0, nil
			}

			pgpKeys = append(append([]string{}, pgpKeys[:i]...), pgpKeys[i+1:]...)
			break
		}
	}

	return pgpKeys, requiredNumberOfVerifiedSignatures, nil
}
