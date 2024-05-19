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

package config

import (
	"fmt"
	"regexp"
	"runtime"
)

// Arch represents a context for a specific architecture
type Arch struct {
	Name           string          // Architecture name
	Attribute      *regexp.Regexp  // Parses assembly attributes
	Function       *regexp.Regexp  // Parses assembly function names
	SourceLabel    *regexp.Regexp  // Parses assembly labels
	Code           *regexp.Regexp  // Parses assembly code
	Symbol         *regexp.Regexp  // Parses assembly symbols
	Data           *regexp.Regexp  // Parses assembly data
	Comment        *regexp.Regexp  // Parses assembly comments
	Const          *regexp.Regexp  // Parses assembly constants
	Label          *regexp.Regexp  // Finds label inside an instruction
	DataLoad       *regexp.Regexp  // Finds instructions with data loads
	JumpInstr      *regexp.Regexp  // Parses assembly jump instructions
	Registers      []string        // Registers to use
	FloatRegisters []string        // Floating point registers
	RetRegister    string          // Register for return values
	BuildTags      string          // Golang build tags
	CommentCh      string          // Assembly comment character
	MovInstr       map[int8]string // Instruction to use to load/store for various sizes
	MovFPInstr     map[int8]string // Instruction to use to load/store floating point values
	LoadInstr      map[int8]string // Instruction for loading registers (optional)
	Disassembler   []string        // Disassembler to use and flags
	ClangFlags     []string        // Flags for clang
}

// For returns a configuration for a given architecture
func For(arch string) (*Arch, error) {
	switch arch {
	case "amd64":
		return AMD64(), nil
	case "arm64":
		return ARM64(), nil
	case "apple":
		return Apple(), nil
	case "neon":
		return Neon(), nil
	case "sve":
		return SVE(), nil
	case "sve2":
		return SVE2(), nil
	case "avx2":
		return Avx2(), nil
	case "avx512":
		return Avx512(), nil
	default:
		return nil, fmt.Errorf("unsupported architecture: %s", arch)
	}
}

// ------------------------------------- AMD64 -------------------------------------

// AMD64 returns a configuration for AMD64 architecture
func AMD64() *Arch {
	arch := &Arch{
		Name:           "amd64",
		Attribute:      regexp.MustCompile(`^\s+\..+$`),
		Function:       regexp.MustCompile(`^\w+:.*$`),
		SourceLabel:    regexp.MustCompile(`^\.[A-Z0-9]+_\d+:.*$`),
		Code:           regexp.MustCompile(`^\s+\w+.+$`),
		Symbol:         regexp.MustCompile(`^\w+\s+<\w+>:$`),
		Data:           regexp.MustCompile(`^\w+:\s+\w+\s+.+$`),
		Comment:        regexp.MustCompile(`^\s*#.*$`),
		Const:          regexp.MustCompile(`^\s+\.((byte|short|long|int|quad)\s+(-?\d+).+|(ascii|asciz)\s+("[^"]+"))$`),
		Label:          regexp.MustCompile(`[A-Z0-9]+_\d+`),
		DataLoad:       regexp.MustCompile(`^(?P<instr>\w+)\s+[^;]+?(?P<register>\b[RXY]\d+);.*?\.(?P<var>\w+)\b`),
		JumpInstr:      regexp.MustCompile(`^(?P<instr>J\w+)[^;]+;.*?[.](?P<label>\w+)$`),
		Registers:      []string{"DI", "SI", "DX", "CX", "R8", "R9"},
		FloatRegisters: []string{"X0", "X1", "X2", "X3", "X4", "X5", "X6", "X7"},
		RetRegister:    "AX",
		BuildTags:      "//go:build !noasm && amd64",
		CommentCh:      "#",
		MovInstr:       map[int8]string{1: "MOVB", 2: "MOVW", 4: "MOVL", 8: "MOVQ"},
		MovFPInstr:     map[int8]string{4: "MOVSS", 8: "MOVSD"},
		LoadInstr:      map[int8]string{1: "MOVBQZX", 2: "MOVWQZX", 4: "MOVLQZX", 8: "MOVQ"},
		ClangFlags:     []string{"--target=x86_64-linux-gnu", "-masm=intel"},
		//Disassembler: []string{"--insn-width", "16"},
	}

	if runtime.GOOS == "darwin" {
		arch.ClangFlags = append(arch.ClangFlags, "-arch", "x86_64", "--sysroot=/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX.sdk/")
		arch.Disassembler = []string{}
		return arch
	}

	return arch
}

// Avx2 returns a configuration for AMD64 architecture with AVX2 support
func Avx2() *Arch {
	arch := AMD64()
	arch.ClangFlags = append(arch.ClangFlags, "-mavx2", "-mfma")
	return arch
}

// Avx512 returns a configuration for AMD64 architecture with AVX512 support
func Avx512() *Arch {
	arch := AMD64()
	arch.ClangFlags = append(arch.ClangFlags, "-mavx", "-mfma", "-mavx512f", "-mavx512dq")
	return arch
}

