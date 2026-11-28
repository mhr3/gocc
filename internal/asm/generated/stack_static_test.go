package generated_test

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/arch/arm64/arm64asm"
	"golang.org/x/arch/x86/x86asm"
)

// The run-time tests only reach what the machine they run on can execute, and
// only fail when a broken instruction is hit with the stack placed badly. The
// checks here read the machine code of every generated file instead: each is
// assembled for its own architecture with the Go assembler, whatever the host,
// and every instruction is decoded again. That covers the text gocc wrote and
// the encodings it kept alike, and AVX-512 code on machines without it.

// asmFunc is one TEXT block of a generated file.
type asmFunc struct {
	name     string
	internal bool // a helper using the C calling convention: TEXT name<>(SB)
	line     int  // of the TEXT directive
	frame    int  // the frame the directive declares
	// realign is the alignment the function gives SP on entry, 0 if it
	// leaves the stack pointer alone.
	realign int
	insts   []asmInst
}

type asmInst struct {
	line   int    // in the source file
	source string // that line
	code   []byte
	symbol string // the symbol the instruction refers to through a relocation
	offset int    // into that symbol
	call   bool   // the relocation is a call or a jump
}

type asmFile struct {
	path   string
	arch   string
	funcs  []*asmFunc
	consts map[string]int // sizes of the data symbols
	source []string
}

var (
	asmTextDirective = regexp.MustCompile(`^TEXT\s+(?:·(\w+)|(\w+)<>)\(SB\),\s*[^,]+,\s*\$(\d+)`)
	asmGloblSize     = regexp.MustCompile(`^GLOBL\s+(\w+)<>\(SB\),.*\$(\d+)$`)
	asmRealign       = regexp.MustCompile(`^ANDQ\s+\$-(\d+), SP$`)
	asmObjdumpLine   = regexp.MustCompile(`^\s+\S+\.s:(\d+)\s+0x[0-9a-f]+\s+([0-9a-f]+)\s`)
	asmRelocation    = regexp.MustCompile(`\[\d+:\d+\]R_(\w+)(?::(\S+))?`)
	asmStaticSymbol  = regexp.MustCompile(`^(\w+)<\d+>(?:\+(\d+))?$`)
)

// loadAssembly assembles a generated file and returns its functions with
// their machine code.
func loadAssembly(t *testing.T, path string) *asmFile {
	t.Helper()

	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Skip("the go tool is needed to assemble the generated files")
	}
	goroot, err := exec.Command(goTool, "env", "GOROOT").Output()
	require.NoError(t, err)

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	file := &asmFile{path: path, consts: map[string]int{}, source: strings.Split(string(raw), "\n")}
	switch {
	case strings.Contains(file.source[0], "amd64"):
		file.arch = "amd64"
	case strings.Contains(file.source[0], "arm64"):
		file.arch = "arm64"
	default:
		t.Fatalf("%s: no architecture in %q", path, file.source[0])
	}

	for i, text := range file.source {
		if match := asmTextDirective.FindStringSubmatch(text); match != nil {
			fn := &asmFunc{name: match[1] + match[2], internal: match[2] != "", line: i + 1}
			fn.frame, _ = strconv.Atoi(match[3])
			file.funcs = append(file.funcs, fn)
		} else if match := asmGloblSize.FindStringSubmatch(text); match != nil {
			file.consts[match[1]], _ = strconv.Atoi(match[2])
		} else if match := asmRealign.FindStringSubmatch(strings.TrimSpace(text)); match != nil && len(file.funcs) > 0 {
			fn := file.funcs[len(file.funcs)-1]
			fn.realign, _ = strconv.Atoi(match[1])
		}
	}

	object := filepath.Join(t.TempDir(), "out.o")
	assemble := exec.Command(goTool, "tool", "asm", "-p", "generated",
		"-I", filepath.Join(strings.TrimSpace(string(goroot)), "pkg", "include"), "-o", object, path)
	assemble.Env = append(os.Environ(), "GOARCH="+file.arch, "GOOS=linux")
	out, err := assemble.CombinedOutput()
	require.NoError(t, err, "%s", out)
	dump := exec.Command(goTool, "tool", "objdump", object)
	dump.Env = assemble.Env
	out, err = dump.Output()
	require.NoError(t, err)

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	scanner.Buffer(make([]byte, 0, 1<<16), 1<<20)
	for scanner.Scan() {
		match := asmObjdumpLine.FindStringSubmatch(scanner.Text())
		if match == nil {
			continue
		}
		inst := asmInst{}
		inst.line, _ = strconv.Atoi(match[1])
		inst.source = strings.TrimSpace(file.source[inst.line-1])
		inst.code, err = hex.DecodeString(match[2])
		require.NoError(t, err)
		if file.arch == "arm64" {
			// objdump prints the instruction word, not its bytes
			inst.code[0], inst.code[1], inst.code[2], inst.code[3] = inst.code[3], inst.code[2], inst.code[1], inst.code[0]
		}
		if reloc := asmRelocation.FindStringSubmatch(scanner.Text()); reloc != nil {
			inst.call = strings.HasPrefix(reloc[1], "CALL")
			if symbol := asmStaticSymbol.FindStringSubmatch(reloc[2]); symbol != nil {
				inst.symbol = symbol[1]
				inst.offset, _ = strconv.Atoi(symbol[2])
			} else {
				inst.symbol = reloc[1] // not one of the file's own
			}
		}

		// the function is the last TEXT directive at or above the line
		var fn *asmFunc
		for _, candidate := range file.funcs {
			if candidate.line <= inst.line {
				fn = candidate
			}
		}
		require.NotNil(t, fn, "%s:%d is outside any function", path, inst.line)
		fn.insts = append(fn.insts, inst)
	}
	require.NoError(t, scanner.Err())
	return file
}

