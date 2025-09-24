package mac_signing

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/vault/sdk/logical"
)

const storageKeyPrefix = "mac_signing_identity/"

func storageKey() string {
	return storageKeyPrefix + macSigningCertificateName
}

func PutCredentials(ctx context.Context, req *logical.Request, creds Credentials) error {
	if _, err := base64.StdEncoding.DecodeString(creds.Certificate); err != nil {
		return fmt.Errorf("invalid base64 certificate: %w", err)
	}
	if _, err := base64.StdEncoding.DecodeString(creds.NotaryKey); err != nil {
		return fmt.Errorf("invalid base64 notary key: %w", err)
	}

	creds.Name = macSigningCertificateName

	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("unable to marshal credentials: %w", err)
	}

	return req.Storage.Put(ctx, &logical.StorageEntry{
		Key:   storageKey(),
		Value: data,
	})
}

func GetCredentials(ctx context.Context, storage logical.Storage) (*Credentials, error) {
	entry, err := storage.Get(ctx, storageKey())
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

func DeleteCredentials(ctx context.Context, req *logical.Request) error {
	return req.Storage.Delete(ctx, storageKey())
}
