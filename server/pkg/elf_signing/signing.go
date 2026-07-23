package elf_signing

import (
	"bufio"
	"bytes"
	"context"
	goelf "debug/elf"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/deckhouse/delivery-kit-sdk/pkg/signature/elf/inhouse"
	"github.com/deckhouse/delivery-kit-sdk/pkg/signver"
	"github.com/deckhouse/delivery-kit-sdk/pkg/signver/hashivault"
	"github.com/hashicorp/go-hclog"
	"github.com/sigstore/sigstore/pkg/cryptoutils"

	"github.com/werf/logboek"
)

type tempFileCloser struct {
	*os.File
	cleanup func() error
}

func (t *tempFileCloser) Close() error { return t.cleanup() }

type ELFSigner struct {
	SignerSettings
	logger hclog.Logger
	sv     *signver.SignerVerifier
}

func NewELFSigner(ctx context.Context, logger hclog.Logger, opts *SignerSettings) (*ELFSigner, error) {
	passFunc := cryptoutils.SkipPassword
	if opts.KeyPassword != "" {
		passFunc = cryptoutils.StaticPasswordFunc([]byte(opts.KeyPassword))
	}

	var signerOpts signver.SignerVerifierOpts
	if strings.HasPrefix(opts.KeyRef, hashivault.ReferenceScheme) {
		signerOpts = signver.SignerVerifierOpts{
			VaultOpts: hashivault.VaultOpts{
				Address:                 opts.VaultOpts.Address,
				TransitSecretEnginePath: opts.VaultOpts.TransitPath,
				Auth: &hashivault.VaultAuth{
					AppRole: &hashivault.AppRoleAuth{
						RoleID:   opts.VaultOpts.AuthRoleID,
						SecretID: opts.VaultOpts.AuthSecretID,
						Path:     opts.VaultOpts.AuthPath,
					},
				},
			},
		}
	}

	sv, err := signver.NewSignerVerifier(ctx, opts.CertRef, opts.IntermediatesRef, signver.KeyOpts{
		KeyRef:             opts.KeyRef,
		PassFunc:           passFunc,
		SignerVerifierOpts: signerOpts,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create signer verifier: %w", err)
	}

	return &ELFSigner{logger: logger, sv: sv}, nil
}

func (s *ELFSigner) TrySignELF(ctx context.Context, releaseFilePath string, data io.Reader) (io.ReadCloser, error) {
	// Peek first 4 bytes to skip disk buffering for non-ELF artifacts.
	br := bufio.NewReader(data)
	magic, err := br.Peek(4)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("peek header of %q: %w", releaseFilePath, err)
	}
	if !bytes.HasPrefix(magic, []byte(goelf.ELFMAG)) {
		logboek.Context(ctx).Default().LogF("Skipping ELF sign for %q: not an ELF file\n", releaseFilePath)
		return io.NopCloser(br), nil
	}

	tmp, err := os.CreateTemp("", "trdl-release-source-*")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}

	var deferErr error
	cleanup := func() error {
		_ = tmp.Close()
		return os.Remove(tmp.Name())
	}

	defer func() {
		if deferErr != nil {
			_ = cleanup()
		}
	}()

	if _, deferErr = io.Copy(tmp, br); deferErr != nil {
		return nil, fmt.Errorf("buffer artifact %q to disk: %w", releaseFilePath, deferErr)
	}

	if deferErr = tmp.Sync(); deferErr != nil {
		return nil, fmt.Errorf("sync temp file: %w", deferErr)
	}

	s.logger.Debug("Buffered ELF artifact to disk for signing", "path", tmp.Name())

	machine, deferErr := readELFMachine(tmp)
	if deferErr != nil {
		return nil, fmt.Errorf("read ELF header of %q: %w", releaseFilePath, deferErr)
	}

	if machine != goelf.EM_X86_64 && machine != goelf.EM_AARCH64 {
		logboek.Context(ctx).Warn().LogF("Unsupported ELF machine %v for %q\n", machine, releaseFilePath)
		return &tempFileCloser{File: tmp, cleanup: cleanup}, nil
	}

	if deferErr = inhouse.Sign(ctx, s.sv, tmp.Name()); deferErr != nil {
		return nil, fmt.Errorf("sign %q: %w", releaseFilePath, deferErr)
	}

	logboek.Context(ctx).Default().LogF("Embedded ELF signature into %q\n", releaseFilePath)

	if _, deferErr = tmp.Seek(0, io.SeekStart); deferErr != nil {
		return nil, fmt.Errorf("seek temp file: %w", deferErr)
	}

	return &tempFileCloser{File: tmp, cleanup: cleanup}, nil
}

func readELFMachine(r io.ReaderAt) (goelf.Machine, error) {
	f, err := goelf.NewFile(r)
	if err != nil {
		return goelf.EM_NONE, fmt.Errorf("parse ELF: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()
	return f.FileHeader.Machine, nil
}