// frameCode reports whether the instruction belongs to the function's body,
// as opposed to the Go assembler's prologue and epilogue and the loads and
// stores of arguments, which address the caller's frame.
func (inst *asmInst) body(fn *asmFunc) bool {
	return inst.line != fn.line && !strings.HasPrefix(inst.source, "RET") && !strings.Contains(inst.source, "(FP)")
}

// asmReach is a way a function is entered: from which Go-visible function,
// and below how many return addresses.
type asmReach struct {
	root  *asmFunc
	depth int
}

// reaches returns, for every function, the ways it is entered. A Go-visible
// function is entered as itself; a helper from every Go-visible function
// that calls it, directly or not.
func (file *asmFile) reaches() map[*asmFunc][]asmReach {
	byName := map[string]*asmFunc{}
	for _, fn := range file.funcs {
		byName[fn.name] = fn
	}

	reaches := map[*asmFunc][]asmReach{}
	for _, root := range file.funcs {
		if root.internal {
			continue
		}
		type visited struct {
			fn    *asmFunc
			depth int
		}
		seen := map[visited]bool{}
		var visit func(fn *asmFunc, depth int)
		visit = func(fn *asmFunc, depth int) {
			if seen[visited{fn, depth}] {
				return
			}
			seen[visited{fn, depth}] = true
			reach := asmReach{root, depth}
			reaches[fn] = append(reaches[fn], reach)

			for _, inst := range fn.insts {
				callee := byName[inst.symbol]
				if !inst.call || callee == nil || !callee.internal {
					continue
				}
				// CALL pushes a return address on amd64, a jump and BL do not
				pushed := 0
				if file.arch == "amd64" && inst.code[0] == 0xe8 {
					pushed = 1
				}
				visit(callee, depth+pushed)
			}
		}
		visit(root, 0)
	}
	return reaches
}

// asmCoverage counts the cases a file exercises, so that a fixture which
// stops forcing its case does not go unnoticed.
type asmCoverage struct {
	stackVectorMoves   map[int]int // by operand size
	legacyStackOperand int         // instructions without VEX reading 128 bits off the stack
	realigned          int         // functions realigning SP
	vectorConstants    int         // vector operands read from a data symbol
}

var (
	amd64AlignedMoves = map[string]bool{
		"MOVAPS": true, "MOVAPD": true, "MOVDQA": true, "MOVNTDQ": true, "MOVNTDQA": true, "MOVNTPS": true, "MOVNTPD": true,
		"VMOVAPS": true, "VMOVAPD": true, "VMOVDQA": true, "VMOVDQA32": true, "VMOVDQA64": true,
		"VMOVNTDQ": true, "VMOVNTDQA": true, "VMOVNTPS": true, "VMOVNTPD": true,
	}
	amd64UnalignedLegacy = map[string]bool{
		"MOVDQU": true, "MOVUPS": true, "MOVUPD": true, "LDDQU": true,
		"PCMPESTRI": true, "PCMPESTRM": true, "PCMPISTRI": true, "PCMPISTRM": true,
	}
)

