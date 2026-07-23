//go:build linux && amd64 && cgo

package elf_signing

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/deckhouse/delivery-kit-sdk/pkg/signature/elf/inhouse"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
)

func TestTrySignELFEmbedsVerifiableSignature(t *testing.T) {
	certs := generateCerts(t, "")

	signer, err := NewELFSigner(context.Background(), hclog.NewNullLogger(), &SignerSettings{
		KeyRef:  certs.PrivRef,
		CertRef: certs.LeafRef,
	})
	require.NoError(t, err)

	original, err := os.ReadFile("testdata/hello.elf")
	require.NoError(t, err)

	signed, err := signer.TrySignELF(context.Background(), "hello.elf", bytes.NewReader(original))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, signed.Close())
	}()

	signedFile, ok := signed.(*tempFileCloser)
	require.True(t, ok)

	require.NoError(t, inhouse.Verify(context.Background(), []string{certs.RootRef}, signedFile.Name()))

	_, err = signed.Read(make([]byte, 1))
	require.True(t, err == nil || err == io.EOF)
}
