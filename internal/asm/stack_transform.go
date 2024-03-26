package asm

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/kelindar/gocc/internal/config"
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
				instr := fmt.Sprintf("%s %s, %d(SP)", arch.CallOp[8], parts[1], pushOffset)
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
				instr := fmt.Sprintf("%s %d(SP), %s", arch.CallOp[8], pushOffset, parts[1])
				if pushOffset < pushOffsetStart {
					panic("unable to rewrite push/pop instructions")
				}
				lineCpy := line
				lineCpy.Disassembled = instr
				lineCpy.Binary = nil
				newLines = append(newLines, lineCpy)
				continue
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
	var foundPrologue, foundEpilogue bool

	for _, line := range function.Lines {
		if strings.HasPrefix(line.Assembly, "stp") && strings.Contains(line.Assembly, "sp") {
			foundPrologue = true
		}

		if strings.HasPrefix(line.Assembly, "ldp") && strings.Contains(line.Assembly, "sp") {
			foundEpilogue = true
		}

		if foundPrologue && foundEpilogue {
			break
		}
	}

	if foundPrologue && foundEpilogue {
		// remove them
		newLines := make([]Line, 0, len(function.Lines))

		for _, line := range function.Lines {
			if (strings.HasPrefix(line.Assembly, "stp") && strings.Contains(line.Assembly, "sp")) ||
				(strings.HasPrefix(line.Assembly, "mov") && strings.Contains(line.Assembly, "sp")) ||
				(strings.HasPrefix(line.Assembly, "ldp") && strings.Contains(line.Assembly, "sp")) {
				lineCpy := line
				lineCpy.Disassembled = "NOP"
				lineCpy.Binary = nil
				newLines = append(newLines, lineCpy)
				continue
			}

			newLines = append(newLines, line)
		}

		function.Lines = newLines
	}

	return function
}
