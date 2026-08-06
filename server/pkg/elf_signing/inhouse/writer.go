// Copyright (c) Flant JSC
// SPDX-License-Identifier: Apache-2.0

package inhouse

import (
	"bytes"
	"fmt"
)

const (
	symtabSectionName = ".symtab"
	strtabSectionName = ".strtab"
	shtRel            = 9
	shtRela           = 4
	shtRelr           = 19
	shdrAlign         = 8
)

// saveELFSignature returns a new ELF image with the signature note section
// added or replaced. Existing program header file offsets are preserved.
//
// Layout matches GNU objcopy --add-section:
//   - note is inserted before .symtab/.strtab/.shstrtab (whichever comes first);
//   - those trailing sections are shifted after the note;
//   - .shstrtab stays last; existing sh_name offsets are preserved (name appended
//     only when missing);
//   - the section-header table is 8-byte aligned.
func saveELFSignature(f *elfFile, payload []byte) ([]byte, error) {
	note := createELFNote(signatureNoteName, payload, signatureNoteType)

	type keptSection struct {
		oldIdx int
		hdr    elf64Shdr
		name   string
		data   []byte
	}

	insertAt := len(f.shdrs)
	for i, name := range f.names {
		if name == signatureSectionName {
			continue
		}
		if name == symtabSectionName || name == strtabSectionName || name == shstrtabSectionName {
			insertAt = i
			break
		}
	}

	var before, after []keptSection
	var shstrtabSec *keptSection

	for i, sh := range f.shdrs {
		name := f.names[i]
		if name == signatureSectionName {
			continue
		}

		var data []byte
		if sh.Type != shtNoBits && sh.Size > 0 {
			raw, err := f.sectionData(i)
			if err != nil {
				return nil, err
			}
			data = append([]byte(nil), raw...)
		}
		sec := keptSection{oldIdx: i, hdr: sh, name: name, data: data}

		switch {
		case name == shstrtabSectionName:
			shstrtabSec = &sec
		case i < insertAt:
			before = append(before, sec)
		default:
			after = append(after, sec)
		}
	}
	if shstrtabSec == nil {
		return nil, fmt.Errorf("ELF has no %s section", shstrtabSectionName)
	}

	shstrtab := shstrtabSec.data
	if len(shstrtab) == 0 {
		shstrtab = []byte{0}
	}
	shstrtab, sigNameOff := ensureShstrtabName(shstrtab, signatureSectionName)

	contentEnd := contentEndOffset(f, insertAt)
	out := make([]byte, contentEnd)
	copy(out, f.data[:contentEnd])

	// Preserve original ident padding / ABI bytes.
	if len(f.data) >= 16 {
		copy(out[7:16], f.data[7:16])
	}

	noteOff := uint64(len(out))
	out = append(out, note...)

	for i := range after {
		if after[i].hdr.Type == shtNoBits || len(after[i].data) == 0 {
			after[i].hdr.Offset = uint64(len(out))
			continue
		}
		after[i].hdr.Offset = uint64(len(out))
		out = append(out, after[i].data...)
	}

	shstrtabOff := uint64(len(out))
	out = append(out, shstrtab...)
	shstrtabSec.hdr.Offset = shstrtabOff
	shstrtabSec.hdr.Size = uint64(len(shstrtab))
	shstrtabSec.data = shstrtab

	sigHdr := elf64Shdr{
		Name:      sigNameOff,
		Type:      shtNote,
		Flags:     shfWrite,
		Addr:      0,
		Offset:    noteOff,
		Size:      uint64(len(note)),
		Link:      0,
		Info:      0,
		Addralign: 1,
		Entsize:   0,
	}

	// objcopy 8-byte-aligns the section header table.
	if pad := len(out) % shdrAlign; pad != 0 {
		out = append(out, make([]byte, shdrAlign-pad)...)
	}

	shdrs := make([]elf64Shdr, 0, len(before)+1+len(after)+1)
	oldToNew := make(map[int]int, len(f.shdrs)+1)

	appendMapped := func(sec keptSection) {
		oldToNew[sec.oldIdx] = len(shdrs)
		shdrs = append(shdrs, sec.hdr)
	}
	for _, sec := range before {
		appendMapped(sec)
	}
	sigIdx := len(shdrs)
	shdrs = append(shdrs, sigHdr)
	for _, sec := range after {
		appendMapped(sec)
	}
	shstrtabIdx := len(shdrs)
	oldToNew[shstrtabSec.oldIdx] = shstrtabIdx
	shdrs = append(shdrs, shstrtabSec.hdr)

	for i := range shdrs {
		if i == sigIdx {
			continue
		}
		shdrs[i].Link = remapSectionIndex(shdrs[i].Link, oldToNew)
		if sectionInfoIsIndex(shdrs[i].Type) {
			shdrs[i].Info = remapSectionIndex(shdrs[i].Info, oldToNew)
		}
	}

	shnum := uint64(len(shdrs))
	shstrndx := uint64(shstrtabIdx)

	if shnum >= shnXIndex {
		if len(shdrs) == 0 || shdrs[0].Type != shtNull {
			return nil, fmt.Errorf("cannot encode extended section count without null section")
		}
		shdrs[0].Size = shnum
		shdrs[0].Link = uint32(shstrndx)
	}

	shoff := uint64(len(out))
	for _, sh := range shdrs {
		out = append(out, encodeShdr(f.byteOrder, sh)...)
	}

	ehdr := f.ehdr
	ehdr.Shoff = shoff
	ehdr.Shentsize = 64
	if shnum < shnXIndex {
		ehdr.Shnum = uint16(shnum)
		ehdr.Shstrndx = uint16(shstrndx)
	} else {
		ehdr.Shnum = 0
		ehdr.Shstrndx = shnXIndex
	}
	if ehdr.Ehsize == 0 {
		ehdr.Ehsize = 64
	}

	if err := writeELFHeader(out, f.byteOrder, ehdr); err != nil {
		return nil, err
	}
	return out, nil
}

