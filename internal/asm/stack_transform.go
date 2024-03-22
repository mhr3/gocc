package asm

import "strings"

func dropStackChangesAmd64(function Function) Function {
	var foundPrologue, foundEpilogue bool

	/*
		BYTE $0x55               // PUSHQ BP;	pushq	%rbp
		WORD $0x8948; BYTE $0xe5 // MOVQ SP, BP;	movq	%rsp, %rbp
		LONG $0xf8e48348         // ANDQ $-0x8, SP;	andq	$-8, %rsp
		WORD $0xaf0f; BYTE $0xfa // IMULL DX, DI;	imull	%edx, %edi
		WORD $0x6348; BYTE $0xc7 // MOVSXD DI, AX;	movslq	%edi, %rax
		WORD $0x0148; BYTE $0xf0 // ADDQ SI, AX;	addq	%rsi, %rax
		WORD $0x8948; BYTE $0x01 // MOVQ AX, 0(CX);	movq	%rax, (%rcx)
		WORD $0x8948; BYTE $0xec // MOVQ BP, SP;	movq	%rbp, %rsp
		BYTE $0x5d               // POPQ BP;	popq	%rbp
		RET                      // retq
	*/

	for i, line := range function.Lines {
		if strings.HasPrefix(line.Assembly, "push") && strings.HasSuffix(line.Assembly, "rbp") {
			foundPrologue = true
		}

		if strings.HasPrefix(line.Assembly, "ret") && i > 0 {
			for j := i - 1; j >= 0; j-- {
				prevLine := function.Lines[j]
				if strings.HasPrefix(prevLine.Assembly, "pop") && strings.HasSuffix(prevLine.Assembly, "rbp") {
					foundEpilogue = true
					break
				}
			}
		}

		if foundPrologue && foundEpilogue {
			break
		}
	}

	if foundPrologue && foundEpilogue {
		// remove them
		newLines := make([]Line, 0, len(function.Lines))

		for _, line := range function.Lines {
			if strings.HasPrefix(line.Assembly, "push") && strings.HasSuffix(line.Assembly, "rbp") ||
				strings.HasPrefix(line.Assembly, "pop") && strings.HasSuffix(line.Assembly, "rbp") {
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

func dropStackChangesArm64(function Function) Function {
	var foundPrologue, foundEpilogue bool

	for i, line := range function.Lines {
		if strings.HasPrefix(line.Assembly, "stp") && strings.Contains(line.Assembly, "sp") {
			foundPrologue = true
		}

		if strings.HasPrefix(line.Assembly, "ret") && i > 0 {
			for j := i - 1; j >= 0; j-- {
				prevLine := function.Lines[j]
				if strings.HasPrefix(prevLine.Assembly, "ldp") && strings.Contains(prevLine.Assembly, "sp") {
					foundEpilogue = true
					break
				}
			}
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
