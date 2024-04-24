package asm

import (
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/kelindar/gocc/internal/config"
	"golang.org/x/arch/arm64/arm64asm"
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
				continue
			}
			if strings.HasPrefix(line.Assembly, "sub") {
				// allocating stack space
				parts := strings.Fields(line.Assembly)
				operand := parts[len(parts)-1]
				if n, err := strconv.Atoi(operand); err == nil {
					if extraStack != 0 {
						panic("failed to analyze stack operations")
					}
					extraStack = n
					stackAllocIdx = i
				}
			}
			rewriteRequired = true
			continue
		}
		if strings.HasPrefix(line.Assembly, "push") && !strings.Contains(line.Assembly, "rbp") {
			rewriteRequired = true
			numPushes++
		}
	}

	if !rewriteRequired {
		// remove them
		newLines := make([]Line, 0, len(function.Lines))

		for _, line := range function.Lines {
			asm := line.Assembly
			if strings.HasPrefix(asm, "push") && strings.HasSuffix(asm, "rbp") ||
				strings.HasPrefix(asm, "pop") && strings.HasSuffix(asm, "rbp") ||
				strings.HasPrefix(asm, "mov") && (strings.HasSuffix(asm, "rsp") || strings.HasSuffix(asm, "rbp")) ||
				strings.HasPrefix(asm, "and") && strings.Contains(asm, "rsp") {
				// we need to drop all of these
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
		fmt.Fprintf(os.Stderr, "WARN: %s: contains complex stack manipulation, running experimental transform\n", function.Name)
		// go really doesn't like messing with SP, so we have two options:
		// 1) skip instructions that change it
		// 2) copy SP to BP and rewrite any instructions working with SP
		//    to refer to BP instead

		// we still need to remove the prologue/epilogue instructions
		newLines := make([]Line, 0, len(function.Lines))
		pushOffsetStart := extraStack
		//pushOffsetStart += -pushOffsetStart & (15)
		pushOffset := pushOffsetStart
		maxOffset := pushOffset

		for i, line := range function.Lines {
			asm := line.Assembly
			if stackAllocIdx == i ||
				strings.HasPrefix(asm, "push") && strings.HasSuffix(asm, "rbp") ||
				strings.HasPrefix(asm, "pop") && strings.HasSuffix(asm, "rbp") ||
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

			if strings.HasPrefix(asm, "push") {
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
			if strings.HasPrefix(asm, "pop") {
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

func checkStackArm64(arch *config.Arch, function Function) Function {
	var (
		rewriteRequired bool
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

			if inst.Op == arm64asm.STP && len(inst.Args) > 2 && inst.Args[0] == arm64asm.X29 && inst.Args[1] == arm64asm.X30 {
				// storing the frame pointer
				imm, ok := inst.Args[2].(arm64asm.MemImmediate)
				// this tells us how much stack space we're using
				if ok && imm.Base == arm64asm.RegSP(arm64asm.SP) && baseStack == 0 {
					n := immFromMemImmediate(imm)
					baseStack = -n
					extraStack = baseStack
				}
			} else if inst.Op == arm64asm.STP && len(inst.Args) > 2 {
				// storing registers other than frame pointer
				imm, ok := inst.Args[2].(arm64asm.MemImmediate)
				if ok && imm.Base == arm64asm.RegSP(arm64asm.SP) && baseStack == 0 {
					n := immFromMemImmediate(imm)
					baseStack = -n
					extraStack = baseStack
				}
				// this could still be fine, as long as it's doing just callee-saved registers
				rewriteRequired = true
			}
			if inst.Op == arm64asm.AND {
				// stack alignment
				// this basically grows the stack, need to adjust for it
				targetReg := inst.Args[0]
				if targetReg == arm64asm.SP {
					// allocating more stack space
					rewriteRequired = true
					// TODO: definitely clear sign that we're doing something with the stack
				}
			}
			if inst.Op == arm64asm.SUB {
				// allocating stack space
				targetReg := inst.Args[0]
				srcReg := inst.Args[1]
				if targetReg == arm64asm.RegSP(arm64asm.SP) || srcReg == arm64asm.RegSP(arm64asm.SP) {
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
	} else {
		fmt.Fprintf(os.Stderr, "WARN: %s: contains complex stack manipulation, running experimental transform\n", function.Name)
		// go really doesn't like messing with SP, so we have two options:
		// 1) skip instructions that change it
		// 2) copy SP to BP and rewrite any instructions working with SP
		//    to refer to BP instead

		// we still need to remove the prologue/epilogue instructions
		newLines := make([]Line, 0, len(function.Lines))

		for _, line := range function.Lines {
			asm := line.Assembly
			// detect everything that touches SP
			if spInstruction.MatchString(asm) {
				inst := decodeArm64Line(line)

				// drop the frame pointer instructions
				if ((inst.Op == arm64asm.STP || inst.Op == arm64asm.LDP) &&
					inst.Args[0] == arm64asm.X29 && inst.Args[1] == arm64asm.X30) ||
					inst.Op == arm64asm.MOV && inst.Args[0] == arm64asm.RegSP(arm64asm.X29) {
					lineCpy := line
					lineCpy.Disassembled = "NOP"
					lineCpy.Binary = nil
					newLines = append(newLines, lineCpy)
					continue
				}

				if inst.Op == arm64asm.STP {
					lineCpy := line
					lineCpy.Disassembled = arm64asm.GoSyntax(inst, 0, nil, nil)
					lineCpy.Binary = nil
					newLines = append(newLines, lineCpy)
					continue
				}
				if inst.Op == arm64asm.LDP {
					lineCpy := line
					lineCpy.Disassembled = arm64asm.GoSyntax(inst, 0, nil, nil)
					lineCpy.Binary = nil
					newLines = append(newLines, lineCpy)
					continue
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

func decodeArm64Line(line Line) arm64asm.Inst {
	code, err := hex.DecodeString(strings.Join(line.Binary, ""))
	if err != nil {
		panic(err)
	}
	inst, err := arm64asm.Decode(code)
	if err != nil {
		panic(err)
	}
	return inst
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
