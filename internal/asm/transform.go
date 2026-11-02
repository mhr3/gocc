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
		functions = cloneSharedInternalHelpers(functions)
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
		transformed, err := transformFunction(arch, function)
		if err != nil {
			return nil, err
		}
		functions[i] = transformed
	}
	if arch != nil {
		var err error
		functions, err = reserveInternalStackFrames(arch, functions)
		if err != nil {
			return nil, err
		}
		if arch.Name == "amd64" {
			functions = realignAmd64Roots(functions)
		}
	}

	return functions, nil
}

// realignAmd64Roots makes a Go-visible AMD64 function realign SP when its
// code, or the code of a helper it owns, depends on stack alignment that Go
// does not provide (see amd64RequiredStackAlign). Writing SP marks the
// function SPWRITE for the Go toolchain, which disables asynchronous
// preemption inside it, so functions that do not need it are left alone.
// The prologue is emitted by Generate (after the argument loads); the
// original SP is restored here before each return so the result store and
// Go's own epilogue see the frame the assembler laid out.
func realignAmd64Roots(functions []Function) []Function {
	for i := range functions {
		function := &functions[i]
		if function.Internal {
			continue
		}
		if required := amd64RequiredStackAlign(function); required > function.StackAlign {
			function.StackAlign = required
		}
		if function.StackAlign == 0 {
			continue
		}
		restore := Line{Disassembled: fmt.Sprintf("MOVQ %d(SP), SP", function.LocalsSize), Comment: "restore SP realigned in the prologue"}
		lines := make([]Line, 0, len(function.Lines)+4)
		for j, line := range function.Lines {
			if !strings.HasPrefix(line.Assembly, "ret") {
				lines = append(lines, line)
				continue
			}
			// storeReturnValue places the result store directly before RET and
			// moved the labels onto it.
			if n := len(lines); j > 0 && n > 0 && isReturnValueStore(lines[n-1]) {
				store := lines[n-1]
				lines = lines[:n-1]
				restore.Labels, store.Labels = store.Labels, nil
				lines = append(lines, restore, store, line)
				continue
			}
			restore.Labels, line.Labels = line.Labels, nil
			lines = append(lines, restore, line)
		}
		function.Lines = lines
	}
	return functions
}

func isReturnValueStore(line Line) bool {
	return line.Assembly == "" && line.Binary == nil && strings.Contains(line.Disassembled, " ret+") && strings.HasSuffix(line.Disassembled, "(FP)")
}

// cloneSharedInternalHelpers gives every Go-visible function its own copy of
// the internal helpers it shares with other Go-visible functions. Helper stack
// slots live in a fixed arena inside one root's frame, so a helper can only
// serve one root; cloning keeps the common C pattern of helpers shared by
// several entry points working at the cost of duplicated code.
func cloneSharedInternalHelpers(functions []Function) []Function {
	index := make(map[string]int, len(functions))
	for i := range functions {
		index[functions[i].Name] = i
	}
	reachable := func(from int) []int {
		var order []int
		seen := map[int]bool{}
		var visit func(int)
		visit = func(i int) {
			for _, line := range functions[i].Lines {
				target, ok := BranchTarget(line)
				if !ok {
					continue
				}
				j, exists := index[target]
				if !exists || !functions[j].Internal || seen[j] {
					continue
				}
				seen[j] = true
				order = append(order, j)
				visit(j)
			}
		}
		visit(from)
		return order
	}

	rootsOf := make(map[int][]int) // helper -> roots reaching it
	reach := make(map[int][]int)   // root -> reachable helpers
	for i := range functions {
		if functions[i].Internal {
			continue
		}
		reach[i] = reachable(i)
		for _, h := range reach[i] {
			rootsOf[h] = append(rootsOf[h], i)
		}
	}
	shared := make(map[int]bool)
	for h, roots := range rootsOf {
		if len(roots) > 1 {
			shared[h] = true
		}
	}
	if len(shared) == 0 {
		return functions
	}

	retarget := func(lines []Line, rename map[string]string) []Line {
		out := make([]Line, len(lines))
		for i, line := range lines {
			out[i] = line
			if target, ok := BranchTarget(line); ok {
				if to, renamed := rename[target]; renamed {
					out[i].Disassembled = strings.Fields(line.Disassembled)[0] + " " + to + "<>(SB)"
				}
			}
		}
		return out
	}

	result := make([]Function, 0, len(functions))
	for i := range functions {
		if !shared[i] {
			result = append(result, functions[i])
		}
	}
	consts := make(map[int]bool)
	for rootIdx := range functions {
		if functions[rootIdx].Internal {
			continue
		}
		rename := make(map[string]string)
		for _, h := range reach[rootIdx] {
			if shared[h] {
				rename[functions[h].Name] = functions[h].Name + "_" + functions[rootIdx].Name
			}
		}
		if len(rename) == 0 {
			continue
		}
		// Exclusive helpers of this root are rewritten in place; shared ones
		// are cloned under the root-specific name.
		for j := range result {
			if result[j].Name == functions[rootIdx].Name || (result[j].Internal && len(rootsOf[index[result[j].Name]]) == 1 && rootsOf[index[result[j].Name]][0] == rootIdx) {
				result[j].Lines = retarget(result[j].Lines, rename)
			}
		}
		for _, h := range reach[rootIdx] {
			if !shared[h] {
				continue
			}
			clone := functions[h]
			clone.Name = rename[clone.Name]
			clone.Lines = retarget(clone.Lines, rename)
			if consts[h] {
				clone.Consts = nil // constants are emitted once, by the first clone
			}
			consts[h] = true
			result = append(result, clone)
		}
	}
	return result
}

