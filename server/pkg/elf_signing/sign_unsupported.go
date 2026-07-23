//go:build !linux || !amd64 || !cgo

package elf_signing

import (
	"context"
	"errors"

	"github.com/deckhouse/delivery-kit-sdk/pkg/signver"
)

const Supported = false

func signELF(_ context.Context, _ *signver.SignerVerifier, _ string) error {
	return errors.New("ELF signing requires a linux/amd64 build with CGO enabled")
}
