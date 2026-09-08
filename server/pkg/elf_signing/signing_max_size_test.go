package elf_signing

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrySignELFRejectsArtifactLargerThanConfiguredLimit(t *testing.T) {
	signer := newTestSigner(t)
	signer.settings.MaxArtifactSize = "4B"

	signed, err := signer.TrySignELF(context.Background(), "tool", bytes.NewReader([]byte{'\x7f', 'E', 'L', 'F', 0}))
	require.Nil(t, signed)
	require.ErrorContains(t, err, "exceeds maximum size 4B")
}
