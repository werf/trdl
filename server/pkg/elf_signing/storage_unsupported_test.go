//go:build !linux || !amd64 || !cgo

package elf_signing

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"
)

func seedSettings(t *testing.T, storage logical.Storage) {
	t.Helper()

	certs := generateCerts(t, "")
	data, err := json.Marshal(SignerSettings{KeyRef: certs.PrivRef, CertRef: certs.LeafRef})
	require.NoError(t, err)
	require.NoError(t, storage.Put(context.Background(), &logical.StorageEntry{Key: storageKey(), Value: data}))
}

func TestPutSettingsRejectedOnUnsupportedPlatform(t *testing.T) {
	certs := generateCerts(t, "")

	settings := SignerSettings{
		KeyRef:  certs.PrivRef,
		CertRef: certs.LeafRef,
	}

	req := &logical.Request{Storage: &logical.InmemStorage{}}

	err := PutSettings(context.Background(), req, settings)
	require.ErrorContains(t, err, "ELF signing requires a linux/amd64 build with CGO enabled")
}

func TestGetSettingsReturnsNilOnUnsupportedPlatform(t *testing.T) {
	req := &logical.Request{Storage: &logical.InmemStorage{}}
	seedSettings(t, req.Storage)

	got, err := GetSettings(context.Background(), req.Storage)
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestDeleteSettingsRemovesEntryOnUnsupportedPlatform(t *testing.T) {
	req := &logical.Request{Storage: &logical.InmemStorage{}}
	seedSettings(t, req.Storage)

	require.NoError(t, DeleteSettings(context.Background(), req))

	entry, err := req.Storage.Get(context.Background(), storageKey())
	require.NoError(t, err)
	require.Nil(t, entry)
}
