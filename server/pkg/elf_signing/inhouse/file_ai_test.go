//go:build ai_tests

package inhouse

import (
	"encoding/binary"
	"strings"
	"testing"
)

func TestAI_ParseExpandsPNXNUMFromSection0(t *testing.T) {
	raw := buildMinimalELF64(t)
	baseline, err := parseELF(raw)
	if err != nil {
		t.Fatalf("parse baseline: %v", err)
	}
	if len(baseline.phdrs) != 1 {
		t.Fatalf("baseline phdr count: got %d, want 1", len(baseline.phdrs))
	}
	baselineHash, err := computeELFHash(baseline)
	if err != nil {
		t.Fatalf("baseline hash: %v", err)
	}

	bo := binary.LittleEndian
	bo.PutUint16(raw[56:58], pnXNum)
	shoff := bo.Uint64(raw[40:48])
	bo.PutUint32(raw[shoff+44:shoff+48], 1) // section 0 sh_info = real phnum

	f, err := parseELF(raw)
	if err != nil {
		t.Fatalf("parse PN_XNUM ELF: %v", err)
	}
	if f.ehdr.Phnum != pnXNum {
		t.Fatalf("e_phnum rewritten: got %#x, want %#x", f.ehdr.Phnum, pnXNum)
	}
	if len(f.phdrs) != 1 {
		t.Fatalf("expanded phdr count: got %d, want 1", len(f.phdrs))
	}
	if f.phdrs[0] != baseline.phdrs[0] {
		t.Fatalf("expanded phdr mismatch:\n got:  %+v\n want: %+v", f.phdrs[0], baseline.phdrs[0])
	}
	gotHash, err := computeELFHash(f)
	if err != nil {
		t.Fatalf("PN_XNUM hash: %v", err)
	}
	if gotHash != baselineHash {
		t.Fatalf("hash diverged under PN_XNUM:\n got:  %s\n want: %s", gotHash, baselineHash)
	}
}

func TestAI_ParseRejectsPNXNUMWithZeroShInfo(t *testing.T) {
	raw := buildMinimalELF64(t)
	bo := binary.LittleEndian
	bo.PutUint16(raw[56:58], pnXNum)

	_, err := parseELF(raw)
	if err == nil {
		t.Fatal("expected PN_XNUM parse error")
	}
	if !strings.Contains(err.Error(), "sh_info is zero") {
		t.Fatalf("unexpected error: %v", err)
	}
}
