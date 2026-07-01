//go:build !(linux && cgo)

package flow

import "context"

// verifyELFSignature is a no-op on platforms without the CGO welf backend.
// ELF signing/verification is only exercised in the linux+cgo E2E environment.
func verifyELFSignature(_ context.Context, _, _ string) error {
	return nil
}
