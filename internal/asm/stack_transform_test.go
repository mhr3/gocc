package asm

import (
	"encoding/binary"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/mhr3/gocc/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStackManipulationAmd64(t *testing.T) {
	testFn := Function{
		Lines: []Line{
			{Assembly: "push\trbp", Disassembled: "PUSHQ BP", Binary: binaryFromHex("55")},
			{Assembly: "mov\trbp, rsp", Disassembled: "MOVQ SP, BP", Binary: binaryFromHex("48 89 e5")},
			{Assembly: "push\trbx", Disassembled: "PUSHQ BX", Binary: binaryFromHex("53")},
			{Assembly: "and\trsp, -8", Disassembled: "ANDQ $-0x8, SP", Binary: binaryFromHex("48 83 e4 f8")},
			{Assembly: "mov\teax, 16", Disassembled: "MOVL $0x10, AX"},
			{Assembly: "lea\trsp, [rbp - 8]", Disassembled: "LEAQ -0x8(BP), SP", Binary: binaryFromHex("48 8d 65 f8")},
			{Assembly: "pop\trbx", Disassembled: "POPQ BX", Binary: binaryFromHex("5b")},
			{Assembly: "pop\trbp", Disassembled: "POPQ BP", Binary: binaryFromHex("5d")},
			{Assembly: "ret", Disassembled: "RET"},
		},
	}

	modified := checkStackUnified(config.AMD64(), testFn)

	require.Equal(t, 0, modified.LocalsSize)

	require.Len(t, modified.Lines, 9)
	assert.Equal(t, "NOP", modified.Lines[0].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[1].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[2].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[3].Disassembled)
	assert.Equal(t, testFn.Lines[4].Disassembled, modified.Lines[4].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[5].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[6].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[7].Disassembled)
	assert.Equal(t, "RET", modified.Lines[8].Disassembled)
}

func TestStackNotPopAmd64(t *testing.T) {
	testFn := Function{
		Lines: []Line{
			{Assembly: "push\trbp", Disassembled: "PUSHQ BP", Binary: binaryFromHex("55")},
			{Assembly: "mov\trbp, rsp", Disassembled: "MOVQ SP, BP", Binary: binaryFromHex("48 89 e5")},
			{Assembly: "popcnt\trdx, qword ptr [rdi + 8*rcx]", Disassembled: "POPCNTQ 0(DI)(CX*8), DX", Binary: binaryFromHex("f3 48 0f b8 14 cf")},
			{Assembly: "pop\trbp", Disassembled: "POPQ BP", Binary: binaryFromHex("5d")},
			{Assembly: "ret", Disassembled: "RET"},
		},
	}

	modified := checkStackUnified(config.AMD64(), testFn)

	require.Equal(t, 0, modified.LocalsSize)

	require.Len(t, modified.Lines, 5)
	assert.Equal(t, "NOP", modified.Lines[0].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[1].Disassembled)
	assert.Equal(t, "POPCNTQ 0(DI)(CX*8), DX", modified.Lines[2].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[3].Disassembled)
	assert.Equal(t, "RET", modified.Lines[4].Disassembled)
}

func TestStackGrowthAmd64(t *testing.T) {
	/*
		     9b5: 55                            push    rbp
		     9b6: 48 89 e5                      mov     rbp, rsp
		     9b9: 41 57                         push    r15
		     9bb: 41 56                         push    r14
		     9bd: 41 55                         push    r13
		     9bf: 41 54                         push    r12
		     9c1: 53                            push    rbx
		     9c2: 48 83 e4 f8                   and     rsp, -0x8
		     9c6: 48 83 ec 18                   sub     rsp, 0x18
		     9f1: 48 89 4c 24 08                mov     qword ptr [rsp + 0x8], rcx
		     a0a: 4c 89 44 24 10                mov     qword ptr [rsp + 0x10], r8
		     b5d: 4c 8b 44 24 10                mov     r8, qword ptr [rsp + 0x10]
		     b62: 48 8b 4c 24 08                mov     rcx, qword ptr [rsp + 0x8]
			 ...
		     d37: 48 8d 65 d8                   lea     rsp, [rbp - 0x28]
		     d3b: 5b                            pop     rbx
		     d3c: 41 5c                         pop     r12
		     d3e: 41 5d                         pop     r13
		     d40: 41 5e                         pop     r14
		     d42: 41 5f                         pop     r15
		     d44: 5d                            pop     rbp
		     d45: c3                            ret
	*/
	testFn := Function{
		Lines: []Line{
			{Assembly: "push\trbp", Disassembled: "PUSHQ BP", Binary: binaryFromHex("55")},
			{Assembly: "mov\trbp, rsp", Disassembled: "MOVQ SP, BP", Binary: binaryFromHex("48 89 e5")},
			{Assembly: "push\tr15", Disassembled: "PUSHQ R15", Binary: binaryFromHex("41 57")},
			{Assembly: "push\tr14", Disassembled: "PUSHQ R14", Binary: binaryFromHex("41 56")},
			{Assembly: "push\tr13", Disassembled: "PUSHQ R13", Binary: binaryFromHex("41 55")},
			{Assembly: "push\tr12", Disassembled: "PUSHQ R12", Binary: binaryFromHex("41 54")},
			{Assembly: "push\trbx", Disassembled: "PUSHQ BX", Binary: binaryFromHex("53")},
			{Assembly: "and\trsp, -8", Disassembled: "ANDQ $-0x8, SP", Binary: binaryFromHex("48 83 e4 f8")},
			{Assembly: "sub\trsp, 24", Disassembled: "SUBQ $0x18, SP", Binary: binaryFromHex("48 83 ec 18")},
			{Assembly: "mov\tqword ptr [rsp + 8], rcx", Disassembled: "MOVQ CX, 0x8(SP)", Binary: binaryFromHex("48 89 4c 24 08")},
			{Assembly: "mov\tqword ptr [rsp + 16], r8", Disassembled: "MOVQ R8, 0x10(SP)", Binary: binaryFromHex("4c 89 44 24 10")},
			{Assembly: "mov\tr8, qword ptr [rsp + 16]", Disassembled: "MOVQ 0x10(SP), R8", Binary: binaryFromHex("4c 8b 44 24 10")},
			{Assembly: "mov\trcx, qword ptr [rsp + 8]", Disassembled: "MOVQ 0x8(SP), CX", Binary: binaryFromHex("48 8b 4c 24 08")},
			{Assembly: "lea\trsp, [rbp - 40]", Disassembled: "LEAQ -0x28(BP), SP", Binary: binaryFromHex("48 8d 65 d8")},
			{Assembly: "pop\trbx", Disassembled: "POPQ BX", Binary: binaryFromHex("5b")},
			{Assembly: "pop\tr12", Disassembled: "POPQ R12", Binary: binaryFromHex("41 5c")},
			{Assembly: "pop\tr13", Disassembled: "POPQ R13", Binary: binaryFromHex("41 5d")},
			{Assembly: "pop\tr14", Disassembled: "POPQ R14", Binary: binaryFromHex("41 5e")},
			{Assembly: "pop\tr15", Disassembled: "POPQ R15", Binary: binaryFromHex("41 5f")},
			{Assembly: "pop\trbp", Disassembled: "POPQ BP", Binary: binaryFromHex("5d")},
			{Assembly: "ret", Disassembled: "RET"},
		},
	}

	modified := checkStackUnified(config.AMD64(), testFn)

	require.Equal(t, 24, modified.LocalsSize)

	require.Len(t, modified.Lines, 21)
	for i := 0; i < 9; i++ {
		assert.Equal(t, "NOP", modified.Lines[i].Disassembled)
	}
	assert.True(t, strings.HasPrefix(modified.Lines[9].Disassembled, "MOV"))
	assert.True(t, strings.HasPrefix(modified.Lines[10].Disassembled, "MOV"))
	assert.True(t, strings.HasPrefix(modified.Lines[11].Disassembled, "MOV"))
	assert.True(t, strings.HasPrefix(modified.Lines[12].Disassembled, "MOV"))
	assert.Equal(t, testFn.Lines[9].Disassembled, modified.Lines[9].Disassembled)
	for i := 13; i < 20; i++ {
		assert.Equal(t, "NOP", modified.Lines[i].Disassembled)
	}
	assert.Equal(t, "RET", modified.Lines[20].Disassembled)
}

func TestQuickStackManipulationArm64(t *testing.T) {
	testFn := Function{
		Lines: []Line{
			{Assembly: "stp	x29, x30, [sp, #-48]!", Binary: wordToLineBinary(0xa9bd7bfd)},
			{Assembly: "cmp	x1, x3", Disassembled: "CMP X1, X3", Binary: wordToLineBinary(0xeb03003f)},
			{Assembly: "str	x21, [sp, #16]", Disassembled: "MOVD R21, 16(RSP)", Binary: wordToLineBinary(0xf9000bf5)},
			{Assembly: "mov	x29, sp", Binary: wordToLineBinary(0x910003fd)},
			{Assembly: "stp	x20, x19, [sp, #32]", Binary: wordToLineBinary(0xa9024ff4)},
			{Assembly: "b.ge\t.LBB3_3", Disassembled: "B.GE .LBB3_3", Binary: wordToLineBinary(0x540000ea)},
			{Assembly: "mov	x8, #-1", Disassembled: "MOVD X8, #-1", Binary: wordToLineBinary(0x92800008)},
			{Assembly: "ldp	x20, x19, [sp, #32]", Binary: wordToLineBinary(0xa9424ff4)},
			{Assembly: "mov	x0, x8", Disassembled: "MOV X0, X8", Binary: wordToLineBinary(0xaa0803e0)},
			{Assembly: "ldr	x21, [sp, #16]", Binary: wordToLineBinary(0xf9400bf5)},
			{Assembly: "ldp	x29, x30, [sp], #48", Binary: wordToLineBinary(0xa8c37bfd)},
			{Assembly: "ret", Binary: wordToLineBinary(0xd65f03c0)},
		},
	}

	modified := checkStackUnified(config.ARM64(), testFn)

	// All saves are callee-saved and NOPed, so GoFrameSize should be 0
	require.Equal(t, 0, modified.LocalsSize)

	require.Len(t, modified.Lines, 12)
	assert.Equal(t, "NOP", modified.Lines[0].Disassembled)
	assert.NotEqual(t, "NOP", modified.Lines[1].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[2].Disassembled)
}

func TestStackManipulationArm64(t *testing.T) {
	testFn := Function{
		Lines: []Line{
			{Assembly: "stp	x29, x30, [sp, #-80]!", Binary: wordToLineBinary(0xa9bb7bfd)},
			{Assembly: "sub	x9, sp, #16", Binary: wordToLineBinary(0xd10043e9)},
			{Assembly: "stp	x26, x25, [sp, #16]", Binary: wordToLineBinary(0xa90167fa)},
			{Assembly: "stp	x24, x23, [sp, #32]", Binary: wordToLineBinary(0xa9025ff8)},
			{Assembly: "mov	x29, sp", Binary: wordToLineBinary(0x910003fd)},
			{Assembly: "stp	x22, x21, [sp, #48]", Binary: wordToLineBinary(0xa90357f6)},
			{Assembly: "stp	x20, x19, [sp, #64]", Binary: wordToLineBinary(0xa9044ff4)},
			{Assembly: "and	sp, x9, #0xfffffffffffffff8", Binary: wordToLineBinary(0x927df13f)},

			{Assembly: "mov	sp, x29", Binary: wordToLineBinary(0x910003bf)},
			{Assembly: "ldp	x20, x19, [sp, #64]", Binary: wordToLineBinary(0xa9444ff4)},
			{Assembly: "ldp	x22, x21, [sp, #48]", Binary: wordToLineBinary(0xa94357f6)},
			{Assembly: "ldp	x24, x23, [sp, #32]", Binary: wordToLineBinary(0xa9425ff8)},
			{Assembly: "ldp	x26, x25, [sp, #16]", Binary: wordToLineBinary(0xa94167fa)},
			{Assembly: "ldp	x29, x30, [sp], #80", Binary: wordToLineBinary(0xa8c57bfd)},
			{Assembly: "ret", Disassembled: "RET", Binary: wordToLineBinary(0xd65f03c0)},
		},
	}

	modified := checkStackUnified(config.ARM64(), testFn)

	require.Equal(t, 80, modified.LocalsSize)

	require.Len(t, modified.Lines, 15)
	assert.Equal(t, "NOP", modified.Lines[0].Disassembled)
	assert.Equal(t, "MOVD $stack-64(SP), R9", modified.Lines[1].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[2].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[4].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[5].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[6].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[7].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[12].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[13].Disassembled)
	assert.Equal(t, "RET", modified.Lines[14].Disassembled)
}

func TestStackRegisterSavingArm64(t *testing.T) {
	testFn := Function{
		Lines: []Line{
			{Assembly: "stp	x29, x30, [sp, #-32]!", Binary: wordToLineBinary(0xa9be7bfd)},
			{Assembly: "stp	x20, x19, [sp, #16]", Binary: wordToLineBinary(0xa9014ff4)},
			{Assembly: "mov	x29, sp", Binary: wordToLineBinary(0x910003fd)},

			{Assembly: "ldp	x20, x19, [sp, #16]", Binary: wordToLineBinary(0xa9414ff4)},
			{Assembly: "ldp	x29, x30, [sp], #32", Binary: wordToLineBinary(0xa8c27bfd)},
			{Assembly: "ret", Disassembled: "RET", Binary: wordToLineBinary(0xd65f03c0)},
		},
	}

	modified := checkStackUnified(config.ARM64(), testFn)

	// All saves are callee-saved and NOPed, so GoFrameSize should be 0
	require.Equal(t, 0, modified.LocalsSize)

	require.Len(t, modified.Lines, 6)
	assert.Equal(t, "NOP", modified.Lines[0].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[1].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[2].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[3].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[4].Disassembled)
	assert.Equal(t, "RET", modified.Lines[5].Disassembled)
}

func wordToLineBinary(word uint32) []string {
	buf := [4]byte{}
	binary.LittleEndian.PutUint32(buf[:], word)
	s := hex.EncodeToString(buf[:])
	return []string{s[:2], s[2:4], s[4:6], s[6:]}
}

func TestReturnInject(t *testing.T) {
	testFn := Function{
		Params: []Param{
			{Type: "long", Name: "l"},
		},
		Ret: &Param{Type: "long"},
		Lines: []Line{
			{Assembly: "pushq\trbp", Disassembled: "PUSHQ BP"},
			{Assembly: "mov\trbp, rsp", Disassembled: "MOVQ SP, BP"},
			{Assembly: "mov\teax, 16", Disassembled: "MOVL $0x10, AX"},
			{Assembly: "pop\trbp", Disassembled: "POPQ BP"},
			{Assembly: "ret", Disassembled: "RET"},
		},
	}

	modified := storeReturnValue(config.AMD64(), testFn)

	require.Len(t, modified.Lines, 6)
	assert.True(t, strings.HasPrefix(modified.Lines[4].Disassembled, "MOV"))
	assert.Contains(t, modified.Lines[4].Disassembled, "ret+8(FP)")
	assert.Equal(t, "RET", modified.Lines[5].Disassembled)
}

// Tests for the unified stack transform implementation

func TestUnifiedStackAnalysisAmd64(t *testing.T) {
	testFn := Function{
		Lines: []Line{
			{Assembly: "push\trbp", Disassembled: "PUSHQ BP", Binary: binaryFromHex("55")},
			{Assembly: "mov\trbp, rsp", Disassembled: "MOVQ SP, BP", Binary: binaryFromHex("48 89 e5")},
			{Assembly: "sub\trsp, 32", Disassembled: "SUBQ $0x20, SP", Binary: binaryFromHex("48 83 ec 20")},
			{Assembly: "mov\teax, 16", Disassembled: "MOVL $0x10, AX"},
			{Assembly: "add\trsp, 32", Disassembled: "ADDQ $0x20, SP", Binary: binaryFromHex("48 83 c4 20")},
			{Assembly: "pop\trbp", Disassembled: "POPQ BP", Binary: binaryFromHex("5d")},
			{Assembly: "ret", Disassembled: "RET"},
		},
	}

	archInfo := newAmd64StackInfo()
	layout := analyzeStackLayout(archInfo, testFn.Lines)

	assert.True(t, layout.FramePointerUsed)
	assert.Equal(t, 32, layout.LocalsSize)
	assert.Equal(t, 32, layout.GoFrameSize)
	assert.True(t, layout.NopIndices[0]) // push rbp
	assert.True(t, layout.NopIndices[1]) // mov rbp, rsp
	assert.True(t, layout.NopIndices[2]) // sub rsp, 32
	assert.True(t, layout.NopIndices[4]) // add rsp, 32
	assert.True(t, layout.NopIndices[5]) // pop rbp
}

func TestUnifiedStackAnalysisArm64(t *testing.T) {
	testFn := Function{
		Lines: []Line{
			{Assembly: "stp	x29, x30, [sp, #-16]!", Binary: wordToLineBinary(0xa9bf7bfd)},
			{Assembly: "mov	x29, sp", Binary: wordToLineBinary(0x910003fd)},
			{Assembly: "mov	sp, x29", Binary: wordToLineBinary(0x910003bf)},
			{Assembly: "ldp	x29, x30, [sp], #16", Binary: wordToLineBinary(0xa8c17bfd)},
			{Assembly: "ret", Disassembled: "RET", Binary: wordToLineBinary(0xd65f03c0)},
		},
	}

	archInfo := newArm64StackInfo()
	layout := analyzeStackLayout(archInfo, testFn.Lines)

	assert.True(t, layout.FramePointerUsed)
	assert.True(t, layout.NopIndices[0]) // stp x29, x30
	assert.True(t, layout.NopIndices[1]) // mov x29, sp
	assert.True(t, layout.NopIndices[2]) // mov sp, x29
	assert.True(t, layout.NopIndices[3]) // ldp x29, x30
}

func TestAlignedToUnalignedConversion(t *testing.T) {
	archInfo := newAmd64StackInfo()

	tests := []struct {
		input    string
		expected string
	}{
		{"MOVAPS", "MOVUPS"},
		{"MOVAPD", "MOVUPD"},
		{"MOVDQA", "MOVDQU"},
		{"VMOVAPS", "VMOVUPS"},
		{"VMOVAPD", "VMOVUPD"},
		{"VMOVDQA", "VMOVDQU"},
		{"VMOVDQA32", "VMOVDQU32"},
		{"VMOVDQA64", "VMOVDQU64"},
		// lowercase variants
		{"movaps", "movups"},
		{"vmovdqa", "vmovdqu"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := archInfo.ToUnalignedInsn(tc.input)
			require.NotNil(t, result)
			assert.Equal(t, tc.expected, *result)
		})
	}

	// Test that non-aligned instructions return nil
	result := archInfo.ToUnalignedInsn("MOVQ")
	assert.Nil(t, result)

	result = archInfo.ToUnalignedInsn("VMOVUPS")
	assert.Nil(t, result)
}