// ------------------------------------- Linux ARM64 -------------------------------------

// ARM64 returns a configuration for ARM64 architecture
func ARM64() *Arch {
	arch := &Arch{
		Name:           "arm64",
		Attribute:      regexp.MustCompile(`^\s+\..+$`),
		Function:       regexp.MustCompile(`^\w+:.*$`),
		SourceLabel:    regexp.MustCompile(`^\.[A-Z0-9]+_\d+:.*$`),
		Code:           regexp.MustCompile(`^\s+\w+.+$`),
		Symbol:         regexp.MustCompile(`^\w+\s+<\w+>:$`),
		Data:           regexp.MustCompile(`^\w+:\s+\w+\s+.+$`),
		Comment:        regexp.MustCompile(`^\s*//.*$`),
		Const:          regexp.MustCompile(`^\s+\.((byte|short|long|int|quad)\s+(-?\d+).+|(ascii|asciz)\s+("[^"]+"))$`),
		Label:          regexp.MustCompile(`[A-Z0-9]+_\d+`),
		DataLoad:       regexp.MustCompile(`^ADRP\s+[^;]+?(?P<register>\bR\d+);.*?[.]?\b(?P<var>[A-Za-z_][A-Za-z0-9_]+)$`),
		JumpInstr:      regexp.MustCompile(`^(?P<instr>.*?)([-]?\d*[(]PC[)]);.*?(?P<label>[Ll_][a-zA-Z0-9_]+)$`),
		Registers:      []string{"R0", "R1", "R2", "R3", "R4", "R5", "R6", "R7"},
		FloatRegisters: []string{"F0", "F1", "F2", "F3", "F4", "F5", "F6", "F7"},
		RetRegister:    "R0",
		BuildTags:      "//go:build !noasm && arm64",
		CommentCh:      "//",
		MovInstr:       map[int8]string{1: "MOVB", 2: "MOVH", 4: "MOVW", 8: "MOVD"},
		MovFPInstr:     map[int8]string{4: "FMOVS", 8: "FMOVD"},
		ClangFlags:     []string{"--target=aarch64-linux-gnu", "-ffixed-x18", "-ffixed-x27", "-ffixed-x28"},
	}

	return arch
}

// Neon returns a configuration for ARM64 architecture with NEON support
func Neon() *Arch {
	arch := ARM64()
	// arm64 requires NEON support, no need to specify it explicitly
	return arch
}

func SVE() *Arch {
	arch := ARM64()
	arch.ClangFlags = append(arch.ClangFlags, "-march=armv8.2-a+sve", "-mfpu=sve")
	return arch
}

func SVE2() *Arch {
	arch := ARM64()
	arch.ClangFlags = append(arch.ClangFlags, "-march=armv8.5-a+sve2", "-mfpu=sve2")
	return arch
}

// ------------------------------------- Apple ARM64 -------------------------------------

// Apple returns a configuration for ARM64 architecture. On my M1 mac, supported features are:
// AESARM, ASIMD, ASIMDDP, ASIMDHP, ASIMDRDM, ATOMICS, CRC32, DCPOP, FCMA, FP, FPHP, GPA, JSCVT, LRCPC, PMULL, SHA1, SHA2, SHA3, SHA512
func Apple() *Arch {
	arch := ARM64()

	arch.SourceLabel = regexp.MustCompile(`^[Ll][a-zA-Z0-9]+(?:_\d+)?:.*$`)
	arch.Comment = regexp.MustCompile(`^\s*;.*$`)
	arch.CommentCh = ";"
	arch.Label = regexp.MustCompile(`[Ll_][a-zA-Z0-9_]+`)
	arch.DataLoad = regexp.MustCompile(`(?P<register>\bR\d+);.*?\b(?P<var>\w+)@PAGE\b`)
	arch.JumpInstr = regexp.MustCompile(`^(?P<instr>.*?)([-]?\d*[(]PC[)]);.*?(?P<label>[Ll_][a-zA-Z0-9_]+)$`)

	if runtime.GOOS != "darwin" {
		// Cross-compiling
		arch.ClangFlags = []string{"--target=aarch64-apple-darwin", "--sysroot=/usr/osxcross/SDK/MacOSX11.3.sdk/"}
		return arch
	}

	arch.ClangFlags = []string{}

	return arch
}

// ------------------------------------- Toolchain -------------------------------------

// FindClang resolves clang compiler to use.
func FindClang() (string, error) {
	return find([]string{
		"clang-19", "clang-18",
		"clang-17", "clang-16",
		"clang-15", "clang-14",
		"clang",
	})
}

// FindClangObjdump resolves clang disassembler to use.
func FindClangObjdump() (string, error) {
	return find([]string{
		"llvm-objdump-19", "llvm-objdump-18",
		"llvm-objdump-17", "llvm-objdump-16",
		"llvm-objdump-15", "llvm-objdump-14",
		"llvm-objdump", "objdump",
	})
}

// FindGoObjdump resolves go toolchain to use.
func FindGoObjdump() (string, error) {
	return find([]string{
		"golang", "go",
	})
}
