package asm

import (
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/mhr3/gocc/internal/config"
	"golang.org/x/arch/arm64/arm64asm"
	"golang.org/x/arch/x86/x86asm"
)

func checkStackAmd64(arch *config.Arch, function Function) Function {
	var (
		rewriteRequired bool
		numPushes       int
		extraStack      int
		stackAllocIdx   = -1
	)

	/*
		BYTE $0x55               // pushq	%rbp
		WORD $0x8948; BYTE $0xe5 // movq	%rsp, %rbp
		LONG $0xf8e48348         // andq	$-8, %rsp
		WORD $0xaf0f; BYTE $0xfa // imull	%edx, %edi
		WORD $0x6348; BYTE $0xc7 // movslq	%edi, %rax
		WORD $0x0148; BYTE $0xf0 // addq	%rsi, %rax
		WORD $0x8948; BYTE $0x01 // movq	%rax, (%rcx)
		---
		WORD $0x8948; BYTE $0xec // movq	%rbp, %rsp
		BYTE $0x5d               // popq	%rbp
		RET                      // retq
	*/
	spInstruction := regexp.MustCompile(`\brsp\b`)

	for i, line := range function.Lines {
		if spInstruction.MatchString(line.Assembly) {
			if strings.HasPrefix(line.Assembly, "mov") && strings.Contains(line.Assembly, "rbp") {
				// moving SP to BP and back
				continue
			}
			if strings.HasPrefix(line.Assembly, "and") {
				// stack alignment
				// FIXME: this basically grows the stack, should adjust for it
				inst := decodeAmd64Line(line)
				if inst.Op != x86asm.AND {
					panic(fmt.Sprintf("unexpected instruction: %q", line.Assembly))
				}
				imm, isImm := inst.Args[1].(x86asm.Imm)
				align := int64(imm)
				if !isImm || align != -8 {
					rewriteRequired = true
				}
				continue
			}
			if strings.HasPrefix(line.Assembly, "sub") {
				// allocating stack space
				inst := decodeAmd64Line(line)
				if inst.Op != x86asm.SUB {
					panic(fmt.Sprintf("unexpected instruction: %q", line.Assembly))
				}
				imm, isImm := inst.Args[1].(x86asm.Imm)
				if !isImm {
					rewriteRequired = true
					continue
				}
				if extraStack != 0 {
					panic("failed to analyze stack operations")
				}
				extraStack = int(imm)
				stackAllocIdx = i
			}
			if strings.HasPrefix(line.Assembly, "lea") {
				continue
			}
			rewriteRequired = true
			continue
		}
		if strings.HasPrefix(line.Assembly, "push") {
			inst := decodeAmd64Line(line)
			if inst.Op != x86asm.PUSH {
				panic(fmt.Sprintf("unexpected instruction: %q", line.Assembly))
			}
			dstReg, _ := inst.Args[0].(x86asm.Reg)
			switch dstReg {
			case x86asm.RBP, x86asm.RBX, x86asm.R12, x86asm.R13, x86asm.R14, x86asm.R15:
				// go's ABI0 doesn't have callee-saved registers
			default:
				rewriteRequired = true
				numPushes++
			}
		}
	}

	if !rewriteRequired {
		// remove them
		newLines := make([]Line, 0, len(function.Lines))

		for _, line := range function.Lines {
			doSkip := false
			asm := line.Assembly
			asmFields := strings.Fields(asm)
			if asmFields[0] == "push" || asmFields[0] == "pop" {
				inst := decodeAmd64Line(line)
				if inst.Op != x86asm.PUSH && inst.Op != x86asm.POP {
					panic(fmt.Sprintf("unexpected instruction: %q", line.Assembly))
				}
				dstReg, _ := inst.Args[0].(x86asm.Reg)
				switch dstReg {
				case x86asm.RBP, x86asm.RBX, x86asm.R12, x86asm.R13, x86asm.R14, x86asm.R15:
					// can be dropped
					doSkip = true
				}
			} else if asmFields[0] == "lea" {
				parts := asmFields
				if len(parts) > 1 && strings.HasPrefix(parts[1], "rsp") {
					// writing into rsp, drop
					doSkip = true
				}
			} else if strings.HasPrefix(asm, "mov") && (strings.HasSuffix(asm, "rsp") || strings.HasSuffix(asm, "rbp")) ||
				strings.HasPrefix(asm, "and") && strings.Contains(asm, "rsp") {
				// we need to drop all of these
				doSkip = true
			}

			if doSkip {
				lineCpy := line
				lineCpy.Disassembled = "NOP"
				lineCpy.Binary = nil
				newLines = append(newLines, lineCpy)
				continue
			}

			newLines = append(newLines, line)
		}

		function.Lines = newLines
	} else {
		// Complex stack manipulation (amd64): rewrite push/pop to use stack offsets
		// Note: This path has known limitations (see FIXME below), warn the user.
		fnName := function.Name
		if fnName == "" {
			fnName = "[unknown]"
		}
		fmt.Fprintf(os.Stderr, "WARN: %s: contains complex stack manipulation, rewriting push/pop\n", fnName)
		newLines := make([]Line, 0, len(function.Lines))
		pushOffsetStart := extraStack
		//pushOffsetStart += -pushOffsetStart & (15)
		pushOffset := pushOffsetStart
		maxOffset := pushOffset

		for i, line := range function.Lines {
			asm := line.Assembly
			asmFields := strings.Fields(asm)
			if asmFields[0] == "push" || asmFields[0] == "pop" {
				inst := decodeAmd64Line(line)
				if inst.Op != x86asm.PUSH && inst.Op != x86asm.POP {
					panic(fmt.Sprintf("unexpected instruction: %q", line.Assembly))
				}
				dstReg, _ := inst.Args[0].(x86asm.Reg)
				switch dstReg {
				case x86asm.RBP, x86asm.RBX, x86asm.R12, x86asm.R13, x86asm.R14, x86asm.R15:
					// can be dropped
					lineCpy := line
					lineCpy.Disassembled = "NOP"
					lineCpy.Binary = nil
					newLines = append(newLines, lineCpy)
					continue
				}
			}
			if stackAllocIdx == i ||
				strings.HasPrefix(asm, "mov") && (strings.HasSuffix(asm, "rsp") || strings.HasSuffix(asm, "rbp")) {
				// we need to drop all of these
				lineCpy := line
				lineCpy.Disassembled = "NOP"
				lineCpy.Binary = nil
				newLines = append(newLines, lineCpy)
				continue
			}

			if strings.HasPrefix(asm, "lea") {
				parts := strings.Fields(asm)
				if len(parts) > 1 && strings.HasPrefix(parts[1], "rsp") {
					// writing into rsp, drop
					lineCpy := line
					lineCpy.Disassembled = "NOP"
					lineCpy.Binary = nil
					newLines = append(newLines, lineCpy)
					continue
				}
			}

			if asmFields[0] == "push" {
				// rewrite to moves and hope they're not dynamic
				parts := strings.Fields(line.Disassembled)
				instr := fmt.Sprintf("%s %s, %d(SP)", arch.MovInstr[8], parts[1], pushOffset)
				pushOffset += 8
				if pushOffset > maxOffset {
					maxOffset = pushOffset
				}
				lineCpy := line
				lineCpy.Disassembled = instr
				lineCpy.Binary = nil
				newLines = append(newLines, lineCpy)
				continue
			}
			if asmFields[0] == "pop" {
				parts := strings.Fields(line.Disassembled)
				pushOffset -= 8
				instr := fmt.Sprintf("%s %d(SP), %s", arch.MovInstr[8], pushOffset, parts[1])
				if pushOffset < pushOffsetStart {
					panic("unable to rewrite push/pop instructions")
				}
				lineCpy := line
				lineCpy.Disassembled = instr
				lineCpy.Binary = nil
				newLines = append(newLines, lineCpy)
				continue
			}
			if strings.HasPrefix(asm, "and") && spInstruction.MatchString(line.Assembly) {
				inst := decodeAmd64Line(line)
				if inst.Op != x86asm.AND {
					panic(fmt.Sprintf("unexpected instruction: %q", line.Assembly))
				}
				imm, isImm := inst.Args[1].(x86asm.Imm)
				align := int64(imm)
				if isImm && align == -8 {
					// drop stack alignment instruction
					lineCpy := line
					lineCpy.Disassembled = "NOP"
					lineCpy.Binary = nil
					newLines = append(newLines, lineCpy)
					continue
				}
			}
			if asm == "ret" {
				// we can encounter more pops
				pushOffset = maxOffset
			}

			// FIXME: we're keeping the SP alignment instruction, won't work if the stack isn't aligned
			// although should be ok if we fit into the red zone

			newLines = append(newLines, line)
		}

		function.Lines = newLines
		function.LocalsSize = maxOffset
	}

	return function
}

