package elf_signing

import (
	"bytes"
	"context"
	goelf "debug/elf"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrySignELFUnsupportedMachineReturnsError(t *testing.T) {
	signer := newTestSigner(t)

	original := minimalELF(t, goelf.EM_386)

	rc, err := signer.TrySignELF(context.Background(), "unsupported.elf", bytes.NewReader(original))
	require.Error(t, err)
	require.Nil(t, rc)
	require.Contains(t, err.Error(), "unsupported ELF machine")
}
