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

	verifiedKeys := make([]string, 0, len(pgpKeys))
	for _, pgpKey := range pgpKeys {
		keyring, err := openpgp.ReadArmoredKeyRing(strings.NewReader(pgpKey))
		if err != nil {
			return nil, 0, err
		}

		verified := false
		signedReader, err := signedReaderFunc()
		if err != nil {
			return nil, 0, err
		}
		for _, pgpSignature := range pgpSignatures {
			_, err := openpgp.CheckArmoredDetachedSignature(keyring, signedReader, strings.NewReader(pgpSignature))
			if err != nil {
				if logger != nil {
					logger.Debug(fmt.Sprintf("[DEBUG-SIGNATURES] Signature verification failed: %s\n>%v<", err, pgpKey))
				}
				continue
			}
			verified = true
			break
		}

		if verified {
			requiredNumberOfVerifiedSignatures--
			verifiedKeys = append(verifiedKeys, pgpKey)
			if requiredNumberOfVerifiedSignatures == 0 {
				return verifiedKeys, 0, nil
			}
		}
	}

	return verifiedKeys, requiredNumberOfVerifiedSignatures, nil
}
