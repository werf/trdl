package elf_signing

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"testing"

	"github.com/deckhouse/delivery-kit-sdk/pkg/signver"
	"github.com/deckhouse/delivery-kit-sdk/pkg/signver/hashivault"
	"github.com/deckhouse/delivery-kit-sdk/test/pkg/cert_utils"
	"github.com/onsi/gomega"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/stretchr/testify/require"
)

func generateCerts(t *testing.T, password string) cert_utils.GenerateCertificatesResult {
	t.Helper()
	gomega.RegisterTestingT(t)

	opts := cert_utils.GenerateCertificatesOptions{
		KeyType:           cert_utils.KeyType_ECDSA_P256,
		UseBase64Encoding: true,
	}
	if password != "" {
		opts.PassFunc = cryptoutils.StaticPasswordFunc([]byte(password))
	}

	return cert_utils.GenerateCertificatesWithOptions(opts)
}

// reencodeKeyPemType rebuilds a base64-encoded PEM private key from an existing
// base64-encoded PEM key, replacing its PEM type.
func reencodeKeyPemType(t *testing.T, encodedKey, pemType string) string {
	t.Helper()

	decoded, err := base64.StdEncoding.DecodeString(encodedKey)
	require.NoError(t, err)

	block, _ := pem.Decode(decoded)
	require.NotNil(t, block)

	reencoded := pem.EncodeToMemory(&pem.Block{Type: pemType, Bytes: block.Bytes})
	return base64.StdEncoding.EncodeToString(reencoded)
}

// encodeRawKey produces a base64-encoded PEM block whose bytes are a raw PKCS8
// key that is not encrypted with go-securesystemslib, so decryption fails.
func encodeRawKey(t *testing.T, certs cert_utils.GenerateCertificatesResult) string {
	t.Helper()

	pkcs8, err := x509.MarshalPKCS8PrivateKey(certs.PrivKey)
	require.NoError(t, err)

	block := pem.EncodeToMemory(&pem.Block{Type: signver.PrivateKeyPemType, Bytes: pkcs8})
	return base64.StdEncoding.EncodeToString(block)
}

func validVaultOpts() VaultSignerOpts {
	return VaultSignerOpts{
		Address:      "https://vault.example.com",
		TransitPath:  "transit",
		AuthPath:     "ar",
		AuthRoleID:   "role-id",
		AuthSecretID: "secret-id",
	}
}