type virtualSP struct {
	arm64asm.RegSP
	name   string
	offset int
}

func (v *virtualSP) String() string {
	// ret-8(SP)
	return fmt.Sprintf("%s%d(SP)", v.name, v.offset)
}

func checkStackArm64(arch *config.Arch, function Function) Function {
	var (
		rewriteRequired bool
		complexManip    bool
		baseStack       int
		extraStack      int
	)

	/*
		// stp	x29, x30, [sp, #-80]!
		// sub	x9, sp, #16
		// stp	x26, x25, [sp, #16]
		// stp	x24, x23, [sp, #32]
		// mov	x29, sp
		// stp	x22, x21, [sp, #48]
		// stp	x20, x19, [sp, #64]
		// and	sp, x9, #0xfffffffffffffff8
		---
		// mov	sp, x29
		// ldp	x20, x19, [sp, #64]
		// ldp	x22, x21, [sp, #48]
		// ldp	x24, x23, [sp, #32]
		// ldp	x26, x25, [sp, #16]
		// ldp	x29, x30, [sp], #80
		// ret
	*/

	spInstruction := regexp.MustCompile(`\bsp\b`)

	for _, line := range function.Lines {
		if spInstruction.MatchString(line.Assembly) {
			inst := decodeArm64Line(line)
			parts := strings.Fields(line.Assembly)

			switch inst.Op {
			case arm64asm.STP:
				if len(inst.Args) > 2 && inst.Args[0] == arm64asm.X29 && inst.Args[1] == arm64asm.X30 {
					// storing the frame pointer
					imm, ok := inst.Args[2].(arm64asm.MemImmediate)
					// this tells us how much stack space we're using
					if ok && imm.Base == arm64asm.RegSP(arm64asm.SP) && baseStack == 0 {
						n := immFromMemImmediate(imm)
						baseStack = -n
						extraStack = baseStack
					}
				} else if len(inst.Args) > 2 {
					imm, ok := inst.Args[2].(arm64asm.MemImmediate)
					if ok && imm.Base == arm64asm.RegSP(arm64asm.SP) && baseStack == 0 {
						n := immFromMemImmediate(imm)
						baseStack = -n
						extraStack = baseStack
					}
					// this could still be fine, as long as it's doing just callee-saved registers
					rewriteRequired = true
				}
			case arm64asm.STR:
				imm, ok := inst.Args[1].(arm64asm.MemImmediate)
				if ok && imm.Base == arm64asm.RegSP(arm64asm.SP) && baseStack == 0 {
					n := immFromMemImmediate(imm)
					baseStack = -n
					extraStack = baseStack
				}
				// this could still be fine, as long as it's doing just callee-saved registers
				rewriteRequired = true
			case arm64asm.AND:
				// stack alignment
				// this basically grows the stack, need to adjust for it
				targetReg := inst.Args[0]
				if targetReg == arm64asm.SP {
					// allocating more stack space
					rewriteRequired = true
					// TODO: definitely clear sign that we're doing something with the stack
					complexManip = true
				}
			case arm64asm.SUB:
				// allocating stack space
				targetReg := inst.Args[0]
				srcReg := inst.Args[1]
				if targetReg == arm64asm.RegSP(arm64asm.SP) || srcReg == arm64asm.RegSP(arm64asm.SP) {
					complexManip = true
					// probably allocating more stack space, either directly or through an extra register
					imm := parts[3]
					imm = strings.TrimPrefix(imm, "#")
					if n, err := strconv.Atoi(imm); err == nil {
						extraStack += n
						rewriteRequired = true
					}
				}
			}
			continue
		}
	}

	if !rewriteRequired {
		if extraStack != 16 {
			panic("failed to detect stack manipulation")
		}
		// remove the frame pointer instructions
		newLines := make([]Line, 0, len(function.Lines))

		for _, line := range function.Lines {
			if spInstruction.MatchString(line.Assembly) {
				if strings.HasPrefix(line.Assembly, "stp") || strings.HasPrefix(line.Assembly, "mov") ||
					strings.HasPrefix(line.Assembly, "ldp") {
					lineCpy := line
					lineCpy.Disassembled = "NOP"
					lineCpy.Binary = nil
					newLines = append(newLines, lineCpy)
					continue
				}
			}
			newLines = append(newLines, line)
		}

		function.Lines = newLines
	} else if !complexManip {
		newLines := make([]Line, 0, len(function.Lines))
		stackAllocator := map[string]int{}
		stackSpace := -extraStack

		for _, line := range function.Lines {
			asm := line.Assembly
			// detect everything that touches SP
			if spInstruction.MatchString(asm) {
				inst := decodeArm64Line(line)
				doSkip := false

				// Go's ABI0 doesn't require callee-saved registers
				if isCalleeSavedRegPair(inst) {
					doSkip = true
				} else if (inst.Op == arm64asm.STR || inst.Op == arm64asm.LDR) && isCalleeSavedReg(inst.Args[0]) {
					doSkip = true
				} else if inst.Op == arm64asm.MOV && inst.Args[0] == arm64asm.RegSP(arm64asm.X29) {
					doSkip = true
				}

				if doSkip {
					lineCpy := line
					lineCpy.Disassembled = "NOP"
					lineCpy.Binary = nil
					newLines = append(newLines, lineCpy)
					continue
				}

				switch inst.Op {
				case arm64asm.STP, arm64asm.STR:
					numRegs, registers := collectSpillRegisters(inst.Args)
					stackAllocator[registers] = stackSpace
					if stackSpace >= 0 {
						panic("stack space allocation failed")
					}
					stackSpace += 8 * numRegs

					if inst.Op == arm64asm.STP || inst.Op == arm64asm.STR {
						argIndex := 2
						if inst.Op == arm64asm.STR {
							argIndex = 1
						}
						imm, ok := inst.Args[argIndex].(arm64asm.MemImmediate)
						// this tells us how much stack space we're using
						if ok && imm.Base == arm64asm.RegSP(arm64asm.SP) {
							replacement := &virtualSP{RegSP: arm64asm.RegSP(arm64asm.SP), name: registers, offset: stackAllocator[registers]}
							inst.Args[argIndex] = replacement
						}
					}

					lineCpy := line
					lineCpy.Disassembled = arm64asm.GoSyntax(inst, 0, nil, nil)
					if idx := strings.Index(lineCpy.Disassembled, registers); idx > 0 {
						lineCpy.Disassembled = lineCpy.Disassembled[:idx] + strings.ToLower(registers) + lineCpy.Disassembled[idx+len(registers):]
					}
					lineCpy.Binary = nil
					newLines = append(newLines, lineCpy)
					continue
				case arm64asm.LDP, arm64asm.LDR:
					_, registers := collectSpillRegisters(inst.Args)

					stackOffset, ok := stackAllocator[registers]
					if ok && inst.Op == arm64asm.LDP || inst.Op == arm64asm.LDR {
						argIndex := 2
						if inst.Op == arm64asm.LDR {
							argIndex = 1
						}
						imm, ok := inst.Args[argIndex].(arm64asm.MemImmediate)
						if ok && imm.Base == arm64asm.RegSP(arm64asm.SP) {
							replacement := &virtualSP{RegSP: arm64asm.RegSP(arm64asm.SP), name: registers, offset: stackOffset}
							inst.Args[argIndex] = replacement
						}
					}

					lineCpy := line
					lineCpy.Disassembled = arm64asm.GoSyntax(inst, 0, nil, nil)
					if idx := strings.Index(lineCpy.Disassembled, registers); idx > 0 {
						lineCpy.Disassembled = lineCpy.Disassembled[:idx] + strings.ToLower(registers) + lineCpy.Disassembled[idx+len(registers):]
					}
					lineCpy.Binary = nil
					newLines = append(newLines, lineCpy)
					continue
				case arm64asm.AND, arm64asm.SUB, arm64asm.ADD:
					if len(inst.Args) > 2 && inst.Args[0] == arm64asm.RegSP(arm64asm.SP) {
						// stack alloc/dealloc/alignment writing back into SP - NOP it
						// (Go's assembler handles stack via the frame size declaration)
						lineCpy := line
						lineCpy.Disassembled = "NOP"
						lineCpy.Binary = nil
						newLines = append(newLines, lineCpy)
						continue
					}
					if inst.Op == arm64asm.SUB && inst.Args[1] == arm64asm.RegSP(arm64asm.SP) {
						// we're allocating stack space, but we already did that, just do a MOVD
						replInst := arm64asm.Inst{Op: arm64asm.MOV, Args: arm64asm.Args{inst.Args[0], inst.Args[1]}}
						lineCpy := line
						lineCpy.Disassembled = arm64asm.GoSyntax(replInst, 0, nil, nil)
						lineCpy.Binary = nil
						newLines = append(newLines, lineCpy)
						continue
					}
				}
			}

			// FIXME: we're keeping the SP alignment instruction, won't work if the stack isn't aligned
			// although should be ok if we fit into the red zone

			newLines = append(newLines, line)
		}

		function.Lines = newLines
		if len(stackAllocator) == 0 {
			function.LocalsSize = 0
		} else {
			function.LocalsSize = extraStack
		}
	} else {
		// Complex stack manipulation (arm64): NOP all callee-saved register save/restore
		// and all SP-modifying instructions. Go manages stack via TEXT declaration.
		newLines := make([]Line, 0, len(function.Lines))

		for _, line := range function.Lines {
			asm := line.Assembly
			if spInstruction.MatchString(asm) {
				inst := decodeArm64Line(line)

				// NOP callee-saved register pairs - Go's ABI0 doesn't require them
				if isCalleeSavedRegPair(inst) {
					lineCpy := line
					lineCpy.Disassembled = "NOP"
					lineCpy.Binary = nil
					newLines = append(newLines, lineCpy)
					continue
				}

				// NOP frame pointer setup (mov x29, sp)
				if inst.Op == arm64asm.MOV && inst.Args[0] == arm64asm.RegSP(arm64asm.X29) {
					lineCpy := line
					lineCpy.Disassembled = "NOP"
					lineCpy.Binary = nil
					newLines = append(newLines, lineCpy)
					continue
				}

				switch inst.Op {
				case arm64asm.STP, arm64asm.LDP:
					// Keep non-callee-saved STP/LDP (e.g., SIMD register spills)
					lineCpy := line
					lineCpy.Disassembled = arm64asm.GoSyntax(inst, 0, nil, nil)
					lineCpy.Binary = nil
					newLines = append(newLines, lineCpy)
					continue
				case arm64asm.STR, arm64asm.LDR:
					// NOP single callee-saved register save/restore
					if isCalleeSavedReg(inst.Args[0]) {
						lineCpy := line
						lineCpy.Disassembled = "NOP"
						lineCpy.Binary = nil
						newLines = append(newLines, lineCpy)
						continue
					}
					lineCpy := line
					lineCpy.Disassembled = arm64asm.GoSyntax(inst, 0, nil, nil)
					lineCpy.Binary = nil
					newLines = append(newLines, lineCpy)
					continue
				case arm64asm.AND, arm64asm.SUB, arm64asm.ADD:
					// NOP any arithmetic that writes to SP (stack alloc/dealloc/alignment)
					// Go's assembler manages stack frame via the declaration, not explicit SP manipulation
					if len(inst.Args) > 0 && inst.Args[0] == arm64asm.RegSP(arm64asm.SP) {
						lineCpy := line
						lineCpy.Disassembled = "NOP"
						lineCpy.Binary = nil
						newLines = append(newLines, lineCpy)
						continue
					}
					// Handle SUB that reads from SP to compute stack-relative address
					if inst.Op == arm64asm.SUB && len(inst.Args) > 1 && inst.Args[1] == arm64asm.RegSP(arm64asm.SP) {
						replInst := arm64asm.Inst{Op: arm64asm.MOV, Args: arm64asm.Args{inst.Args[0], inst.Args[1]}}
						lineCpy := line
						lineCpy.Disassembled = arm64asm.GoSyntax(replInst, 0, nil, nil)
						lineCpy.Binary = nil
						newLines = append(newLines, lineCpy)
						continue
					}
				}
			}

			// FIXME: we're keeping the SP alignment instruction, won't work if the stack isn't aligned
			// although should be ok if we fit into the red zone

			newLines = append(newLines, line)
		}

		function.Lines = newLines
		// FIXME: we're doing extra 16bytes (which C uses for x29/x30)
		function.LocalsSize = extraStack
	}

	return function
}

