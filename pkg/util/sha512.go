package util

import (
	"crypto/sha512"
	"fmt"
)

func Sha512Checksum(data []byte) string {
	h := sha512.New()
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}
