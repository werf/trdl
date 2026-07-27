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
	"sync"

	"github.com/deckhouse/delivery-kit-sdk/pkg/signver"
	"github.com/deckhouse/delivery-kit-sdk/pkg/signver/hashivault"
	"github.com/hashicorp/go-hclog"
	"github.com/sigstore/sigstore/pkg/cryptoutils"

	"github.com/werf/logboek"
)

type tempFileCloser struct {
	*os.File
	cleanup   func() error
	closeOnce sync.Once
	closeErr  error
}

func (t *tempFileCloser) Close() error {
	t.closeOnce.Do(func() {
		t.closeErr = t.cleanup()
	})
	return t.closeErr
}

type ELFSigner struct {
	settings *SignerSettings

	logger hclog.Logger
	sv     *signver.SignerVerifier
	svOnce sync.Once
	svErr  error
}

func NewELFSigner(logger hclog.Logger, opts *SignerSettings) *ELFSigner {
	return &ELFSigner{logger: logger, settings: opts}
}

func (s *ELFSigner) getSignerVerifier(ctx context.Context) (*signver.SignerVerifier, error) {
	s.svOnce.Do(func() {
		s.sv, s.svErr = buildSignerVerifier(ctx, s.settings)
	})

	return s.sv, s.svErr
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

	if _, deferErr = tmp.Seek(0, io.SeekStart); deferErr != nil {
		return nil, fmt.Errorf("seek temp file: %w", deferErr)
	}

	if machine != goelf.EM_X86_64 && machine != goelf.EM_AARCH64 {
		deferErr = fmt.Errorf("unsupported ELF machine %v for %q", machine, releaseFilePath)
		return nil, deferErr
	}

	sv, deferErr := s.getSignerVerifier(ctx)
	if deferErr != nil {
		return nil, fmt.Errorf("get signer verifier: %w", deferErr)
	}

	if deferErr = signELF(ctx, sv, tmp.Name()); deferErr != nil {
		return nil, fmt.Errorf("sign %q: %w", releaseFilePath, deferErr)
	}

	logboek.Context(ctx).Default().LogF("Embedded ELF signature into %q\n", releaseFilePath)

	// signELF rewrites the file via an external process; reopen by path so the
	// returned reader is not tied to how the SDK finalizes the on-disk write.
	signed, deferErr := os.Open(tmp.Name())
	if deferErr != nil {
		return nil, fmt.Errorf("reopen signed file %q: %w", releaseFilePath, deferErr)
	}
	_ = tmp.Close()

	cleanup = func() error {
		_ = signed.Close()
		return os.Remove(signed.Name())
	}

	return &tempFileCloser{File: signed, cleanup: cleanup}, nil
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

func buildSignerVerifier(ctx context.Context, settings *SignerSettings) (*signver.SignerVerifier, error) {
	passFunc := cryptoutils.SkipPassword
	if settings.KeyPassword != "" {
		passFunc = cryptoutils.StaticPasswordFunc([]byte(settings.KeyPassword))
	}

	var signerOpts signver.SignerVerifierOpts
	if strings.HasPrefix(settings.KeyRef, hashivault.ReferenceScheme) {
		signerOpts = signver.SignerVerifierOpts{
			VaultOpts: hashivault.VaultOpts{
				Address:                 settings.VaultOpts.Address,
				TransitSecretEnginePath: settings.VaultOpts.TransitPath,
				Auth: &hashivault.VaultAuth{
					AppRole: &hashivault.AppRoleAuth{
						RoleID:   settings.VaultOpts.AuthRoleID,
						SecretID: settings.VaultOpts.AuthSecretID,
						Path:     settings.VaultOpts.AuthPath,
					},
				},
			},
		}
	}

	sv, err := signver.NewSignerVerifier(ctx, settings.CertRef, settings.IntermediatesRef, signver.KeyOpts{
		KeyRef:             settings.KeyRef,
		PassFunc:           passFunc,
		SignerVerifierOpts: signerOpts,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create signer verifier: %w", err)
	}

	return sv, nil
}
