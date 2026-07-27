//go:build !linux || !amd64 || !cgo

package elf_signing

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrySignELFOnUnsupportedPlatformReturnsError(t *testing.T) {
	signer := newTestSigner(t)

	elf, err := os.ReadFile("testdata/hello.elf")
	require.NoError(t, err)

	rc, err := signer.TrySignELF(context.Background(), "hello.elf", bytes.NewReader(elf))
	if rc != nil {
		defer func() { _ = rc.Close() }()
	}
	require.ErrorContains(t, err, "ELF signing requires a linux/amd64 build with CGO enabled")
}

func TestTrySignELFPassesThroughNonELF(t *testing.T) {
	signer := newTestSigner(t)

	payload := []byte("this is not an ELF binary")

	rc, err := signer.TrySignELF(context.Background(), "script.sh", bytes.NewReader(payload))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, rc.Close())
	}()

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, payload, got)
}
