//go:build linux && amd64 && cgo

package elf_signing

import (
	"bytes"
	"context"
	goelf "debug/elf"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/deckhouse/delivery-kit-sdk/pkg/signature/elf/inhouse"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
)

func TestTrySignELFReturnedReaderVerifies(t *testing.T) {
	certs := generateCerts(t, "")

	signer := NewELFSigner(hclog.NewNullLogger(), &SignerSettings{
		KeyRef:           certs.PrivRef,
		CertRef:          certs.LeafRef,
		IntermediatesRef: certs.IntermediatesRef,
	})

	original, err := os.ReadFile("testdata/hello.elf")
	require.NoError(t, err)

	signed, err := signer.TrySignELF(context.Background(), "hello.elf", bytes.NewReader(original))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, signed.Close())
	}()

	got, err := io.ReadAll(signed)
	require.NoError(t, err)
	require.NotEmpty(t, got)

	_, err = goelf.NewFile(bytes.NewReader(got))
	require.NoError(t, err)

	verifyPath := filepath.Join(t.TempDir(), "signed.elf")
	require.NoError(t, os.WriteFile(verifyPath, got, 0o644))

	require.NoError(t, inhouse.Verify(context.Background(), []string{certs.RootRef}, verifyPath))
}
