// Copyright (c) Flant JSC
// SPDX-License-Identifier: Apache-2.0

package inhouse

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/deckhouse/delivery-kit-sdk/pkg/signature"
	elfsig "github.com/deckhouse/delivery-kit-sdk/pkg/signature/elf"
	"github.com/deckhouse/delivery-kit-sdk/pkg/signver"
)

// Sign embeds a delivery-kit signature into the provided ELF image and returns
// the modified ELF bytes. The input buffer is not modified.
func Sign(ctx context.Context, signerVerifier *signver.SignerVerifier, elfBytes []byte) ([]byte, error) {
	f, err := parseELF(elfBytes)
	if err != nil {
		return nil, err
	}
	if !supportedMachine(f.ehdr.Machine) {
		return nil, fmt.Errorf("unsupported ELF machine %d", f.ehdr.Machine)
	}

	hash, err := computeELFHash(f)
	if err != nil {
		return nil, fmt.Errorf("compute elf hash failed: %w", err)
	}

	bundle, err := signature.Sign(ctx, signerVerifier, hash)
	if err != nil {
		return nil, fmt.Errorf("sign bundle: %w", err)
	}
	bundleBytes, err := json.Marshal(bundle)
	if err != nil {
		return nil, fmt.Errorf("marshal new signature bundle: %w", err)
	}

	signed, err := saveELFSignature(f, bundleBytes)
	if err != nil {
		return nil, fmt.Errorf("saving ELF signature failed: %w", err)
	}

	// Recompute after rewrite. If the hash changed (rare non-idempotent layout
	// effects), resign against the updated image.
	upd, err := parseELF(signed)
	if err != nil {
		return nil, fmt.Errorf("parse signed ELF: %w", err)
	}
	updatedHash, err := computeELFHash(upd)
	if err != nil {
		return nil, fmt.Errorf("compute updated elf hash failed: %w", err)
	}
	if updatedHash == hash {
		return signed, nil
	}

	updatedBundle, err := signature.Sign(ctx, signerVerifier, updatedHash)
	if err != nil {
		return nil, fmt.Errorf("sign updated bundle: %w", err)
	}
	updatedBundleBytes, err := json.Marshal(updatedBundle)
	if err != nil {
		return nil, fmt.Errorf("marshal updated signature bundle: %w", err)
	}
	resigned, err := saveELFSignature(upd, updatedBundleBytes)
	if err != nil {
		return nil, fmt.Errorf("saving updated ELF signature failed: %w", err)
	}
	return resigned, nil
}

// Verify checks the delivery-kit signature embedded in the provided ELF image.
func Verify(ctx context.Context, rootCertRefs []string, elfBytes []byte) error {
	f, err := parseELF(elfBytes)
	if err != nil {
		return err
	}

	hash, err := computeELFHash(f)
	if err != nil {
		return fmt.Errorf("compute elf hash failed: %w", err)
	}

	sig, err := getELFSignature(f)
	if err != nil {
		return fmt.Errorf("get elf signature failed: %w", err)
	}
	if len(sig) == 0 {
		return elfsig.ErrNoSignatureSection
	}

	var bundle *signature.Bundle
	if err := json.Unmarshal(sig, &bundle); err != nil {
		return fmt.Errorf("unmarshal signature bundle: %w", err)
	}
	if bundle == nil {
		return fmt.Errorf("signature bundle is null")
	}

	if err := signature.VerifyBundle(ctx, *bundle, hash, rootCertRefs); err != nil {
		return fmt.Errorf("verify signature bundle: %w", err)
	}
	return nil
}
