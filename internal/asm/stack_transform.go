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

// StackOpKind represents the type of stack-related operation
type StackOpKind int

const (
	StackOpNone          StackOpKind = iota
	StackOpPush                      // push reg (AMD64) or str/stp with pre-decrement (ARM64)
	StackOpPop                       // pop reg (AMD64) or ldr/ldp with post-increment (ARM64)
	StackOpAlloc                     // sub rsp, N (AMD64) or sub sp, sp, N (ARM64)
	StackOpDealloc                   // add rsp, N (AMD64) or add sp, sp, N (ARM64)
	StackOpAlign                     // and rsp, -N (AMD64) or and sp, xN, -N (ARM64)
	StackOpAlignPrep                 // sub xN, sp, #M (ARM64) - preparation for alignment, rewrite to MOV
	StackOpFrameSetup                // mov rbp, rsp (AMD64) or mov x29, sp (ARM64)
	StackOpFrameTeardown             // mov rsp, rbp (AMD64) or mov sp, x29 (ARM64)
)

// StackOp represents a parsed stack-related operation
type StackOp struct {
	Kind      StackOpKind // Type of stack operation
	Reg       string      // Register involved (if any)
	Reg2      string      // Second register (for stp/ldp)
	Size      int         // Size of operation in bytes
	Offset    int         // Offset for memory operations
	Immediate int64       // Immediate value (for sub/and)
	LineIndex int         // Index of the line in the function
}

// SavedReg represents a register saved to the stack
type SavedReg struct {
	Reg           string // Architecture-neutral name (e.g., "R12", "X19")
	COffset       int    // Original C offset from SP after push
	IsCalleeSaved bool   // Can be removed in Go (no callee-saved regs in Go ABI0)
}

// StackSlot represents a named stack slot in Go assembly
type StackSlot struct {
	Name     string // Symbolic name for Go asm (e.g., "local", "spill")
	COffset  int    // Original C offset
	GoOffset int    // Go offset for name-N(SP) format
	Size     int    // Size in bytes
}

// StackLayout captures the C function's stack frame structure
type StackLayout struct {
	// Frame pointer setup detected
	FramePointerUsed bool

	// Registers saved to stack (callee-saved in C, not needed in Go)
	SavedRegs []SavedReg

	// Stack space allocated via sub rsp, N / stp with pre-decrement
	LocalsSize int

	// Alignment requirement detected (from and sp, -N)
	// Go can only guarantee 8-byte alignment, so we'll use unaligned instructions
	Alignment int64

	// Total frame size needed for Go (declared in TEXT $N-M)
	GoFrameSize int

	// Named stack slots for Go output
	Slots []StackSlot

	// Indices of lines that should be NOPed out
	NopIndices map[int]bool

	// Indices of lines with stack allocation (sub rsp, N)
	AllocIndices map[int]bool
}

// ArchStackInfo provides architecture-specific stack details
type ArchStackInfo interface {
	// Name returns the architecture name ("amd64" or "arm64")
	Name() string

	// FramePointerReg returns the frame pointer register name (RBP, X29)
	FramePointerReg() string

	// StackPointerReg returns the stack pointer register name (RSP, SP)
	StackPointerReg() string

	// LinkReg returns the link register name (empty for AMD64, X30 for ARM64)
	LinkReg() string

	// IsCalleeSaved checks if a register is callee-saved in C ABI
	IsCalleeSaved(reg string) bool

	// ParseStackOp parses a line and returns a StackOp if it's stack-related
	ParseStackOp(idx int, line Line) *StackOp

	// PtrSize returns the pointer size (8 for 64-bit)
	PtrSize() int

	// ToUnalignedInsn converts an aligned instruction to its unaligned equivalent
	// Returns nil if the instruction doesn't need conversion
	ToUnalignedInsn(insn string) *string

	// SPRegex returns a regex that matches the stack pointer register
	SPRegex() *regexp.Regexp

	// HasStackMemoryRef returns true if the operands contain a stack-relative memory reference
	HasStackMemoryRef(operands string) bool
}

// amd64StackInfo implements ArchStackInfo for AMD64
type amd64StackInfo struct {
	spRegex    *regexp.Regexp
	spMemRegex *regexp.Regexp // matches stack memory references
}