// amd64Alignment returns the alignment the instruction needs its memory
// operand to have, 0 when it takes it at any address.
func amd64Alignment(inst x86asm.Inst) int {
	name := inst.Op.String()
	switch {
	case amd64AlignedMoves[name]:
		return inst.MemBytes
	case strings.HasPrefix(name, "V") || inst.MemBytes != 16 || amd64UnalignedLegacy[name]:
		return 0
	}
	// without VEX, a 128-bit operand of a vector instruction must be aligned
	for _, arg := range inst.Args {
		if reg, ok := arg.(x86asm.Reg); ok && reg >= x86asm.X0 && reg <= x86asm.X15 {
			return 16
		}
	}
	return 0
}

func amd64Vector(inst x86asm.Inst) bool {
	for _, arg := range inst.Args {
		if reg, ok := arg.(x86asm.Reg); ok && reg >= x86asm.X0 {
			return strings.ContainsAny(reg.String()[:1], "XYZ")
		}
	}
	return false
}

// goDataAlignment is the alignment the Go linker gives a data symbol of the
// given size: the largest power of two that fits, 32 at most.
func goDataAlignment(size int) int {
	align := 32
	for align > size && align > 1 {
		align /= 2
	}
	return align
}

// checkAmd64 returns what is wrong with the stack and constant operands of
// an amd64 file.
func checkAmd64(file *asmFile) (problems []string, coverage asmCoverage) {
	coverage.stackVectorMoves = map[int]int{}
	reaches := file.reaches()

	for _, fn := range file.funcs {
		if fn.realign > 0 {
			coverage.realigned++
		}
		for _, inst := range fn.insts {
			report := func(format string, args ...any) {
				problems = append(problems, fmt.Sprintf("%s:%d: %s: %s  [%s]", filepath.Base(file.path), inst.line, fn.name, fmt.Sprintf(format, args...), inst.source))
			}

			decoded, err := x86asm.Decode(inst.code, 64)
			if err != nil {
				continue
			}
			if inst.body(fn) && (decoded.Op == x86asm.PUSH || decoded.Op == x86asm.POP) {
				report("moves the stack pointer inside a fixed frame")
			}

			var mem *x86asm.Mem
			for _, arg := range decoded.Args {
				if m, ok := arg.(x86asm.Mem); ok {
					mem = &m
				}
			}
			if mem == nil {
				continue
			}
			need := amd64Alignment(decoded)

			switch {
			case inst.symbol != "" && !inst.call:
				size, known := file.consts[inst.symbol]
				if !known {
					break // thread-local storage in the prologue
				}
				if amd64Vector(decoded) {
					coverage.vectorConstants++
				}
				if need > goDataAlignment(size) || (need > 0 && inst.offset%need != 0) {
					report("needs %s+%d aligned to %d bytes; the linker aligns its %d bytes to %d", inst.symbol, inst.offset, need, size, goDataAlignment(size))
				}

			case mem.Base == x86asm.RIP || (mem.Base == 0 && mem.Index == 0 && mem.Segment == 0):
				// The displacement of a reference to data is filled in by a
				// relocation. Without one this reads whatever sits at that
				// distance, or at that address.
				report("has a memory operand that refers to no symbol")

			case mem.Base == x86asm.RSP:
				if decoded.MemBytes >= 16 && amd64Vector(decoded) {
					if strings.Contains(decoded.Op.String(), "MOV") {
						coverage.stackVectorMoves[decoded.MemBytes]++
					} else if need > 0 {
						coverage.legacyStackOperand++
					}
				}

				for _, reach := range reaches[fn] {
					// the operand, from the stack pointer the Go-visible
					// function ends up with
					at := int(mem.Disp) - 8*reach.depth
					if need > 0 {
						if reach.root.realign < need {
							report("needs its stack operand aligned to %d bytes, but %s aligns the stack to %d", need, reach.root.name, max(reach.root.realign, 8))
						} else if mem.Index == 0 && at%need != 0 {
							report("needs its stack operand aligned to %d bytes, but it is %d bytes above the aligned stack pointer of %s", need, at, reach.root.name)
						}
					}

					// Realigning moves the stack pointer up by as much as the
					// alignment, which the frame was enlarged by.
					limit := reach.root.frame - reach.root.realign
					if inst.body(fn) && mem.Index == 0 && (at < 0 || at+decoded.MemBytes > limit) {
						report("touches %d bytes at %d of the frame of %s, which has %d", decoded.MemBytes, at, reach.root.name, limit)
					}
				}
			}
		}
	}
	return problems, coverage
}

