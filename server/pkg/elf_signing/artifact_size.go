package elf_signing

import (
	"fmt"
	"math"
	"strconv"
)

const defaultMaxArtifactSize = "512MiB"

func parseArtifactSize(value string) (int64, error) {
	digits := 0
	for digits < len(value) && value[digits] >= '0' && value[digits] <= '9' {
		digits++
	}
	if digits == 0 || digits == len(value) {
		return 0, fmt.Errorf("invalid IEC size %q", value)
	}

	amount, err := strconv.ParseInt(value[:digits], 10, 64)
	if err != nil || amount <= 0 {
		return 0, fmt.Errorf("invalid IEC size %q", value)
	}
	multiplier, ok := map[string]int64{
		"B":   1,
		"KiB": 1 << 10,
		"MiB": 1 << 20,
		"GiB": 1 << 30,
		"TiB": 1 << 40,
		"PiB": 1 << 50,
		"EiB": 1 << 60,
	}[value[digits:]]
	if !ok || amount > math.MaxInt64/multiplier {
		return 0, fmt.Errorf("invalid IEC size %q", value)
	}

	return amount * multiplier, nil
}
