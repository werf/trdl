// Copyright (c) Flant JSC
// SPDX-License-Identifier: Apache-2.0

package inhouse

import (
	"debug/elf"
	"encoding/binary"
	"fmt"

	elfsig "github.com/deckhouse/delivery-kit-sdk/pkg/signature/elf"
)

const (
	elfMagic     = "\x7fELF"
	eiClassELF64 = 2
	eiDataLE     = 1
	evCurrent    = 1
	shtNull      = 0
	shtNote      = 7
	shtNoBits    = 8
	shfWrite     = 0x1
	shnXIndex    = 0xffff
	pnXNum       = 0xffff
)

type elfFile struct {
	data      []byte
	byteOrder binary.ByteOrder
	ehdr      elf64Ehdr
	phdrs     []elf64Phdr
	shdrs     []elf64Shdr
	names     []string
}

type elf64Ehdr struct {
	Type      uint16
	Machine   uint16
	Version   uint32
	Entry     uint64
	Phoff     uint64
	Shoff     uint64
	Flags     uint32
	Ehsize    uint16
	Phentsize uint16
	Phnum     uint16
	Shentsize uint16
	Shnum     uint16
	Shstrndx  uint16
}

type elf64Phdr struct {
	Type   uint32
	Flags  uint32
	Offset uint64
	Vaddr  uint64
	Paddr  uint64
	Filesz uint64
	Memsz  uint64
	Align  uint64
}

type elf64Shdr struct {
	Name      uint32
	Type      uint32
	Flags     uint64
	Addr      uint64
	Offset    uint64
	Size      uint64
	Link      uint32
	Info      uint32
	Addralign uint64
	Entsize   uint64
}

func parseELF(data []byte) (*elfFile, error) {
	if len(data) < 64 {
		return nil, elfsig.ErrNotELF
	}
	if string(data[0:4]) != elfMagic {
		return nil, elfsig.ErrNotELF
	}
	if data[4] != eiClassELF64 {
		return nil, fmt.Errorf("unsupported ELF class: only ELF64 is supported")
	}
	if data[5] != eiDataLE {
		return nil, fmt.Errorf("unsupported ELF endianness: only little-endian is supported")
	}
	if data[6] != evCurrent {
		return nil, elfsig.ErrNotELF
	}

	bo := binary.LittleEndian
	ehdr := elf64Ehdr{
		Type:      bo.Uint16(data[16:18]),
		Machine:   bo.Uint16(data[18:20]),
		Version:   bo.Uint32(data[20:24]),
		Entry:     bo.Uint64(data[24:32]),
		Phoff:     bo.Uint64(data[32:40]),
		Shoff:     bo.Uint64(data[40:48]),
		Flags:     bo.Uint32(data[48:52]),
		Ehsize:    bo.Uint16(data[52:54]),
		Phentsize: bo.Uint16(data[54:56]),
		Phnum:     bo.Uint16(data[56:58]),
		Shentsize: bo.Uint16(data[58:60]),
		Shnum:     bo.Uint16(data[60:62]),
		Shstrndx:  bo.Uint16(data[62:64]),
	}

	if ehdr.Shoff == 0 && ehdr.Shnum == 0 {
		return nil, elfsig.ErrNoSections
	}
	if ehdr.Shentsize != 0 && ehdr.Shentsize != 64 {
		return nil, fmt.Errorf("unsupported section header entry size %d", ehdr.Shentsize)
	}
	if ehdr.Phentsize != 0 && ehdr.Phentsize != 56 {
		return nil, fmt.Errorf("unsupported program header entry size %d", ehdr.Phentsize)
	}

	f := &elfFile{
		data:      data,
		byteOrder: bo,
		ehdr:      ehdr,
	}

	if err := f.parseProgramHeaders(); err != nil {
		return nil, err
	}
	if err := f.parseSectionHeaders(); err != nil {
		return nil, err
	}
	return f, nil
}

func (f *elfFile) parseProgramHeaders() error {
	phnum := uint64(f.ehdr.Phnum)
	if f.ehdr.Phnum == pnXNum {
		// Actual count is in section header 0's sh_info (gABI extended numbering).
		if f.ehdr.Shoff == 0 || f.ehdr.Shoff+64 > uint64(len(f.data)) {
			return fmt.Errorf("PN_XNUM: section header 0 out of bounds")
		}
		sh0 := f.readShdr(f.ehdr.Shoff)
		phnum = uint64(sh0.Info)
		if phnum == 0 {
			return fmt.Errorf("PN_XNUM: section 0 sh_info is zero")
		}
	}
	if phnum == 0 {
		return nil
	}
	if f.ehdr.Phoff == 0 {
		return fmt.Errorf("program headers claimed but e_phoff is zero")
	}
	if f.ehdr.Phentsize != 56 {
		return fmt.Errorf("program headers claimed but entry size is %d", f.ehdr.Phentsize)
	}

	tableSize := phnum * 56
	if tableSize/56 != phnum || f.ehdr.Phoff+tableSize < f.ehdr.Phoff {
		return fmt.Errorf("program headers size overflows")
	}
	need := f.ehdr.Phoff + tableSize
	if need > uint64(len(f.data)) {
		return fmt.Errorf("program headers out of bounds")
	}

	f.phdrs = make([]elf64Phdr, phnum)
	for i := uint64(0); i < phnum; i++ {
		off := f.ehdr.Phoff + i*uint64(f.ehdr.Phentsize)
		raw := f.data[off : off+56]
		f.phdrs[i] = elf64Phdr{
			Type:   f.byteOrder.Uint32(raw[0:4]),
			Flags:  f.byteOrder.Uint32(raw[4:8]),
			Offset: f.byteOrder.Uint64(raw[8:16]),
			Vaddr:  f.byteOrder.Uint64(raw[16:24]),
			Paddr:  f.byteOrder.Uint64(raw[24:32]),
			Filesz: f.byteOrder.Uint64(raw[32:40]),
			Memsz:  f.byteOrder.Uint64(raw[40:48]),
			Align:  f.byteOrder.Uint64(raw[48:56]),
		}
	}
	return nil
}

