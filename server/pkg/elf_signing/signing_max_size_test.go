package elf_signing

import (
	"bytes"
	"context"
	"os"
	"strconv"
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

func TestTrySignELFAcceptsArtifactAtConfiguredLimit(t *testing.T) {
	artifact, err := os.ReadFile("testdata/hello.elf")
	require.NoError(t, err)
	signer := newTestSigner(t)
	signer.settings.MaxArtifactSize = strconv.Itoa(len(artifact)) + "B"

	signed, err := signer.TrySignELF(context.Background(), "tool", bytes.NewReader(artifact))
	require.NoError(t, err)
	require.NotNil(t, signed)
}