// isCalleeSavedRegPair returns true if the STP/LDP instruction saves/restores
// a callee-saved register pair that Go's ABI0 doesn't require preserving.
// Only matches SP-based prologue/epilogue patterns.
func isCalleeSavedRegPair(inst arm64asm.Inst) bool {
	if inst.Op != arm64asm.STP && inst.Op != arm64asm.LDP {
		return false
	}

	// Only match SP-based prologue/epilogue patterns
	if len(inst.Args) > 2 {
		if mem, ok := inst.Args[2].(arm64asm.MemImmediate); ok {
			if mem.Base != arm64asm.RegSP(arm64asm.SP) {
				return false
			}
		}
	}

	r0, ok0 := inst.Args[0].(arm64asm.Reg)
	r1, ok1 := inst.Args[1].(arm64asm.Reg)
	if !ok0 || !ok1 {
		return false
	}

	// Accept either ordering for robustness (compilers may use ascending or descending)
	isPair := func(a, b, x, y arm64asm.Reg) bool {
		return (a == x && b == y) || (a == y && b == x)
	}

	switch {
	case isPair(r0, r1, arm64asm.X19, arm64asm.X20):
		return true
	case isPair(r0, r1, arm64asm.X21, arm64asm.X22):
		return true
	case isPair(r0, r1, arm64asm.X23, arm64asm.X24):
		return true
	case isPair(r0, r1, arm64asm.X25, arm64asm.X26):
		return true
	case isPair(r0, r1, arm64asm.X27, arm64asm.X28):
		return true
	case isPair(r0, r1, arm64asm.X29, arm64asm.X30):
		return true
	}
	return false
}

