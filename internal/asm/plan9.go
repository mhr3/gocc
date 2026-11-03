package asm

import (
	"bytes"
	"debug/elf"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/mhr3/gocc/internal/config"
	"golang.org/x/arch/arm64/arm64asm"
	"golang.org/x/arch/x86/x86asm"
)

// Plan9Decoder translates the machine code of a function's instruction into
// cmd/asm's Plan 9 spelling.
type Plan9Decoder interface {
	DecodeInstruction(symName string, binary []string) (string, error)
}

// ObjectDecoder decodes instructions of an ELF relocatable object produced
// by clang. It keeps the code of every function so PC-relative operands can
// be rendered relative to the instruction's real position.
type ObjectDecoder struct {
	arch    *config.Arch
	symbols map[string]objectSymbol
}

type objectSymbol struct {
	code  []byte // the whole section the symbol lives in
	start uint64 // symbol offset within the section
	size  uint64
}

// NewObjectDecoder reads the executable sections and function symbols of an
// ELF object file.
func NewObjectDecoder(arch *config.Arch, path string) (*ObjectDecoder, error) {
	file, err := elf.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	symbols, err := file.Symbols()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	sections := make(map[elf.SectionIndex][]byte)
	decoder := &ObjectDecoder{arch: arch, symbols: make(map[string]objectSymbol, len(symbols))}
	for _, sym := range symbols {
		if elf.ST_TYPE(sym.Info) != elf.STT_FUNC || int(sym.Section) >= len(file.Sections) {
			continue
		}
		section := file.Sections[sym.Section]
		if section.Flags&elf.SHF_EXECINSTR == 0 {
			continue
		}
		code, cached := sections[sym.Section]
		if !cached {
			if code, err = section.Data(); err != nil {
				return nil, fmt.Errorf("%s: section %s: %w", path, section.Name, err)
			}
			sections[sym.Section] = code
		}
		// Relocatable objects address symbols relative to their section.
		start := sym.Value - section.Addr
		if start > uint64(len(code)) {
			return nil, fmt.Errorf("%s: symbol %s lies outside section %s", path, sym.Name, section.Name)
		}
		decoder.symbols[sym.Name] = objectSymbol{code: code, start: start, size: sym.Size}
	}
	return decoder, nil
}

// DecodeInstruction renders the instruction encoded by binary (hex bytes in
// memory order) that belongs to symName. An empty string means the bytes do
// not form exactly one decodable instruction; the caller then keeps the raw
// encoding.
func (d *ObjectDecoder) DecodeInstruction(symName string, binary []string) (string, error) {
	sym, ok := d.symbols[symName]
	if !ok {
		return "", fmt.Errorf("symbol %q not found in object file", symName)
	}
	code, err := hex.DecodeString(strings.Join(binary, ""))
	if err != nil {
		return "", fmt.Errorf("failed to decode binary: %v", err)
	}
	// Locate the instruction so PC-relative operands are rendered against
	// its real position. The bytes are unique often enough; when they are not,
	// the first occurrence is still within the same function.
	body := sym.code[sym.start:]
	if sym.size > 0 && sym.size <= uint64(len(body)) {
		body = body[:sym.size]
	}
	offset := bytes.Index(body, code)
	if offset == -1 {
		return "", fmt.Errorf("instruction %q not found in %q", strings.Join(binary, " "), symName)
	}
	pc := sym.start + uint64(offset)

	switch d.arch.Name {
	case "amd64":
		inst, err := x86asm.Decode(code, 64)
		if err != nil || inst.Len != len(code) {
			return "", nil
		}
		return x86asm.GoSyntax(inst, pc, nil), nil
	case "arm64":
		inst, err := arm64asm.Decode(code)
		if err != nil || len(code) != 4 {
			return "", nil
		}
		return arm64asm.GoSyntax(inst, pc, nil, sectionReader(sym.code)), nil
	}
	return "", fmt.Errorf("no Plan 9 decoder for architecture %s", d.arch.Name)
}

// sectionReader lets arm64asm resolve literal-pool loads (LDR Rt, label).
type sectionReader []byte

func (r sectionReader) ReadAt(data []byte, off int64) (int, error) {
	if off < 0 || off >= int64(len(r)) {
		return 0, fmt.Errorf("offset %d out of section", off)
	}
	n := copy(data, r[off:])
	if n < len(data) {
		return n, fmt.Errorf("short read")
	}
	return n, nil
}
