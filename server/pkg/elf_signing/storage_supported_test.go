//go:build linux && amd64 && cgo

package elf_signing

import (
	"context"
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
