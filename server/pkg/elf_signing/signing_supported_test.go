package elf_signing

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/deckhouse/delivery-kit-sdk/pkg/signature/elf/inhouse"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
)

func TestTrySignELFEmbedsVerifiableSignature(t *testing.T) {
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
	signedData, err := io.ReadAll(signed)
	require.NoError(t, err)
	require.NoError(t, inhouse.VerifyBytes(context.Background(), []string{certs.RootRef}, signedData))
}
