package elf_signing

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/vault/sdk/logical"
)

const storageKeyPrefix = "elf_signing_identity/"

func storageKey() string {
	return storageKeyPrefix + "credentials"
}

func PutSettings(ctx context.Context, req *logical.Request, settings SignerSettings) error {
	if err := validateSettings(settings); err != nil {
		return fmt.Errorf("validate elf signing settings: %w", err)
	}

	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("marshal ELF settings: %w", err)
	}

	return req.Storage.Put(ctx, &logical.StorageEntry{
		Key:   storageKey(),
		Value: data,
	})
}

func GetSettings(ctx context.Context, storage logical.Storage) (*SignerSettings, error) {
	entry, err := storage.Get(ctx, storageKey())
	if err != nil {
		return nil, fmt.Errorf("get ELF settings: %w", err)
	}

	if entry == nil {
		return nil, nil
	}

	var settings SignerSettings
	if err := json.Unmarshal(entry.Value, &settings); err != nil {
		return nil, fmt.Errorf("unmarshal ELF settings: %w", err)
	}

	return &settings, nil
}

func DeleteSettings(ctx context.Context, req *logical.Request) error {
	return req.Storage.Delete(ctx, storageKey())
}
