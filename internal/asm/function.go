// Copyright 2022 gorse Project Authors
// Copyright 2023 Roman Atachiants
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package asm

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/types"
	"regexp"
	"strconv"
	"strings"

	"github.com/mhr3/gocc/internal/config"
)

var constSizes = map[string]int{
	"byte":  1,
	"short": 2,
	"hword": 2,
	"word":  4,
	"long":  4,
	"int":   4,
	"quad":  8,
}

type GoFunction struct {
	Name string
	Expr *ast.FuncType
}

// ------------------------------------- Function -------------------------------------

type Function struct {
	Name       string  `json:"name"`
	SourcePath string  `json:"sourcePath"`
	Position   int     `json:"position"`
	Params     []Param `json:"params"`
	Consts     []Const `json:"consts,omitempty"`
	Lines      []Line  `json:"lines"`
	LocalsSize int     `json:"localsSize,omitempty"`
	Ret        *Param  `json:"return,omitempty"`
	GoFunc     GoFunction
}

func (f *GoFunction) NumResults() int {
	return f.Expr.Results.NumFields()
}

func (f *GoFunction) iterFieldList(fl *ast.FieldList, fn func(name, typ string)) {
	if fl == nil {
		return
	}
	for _, field := range fl.List {
		typ := types.ExprString(field.Type)
		if len(field.Names) == 0 {
			fn("", typ)
		}
		for _, name := range field.Names {
			fn(name.Name, typ)
		}
	}
}

func (f *GoFunction) ForEachParam(fn func(name, typ string)) {
	f.iterFieldList(f.Expr.Params, fn)
}

func (f *GoFunction) ForEachResult(fn func(name, typ string)) {
	f.iterFieldList(f.Expr.Results, fn)
}

func (f *Function) ParamsSize(arch *config.Arch) (int, []int) {
	// based on go vet's go/analysis/passes/asmdecl/asmdecl.go
	var sizes types.Sizes
	if arch == nil {
		sizes = types.SizesFor("gc", "amd64")
	} else {
		sizes = types.SizesFor("gc", arch.Name)
	}

	offset := 0
	offsets := make([]int, len(f.Params))

	for i, param := range f.Params {
		t := param.EquivalentType()

		align := int(sizes.Alignof(t))
		size := int(sizes.Sizeof(t))
		offset += -offset & (align - 1)

		offsets[i] = offset

		offset += size
	}

	// we calc the return size separately, but still account for it
	if f.Ret != nil {
		maxAlign := int(sizes.Alignof(types.Typ[types.Int64]))
		offset += -offset & (maxAlign - 1)
	}

	return offset, offsets
}

// String returns the function signature for a Go stub
func (f *Function) String() string {
	if f.GoFunc.Expr == nil {
		// should probably panic
		return "/* no Go function */"
	}

	var builder strings.Builder

	builder.WriteString("\n//go:noescape\n")
	builder.WriteString(fmt.Sprintf("func %s(", f.GoFunc.Name))
	paramIdx := 0
	f.GoFunc.ForEachParam(func(name, typ string) {
		if paramIdx > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(name)
		builder.WriteByte(' ')
		builder.WriteString(typ)
		paramIdx++
	})
	builder.WriteString(")")
	switch f.GoFunc.NumResults() {
	case 0:
	case 1:
		builder.WriteByte(' ')
		f.GoFunc.ForEachResult(func(_, typ string) {
			builder.WriteString(typ)
		})
	default:
		builder.WriteString(" (")
		resultIdx := 0
		f.GoFunc.ForEachResult(func(name, typ string) {
			if resultIdx > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(name)
			builder.WriteByte(' ')
			builder.WriteString(typ)
			resultIdx++
		})
		builder.WriteString(")")
	}
	builder.WriteByte('\n')
	return builder.String()
}

// ------------------------------------- Code -------------------------------------

// Line represents a line of assembly code
type Line struct {
	Labels       []string `json:"labels,omitempty"`       // Labels for the line
	Binary       []string `json:"binary"`                 // Binary representation of the line
	Assembly     string   `json:"assembly"`               // Assembly representation of the line
	Disassembled string   `json:"disassembled,omitempty"` // Disassembled representation of the line
}

