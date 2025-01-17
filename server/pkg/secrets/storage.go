package secrets

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/vault/sdk/logical"
)

const (
	storageKeyPrefixSecret = "build_secret/"
)

type secretStorage struct {
	Id   string
	Data string
}

func secretIdStorageKey(name string) string {
	return storageKeyPrefixSecret + name
}

func CreateSecret(ctx context.Context, req *logical.Request, s secretStorage) (*logical.Response, error) {
	secretIdStorageKey := secretIdStorageKey(s.Id)
	entry, err := req.Storage.Get(ctx, secretIdStorageKey)
	if err != nil {
		return nil, fmt.Errorf("can't check if secret exists: %w", err)
	}
	if entry != nil {
		return nil, fmt.Errorf("secret with id %s already exists", s.Id)
	}

	if err := req.Storage.Put(ctx, &logical.StorageEntry{
		Key:   secretIdStorageKey,
		Value: []byte(s.Data),
	}); err != nil {
		return nil, fmt.Errorf("unable to put secret: %w", err)
	}

	return nil, nil
}

func GetSecrets(ctx context.Context, storage logical.Storage) ([]Secret, error) {
	list, err := storage.List(ctx, storageKeyPrefixSecret)
	if err != nil {
		return nil, err
	}

	var secrets []Secret
	for _, name := range list {
		storageEntryKey := secretIdStorageKey(name)
		e, err := storage.Get(ctx, storageEntryKey)
		if err != nil {
			return nil, err
		}
		if e == nil {
			continue
		}

		secrets = append(secrets,
			Secret{
				Id:   strings.TrimPrefix(e.Key, storageKeyPrefixSecret),
				Data: e.Value,
			},
		)
	}

	return secrets, nil
}

func DeleteSecret(ctx context.Context, req *logical.Request, s secretStorage) error {
	return req.Storage.Delete(ctx, secretIdStorageKey(s.Id))
}
