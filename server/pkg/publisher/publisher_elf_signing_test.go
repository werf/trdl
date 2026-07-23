//go:build !linux || !amd64 || !cgo

package publisher

import (
	"bytes"
	"context"
	goelf "debug/elf"
	"encoding/binary"
	"io"
	"testing"

	"github.com/deckhouse/delivery-kit-sdk/test/pkg/cert_utils"
	"github.com/hashicorp/go-hclog"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/require"

	"github.com/werf/trdl/server/pkg/elf_signing"
	"github.com/werf/trdl/server/pkg/util"
)

type stageTargetFailRepository struct {
	t *testing.T
}

func (r *stageTargetFailRepository) Init() error                       { return nil }
func (r *stageTargetFailRepository) SetPrivKeys(TufRepoPrivKeys) error { return nil }
func (r *stageTargetFailRepository) GetPrivKeys() TufRepoPrivKeys      { return TufRepoPrivKeys{} }
func (r *stageTargetFailRepository) GenPrivKeys() error                { return nil }
func (r *stageTargetFailRepository) UpdateTimestamps(context.Context, util.Clock) error {
	return nil
}
func (r *stageTargetFailRepository) CommitStaged(context.Context) error { return nil }
func (r *stageTargetFailRepository) GetTargets(context.Context) ([]string, error) {
	return nil, nil
}

func (r *stageTargetFailRepository) RotatePrivKeys(context.Context) (bool, TufRepoPrivKeys, error) {
	return false, TufRepoPrivKeys{}, nil
}

func (r *stageTargetFailRepository) StageTarget(context.Context, string, io.Reader) error {
	r.t.Fatal("StageTarget must not be called when ELF signing fails")
	return nil
}

func TestStageReleaseTargetPropagatesELFSigningErrorWithoutPanic(t *testing.T) {
	gomega.RegisterTestingT(t)

	certs := cert_utils.GenerateCertificatesWithOptions(cert_utils.GenerateCertificatesOptions{
		KeyType:           cert_utils.KeyType_ECDSA_P256,
		UseBase64Encoding: true,
	})

	elfSigner, err := elf_signing.NewELFSigner(context.Background(), hclog.NewNullLogger(), &elf_signing.SignerSettings{
		KeyRef:           certs.PrivRef,
		CertRef:          certs.LeafRef,
		IntermediatesRef: certs.IntermediatesRef,
	})
	require.NoError(t, err)

	publisher := &Publisher{}
	repository := &stageTargetFailRepository{t: t}

	elfBinary := minimalELFHeader(t, goelf.EM_X86_64)

	err = publisher.StageReleaseTarget(
		context.Background(),
		repository,
		"v1.0.0",
		"linux-amd64/hello",
		bytes.NewReader(elfBinary),
		elfSigner,
	)

	require.Error(t, err)
	require.ErrorContains(t, err, "try signing artifact")
	require.ErrorContains(t, err, "ELF signing requires a linux/amd64 build with CGO enabled")
}

func minimalELFHeader(t *testing.T, machine goelf.Machine) []byte {
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
