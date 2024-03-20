package asm

import (
	"fmt"

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
	function = transformReturn(arch, function)

	return function
}

func transformReturn(arch *config.Arch, function Function) Function {
	if function.Ret == nil {
		return function
	}

	offset := 8 * len(function.Params)
	//for _, param := range function.Params {
	//	offset += param.Size()
	//}
	op, ok := arch.CallOp[int8(function.Ret.Size())]
	if !ok {
		panic(fmt.Errorf("unable to store return value with size %d", function.Ret.Size()))
	}

	// FIXME: float return values (uses XMM registers on amd64)
	retInstr := fmt.Sprintf("%s %s, ret+%d(FP)", op, arch.RetRegister, offset)

	// we need to inject a new MOV instruction to move the return value to stack
	for i := 0; i < len(function.Lines); i++ {
		line := function.Lines[i]
		if line.Assembly == "ret" || line.Assembly == "retq" {
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
