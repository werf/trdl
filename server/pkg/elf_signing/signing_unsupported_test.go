package elf_signing

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

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