func newAmd64StackInfo() *amd64StackInfo {
	return &amd64StackInfo{
		spRegex: regexp.MustCompile(`\brsp\b`),
		// Matches Go syntax: (SP) or x86 syntax: [rsp...]
		spMemRegex: regexp.MustCompile(`\(SP\)|\[rsp[^\]]*\]`),
	}
}

func (a *amd64StackInfo) Name() string                    { return "amd64" }
func (a *amd64StackInfo) FramePointerReg() string         { return "rbp" }
func (a *amd64StackInfo) StackPointerReg() string         { return "rsp" }
func (a *amd64StackInfo) LinkReg() string                 { return "" }
func (a *amd64StackInfo) PtrSize() int                    { return 8 }
func (a *amd64StackInfo) SPRegex() *regexp.Regexp         { return a.spRegex }
func (a *amd64StackInfo) HasStackMemoryRef(s string) bool { return a.spMemRegex.MatchString(s) }

func (a *amd64StackInfo) IsCalleeSaved(reg string) bool {
	switch strings.ToLower(reg) {
	case "rbp", "rbx", "r12", "r13", "r14", "r15",
		"bp", "bx": // Go register names
		return true
	}
	return false
}

// amd64AlignedToUnaligned maps aligned SIMD instructions to unaligned equivalents
var amd64AlignedToUnaligned = map[string]string{
	// SSE
	"MOVAPS": "MOVUPS",
	"MOVAPD": "MOVUPD",
	"MOVDQA": "MOVDQU",
	"movaps": "movups",
	"movapd": "movupd",
	"movdqa": "movdqu",
	// AVX
	"VMOVAPS": "VMOVUPS",
	"VMOVAPD": "VMOVUPD",
	"VMOVDQA": "VMOVDQU",
	"vmovaps": "vmovups",
	"vmovapd": "vmovupd",
	"vmovdqa": "vmovdqu",
	// AVX-512
	"VMOVDQA32": "VMOVDQU32",
	"VMOVDQA64": "VMOVDQU64",
	"vmovdqa32": "vmovdqu32",
	"vmovdqa64": "vmovdqu64",
}

func (a *amd64StackInfo) ToUnalignedInsn(insn string) *string {
	if unaligned, ok := amd64AlignedToUnaligned[insn]; ok {
		return &unaligned
	}
	return nil
}