func remapSectionIndex(idx uint32, oldToNew map[int]int) uint32 {
	if idx == 0 {
		return 0
	}
	if newIdx, ok := oldToNew[int(idx)]; ok {
		return uint32(newIdx)
	}
	return idx
}

func sectionInfoIsIndex(typ uint32) bool {
	return typ == shtRel || typ == shtRela || typ == shtRelr
}

// contentEndOffset is the file offset where the signature note is written:
// after all sections that stay in place (those before insertAt), excluding
// the to-be-relocated trailing sections, .shstrtab, and any existing signature.
func contentEndOffset(f *elfFile, insertAt int) uint64 {
	var end uint64 = 64
	if f.ehdr.Phoff > 0 && f.ehdr.Phentsize > 0 {
		phEnd := f.ehdr.Phoff + uint64(len(f.phdrs))*uint64(f.ehdr.Phentsize)
		if phEnd > end {
			end = phEnd
		}
	}
	for i, sh := range f.shdrs {
		if i >= insertAt {
			continue
		}
		if f.names[i] == signatureSectionName || f.names[i] == shstrtabSectionName {
			continue
		}
		if sh.Type == shtNoBits || sh.Size == 0 {
			continue
		}
		secEnd := sh.Offset + sh.Size
		if secEnd > end {
			end = secEnd
		}
	}
	for _, ph := range f.phdrs {
		if ph.Filesz == 0 {
			continue
		}
		phEnd := ph.Offset + ph.Filesz
		if phEnd > end {
			end = phEnd
		}
	}
	if end > uint64(len(f.data)) {
		end = uint64(len(f.data))
	}

	// Drop a trailing section-header table so it can be rewritten at EOF.
	if f.ehdr.Shoff > 0 {
		shdrStart := f.ehdr.Shoff
		shdrEnd := shdrStart + uint64(len(f.shdrs))*64
		if shdrEnd == uint64(len(f.data)) && shdrStart < end {
			end = shdrStart
		}
	}
	return end
}

// ensureShstrtabName returns shstrtab with name present as a full C string,
// appending it when missing. Existing string offsets are preserved.
func ensureShstrtabName(shstrtab []byte, name string) ([]byte, uint32) {
	if off, ok := findShstrtabName(shstrtab, name); ok {
		return shstrtab, off
	}
	off := uint32(len(shstrtab))
	out := make([]byte, len(shstrtab), len(shstrtab)+len(name)+1)
	copy(out, shstrtab)
	out = append(out, name...)
	out = append(out, 0)
	return out, off
}

func findShstrtabName(shstrtab []byte, name string) (uint32, bool) {
	want := append([]byte(name), 0)
	for i := 0; i < len(shstrtab); {
		if i+len(want) <= len(shstrtab) && bytes.Equal(shstrtab[i:i+len(want)], want) {
			return uint32(i), true
		}
		for i < len(shstrtab) && shstrtab[i] != 0 {
			i++
		}
		i++ // skip NUL
	}
	return 0, false
}

func getELFSignature(f *elfFile) ([]byte, error) {
	idx := f.findSection(signatureSectionName)
	if idx < 0 {
		return nil, nil
	}
	data, err := f.sectionData(idx)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	note, err := parseELFNote(data)
	if err != nil {
		return nil, fmt.Errorf("parse signature note: %w", err)
	}
	if note.Type != signatureNoteType {
		return nil, fmt.Errorf("unexpected note type: expected %#x, got %#x", signatureNoteType, note.Type)
	}
	return note.Desc, nil
}
