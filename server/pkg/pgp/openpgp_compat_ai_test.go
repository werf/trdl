package pgp

import (
	"bytes"
	"encoding/hex"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/stretchr/testify/require"
)

const legacyKeyFingerprint = "207775ba4caa06b36933254ac70f7f22be8fb479"

func TestAI_LegacyOpenPGPArtifactsRemainUsable(t *testing.T) {
	privateKey, err := os.ReadFile("testdata/legacy_private_key.asc")
	require.NoError(t, err)
	publicKey, err := os.ReadFile("testdata/legacy_public_key.asc")
	require.NoError(t, err)
	signedData, err := os.ReadFile("testdata/legacy_signed_data")
	require.NoError(t, err)
	armoredSignature, err := os.ReadFile("testdata/legacy_armored_signature.asc")
	require.NoError(t, err)
	binarySignature, err := os.ReadFile("testdata/legacy_detached_signature.bin")
	require.NoError(t, err)

	t.Run("private key stored by a previous release still parses", func(t *testing.T) {
		key, err := ParseRSASigningKey(bytes.NewReader(privateKey))
		require.NoError(t, err)
		require.Equal(t, legacyKeyFingerprint, hex.EncodeToString(key.Entity.PrimaryKey.Fingerprint))
		require.NotNil(t, key.Entity.PrivateKey)

		signature := bytes.NewBuffer(nil)
		require.NoError(t, SignDataStream(signature, bytes.NewReader(signedData), key))
		require.NotEmpty(t, signature.Bytes())
	})

	t.Run("trusted public key stored by a previous release stays valid", func(t *testing.T) {
		require.NoError(t, IsValidGPGPublicKey(string(publicKey)))
	})

	t.Run("armored gpg signature verifies against the legacy public key", func(t *testing.T) {
		remainingKeys, remainingRequired, err := VerifyPGPSignatures(
			[]string{string(armoredSignature)},
			func() (io.Reader, error) { return bytes.NewReader(signedData), nil },
			[]string{string(publicKey)},
			1,
			nil,
		)
		require.NoError(t, err)
		require.Zero(t, remainingRequired)
		require.Len(t, remainingKeys, 1)
	})

	t.Run("binary signature produced by a previous release still verifies", func(t *testing.T) {
		keyring, err := openpgp.ReadArmoredKeyRing(strings.NewReader(string(publicKey)))
		require.NoError(t, err)

		_, err = openpgp.CheckDetachedSignature(keyring, bytes.NewReader(signedData), bytes.NewReader(binarySignature), nil)
		require.NoError(t, err)
	})
}
