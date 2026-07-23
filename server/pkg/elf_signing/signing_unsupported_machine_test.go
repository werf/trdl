package elf_signing

import (
	"bytes"
	"context"
	goelf "debug/elf"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrySignELFUnsupportedMachinePassesThroughIntact(t *testing.T) {
	signer := newTestSigner(t)

	original := minimalELF(t, goelf.EM_386)

	rc, err := signer.TrySignELF(context.Background(), "unsupported.elf", bytes.NewReader(original))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, rc.Close())
	}()

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, original, got)
}
