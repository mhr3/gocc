package asm

import (
	"fmt"
	"strings"

	"github.com/mhr3/gocc/internal/config"
)

func ApplyTransforms(arch *config.Arch, functions []Function) []Function {
	for i, function := range functions {
		functions[i] = transformFunction(arch, function)
	}

	return functions
}

func transformFunction(arch *config.Arch, function Function) Function {
	// Apply the transforms
	function = transformReturns(arch, function)
	function = rewriteJumpsAndLoads(arch, function)
	function = checkStackManipulation(arch, function)
	function = storeReturnValue(arch, function)

	// weird type of transform, but we'll keep it here for now
	if !(arch != nil && arch.Name == "arm64" && function.Internal) {
		// Keep raw encodings for the hidden C frames of internal ARM64 helpers.
		// In particular, exposing SUB/ADD RSP to cmd/asm makes the linker treat
		// these NOFRAME helpers as ordinary Go nosplit frames.
		function = removeBinaryInstructions(arch, function)
	}

	return function
}

func transformReturns(_ *config.Arch, function Function) Function {
	for i := 0; i < len(function.Lines); i++ {
		line := function.Lines[i]
		if strings.HasPrefix(line.Assembly, "ret") {
			// we need to remove the binary representation of the return instruction
			function.Lines[i].Binary = nil
			function.Lines[i].Disassembled = "RET"
		}
	}

	return function
}

func rewriteJumpsAndLoads(arch *config.Arch, function Function) Function {
	if arch == nil {
		return function
	}

	for i, line := range function.Lines {
		// rewrite some instructions
		parts := []string{line.Assembly}
		if line.Disassembled != "" {
			parts = append([]string{line.Disassembled}, parts...)
		}
		combined := strings.Join(parts, ";\t")

		// FIXME: cleanup
		if arch.JumpInstr != nil && arch.JumpInstr.MatchString(combined) {
			reParams := getRegexpParams(arch.JumpInstr, combined)
			rewritten := fmt.Sprintf("%s %s", strings.ToUpper(reParams["instr"]), reParams["label"])
			function.Lines[i].Disassembled = rewritten
			function.Lines[i].Binary = nil
			continue
		}

		switch arch.Name {
		case "amd64":
			if arch.DataLoad.MatchString(combined) {
				rewriteLoadAmd64(arch, function, line, combined, function.Lines[i:])
			}
		case "arm64":
			if arch.DataLoad.MatchString(combined) {
				rewriteLoadArm64(arch, function, line, combined, function.Lines[i:])
			}
		}
	}

	return function
}

func checkStackManipulation(arch *config.Arch, function Function) Function {
	if arch == nil {
		return function
	}

	switch arch.Name {
	case "amd64":
		return checkStackUnified(arch, function)
	case "arm64":
		if function.Internal {
			// Internal C-ABI helpers keep Clang's exact frame. A Go prologue would
			// both clobber C callee-saved registers and attempt stack growth without
			// knowing that the arguments live in registers. The public translated
			// entry point remains a normal, stack-splitting Go frame.
			function.LocalsSize = 0
			return function
		}
		return checkStackUnified(arch, function)
	}

	panic(fmt.Sprintf("no stack checking function for architecture: %s", arch.Name))
}

func removeBinaryInstructions(arch *config.Arch, function Function) Function {
	if arch == nil {
		return function
	}

	switch arch.Name {
	case "amd64":
		return removeBinaryInstructionsAmd64(arch, function)
	case "arm64":
		return removeBinaryInstructionsArm64(arch, function)
	}

	return function
}

func storeReturnValue(arch *config.Arch, function Function) Function {
	if function.Ret == nil {
		return function
	}

	offset, _ := function.ParamsSize(arch)
	retSz := int8(function.Ret.Size())
	op, ok := arch.MovInstr[retSz]
	if !ok {
		panic(fmt.Errorf("unable to store return value with size %d", function.Ret.Size()))
	}

	retRegister := arch.RetRegister
	if function.Ret.IsFloatingPoint() {
		op = arch.MovFPInstr[retSz]
		retRegister = arch.FloatRegisters[0]
	}
	retInstr := fmt.Sprintf("%s %s, ret+%d(FP)", op, retRegister, offset)

	// we need to inject a new MOV instruction to store the return value on stack
	for i := 0; i < len(function.Lines); i++ {
		line := function.Lines[i]
		if strings.HasPrefix(line.Assembly, "ret") {
			function.Lines = append(function.Lines[:i], append([]Line{
				{
					Labels:       line.Labels,
					Disassembled: retInstr,
				},
			}, function.Lines[i:]...)...)
			// we moved the labels to the new instruction, so we need to remove them from the old one
			function.Lines[i+1].Labels = nil
			i++
		}
	}

	return function
}
