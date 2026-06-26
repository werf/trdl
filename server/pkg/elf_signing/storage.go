package elf_signing

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	"github.com/deckhouse/delivery-kit-sdk/pkg/signver"
	"github.com/deckhouse/delivery-kit-sdk/pkg/signver/hashivault"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/secure-systems-lab/go-securesystemslib/encrypted"
)

const storageKeyPrefix = "elf_signing_identity/"

func storageKey() string {
	return storageKeyPrefix + "credentials"
}

func PutSettings(ctx context.Context, req *logical.Request, settings SignerSettings) error {
	if strings.Contains(settings.KeyRef, "://") {
		if !strings.HasPrefix(settings.KeyRef, hashivault.ReferenceScheme) {
			return fmt.Errorf("invalid key reference: %s", settings.KeyRef)
		}
	} else {
		decoded, err := base64.StdEncoding.DecodeString(settings.KeyRef)
		if err != nil {
			return fmt.Errorf("invalid base64 key: %w", err)
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
		return fmt.Errorf("invalid base64 certificate: %w", err)
	}

	if settings.IntermediatesRef != "" {
		if _, err := base64.StdEncoding.DecodeString(settings.IntermediatesRef); err != nil {
			return fmt.Errorf("invalid base64 intermediates certificate: %w", err)
		}
	}

	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("unable to marshal ELF settings: %w", err)
	}

	return req.Storage.Put(ctx, &logical.StorageEntry{
		Key:   storageKey(),
		Value: data,
	})
}

func GetSettings(ctx context.Context, storage logical.Storage) (*SignerSettings, error) {
	entry, err := storage.Get(ctx, storageKey())
	if err != nil {
		return nil, err
	}

	if entry == nil {
		return nil, nil
	}

	var settings SignerSettings
	if err := json.Unmarshal(entry.Value, &settings); err != nil {
		return nil, fmt.Errorf("unable to unmarshal ELF settings: %w", err)
	}

	return &settings, nil
}

func DeleteSettings(ctx context.Context, req *logical.Request) error {
	return req.Storage.Delete(ctx, storageKey())
}
