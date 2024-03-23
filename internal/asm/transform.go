package asm

import (
	"fmt"
	"strings"

	"github.com/kelindar/gocc/internal/config"
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
	function = dropStackManipulation(arch, function)
	function = storeReturnValue(arch, function)

	return function
}

func transformReturns(arch *config.Arch, function Function) Function {
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

		if arch.JumpInstr != nil && arch.JumpInstr.MatchString(combined) {
			reParams := getRegexpParams(arch.JumpInstr, combined)
			rewritten := fmt.Sprintf("%s %s", strings.ToUpper(reParams["instr"]), reParams["label"])
			function.Lines[i].Disassembled = rewritten
			function.Lines[i].Binary = nil
		} else if arch.DataLoad != nil && arch.DataLoad.MatchString(combined) {
			reParams := getRegexpParams(arch.DataLoad, combined)
			op := arch.CallOp[8]
			addrMode := "$" // absolute addressing
			if instr, ok := reParams["instr"]; ok {
				op = instr
				addrMode = ""
			}
			rewritten := fmt.Sprintf("%s %s%s<>(SB), %s", op, addrMode, reParams["var"], reParams["register"])
			function.Lines[i].Disassembled = rewritten
			function.Lines[i].Binary = nil
		}
	}

	return function
}

func dropStackManipulation(arch *config.Arch, function Function) Function {
	if arch == nil {
		return function
	}

	switch arch.Name {
	case "arm64":
		return dropStackChangesArm64(function)
	case "amd64":
		return dropStackChangesAmd64(function)
	}

	return function
}

func storeReturnValue(arch *config.Arch, function Function) Function {
	if function.Ret == nil {
		return function
	}

	offset, _ := function.ParamsSize(arch)
	op, ok := arch.CallOp[int8(function.Ret.Size())]
	if !ok {
		panic(fmt.Errorf("unable to store return value with size %d", function.Ret.Size()))
	}

	// FIXME: float return values (uses X0 register on amd64, F0 on arm64)
	retInstr := fmt.Sprintf("%s %s, ret+%d(FP)", op, arch.RetRegister, offset)

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
