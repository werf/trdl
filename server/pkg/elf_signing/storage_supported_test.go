package elf_signing

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"
)

func TestPutSettingsStoresValidSettings(t *testing.T) {
	certs := generateCerts(t, "")

	settings := SignerSettings{
		KeyRef:  certs.PrivRef,
		CertRef: certs.LeafRef,
	}

	req := &logical.Request{Storage: &logical.InmemStorage{}}

	require.NoError(t, PutSettings(context.Background(), req, settings))

	got, err := GetSettings(context.Background(), req.Storage)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, settings.KeyRef, got.KeyRef)
	require.Equal(t, settings.CertRef, got.CertRef)
}

func TestGetSettingsDefaultsMaxArtifactSizeForLegacyConfiguration(t *testing.T) {
	certs := generateCerts(t, "")
	storage := &logical.InmemStorage{}
	legacy := SignerSettings{KeyRef: certs.PrivRef, CertRef: certs.LeafRef}
	data, err := json.Marshal(legacy)
	require.NoError(t, err)
	require.NoError(t, storage.Put(context.Background(), &logical.StorageEntry{Key: storageKey(), Value: data}))

	settings, err := GetSettings(context.Background(), storage)
	require.NoError(t, err)
	require.Equal(t, defaultMaxArtifactSize, settings.MaxArtifactSize)
}

func TestPutGetDeleteSettingsRoundTrip(t *testing.T) {
	certs := generateCerts(t, "")

	settings := SignerSettings{
		KeyRef:  certs.PrivRef,
		CertRef: certs.LeafRef,
	}

	req := &logical.Request{Storage: &logical.InmemStorage{}}

	require.NoError(t, PutSettings(context.Background(), req, settings))

	got, err := GetSettings(context.Background(), req.Storage)
	require.NoError(t, err)
	require.NotNil(t, got)

	require.NoError(t, DeleteSettings(context.Background(), req))

	got, err = GetSettings(context.Background(), req.Storage)
	require.NoError(t, err)
	require.Nil(t, got)
}