func (a *amd64StackInfo) ParseStackOp(idx int, line Line) *StackOp {
	asm := line.Assembly
	if asm == "" {
		return nil
	}

	fields := strings.Fields(asm)
	if len(fields) == 0 {
		return nil
	}

	instr := strings.ToLower(fields[0])

	// Handle push/pop
	if instr == "push" {
		inst := decodeAmd64Line(line)
		if inst.Op != x86asm.PUSH {
			return nil
		}
		reg, ok := inst.Args[0].(x86asm.Reg)
		if !ok {
			return nil
		}
		return &StackOp{
			Kind:      StackOpPush,
			Reg:       strings.ToLower(reg.String()),
			Size:      8,
			LineIndex: idx,
		}
	}

	if instr == "pop" {
		inst := decodeAmd64Line(line)
		if inst.Op != x86asm.POP {
			return nil
		}
		reg, ok := inst.Args[0].(x86asm.Reg)
		if !ok {
			return nil
		}
		return &StackOp{
			Kind:      StackOpPop,
			Reg:       strings.ToLower(reg.String()),
			Size:      8,
			LineIndex: idx,
		}
	}

	// Check if instruction involves SP
	if !a.spRegex.MatchString(asm) {
		return nil
	}

	// Handle mov rbp, rsp (frame setup)
	if strings.HasPrefix(instr, "mov") && strings.Contains(asm, "rbp") {
		inst := decodeAmd64Line(line)
		if inst.Op == x86asm.MOV {
			dst, dstOk := inst.Args[0].(x86asm.Reg)
			src, srcOk := inst.Args[1].(x86asm.Reg)
			if dstOk && srcOk {
				if dst == x86asm.RBP && src == x86asm.RSP {
					return &StackOp{Kind: StackOpFrameSetup, LineIndex: idx}
				}
				if dst == x86asm.RSP && src == x86asm.RBP {
					return &StackOp{Kind: StackOpFrameTeardown, LineIndex: idx}
				}
			}
		}
	}

	// Handle sub rsp, N (stack allocation)
	if instr == "sub" {
		inst := decodeAmd64Line(line)
		if inst.Op == x86asm.SUB {
			dst, dstOk := inst.Args[0].(x86asm.Reg)
			imm, immOk := inst.Args[1].(x86asm.Imm)
			if dstOk && immOk && dst == x86asm.RSP {
				return &StackOp{
					Kind:      StackOpAlloc,
					Immediate: int64(imm),
					LineIndex: idx,
				}
			}
		}
	}

	// Handle add rsp, N (stack deallocation)
	if instr == "add" {
		inst := decodeAmd64Line(line)
		if inst.Op == x86asm.ADD {
			dst, dstOk := inst.Args[0].(x86asm.Reg)
			imm, immOk := inst.Args[1].(x86asm.Imm)
			if dstOk && immOk && dst == x86asm.RSP {
				return &StackOp{
					Kind:      StackOpDealloc,
					Immediate: int64(imm),
					LineIndex: idx,
				}
			}
		}
	}

	// Handle and rsp, -N (stack alignment)
	if instr == "and" {
		inst := decodeAmd64Line(line)
		if inst.Op == x86asm.AND {
			dst, dstOk := inst.Args[0].(x86asm.Reg)
			imm, immOk := inst.Args[1].(x86asm.Imm)
			if dstOk && immOk && dst == x86asm.RSP {
				return &StackOp{
					Kind:      StackOpAlign,
					Immediate: int64(imm),
					LineIndex: idx,
				}
			}
		}
	}

	// Handle lea rsp, [rbp - N] (frame teardown variant)
	if instr == "lea" {
		inst := decodeAmd64Line(line)
		if inst.Op == x86asm.LEA {
			dst, dstOk := inst.Args[0].(x86asm.Reg)
			if dstOk && dst == x86asm.RSP {
				return &StackOp{Kind: StackOpFrameTeardown, LineIndex: idx}
			}
		}
	}

	return nil
}

// arm64StackInfo implements ArchStackInfo for ARM64
type arm64StackInfo struct {
	spRegex    *regexp.Regexp
	spMemRegex *regexp.Regexp // matches stack memory references
}

func newArm64StackInfo() *arm64StackInfo {
	return &arm64StackInfo{
		spRegex: regexp.MustCompile(`\bsp\b`),
		// Matches Go syntax: (RSP) or (SP), or ARM syntax: [sp...]
		spMemRegex: regexp.MustCompile(`\(R?SP\)|\[sp[^\]]*\]`),
	}
}

func (a *arm64StackInfo) Name() string                    { return "arm64" }
func (a *arm64StackInfo) FramePointerReg() string         { return "x29" }
func (a *arm64StackInfo) StackPointerReg() string         { return "sp" }
func (a *arm64StackInfo) LinkReg() string                 { return "x30" }
func (a *arm64StackInfo) PtrSize() int                    { return 8 }
func (a *arm64StackInfo) SPRegex() *regexp.Regexp         { return a.spRegex }
func (a *arm64StackInfo) HasStackMemoryRef(s string) bool { return a.spMemRegex.MatchString(s) }

func (a *arm64StackInfo) IsCalleeSaved(reg string) bool {
	switch strings.ToLower(reg) {
	case "x19", "x20", "x21", "x22", "x23", "x24",
		"x25", "x26", "x27", "x28", "x29", "x30":
		return true
	}
	return false
}

func (a *arm64StackInfo) ToUnalignedInsn(insn string) *string {
	// ARM64 NEON doesn't require alignment for most instructions
	// SVE might, but we'll handle that if needed
	return nil
}

