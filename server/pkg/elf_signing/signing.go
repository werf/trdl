package elf_signing

import (
	"bufio"
	"bytes"
	"context"
	goelf "debug/elf"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/deckhouse/delivery-kit-sdk/pkg/signature/elf/inhouse"
	"github.com/deckhouse/delivery-kit-sdk/pkg/signver"
	"github.com/deckhouse/delivery-kit-sdk/pkg/signver/hashivault"
	"github.com/hashicorp/go-hclog"
	"github.com/sigstore/sigstore/pkg/cryptoutils"

	"github.com/werf/logboek"
)

type ELFSigner struct {
	settings *SignerSettings

	logger hclog.Logger
	sv     *signver.SignerVerifier
	svOnce sync.Once
	svErr  error
}

func NewELFSigner(logger hclog.Logger, opts *SignerSettings) *ELFSigner {
	if opts.MaxArtifactSize == "" {
		settings := *opts
		settings.MaxArtifactSize = defaultMaxArtifactSize
		opts = &settings
	}
	return &ELFSigner{logger: logger, settings: opts}
}

func (s *ELFSigner) getSignerVerifier(ctx context.Context) (*signver.SignerVerifier, error) {
	s.svOnce.Do(func() {
		s.sv, s.svErr = buildSignerVerifier(ctx, s.settings)
	})

	return s.sv, s.svErr
}

func (s *ELFSigner) TrySignELF(ctx context.Context, releaseFilePath string, data io.Reader) (io.ReadCloser, error) {
	br := bufio.NewReader(data)
	magic, err := br.Peek(4)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("peek header of %q: %w", releaseFilePath, err)
	}
	if !bytes.HasPrefix(magic, []byte(goelf.ELFMAG)) {
		logboek.Context(ctx).Default().LogF("Skipping ELF sign for %q: not an ELF file\n", releaseFilePath)
		return io.NopCloser(br), nil
	}

	maxArtifactSize, err := parseArtifactSize(s.settings.MaxArtifactSize)
	if err != nil {
		return nil, fmt.Errorf("parse maximum artifact size: %w", err)
	}
	artifact, err := io.ReadAll(io.LimitReader(br, maxArtifactSize))
	if err != nil {
		return nil, fmt.Errorf("buffer artifact %q: %w", releaseFilePath, err)
	}
	if int64(len(artifact)) == maxArtifactSize {
		if _, err := br.Peek(1); err == nil {
			return nil, fmt.Errorf("ELF artifact %q exceeds maximum size %s", releaseFilePath, s.settings.MaxArtifactSize)
		} else if !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("check artifact size %q: %w", releaseFilePath, err)
		}
	}

	machine, err := readELFMachine(bytes.NewReader(artifact))
	if err != nil {
		return nil, fmt.Errorf("read ELF header of %q: %w", releaseFilePath, err)
	}

	if machine != goelf.EM_X86_64 && machine != goelf.EM_AARCH64 {
		return nil, fmt.Errorf("unsupported ELF machine %v for %q", machine, releaseFilePath)
	}

	sv, err := s.getSignerVerifier(ctx)
	if err != nil {
		return nil, fmt.Errorf("get signer verifier: %w", err)
	}

	signed, err := inhouse.SignBytes(ctx, sv, artifact)
	if err != nil {
		return nil, fmt.Errorf("sign %q: %w", releaseFilePath, err)
	}

	logboek.Context(ctx).Default().LogF("Embedded ELF signature into %q\n", releaseFilePath)

	return io.NopCloser(bytes.NewReader(signed)), nil
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
