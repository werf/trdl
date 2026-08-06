package elf_signing

import (
	"bytes"
	"context"
	goelf "debug/elf"
	"io"
	"os"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"

	"github.com/werf/trdl/server/pkg/elf_signing/inhouse"
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

	require.NoError(t, inhouse.Verify(context.Background(), []string{certs.RootRef}, got))
}
