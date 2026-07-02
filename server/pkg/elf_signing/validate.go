package elf_signing

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	"github.com/deckhouse/delivery-kit-sdk/pkg/signver"
	"github.com/deckhouse/delivery-kit-sdk/pkg/signver/hashivault"
	"github.com/secure-systems-lab/go-securesystemslib/encrypted"
)

func validateSettings(settings SignerSettings) error {
	if strings.Contains(settings.KeyRef, "://") {
		if !strings.HasPrefix(settings.KeyRef, hashivault.ReferenceScheme) {
			return fmt.Errorf("invalid key reference: %q", settings.KeyRef)
		}

		vaultFields := []struct{ name, val string }{
			{fieldNameELFSigningVaultAddr, settings.VaultOpts.Address},
			{fieldNameELFSigningVaultTransitPath, settings.VaultOpts.TransitPath},
			{fieldNameELFSigningVaultAuthRoleID, settings.VaultOpts.AuthRoleID},
			{fieldNameELFSigningVaultAuthSecretID, settings.VaultOpts.AuthSecretID},
			{fieldNameELFSigningVaultAuthPath, settings.VaultOpts.AuthPath},
		}

		for _, field := range vaultFields {
			if field.val == "" {
				return fmt.Errorf("%q is required for vault key reference", field.name)
			}
		}
	} else {
		decoded, err := base64.StdEncoding.DecodeString(settings.KeyRef)
		if err != nil {
			return fmt.Errorf("decode base64 key: %w", err)
		}

		p, _ := pem.Decode(decoded)
		if p == nil {
			return errors.New("invalid pem block")
		}

		if p.Type != signver.PrivateKeyPemType && p.Type != signver.DeliveryKitPrivateKeyPemType {
			return fmt.Errorf("unsupported pem type: %s", p.Type)
		}

		var pass []byte
		if settings.KeyPassword != "" {
			pass = []byte(settings.KeyPassword)
		}

		derBytes, err := encrypted.Decrypt(p.Bytes, pass)
		if err != nil {
			return fmt.Errorf("decrypt private key (check password): %w", err)
		}

		if _, err := x509.ParsePKCS8PrivateKey(derBytes); err != nil {
			return fmt.Errorf("parse private key: %w", err)
		}
	}

	if _, err := base64.StdEncoding.DecodeString(settings.CertRef); err != nil {
		return fmt.Errorf("decode base64 certificate: %w", err)
	}

	if settings.IntermediatesRef != "" {
		if _, err := base64.StdEncoding.DecodeString(settings.IntermediatesRef); err != nil {
			return fmt.Errorf("decode base64 intermediates certificate: %w", err)
		}
	}

	return nil
}
