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
	"errors"
	"fmt"
	"go/ast"
	"go/types"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

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
	"xword": 8,
}

type GoFunction struct {
	Name string
	Expr *ast.FuncType
}

// GoVariant is a further Go signature of a function, declared with a
// gocc-variant directive: the function's own with some of its []byte
// parameters taken as strings.
//
// A string is one word shorter than a slice, so the two signatures have
// different frames and, under the stack-based ABI of assembly functions,
// keep their result in different places. The function is therefore given the
// frame of the []byte form, and the entry point of a variant is declared
// with a blank int after each such string. That makes the frames identical
// and the entry point a single jump. A Go wrapper with the signature the
// user asked for supplies the blank word.
//
// The C function takes a pointer and a length for such a parameter. The
// third word, the capacity of the slice or the blank int, is in the frame
// but is not passed on (see Param.FrameGap).
type GoVariant struct {
	GoFunction
	// Padded reports, for each parameter, whether it is a string standing in
	// for a []byte of the function.
	Padded []bool
}

// InternalName is the name of the padded assembly entry point.
func (v *GoVariant) InternalName() string {
	first, size := utf8.DecodeRuneInString(v.Name)
	return string(unicode.ToLower(first)) + v.Name[size:] + "Internal"
}

// ------------------------------------- Function -------------------------------------

type Function struct {
	Name            string  `json:"name"`
	Internal        bool    `json:"internal,omitempty"`
	PreserveCABI    bool    `json:"preserveCABI,omitempty"`
	SourcePath      string  `json:"sourcePath"`
	Position        int     `json:"position"`
	Params          []Param `json:"params"`
	Consts          []Const `json:"consts,omitempty"`
	Lines           []Line  `json:"lines"`
	LocalsSize      int     `json:"localsSize,omitempty"`
	HiddenStackSize int     `json:"hiddenStackSize,omitempty"`
	// StackAlign is the stack alignment the C code depends on, when its
	// instructions require more than the 8 bytes Go guarantees on AMD64. A
	// non-zero value makes the generated prologue realign SP within the frame
	// and the epilogue restore it; zero leaves SP untouched.
	StackAlign int    `json:"stackAlign,omitempty"`
	Ret        *Param `json:"return,omitempty"`
	// GoFunc is the Go signature the assembly is declared with. For a
	// function with variants it is the one that takes []byte wherever any of
	// them does, whichever directive it came from.
	GoFunc   GoFunction
	Variants []GoVariant
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

		offset += size + param.FrameGap
	}

	// we calc the return size separately, but still account for it
	if f.Ret != nil {
		maxAlign := int(sizes.Alignof(types.Typ[types.Int64]))
		offset += -offset & (maxAlign - 1)
	}

	return offset, offsets
}

