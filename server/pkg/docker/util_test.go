package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateImageNameWithDigest_Valid(t *testing.T) {
	for _, validName := range []string{
		"repo@sha256:db6697a61d5679b7ca69dbde3dad6be0d17064d5b6b0e9f7be8d456ebb337209",
		"repo:tag@sha256:db6697a61d5679b7ca69dbde3dad6be0d17064d5b6b0e9f7be8d456ebb337209",
	} {
		t.Run(validName, func(t *testing.T) {
			err := ValidateImageNameWithDigest(validName)
			assert.Nil(t, err)
		})
	}
}

func TestValidateImageNameWithDigest_Invalid(t *testing.T) {
	for _, invalidName := range []string{
		"repo",
		"repo:tag",
		"repo:tag@123",
	} {
		t.Run(invalidName, func(t *testing.T) {
			err := ValidateImageNameWithDigest(invalidName)
			assert.Equal(t, err, ErrImageNameWithoutRequiredDigest)
		})
	}
}
