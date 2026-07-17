//go:build ai_tests

package elf_signing

import (
	"testing"

	"github.com/deckhouse/delivery-kit-sdk/pkg/signver/hashivault"
	"github.com/stretchr/testify/require"
)

func TestAI_ValidateELFSettingsRejectsPasswordWithVaultKeyReference(t *testing.T) {
	certs := generateCerts(t, "")
	settings := SignerSettings{
		KeyRef:      hashivault.ReferenceScheme + "my-key",
		KeyPassword: "s3cret",
		CertRef:     certs.LeafRef,
		VaultOpts:   validVaultOpts(),
	}
	require.ErrorContains(t, validateSettings(settings), `"password" must not be set for vault key reference`)
}
