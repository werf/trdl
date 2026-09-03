// Copyright (c) Flant JSC
// SPDX-License-Identifier: Apache-2.0

package inhouse

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// computeELFHash returns a lowercase hex SHA-256 digest matching welf_compute_elf_hash.
//
// It hashes host-endian GElf_Phdr/GElf_Shdr layouts (Elf64_* on this implementation)
// with selected fields zeroed, then raw section bytes excluding the signature section,
// legacy bsign "signature", .shstrtab and SHT_NOBITS sections.
//
// Section 0 (SHT_NULL) is skipped to match libelf elf_nextscn(), which never yields it.
func computeELFHash(f *elfFile) (string, error) {
	h := sha256.New()
	bo := f.byteOrder

	for i := range f.phdrs {
		var buf [56]byte
		phdr := f.phdrs[i]
		bo.PutUint32(buf[0:4], phdr.Type)
		bo.PutUint32(buf[4:8], phdr.Flags)
		// p_offset, p_vaddr, p_paddr, p_filesz, p_memsz, p_align remain zero.
		if _, err := h.Write(buf[:]); err != nil {
			return "", err
		}
	}

	for i := 1; i < len(f.shdrs); i++ {
		name := f.names[i]
		sh := f.shdrs[i]
		if name == signatureSectionName || name == bsignSectionName || name == shstrtabSectionName {
			continue
		}
		if sh.Type == shtNoBits {
			continue
		}

		var hdr [64]byte
		bo.PutUint32(hdr[0:4], sh.Name)
		bo.PutUint32(hdr[4:8], sh.Type)
		bo.PutUint64(hdr[8:16], sh.Flags)
		// sh_addr, sh_offset, sh_size, sh_link, sh_info, sh_addralign, sh_entsize remain zero.
		if _, err := h.Write(hdr[:]); err != nil {
			return "", err
		}

		data, err := f.sectionData(i)
		if err != nil {
			return "", err
		}
		if len(data) == 0 {
			continue
		}

		const chunkSize = 4096
		for off := 0; off < len(data); off += chunkSize {
			end := off + chunkSize
			if end > len(data) {
				end = len(data)
			}
			if _, err := h.Write(data[off:end]); err != nil {
				return "", err
			}
		}
	}

	sum := h.Sum(nil)
	return hex.EncodeToString(sum), nil
}

func encodeShdr(bo binary.ByteOrder, s elf64Shdr) []byte {
	buf := make([]byte, 64)
	bo.PutUint32(buf[0:4], s.Name)
	bo.PutUint32(buf[4:8], s.Type)
	bo.PutUint64(buf[8:16], s.Flags)
	bo.PutUint64(buf[16:24], s.Addr)
	bo.PutUint64(buf[24:32], s.Offset)
	bo.PutUint64(buf[32:40], s.Size)
	bo.PutUint32(buf[40:44], s.Link)
	bo.PutUint32(buf[44:48], s.Info)
	bo.PutUint64(buf[48:56], s.Addralign)
	bo.PutUint64(buf[56:64], s.Entsize)
	return buf
}

func writeELFHeader(dst []byte, bo binary.ByteOrder, ehdr elf64Ehdr) error {
	if len(dst) < 64 {
		return fmt.Errorf("buffer too small for ELF header")
	}
	copy(dst[0:4], []byte(elfMagic))
	dst[4] = eiClassELF64
	dst[5] = eiDataLE
	dst[6] = evCurrent
	// EI_OSABI / EI_ABIVERSION / padding left as-is by caller when patching,
	// or zeroed for fresh buffers. Preserve existing ABI bytes when present.
	bo.PutUint16(dst[16:18], ehdr.Type)
	bo.PutUint16(dst[18:20], ehdr.Machine)
	bo.PutUint32(dst[20:24], ehdr.Version)
	bo.PutUint64(dst[24:32], ehdr.Entry)
	bo.PutUint64(dst[32:40], ehdr.Phoff)
	bo.PutUint64(dst[40:48], ehdr.Shoff)
	bo.PutUint32(dst[48:52], ehdr.Flags)
	bo.PutUint16(dst[52:54], ehdr.Ehsize)
	bo.PutUint16(dst[54:56], ehdr.Phentsize)
	bo.PutUint16(dst[56:58], ehdr.Phnum)
	bo.PutUint16(dst[58:60], ehdr.Shentsize)
	bo.PutUint16(dst[60:62], ehdr.Shnum)
	bo.PutUint16(dst[62:64], ehdr.Shstrndx)
	return nil
}
