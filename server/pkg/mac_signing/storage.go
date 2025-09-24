package mac_signing

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/vault/sdk/logical"
)

const storageKeyPrefix = "mac_signing/"

func storageKey(name string) string {
	return storageKeyPrefix + name
}

func PutCredentials(ctx context.Context, req *logical.Request, creds Credentials) error {
	if _, err := base64.StdEncoding.DecodeString(creds.Certificate); err != nil {
		return fmt.Errorf("invalid base64 certificate: %w", err)
	}
	if _, err := base64.StdEncoding.DecodeString(creds.NotaryKey); err != nil {
		return fmt.Errorf("invalid base64 notary key: %w", err)
	}
	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("unable to marshal credentials: %w", err)
	}

	return req.Storage.Put(ctx, &logical.StorageEntry{
		Key:   storageKey(creds.Name),
		Value: data,
	})
}

func GetCredentials(ctx context.Context, storage logical.Storage, name string) (*Credentials, error) {
	entry, err := storage.Get(ctx, storageKey(name))
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}

	var creds Credentials
	if err := json.Unmarshal(entry.Value, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

func DeleteCredentials(ctx context.Context, req *logical.Request, name string) error {
	return req.Storage.Delete(ctx, storageKey(name))
}

func GetDefaultCredentials(ctx context.Context, storage logical.Storage) (*Credentials, error) {
	list, err := storage.List(ctx, storageKeyPrefix)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	}

	return GetCredentials(ctx, storage, list[0])
}
