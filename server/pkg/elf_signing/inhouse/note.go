// Copyright (c) Flant JSC
// SPDX-License-Identifier: Apache-2.0

package inhouse

import (
	"encoding/binary"
	"fmt"
)

const (
	signatureSectionName = ".note.delivery-kit.signature"
	signatureNoteName    = "delivery-kit.signature"
	signatureNoteType    = 0x31415926
	bsignSectionName     = "signature"
	shstrtabSectionName  = ".shstrtab"
)

// createELFNote builds a single ELF note payload.
func createELFNote(name string, desc []byte, noteType uint32) []byte {
	nameBytes := append([]byte(name), 0)
	nameSize := uint32(len(nameBytes))
	descSize := uint32(len(desc))
	namePadded := (nameSize + 3) &^ 3
	descPadded := (descSize + 3) &^ 3
	total := 12 + namePadded + descPadded

	buf := make([]byte, total)
	binary.LittleEndian.PutUint32(buf[0:4], nameSize)
	binary.LittleEndian.PutUint32(buf[4:8], descSize)
	binary.LittleEndian.PutUint32(buf[8:12], noteType)
	copy(buf[12:], nameBytes)
	if descSize > 0 {
		copy(buf[12+namePadded:], desc)
	}
	return buf
}

type elfNote struct {
	Name string
	Desc []byte
	Type uint32
}

func parseELFNote(data []byte) (*elfNote, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("note data size too small for ELF note header")
	}

	nameSize := binary.LittleEndian.Uint32(data[0:4])
	descSize := binary.LittleEndian.Uint32(data[4:8])
	noteType := binary.LittleEndian.Uint32(data[8:12])
	if nameSize == 0 {
		return nil, fmt.Errorf("note name size is zero")
	}

	namePadded := (nameSize + 3) &^ 3
	descOffset := 12 + namePadded
	if int(descOffset)+int(descSize) > len(data) {
		return nil, fmt.Errorf("note descriptor out of bounds")
	}

	nameEnd := 12 + nameSize
	if int(nameEnd) > len(data) {
		return nil, fmt.Errorf("note name out of bounds")
	}
	name := data[12:nameEnd]
	if len(name) > 0 && name[len(name)-1] == 0 {
		name = name[:len(name)-1]
	}

	var desc []byte
	if descSize > 0 {
		desc = append([]byte(nil), data[descOffset:descOffset+descSize]...)
	}

	return &elfNote{
		Name: string(name),
		Desc: desc,
		Type: noteType,
	}, nil
}
