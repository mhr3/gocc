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
	function = checkStackManipulation(arch, function)
	function = storeReturnValue(arch, function)

	// weird type of transform, but we'll keep it here for now
	function = removeBinaryInstructions(arch, function)

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

		if arch.JumpInstr != nil && arch.JumpInstr.MatchString(combined) {
			reParams := getRegexpParams(arch.JumpInstr, combined)
			rewritten := fmt.Sprintf("%s %s", strings.ToUpper(reParams["instr"]), reParams["label"])
			function.Lines[i].Disassembled = rewritten
			function.Lines[i].Binary = nil
		} else if arch.DataLoad != nil && arch.DataLoad.MatchString(combined) {
			reParams := getRegexpParams(arch.DataLoad, combined)
			// FIXME: this is extremely fragile
			op := arch.MovInstr[8]
			register := reParams["register"]
			symbol := reParams["var"]
			addrMode := "$" // absolute addressing
			if instr, ok := reParams["instr"]; ok {
				// sigh, why oh why do we have to do this?
				switch {
				case instr == "MOVDQA":
					op = "MOVO"
				case instr == "MOVDQU":
					op = "MOVOU"
				default:
					op = instr
				}

				addrMode = ""
			}
			if strings.HasPrefix(register, "X") {
				// the disassembler gets this wrong sometimes
				switch {
				case strings.Contains(line.Assembly, "xmm"):
					// all good
				case strings.Contains(line.Assembly, "ymm"):
					register = "Y" + register[1:]
				case strings.Contains(line.Assembly, "zmm"):
					register = "Z" + register[1:] // does go even support this?
				}
			}
			rewritten := fmt.Sprintf("%s %s%s<>(SB), %s", op, addrMode, symbol, register)
			function.Lines[i].Disassembled = rewritten
			function.Lines[i].Binary = nil
		}
	}

	return function
}

func checkStackManipulation(arch *config.Arch, function Function) Function {
	if arch == nil {
		return function
	}

	switch arch.Name {
	case "arm64":
		return checkStackArm64(arch, function)
	case "amd64":
		return checkStackAmd64(arch, function)
	}

	return function
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