// writeSignature writes "func name(params) results". A parameter flagged in
// padded is followed by a blank int.
func (f *GoFunction) writeSignature(builder *strings.Builder, name string, padded []bool) {
	fmt.Fprintf(builder, "func %s(", name)
	paramIdx := 0
	f.ForEachParam(func(name, typ string) {
		if paramIdx > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(name)
		builder.WriteByte(' ')
		builder.WriteString(typ)
		if paramIdx < len(padded) && padded[paramIdx] {
			builder.WriteString(", _ int")
		}
		paramIdx++
	})
	builder.WriteString(")")
	switch f.NumResults() {
	case 0:
	case 1:
		builder.WriteByte(' ')
		f.ForEachResult(func(_, typ string) {
			builder.WriteString(typ)
		})
	default:
		builder.WriteString(" (")
		resultIdx := 0
		f.ForEachResult(func(name, typ string) {
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
}

// writeStub writes the Go wrapper of a variant of target and the declaration
// of the assembly entry point it calls.
func (v *GoVariant) writeStub(builder *strings.Builder, target string) {
	internal := v.InternalName()

	fmt.Fprintf(builder, "\n// %s is %s with string arguments.\n", v.Name, target)
	v.writeSignature(builder, v.Name, nil)
	builder.WriteString(" {\n\t")
	if v.NumResults() > 0 {
		builder.WriteString("return ")
	}
	builder.WriteString(internal)
	builder.WriteByte('(')
	paramIdx := 0
	v.ForEachParam(func(name, _ string) {
		if paramIdx > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(name)
		if v.Padded[paramIdx] {
			builder.WriteString(", 0")
		}
		paramIdx++
	})
	builder.WriteString(")\n}\n")

	fmt.Fprintf(builder, "\n// %s jumps to %s.\n", internal, target)
	builder.WriteString("// The int after a string takes the place of the capacity of the []byte it\n")
	builder.WriteString("// stands for, which gives both functions the same frame. It is not read.\n//\n//go:noescape\n")
	v.writeSignature(builder, internal, v.Padded)
	builder.WriteByte('\n')
}

// String returns the Go stub of the function and of its variants.
func (f *Function) String() string {
	if f.GoFunc.Expr == nil {
		// should probably panic
		return "/* no Go function */"
	}

	var builder strings.Builder

	builder.WriteString("\n//go:noescape\n")
	f.GoFunc.writeSignature(&builder, f.GoFunc.Name, nil)
	builder.WriteByte('\n')
	for i := range f.Variants {
		f.Variants[i].writeStub(&builder, f.GoFunc.Name)
	}
	return builder.String()
}

// ------------------------------------- Code -------------------------------------

// Line represents a line of assembly code
type Line struct {
	Labels       []string `json:"labels,omitempty"`       // Labels for the line
	Binary       []string `json:"binary"`                 // Binary representation of the line
	Assembly     string   `json:"assembly"`               // Assembly representation of the line
	Disassembled string   `json:"disassembled,omitempty"` // Disassembled representation of the line
	Comment      string   `json:"comment,omitempty"`      // Optional generated-code comment replacing Assembly
	Heading      string   `json:"heading,omitempty"`      // Comment lines to put above the line and its labels
}

// Compile returns the string representation of a line in PLAN9 assembly
func (line *Line) Compile(arch *config.Arch) string {
	var builder strings.Builder
	if line.Heading != "" {
		builder.WriteString("\n")
		for _, text := range strings.Split(line.Heading, "\n") {
			builder.WriteString("// " + text + "\n")
		}
	}
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
		addInstructionComment(&builder, &Line{Assembly: line.Assembly, Disassembled: comment, Comment: line.Comment})
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
	comment := line.Assembly
	if line.Comment != "" {
		comment = line.Comment
	}
	if line.Disassembled != "" && comment != "" {
		fmt.Fprintf(builder, "\t// %-36s // %s", line.Disassembled, comment)
	} else if line.Disassembled != "" {
		builder.WriteString("\t// ")
		builder.WriteString(line.Disassembled)
	} else if comment != "" {
		fmt.Fprintf(builder, "\t// %-36s // %s", "?", comment)
	}
}

// Param represents a function parameter
type Param struct {
	Type      string `json:"type"`                // Type of the parameter (C type)
	Name      string `json:"name"`                // Name of the parameter
	IsPointer bool   `json:"isPointer,omitempty"` // Whether the parameter is a pointer
	// GoName is the name of the argument slot in the Go frame as go vet's
	// asmdecl expects it: the Go parameter name, with _len/_cap suffixes for
	// the length and capacity words of strings and slices. Empty for
	// internal helpers, which have no Go signature.
	GoName string `json:"goName,omitempty"`
	// FrameGap is the number of bytes of the Go argument frame that follow
	// this parameter and are not passed to the C function: the capacity of
	// a slice whose C counterpart is a pointer and a length.
	FrameGap int `json:"frameGap,omitempty"`
}

// FrameName returns the name to use in a name+offset(FP) reference.
func (p *Param) FrameName() string {
	if p.GoName != "" {
		return p.GoName
	}
	return p.Name
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

var directiveParseRegex = regexp.MustCompile(`\.((zero|byte|short|hword|word|long|int|quad|xword)\s+(-?(?:0[xX][0-9a-fA-F]+|\d+))(.*)|(ascii|asciz)\s+("(?:[\\].|[^\\"])+"))$`)

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

	value, err := strconv.ParseUint(match[3], 0, 64)
	if errors.Is(err, strconv.ErrSyntax) {
		val, parseErr := strconv.ParseInt(match[3], 0, 64)
		if parseErr == nil {
			value = uint64(val)
			err = nil
		}
	}
	if err != nil {
		panic(fmt.Sprintf("gocc: invalid constant value in data: %v", err))
	}

	typeSz, ok := constSizes[constType]
	if !ok {
		panic(fmt.Sprintf("gocc: invalid constant type: %s", constType))
	}
	return NewConstLineFromUint(typeSz, value)
}