func TestValidateELFSettings(t *testing.T) {
	t.Run("valid local key without password", func(t *testing.T) {
		certs := generateCerts(t, "")
		settings := SignerSettings{
			KeyRef:  certs.PrivRef,
			CertRef: certs.LeafRef,
		}
		require.NoError(t, validateSettings(settings))
	})

	t.Run("valid local key with password", func(t *testing.T) {
		certs := generateCerts(t, "s3cret")
		settings := SignerSettings{
			KeyRef:      certs.PrivRef,
			KeyPassword: "s3cret",
			CertRef:     certs.LeafRef,
		}
		require.NoError(t, validateSettings(settings))
	})

	t.Run("valid local key with intermediates", func(t *testing.T) {
		certs := generateCerts(t, "")
		settings := SignerSettings{
			KeyRef:           certs.PrivRef,
			CertRef:          certs.LeafRef,
			IntermediatesRef: certs.IntermediatesRef,
		}
		require.NoError(t, validateSettings(settings))
	})

	t.Run("invalid base64 key", func(t *testing.T) {
		certs := generateCerts(t, "")
		settings := SignerSettings{
			KeyRef:  "not!base64",
			CertRef: certs.LeafRef,
		}
		require.ErrorContains(t, validateSettings(settings), "decode base64 key")
	})

	t.Run("not a pem block", func(t *testing.T) {
		certs := generateCerts(t, "")
		settings := SignerSettings{
			KeyRef:  base64.StdEncoding.EncodeToString([]byte("just some text")),
			CertRef: certs.LeafRef,
		}
		require.ErrorContains(t, validateSettings(settings), "invalid pem block")
	})

	t.Run("unsupported pem type", func(t *testing.T) {
		certs := generateCerts(t, "")
		settings := SignerSettings{
			KeyRef:  reencodeKeyPemType(t, certs.PrivRef, "RSA PRIVATE KEY"),
			CertRef: certs.LeafRef,
		}
		require.ErrorContains(t, validateSettings(settings), "unsupported pem type")
	})

	t.Run("wrong password", func(t *testing.T) {
		certs := generateCerts(t, "correct")
		settings := SignerSettings{
			KeyRef:      certs.PrivRef,
			KeyPassword: "wrong",
			CertRef:     certs.LeafRef,
		}
		require.ErrorContains(t, validateSettings(settings), "decrypt private key")
	})

	t.Run("undecryptable key bytes", func(t *testing.T) {
		certs := generateCerts(t, "")
		settings := SignerSettings{
			KeyRef:  encodeRawKey(t, certs),
			CertRef: certs.LeafRef,
		}
		require.ErrorContains(t, validateSettings(settings), "decrypt private key")
	})

	t.Run("invalid base64 certificate", func(t *testing.T) {
		certs := generateCerts(t, "")
		settings := SignerSettings{
			KeyRef:  certs.PrivRef,
			CertRef: "not!base64",
		}
		require.ErrorContains(t, validateSettings(settings), "decode base64 certificate")
	})

	t.Run("invalid base64 intermediates", func(t *testing.T) {
		certs := generateCerts(t, "")
		settings := SignerSettings{
			KeyRef:           certs.PrivRef,
			CertRef:          certs.LeafRef,
			IntermediatesRef: "not!base64",
		}
		require.ErrorContains(t, validateSettings(settings), "decode base64 intermediates certificate")
	})

	t.Run("valid vault key reference", func(t *testing.T) {
		certs := generateCerts(t, "")
		settings := SignerSettings{
			KeyRef:    hashivault.ReferenceScheme + "my-key",
			CertRef:   certs.LeafRef,
			VaultOpts: validVaultOpts(),
		}
		require.NoError(t, validateSettings(settings))
	})

	t.Run("unknown key reference scheme", func(t *testing.T) {
		certs := generateCerts(t, "")
		settings := SignerSettings{
			KeyRef:    "awskms://my-key",
			CertRef:   certs.LeafRef,
			VaultOpts: validVaultOpts(),
		}
		require.ErrorContains(t, validateSettings(settings), "invalid key reference")
	})

	vaultFieldCases := []struct {
		name    string
		mutate  func(*VaultSignerOpts)
		wantMsg string
	}{
		{
			name:    "missing vault address",
			mutate:  func(o *VaultSignerOpts) { o.Address = "" },
			wantMsg: `"vault_addr" is required for vault key reference`,
		},
		{
			name:    "missing vault transit path",
			mutate:  func(o *VaultSignerOpts) { o.TransitPath = "" },
			wantMsg: `"vault_transit_path" is required for vault key reference`,
		},
		{
			name:    "missing vault auth role id",
			mutate:  func(o *VaultSignerOpts) { o.AuthRoleID = "" },
			wantMsg: `"vault_auth_role_id" is required for vault key reference`,
		},
		{
			name:    "missing vault auth secret id",
			mutate:  func(o *VaultSignerOpts) { o.AuthSecretID = "" },
			wantMsg: `"vault_auth_secret_id" is required for vault key reference`,
		},
		{
			name:    "missing vault auth path",
			mutate:  func(o *VaultSignerOpts) { o.AuthPath = "" },
			wantMsg: `"vault_auth_path" is required for vault key reference`,
		},
	}

	for _, tc := range vaultFieldCases {
		t.Run(tc.name, func(t *testing.T) {
			certs := generateCerts(t, "")
			opts := validVaultOpts()
			tc.mutate(&opts)
			settings := SignerSettings{
				KeyRef:    hashivault.ReferenceScheme + "my-key",
				CertRef:   certs.LeafRef,
				VaultOpts: opts,
			}
			require.ErrorContains(t, validateSettings(settings), tc.wantMsg)
		})
	}
}
