//go:build linux && cgo

package flow

import (
	"context"

	"github.com/deckhouse/delivery-kit-sdk/pkg/signature/elf/inhouse"
)

// verifyELFSignature validates the embedded signature of an ELF binary against
// the provided base64-encoded PEM root CA. It relies on the CGO welf-backed
// implementation, so it is only compiled on linux with cgo enabled.
func verifyELFSignature(ctx context.Context, rootCARef, path string) error {
	return inhouse.Verify(ctx, []string{rootCARef}, path)
}