func (a *arm64StackInfo) ParseStackOp(idx int, line Line) *StackOp {
	asm := line.Assembly
	if asm == "" {
		return nil
	}

	// Check if instruction involves SP
	if !a.spRegex.MatchString(asm) {
		return nil
	}

	inst := decodeArm64Line(line)
	fields := strings.Fields(asm)
	if len(fields) == 0 {
		return nil
	}

	switch inst.Op {
	case arm64asm.STP:
		// Check for frame pointer save: stp x29, x30, [sp, #-N]!
		if len(inst.Args) >= 3 {
			reg1, ok1 := inst.Args[0].(arm64asm.Reg)
			reg2, ok2 := inst.Args[1].(arm64asm.Reg)
			mem, memOk := inst.Args[2].(arm64asm.MemImmediate)
			if ok1 && ok2 && memOk && mem.Base == arm64asm.RegSP(arm64asm.SP) {
				imm := immFromMemImmediate(mem)
				isPreIndex := mem.Mode == arm64asm.AddrPreIndex
				return &StackOp{
					Kind:      StackOpPush,
					Reg:       strings.ToLower(reg1.String()),
					Reg2:      strings.ToLower(reg2.String()),
					Size:      16,
					Offset:    imm,
					Immediate: int64(-imm) * boolToInt(isPreIndex),
					LineIndex: idx,
				}
			}
		}

	case arm64asm.LDP:
		// Check for frame pointer restore: ldp x29, x30, [sp], #N
		if len(inst.Args) >= 3 {
			reg1, ok1 := inst.Args[0].(arm64asm.Reg)
			reg2, ok2 := inst.Args[1].(arm64asm.Reg)
			mem, memOk := inst.Args[2].(arm64asm.MemImmediate)
			if ok1 && ok2 && memOk && mem.Base == arm64asm.RegSP(arm64asm.SP) {
				imm := immFromMemImmediate(mem)
				return &StackOp{
					Kind:      StackOpPop,
					Reg:       strings.ToLower(reg1.String()),
					Reg2:      strings.ToLower(reg2.String()),
					Size:      16,
					Offset:    imm,
					LineIndex: idx,
				}
			}
		}

	case arm64asm.STR:
		// Single register store - treat like STP but with one register
		if len(inst.Args) >= 2 {
			reg, regOk := inst.Args[0].(arm64asm.Reg)
			mem, memOk := inst.Args[1].(arm64asm.MemImmediate)
			if regOk && memOk && mem.Base == arm64asm.RegSP(arm64asm.SP) {
				imm := immFromMemImmediate(mem)
				return &StackOp{
					Kind:      StackOpPush,
					Reg:       strings.ToLower(reg.String()),
					Size:      8,
					Offset:    imm,
					LineIndex: idx,
				}
			}
		}

	case arm64asm.LDR:
		// Single register load - treat like LDP but with one register
		if len(inst.Args) >= 2 {
			reg, regOk := inst.Args[0].(arm64asm.Reg)
			mem, memOk := inst.Args[1].(arm64asm.MemImmediate)
			if regOk && memOk && mem.Base == arm64asm.RegSP(arm64asm.SP) {
				imm := immFromMemImmediate(mem)
				return &StackOp{
					Kind:      StackOpPop,
					Reg:       strings.ToLower(reg.String()),
					Size:      8,
					Offset:    imm,
					LineIndex: idx,
				}
			}
		}

	case arm64asm.SUB:
		// sub sp, sp, #N or sub xN, sp, #N
		if len(inst.Args) >= 3 {
			dst := inst.Args[0]
			src := inst.Args[1]
			if dst == arm64asm.RegSP(arm64asm.SP) || src == arm64asm.RegSP(arm64asm.SP) {
				// Parse immediate from assembly since Args[2] might be complex
				if len(fields) > 3 {
					immStr := strings.TrimPrefix(fields[3], "#")
					if n, err := strconv.ParseInt(immStr, 0, 64); err == nil {
						kind := StackOpAlloc
						reg := ""
						if dst != arm64asm.RegSP(arm64asm.SP) {
							// sub xN, sp, #M - computing an address for alignment
							// This should be rewritten to MOV xN, SP
							kind = StackOpAlignPrep
							if dstReg, ok := dst.(arm64asm.Reg); ok {
								reg = strings.ToLower(dstReg.String())
							}
						}
						return &StackOp{
							Kind:      kind,
							Reg:       reg,
							Immediate: n,
							LineIndex: idx,
						}
					}
				}
			}
		}

	case arm64asm.ADD:
		// add sp, sp, #N (dealloc)
		if len(inst.Args) >= 3 {
			dst := inst.Args[0]
			src := inst.Args[1]
			if dst == arm64asm.RegSP(arm64asm.SP) && src == arm64asm.RegSP(arm64asm.SP) {
				if len(fields) > 3 {
					immStr := strings.TrimPrefix(fields[3], "#")
					if n, err := strconv.ParseInt(immStr, 0, 64); err == nil {
						return &StackOp{
							Kind:      StackOpDealloc,
							Immediate: n,
							LineIndex: idx,
						}
					}
				}
			}
		}

	case arm64asm.AND:
		// and sp, xN, #-M (alignment)
		if len(inst.Args) >= 1 {
			dst := inst.Args[0]
			if dst == arm64asm.RegSP(arm64asm.SP) || dst == arm64asm.SP {
				return &StackOp{
					Kind:      StackOpAlign,
					LineIndex: idx,
				}
			}
		}

	case arm64asm.MOV:
		// mov x29, sp (frame setup) or mov sp, x29 (frame teardown)
		if len(inst.Args) >= 2 {
			dst := inst.Args[0]
			src := inst.Args[1]
			if dst == arm64asm.RegSP(arm64asm.X29) && src == arm64asm.RegSP(arm64asm.SP) {
				return &StackOp{Kind: StackOpFrameSetup, LineIndex: idx}
			}
			if dst == arm64asm.RegSP(arm64asm.SP) && src == arm64asm.RegSP(arm64asm.X29) {
				return &StackOp{Kind: StackOpFrameTeardown, LineIndex: idx}
			}
		}
	}

	return nil
}

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// getArchStackInfo returns the appropriate ArchStackInfo for the given architecture
func getArchStackInfo(arch *config.Arch) ArchStackInfo {
	if arch == nil {
		return newAmd64StackInfo()
	}
	switch arch.Name {
	case "arm64":
		return newArm64StackInfo()
	case "amd64":
		return newAmd64StackInfo()
	}
	panic(fmt.Sprintf("no ArchStackInfo for architecture: %s", arch.Name))
}