// arm64StackOperand returns the offset and size of the instruction's operand
// on the stack, and whether it moves the stack pointer.
func arm64StackOperand(inst arm64asm.Inst) (offset, size int, moves, ok bool) {
	regSize := func(arg arm64asm.Arg) int {
		switch name := fmt.Sprint(arg); name[0] {
		case 'X':
			return 8
		case 'W', 'S':
			return 4
		case 'Q':
			return 16
		case 'D':
			return 8
		case 'H':
			return 2
		default:
			return 1
		}
	}

	if dest, isSP := inst.Args[0].(arm64asm.RegSP); isSP && dest == arm64asm.RegSP(arm64asm.SP) {
		switch inst.Op {
		case arm64asm.ADD, arm64asm.SUB, arm64asm.MOV, arm64asm.AND:
			return 0, 0, true, true
		}
	}
	if inst.Op == arm64asm.ADD {
		// the address of something on the stack
		if base, isSP := inst.Args[1].(arm64asm.RegSP); isSP && base == arm64asm.RegSP(arm64asm.SP) {
			if imm, isImm := inst.Args[2].(arm64asm.ImmShift); isImm {
				value, err := strconv.ParseInt(strings.TrimPrefix(strings.SplitN(imm.String(), ",", 2)[0], "#"), 0, 64)
				if err == nil && !strings.Contains(imm.String(), "LSL") {
					return int(value), 0, false, true
				}
			}
		}
		return 0, 0, false, false
	}

	for i, arg := range inst.Args {
		mem, isMem := arg.(arm64asm.MemImmediate)
		if !isMem || mem.Base != arm64asm.RegSP(arm64asm.SP) {
			continue
		}
		if mem.Mode != arm64asm.AddrOffset {
			return 0, 0, true, true
		}
		if text := strings.Trim(mem.String(), "[]"); strings.Contains(text, "#") {
			value, _ := strconv.ParseInt(text[strings.Index(text, "#")+1:], 0, 64)
			offset = int(value)
		}

		switch inst.Op {
		case arm64asm.STRB, arm64asm.LDRB, arm64asm.LDRSB, arm64asm.STURB, arm64asm.LDURB:
			size = 1
		case arm64asm.STRH, arm64asm.LDRH, arm64asm.LDRSH, arm64asm.STURH, arm64asm.LDURH:
			size = 2
		case arm64asm.LDRSW, arm64asm.LDURSW:
			size = 4
		case arm64asm.STP, arm64asm.LDP:
			size = 2 * regSize(inst.Args[0])
		case arm64asm.STR, arm64asm.LDR, arm64asm.STUR, arm64asm.LDUR:
			size = regSize(inst.Args[0])
		default:
			return 0, 0, false, false
		}
		_ = i
		return offset, size, false, true
	}
	return 0, 0, false, false
}

// checkArm64 returns what is wrong with the stack operands of an arm64 file.
func checkArm64(file *asmFile) (problems []string) {
	reaches := file.reaches()

	for _, fn := range file.funcs {
		for _, inst := range fn.insts {
			if !inst.body(fn) {
				continue
			}
			decoded, err := arm64asm.Decode(inst.code)
			if err != nil {
				continue
			}
			offset, size, moves, ok := arm64StackOperand(decoded)
			if !ok {
				continue
			}
			report := func(format string, args ...any) {
				problems = append(problems, fmt.Sprintf("%s:%d: %s: %s  [%s]", filepath.Base(file.path), inst.line, fn.name, fmt.Sprintf(format, args...), inst.source))
			}
			if moves {
				report("moves the stack pointer inside a fixed frame")
				continue
			}
			for _, reach := range reaches[fn] {
				// The Go assembler keeps the return address at 0(RSP), and the
				// top 8 bytes of what it allocates hold the frame pointer that
				// the calling Go function saved. Neither belongs to the C code.
				if offset < 8 || offset+size > reach.root.frame+8 {
					report("touches %d bytes at %d(RSP); the frame of %s spans 8 to %d", size, offset, reach.root.name, reach.root.frame+8)
				}
			}
		}
	}
	return problems
}

// generatedAssembly returns the generated files of this directory.
func generatedAssembly(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob("test_*.s")
	require.NoError(t, err)
	require.NotEmpty(t, files)
	return files
}

func TestStackOperandsOfGeneratedAssembly(t *testing.T) {
	for _, path := range generatedAssembly(t) {
		t.Run(path, func(t *testing.T) {
			file := loadAssembly(t, path)

			var problems []string
			if file.arch == "arm64" {
				problems = checkArm64(file)
			} else {
				problems, _ = checkAmd64(file)
			}
			for _, problem := range problems {
				t.Error(problem)
			}
		})
	}
}

