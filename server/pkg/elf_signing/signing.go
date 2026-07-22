package elf_signing

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/deckhouse/delivery-kit-sdk/pkg/signature/elf/inhouse"
	"github.com/deckhouse/delivery-kit-sdk/pkg/signver"
	"github.com/deckhouse/delivery-kit-sdk/pkg/signver/hashivault"
	"github.com/sigstore/sigstore/pkg/cryptoutils"

	"github.com/werf/logboek"
)

// signMu serializes ELF signing across the whole process. The delivery-kit-sdk
// configures Vault transit access only through process-global env vars
// (VAULT_ADDR, role/secret id, etc.), so concurrent signings from different
// plugin mounts would otherwise race and leak settings into each other.
var signMu sync.Mutex

type origEnvVar struct {
	value   string
	present bool
}

func restoreEnvVar(envKey string, orig origEnvVar) error {
	if orig.present {
		if err := os.Setenv(envKey, orig.value); err != nil {
			return fmt.Errorf("failed to restore env var %q: %w", envKey, err)
		}
		return nil
	}
	if err := os.Unsetenv(envKey); err != nil {
		return fmt.Errorf("failed to unset env var %q: %w", envKey, err)
	}
	return nil
}

func setVaultEnvVars(opts VaultSignerOpts) (func() error, error) {
	envs := map[string]string{
		"VAULT_ADDR":                 opts.Address,
		"TRANSIT_SECRET_ENGINE_PATH": opts.TransitPath,
		"WERF_VAULT_AUTH_PATH":       opts.AuthPath,
		"WERF_VAULT_AUTH_ROLE_ID":    opts.AuthRoleID,
		"WERF_VAULT_AUTH_SECRET_ID":  opts.AuthSecretID,
	}

	applied := map[string]origEnvVar{}
	for envKey, val := range envs {
		origVal, present := os.LookupEnv(envKey)
		if err := os.Setenv(envKey, val); err != nil {
			rollback := fmt.Errorf("failed to set env var %q: %w", envKey, err)
			for appliedKey, orig := range applied {
				if rerr := restoreEnvVar(appliedKey, orig); rerr != nil {
					rollback = errors.Join(rollback, rerr)
				}
			}
			return nil, rollback
		}
		applied[envKey] = origEnvVar{value: origVal, present: present}
	}

	return func() error {
		var restoreErr error
		for envKey, orig := range applied {
			if err := restoreEnvVar(envKey, orig); err != nil {
				restoreErr = errors.Join(restoreErr, err)
			}
		}
		return restoreErr
	}, nil
}

func Sign(ctx context.Context, path string, opts SignerSettings) error {
	signMu.Lock()
	defer signMu.Unlock()

	if strings.HasPrefix(opts.KeyRef, hashivault.ReferenceScheme) {
		restoreEnv, err := setVaultEnvVars(opts.VaultOpts)
		if err != nil {
			return fmt.Errorf("set vault env vars: %w", err)
		}

		defer func() {
			if err := restoreEnv(); err != nil {
				logboek.Context(ctx).Warn().LogF("failed to restore vault env vars: %w\n", err)
			}
		}()

		return signWithSignerVerifier(ctx, path, opts)
	}

	return signWithSignerVerifier(ctx, path, opts)
}

func signWithSignerVerifier(ctx context.Context, path string, opts SignerSettings) error {
	passFunc := cryptoutils.SkipPassword
	if opts.KeyPassword != "" {
		passFunc = cryptoutils.StaticPasswordFunc([]byte(opts.KeyPassword))
	}

	sv, err := signver.NewSignerVerifier(ctx, opts.CertRef, opts.IntermediatesRef, signver.KeyOpts{
		KeyRef:   opts.KeyRef,
		PassFunc: passFunc,
	})
	if err != nil {
		return fmt.Errorf("failed to create signer verifier: %w", err)
	}

	if err = inhouse.Sign(ctx, sv, path); err != nil {
		return fmt.Errorf("failed to sign data: %w", err)
	}

	return nil
}
