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
		MaxArtifactSize:  defaultMaxArtifactSize,
	})

	original, err := os.ReadFile("testdata/hello.elf")
	require.NoError(t, err)

	signed, err := signer.TrySignELF(context.Background(), "hello.elf", bytes.NewReader(original))
	require.NoError(t, err)
	got, err := io.ReadAll(signed)
	require.NoError(t, err)
	require.NotEmpty(t, got)

	_, err = goelf.NewFile(bytes.NewReader(got))
	require.NoError(t, err)

	verifyPath := filepath.Join(t.TempDir(), "signed.elf")
	require.NoError(t, os.WriteFile(verifyPath, got, 0o644))

	require.NoError(t, inhouse.Verify(context.Background(), []string{certs.RootRef}, verifyPath))
}

func TestTrySignELFResignsAndRejectsInvalidSignature(t *testing.T) {
	certs := generateCerts(t, "")
	signer := NewELFSigner(hclog.NewNullLogger(), &SignerSettings{
		KeyRef:           certs.PrivRef,
		CertRef:          certs.LeafRef,
		IntermediatesRef: certs.IntermediatesRef,
		MaxArtifactSize:  defaultMaxArtifactSize,
	})
	original, err := os.ReadFile("testdata/hello.elf")
	require.NoError(t, err)

	first, err := signer.TrySignELF(context.Background(), "hello.elf", bytes.NewReader(original))
	require.NoError(t, err)
	firstBytes, err := io.ReadAll(first)
	require.NoError(t, err)

	second, err := signer.TrySignELF(context.Background(), "hello.elf", bytes.NewReader(firstBytes))
	require.NoError(t, err)
	secondBytes, err := io.ReadAll(second)
	require.NoError(t, err)

	verifyPath := filepath.Join(t.TempDir(), "resigned.elf")
	require.NoError(t, os.WriteFile(verifyPath, secondBytes, 0o644))
	require.NoError(t, inhouse.Verify(context.Background(), []string{certs.RootRef}, verifyPath))

	elf, err := goelf.NewFile(bytes.NewReader(secondBytes))
	require.NoError(t, err)
	section := elf.Section(".text")
	require.NotNil(t, section)
	secondBytes[section.Offset] ^= 1
	require.NoError(t, os.WriteFile(verifyPath, secondBytes, 0o644))
	require.ErrorContains(t, inhouse.Verify(context.Background(), []string{certs.RootRef}, verifyPath), "signature verification")
}

func TestTrySignELFRejectsCorruptedELF(t *testing.T) {
	signer := newTestSigner(t)
	corrupted, err := os.ReadFile("testdata/hello.elf")
	require.NoError(t, err)
	corrupted[4] = 0

	signed, err := signer.TrySignELF(context.Background(), "hello.elf", bytes.NewReader(corrupted))
	// AI_COMMENT: this bare require.Error passes for the wrong reason — signing currently fails on the empty MaxArtifactSize before the corrupted ELF header is ever parsed, so nothing about corrupted-ELF handling is proven. Assert the specific error, e.g. require.ErrorContains(t, err, "read ELF header").
	require.Error(t, err)
	require.Nil(t, signed)
}
