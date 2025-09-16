package certificates

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/vault/sdk/logical"
)

const (
	storageKeyPrefixCertificate = "build_certificate/"
)

type certificateStorage struct {
	Name     string
	Data     string
	Password string
}

func certificateNameStorageKey(name string) string {
	return storageKeyPrefixCertificate + name
}

func PutCertificate(ctx context.Context, req *logical.Request, s certificateStorage) (*logical.Response, error) {
	certificateNameStorageKey := certificateNameStorageKey(s.Name)
	entry, err := req.Storage.Get(ctx, certificateNameStorageKey)
	if err != nil {
		return nil, fmt.Errorf("can't check if certificate exists: %w", err)
	}
	if entry != nil {
		return nil, fmt.Errorf("certificate with id %s already exists", s.Name)
	}
	if _, err := base64.StdEncoding.DecodeString(s.Data); err != nil {
		return nil, fmt.Errorf("invalid base64 data: %w", err)
	}
	dataBytes, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal certificate: %w", err)
	}

	if err := req.Storage.Put(ctx, &logical.StorageEntry{
		Key:   certificateNameStorageKey,
		Value: dataBytes,
	}); err != nil {
		return nil, fmt.Errorf("unable to put certificate: %w", err)
	}

	return nil, nil
}

func GetCertificates(ctx context.Context, storage logical.Storage) ([]Certificate, error) {
	list, err := storage.List(ctx, storageKeyPrefixCertificate)
	if err != nil {
		return nil, err
	}

	var certificates []Certificate
	for _, name := range list {
		key := certificateNameStorageKey(name)
		e, err := storage.Get(ctx, key)
		if err != nil {
			return nil, err
		}
		if e == nil {
			continue
		}

		var cert Certificate
		if err := json.Unmarshal(e.Value, &cert); err != nil {
			return nil, fmt.Errorf("unable to unmarshal certificate %s: %w", name, err)
		}

		certificates = append(certificates, cert)
	}

	return certificates, nil
}

func DeleteCertificate(ctx context.Context, req *logical.Request, s certificateStorage) error {
	return req.Storage.Delete(ctx, certificateNameStorageKey(s.Name))
}