func TestStackLayoutOffsetTranslation(t *testing.T) {
	layout := &StackLayout{
		GoFrameSize: 48,
	}

	// Test translation: C offset 0 should map to stack-48(SP)
	assert.Equal(t, 48, layout.TranslateOffset(0))

	// C offset 8 should map to stack-40(SP)
	assert.Equal(t, 40, layout.TranslateOffset(8))

	// C offset 40 should map to stack-8(SP)
	assert.Equal(t, 8, layout.TranslateOffset(40))
}

func TestStackLayoutFormatStackRef(t *testing.T) {
	layout := &StackLayout{
		GoFrameSize: 48,
	}

	assert.Equal(t, "stack-48(SP)", layout.FormatStackRef(0, ""))
	assert.Equal(t, "local-40(SP)", layout.FormatStackRef(8, "local"))
	assert.Equal(t, "spill-8(SP)", layout.FormatStackRef(40, "spill"))
}

func TestUnifiedStackTransformSimpleAmd64(t *testing.T) {
	testFn := Function{
		Lines: []Line{
			{Assembly: "push\trbp", Disassembled: "PUSHQ BP", Binary: binaryFromHex("55")},
			{Assembly: "mov\trbp, rsp", Disassembled: "MOVQ SP, BP", Binary: binaryFromHex("48 89 e5")},
			{Assembly: "and\trsp, -8", Disassembled: "ANDQ $-0x8, SP", Binary: binaryFromHex("48 83 e4 f8")},
			{Assembly: "mov\teax, 42", Disassembled: "MOVL $0x2a, AX"},
			{Assembly: "pop\trbp", Disassembled: "POPQ BP", Binary: binaryFromHex("5d")},
			{Assembly: "ret", Disassembled: "RET"},
		},
	}

	modified := checkStackUnified(config.AMD64(), testFn)

	require.Len(t, modified.Lines, 6)
	assert.Equal(t, "NOP", modified.Lines[0].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[1].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[2].Disassembled)
	assert.Equal(t, "MOVL $0x2a, AX", modified.Lines[3].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[4].Disassembled)
	assert.Equal(t, "RET", modified.Lines[5].Disassembled)
}

func TestArchStackInfoIsCalleeSaved(t *testing.T) {
	amd64 := newAmd64StackInfo()
	arm64 := newArm64StackInfo()

	// AMD64 callee-saved registers
	assert.True(t, amd64.IsCalleeSaved("rbp"))
	assert.True(t, amd64.IsCalleeSaved("rbx"))
	assert.True(t, amd64.IsCalleeSaved("r12"))
	assert.True(t, amd64.IsCalleeSaved("r13"))
	assert.True(t, amd64.IsCalleeSaved("r14"))
	assert.True(t, amd64.IsCalleeSaved("r15"))
	assert.False(t, amd64.IsCalleeSaved("rax"))
	assert.False(t, amd64.IsCalleeSaved("rcx"))

	// ARM64 callee-saved registers
	assert.True(t, arm64.IsCalleeSaved("x19"))
	assert.True(t, arm64.IsCalleeSaved("x20"))
	assert.True(t, arm64.IsCalleeSaved("x29"))
	assert.True(t, arm64.IsCalleeSaved("x30"))
	assert.False(t, arm64.IsCalleeSaved("x0"))
	assert.False(t, arm64.IsCalleeSaved("x10"))
}
