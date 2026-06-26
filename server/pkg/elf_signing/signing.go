package elf_signing

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/deckhouse/delivery-kit-sdk/pkg/signature/elf/inhouse"
	"github.com/deckhouse/delivery-kit-sdk/pkg/signver"
	"github.com/deckhouse/delivery-kit-sdk/pkg/signver/hashivault"
	"github.com/samber/lo"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
)

// signMu serializes ELF signing across the whole process. The delivery-kit-sdk
// configures Vault transit access only through process-global env vars
// (VAULT_ADDR, role/secret id, etc.), so concurrent signings from different
// plugin mounts would otherwise race and leak settings into each other.
var signMu sync.Mutex

func setVaultEnvVars(opts VaultSignerOpts) func() {
	origEnvs := map[string]string{}
	envs := map[string]string{
		"VAULT_ADDR":                 opts.Address,
		"TRANSIT_SECRET_ENGINE_PATH": opts.TransitPath,
		"WERF_VAULT_AUTH_PATH":       lo.CoalesceOrEmpty(opts.AuthPath, "ar"),
		"VAULT_ROLE_ID":              opts.AuthRoleID,
		"VAULT_SECRET_ID":            opts.AuthSecretID,
	}

	for envKey, val := range envs {
		origVal, _ := os.LookupEnv(envKey)
		origEnvs[envKey] = origVal

		if err := os.Setenv(envKey, val); err != nil {
			panic(fmt.Errorf("failed to set env var %q: %w", envKey, err))
		}
	}

	return func() {
		for envKey, origVal := range origEnvs {
			if err := os.Setenv(envKey, origVal); err != nil {
				panic(fmt.Errorf("failed to restore env var %q: %w", envKey, err))
			}
		}
	}
}

func Sign(ctx context.Context, path string, opts SignerSettings) error {
	signMu.Lock()
	defer signMu.Unlock()

	if strings.HasPrefix(opts.KeyRef, hashivault.ReferenceScheme) {
		restoreEnv := setVaultEnvVars(opts.VaultOpts)
		defer restoreEnv()
	}

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
