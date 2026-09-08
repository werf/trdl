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

func TestTrySignELFResignsAndRejectsInvalidSignature(t *testing.T) {
	certs := generateCerts(t, "")
	signer := NewELFSigner(hclog.NewNullLogger(), &SignerSettings{
		KeyRef:           certs.PrivRef,
		CertRef:          certs.LeafRef,
		IntermediatesRef: certs.IntermediatesRef,
	})
	original, err := os.ReadFile("testdata/hello.elf")
	require.NoError(t, err)

	first, err := signer.TrySignELF(context.Background(), "hello.elf", bytes.NewReader(original))
	require.NoError(t, err)
	firstBytes, err := io.ReadAll(first)
	require.NoError(t, err)
	require.NoError(t, first.Close())

	second, err := signer.TrySignELF(context.Background(), "hello.elf", bytes.NewReader(firstBytes))
	require.NoError(t, err)
	secondBytes, err := io.ReadAll(second)
	require.NoError(t, err)
	require.NoError(t, second.Close())

	verifyPath := filepath.Join(t.TempDir(), "resigned.elf")
	require.NoError(t, os.WriteFile(verifyPath, secondBytes, 0o644))
	require.NoError(t, inhouse.Verify(context.Background(), []string{certs.RootRef}, verifyPath))

	elf, err := goelf.NewFile(bytes.NewReader(secondBytes))
	require.NoError(t, err)
	section := elf.Section(".note.delivery-kit.signature")
	require.NotNil(t, section)
	noteOffset := int(section.Offset)
	nameSize := elf.ByteOrder.Uint32(secondBytes[noteOffset : noteOffset+4])
	signatureOffset := noteOffset + 12 + int((nameSize+3)&^3)
	secondBytes[signatureOffset] ^= 1
	require.NoError(t, os.WriteFile(verifyPath, secondBytes, 0o644))
	require.Error(t, inhouse.Verify(context.Background(), []string{certs.RootRef}, verifyPath))
}

func TestTrySignELFRejectsCorruptedELF(t *testing.T) {
	signer := newTestSigner(t)
	corrupted, err := os.ReadFile("testdata/hello.elf")
	require.NoError(t, err)
	corrupted[4] = 0

	signed, err := signer.TrySignELF(context.Background(), "hello.elf", bytes.NewReader(corrupted))
	require.Error(t, err)
	require.Nil(t, signed)
}

func TestSignELFPreservesExecutableMode(t *testing.T) {
	certs := generateCerts(t, "")
	signer := NewELFSigner(hclog.NewNullLogger(), &SignerSettings{
		KeyRef:           certs.PrivRef,
		CertRef:          certs.LeafRef,
		IntermediatesRef: certs.IntermediatesRef,
	})
	original, err := os.ReadFile("testdata/hello.elf")
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "hello.elf")
	require.NoError(t, os.WriteFile(path, original, 0o751))
	require.NoError(t, os.Chmod(path, 0o751))

	sv, err := signer.getSignerVerifier(context.Background())
	require.NoError(t, err)
	require.NoError(t, signELF(context.Background(), sv, path))

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o751), info.Mode().Perm())
	require.NoError(t, inhouse.Verify(context.Background(), []string{certs.RootRef}, path))
}