// Compile returns the string representation of a line in PLAN9 assembly
func (line *Line) Compile(arch *config.Arch) string {
	var builder strings.Builder
	for _, label := range line.Labels {
		builder.WriteString(label)
		builder.WriteString(":\n")
	}

	builder.WriteString("\t")

	if len(line.Binary) == 0 && line.Disassembled != "" {
		builder.WriteString(line.Disassembled)
		comment := "<--"
		if line.Disassembled == "NOP" {
			comment = "(skipped)"
		}
		addInstructionComment(&builder, &Line{Assembly: line.Assembly, Disassembled: comment})
		builder.WriteString("\n")
		return builder.String()
	}

	// Special case for arm64, since it's a RISC architecture
	if arch != nil && arch.Name == "arm64" && len(line.Binary) == 4 {
		builder.WriteString(fmt.Sprintf("WORD $0x%v%v%v%v",
			line.Binary[3], line.Binary[2], line.Binary[1], line.Binary[0]))

		addInstructionComment(&builder, line)
		builder.WriteString("\n")
		return builder.String()
	}

	// Dynamic length, assuming WORD = 32-bit
	for pos := 0; pos < len(line.Binary); {
		if pos > 0 {
			builder.WriteString("; ")
		}

		switch {
		case len(line.Binary)-pos >= 8:
			builder.WriteString(fmt.Sprintf("QUAD $0x%v%v%v%v%v%v%v%v",
				line.Binary[pos+7], line.Binary[pos+6], line.Binary[pos+5], line.Binary[pos+4],
				line.Binary[pos+3], line.Binary[pos+2], line.Binary[pos+1], line.Binary[pos]))
			pos += 8
		case len(line.Binary)-pos >= 4:
			builder.WriteString(fmt.Sprintf("LONG $0x%v%v%v%v",
				line.Binary[pos+3], line.Binary[pos+2], line.Binary[pos+1], line.Binary[pos]))
			pos += 4
		case len(line.Binary)-pos >= 2:
			builder.WriteString(fmt.Sprintf("WORD $0x%v%v", line.Binary[pos+1], line.Binary[pos]))
			pos += 2
		case len(line.Binary)-pos >= 1:
			builder.WriteString(fmt.Sprintf("BYTE $0x%v", line.Binary[pos]))
			pos += 1
		}
	}

	addInstructionComment(&builder, line)
	builder.WriteString("\n")
	return builder.String()
}

func addInstructionComment(builder *strings.Builder, line *Line) {
	if line.Disassembled != "" && line.Assembly != "" {
		fmt.Fprintf(builder, "\t// %-36s // %s", line.Disassembled, line.Assembly)
	} else if line.Disassembled != "" {
		builder.WriteString("\t// ")
		builder.WriteString(line.Disassembled)
	} else if line.Assembly != "" {
		fmt.Fprintf(builder, "\t// %-36s // %s", "?", line.Assembly)
	}
}

// Param represents a function parameter
type Param struct {
	Type      string `json:"type"`                // Type of the parameter (C type)
	Name      string `json:"name"`                // Name of the parameter
	IsPointer bool   `json:"isPointer,omitempty"` // Whether the parameter is a pointer
}

func (p *Param) CTypeStr() string {
	if p.IsPointer {
		return fmt.Sprintf("%s*", p.Type)
	}
	return p.Type
}

func (p *Param) CString() string {
	if p.Name != "" {
		return fmt.Sprintf("%s %s", p.CTypeStr(), p.Name)
	}
	return p.CTypeStr()
}

func (p *Param) Size() int {
	// these are for 64-bit systems
	if p.IsPointer {
		return 8
	}

	switch p.Type {
	case "byte", "int8_t", "uint8_t", "bool", "char", "unsignedchar":
		return 1
	case "int16_t", "uint16_t", "short", "unsignedshort":
		return 2
	case "int32_t", "uint32_t", "float", "int", "unsignedint":
		return 4
	case "int64_t", "uint64_t", "double", "long", "unsignedlong", "longlong", "unsignedlonglong":
		// long is 4 bytes on Windows, 8 bytes elsewhere
		return 8
	default:
		return 8
	}
}

func (p *Param) IsFloatingPoint() bool {
	if p.IsPointer {
		return false
	}

	switch p.Type {
	case "float", "double":
		return true
	default:
		return false
	}
}

func (p *Param) EquivalentType() types.Type {
	if p.IsPointer {
		return types.Typ[types.UnsafePointer]
	}

	switch p.Size() {
	case 1:
		return types.Typ[types.Uint8]
	case 2:
		return types.Typ[types.Int16]
	case 4:
		return types.Typ[types.Int32]
	case 8:
		return types.Typ[types.Int64]
	default:
		// panic?
		return types.Typ[types.Uintptr]
	}
}

// String returns the Go string representation of a parameter
func (p *Param) String() string {
	if p.IsPointer {
		return fmt.Sprintf("%s unsafe.Pointer", p.Name)
	}

	switch p.Type {
	case "int16_t":
		return fmt.Sprintf("%s int16", p.Name)
	case "int32_t":
		return fmt.Sprintf("%s int32", p.Name)
	case "int64_t":
		return fmt.Sprintf("%s int64", p.Name)
	case "uint16_t":
		return fmt.Sprintf("%s uint16", p.Name)
	case "uint32_t":
		return fmt.Sprintf("%s uint32", p.Name)
	case "uint64_t":
		return fmt.Sprintf("%s uint64", p.Name)
	case "float":
		return fmt.Sprintf("%s float32", p.Name)
	case "double":
		return fmt.Sprintf("%s float64", p.Name)
	case "unsignedlonglong":
		return fmt.Sprintf("%s uint64", p.Name)
	case "unsignedint":
		return fmt.Sprintf("%s uint32", p.Name)
	case "longlong":
		return fmt.Sprintf("%s int64", p.Name)
	case "int":
		return fmt.Sprintf("%s int32", p.Name)
	case "bool":
		return fmt.Sprintf("%s bool", p.Name)
	default:
		panic(fmt.Sprintf("gocc: unknown type %s", p.Type))
	}
}

