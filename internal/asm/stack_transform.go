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
					// Only treat as stack allocation if it has SP writeback (pre-index mode)
					// Plain [sp, #N] (AddrOffset) is just a save, not an allocation
					if ok && imm.Base == arm64asm.RegSP(arm64asm.SP) && imm.Mode == arm64asm.AddrPreIndex && baseStack == 0 {
						n := immFromMemImmediate(imm)
						baseStack = -n
						extraStack = baseStack
					}
				} else if len(inst.Args) > 2 {
					imm, ok := inst.Args[2].(arm64asm.MemImmediate)
					// Only treat as stack allocation if it has SP writeback (pre-index mode)
					if ok && imm.Base == arm64asm.RegSP(arm64asm.SP) && imm.Mode == arm64asm.AddrPreIndex && baseStack == 0 {
						n := immFromMemImmediate(imm)
						baseStack = -n
						extraStack = baseStack
					}
					// this could still be fine, as long as it's doing just callee-saved registers
					rewriteRequired = true
				}
			case arm64asm.STR:
				imm, ok := inst.Args[1].(arm64asm.MemImmediate)
				// Only treat as stack allocation if it has SP writeback (pre-index mode)
				if ok && imm.Base == arm64asm.RegSP(arm64asm.SP) && imm.Mode == arm64asm.AddrPreIndex && baseStack == 0 {
					n := immFromMemImmediate(imm)
					baseStack = -n
					extraStack = baseStack
				}
				// this could still be fine, as long as it's doing just callee-saved registers
				rewriteRequired = true
			case arm64asm.MOV:
				// Check for dynamic stack allocation pattern: mov sp, <reg>
				// This is used by VLAs/alloca when computing new SP in a temp register
				// Pattern: mov x8, sp; sub x8, x8, x12; mov sp, x8
				// The only valid "mov sp, <reg>" is "mov sp, x29" for frame pointer restore
				targetReg := inst.Args[0]
				srcReg := inst.Args[1]
				if targetReg == arm64asm.RegSP(arm64asm.SP) {
					// Compare by string since RegSP type encodes X29 differently than arm64asm.X29
					if srcReg != nil && srcReg.String() != "X29" {
						// mov sp, <non-x29> indicates VLA/alloca - dynamic stack size
						panic(fmt.Sprintf("%s: dynamic stack allocation detected (mov sp, %v) - "+
							"alloca() and VLAs are not supported because Go requires stack size at compile time",
							function.Name, srcReg))
					}
				}
			case arm64asm.AND:
				// stack alignment
				// this basically grows the stack, need to adjust for it
				targetReg := inst.Args[0]
				if targetReg == arm64asm.RegSP(arm64asm.SP) {
					// allocating more stack space via alignment
					rewriteRequired = true
					complexManip = true
				}
			case arm64asm.SUB:
				// allocating stack space
				targetReg := inst.Args[0]
				srcReg := inst.Args[1]
				if targetReg == arm64asm.RegSP(arm64asm.SP) || srcReg == arm64asm.RegSP(arm64asm.SP) {
					complexManip = true
					// Check if this is dynamic stack allocation (register operand instead of immediate)
					// This happens with alloca() or VLAs - impossible to handle since Go needs
					// stack size known at compile time
					if len(parts) > 3 {
						imm := parts[3]
						imm = strings.TrimPrefix(imm, "#")
						if n, err := strconv.Atoi(imm); err == nil {
							extraStack += n
							rewriteRequired = true
						} else {
							// Not an immediate - this is dynamic stack allocation
							panic(fmt.Sprintf("%s: dynamic stack allocation detected (sub sp, sp, %s) - "+
								"alloca() and VLAs are not supported because Go requires stack size at compile time",
								function.Name, parts[3]))
						}
					}
				}
			}
			continue
		}
	}

	if !rewriteRequired {
		// No complex stack manipulation detected
		if extraStack == 0 {
			// Leaf function - no stack usage at all, nothing to transform
			return function
		}
		if extraStack != 16 {
			// Unexpected pattern - only expect simple 16-byte frame (x29,x30 save)
			panic(fmt.Sprintf("unexpected stack pattern: extraStack=%d, expected 0 or 16", extraStack))
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
		// First pass: check if x30 (LR) is used as a scratch register, and collect
		// callee-saved STP/LDP and STR/LDR info to distinguish actual prologue/epilogue
		// saves from data stores.
		//
		// The key insight is: only NOP callee-saved store/load pairs when BOTH exist
		// with the same (registers, offset). This is sound because:
		// - STP x21,x22 at offset 32 + LDP x21,x22 at offset 32 → register save → NOP
		// - STP x21,x22 at offset 32 + LDP x0,x1 at offset 32 → data store → KEEP
		// - STP x21,x22 at offset 16 + LDP x21,x22 at offset 32 → data movement → KEEP
		// - STP x21,x22 with no matching LDP → might be data, keep to be safe
		x30UsedAsScratch := false

		// calleeSaveInfo tracks whether a (regs, offset) slot has stores and/or loads
		type calleeSaveInfo struct {
			hasStore bool
			hasLoad  bool
		}
		calleeSaveSlots := make(map[calleeSaveKey]*calleeSaveInfo)

		for _, line := range function.Lines {
			if len(line.Binary) == 0 {
				continue // Skip labels, directives, etc.
			}
			inst := decodeArm64Line(line)

			// Check for LR scratch usage (but LR save/restore doesn't count as scratch)
			if !isLRStackSaveRestore(inst) && usesLRAsScratch(inst) {
				x30UsedAsScratch = true
			}

			// Collect callee-saved store/load info
			if key, ok := getCalleeSaveKey(inst); ok {
				info := calleeSaveSlots[key]
				if info == nil {
					info = &calleeSaveInfo{}
					calleeSaveSlots[key] = info
				}
				switch inst.Op {
				case arm64asm.STP, arm64asm.STR:
					info.hasStore = true
				case arm64asm.LDP, arm64asm.LDR:
					info.hasLoad = true
				}
			}
		}

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
				// Exception: if x30 is used as scratch, preserve its SP-based save/restore
				if x30UsedAsScratch && isLRStackSaveRestore(inst) {
					// Keep LR save/restore - it's used as scratch register
					lineCpy := line
					lineCpy.Disassembled = arm64asm.GoSyntax(inst, 0, nil, nil)
					lineCpy.Binary = nil
					newLines = append(newLines, lineCpy)
					continue
				}

				// Check if this is a callee-saved pair that should be NOPed.
				// Only NOP when BOTH a store AND load exist for the same (regs, offset).
				// This is the sound, conservative rule that avoids incorrectly NOPing data stores.
				if key, ok := getCalleeSaveKey(inst); ok {
					info := calleeSaveSlots[key]
					if info != nil && info.hasStore && info.hasLoad {
						doSkip = true
					}
				} else if inst.Op == arm64asm.MOV {
					// NOP frame pointer save (mov x29, sp) and restore (mov sp, x29)
					isFPSave := inst.Args[0] == arm64asm.RegSP(arm64asm.X29) && inst.Args[1] == arm64asm.RegSP(arm64asm.SP)
					isFPRestore := inst.Args[0] == arm64asm.RegSP(arm64asm.SP) && inst.Args[1] == arm64asm.RegSP(arm64asm.X29)
					if isFPSave || isFPRestore {
						doSkip = true
					}
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
		
		// First pass: check if x30 (LR) is used as a scratch register.
		// If so, we must preserve its save/restore to maintain return address.
		// Note: We check ALL instructions, not just non-SP ones, because clang may
		// emit instructions like "add x30, sp, #32" that write to LR while referencing SP.
		x30UsedAsScratch := false
		for _, line := range function.Lines {
			if len(line.Binary) == 0 {
				continue // Skip labels, directives, etc.
			}
			inst := decodeArm64Line(line)
			
			// Skip LR save/restore instructions - these don't count as "scratch use"
			if isLRStackSaveRestore(inst) {
				continue
			}
			
			// If this instruction writes to LR, it's using LR as scratch
			if usesLRAsScratch(inst) {
				x30UsedAsScratch = true
				break
			}
		}
		
		newLines := make([]Line, 0, len(function.Lines))

		for _, line := range function.Lines {
			asm := line.Assembly
			if spInstruction.MatchString(asm) {
				inst := decodeArm64Line(line)

				// If x30 is used as scratch, preserve its SP-based save/restore
				if x30UsedAsScratch && isLRStackSaveRestore(inst) {
					lineCpy := line
					lineCpy.Disassembled = arm64asm.GoSyntax(inst, 0, nil, nil)
					lineCpy.Binary = nil
					newLines = append(newLines, lineCpy)
					continue
				}

				// NOP callee-saved register pairs - Go's ABI0 doesn't require them
				if isCalleeSavedRegPair(inst) {
					lineCpy := line
					lineCpy.Disassembled = "NOP"
					lineCpy.Binary = nil
					newLines = append(newLines, lineCpy)
					continue
				}

				// NOP frame pointer operations:
				// - mov x29, sp (prologue: save SP to frame pointer)
				// - mov sp, x29 (epilogue: restore SP from frame pointer)
				// Both must be NOPed together to avoid stack corruption
				if inst.Op == arm64asm.MOV {
					isFPSave := inst.Args[0] == arm64asm.RegSP(arm64asm.X29) && inst.Args[1] == arm64asm.RegSP(arm64asm.SP)
					isFPRestore := inst.Args[0] == arm64asm.RegSP(arm64asm.SP) && inst.Args[1] == arm64asm.RegSP(arm64asm.X29)
					if isFPSave || isFPRestore {
						lineCpy := line
						lineCpy.Disassembled = "NOP"
						lineCpy.Binary = nil
						newLines = append(newLines, lineCpy)
						continue
					}
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

// calleeSaveKey identifies a callee-saved register store/load by its registers and stack offset.
// Used to match STP/STR with corresponding LDP/LDR to distinguish prologue/epilogue saves
// from data stores that happen to use callee-saved registers.
type calleeSaveKey struct {
	regs   string // Normalized register pair (e.g., "X19X20" or "X19" for single reg)
	offset int    // Stack offset (0 for pre/post-index modes that implicitly use offset 0)
}

// getCalleeSaveKey extracts the key for matching callee-saved register store/load pairs.
// Returns (key, true) if the instruction is a callee-saved STP/LDP/STR/LDR to SP.
// Returns (zero, false) otherwise.
func getCalleeSaveKey(inst arm64asm.Inst) (calleeSaveKey, bool) {
	switch inst.Op {
	case arm64asm.STP, arm64asm.LDP:
		if !isCalleeSavedRegPair(inst) {
			return calleeSaveKey{}, false
		}
		// Get registers in normalized order (smaller register first)
		r0 := inst.Args[0].(arm64asm.Reg)
		r1 := inst.Args[1].(arm64asm.Reg)
		if r0 > r1 {
			r0, r1 = r1, r0
		}
		regs := r0.String() + r1.String()

		// Get offset from memory operand
		mem := inst.Args[2].(arm64asm.MemImmediate)
		offset := immFromMemImmediate(mem)
		// For pre/post-index modes, the effective stack slot is at offset 0
		// (pre-index: stores at [sp+imm], then sp+=imm; post-index: loads from [sp], then sp+=imm)
		if mem.Mode == arm64asm.AddrPreIndex || mem.Mode == arm64asm.AddrPostIndex {
			offset = 0
		}

		return calleeSaveKey{regs: regs, offset: offset}, true

	case arm64asm.STR, arm64asm.LDR:
		if !isCalleeSavedReg(inst.Args[0]) {
			return calleeSaveKey{}, false
		}
		// Check if SP-based
		mem, ok := inst.Args[1].(arm64asm.MemImmediate)
		if !ok || mem.Base != arm64asm.RegSP(arm64asm.SP) {
			return calleeSaveKey{}, false
		}
		reg := inst.Args[0].(arm64asm.Reg)
		regs := reg.String()
		offset := immFromMemImmediate(mem)
		if mem.Mode == arm64asm.AddrPreIndex || mem.Mode == arm64asm.AddrPostIndex {
			offset = 0
		}

		return calleeSaveKey{regs: regs, offset: offset}, true
	}

	return calleeSaveKey{}, false
}

// isLR returns true if the argument is the link register (x30 or w30).
//
// Per AAPCS64 (Procedure Call Standard for the Arm 64-bit Architecture):
// - x30 is the Link Register (LR) holding the return address
// - x30 is NOT callee-saved; the callee may use it as a scratch register
// - If x30 is clobbered, it must be saved/restored to preserve the return address
// - RET instruction implicitly uses x30 as the return address
//
// See: https://github.com/ARM-software/abi-aa/blob/main/aapcs64/aapcs64.rst
// See: https://developer.arm.com/documentation/102374/latest/ (Table 2: General-purpose registers)
func isLR(arg arm64asm.Arg) bool {
	// Check for arm64asm.Reg type
	if reg, ok := arg.(arm64asm.Reg); ok {
		return reg == arm64asm.X30 || reg == arm64asm.W30
	}
	// Check for arm64asm.RegSP type (decoder uses this for X29/X30/W29/W30 in some contexts)
	if regSP, ok := arg.(arm64asm.RegSP); ok {
		return regSP == arm64asm.RegSP(arm64asm.X30) || regSP == arm64asm.RegSP(arm64asm.W30)
	}
	return false
}

// regPairContainsLR returns true if the STP/LDP instruction involves the link register.
func regPairContainsLR(inst arm64asm.Inst) bool {
	if inst.Op != arm64asm.STP && inst.Op != arm64asm.LDP {
		return false
	}
	return isLR(inst.Args[0]) || isLR(inst.Args[1])
}

// memBaseIsSP returns true if the instruction's memory operand uses SP as base.
func memBaseIsSP(inst arm64asm.Inst) bool {
	for _, arg := range inst.Args {
		if mem, ok := arg.(arm64asm.MemImmediate); ok {
			return mem.Base == arm64asm.RegSP(arm64asm.SP)
		}
	}
	return false
}

// isLRStackSaveRestore returns true if the instruction is saving/restoring LR to/from stack.
func isLRStackSaveRestore(inst arm64asm.Inst) bool {
	switch inst.Op {
	case arm64asm.STP, arm64asm.LDP:
		return regPairContainsLR(inst) && memBaseIsSP(inst)
	case arm64asm.STR, arm64asm.LDR:
		return isLR(inst.Args[0]) && memBaseIsSP(inst)
	default:
		return false
	}
}

// usesLRAsScratch returns true if the instruction uses LR (x30/w30) as a general-purpose
// scratch register. This excludes:
// - RET: reads LR for return address (doesn't modify it)
// - BL/BLR: writes LR as part of call semantics (not scratch use)
// - STR/STP: reads LR to store to memory (doesn't write to it)
// - LDR/LDP with LR as destination: handled by isLRStackSaveRestore
//
// If any instruction uses LR as scratch, we must preserve its prologue/epilogue save/restore.
func usesLRAsScratch(inst arm64asm.Inst) bool {
	if len(inst.Args) == 0 {
		return false
	}
	switch inst.Op {
	case arm64asm.RET:
		return false // reads LR, doesn't write
	case arm64asm.BL, arm64asm.BLR:
		return false // call semantics, not scratch use
	case arm64asm.STR, arm64asm.STP:
		return false // stores register value to memory, doesn't write to register
	case arm64asm.LDR, arm64asm.LDP:
		return false // loads are stack save/restore, handled separately
	}
	// Most A64 instructions put the destination in Args[0]
	return isLR(inst.Args[0])
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
