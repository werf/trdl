//go:build !linux || !amd64 || !cgo

package elf_signing

import (
	"context"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"
)

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
