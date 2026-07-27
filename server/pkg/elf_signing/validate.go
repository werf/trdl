package elf_signing

import (
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	"github.com/deckhouse/delivery-kit-sdk/pkg/signver"
	"github.com/deckhouse/delivery-kit-sdk/pkg/signver/hashivault"
	"github.com/secure-systems-lab/go-securesystemslib/encrypted"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
)

func validateSettings(settings SignerSettings) error {
	if settings.KeyRef == "" {
		return fmt.Errorf("%q is required", fieldNameELFSigningKey)
	}

	var localKey crypto.PrivateKey

	if strings.HasPrefix(settings.KeyRef, hashivault.ReferenceScheme) {
		if err := hashivault.ValidReference(settings.KeyRef); err != nil {
			return fmt.Errorf("invalid key reference %q: %w", settings.KeyRef, err)
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

		if settings.KeyPassword != "" {
			return fmt.Errorf("%q must not be set for vault key reference", fieldNameELFSigningKeyPass)
		}
	} else if strings.Contains(settings.KeyRef, "://") {
		return fmt.Errorf("invalid key reference: %q", settings.KeyRef)
	} else {
		decoded, err := base64.StdEncoding.DecodeString(settings.KeyRef)
		if err != nil {
			return fmt.Errorf("decode base64 key: %w", err)
		}

		p, _ := pem.Decode(decoded)
		if p == nil {
			return errors.New("invalid key pem block")
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

		localKey, err = x509.ParsePKCS8PrivateKey(derBytes)
		if err != nil {
			return fmt.Errorf("parse private key: %w", err)
		}
	}

	if settings.CertRef == "" {
		return fmt.Errorf("%q is required", fieldNameELFSigningCertificate)
	}

	certBytes, err := base64.StdEncoding.DecodeString(settings.CertRef)
	if err != nil {
		return fmt.Errorf("decode base64 certificate: %w", err)
	}

	certBlock, _ := pem.Decode(certBytes)
	if certBlock == nil {
		return errors.New("invalid certificate pem block")
	}

	if certBlock.Type != "CERTIFICATE" {
		return fmt.Errorf("unsupported certificate pem type: %s", certBlock.Type)
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return fmt.Errorf("parse certificate: %w", err)
	}

	if localKey != nil {
		signer, ok := localKey.(crypto.Signer)
		if !ok {
			return errors.New("private key does not implement crypto.Signer")
		}

		if err := cryptoutils.EqualKeys(signer.Public(), cert.PublicKey); err != nil {
			return fmt.Errorf("certificate does not match private key: %w", err)
		}
	}

	if settings.IntermediatesRef != "" {
		intBytes, err := base64.StdEncoding.DecodeString(settings.IntermediatesRef)
		if err != nil {
			return fmt.Errorf("decode base64 intermediates certificate: %w", err)
		}

		certs, err := cryptoutils.UnmarshalCertificatesFromPEM(intBytes)
		if err != nil {
			return fmt.Errorf("parse intermediates certificate: %w", err)
		}

		if len(certs) == 0 {
			return errors.New("invalid intermediates pem block")
		}
	}

	return nil
}