// analyzeStackLayout performs the first pass analysis to build StackLayout
func analyzeStackLayout(archInfo ArchStackInfo, lines []Line) *StackLayout {
	layout := &StackLayout{
		NopIndices:   make(map[int]bool),
		AllocIndices: make(map[int]bool),
	}

	// Parse all stack operations
	var ops []*StackOp
	for i, line := range lines {
		if op := archInfo.ParseStackOp(i, line); op != nil {
			ops = append(ops, op)
		}
	}

	// Analyze the operations
	for _, op := range ops {
		switch op.Kind {
		case StackOpFrameSetup:
			layout.FramePointerUsed = true
			layout.NopIndices[op.LineIndex] = true

		case StackOpFrameTeardown:
			layout.NopIndices[op.LineIndex] = true

		case StackOpPush:
			isCalleeSaved := archInfo.IsCalleeSaved(op.Reg)
			if op.Reg2 != "" {
				isCalleeSaved = isCalleeSaved && archInfo.IsCalleeSaved(op.Reg2)
			}

			layout.SavedRegs = append(layout.SavedRegs, SavedReg{
				Reg:           op.Reg,
				IsCalleeSaved: isCalleeSaved,
			})
			if op.Reg2 != "" {
				layout.SavedRegs = append(layout.SavedRegs, SavedReg{
					Reg:           op.Reg2,
					IsCalleeSaved: isCalleeSaved,
				})
			}

			// If this is a callee-saved register push, NOP it
			if isCalleeSaved {
				layout.NopIndices[op.LineIndex] = true
			} else {
				// Non-callee-saved push contributes to GoFrameSize
				layout.GoFrameSize += op.Size
			}

			// If this is a pre-indexed push (stp x29, x30, [sp, #-N]!)
			// the immediate tells us about stack allocation
			if op.Immediate > 0 {
				layout.LocalsSize += int(op.Immediate)
			}

		case StackOpPop:
			isCalleeSaved := archInfo.IsCalleeSaved(op.Reg)
			if op.Reg2 != "" {
				isCalleeSaved = isCalleeSaved && archInfo.IsCalleeSaved(op.Reg2)
			}

			// If this is a callee-saved register pop, NOP it
			if isCalleeSaved {
				layout.NopIndices[op.LineIndex] = true
			}

		case StackOpAlloc:
			layout.LocalsSize += int(op.Immediate)
			layout.AllocIndices[op.LineIndex] = true
			layout.NopIndices[op.LineIndex] = true

		case StackOpDealloc:
			// Stack deallocation is part of epilogue, NOP it
			layout.NopIndices[op.LineIndex] = true

		case StackOpAlign:
			if op.Immediate != 0 && op.Immediate != -8 {
				layout.Alignment = op.Immediate
			}
			// NOP out alignment instructions since Go handles alignment differently
			layout.NopIndices[op.LineIndex] = true
		}
	}

	// Calculate Go frame size: locals + space for non-callee-saved spills
	// But if all saved registers are callee-saved (NOPed) and there are no
	// other stack-referencing operations, we don't need stack space
	allCalleeSaved := true
	for _, reg := range layout.SavedRegs {
		if !reg.IsCalleeSaved {
			allCalleeSaved = false
			break
		}
	}

	// Check if any operations still need the stack frame
	hasStackRefs := false
	for _, op := range ops {
		switch op.Kind {
		case StackOpAlignPrep:
			// Address computation that references stack
			hasStackRefs = true
		case StackOpAlloc:
			// Explicit stack allocation (sub rsp, N) means we have locals
			hasStackRefs = true
		}
		if hasStackRefs {
			break
		}
	}

	if allCalleeSaved && len(layout.SavedRegs) > 0 && !hasStackRefs {
		// All register saves are callee-saved and no stack references, so we don't need any stack space
		layout.GoFrameSize = 0
	} else {
		layout.GoFrameSize += layout.LocalsSize
	}

	return layout
}

