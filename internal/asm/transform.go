package asm

import (
	"fmt"
	"strings"

	"github.com/mhr3/gocc/internal/config"
	"golang.org/x/arch/arm64/arm64asm"
	"golang.org/x/arch/x86/x86asm"
)

func ApplyTransforms(arch *config.Arch, functions []Function) ([]Function, error) {
	if arch != nil {
		if err := rejectRecursiveInternalCalls(functions); err != nil {
			return nil, err
		}
		for _, function := range functions {
			if function.Internal {
				if err := rejectInternalStackArguments(arch, function); err != nil {
					return nil, err
				}
			}
		}
	}
	for i, function := range functions {
		functions[i] = transformFunction(arch, function)
	}
	if arch != nil {
		var err error
		functions, err = reserveInternalStackFrames(arch, functions)
		if err != nil {
			return nil, err
		}
	}

	return functions, nil
}

func rejectInternalStackArguments(arch *config.Arch, function Function) error {
	if arch.Name != "amd64" && arch.Name != "arm64" {
		return nil
	}

	archInfo := getArchStackInfo(arch)
	layout := analyzeStackLayout(archInfo, function.Lines, true)
	firstStackArgument := int64(layout.LocalsSize)
	if arch.Name == "amd64" {
		firstStackArgument += 8 // Skip x86 CALL's return address.
	}
	for _, line := range function.Lines {
		if len(line.Binary) == 0 {
			continue
		}
		usesStackArgument := false
		switch arch.Name {
		case "amd64":
			if !strings.Contains(line.Assembly, "rsp") {
				continue
			}
			for _, arg := range decodeAmd64Line(line).Args {
				memory, ok := arg.(x86asm.Mem)
				if ok && memory.Base == x86asm.RSP && memory.Disp >= firstStackArgument {
					usesStackArgument = true
					break
				}
			}
		case "arm64":
			if !strings.Contains(line.Assembly, "sp") {
				continue
			}
			for _, arg := range decodeArm64Line(line).Args {
				memory, ok := arg.(arm64asm.MemImmediate)
				if ok && memory.Base == arm64asm.RegSP(arm64asm.SP) && memory.Mode == arm64asm.AddrOffset &&
					int64(immFromMemImmediate(memory)) >= firstStackArgument {
					usesStackArgument = true
					break
				}
			}
		}
		if usesStackArgument {
			return fmt.Errorf(
				"internal helper %q uses stack-passed C arguments, which are unsupported",
				function.Name,
			)
		}
	}
	return nil
}

func rejectRecursiveInternalCalls(functions []Function) error {
	internal := make(map[string]int)
	for i := range functions {
		if functions[i].Internal {
			internal[functions[i].Name] = i
		}
	}

	edges := make(map[string][]string, len(internal))
	for name, functionIdx := range internal {
		for _, line := range functions[functionIdx].Lines {
			fields := strings.Fields(line.Disassembled)
			if len(fields) != 2 || fields[0] != "CALL" {
				continue
			}
			target := strings.TrimSuffix(fields[1], "<>(SB)")
			if target == fields[1] {
				continue
			}
			if _, ok := internal[target]; ok {
				edges[name] = append(edges[name], target)
			}
		}
	}

	const (
		unvisited = iota
		visiting
		visited
	)
	state := make(map[string]int, len(internal))
	path := make([]string, 0, len(internal))
	pathIndex := make(map[string]int, len(internal))
	var visit func(string) error
	visit = func(name string) error {
		state[name] = visiting
		pathIndex[name] = len(path)
		path = append(path, name)
		for _, target := range edges[name] {
			switch state[target] {
			case unvisited:
				if err := visit(target); err != nil {
					return err
				}
			case visiting:
				start := pathIndex[target]
				cycle := append(append([]string(nil), path[start:]...), target)
				return fmt.Errorf("recursive internal helper call graph: %s", strings.Join(cycle, " -> "))
			}
		}
		path = path[:len(path)-1]
		delete(pathIndex, name)
		state[name] = visited
		return nil
	}

	// Walk in source order to keep diagnostics deterministic.
	for i := range functions {
		name := functions[i].Name
		if !functions[i].Internal || state[name] != unvisited {
			continue
		}
		if err := visit(name); err != nil {
			return err
		}
	}
	return nil
}

func transformFunction(arch *config.Arch, function Function) Function {
	// Apply the transforms
	function = transformReturns(arch, function)
	function = rewriteJumpsAndLoads(arch, function)
	function = checkStackManipulation(arch, function)
	function = storeReturnValue(arch, function)

	// weird type of transform, but we'll keep it here for now
	if !(arch != nil && arch.Name == "arm64" && function.Internal) {
		// Internal ARM64 helpers retain raw encodings where Plan 9 assembly lacks
		// an exact spelling. Their stack references were already rebased above.
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
	case "amd64", "arm64":
		if function.Internal {
			// Internal helpers use the C register ABI, so they cannot safely run a
			// Go morestack prologue. Flatten their C frames into fixed slots that a
			// Go-visible caller reserves and expose the measured size for the global
			// reservation pass.
			archInfo := getArchStackInfo(arch)
			layout := analyzeStackLayout(archInfo, function.Lines, true)
			function = rewriteStackOpsWithLinkage(arch, archInfo, layout, function, false)
			function.HiddenStackSize = function.LocalsSize
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
