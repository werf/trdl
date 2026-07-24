//go:build ai_tests

package elf_signing

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAI_TempFileCloserCloseIsIdempotent(t *testing.T) {
	f, err := os.CreateTemp("", "trdl-close-idempotent-*")
	require.NoError(t, err)
	name := f.Name()

	calls := 0
	closer := &tempFileCloser{
		File: f,
		cleanup: func() error {
			calls++
			_ = f.Close()
			return os.Remove(name)
		},
	}

	require.NoError(t, closer.Close())
	require.NoError(t, closer.Close())
	require.Equal(t, 1, calls)

	_, statErr := os.Stat(name)
	require.True(t, os.IsNotExist(statErr))
}
