package elf_signing

import (
	"bytes"
	"context"
	goelf "debug/elf"
	"encoding/binary"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
)

func newTestSigner(t *testing.T) *ELFSigner {
	t.Helper()

	certs := generateCerts(t, "")

	signer, err := NewELFSigner(context.Background(), hclog.NewNullLogger(), &SignerSettings{
		KeyRef:  certs.PrivRef,
		CertRef: certs.LeafRef,
	})
	require.NoError(t, err)

	return signer
}

func minimalELF(t *testing.T, machine goelf.Machine) []byte {
	t.Helper()

	var buf bytes.Buffer

	buf.Write([]byte(goelf.ELFMAG))
	buf.WriteByte(byte(goelf.ELFCLASS64))
	buf.WriteByte(byte(goelf.ELFDATA2LSB))
	buf.WriteByte(byte(goelf.EV_CURRENT))
	buf.WriteByte(byte(goelf.ELFOSABI_NONE))
	buf.Write(make([]byte, 8))

	for _, v := range []uint16{uint16(goelf.ET_EXEC), uint16(machine)} {
		require.NoError(t, binary.Write(&buf, binary.LittleEndian, v))
	}
	require.NoError(t, binary.Write(&buf, binary.LittleEndian, uint32(goelf.EV_CURRENT)))
	for range 3 {
		require.NoError(t, binary.Write(&buf, binary.LittleEndian, uint64(0)))
	}
	require.NoError(t, binary.Write(&buf, binary.LittleEndian, uint32(0)))
	require.NoError(t, binary.Write(&buf, binary.LittleEndian, uint16(64)))
	for range 5 {
		require.NoError(t, binary.Write(&buf, binary.LittleEndian, uint16(0)))
	}

	return buf.Bytes()
}