// TranslateOffset converts a C stack offset to Go's stack-N(SP) format
// cOffset is the offset from the C stack pointer after prologue
// Returns the offset for use in stack-N(SP) notation
func (s *StackLayout) TranslateOffset(cOffset int) int {
	// In Go asm, stack-0(SP) is at the top of the frame (highest address)
	// stack-N(SP) is N bytes below the top
	// We need to map C's [rsp+offset] to Go's stack-(GoFrameSize-offset)(SP)
	return s.GoFrameSize - cOffset
}

// FormatStackRef formats a stack reference in Go assembly style
// Returns format like "stack-N(SP)" where N is the offset from top of frame
func (s *StackLayout) FormatStackRef(cOffset int, name string) string {
	goOffset := s.TranslateOffset(cOffset)
	if name == "" {
		name = "stack"
	}
	return fmt.Sprintf("%s-%d(SP)", name, goOffset)
}

// rewriteStackOps performs the second pass to rewrite stack operations
func rewriteStackOps(arch *config.Arch, archInfo ArchStackInfo, layout *StackLayout, function Function) Function {
	newLines := make([]Line, 0, len(function.Lines))

	// Track push offset for rewriting non-callee-saved pushes to MOVs
	pushOffset := layout.LocalsSize

	for i, line := range function.Lines {
		// Check if this line should be NOPed
		if layout.NopIndices[i] {
			lineCpy := line
			lineCpy.Disassembled = "NOP"
			lineCpy.Binary = nil
			newLines = append(newLines, lineCpy)
			continue
		}

		// Parse this line's stack operation (if any)
		op := archInfo.ParseStackOp(i, line)
		if op != nil {
			switch op.Kind {
			case StackOpPush:
				// Non-callee-saved push: rewrite to MOV
				if !archInfo.IsCalleeSaved(op.Reg) {
					movInstr := arch.MovInstr[8]
					parts := strings.Fields(line.Disassembled)
					var reg string
					if len(parts) > 1 {
						reg = parts[1]
					} else {
						reg = strings.ToUpper(op.Reg)
					}
					instr := fmt.Sprintf("%s %s, stack-%d(SP)", movInstr, reg, layout.GoFrameSize-pushOffset)
					pushOffset += 8
					lineCpy := line
					lineCpy.Disassembled = instr
					lineCpy.Binary = nil
					newLines = append(newLines, lineCpy)
					continue
				}

			case StackOpPop:
				// Non-callee-saved pop: rewrite to MOV
				if !archInfo.IsCalleeSaved(op.Reg) {
					pushOffset -= 8
					movInstr := arch.MovInstr[8]
					parts := strings.Fields(line.Disassembled)
					var reg string
					if len(parts) > 1 {
						reg = parts[1]
					} else {
						reg = strings.ToUpper(op.Reg)
					}
					instr := fmt.Sprintf("%s stack-%d(SP), %s", movInstr, layout.GoFrameSize-pushOffset, reg)
					lineCpy := line
					lineCpy.Disassembled = instr
					lineCpy.Binary = nil
					newLines = append(newLines, lineCpy)
					continue
				}

			case StackOpAlignPrep:
				// sub xN, sp, #M - compute the translated address into xN
				// In C: xN = sp - M (address M bytes below current SP)
				// In Go: the top of stack is stack-LocalsSize(SP), so M bytes
				// "below" (toward SP) is stack-(LocalsSize-M)(SP)
				reg := strings.ToUpper(op.Reg)
				if reg == "" {
					reg = "R9" // fallback
				}
				// Load effective address of the stack slot into the register
				// Using MOVD with $ prefix to get the address
				goOffset := layout.LocalsSize - int(op.Immediate)
				instr := fmt.Sprintf("MOVD $stack-%d(SP), %s", goOffset, reg)
				lineCpy := line
				lineCpy.Disassembled = instr
				lineCpy.Binary = nil
				newLines = append(newLines, lineCpy)
				continue
			}
		}

		// Check for aligned instructions that need to be converted to unaligned
		// Only convert if there's a stack memory operand (register-to-register or
		// non-stack memory accesses don't need alignment adjustment)
		if line.Disassembled != "" {
			fields := strings.Fields(line.Disassembled)
			if len(fields) > 0 {
				if unaligned := archInfo.ToUnalignedInsn(fields[0]); unaligned != nil {
					operands := line.Disassembled[len(fields[0]):]
					// Only convert if operands reference the stack
					if archInfo.HasStackMemoryRef(operands) {
						lineCpy := line
						lineCpy.Disassembled = *unaligned + operands
						lineCpy.Binary = nil
						newLines = append(newLines, lineCpy)
						continue
					}
				}
			}
		}

		// Keep the line as-is
		newLines = append(newLines, line)
	}

	function.Lines = newLines
	function.LocalsSize = layout.GoFrameSize

	return function
}