// The fixtures only guard the cases they still force. This fails when a
// compiler stops spilling, reading an operand off the stack or loading a
// constant where test_stack.c is written to make it do so.
func TestStackFixturesForceTheirCases(t *testing.T) {
	for _, tc := range []struct {
		file    string
		vector  int // bytes
		realign bool
		legacy  bool
	}{
		{file: "test_stack_amd64_sse2.s", vector: 16, realign: true, legacy: true},
		{file: "test_stack_amd64_sse4.s", vector: 16, realign: true, legacy: true},
		{file: "test_stack_amd64_avx2.s", vector: 32, realign: true},
		{file: "test_stack_amd64_avx512.s", vector: 64, realign: true},
	} {
		t.Run(tc.file, func(t *testing.T) {
			_, coverage := checkAmd64(loadAssembly(t, tc.file))

			assert.NotZero(t, coverage.stackVectorMoves[tc.vector], "no %d-byte vector is moved to or from the stack", tc.vector)
			assert.NotZero(t, coverage.vectorConstants, "no vector constant is read from a data symbol")
			if tc.realign {
				assert.NotZero(t, coverage.realigned, "no function has to realign the stack")
			}
			if tc.legacy {
				assert.NotZero(t, coverage.legacyStackOperand, "no instruction without VEX reads its operand off the stack")
			}
		})
	}
}