func (f *elfFile) parseSectionHeaders() error {
	shnum := uint64(f.ehdr.Shnum)
	shoff := f.ehdr.Shoff
	shstrndx := uint64(f.ehdr.Shstrndx)

	if f.ehdr.Shnum == 0 {
		// Extended section numbering: section 0 sh_size holds the real count.
		if shoff == 0 || shoff+64 > uint64(len(f.data)) {
			return fmt.Errorf("extended section numbering: section 0 out of bounds")
		}
		sh0 := f.readShdr(shoff)
		shnum = sh0.Size
		if f.ehdr.Shstrndx == shnXIndex {
			shstrndx = uint64(sh0.Link)
		}
	}

	if shnum == 0 {
		return elfsig.ErrNoSections
	}
	need := shoff + shnum*64
	if need > uint64(len(f.data)) {
		return fmt.Errorf("section headers out of bounds")
	}

	f.shdrs = make([]elf64Shdr, shnum)
	for i := uint64(0); i < shnum; i++ {
		f.shdrs[i] = f.readShdr(shoff + i*64)
	}

	if shstrndx >= shnum {
		return fmt.Errorf("invalid e_shstrndx %d", shstrndx)
	}
	strtab := f.shdrs[shstrndx]
	if strtab.Type == shtNoBits {
		return fmt.Errorf(".shstrtab is SHT_NOBITS")
	}
	if strtab.Offset+strtab.Size > uint64(len(f.data)) {
		return fmt.Errorf(".shstrtab out of bounds")
	}
	strtabData := f.data[strtab.Offset : strtab.Offset+strtab.Size]

	f.names = make([]string, shnum)
	for i := range f.shdrs {
		name, err := readCString(strtabData, f.shdrs[i].Name)
		if err != nil {
			return fmt.Errorf("section %d name: %w", i, err)
		}
		f.names[i] = name
	}

	// Preserve resolved counts for later rewrite.
	f.ehdr.Shnum = uint16(shnum)
	if shnum >= shnXIndex {
		f.ehdr.Shnum = 0
	}
	f.ehdr.Shstrndx = uint16(shstrndx)
	if shstrndx >= shnXIndex {
		f.ehdr.Shstrndx = shnXIndex
	}
	return nil
}

func (f *elfFile) readShdr(off uint64) elf64Shdr {
	raw := f.data[off : off+64]
	return elf64Shdr{
		Name:      f.byteOrder.Uint32(raw[0:4]),
		Type:      f.byteOrder.Uint32(raw[4:8]),
		Flags:     f.byteOrder.Uint64(raw[8:16]),
		Addr:      f.byteOrder.Uint64(raw[16:24]),
		Offset:    f.byteOrder.Uint64(raw[24:32]),
		Size:      f.byteOrder.Uint64(raw[32:40]),
		Link:      f.byteOrder.Uint32(raw[40:44]),
		Info:      f.byteOrder.Uint32(raw[44:48]),
		Addralign: f.byteOrder.Uint64(raw[48:56]),
		Entsize:   f.byteOrder.Uint64(raw[56:64]),
	}
}

func (f *elfFile) sectionData(i int) ([]byte, error) {
	sh := f.shdrs[i]
	if sh.Type == shtNoBits || sh.Size == 0 {
		return nil, nil
	}
	if sh.Offset+sh.Size > uint64(len(f.data)) {
		return nil, fmt.Errorf("section %q data out of bounds", f.names[i])
	}
	return f.data[sh.Offset : sh.Offset+sh.Size], nil
}

func (f *elfFile) findSection(name string) int {
	for i, n := range f.names {
		if n == name {
			return i
		}
	}
	return -1
}

func readCString(data []byte, offset uint32) (string, error) {
	if uint64(offset) >= uint64(len(data)) {
		return "", fmt.Errorf("string offset out of bounds")
	}
	end := offset
	for end < uint32(len(data)) && data[end] != 0 {
		end++
	}
	return string(data[offset:end]), nil
}

func supportedMachine(machine uint16) bool {
	return machine == uint16(elf.EM_X86_64) || machine == uint16(elf.EM_AARCH64)
}