// isCalleeSavedReg returns true if the register is a callee-saved register
// that Go's ABI0 doesn't require preserving.
func isCalleeSavedReg(arg arm64asm.Arg) bool {
	switch arg {
	case arm64asm.X19, arm64asm.X20, arm64asm.X21, arm64asm.X22, arm64asm.X23, arm64asm.X24,
		arm64asm.X25, arm64asm.X26, arm64asm.X27, arm64asm.X28, arm64asm.X29, arm64asm.X30:
		return true
	}
	return false
}

func decodeAmd64Line(line Line) x86asm.Inst {
	binary := strings.Join(line.Binary, "")
	code, err := hex.DecodeString(binary)
	if err != nil {
		panic(err)
	}
	inst, err := x86asm.Decode(code, 64)
	if err != nil {
		panic(fmt.Errorf("failed to decode instruction: %v (%q)", err, binary))
	}
	return inst
}

func decodeArm64Line(line Line) arm64asm.Inst {
	binary := strings.Join(line.Binary, "")
	code, err := hex.DecodeString(binary)
	if err != nil {
		panic(err)
	}
	inst, err := arm64asm.Decode(code)
	if err != nil {
		panic(fmt.Errorf("failed to decode instruction: %v (%q)", err, binary))
	}
	return inst
}

func collectSpillRegisters(args arm64asm.Args) (numRegs int, registers string) {
	for _, arg := range args {
		if _, isReg := arg.(arm64asm.Reg); isReg {
			numRegs++
			registers += arg.String()
		}
	}
	if len(registers) > 0 {
		registers += "SPILL"
	}
	return
}

func immFromMemImmediate(imm arm64asm.MemImmediate) int {
	// no imm.Imm :facepalm:
	switch imm.Mode {
	case arm64asm.AddrOffset, arm64asm.AddrPreIndex, arm64asm.AddrPostIndex:
		s := imm.String()
		commaIdx := strings.Index(s, ",")
		if commaIdx == -1 {
			return 0
		}
		s = s[commaIdx+1:]
		s = strings.TrimPrefix(s, "#")
		s = strings.TrimSuffix(s, "!")
		s = strings.TrimSuffix(s, "]")
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
}
