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
	StackOpFrameSetup                // mov rbp, rsp (AMD64) or mov xN, sp / add xN, sp, #N (ARM64)
	StackOpFrameTeardown             // mov rsp, rbp (AMD64) or mov sp, xN (ARM64)
	StackOpSpill                     // fixed-offset store into an existing stack frame
	StackOpReload                    // fixed-offset load from an existing stack frame
	StackOpUnsupported               // any other write to the stack pointer (dynamic allocation)
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
	// ARM64 frame-setup lines that must be retained, mapped to their offset
	// from the final hardware RSP.
	FrameSetupOffsets map[int]int

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

	// PreserveCalleeSaved keeps the compiler's C-ABI save/restore pairs for
	// helper functions that call each other using the C register convention.
	PreserveCalleeSaved bool
	ResolvedOffsets     map[int]int

	// RetainedSaves marks callee-saved register saves/restores that must be
	// kept as fixed-slot moves even when callee-saved registers are otherwise
	// removable (for example the ARM64 link register when the body reuses it).
	RetainedSaves map[int]bool
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

	// WritesRegister reports whether any of lines[from:to] may modify reg.
	WritesRegister(lines []Line, from, to int, reg string) bool
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

func (a *amd64StackInfo) WritesRegister(lines []Line, from, to int, reg string) bool {
	ref := frameRegisterRef("amd64", reg)
	for i := from; i < to && i < len(lines); i++ {
		line := lines[i]
		if len(line.Binary) == 0 || !ref.MatchString(line.Assembly) {
			continue
		}
		inst, ok := tryDecodeAmd64Line(line)
		if !ok {
			// Unknown encoding (EVEX): Intel syntax names the destination first.
			fields := strings.Fields(line.Assembly)
			if len(fields) > 1 && ref.MatchString(strings.TrimSuffix(fields[1], ",")) {
				return true
			}
			continue
		}
		switch inst.Op {
		case x86asm.CMP, x86asm.TEST, x86asm.PUSH, x86asm.POP:
			// Epilogue pops on other control-flow paths restore the register;
			// they are rewritten to fixed-slot loads and never precede a
			// teardown on the same path.
			continue
		}
		if len(inst.Args) == 0 {
			continue
		}
		if dst, ok := inst.Args[0].(x86asm.Reg); ok && ref.MatchString(strings.ToLower(dst.String())) {
			return true
		}
	}
	return false
}

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
	"MOVDQA": "MOVOU",
	// AVX
	"VMOVAPS": "VMOVUPS",
	"VMOVAPD": "VMOVUPD",
	"VMOVDQA": "VMOVDQU",
	// AVX-512
	"VMOVDQA32": "VMOVDQU32",
	"VMOVDQA64": "VMOVDQU64",
}

func (a *amd64StackInfo) ToUnalignedInsn(insn string) *string {
	if unaligned, ok := amd64AlignedToUnaligned[insn]; ok {
		return &unaligned
	}
	return nil
}