// The checks above are only worth something if they fail on broken code.
// Each case here is the kind of code gocc must not write.
func TestStackChecksRejectBrokenAssembly(t *testing.T) {
	const amd64Header = "//go:build amd64\n#include \"textflag.h\"\n"
	const arm64Header = "//go:build arm64\n#include \"textflag.h\"\n"
	// what gocc puts around a function that realigns: 32 bytes of locals,
	// 16 to slide SP up by and a slot for the SP to return to
	const realigned = "TEXT ·root(SB), 0, $56-0\n\tMOVQ SP, R11\n\tADDQ $16, SP\n\tANDQ $-16, SP\n\tMOVQ R11, 32(SP)\n"
	const restore = "\tMOVQ 32(SP), SP\n\tRET\n"

	for _, tc := range []struct {
		name, source string
		problems     []string // a part of each expected problem, none for valid code
	}{
		{
			name:   "realigned frame",
			source: amd64Header + realigned + "\tMOVO X0, 16(SP)\n\tPXOR 0(SP), X1\n\tCALL helper<>(SB)\n" + restore + "TEXT helper<>(SB), NOSPLIT|NOFRAME, $0-0\n\tPXOR 24(SP), X1\n\tRET\n",
		},
		{
			name:     "aligned move to an unaligned frame",
			source:   amd64Header + "TEXT ·root(SB), NOSPLIT, $32-0\n\tMOVO X0, 16(SP)\n\tVMOVDQA Y0, 0(SP)\n\tRET\n",
			problems: []string{"aligned to 16 bytes, but root aligns the stack to 8", "aligned to 32 bytes, but root aligns the stack to 8"},
		},
		{
			name:     "operand without VEX on an unaligned frame",
			source:   amd64Header + "TEXT ·root(SB), NOSPLIT, $32-0\n\tPXOR 16(SP), X0\n\tRET\n",
			problems: []string{"aligned to 16 bytes, but root aligns the stack to 8"},
		},
		{
			name:     "frame aligned less than the operand",
			source:   amd64Header + realigned + "\tVMOVDQA Y0, 0(SP)\n" + restore,
			problems: []string{"aligned to 32 bytes, but root aligns the stack to 16"},
		},
		{
			name:     "operand off the alignment of the frame",
			source:   amd64Header + realigned + "\tPXOR 8(SP), X0\n" + restore,
			problems: []string{"it is 8 bytes above the aligned stack pointer"},
		},
		{
			// the return address sits between the helper and the aligned frame
			name:     "helper operand off the alignment of the frame",
			source:   amd64Header + realigned + "\tCALL helper<>(SB)\n" + restore + "TEXT helper<>(SB), NOSPLIT|NOFRAME, $0-0\n\tPXOR 16(SP), X1\n\tRET\n",
			problems: []string{"it is 8 bytes above the aligned stack pointer"},
		},
		{
			name:     "operand past the frame",
			source:   amd64Header + "TEXT ·root(SB), NOSPLIT, $32-0\n\tMOVQ AX, 24(SP)\n\tMOVQ AX, 32(SP)\n\tMOVL AX, 30(SP)\n\tRET\n",
			problems: []string{"touches 8 bytes at 32 of the frame of root, which has 32", "touches 4 bytes at 30 of the frame of root, which has 32"},
		},
		{
			// realigning can take up to the alignment away from the frame
			name:     "operand in the part of the frame realigning uses up",
			source:   amd64Header + realigned + "\tMOVQ AX, 40(SP)\n" + restore,
			problems: []string{"touches 8 bytes at 40 of the frame of root, which has 40"},
		},
		{
			name:     "helper operand below the frame",
			source:   amd64Header + "TEXT ·root(SB), 0, $32-0\n\tCALL helper<>(SB)\n\tRET\nTEXT helper<>(SB), NOSPLIT|NOFRAME, $0-0\n\tMOVQ AX, 0(SP)\n\tRET\n",
			problems: []string{"touches 8 bytes at -8 of the frame of root"},
		},
		{
			name:     "push in a fixed frame",
			source:   amd64Header + "TEXT ·root(SB), NOSPLIT, $32-0\n\tPUSHQ AX\n\tPOPQ AX\n\tRET\n",
			problems: []string{"moves the stack pointer", "moves the stack pointer"},
		},
		{
			name:     "operand that lost its symbol",
			source:   amd64Header + "TEXT ·root(SB), NOSPLIT, $0-0\n\tVPBROADCASTQ 0, Z1\n\tLONG $0x0d6f0f66; LONG $0x00000000\n\tRET\n",
			problems: []string{"refers to no symbol", "refers to no symbol"},
		},
		{
			name: "aligned load of a constant the linker aligns less",
			source: amd64Header + "DATA wide<>+0x00(SB)/8, $1\nGLOBL wide<>(SB), (RODATA|NOPTR), $64\n" +
				"DATA short<>+0x00(SB)/8, $1\nGLOBL short<>(SB), (RODATA|NOPTR), $8\n" +
				"TEXT ·root(SB), NOSPLIT, $0-0\n\tVMOVDQU64 wide<>(SB), Z0\n\tVMOVDQA wide<>(SB), Y0\n\tVMOVDQA64 wide<>(SB), Z0\n\tMOVO wide<>+8(SB), X0\n\tPXOR short<>(SB), X0\n\tRET\n",
			problems: []string{"needs wide+0 aligned to 64 bytes", "needs wide+8 aligned to 16 bytes", "needs short+0 aligned to 16 bytes"},
		},
		{
			name:   "arm64 frame",
			source: arm64Header + "TEXT ·root(SB), 0, $32-0\n\tMOVD R0, 8(RSP)\n\tFMOVQ F0, 24(RSP)\n\tBL helper<>(SB)\n\tRET\nTEXT helper<>(SB), NOSPLIT|NOFRAME, $0-0\n\tMOVD R0, 32(RSP)\n\tRET\n",
		},
		{
			name:     "arm64 operand outside the frame",
			source:   arm64Header + "TEXT ·root(SB), 0, $32-0\n\tMOVD R0, (RSP)\n\tMOVD R0, 40(RSP)\n\tFMOVQ F0, 32(RSP)\n\tBL helper<>(SB)\n\tRET\nTEXT helper<>(SB), NOSPLIT|NOFRAME, $0-0\n\tMOVD R1, 40(RSP)\n\tRET\n",
			problems: []string{"touches 8 bytes at 0(RSP)", "touches 8 bytes at 40(RSP)", "touches 16 bytes at 32(RSP)", "helper: touches 8 bytes at 40(RSP)"},
		},
		{
			name:     "arm64 stack pointer moved",
			source:   arm64Header + "TEXT ·root(SB), 0, $32-0\n\tSUB $16, RSP\n\tMOVD.W R0, -16(RSP)\n\tRET\n",
			problems: []string{"moves the stack pointer", "moves the stack pointer"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "broken.s")
			require.NoError(t, os.WriteFile(path, []byte(tc.source), 0o644))
			file := loadAssembly(t, path)

			var problems []string
			if file.arch == "arm64" {
				problems = checkArm64(file)
			} else {
				problems, _ = checkAmd64(file)
			}
			require.Len(t, problems, len(tc.problems), "%s", strings.Join(problems, "\n"))
			for i, want := range tc.problems {
				assert.Contains(t, problems[i], want)
			}
		})
	}
}
