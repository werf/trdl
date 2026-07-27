//go:build linux && amd64 && cgo

package elf_signing

import (
	"context"

	"github.com/deckhouse/delivery-kit-sdk/pkg/signature/elf/inhouse"
	"github.com/deckhouse/delivery-kit-sdk/pkg/signver"
)

const supported = true

func signELF(ctx context.Context, sv *signver.SignerVerifier, path string) error {
	return inhouse.Sign(ctx, sv, path)
}
