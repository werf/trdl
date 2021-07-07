package tasks_manager

import (
	"context"

	"github.com/hashicorp/vault/sdk/logical"
)

func getConfiguration(ctx context.Context, storage logical.Storage) (*configuration, error) {
	raw, err := storage.Get(ctx, storageKeyConfiguration)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}

	config := new(configuration)
	if err := raw.DecodeJSON(config); err != nil {
		return nil, err
	}

	return config, nil
}

func putConfiguration(ctx context.Context, storage logical.Storage, raw map[string]interface{}) error {
	entry, err := logical.StorageEntryJSON(storageKeyConfiguration, raw)
	if err != nil {
		return err
	}

	if err := storage.Put(ctx, entry); err != nil {
		return err
	}

	return err
}
