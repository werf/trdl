package elf_signing

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseArtifactSize(t *testing.T) {
	for _, testCase := range []struct {
		value string
		want  int64
	}{
		{value: "1B", want: 1},
		{value: "512MiB", want: 512 << 20},
		{value: "1GiB", want: 1 << 30},
		{value: "7EiB", want: 7 << 60},
	} {
		got, err := parseArtifactSize(testCase.value)
		require.NoError(t, err)
		require.Equal(t, testCase.want, got)
	}

	for _, value := range []string{"", "0B", "-1MiB", "1MB", "9EiB", "1.5MiB"} {
		_, err := parseArtifactSize(value)
		require.Error(t, err)
	}
}
