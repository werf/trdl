package pgp

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/hashicorp/go-hclog"
	"github.com/samber/lo"
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

			signer, err := openpgp.CheckArmoredDetachedSignature(keyring, signedReader, strings.NewReader(pgpSignature), nil)
			if err != nil {
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

			pgpKeys, err = withoutPGPKeysOfEntity(pgpKeys, signer)
			if err != nil {
				return nil, 0, err
			}
			break
		}
	}

	return pgpKeys, requiredNumberOfVerifiedSignatures, nil
}

// withoutPGPKeysOfEntity drops every trusted key containing the entity, so a signer whose key is trusted
// under several names is counted only once.
func withoutPGPKeysOfEntity(pgpKeys []string, entity *openpgp.Entity) ([]string, error) {
	var result []string
	for _, pgpKey := range pgpKeys {
		keyring, err := openpgp.ReadArmoredKeyRing(strings.NewReader(pgpKey))
		if err != nil {
			return nil, fmt.Errorf("read trusted pgp public key: %w", err)
		}

		if !keyringContainsEntity(keyring, entity) {
			result = append(result, pgpKey)
		}
	}

	return result, nil
}

func keyringContainsEntity(keyring openpgp.EntityList, entity *openpgp.Entity) bool {
	return lo.ContainsBy(keyring, func(e *openpgp.Entity) bool {
		return bytes.Equal(e.PrimaryKey.Fingerprint, entity.PrimaryKey.Fingerprint)
	})
}
