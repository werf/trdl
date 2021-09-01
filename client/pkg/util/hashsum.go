package util

import (
	"crypto/sha512"
	"fmt"
	"strings"

	"github.com/spaolacci/murmur3"
)

func Sha512Checksum(data []byte) string {
	h := sha512.New()
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func MurmurHash(args ...string) string {
	h32 := murmur3.New32()
	_, _ = h32.Write([]byte(strings.Join(args, ":::")))
	sum := h32.Sum32()
	return fmt.Sprintf("%x", sum)
}
