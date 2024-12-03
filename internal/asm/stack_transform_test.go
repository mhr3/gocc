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
			{Assembly: "lea\trsp, [rbp - 8]", Disassembled: "LEAQ -0x8(BP), SP"},
			{Assembly: "pop\trbx", Disassembled: "POPQ BX", Binary: binaryFromHex("5b")},
			{Assembly: "pop\trbp", Disassembled: "POPQ BP", Binary: binaryFromHex("5d")},
			{Assembly: "ret", Disassembled: "RET"},
		},
	}

	modified := checkStackAmd64(config.AMD64(), testFn)

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

	modified := checkStackAmd64(config.AMD64(), testFn)

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

	modified := checkStackAmd64(config.AMD64(), testFn)

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

	modified := checkStackArm64(config.ARM64(), testFn)

	require.Equal(t, 96, modified.LocalsSize)

	require.Len(t, modified.Lines, 15)
	assert.Equal(t, "NOP", modified.Lines[0].Disassembled)
	assert.True(t, strings.HasPrefix(modified.Lines[1].Disassembled, "MOV"))
	assert.True(t, strings.HasPrefix(modified.Lines[2].Disassembled, "STP"))
	assert.Equal(t, "NOP", modified.Lines[4].Disassembled)
	assert.True(t, strings.HasPrefix(modified.Lines[5].Disassembled, "STP"))
	assert.True(t, strings.HasPrefix(modified.Lines[6].Disassembled, "STP"))
	assert.Equal(t, "NOP", modified.Lines[7].Disassembled)
	assert.True(t, strings.HasPrefix(modified.Lines[12].Disassembled, "LDP"))
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

	modified := checkStackArm64(config.ARM64(), testFn)

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