func (a *amd64StackInfo) ParseStackOp(idx int, line Line) *StackOp {
	asm := line.Assembly
	if asm == "" || len(line.Binary) == 0 {
		return nil
	}

	fields := strings.Fields(asm)
	if len(fields) == 0 {
		return nil
	}

	instr := strings.ToLower(fields[0])
	if instr != "push" && instr != "pop" && !a.spRegex.MatchString(asm) {
		return nil
	}

	inst, ok := tryDecodeAmd64Line(line)
	if !ok {
		// x86asm has no EVEX support; AVX-512 instructions never write RSP.
		return nil
	}
	unsupported := &StackOp{Kind: StackOpUnsupported, LineIndex: idx}

	switch inst.Op {
	case x86asm.PUSH, x86asm.POP:
		reg, ok := inst.Args[0].(x86asm.Reg)
		if !ok {
			return unsupported
		}
		kind := StackOpPush
		if inst.Op == x86asm.POP {
			kind = StackOpPop
		}
		return &StackOp{
			Kind:      kind,
			Reg:       strings.ToLower(reg.String()),
			Size:      8,
			LineIndex: idx,
		}

	case x86asm.MOV:
		dst, dstOk := inst.Args[0].(x86asm.Reg)
		src, srcOk := inst.Args[1].(x86asm.Reg)
		if dstOk && srcOk && dst == x86asm.RBP && src == x86asm.RSP {
			return &StackOp{Kind: StackOpFrameSetup, Reg: "rbp", LineIndex: idx}
		}
		if dstOk && srcOk && dst == x86asm.RSP && src == x86asm.RBP {
			return &StackOp{Kind: StackOpFrameTeardown, Reg: "rbp", LineIndex: idx}
		}

	case x86asm.SUB, x86asm.ADD, x86asm.AND:
		dst, dstOk := inst.Args[0].(x86asm.Reg)
		if !dstOk || dst != x86asm.RSP {
			return nil
		}
		imm, immOk := inst.Args[1].(x86asm.Imm)
		if !immOk {
			return unsupported
		}
		switch inst.Op {
		case x86asm.SUB:
			return &StackOp{Kind: StackOpAlloc, Immediate: int64(imm), LineIndex: idx}
		case x86asm.ADD:
			return &StackOp{Kind: StackOpDealloc, Immediate: int64(imm), LineIndex: idx}
		default:
			return &StackOp{Kind: StackOpAlign, Immediate: int64(imm), LineIndex: idx}
		}

	case x86asm.LEA:
		dst, dstOk := inst.Args[0].(x86asm.Reg)
		if !dstOk || dst != x86asm.RSP {
			return nil
		}
		mem, memOk := inst.Args[1].(x86asm.Mem)
		if !memOk || mem.Index != 0 {
			return unsupported
		}
		switch mem.Base {
		case x86asm.RBP:
			// lea rsp, [rbp - N]: frame teardown variant
			return &StackOp{Kind: StackOpFrameTeardown, Reg: "rbp", LineIndex: idx}
		case x86asm.RSP:
			if mem.Disp < 0 {
				return &StackOp{Kind: StackOpAlloc, Immediate: -mem.Disp, LineIndex: idx}
			}
			return &StackOp{Kind: StackOpDealloc, Immediate: mem.Disp, LineIndex: idx}
		}
		return unsupported
	}

	// Any other instruction that writes RSP cannot be mapped onto a fixed Go frame.
	if len(inst.Args) > 0 {
		if dst, ok := inst.Args[0].(x86asm.Reg); ok && dst == x86asm.RSP {
			return unsupported
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

func (a *arm64StackInfo) WritesRegister(lines []Line, from, to int, reg string) bool {
	ref := frameRegisterRef("arm64", reg)
	for i := from; i < to && i < len(lines); i++ {
		line := lines[i]
		if len(line.Binary) == 0 || !ref.MatchString(line.Assembly) {
			continue
		}
		inst := decodeArm64Line(line)
		switch inst.Op {
		case arm64asm.STR, arm64asm.STP, arm64asm.STUR, arm64asm.STRB, arm64asm.STRH, arm64asm.STURB, arm64asm.STURH,
			arm64asm.CMP, arm64asm.CMN, arm64asm.TST, arm64asm.B, arm64asm.BL, arm64asm.CBZ, arm64asm.CBNZ,
			arm64asm.TBZ, arm64asm.TBNZ:
			continue
		case arm64asm.LDR, arm64asm.LDP:
			// Epilogue reloads from the stack on other control-flow paths.
			if op := a.ParseStackOp(i, line); op != nil && (op.Kind == StackOpPop || op.Kind == StackOpReload) {
				continue
			}
		}
		if len(inst.Args) == 0 {
			continue
		}
		var dst string
		switch arg := inst.Args[0].(type) {
		case arm64asm.Reg:
			dst = strings.ToLower(arg.String())
		case arm64asm.RegSP:
			dst = strings.ToLower(arg.String())
		default:
			continue
		}
		if ref.MatchString(dst) {
			return true
		}
	}
	return false
}

func (a *arm64StackInfo) IsCalleeSaved(reg string) bool {
	switch strings.ToLower(reg) {
	case "x19", "x20", "x21", "x22", "x23", "x24",
		"x25", "x26", "x27", "x28", "x29", "x30",
		"d8", "d9", "d10", "d11", "d12", "d13", "d14", "d15":
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
		if len(inst.Args) >= 3 {
			reg1, ok1 := inst.Args[0].(arm64asm.Reg)
			reg2, ok2 := inst.Args[1].(arm64asm.Reg)
			mem, memOk := inst.Args[2].(arm64asm.MemImmediate)
			if ok1 && ok2 && memOk && mem.Base == arm64asm.RegSP(arm64asm.SP) {
				imm := immFromMemImmediate(mem)
				isPreIndex := mem.Mode == arm64asm.AddrPreIndex
				kind := StackOpSpill
				if isPreIndex {
					kind = StackOpPush
				}
				return &StackOp{
					Kind:      kind,
					Reg:       strings.ToLower(reg1.String()),
					Reg2:      strings.ToLower(reg2.String()),
					Size:      arm64RegSize(reg1) + arm64RegSize(reg2),
					Offset:    imm,
					Immediate: int64(-imm) * boolToInt(isPreIndex),
					LineIndex: idx,
				}
			}
		}

	case arm64asm.LDP:
		if len(inst.Args) >= 3 {
			reg1, ok1 := inst.Args[0].(arm64asm.Reg)
			reg2, ok2 := inst.Args[1].(arm64asm.Reg)
			mem, memOk := inst.Args[2].(arm64asm.MemImmediate)
			if ok1 && ok2 && memOk && mem.Base == arm64asm.RegSP(arm64asm.SP) {
				imm := immFromMemImmediate(mem)
				kind := StackOpReload
				if mem.Mode == arm64asm.AddrPostIndex {
					kind = StackOpPop
				}
				return &StackOp{
					Kind:      kind,
					Reg:       strings.ToLower(reg1.String()),
					Reg2:      strings.ToLower(reg2.String()),
					Size:      arm64RegSize(reg1) + arm64RegSize(reg2),
					Offset:    imm,
					LineIndex: idx,
				}
			}
		}

	case arm64asm.STR:
		if len(inst.Args) >= 2 {
			reg, regOk := inst.Args[0].(arm64asm.Reg)
			mem, memOk := inst.Args[1].(arm64asm.MemImmediate)
			if regOk && memOk && mem.Base == arm64asm.RegSP(arm64asm.SP) {
				imm := immFromMemImmediate(mem)
				isPreIndex := mem.Mode == arm64asm.AddrPreIndex
				kind := StackOpSpill
				if isPreIndex {
					kind = StackOpPush
				}
				return &StackOp{
					Kind:      kind,
					Reg:       strings.ToLower(reg.String()),
					Size:      arm64RegSize(reg),
					Offset:    imm,
					Immediate: int64(-imm) * boolToInt(isPreIndex),
					LineIndex: idx,
				}
			}
		}

	case arm64asm.LDR:
		if len(inst.Args) >= 2 {
			reg, regOk := inst.Args[0].(arm64asm.Reg)
			mem, memOk := inst.Args[1].(arm64asm.MemImmediate)
			if regOk && memOk && mem.Base == arm64asm.RegSP(arm64asm.SP) {
				imm := immFromMemImmediate(mem)
				kind := StackOpReload
				if mem.Mode == arm64asm.AddrPostIndex {
					kind = StackOpPop
				}
				return &StackOp{
					Kind:      kind,
					Reg:       strings.ToLower(reg.String()),
					Size:      arm64RegSize(reg),
					Offset:    imm,
					LineIndex: idx,
				}
			}
		}

	case arm64asm.SUB, arm64asm.ADD:
		// sub sp, sp, #N (alloc), add sp, sp, #N (dealloc),
		// mov/add xN, sp, #N (frame pointer or address of a local),
		// sub xN, sp, #N (address below SP, used by stack realignment).
		if len(inst.Args) < 3 {
			return nil
		}
		dst, dstOk := inst.Args[0].(arm64asm.RegSP)
		src := inst.Args[1]
		if !dstOk || src != arm64asm.RegSP(arm64asm.SP) {
			break
		}
		imm, immOk := arm64SPImmediate(fields)
		if !immOk {
			if dst == arm64asm.RegSP(arm64asm.SP) {
				return &StackOp{Kind: StackOpUnsupported, LineIndex: idx}
			}
			return nil
		}
		if inst.Op == arm64asm.SUB {
			imm = -imm
		}
		if dst == arm64asm.RegSP(arm64asm.SP) {
			if imm <= 0 {
				return &StackOp{Kind: StackOpAlloc, Immediate: -imm, LineIndex: idx}
			}
			return &StackOp{Kind: StackOpDealloc, Immediate: imm, LineIndex: idx}
		}
		return &StackOp{Kind: StackOpFrameSetup, Reg: strings.ToLower(dst.String()), Offset: int(imm), LineIndex: idx}

	case arm64asm.AND:
		// and sp, xN, #-M (alignment)
		if len(inst.Args) >= 2 && inst.Args[0] == arm64asm.RegSP(arm64asm.SP) {
			op := &StackOp{Kind: StackOpAlign, LineIndex: idx}
			switch src := inst.Args[1].(type) {
			case arm64asm.RegSP:
				op.Reg = strings.ToLower(src.String())
			case arm64asm.Reg:
				op.Reg = strings.ToLower(src.String())
			}
			if len(fields) > 3 {
				if mask, err := strconv.ParseUint(strings.TrimPrefix(fields[3], "#"), 0, 64); err == nil {
					op.Immediate = int64(mask)
				}
			}
			return op
		}

	case arm64asm.MOV:
		// mov xN, sp (frame setup / address of locals) or mov sp, xN (frame teardown)
		if len(inst.Args) >= 2 {
			dst, dstOk := inst.Args[0].(arm64asm.RegSP)
			src, srcOk := inst.Args[1].(arm64asm.RegSP)
			if dstOk && srcOk && src == arm64asm.RegSP(arm64asm.SP) && dst != arm64asm.RegSP(arm64asm.SP) {
				return &StackOp{Kind: StackOpFrameSetup, Reg: strings.ToLower(dst.String()), LineIndex: idx}
			}
			if dstOk && srcOk && dst == arm64asm.RegSP(arm64asm.SP) {
				return &StackOp{Kind: StackOpFrameTeardown, Reg: strings.ToLower(src.String()), LineIndex: idx}
			}
		}
	}

	// Any other instruction that writes SP cannot be mapped onto a fixed Go frame.
	if len(inst.Args) > 0 {
		if dst, ok := inst.Args[0].(arm64asm.RegSP); ok && dst == arm64asm.RegSP(arm64asm.SP) {
			return &StackOp{Kind: StackOpUnsupported, LineIndex: idx}
		}
	}
	return nil
}

// arm64SPImmediate parses the immediate of `op sp/xN, sp, #imm[, lsl #12]`.
func arm64SPImmediate(fields []string) (int64, bool) {
	if len(fields) < 4 || !strings.HasPrefix(fields[3], "#") {
		return 0, false
	}
	imm, err := strconv.ParseInt(strings.TrimSuffix(strings.TrimPrefix(fields[3], "#"), ","), 0, 64)
	if err != nil {
		return 0, false
	}
	if len(fields) >= 6 && strings.EqualFold(fields[4], "lsl") {
		shift, err := strconv.ParseInt(strings.TrimPrefix(fields[5], "#"), 0, 64)
		if err != nil {
			return 0, false
		}
		imm <<= uint(shift)
	} else if len(fields) != 4 {
		return 0, false
	}
	return imm, true
}

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func arm64RegSize(reg arm64asm.Reg) int {
	name := reg.String()
	if name == "" {
		return 8
	}

	switch name[0] {
	case 'B':
		return 1
	case 'H':
		return 2
	case 'W', 'S':
		return 4
	case 'Q':
		return 16
	default:
		return 8
	}
}

func stackOpCalleeSaved(archInfo ArchStackInfo, op *StackOp) bool {
	if !archInfo.IsCalleeSaved(op.Reg) {
		return false
	}
	return op.Reg2 == "" || archInfo.IsCalleeSaved(op.Reg2)
}

func stackSlotKey(op *StackOp) string {
	return fmt.Sprintf("%s/%s/%d/%d", op.Reg, op.Reg2, op.Offset, op.Size)
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

// frameRegisterRef matches any spelling of a general-purpose register in the
// compiler's assembly text.
func frameRegisterRef(archName, reg string) *regexp.Regexp {
	reg = strings.ToLower(reg)
	if archName == "arm64" {
		return regexp.MustCompile(`\b[xw]` + strings.TrimLeft(reg, "xw") + `\b`)
	}
	switch reg {
	case "rbp", "ebp", "bp":
		return regexp.MustCompile(`\b(?:rbp|ebp|bpl|bp)\b`)
	}
	return regexp.MustCompile(`\b` + regexp.QuoteMeta(reg) + `\b`)
}

var arm64LinkRegisterRef = regexp.MustCompile(`\b[xw]30\b`)

// arm64LinkRegisterScratch reports whether the body uses X30 for anything
// other than saving/restoring it. Go's leaf epilogue returns through the live
// X30, so such functions must keep the compiler's LR save and restore.
func arm64LinkRegisterScratch(lines []Line, ops []*StackOp) bool {
	isStackOp := make(map[int]bool, len(ops))
	for _, op := range ops {
		switch op.Kind {
		case StackOpPush, StackOpPop, StackOpSpill, StackOpReload:
			isStackOp[op.LineIndex] = true
		}
	}
	for i, line := range lines {
		if isStackOp[i] || strings.HasPrefix(line.Assembly, "ret") {
			continue
		}
		if arm64LinkRegisterRef.MatchString(line.Assembly) || strings.HasPrefix(line.Disassembled, "CALL ") {
			return true
		}
	}
	return false
}

func stackOpTouches(op *StackOp, reg string) bool {
	return op.Reg == reg || op.Reg2 == reg
}

// analyzeStackLayout performs the first pass analysis to build StackLayout
func analyzeStackLayout(archInfo ArchStackInfo, lines []Line, preserveCalleeSaved bool) (*StackLayout, error) {
	layout := &StackLayout{
		FrameSetupOffsets:   make(map[int]int),
		NopIndices:          make(map[int]bool),
		AllocIndices:        make(map[int]bool),
		PreserveCalleeSaved: preserveCalleeSaved,
		ResolvedOffsets:     make(map[int]int),
		RetainedSaves:       make(map[int]bool),
	}

	// Parse all stack operations
	var ops []*StackOp
	for i, line := range lines {
		op := archInfo.ParseStackOp(i, line)
		if op == nil {
			continue
		}
		if op.Kind == StackOpUnsupported {
			return nil, fmt.Errorf(
				"unsupported stack pointer update %q: dynamic stack allocation (VLA, alloca) cannot be mapped onto a fixed Go frame",
				line.Assembly,
			)
		}
		ops = append(ops, op)
	}

	// Saves that must stay even though the register is callee-saved in C.
	linkRegisterScratch := archInfo.Name() == "arm64" && arm64LinkRegisterScratch(lines, ops)
	retain := func(op *StackOp) bool {
		return linkRegisterScratch && stackOpTouches(op, "x30")
	}
	removableSave := func(op *StackOp) bool {
		if !stackOpCalleeSaved(archInfo, op) {
			return false
		}
		return !layout.PreserveCalleeSaved && !retain(op)
	}

	// A fixed-offset ARM64 store is only a removable C-ABI register save when
	// the function also restores the same register(s) from the same slot.  A
	// store by itself may be ordinary program data (including stack zeroing).
	spillKeys := make(map[string]bool)
	reloadKeys := make(map[string]bool)
	for _, op := range ops {
		switch op.Kind {
		case StackOpSpill:
			if stackOpCalleeSaved(archInfo, op) {
				spillKeys[stackSlotKey(op)] = true
			}
		case StackOpReload:
			if stackOpCalleeSaved(archInfo, op) {
				reloadKeys[stackSlotKey(op)] = true
			}
		}
	}

	// Analyze the operations. Track where the compiler copies SP into a
	// register so the copy can be recreated relative to the fixed Go frame if
	// the body actually uses it.
	frameSetupDepths := make(map[int]int)
	frameSetupOps := make(map[int]*StackOp)
	lastFrameSetup := make(map[string]*StackOp)
	alignAllocs := make(map[int]int)
	for _, op := range ops {
		switch op.Kind {
		case StackOpFrameSetup:
			layout.FramePointerUsed = true
			frameSetupDepths[op.LineIndex] = layout.LocalsSize
			frameSetupOps[op.LineIndex] = op
			lastFrameSetup[op.Reg] = op
			layout.NopIndices[op.LineIndex] = true

		case StackOpFrameTeardown:
			setup, ok := lastFrameSetup[op.Reg]
			if !ok || archInfo.WritesRegister(lines, setup.LineIndex+1, op.LineIndex, op.Reg) {
				return nil, fmt.Errorf(
					"unsupported stack pointer update %q: %s does not hold a fixed stack pointer offset (dynamic stack allocation such as VLA/alloca is not supported)",
					lines[op.LineIndex].Assembly, op.Reg,
				)
			}
			layout.NopIndices[op.LineIndex] = true

		case StackOpPush:
			isCalleeSaved := stackOpCalleeSaved(archInfo, op)
			isRemovable := removableSave(op)

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
			if isRemovable {
				layout.NopIndices[op.LineIndex] = true
			} else if isCalleeSaved {
				layout.RetainedSaves[op.LineIndex] = true
			}

			// If this is a pre-indexed push (stp x29, x30, [sp, #-N]!)
			// the immediate tells us about stack allocation
			if op.Immediate > 0 {
				layout.LocalsSize += int(op.Immediate)
			} else if archInfo.Name() == "amd64" && !isRemovable {
				layout.LocalsSize += op.Size
			}

		case StackOpPop:
			// If this is a callee-saved register pop, NOP it
			if removableSave(op) {
				layout.NopIndices[op.LineIndex] = true
			} else if stackOpCalleeSaved(archInfo, op) {
				layout.RetainedSaves[op.LineIndex] = true
			}

		case StackOpSpill:
			if spillKeys[stackSlotKey(op)] && reloadKeys[stackSlotKey(op)] && removableSave(op) {
				layout.SavedRegs = append(layout.SavedRegs, SavedReg{
					Reg:           op.Reg,
					IsCalleeSaved: true,
				})
				if op.Reg2 != "" {
					layout.SavedRegs = append(layout.SavedRegs, SavedReg{
						Reg:           op.Reg2,
						IsCalleeSaved: true,
					})
				}
				layout.NopIndices[op.LineIndex] = true
			}

		case StackOpReload:
			if spillKeys[stackSlotKey(op)] && reloadKeys[stackSlotKey(op)] && removableSave(op) {
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
			if op.Reg != "" && op.Reg != "sp" {
				// and sp, xN, #-A realigns an area reserved by sub xN, sp, #size.
				// Go only guarantees 16-byte alignment, so keep the size and
				// drop the realignment.
				setup, ok := lastFrameSetup[op.Reg]
				if !ok || setup.Offset >= 0 || archInfo.WritesRegister(lines, setup.LineIndex+1, op.LineIndex, op.Reg) {
					return nil, fmt.Errorf(
						"unsupported stack realignment %q: %s does not hold a fixed stack pointer offset",
						lines[op.LineIndex].Assembly, op.Reg,
					)
				}
				layout.LocalsSize += -setup.Offset
				alignAllocs[op.LineIndex] = -setup.Offset
				layout.AllocIndices[op.LineIndex] = true
			}
			// NOP out alignment instructions since Go handles alignment differently
			layout.NopIndices[op.LineIndex] = true
		}
	}

	// Recreate SP copies that the body still reads. Saving and restoring the
	// register itself does not count as a read of the copied address.
	saveRestoreOf := make(map[int]*StackOp)
	for _, op := range ops {
		switch op.Kind {
		case StackOpPush, StackOpPop, StackOpSpill, StackOpReload:
			saveRestoreOf[op.LineIndex] = op
		}
	}
	hasCalls := false
	for _, line := range lines {
		if _, ok := BranchTarget(line); ok {
			hasCalls = true
			break
		}
	}
	for lineIndex, setupDepth := range frameSetupDepths {
		op := frameSetupOps[lineIndex]
		ref := frameRegisterRef(archInfo.Name(), op.Reg)
		// A call consumes argument registers implicitly, so the copied
		// address may be live even if no instruction names the register.
		used := hasCalls
		for i, line := range lines {
			if used {
				break
			}
			if layout.NopIndices[i] {
				continue
			}
			if save, ok := saveRestoreOf[i]; ok && stackOpTouches(save, op.Reg) && stackOpCalleeSaved(archInfo, save) {
				continue
			}
			if ref.MatchString(line.Assembly) {
				used = true
				break
			}
		}
		if used {
			layout.NopIndices[lineIndex] = false
			layout.FrameSetupOffsets[lineIndex] = layout.LocalsSize - setupDepth + op.Offset
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

	// Check if any operations still need the stack frame.
	hasStackRefs := false
	for _, op := range ops {
		// Explicit stack allocation (sub rsp, N) means we have locals
		if op.Kind == StackOpAlloc || alignAllocs[op.LineIndex] > 0 {
			hasStackRefs = true
			break
		}
	}
	if !hasStackRefs {
		for i, line := range lines {
			if layout.NopIndices[i] {
				continue
			}
			if archInfo.SPRegex().MatchString(line.Assembly) ||
				archInfo.HasStackMemoryRef(line.Disassembled) {
				hasStackRefs = true
				break
			}
		}
	}

	if !layout.PreserveCalleeSaved && allCalleeSaved && len(layout.SavedRegs) > 0 && !hasStackRefs {
		// All register saves are callee-saved and no stack references, so we don't need any stack space
		layout.GoFrameSize = 0
	} else {
		// cmd/asm requires 8-byte (AMD64) or 16-byte (ARM64) frame sizes; the
		// compiler's allocation may be smaller when it has no call boundary.
		align := archInfo.PtrSize()
		if archInfo.Name() == "arm64" {
			align = 16
		}
		layout.GoFrameSize += (layout.LocalsSize + align - 1) &^ (align - 1)
	}

	// Resolve every compiler stack reference against the final fixed Go frame.
	depth := 0
	pushOffsets := make(map[string]int)
	spillOffsets := make(map[string]int)
	registerKey := func(op *StackOp) string { return op.Reg + "/" + op.Reg2 }
	for _, op := range ops {
		switch op.Kind {
		case StackOpPush:
			// ARM64 pre-indexed saves combine the save with stack allocation, and
			// that allocation remains part of the fixed Go frame even when the
			// callee-save itself is removed. On AMD64, an ordinary PUSH only
			// contributes to the compacted frame when it is retained. Internal
			// helpers preserve their C-ABI saves, so those pushes are not NOPed.
			if op.Immediate > 0 {
				depth += int(op.Immediate)
			} else if archInfo.Name() == "amd64" && !layout.NopIndices[op.LineIndex] {
				depth += op.Size
			}
			layout.ResolvedOffsets[op.LineIndex] = layout.LocalsSize - depth
			pushOffsets[registerKey(op)] = layout.ResolvedOffsets[op.LineIndex]
		case StackOpSpill:
			layout.ResolvedOffsets[op.LineIndex] = layout.LocalsSize - depth + op.Offset
			if spillKeys[stackSlotKey(op)] && reloadKeys[stackSlotKey(op)] {
				spillOffsets[stackSlotKey(op)] = layout.ResolvedOffsets[op.LineIndex]
			}
		case StackOpReload:
			if offset, ok := spillOffsets[stackSlotKey(op)]; ok {
				layout.ResolvedOffsets[op.LineIndex] = offset
			} else {
				layout.ResolvedOffsets[op.LineIndex] = layout.LocalsSize - depth + op.Offset
			}
		case StackOpAlloc:
			depth += int(op.Immediate)
		case StackOpAlign:
			depth += alignAllocs[op.LineIndex]
		case StackOpDealloc:
			// Epilogues are often duplicated on multiple control-flow paths. Keep
			// the body depth fixed; restore operations resolve through their
			// matching prologue save slot below.
		case StackOpPop:
			if offset, ok := pushOffsets[registerKey(op)]; ok {
				layout.ResolvedOffsets[op.LineIndex] = offset
			} else {
				layout.ResolvedOffsets[op.LineIndex] = layout.LocalsSize - depth
			}
		}
	}

	return layout, nil
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

var arm64RSPMemoryRef = regexp.MustCompile(`[+-]?[0-9]*\(RSP\)`)
var arm64RSPBaseAdd = regexp.MustCompile(`^ADD \$([0-9]+), RSP, (R[0-9]+)$`)
var arm64RSPBaseMove = regexp.MustCompile(`^MOVD RSP, (R[0-9]+)$`)

func arm64PairRegisters(line Line) (reg1, reg2 string, elementSize int, ok bool) {
	fields := strings.Fields(line.Assembly)
	if len(fields) < 3 || (fields[0] != "stp" && fields[0] != "ldp") {
		return "", "", 0, false
	}

	reg1 = strings.TrimSuffix(fields[1], ",")
	reg2 = strings.TrimSuffix(fields[2], ",")
	_, _, _, elementSize = arm64SavedRegMove(reg1)
	return reg1, reg2, elementSize, true
}

func arm64PairPlan9Opcode(reg string, load bool) string {
	reg = strings.ToLower(reg)
	switch reg[0] {
	case 'w':
		if load {
			return "LDPW"
		}
		return "STPW"
	case 'q':
		if load {
			return "FLDPQ"
		}
		return "FSTPQ"
	case 'd':
		if load {
			return "FLDPD"
		}
		return "FSTPD"
	case 's':
		if load {
			return "FLDPS"
		}
		return "FSTPS"
	default:
		if load {
			return "LDP"
		}
		return "STP"
	}
}

func arm64PairOffsetEncodable(offset, elementSize int) bool {
	return elementSize > 0 && offset%elementSize == 0 &&
		offset >= -64*elementSize && offset <= 63*elementSize
}

func arm64SplitContinuationComment(line Line) string {
	fields := strings.Fields(line.Assembly)
	if len(fields) == 0 {
		return "split continuation of preceding pair instruction"
	}
	return "split continuation of preceding " + strings.ToUpper(fields[0])
}

func splitArm64StackPair(line Line, offset int) []Line {
	reg1, reg2, _, ok := arm64PairRegisters(line)
	if !ok {
		return []Line{line}
	}

	regs := []string{reg1, reg2}
	result := make([]Line, 0, len(regs))
	isLoad := strings.HasPrefix(strings.TrimSpace(line.Assembly), "ldp")
	for i, reg := range regs {
		goReg, store, load, size := arm64SavedRegMove(reg)
		instruction := fmt.Sprintf("%s %s, %d(RSP)", store, goReg, offset)
		if isLoad {
			instruction = fmt.Sprintf("%s %d(RSP), %s", load, offset, goReg)
		}
		rewritten := Line{Assembly: line.Assembly, Disassembled: instruction}
		if i == 0 {
			rewritten.Labels = line.Labels
		} else {
			rewritten.Comment = arm64SplitContinuationComment(line)
		}
		result = append(result, rewritten)
		offset += size
	}
	return result
}

// shiftArm64CStackRef keeps translated C locals above the linkage word that
// cmd/asm reserves at 0(RSP) for LR whenever a Go assembly function has a
// frame. C sees its post-prologue SP as the bottom of its local area, whereas
// Go's hardware RSP still points at that linkage word. Stack pairs remain
// paired when the final scaled offset is encodable and split otherwise.
func shiftArm64CStackRef(line Line, bias int) []Line {
	if !strings.Contains(line.Disassembled, "RSP") {
		return []Line{line}
	}

	shiftedOffset := 0
	foundStackRef := false
	rewritten := arm64RSPMemoryRef.ReplaceAllStringFunc(line.Disassembled, func(ref string) string {
		offsetText := strings.TrimSuffix(ref, "(RSP)")
		offset := 0
		if offsetText != "" {
			offset, _ = strconv.Atoi(offsetText)
		}
		shiftedOffset = offset + bias
		foundStackRef = true
		return fmt.Sprintf("%d(RSP)", shiftedOffset)
	})
	if match := arm64RSPBaseAdd.FindStringSubmatch(rewritten); match != nil {
		offset, _ := strconv.Atoi(match[1])
		rewritten = fmt.Sprintf("ADD $%d, RSP, %s", offset+bias, match[2])
	} else if match := arm64RSPBaseMove.FindStringSubmatch(rewritten); match != nil {
		rewritten = fmt.Sprintf("ADD $%d, RSP, %s", bias, match[1])
	}
	if rewritten != line.Disassembled {
		line.Disassembled = rewritten
		line.Binary = nil
	}

	if foundStackRef {
		_, _, elementSize, pair := arm64PairRegisters(line)
		if pair && !arm64PairOffsetEncodable(shiftedOffset, elementSize) {
			return splitArm64StackPair(line, shiftedOffset)
		}
	}
	return []Line{line}
}

func shiftArm64CStackRefs(lines []Line, bias int) []Line {
	shifted := make([]Line, 0, len(lines))
	for _, line := range lines {
		shifted = append(shifted, shiftArm64CStackRef(line, bias)...)
	}
	return shifted
}

func arm64SavedRegMove(reg string) (goReg, store, load string, size int) {
	reg = strings.ToLower(reg)
	if reg == "xzr" {
		return "ZR", "MOVD", "MOVD", 8
	}
	if reg == "wzr" {
		return "ZR", "MOVW", "MOVW", 4
	}
	switch reg[0] {
	case 'd':
		return "F" + reg[1:], "FMOVD", "FMOVD", 8
	case 'q':
		return "F" + reg[1:], "FMOVQ", "FMOVQ", 16
	case 's':
		return "F" + reg[1:], "FMOVS", "FMOVS", 4
	case 'w':
		return "R" + reg[1:], "MOVW", "MOVW", 4
	default:
		return "R" + reg[1:], "MOVD", "MOVD", 8
	}
}

func rewriteArm64SavedPair(op *StackOp, line Line, layout *StackLayout) []Line {
	offset := layout.ResolvedOffsets[op.LineIndex]
	regs := []string{op.Reg}
	if op.Reg2 != "" {
		regs = append(regs, op.Reg2)
	}
	result := make([]Line, 0, len(regs))
	for i, reg := range regs {
		goReg, store, load, size := arm64SavedRegMove(reg)
		instruction := fmt.Sprintf("%s %s, %d(RSP)", store, goReg, offset)
		if op.Kind == StackOpPop || op.Kind == StackOpReload {
			instruction = fmt.Sprintf("%s %d(RSP), %s", load, offset, goReg)
		}
		rewritten := Line{Assembly: line.Assembly, Disassembled: instruction}
		if i == 0 {
			rewritten.Labels = line.Labels
		} else {
			rewritten.Comment = arm64SplitContinuationComment(line)
		}
		result = append(result, rewritten)
		offset += size
	}
	return result
}

func rewriteArm64FixedPair(op *StackOp, line Line, layout *StackLayout) Line {
	disassembled := arm64asm.GoSyntax(decodeArm64Line(line), 0, nil, nil)
	if space := strings.IndexByte(disassembled, ' '); space >= 0 {
		disassembled = disassembled[space+1:]
	}
	opcode := arm64PairPlan9Opcode(op.Reg, op.Kind == StackOpPop || op.Kind == StackOpReload)
	disassembled = opcode + " " + disassembled
	disassembled = arm64RSPMemoryRef.ReplaceAllString(
		disassembled, fmt.Sprintf("%d(RSP)", layout.ResolvedOffsets[op.LineIndex]),
	)
	line.Disassembled = disassembled
	line.Binary = nil
	return line
}

func amd64SavedRegName(reg string) string {
	reg = strings.ToLower(reg)
	switch reg {
	case "rbp":
		return "BP"
	case "rbx":
		return "BX"
	default:
		return strings.ToUpper(reg)
	}
}

func rewriteSavedRegisters(archInfo ArchStackInfo, op *StackOp, line Line, layout *StackLayout) []Line {
	if archInfo.Name() == "arm64" {
		return rewriteArm64SavedPair(op, line, layout)
	}

	offset := layout.ResolvedOffsets[op.LineIndex]
	reg := amd64SavedRegName(op.Reg)
	instruction := fmt.Sprintf("MOVQ %s, %d(SP)", reg, offset)
	if op.Kind == StackOpPop || op.Kind == StackOpReload {
		instruction = fmt.Sprintf("MOVQ %d(SP), %s", offset, reg)
	}
	return []Line{{Labels: line.Labels, Assembly: line.Assembly, Disassembled: instruction}}
}

// rewriteStackOps performs the second pass to rewrite stack operations
func rewriteStackOps(arch *config.Arch, archInfo ArchStackInfo, layout *StackLayout, function Function) Function {
	return rewriteStackOpsWithLinkage(arch, archInfo, layout, function, true)
}

func rewriteStackOpsWithLinkage(arch *config.Arch, archInfo ArchStackInfo, layout *StackLayout, function Function, linkageBias bool) Function {
	newLines := make([]Line, 0, len(function.Lines))
	hardwareSP := "SP"
	if archInfo.Name() == "arm64" {
		hardwareSP = "RSP"
	}

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
			case StackOpFrameSetup:
				if offset, ok := layout.FrameSetupOffsets[i]; ok {
					var instr string
					if archInfo.Name() == "arm64" {
						reg := "R" + strings.TrimLeft(op.Reg, "x")
						instr = fmt.Sprintf("MOVD RSP, %s", reg)
						if offset != 0 {
							instr = fmt.Sprintf("ADD $%d, RSP, %s", offset, reg)
						}
					} else {
						instr = fmt.Sprintf("LEAQ %d(SP), %s", offset, amd64SavedRegName(op.Reg))
					}
					line.Disassembled = instr
					line.Binary = nil
					newLines = append(newLines, line)
					continue
				}

			case StackOpPush:
				if (layout.PreserveCalleeSaved || layout.RetainedSaves[i]) && stackOpCalleeSaved(archInfo, op) {
					if archInfo.Name() == "arm64" && op.Reg2 != "" {
						newLines = append(newLines, rewriteArm64FixedPair(op, line, layout))
					} else {
						newLines = append(newLines, rewriteSavedRegisters(archInfo, op, line, layout)...)
					}
					continue
				}
				// Non-callee-saved push: rewrite to MOV
				if !stackOpCalleeSaved(archInfo, op) {
					if archInfo.Name() == "arm64" && op.Immediate > 0 {
						// A fixed Go frame has already performed the allocation. Remove
						// writeback now; the final linkage/arena rebase will retain this
						// fixed pair if its scaled immediate remains encodable.
						if op.Reg2 != "" {
							newLines = append(newLines, rewriteArm64FixedPair(op, line, layout))
						} else {
							newLines = append(newLines, rewriteSavedRegisters(archInfo, op, line, layout)...)
						}
						continue
					}
					movInstr := arch.MovInstr[8]
					parts := strings.Fields(line.Disassembled)
					var reg string
					if len(parts) > 1 {
						reg = parts[1]
					} else {
						reg = strings.ToUpper(op.Reg)
					}
					offset := layout.ResolvedOffsets[i]
					instr := fmt.Sprintf("%s %s, %d(%s)", movInstr, reg, offset, hardwareSP)
					lineCpy := line
					lineCpy.Disassembled = instr
					lineCpy.Binary = nil
					newLines = append(newLines, lineCpy)
					continue
				}

			case StackOpPop:
				if (layout.PreserveCalleeSaved || layout.RetainedSaves[i]) && stackOpCalleeSaved(archInfo, op) {
					if archInfo.Name() == "arm64" && op.Reg2 != "" {
						newLines = append(newLines, rewriteArm64FixedPair(op, line, layout))
					} else {
						newLines = append(newLines, rewriteSavedRegisters(archInfo, op, line, layout)...)
					}
					continue
				}
				// Non-callee-saved pop: rewrite to MOV
				if !stackOpCalleeSaved(archInfo, op) {
					if archInfo.Name() == "arm64" && op.Offset > 0 {
						// Post-indexed load that also deallocates (ldr x0, [sp], #N).
						if op.Reg2 != "" {
							newLines = append(newLines, rewriteArm64FixedPair(op, line, layout))
						} else {
							newLines = append(newLines, rewriteSavedRegisters(archInfo, op, line, layout)...)
						}
						continue
					}
					movInstr := arch.MovInstr[8]
					parts := strings.Fields(line.Disassembled)
					var reg string
					if len(parts) > 1 {
						reg = parts[1]
					} else {
						reg = strings.ToUpper(op.Reg)
					}
					offset := layout.ResolvedOffsets[i]
					instr := fmt.Sprintf("%s %d(%s), %s", movInstr, offset, hardwareSP, reg)
					lineCpy := line
					lineCpy.Disassembled = instr
					lineCpy.Binary = nil
					newLines = append(newLines, lineCpy)
					continue
				}

			case StackOpSpill, StackOpReload:
				if archInfo.Name() == "arm64" && op.Reg2 != "" {
					// Keep the fixed pair provisionally. The final linkage/arena bias
					// determines whether its scaled immediate fits or it must be split.
					newLines = append(newLines, rewriteArm64FixedPair(op, line, layout))
					continue
				}
				if layout.PreserveCalleeSaved && stackOpCalleeSaved(archInfo, op) {
					newLines = append(newLines, rewriteSavedRegisters(archInfo, op, line, layout)...)
					continue
				}
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
	if archInfo.Name() == "arm64" && layout.GoFrameSize > 0 && linkageBias {
		// cmd/asm adds a 16-byte linkage/alignment area to framed ARM64
		// functions. Keeping the emulated C SP above both words also retains
		// the 16-byte alignment assumed by Clang's vector spills.
		function.Lines = shiftArm64CStackRefs(function.Lines, 2*archInfo.PtrSize())
	}

	return function
}

// reserveInternalStackFrames flattens every internal C-ABI helper frame into
// its owning Go-visible function's frame. Internal helper graphs must be
// disjoint between exported roots, which lets each helper be rebased once into
// a root-specific arena without making unrelated entry points reserve it.
var amd64RSPMemoryRef = regexp.MustCompile(`(?i)(-?(?:0x[0-9a-f]+|[0-9]+))?\((?:R?SP)\)`)
var amd64RSPAdd = regexp.MustCompile(`^ADDQ (?:R?SP), ([A-Z][A-Z0-9]*)$`)
var amd64RSPMove = regexp.MustCompile(`^MOVQ (?:R?SP), ([A-Z][A-Z0-9]*)$`)

// amd64ModRMOffset returns the index of the ModRM byte of a 64-bit x86
// instruction encoding (-1 if the encoding is not understood) and whether
// the instruction uses an EVEX prefix, whose 8-bit displacements are scaled.
func amd64ModRMOffset(code []byte) (int, bool) {
	i := 0
	for i < len(code) {
		switch code[i] {
		case 0x66, 0x67, 0xF2, 0xF3, 0xF0, 0x2E, 0x36, 0x3E, 0x26, 0x64, 0x65:
			i++
			continue
		}
		break
	}
	if i < len(code) && code[i]&0xF0 == 0x40 { // REX
		i++
	}
	if i >= len(code) {
		return -1, false
	}
	evex := false
	switch code[i] {
	case 0xC5: // VEX2
		i += 3
	case 0xC4: // VEX3
		i += 4
	case 0x62: // EVEX
		evex = true
		i += 5
	case 0x0F:
		i++
		if i < len(code) && (code[i] == 0x38 || code[i] == 0x3A) {
			i++
		}
		i++
	default:
		i++
	}
	if i >= len(code) {
		return -1, false
	}
	return i, evex
}

// patchAmd64RSPDisplacement re-encodes an instruction whose memory operand is
// RSP-relative so that its displacement is shifted by bias. It verifies the
// ModRM/SIB location against the decoder before touching any byte.
func patchAmd64RSPDisplacement(line Line, bias int64) ([]string, bool) {
	code, err := hex.DecodeString(strings.Join(line.Binary, ""))
	if err != nil {
		return nil, false
	}
	inst, err := x86asm.Decode(code, 64)
	if err != nil || inst.Len != len(code) {
		return nil, false
	}
	var disp int64
	found := false
	for _, arg := range inst.Args {
		if mem, ok := arg.(x86asm.Mem); ok && mem.Base == x86asm.RSP {
			disp = mem.Disp
			found = true
			break
		}
	}
	if !found {
		return nil, false
	}

	modrm, evex := amd64ModRMOffset(code)
	if modrm < 0 || modrm+1 >= len(code) {
		return nil, false
	}
	mod := code[modrm] >> 6
	if code[modrm]&7 != 4 || code[modrm+1]&7 != 4 || mod == 3 {
		return nil, false // not a SIB form with an RSP/R12 base
	}
	dispOffset := modrm + 2
	var dispLen int
	var encodedDisp int64
	switch mod {
	case 0:
		dispLen = 0
	case 1:
		dispLen = 1
		if dispOffset+1 > len(code) {
			return nil, false
		}
		encodedDisp = int64(int8(code[dispOffset]))
	case 2:
		dispLen = 4
		if dispOffset+4 > len(code) {
			return nil, false
		}
		encodedDisp = int64(int32(uint32(code[dispOffset]) | uint32(code[dispOffset+1])<<8 |
			uint32(code[dispOffset+2])<<16 | uint32(code[dispOffset+3])<<24))
	}
	if encodedDisp != disp {
		return nil, false
	}

	newDisp := disp + bias
	patched := append([]byte(nil), code[:modrm]...)
	if newDisp == 0 {
		patched = append(patched, code[modrm]&0x3F, code[modrm+1])
	} else if !evex && newDisp >= -128 && newDisp <= 127 {
		patched = append(patched, code[modrm]&0x3F|0x40, code[modrm+1], byte(int8(newDisp)))
	} else if newDisp >= -1<<31 && newDisp < 1<<31 {
		patched = append(patched, code[modrm]&0x3F|0x80, code[modrm+1],
			byte(newDisp), byte(newDisp>>8), byte(newDisp>>16), byte(newDisp>>24))
	} else {
		return nil, false
	}
	patched = append(patched, code[dispOffset+dispLen:]...)

	if reencoded, err := x86asm.Decode(patched, 64); err != nil || reencoded.Op != inst.Op || reencoded.Len != len(patched) {
		return nil, false
	}
	binary := make([]string, len(patched))
	for i, b := range patched {
		binary[i] = fmt.Sprintf("%02x", b)
	}
	return binary, true
}

func shiftAmd64CStackRef(line Line, bias int) Line {
	if bias == 0 || (!strings.Contains(line.Disassembled, "SP") && !strings.Contains(line.Disassembled, "sp")) {
		return line
	}

	// Encodings are retained when cmd/asm has no trusted spelling for the
	// instruction. Rebase those in place instead of forcing the decoder's text.
	if len(line.Binary) > 0 {
		if patched, ok := patchAmd64RSPDisplacement(line, int64(bias)); ok {
			line.Binary = patched
			line.Disassembled = amd64RSPMemoryRef.ReplaceAllStringFunc(line.Disassembled, func(ref string) string {
				match := amd64RSPMemoryRef.FindStringSubmatch(ref)
				offset := int64(0)
				if match[1] != "" {
					offset, _ = strconv.ParseInt(match[1], 0, 64)
				}
				return fmt.Sprintf("%d(SP)", int(offset)+bias)
			})
			return line
		}
	}
	rewritten := amd64RSPMemoryRef.ReplaceAllStringFunc(line.Disassembled, func(ref string) string {
		match := amd64RSPMemoryRef.FindStringSubmatch(ref)
		offset := int64(0)
		if match[1] != "" {
			offset, _ = strconv.ParseInt(match[1], 0, 64)
		}
		return fmt.Sprintf("%d(SP)", int(offset)+bias)
	})
	if match := amd64RSPAdd.FindStringSubmatch(rewritten); match != nil {
		rewritten = fmt.Sprintf("LEAQ %d(SP), %s", bias, match[1])
	}
	if match := amd64RSPMove.FindStringSubmatch(rewritten); match != nil {
		rewritten = fmt.Sprintf("LEAQ %d(SP), %s", bias, match[1])
	}
	if strings.HasPrefix(rewritten, "MOVZX ") {
		switch {
		case strings.Contains(line.Assembly, "byte ptr"):
			rewritten = "MOVBQZX" + strings.TrimPrefix(rewritten, "MOVZX")
		case strings.Contains(line.Assembly, "word ptr"):
			rewritten = "MOVWQZX" + strings.TrimPrefix(rewritten, "MOVZX")
		}
	} else if strings.HasPrefix(rewritten, "MOVSX ") {
		switch {
		case strings.Contains(line.Assembly, "byte ptr"):
			rewritten = "MOVBQSX" + strings.TrimPrefix(rewritten, "MOVSX")
		case strings.Contains(line.Assembly, "word ptr"):
			rewritten = "MOVWQSX" + strings.TrimPrefix(rewritten, "MOVSX")
		}
	} else if strings.HasPrefix(rewritten, "CMOV") {
		// GNU/LLVM spell conditional moves without an operand-size suffix;
		// cmd/asm requires it before the condition code (CMOVQNE, etc.).
		space := strings.IndexByte(rewritten, ' ')
		if space > len("CMOV") {
			suffix := ""
			switch {
			case strings.Contains(line.Assembly, "qword ptr"):
				suffix = "Q"
			case strings.Contains(line.Assembly, "dword ptr"):
				suffix = "L"
			case strings.Contains(line.Assembly, "word ptr"):
				suffix = "W"
			}
			if suffix != "" {
				rewritten = "CMOV" + suffix + rewritten[len("CMOV"):]
			}
		}
	}
	if rewritten != line.Disassembled {
		line.Disassembled = rewritten
		line.Binary = nil
	}
	return line
}

// BranchTarget returns the function targeted by a translated direct call or
// tail call (`CALL name<>(SB)` / `JMP name<>(SB)`).
func BranchTarget(line Line) (string, bool) {
	fields := strings.Fields(line.Disassembled)
	if len(fields) != 2 || (fields[0] != "CALL" && fields[0] != "JMP") {
		return "", false
	}
	target := strings.TrimSuffix(fields[1], "<>(SB)")
	if target == fields[1] {
		return "", false
	}
	return target, true
}

func assignInternalFunctionOwners(functions []Function) ([]int, error) {
	functionByName := make(map[string]int, len(functions))
	for i := range functions {
		functionByName[functions[i].Name] = i
	}

	// A translated C CALL always uses the C register ABI. Calling an exported
	// function that has a Go ABI entry sequence would therefore be invalid.
	for i := range functions {
		for _, line := range functions[i].Lines {
			targetName, ok := BranchTarget(line)
			if !ok {
				continue
			}
			targetIdx, exists := functionByName[targetName]
			if exists && !functions[targetIdx].Internal {
				return nil, fmt.Errorf("C-ABI call from %q to exported function %q is unsupported", functions[i].Name, targetName)
			}
		}
	}

	owners := make([]int, len(functions))
	for i := range owners {
		owners[i] = -1
	}
	ownerPaths := make([][]string, len(functions))

	for rootIdx := range functions {
		if functions[rootIdx].Internal {
			continue
		}
		visited := make(map[int]bool)
		var visit func(int, []string) error
		visit = func(callerIdx int, path []string) error {
			for _, line := range functions[callerIdx].Lines {
				targetName, ok := BranchTarget(line)
				if !ok {
					continue
				}
				targetIdx, exists := functionByName[targetName]
				if !exists || !functions[targetIdx].Internal {
					continue
				}

				targetPath := append(append([]string(nil), path...), targetName)
				if owner := owners[targetIdx]; owner != -1 && owner != rootIdx {
					return fmt.Errorf(
						"internal helper %q is reachable from multiple exported functions: %s; %s",
						targetName, strings.Join(ownerPaths[targetIdx], " -> "), strings.Join(targetPath, " -> "),
					)
				}
				if owners[targetIdx] == -1 {
					owners[targetIdx] = rootIdx
					ownerPaths[targetIdx] = targetPath
				}
				if visited[targetIdx] {
					continue
				}
				visited[targetIdx] = true
				if err := visit(targetIdx, targetPath); err != nil {
					return err
				}
			}
			return nil
		}
		if err := visit(rootIdx, []string{functions[rootIdx].Name}); err != nil {
			return nil, err
		}
	}
	return owners, nil
}

// helperCallDepths returns, for every internal helper owned by root, the
// minimum and maximum number of x86 CALL return addresses that sit between
// the root's SP and the helper's SP. A tail call (JMP) reuses the caller's
// return address, so it does not add depth.
func helperCallDepths(functions []Function, rootIdx int, owners []int) (minDepth, maxDepth map[int]int) {
	functionByName := make(map[string]int, len(functions))
	for i := range functions {
		functionByName[functions[i].Name] = i
	}
	minDepth = make(map[int]int)
	maxDepth = make(map[int]int)
	var visit func(callerIdx, depth int)
	visit = func(callerIdx, depth int) {
		for _, line := range functions[callerIdx].Lines {
			target, ok := BranchTarget(line)
			if !ok {
				continue
			}
			targetIdx, exists := functionByName[target]
			if !exists || owners[targetIdx] != rootIdx {
				continue
			}
			targetDepth := depth
			if strings.HasPrefix(line.Disassembled, "CALL ") {
				targetDepth++
			}
			lo, seen := minDepth[targetIdx]
			if seen && targetDepth >= lo && targetDepth <= maxDepth[targetIdx] {
				continue
			}
			if !seen || targetDepth < lo {
				minDepth[targetIdx] = targetDepth
			}
			if !seen || targetDepth > maxDepth[targetIdx] {
				maxDepth[targetIdx] = targetDepth
			}
			visit(targetIdx, targetDepth)
		}
	}
	visit(rootIdx, 0)
	return minDepth, maxDepth
}

// amd64NeedsAlignedStack reports whether a helper's code depends on the
// alignment of its stack pointer: it asked for explicit realignment, or it
// uses legacy SSE instructions with 128-bit stack operands (aligned moves are
// rewritten to their unaligned forms and do not count).
func amd64NeedsAlignedStack(function *Function) bool {
	if function.StackAlign > 0 {
		return true
	}
	archInfo := newAmd64StackInfo()
	for _, line := range function.Lines {
		if !amd64LegacySSEStackOperand.MatchString(line.Assembly) {
			continue
		}
		if archInfo.ToUnalignedInsn(strings.ToUpper(strings.Fields(line.Assembly)[0])) == nil {
			return true
		}
	}
	return false
}

func reserveInternalStackFrames(arch *config.Arch, functions []Function) ([]Function, error) {
	owners, err := assignInternalFunctionOwners(functions)
	if err != nil {
		return nil, err
	}

	for rootIdx := range functions {
		if functions[rootIdx].Internal {
			continue
		}

		hasHelpers := false
		for helperIdx := range functions {
			if owners[helperIdx] == rootIdx {
				hasHelpers = true
				break
			}
		}
		if !hasHelpers {
			continue
		}

		rootLocals := functions[rootIdx].LocalsSize
		if arch.Name == "arm64" {
			// BL does not touch SP, and Go keeps RSP 16-byte aligned, so every
			// helper sees the same SP as its root. Slots start above the root's
			// locals and the 16-byte linkage area cmd/asm reserves.
			reserved := 0
			for helperIdx := range functions {
				if owners[helperIdx] != rootIdx || functions[helperIdx].HiddenStackSize == 0 {
					continue
				}
				reserved += -reserved & 15
				functions[helperIdx].Lines = shiftArm64CStackRefs(functions[helperIdx].Lines, rootLocals+16+reserved)
				reserved += functions[helperIdx].HiddenStackSize
			}
			reserved += -reserved & 15
			functions[rootIdx].LocalsSize = rootLocals + reserved
			continue
		}

		// Every x86 CALL pushes an 8-byte return address, so a helper's SP is
		// 8*depth below the root's realigned SP. The C code expects its own SP to
		// be aligned after the prologue, so each helper's slot bias is chosen so
		// that bias-8*depth is aligned for every depth the helper is reached at.
		minDepth, maxDepth := helperCallDepths(functions, rootIdx, owners)
		reserved := rootLocals
		for helperIdx := range functions {
			if owners[helperIdx] != rootIdx || functions[helperIdx].HiddenStackSize == 0 {
				continue
			}
			helper := &functions[helperIdx]
			align := 16
			if helper.StackAlign > align {
				align = helper.StackAlign
			}
			if align > functions[rootIdx].StackAlign {
				// The root's realigned SP is the base every helper slot is
				// measured from, so it must satisfy the strictest helper.
				functions[rootIdx].StackAlign = align
			}
			lo, hi := minDepth[helperIdx], maxDepth[helperIdx]
			if (hi-lo)*8%align != 0 && amd64NeedsAlignedStack(helper) {
				return nil, fmt.Errorf(
					"internal helper %q is called at depths %d and %d, which cannot both satisfy its %d-byte stack alignment",
					helper.Name, lo, hi, align,
				)
			}
			reserved += -reserved & (align - 1)
			bias := reserved + 8*hi
			for lineIdx := range helper.Lines {
				helper.Lines[lineIdx] = shiftAmd64CStackRef(helper.Lines[lineIdx], bias)
			}
			reserved += helper.HiddenStackSize + 8*(hi-lo)
		}
		reserved += -reserved & 15
		functions[rootIdx].LocalsSize = reserved
	}
	return functions, nil
}

// checkStackUnified is the new unified stack checking function
func checkStackUnified(arch *config.Arch, function Function) (Function, error) {
	archInfo := getArchStackInfo(arch)

	// Pass 1: Analyze stack layout
	layout, err := analyzeStackLayout(archInfo, function.Lines, function.Internal || function.PreserveCABI)
	if err != nil {
		return function, err
	}

	warnStackAlignment(archInfo, layout, function)
	if layout.Alignment < 0 && archInfo.Name() == "amd64" {
		// Go keeps ARM64 RSP 16-byte aligned and NEON has no alignment faults;
		// only AMD64 roots need the realignment prologue (see realignAmd64Roots).
		function.StackAlign = int(-layout.Alignment)
	}

	// Pass 2: Rewrite stack operations
	return rewriteStackOps(arch, archInfo, layout, function), nil
}

// amd64LegacySSEStackOperand matches a non-VEX SSE instruction with a stack
// memory operand. Unlike VEX/EVEX encodings, legacy SSE requires 16-byte
// aligned memory operands, which a Go frame does not guarantee.
var amd64LegacySSEStackOperand = regexp.MustCompile(`^[a-uw-z]\w*\s.*\bxmmword ptr \[rsp\b`)

// warnStackAlignment reports stack alignment the fixed Go frame cannot
// provide. AMD64 roots realign SP to whatever the compiler asked for (see
// realignAmd64Roots); ARM64 RSP is 16-byte aligned and larger requests are
// dropped, which is harmless for NEON/SVE but not for code that inspects
// addresses.
func warnStackAlignment(archInfo ArchStackInfo, layout *StackLayout, function Function) {
	if archInfo.Name() != "arm64" || layout.Alignment == 0 || -layout.Alignment <= 16 {
		return
	}
	fnName := function.Name
	if fnName == "" {
		fnName = "[unknown]"
	}
	fmt.Fprintf(os.Stderr, "WARN: %s: requests %d-byte stack alignment, Go only guarantees 16 bytes on arm64\n", fnName, -layout.Alignment)
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

func decodeAmd64Line(line Line) x86asm.Inst {
	inst, ok := tryDecodeAmd64Line(line)
	if !ok {
		panic(fmt.Errorf("failed to decode instruction %q", strings.Join(line.Binary, "")))
	}
	return inst
}

// tryDecodeAmd64Line decodes the machine code of a line; it reports false for
// encodings x86asm does not understand (notably EVEX/AVX-512).
func tryDecodeAmd64Line(line Line) (x86asm.Inst, bool) {
	code, err := hex.DecodeString(strings.Join(line.Binary, ""))
	if err != nil {
		return x86asm.Inst{}, false
	}
	inst, err := x86asm.Decode(code, 64)
	if err != nil || inst.Len != len(code) {
		return x86asm.Inst{}, false
	}
	return inst, true
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
