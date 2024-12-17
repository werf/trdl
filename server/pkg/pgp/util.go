package pgp

import (
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/go-hclog"
	"golang.org/x/crypto/openpgp"
)

func VerifyPGPSignatures(pgpSignatures []string, signedReaderFunc func() (io.Reader, error), pgpKeys []string, requiredNumberOfVerifiedSignatures int, logger hclog.Logger) ([]string, int, error) {
	if requiredNumberOfVerifiedSignatures == 0 {
		return pgpKeys, 0, nil
	}

	signedReader, err := signedReaderFunc()
	if err != nil {
		return nil, 0, err
	}

	verifiedKeys := make([]string, 0, len(pgpKeys))
	for _, pgpKey := range pgpKeys {
		keyring, err := openpgp.ReadArmoredKeyRing(strings.NewReader(pgpKey))
		if err != nil {
			return nil, 0, err
		}

		for _, pgpSignature := range pgpSignatures {
			_, err := openpgp.CheckArmoredDetachedSignature(keyring, signedReader, strings.NewReader(pgpSignature))
			if err != nil {
				if logger != nil {
					logger.Debug(fmt.Sprintf("[DEBUG-SIGNATURES] VerifyPGPSignatures -- will skip pgpKey due to error: %s\n>%v<", err, pgpKey))
				}
				continue
			}
			requiredNumberOfVerifiedSignatures--
			if requiredNumberOfVerifiedSignatures == 0 {
				return verifiedKeys, 0, nil
			}
			verifiedKeys = append(verifiedKeys, pgpKey)
			break
		}
	}
	return verifiedKeys, requiredNumberOfVerifiedSignatures, nil
}