// checkStackUnified is the new unified stack checking function
func checkStackUnified(arch *config.Arch, function Function) Function {
	archInfo := getArchStackInfo(arch)

	// Pass 1: Analyze stack layout
	layout := analyzeStackLayout(archInfo, function.Lines)

	// Check if we need complex rewrite
	needsComplexRewrite := false
	for _, reg := range layout.SavedRegs {
		if !reg.IsCalleeSaved {
			needsComplexRewrite = true
			break
		}
	}

	// Also need complex rewrite if alignment > 8
	if layout.Alignment != 0 && layout.Alignment != -8 {
		needsComplexRewrite = true
	}

	if needsComplexRewrite {
		fnName := function.Name
		if fnName == "" {
			fnName = "[unknown]"
		}
		fmt.Fprintf(os.Stderr, "WARN: %s: contains complex stack manipulation, running experimental transform\n", fnName)
	}

	// Pass 2: Rewrite stack operations
	return rewriteStackOps(arch, archInfo, layout, function)
}

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
		fnName := function.Name
		if fnName == "" {
			fnName = "[unknown]"
		}
		fmt.Fprintf(os.Stderr, "WARN: %s: contains complex stack manipulation, running experimental transform\n", fnName)
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
					// this tells us how much stack space we're using
					if ok && imm.Base == arm64asm.RegSP(arm64asm.SP) && baseStack == 0 {
						n := immFromMemImmediate(imm)
						baseStack = -n
						extraStack = baseStack
					}
				} else if len(inst.Args) > 2 {
					imm, ok := inst.Args[2].(arm64asm.MemImmediate)
					if ok && imm.Base == arm64asm.RegSP(arm64asm.SP) && baseStack == 0 {
						n := immFromMemImmediate(imm)
						baseStack = -n
						extraStack = baseStack
					}
					// this could still be fine, as long as it's doing just callee-saved registers
					rewriteRequired = true
				}
			case arm64asm.STR:
				imm, ok := inst.Args[1].(arm64asm.MemImmediate)
				if ok && imm.Base == arm64asm.RegSP(arm64asm.SP) && baseStack == 0 {
					n := immFromMemImmediate(imm)
					baseStack = -n
					extraStack = baseStack
				}
				// this could still be fine, as long as it's doing just callee-saved registers
				rewriteRequired = true
			case arm64asm.AND:
				// stack alignment
				// this basically grows the stack, need to adjust for it
				targetReg := inst.Args[0]
				if targetReg == arm64asm.SP {
					// allocating more stack space
					rewriteRequired = true
					// TODO: definitely clear sign that we're doing something with the stack
					complexManip = true
				}
			case arm64asm.SUB:
				// allocating stack space
				targetReg := inst.Args[0]
				srcReg := inst.Args[1]
				if targetReg == arm64asm.RegSP(arm64asm.SP) || srcReg == arm64asm.RegSP(arm64asm.SP) {
					complexManip = true
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
	} else if !complexManip {
		fnName := function.Name
		if fnName == "" {
			fnName = "[unknown]"
		}
		fmt.Fprintf(os.Stderr, "WARN: %s: contains stack manipulation, running experimental transform\n", fnName)

		newLines := make([]Line, 0, len(function.Lines))
		stackAllocator := map[string]int{}
		stackSpace := -extraStack

		for _, line := range function.Lines {
			asm := line.Assembly
			// detect everything that touches SP
			if spInstruction.MatchString(asm) {
				inst := decodeArm64Line(line)
				doSkip := false

				switch inst.Op {
				case arm64asm.STP, arm64asm.LDP:
					switch {
					// go's ABI0 doesn't require callee-saved registers
					case inst.Args[0] == arm64asm.X20 && inst.Args[1] == arm64asm.X19:
						fallthrough
					case inst.Args[0] == arm64asm.X22 && inst.Args[1] == arm64asm.X21:
						fallthrough
					case inst.Args[0] == arm64asm.X24 && inst.Args[1] == arm64asm.X23:
						fallthrough
					case inst.Args[0] == arm64asm.X26 && inst.Args[1] == arm64asm.X25:
						fallthrough
					case inst.Args[0] == arm64asm.X28 && inst.Args[1] == arm64asm.X27:
						fallthrough
					case inst.Args[0] == arm64asm.X29 && inst.Args[1] == arm64asm.X30:
						doSkip = true
					}
				case arm64asm.STR, arm64asm.LDR:
					switch inst.Args[0] {
					// go's ABI0 doesn't require callee-saved registers
					case arm64asm.X19, arm64asm.X20, arm64asm.X21, arm64asm.X22, arm64asm.X23, arm64asm.X24,
						arm64asm.X25, arm64asm.X26, arm64asm.X27, arm64asm.X28, arm64asm.X29, arm64asm.X30:
						doSkip = true
					}
				case arm64asm.MOV:
					if inst.Args[0] == arm64asm.RegSP(arm64asm.X29) {
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
				case arm64asm.AND, arm64asm.SUB:
					if len(inst.Args) > 2 && inst.Args[0] == arm64asm.RegSP(arm64asm.SP) {
						// stack alloc/alignment writing back into RSP
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
		fnName := function.Name
		if fnName == "" {
			fnName = "[unknown]"
		}
		fmt.Fprintf(os.Stderr, "WARN: %s: contains complex stack manipulation, running experimental transform\n", fnName)
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

				switch inst.Op {
				case arm64asm.STP, arm64asm.STR:
					lineCpy := line
					lineCpy.Disassembled = arm64asm.GoSyntax(inst, 0, nil, nil)
					lineCpy.Binary = nil
					newLines = append(newLines, lineCpy)
					continue
				case arm64asm.LDP, arm64asm.LDR:
					lineCpy := line
					lineCpy.Disassembled = arm64asm.GoSyntax(inst, 0, nil, nil)
					lineCpy.Binary = nil
					newLines = append(newLines, lineCpy)
					continue
				case arm64asm.AND, arm64asm.SUB:
					if len(inst.Args) > 2 && inst.Args[0] == arm64asm.RegSP(arm64asm.SP) {
						// stack alloc/alignment writing back into RSP
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
		// FIXME: we're doing extra 16bytes (which C uses for x29/x30)
		function.LocalsSize = extraStack
	}

	return function
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