func rejectInternalStackArguments(arch *config.Arch, function Function) error {
	if arch.Name != "amd64" && arch.Name != "arm64" {
		return nil
	}

	archInfo := getArchStackInfo(arch)
	layout, err := analyzeStackLayout(archInfo, function.Lines, true)
	if err != nil {
		return fmt.Errorf("%s: %w", function.Name, err)
	}
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
			inst, ok := tryDecodeAmd64Line(line)
			if !ok {
				continue
			}
			for _, arg := range inst.Args {
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
			target, ok := BranchTarget(line)
			if !ok {
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

func transformFunction(arch *config.Arch, function Function) (Function, error) {
	// Apply the transforms
	function = expandTailCalls(function)
	function = transformReturns(arch, function)
	function = rewriteJumpsAndLoads(arch, function)
	function, err := checkStackManipulation(arch, function)
	if err != nil {
		return function, fmt.Errorf("%s: %w", function.Name, err)
	}
	function = storeReturnValue(arch, function)

	// weird type of transform, but we'll keep it here for now
	if !(arch != nil && arch.Name == "arm64" && function.Internal) {
		// Internal ARM64 helpers retain raw encodings where Plan 9 assembly lacks
		// an exact spelling. Their stack references were already rebased above.
		function = removeBinaryInstructions(arch, function)
	}

	return function, nil
}

// expandTailCalls turns a Go-visible function's tail calls into a call plus
// return. The C compiler had already torn down the frame before the tail
// call, so the helper runs with the root's fixed frame intact, and the usual
// return bridge stores the helper's result into the Go result slot. Internal
// helpers keep their tail calls: they share the C register ABI with their
// callee and the caller's return address is still valid.
func expandTailCalls(function Function) Function {
	if function.Internal {
		return function
	}
	expanded := make([]Line, 0, len(function.Lines))
	for _, line := range function.Lines {
		target, ok := BranchTarget(line)
		if !ok || !strings.HasPrefix(line.Disassembled, "JMP ") {
			expanded = append(expanded, line)
			continue
		}
		call := line
		call.Disassembled = fmt.Sprintf("CALL %s<>(SB)", target)
		expanded = append(expanded, call, Line{Assembly: "ret", Disassembled: "RET"})
	}
	function.Lines = expanded
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

func checkStackManipulation(arch *config.Arch, function Function) (Function, error) {
	if arch == nil {
		return function, nil
	}

	switch arch.Name {
	case "amd64", "arm64":
		if function.Internal {
			// Internal helpers use the C register ABI, so they cannot safely run a
			// Go morestack prologue. Flatten their C frames into fixed slots that a
			// Go-visible caller reserves and expose the measured size for the global
			// reservation pass.
			archInfo := getArchStackInfo(arch)
			layout, err := analyzeStackLayout(archInfo, function.Lines, true)
			if err != nil {
				return function, err
			}
			function = rewriteStackOpsWithLinkage(arch, archInfo, layout, function, false)
			function.HiddenStackSize = function.LocalsSize
			function.LocalsSize = 0
			return function, nil
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