// ------------------------------------- Constants -------------------------------------

type Const struct {
	Label string      `json:"label"` // Label of the constant
	Lines []ConstLine `json:"lines"` // Lines of the constant
}

type ConstLine struct {
	Size int    `json:"size"` // Size of the constant
	Data []byte `json:"data"` // Data of the constant
}

func NewConstLine(data []byte) ConstLine {
	return ConstLine{
		Size: len(data),
		Data: data,
	}
}

func NewConstLineFromUint(size int, value uint64) ConstLine {
	data := make([]byte, size)
	switch size {
	case 1:
		data[0] = byte(value)
	case 2:
		binary.LittleEndian.PutUint16(data, uint16(value))
	case 4:
		binary.LittleEndian.PutUint32(data, uint32(value))
	case 8:
		binary.LittleEndian.PutUint64(data, value)
	default:
		panic("invalid const line size")
	}

	return ConstLine{
		Size: size,
		Data: data,
	}
}

func (cl *ConstLine) ValueAsHex() string {
	return hex.EncodeToString(cl.Data)
}

// Compile returns the string representation of a line in PLAN9 assembly
func (c *Const) Compile(arch *config.Arch) string {
	if arch.Name != "amd64" && arch.Name != "arm64" {
		panic("gocc: only amd64 is supported for constants")
	}

	var output strings.Builder
	var totalSize int
	for _, d := range c.Lines {
		// Write the DATA instruction.
		data := d.Data
		for len(data) > 0 {
			switch {
			case len(data) >= 8:
				fmt.Fprintf(&output, "DATA %s<>+%#02x(SB)/8, $%#016x\n", c.Label, totalSize, binary.LittleEndian.Uint64(data))
				data = data[8:]
				totalSize += 8
			case len(data) >= 4:
				fmt.Fprintf(&output, "DATA %s<>+%#02x(SB)/4, $%#08x\n", c.Label, totalSize, binary.LittleEndian.Uint32(data))
				data = data[4:]
				totalSize += 4
			case len(data) >= 2:
				fmt.Fprintf(&output, "DATA %s<>+%#02x(SB)/2, $%#04x\n", c.Label, totalSize, binary.LittleEndian.Uint16(data))
				data = data[2:]
				totalSize += 2
			default:
				fmt.Fprintf(&output, "DATA %s<>+%#02x(SB)/1, $%#02x\n", c.Label, totalSize, data[0])
				data = data[1:]
				totalSize++
			}
		}
	}

	// Write the GLOBL instruction (8=RODATA, 16=NOPTR)
	output.WriteString(fmt.Sprintf("GLOBL %s<>(SB), (RODATA|NOPTR), $%d\n", c.Label, totalSize))
	return output.String()
}

var directiveParseRegex = regexp.MustCompile(`\.((zero|byte|short|hword|word|long|int|quad)\s+(-?\d+)(.*)|(ascii|asciz)\s+("[^"]+"))$`)

// parseConstLine parses a line in the constant section
func parseConstLine(arch *config.Arch, constType string, line string) ConstLine {
	if arch.Name != "amd64" && arch.Name != "arm64" {
		panic("gocc: only amd64 is supported for constants")
	}

	match := directiveParseRegex.FindStringSubmatch(line)
	if len(match) != 7 {
		panic(fmt.Errorf("gocc: unimplemented constant directive: %q", constType))
	}

	switch constType {
	case "ascii", "asciz":
		s, err := strconv.Unquote(match[6])
		if err != nil {
			panic(fmt.Sprintf("gocc: invalid constant value in data: %v", err))
		}
		if match[5] == "asciz" {
			s += "\x00"
		}

		return NewConstLine([]byte(s))
	case "zero":
		value, err := strconv.ParseUint(match[3], 10, 64)
		if err != nil {
			panic(fmt.Sprintf("gocc: invalid constant value in data: %v", err))
		}
		zeroSz := int(value)
		fillVal := byte(0)
		if strings.Contains(match[4], ",") {
			parts := strings.SplitN(match[4], ",", 2)
			val, err := strconv.Atoi(parts[1])
			if err != nil {
				panic(fmt.Sprintf("gocc: invalid constant value in data: %v", err))
			}
			fillVal = byte(val)
		}

		return NewConstLine(bytes.Repeat([]byte{fillVal}, zeroSz))
	}

	value, err := strconv.ParseUint(match[3], 10, 64)
	if err != nil {
		panic(fmt.Sprintf("gocc: invalid constant value in data: %v", err))
	}

	typeSz, ok := constSizes[constType]
	if !ok {
		panic(fmt.Sprintf("gocc: invalid constant type: %s", constType))
	}
	return NewConstLineFromUint(typeSz, value)
}
