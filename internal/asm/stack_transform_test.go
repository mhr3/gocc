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

func TestAmd64RetainedPushUsesCompactedFrame(t *testing.T) {
	testFn := Function{
		Lines: []Line{
			{Assembly: "push\trbp", Disassembled: "PUSHQ BP", Binary: binaryFromHex("55")},
			{Assembly: "push\tr15", Disassembled: "PUSHQ R15", Binary: binaryFromHex("41 57")},
			{Assembly: "push\tr14", Disassembled: "PUSHQ R14", Binary: binaryFromHex("41 56")},
			{Assembly: "push\tr13", Disassembled: "PUSHQ R13", Binary: binaryFromHex("41 55")},
			{Assembly: "push\tr12", Disassembled: "PUSHQ R12", Binary: binaryFromHex("41 54")},
			{Assembly: "push\trbx", Disassembled: "PUSHQ BX", Binary: binaryFromHex("53")},
			{Assembly: "push\trax", Disassembled: "PUSHQ AX", Binary: binaryFromHex("50")},
		},
	}

	modified := checkStackUnified(config.AMD64(), testFn)

	require.Equal(t, 8, modified.LocalsSize)
	for i := 0; i < 6; i++ {
		assert.Equal(t, "NOP", modified.Lines[i].Disassembled)
	}
	assert.Equal(t, "MOVQ AX, 0(SP)", modified.Lines[6].Disassembled)
}

func TestAmd64InternalPushUsesPreservedFrame(t *testing.T) {
	testFn := Function{
		Internal: true,
		Lines: []Line{
			{Assembly: "push\trbp", Disassembled: "PUSHQ BP", Binary: binaryFromHex("55")},
			{Assembly: "push\tr15", Disassembled: "PUSHQ R15", Binary: binaryFromHex("41 57")},
			{Assembly: "push\tr14", Disassembled: "PUSHQ R14", Binary: binaryFromHex("41 56")},
			{Assembly: "push\tr13", Disassembled: "PUSHQ R13", Binary: binaryFromHex("41 55")},
			{Assembly: "push\tr12", Disassembled: "PUSHQ R12", Binary: binaryFromHex("41 54")},
			{Assembly: "push\trbx", Disassembled: "PUSHQ BX", Binary: binaryFromHex("53")},
			{Assembly: "push\trax", Disassembled: "PUSHQ AX", Binary: binaryFromHex("50")},
		},
	}

	modified := checkStackManipulation(config.AMD64(), testFn)

	assert.Equal(t, 0, modified.LocalsSize)
	require.Equal(t, 56, modified.HiddenStackSize)
	assert.Equal(t, "MOVQ BP, 48(SP)", modified.Lines[0].Disassembled)
	assert.Equal(t, "MOVQ BX, 8(SP)", modified.Lines[5].Disassembled)
	assert.Equal(t, "MOVQ AX, 0(SP)", modified.Lines[6].Disassembled)
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

/*

     ef8: a9bb7bfd      stp     x29, x30, [sp, #-0x50]!
     efc: a90167fa      stp     x26, x25, [sp, #0x10]
     f00: 910003fd      mov     x29, sp
     f10: d10043ff      sub     sp, sp, #0x10
    1074: 54fffec3      b.lo    0x104c <index_fold+0x154>
    1078: b4fff528      cbz     x8, 0xf1c <index_fold+0x24>
    107c: 910003ea      mov     x10, sp
    1080: 8b08012b      add     x11, x9, x8
    1084: a9007fff      stp     xzr, xzr, [sp]
    1088: aa08014a      orr     x10, x10, x8
    108c: 37184f81      tbnz    w1, #0x3, 0x1a7c <index_fold+0xb84>
    1090: 37105701      tbnz    w1, #0x2, 0x1b70 <index_fold+0xc78>
    1094: 3940012c      ldrb    w12, [x9]
    1098: d341fd0d      lsr     x13, x8, #1
    109c: 910003ee      mov     x14, sp
    10a0: 390003ec      strb    w12, [sp]
    10a4: 386d692c      ldrb    w12, [x9, x13]
    10a8: aa0d01cd      orr     x13, x14, x13
    10ac: 390001ac      strb    w12, [x13]
    10b0: 385ff16b      ldurb   w11, [x11, #-0x1]
    10b4: 381ff14b      sturb   w11, [x10, #-0x1]

		stp     x29, x30, [sp, #-80]!           // 16-byte Folded Spill
        stp     x26, x25, [sp, #16]             // 16-byte Folded Spill
        mov     x29, sp
        sub     sp, sp, #16
	    b.lo    .LBB4_19
.LBB4_21:
        cbz     x8, .LBB4_1
// %bb.22:
        mov     x10, sp
        add     x11, x9, x8
        stp     xzr, xzr, [sp]
        orr     x10, x10, x8
        tbnz    w1, #3, .LBB4_143
// %bb.23:
        tbnz    w1, #2, .LBB4_154
// %bb.24:
        ldrb    w12, [x9]
        lsr     x13, x8, #1
        mov     x14, sp
        strb    w12, [sp]
        ldrb    w12, [x9, x13]
        orr     x13, x14, x13
        strb    w12, [x13]
        ldurb   w11, [x11, #-1]
        sturb   w11, [x10, #-1]
*/

func TestStackOpsArm64(t *testing.T) {
	testFn := Function{
		Lines: []Line{
			{Assembly: "stp\tx29, x30, [sp, #-80]!", Binary: wordToLineBinary(0xa9bb7bfd)},
			{Assembly: "stp\tx26, x25, [sp, #16]", Binary: wordToLineBinary(0xa90167fa)},
			{Assembly: "mov\tx29, sp", Binary: wordToLineBinary(0x910003fd)},
			{Assembly: "sub\tsp, sp, #16", Binary: wordToLineBinary(0xd10043ff)},
			{Assembly: "b.lo\t.LBB4_19", Binary: wordToLineBinary(0x54fffec3)},
			{Assembly: "cbz\tx8, .LBB4_1", Binary: wordToLineBinary(0xb4fff528)},
			{Assembly: "mov\tx10, sp", Binary: wordToLineBinary(0x910003ea)},
			{Assembly: "add\tx11, x9, x8", Binary: wordToLineBinary(0x8b08012b)},
			{Assembly: "stp\txzr, xzr, [sp]", Binary: wordToLineBinary(0xa9007fff)},
			{Assembly: "orr\tx10, x10, x8", Binary: wordToLineBinary(0xaa08014a)},
			{Assembly: "tbnz\tw1, #3, .LBB4_143", Binary: wordToLineBinary(0x37184f81)},
			{Assembly: "tbnz\tw1, #2, .LBB4_154", Binary: wordToLineBinary(0x37105701)},
			{Assembly: "ldrb\tw12, [x9]", Binary: wordToLineBinary(0x3940012c)},
			{Assembly: "lsr\tx13, x8, #1", Binary: wordToLineBinary(0xd341fd0d)},
			{Assembly: "mov\tx14, sp", Binary: wordToLineBinary(0x910003ee)},
			{Assembly: "strb\tw12, [sp]", Binary: wordToLineBinary(0x390003ec)},
			{Assembly: "ldrb\tw12, [x9, x13]", Binary: wordToLineBinary(0x386d692c)},
			{Assembly: "orr\tx13, x14, x13", Binary: wordToLineBinary(0xaa0d01cd)},
			{Assembly: "strb\tw12, [x13]", Binary: wordToLineBinary(0x390001ac)},
			{Assembly: "ldurb\tw11, [x11, #-0x1]", Binary: wordToLineBinary(0x385ff16b)},
			{Assembly: "sturb\tw11, [x10, #-0x1]", Binary: wordToLineBinary(0x381ff14b)},
		},
	}

	modified := checkStackUnified(config.ARM64(), testFn)

	require.Equal(t, 96, modified.LocalsSize)

	require.Len(t, modified.Lines, 21)
	assert.Equal(t, "NOP", modified.Lines[0].Disassembled)
	// This excerpt has no matching restore, so it is not safe to assume that
	// the fixed-offset store is only a C-ABI register save. Its final offset
	// fits STP's scaled immediate, so the pair remains intact.
	assert.Equal(t, "STP (R26, R25), 48(RSP)", modified.Lines[1].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[2].Disassembled)
	assert.Equal(t, "STP (ZR, ZR), 16(RSP)", modified.Lines[8].Disassembled)
	assert.Equal(t, testFn.Lines[15].Binary, modified.Lines[15].Binary)
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
	// Reading SP into a general register is safe; keeping the original
	// instruction also avoids mixing hardware RSP with Go's pseudo-SP.
	assert.Equal(t, testFn.Lines[1].Binary, modified.Lines[1].Binary)
	assert.Equal(t, "NOP", modified.Lines[2].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[4].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[5].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[6].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[7].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[12].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[13].Disassembled)
	assert.Equal(t, "RET", modified.Lines[14].Disassembled)
}

func TestArm64StackDataKeepsFrame(t *testing.T) {
	testFn := Function{
		Lines: []Line{
			{Assembly: "stp\tx29, x30, [sp, #-32]!", Binary: wordToLineBinary(0xa9be7bfd)},
			{Assembly: "mov\tx29, sp", Binary: wordToLineBinary(0x910003fd)},
			{Assembly: "stp\txzr, xzr, [sp]", Binary: wordToLineBinary(0xa9007fff)},
			{Assembly: "stp\tq0, q0, [sp]", Binary: wordToLineBinary(0xad0003e0)},
			{Assembly: "str\tx0, [sp, #16]", Binary: wordToLineBinary(0xf9000be0)},
			{Assembly: "ldr\tx0, [sp, #16]", Binary: wordToLineBinary(0xf9400be0)},
			{Assembly: "ldp\tx29, x30, [sp], #32", Binary: wordToLineBinary(0xa8c27bfd)},
			{Assembly: "ret", Disassembled: "RET", Binary: wordToLineBinary(0xd65f03c0)},
		},
	}

	modified := checkStackUnified(config.ARM64(), testFn)

	require.Equal(t, 32, modified.LocalsSize)
	assert.Equal(t, "NOP", modified.Lines[0].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[1].Disassembled)
	assert.Equal(t, "STP (ZR, ZR), 16(RSP)", modified.Lines[2].Disassembled)
	assert.Equal(t, "FSTPQ (F0, F0), 16(RSP)", modified.Lines[3].Disassembled)
	assert.Equal(t, testFn.Lines[4].Binary, modified.Lines[4].Binary)
	assert.Equal(t, testFn.Lines[5].Binary, modified.Lines[5].Binary)
	assert.Equal(t, "NOP", modified.Lines[6].Disassembled)
}

func TestArm64PreindexedDataStoreUsesFixedGoFrame(t *testing.T) {
	testFn := Function{
		Lines: []Line{
			{Assembly: "stp\txzr, xzr, [sp, #-16]!", Binary: wordToLineBinary(0xa9bf7fff)},
			{Assembly: "add\tsp, sp, #16", Binary: wordToLineBinary(0x910043ff)},
			{Assembly: "ret", Disassembled: "RET", Binary: wordToLineBinary(0xd65f03c0)},
		},
	}

	modified := checkStackUnified(config.ARM64(), testFn)

	require.Equal(t, 16, modified.LocalsSize)
	assert.Empty(t, modified.Lines[0].Binary)
	assert.Equal(t, "STP (ZR, ZR), 16(RSP)", modified.Lines[0].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[1].Disassembled)
}

func TestArm64StackPairKeptWhenFinalOffsetIsEncodable(t *testing.T) {
	tests := []struct {
		name, assembly, disassembled, want string
		bias                               int
	}{
		{name: "32-bit", assembly: "stp\tw8, w9, [sp]", disassembled: "STPW (R8, R9), 248(RSP)", bias: 4, want: "STPW (R8, R9), 252(RSP)"},
		{name: "64-bit", assembly: "stp\tx8, x9, [sp]", disassembled: "STP (R8, R9), 496(RSP)", bias: 8, want: "STP (R8, R9), 504(RSP)"},
		{name: "128-bit", assembly: "stp\tq0, q1, [sp]", disassembled: "FSTPQ (F0, F1), 992(RSP)", bias: 16, want: "FSTPQ (F0, F1), 1008(RSP)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shifted := shiftArm64CStackRef(Line{Assembly: tt.assembly, Disassembled: tt.disassembled}, tt.bias)
			require.Len(t, shifted, 1)
			assert.Equal(t, tt.want, shifted[0].Disassembled)
		})
	}
}

func TestArm64StackPairSplitWhenFinalOffsetIsNotEncodable(t *testing.T) {
	tests := []struct {
		name, assembly, disassembled string
		bias                         int
		want                         []string
	}{
		{name: "32-bit", assembly: "stp\tw8, w9, [sp]", disassembled: "STPW (R8, R9), 248(RSP)", bias: 8, want: []string{"MOVW R8, 256(RSP)", "MOVW R9, 260(RSP)"}},
		{name: "64-bit", assembly: "stp\tx8, x9, [sp]", disassembled: "STP (R8, R9), 496(RSP)", bias: 16, want: []string{"MOVD R8, 512(RSP)", "MOVD R9, 520(RSP)"}},
		{name: "128-bit", assembly: "stp\tq0, q1, [sp]", disassembled: "FSTPQ (F0, F1), 992(RSP)", bias: 32, want: []string{"FMOVQ F0, 1024(RSP)", "FMOVQ F1, 1040(RSP)"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shifted := shiftArm64CStackRef(Line{Labels: []string{"target"}, Assembly: tt.assembly, Disassembled: tt.disassembled}, tt.bias)
			require.Len(t, shifted, 2)
			assert.Equal(t, tt.want[0], shifted[0].Disassembled)
			assert.Equal(t, tt.want[1], shifted[1].Disassembled)
			assert.Equal(t, []string{"target"}, shifted[0].Labels)
			assert.Empty(t, shifted[1].Labels)
			assert.Empty(t, shifted[0].Comment)
			assert.Equal(t, "split continuation of preceding STP", shifted[1].Comment)
			assert.Contains(t, shifted[1].Compile(config.ARM64()), "// split continuation of preceding STP")
		})
	}
}

func TestArm64StackPairLoadUsesSameFinalOffsetRule(t *testing.T) {
	line := Line{
		Assembly:     "ldp\tx8, x9, [sp]",
		Disassembled: "LDP 496(RSP), (R8, R9)",
	}

	kept := shiftArm64CStackRef(line, 8)
	require.Len(t, kept, 1)
	assert.Equal(t, "LDP 504(RSP), (R8, R9)", kept[0].Disassembled)

	split := shiftArm64CStackRef(line, 16)
	require.Len(t, split, 2)
	assert.Equal(t, "MOVD 512(RSP), R8", split[0].Disassembled)
	assert.Equal(t, "MOVD 520(RSP), R9", split[1].Disassembled)
	assert.Equal(t, "split continuation of preceding LDP", split[1].Comment)
}

func TestArm64FramePointerIsRebasedToGoFrame(t *testing.T) {
	testFn := Function{
		Lines: []Line{
			{Assembly: "stp\tx29, x30, [sp, #-32]!", Binary: wordToLineBinary(0xa9be7bfd)},
			{Assembly: "str\tx19, [sp, #16]", Binary: wordToLineBinary(0xf9000bf3)},
			{Assembly: "mov\tx29, sp", Binary: wordToLineBinary(0x910003fd)},
			{Assembly: "sub\tsp, sp, #48", Binary: wordToLineBinary(0xd100c3ff)},
			{Assembly: "add\tx5, x29, #24", Binary: wordToLineBinary(0x910063a5)},
			{Assembly: "add\tsp, sp, #48", Binary: wordToLineBinary(0x9100c3ff)},
			{Assembly: "ldr\tx19, [sp, #16]", Binary: wordToLineBinary(0xf9400bf3)},
			{Assembly: "ldp\tx29, x30, [sp], #32", Binary: wordToLineBinary(0xa8c27bfd)},
			{Assembly: "ret", Disassembled: "RET", Binary: wordToLineBinary(0xd65f03c0)},
		},
	}

	modified := checkStackUnified(config.ARM64(), testFn)

	require.Equal(t, 80, modified.LocalsSize)
	assert.Equal(t, "NOP", modified.Lines[0].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[1].Disassembled)
	assert.Equal(t, "ADD $64, RSP, R29", modified.Lines[2].Disassembled)
	assert.Equal(t, testFn.Lines[4].Binary, modified.Lines[4].Binary)
	assert.Equal(t, "NOP", modified.Lines[6].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[7].Disassembled)
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
	layout := analyzeStackLayout(archInfo, testFn.Lines, false)

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
	layout := analyzeStackLayout(archInfo, testFn.Lines, false)

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

func TestAlignedToUnalignedOnlyForStackMemory(t *testing.T) {
	// Test that aligned instructions are only converted when there's a STACK memory operand
	// Register-to-register and non-stack memory accesses should be kept as-is
	testFn := Function{
		Lines: []Line{
			// Register-to-register: should NOT be converted
			{Disassembled: "VMOVDQA X4, X5"},
			{Disassembled: "MOVAPS X0, X1"},
			// Non-stack memory operand: should NOT be converted
			{Disassembled: "MOVAPS 0(AX), X0"},
			{Disassembled: "VMOVDQA X4, 0(BX)"},
			// Stack memory operand (Go syntax): should be converted
			{Disassembled: "VMOVDQA X4, 16(SP)"},
			{Disassembled: "MOVAPS 0(SP), X0"},
			// Stack memory operand (x86 syntax in Assembly field): check Disassembled
			{Assembly: "vmovdqa xmm4, [rsp+16]", Disassembled: "VMOVDQA X4, 16(SP)"},
		},
	}

	modified := checkStackUnified(config.AMD64(), testFn)

	require.Len(t, modified.Lines, 7)

	// Register-to-register: unchanged
	assert.Equal(t, "VMOVDQA X4, X5", modified.Lines[0].Disassembled)
	assert.Equal(t, "MOVAPS X0, X1", modified.Lines[1].Disassembled)

	// Non-stack memory: unchanged
	assert.Equal(t, "MOVAPS 0(AX), X0", modified.Lines[2].Disassembled)
	assert.Equal(t, "VMOVDQA X4, 0(BX)", modified.Lines[3].Disassembled)

	// Stack memory: converted to unaligned
	assert.Equal(t, "VMOVDQU X4, 16(SP)", modified.Lines[4].Disassembled)
	assert.Equal(t, "MOVUPS 0(SP), X0", modified.Lines[5].Disassembled)
	assert.Equal(t, "VMOVDQU X4, 16(SP)", modified.Lines[6].Disassembled)
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

func TestReserveInternalStackFramesLeavesLeafFunctionsUnchanged(t *testing.T) {
	for _, arch := range []*config.Arch{config.AMD64(), config.ARM64()} {
		t.Run(arch.Name, func(t *testing.T) {
			functions := []Function{
				{Name: "leaf_zero"},
				{Name: "leaf_with_locals", LocalsSize: 32},
				{Name: "another_leaf"},
			}

			modified, err := reserveInternalStackFrames(arch, functions)
			require.NoError(t, err)

			assert.Equal(t, 0, modified[0].LocalsSize)
			assert.Equal(t, 32, modified[1].LocalsSize)
			assert.Equal(t, 0, modified[2].LocalsSize)
		})
	}
}

func TestReserveInternalStackFramesArm64(t *testing.T) {
	functions := []Function{
		{
			Name:       "entry",
			LocalsSize: 64,
			Lines: []Line{
				{Disassembled: "CALL helper_one<>(SB)"},
				{Disassembled: "CALL helper_two<>(SB)"},
			},
		},
		{
			Name:            "helper_one",
			Internal:        true,
			HiddenStackSize: 32,
			Lines: []Line{{
				Disassembled: "MOVD R0, 0(RSP)",
				Binary:       wordToLineBinary(0xf90003e0),
			}},
		},
		{
			Name:            "helper_two",
			Internal:        true,
			HiddenStackSize: 48,
			Lines: []Line{{
				Disassembled: "ADD $8, RSP, R1",
				Binary:       wordToLineBinary(0x910023e1),
			}},
		},
	}

	modified, err := reserveInternalStackFrames(config.ARM64(), functions)
	require.NoError(t, err)

	// 64 bytes of visible locals plus both disjoint helper slots.
	assert.Equal(t, 144, modified[0].LocalsSize)
	// Helper slots begin above the visible locals and ARM64 linkage area.
	assert.Equal(t, "MOVD R0, 80(RSP)", modified[1].Lines[0].Disassembled)
	assert.Equal(t, "ADD $120, RSP, R1", modified[2].Lines[0].Disassembled)
	assert.Empty(t, modified[1].Lines[0].Binary)
	assert.Empty(t, modified[2].Lines[0].Binary)
}

func TestReserveInternalStackFramesAmd64RebasesStackAddress(t *testing.T) {
	functions := []Function{
		{Name: "entry", LocalsSize: 64, Lines: []Line{{Disassembled: "CALL helper<>(SB)"}}},
		{
			Name:            "helper",
			Internal:        true,
			HiddenStackSize: 32,
			Lines: []Line{
				{Assembly: "mov\tr8, rsp", Disassembled: "MOVQ SP, R8", Binary: binaryFromHex("49 89 e0")},
				{Assembly: "mov\tqword ptr [rsp + 8], rax", Disassembled: "MOVQ AX, 8(SP)", Binary: binaryFromHex("48 89 44 24 08")},
			},
		},
	}

	modified, err := reserveInternalStackFrames(config.AMD64(), functions)
	require.NoError(t, err)

	// A depth guard separates each flattened helper slot from CALL return
	// addresses. Both stack memory and pointers to a C local must receive the
	// same slot bias.
	assert.Equal(t, 120, modified[0].LocalsSize)
	assert.Equal(t, "LEAQ 72(SP), R8", modified[1].Lines[0].Disassembled)
	assert.Equal(t, "MOVQ AX, 80(SP)", modified[1].Lines[1].Disassembled)
	assert.Empty(t, modified[1].Lines[0].Binary)
	assert.Empty(t, modified[1].Lines[1].Binary)
}

func TestApplyTransformsRebasesAmd64NonCalleeSavedPush(t *testing.T) {
	functions := []Function{
		{Name: "entry", Lines: []Line{{Disassembled: "CALL helper<>(SB)"}}},
		{
			Name:     "helper",
			Internal: true,
			Lines: []Line{
				{Assembly: "push\tr14", Disassembled: "PUSHQ R14", Binary: binaryFromHex("41 56")},
				{Assembly: "push\trbx", Disassembled: "PUSHQ BX", Binary: binaryFromHex("53")},
				{Assembly: "push\trax", Disassembled: "PUSHQ AX", Binary: binaryFromHex("50")},
				{Disassembled: "CALL leaf<>(SB)"},
				{Assembly: "add\trsp, 8", Disassembled: "ADDQ $0x8, SP", Binary: binaryFromHex("48 83 c4 08")},
				{Assembly: "pop\trbx", Disassembled: "POPQ BX", Binary: binaryFromHex("5b")},
				{Assembly: "pop\tr14", Disassembled: "POPQ R14", Binary: binaryFromHex("41 5e")},
				{Assembly: "ret", Disassembled: "RET", Binary: binaryFromHex("c3")},
			},
		},
		{Name: "leaf", Internal: true, Lines: []Line{{Assembly: "ret", Disassembled: "RET", Binary: binaryFromHex("c3")}}},
	}

	modified, err := ApplyTransforms(config.AMD64(), functions)
	require.NoError(t, err)

	assert.Equal(t, 64, modified[0].LocalsSize)
	assert.Equal(t, 0, modified[1].LocalsSize)
	assert.Equal(t, 24, modified[1].HiddenStackSize)
	assert.Equal(t, "MOVQ R14, 32(SP)", modified[1].Lines[0].Disassembled)
	assert.Equal(t, "MOVQ BX, 24(SP)", modified[1].Lines[1].Disassembled)
	assert.Equal(t, "MOVQ AX, 16(SP)", modified[1].Lines[2].Disassembled)
	assert.NotContains(t, modified[1].Lines[2].Disassembled, "stack")
}

func TestApplyTransformsReservesAmd64CalleeSavedOnlyHelper(t *testing.T) {
	functions := []Function{
		{Name: "entry", Lines: []Line{{Disassembled: "CALL helper<>(SB)"}}},
		{
			Name:     "helper",
			Internal: true,
			Lines: []Line{
				{Assembly: "push\tr14", Disassembled: "PUSHQ R14", Binary: binaryFromHex("41 56")},
				{Assembly: "push\trbx", Disassembled: "PUSHQ BX", Binary: binaryFromHex("53")},
				{Assembly: "pop\trbx", Disassembled: "POPQ BX", Binary: binaryFromHex("5b")},
				{Assembly: "pop\tr14", Disassembled: "POPQ R14", Binary: binaryFromHex("41 5e")},
				{Assembly: "ret", Disassembled: "RET", Binary: binaryFromHex("c3")},
			},
		},
	}

	modified, err := ApplyTransforms(config.AMD64(), functions)
	require.NoError(t, err)

	assert.Equal(t, 40, modified[0].LocalsSize)
	assert.Equal(t, 16, modified[1].HiddenStackSize)
	assert.Equal(t, "MOVQ R14, 16(SP)", modified[1].Lines[0].Disassembled)
	assert.Equal(t, "MOVQ BX, 8(SP)", modified[1].Lines[1].Disassembled)
	assert.Equal(t, "MOVQ 8(SP), BX", modified[1].Lines[2].Disassembled)
	assert.Equal(t, "MOVQ 16(SP), R14", modified[1].Lines[3].Disassembled)
}

func TestApplyTransformsSupportsSharedInternalHelperAtDifferentDepths(t *testing.T) {
	functions := []Function{
		{
			Name: "A",
			Lines: []Line{
				{Disassembled: "CALL F<>(SB)"},
				{Disassembled: "CALL B<>(SB)"},
			},
		},
		{
			Name:     "B",
			Internal: true,
			Lines: []Line{
				{Assembly: "push\trax", Disassembled: "PUSHQ AX", Binary: binaryFromHex("50")},
				{Disassembled: "CALL F<>(SB)"},
				{Assembly: "pop\trax", Disassembled: "POPQ AX", Binary: binaryFromHex("58")},
				{Assembly: "ret", Disassembled: "RET", Binary: binaryFromHex("c3")},
			},
		},
		{
			Name:     "F",
			Internal: true,
			Lines: []Line{
				{Assembly: "push\tr14", Disassembled: "PUSHQ R14", Binary: binaryFromHex("41 56")},
				{Assembly: "push\trbx", Disassembled: "PUSHQ BX", Binary: binaryFromHex("53")},
				{Assembly: "pop\trbx", Disassembled: "POPQ BX", Binary: binaryFromHex("5b")},
				{Assembly: "pop\tr14", Disassembled: "POPQ R14", Binary: binaryFromHex("41 5e")},
				{Assembly: "ret", Disassembled: "RET", Binary: binaryFromHex("c3")},
			},
		},
	}

	modified, err := ApplyTransforms(config.AMD64(), functions)
	require.NoError(t, err)

	// Two internal helpers give every arena 16 bytes of downward call-depth
	// slack. B is based at 16(SP), while F is based at 48(SP).
	assert.Equal(t, 80, modified[0].LocalsSize)
	assert.Equal(t, 8, modified[1].HiddenStackSize)
	assert.Equal(t, "MOVQ AX, 16(SP)", modified[1].Lines[0].Disassembled)
	assert.Equal(t, 16, modified[2].HiddenStackSize)
	assert.Equal(t, "MOVQ R14, 56(SP)", modified[2].Lines[0].Disassembled)
	assert.Equal(t, "MOVQ BX, 48(SP)", modified[2].Lines[1].Disassembled)

	// Model addresses relative to A's hardware SP. A direct CALL places F's
	// 16-byte frame at [40, 56); through B, the extra return address slides it
	// down to [32, 48). B's active frame is [8, 16), and return addresses are
	// below zero, so neither F invocation can overlap live caller state.
	const (
		callSlot    = 8
		bBias       = 16
		bSize       = 8
		fBias       = 48
		fSize       = 16
		directDepth = 1
		nestedDepth = 2
	)
	bStart := bBias - callSlot
	fDirectStart := fBias - directDepth*callSlot
	fNestedStart := fBias - nestedDepth*callSlot
	assert.Equal(t, 40, fDirectStart)
	assert.Equal(t, 32, fNestedStart)
	assert.LessOrEqual(t, bStart+bSize, fNestedStart)
	assert.LessOrEqual(t, fDirectStart+fSize, modified[0].LocalsSize)
	assert.LessOrEqual(t, fNestedStart+fSize, modified[0].LocalsSize)
}

func TestApplyTransformsRejectsAmd64InternalStackArguments(t *testing.T) {
	functions := []Function{
		{Name: "entry", Lines: []Line{{Disassembled: "CALL helper<>(SB)"}}},
		{
			Name:     "helper",
			Internal: true,
			Lines: []Line{
				{Assembly: "sub\trsp, 32", Disassembled: "SUBQ $0x20, SP", Binary: binaryFromHex("48 83 ec 20")},
				// At function entry the return address occupies [rsp], so after
				// allocating 32 bytes the first stack argument is [rsp + 40].
				{Assembly: "mov\trax, qword ptr [rsp + 40]", Disassembled: "MOVQ 40(SP), AX", Binary: binaryFromHex("48 8b 44 24 28")},
				{Assembly: "add\trsp, 32", Disassembled: "ADDQ $0x20, SP", Binary: binaryFromHex("48 83 c4 20")},
				{Assembly: "ret", Disassembled: "RET", Binary: binaryFromHex("c3")},
			},
		},
	}

	_, err := ApplyTransforms(config.AMD64(), functions)
	require.EqualError(t, err, `internal helper "helper" uses stack-passed C arguments, which are unsupported`)
}

func TestApplyTransformsRejectsArm64InternalStackArguments(t *testing.T) {
	functions := []Function{
		{Name: "entry", Lines: []Line{{Disassembled: "CALL helper<>(SB)"}}},
		{
			Name:     "helper",
			Internal: true,
			Lines: []Line{
				{Assembly: "stp\tx29, x30, [sp, #-32]!", Binary: wordToLineBinary(0xa9be7bfd)},
				// AArch64 keeps the return address in X30. After allocating 32
				// bytes, the first stack argument is therefore at [sp + 32].
				{Assembly: "ldr\tx0, [sp, #32]", Disassembled: "MOVD 32(RSP), R0", Binary: wordToLineBinary(0xf94013e0)},
				{Assembly: "ldp\tx29, x30, [sp], #32", Binary: wordToLineBinary(0xa8c27bfd)},
				{Assembly: "ret", Disassembled: "RET", Binary: wordToLineBinary(0xd65f03c0)},
			},
		},
	}

	_, err := ApplyTransforms(config.ARM64(), functions)
	require.EqualError(t, err, `internal helper "helper" uses stack-passed C arguments, which are unsupported`)
}

func TestAssignInternalFunctionOwnersAcceptsDisjointGraphs(t *testing.T) {
	functions := []Function{
		{Name: "FN1", Lines: []Line{{Disassembled: "CALL A<>(SB)"}}},
		{Name: "A", Internal: true, Lines: []Line{{Disassembled: "CALL D<>(SB)"}}},
		{Name: "D", Internal: true, Lines: []Line{{Disassembled: "CALL C<>(SB)"}}},
		{Name: "C", Internal: true},
		{Name: "FN2", Lines: []Line{{Disassembled: "CALL B<>(SB)"}}},
		{Name: "B", Internal: true, Lines: []Line{{Disassembled: "CALL E<>(SB)"}}},
		{Name: "E", Internal: true},
	}

	owners, err := assignInternalFunctionOwners(functions)
	require.NoError(t, err)
	assert.Equal(t, []int{-1, 0, 0, 0, -1, 4, 4}, owners)
}

func TestReserveInternalStackFramesUsesPerRootArenas(t *testing.T) {
	functions := []Function{
		{Name: "FN1", LocalsSize: 16, Lines: []Line{{Disassembled: "CALL A<>(SB)"}}},
		{
			Name:            "A",
			Internal:        true,
			HiddenStackSize: 32,
			Lines:           []Line{{Disassembled: "MOVQ AX, 0(SP)"}},
		},
		{Name: "FN2", Lines: []Line{{Disassembled: "CALL B<>(SB)"}}},
		{
			Name:            "B",
			Internal:        true,
			HiddenStackSize: 64,
			Lines:           []Line{{Disassembled: "MOVQ BX, 0(SP)"}},
		},
	}

	modified, err := reserveInternalStackFrames(config.AMD64(), functions)
	require.NoError(t, err)
	assert.Equal(t, 72, modified[0].LocalsSize)
	assert.Equal(t, "MOVQ AX, 24(SP)", modified[1].Lines[0].Disassembled)
	assert.Equal(t, 88, modified[2].LocalsSize)
	assert.Equal(t, "MOVQ BX, 8(SP)", modified[3].Lines[0].Disassembled)
}

func TestAssignInternalFunctionOwnersRejectsSharedHelper(t *testing.T) {
	functions := []Function{
		{Name: "FN1", Lines: []Line{{Disassembled: "CALL A<>(SB)"}, {Disassembled: "CALL B<>(SB)"}}},
		{Name: "A", Internal: true, Lines: []Line{{Disassembled: "CALL D<>(SB)"}}},
		{Name: "D", Internal: true, Lines: []Line{{Disassembled: "CALL C<>(SB)"}}},
		{Name: "C", Internal: true},
		{Name: "FN2", Lines: []Line{{Disassembled: "CALL B<>(SB)"}}},
		{Name: "B", Internal: true, Lines: []Line{{Disassembled: "CALL E<>(SB)"}}},
		{Name: "E", Internal: true},
	}

	_, err := assignInternalFunctionOwners(functions)
	require.EqualError(t, err,
		`internal helper "B" is reachable from multiple exported functions: FN1 -> B; FN2 -> B`)
}

func TestApplyTransformsRejectsSharedInternalHelper(t *testing.T) {
	functions := []Function{
		{Name: "FN1", Lines: []Line{{Disassembled: "CALL shared<>(SB)"}}},
		{Name: "FN2", Lines: []Line{{Disassembled: "CALL shared<>(SB)"}}},
		{Name: "shared", Internal: true},
	}

	_, err := ApplyTransforms(config.AMD64(), functions)
	require.EqualError(t, err,
		`internal helper "shared" is reachable from multiple exported functions: FN1 -> shared; FN2 -> shared`)
}

func TestAssignInternalFunctionOwnersRejectsExportedCallee(t *testing.T) {
	functions := []Function{
		{Name: "FN1", Lines: []Line{{Disassembled: "CALL FN2<>(SB)"}}},
		{Name: "FN2"},
	}

	_, err := assignInternalFunctionOwners(functions)
	require.EqualError(t, err, `C-ABI call from "FN1" to exported function "FN2" is unsupported`)
}

func TestApplyTransformsFlattensInternalArm64Frame(t *testing.T) {
	functions := []Function{
		{Name: "entry", Lines: []Line{{Disassembled: "CALL helper<>(SB)"}}},
		{
			Name:     "helper",
			Internal: true,
			Lines: []Line{
				{Assembly: "stp\tx29, x30, [sp, #-32]!", Binary: wordToLineBinary(0xa9be7bfd)},
				{Assembly: "stp\tx8, x8, [sp]", Disassembled: "STP (R8, R8), (RSP)", Binary: wordToLineBinary(0xa90023e8)},
				{Assembly: "str\tx0, [sp, #16]", Disassembled: "MOVD R0, 16(RSP)", Binary: wordToLineBinary(0xf9000be0)},
				{Assembly: "ldr\tx0, [sp, #16]", Disassembled: "MOVD 16(RSP), R0", Binary: wordToLineBinary(0xf9400be0)},
				{Assembly: "ldp\tx29, x30, [sp], #32", Binary: wordToLineBinary(0xa8c27bfd)},
				{Assembly: "ret", Disassembled: "RET", Binary: wordToLineBinary(0xd65f03c0)},
			},
		},
	}

	modified, err := ApplyTransforms(config.ARM64(), functions)
	require.NoError(t, err)

	assert.Equal(t, 32, modified[0].LocalsSize)
	assert.Equal(t, 0, modified[1].LocalsSize)
	assert.Equal(t, 32, modified[1].HiddenStackSize)
	assert.Equal(t, "STP (R8, R8), 16(RSP)", modified[1].Lines[1].Disassembled)
	for _, line := range modified[1].Lines {
		assert.NotContains(t, line.Disassembled, "RSP, RSP")
		if strings.Contains(line.Disassembled, "(RSP)") {
			assert.Empty(t, line.Binary)
		}
	}
}

func TestApplyTransformsRejectsDirectInternalRecursion(t *testing.T) {
	for _, arch := range []*config.Arch{config.ARM64(), config.AMD64()} {
		t.Run(arch.Name, func(t *testing.T) {
			functions := []Function{{
				Name:     "recursive_helper",
				Internal: true,
				Lines: []Line{{
					Disassembled: "CALL recursive_helper<>(SB)",
				}},
			}}

			_, err := ApplyTransforms(arch, functions)
			require.EqualError(t, err, "recursive internal helper call graph: recursive_helper -> recursive_helper")
		})
	}
}

func TestApplyTransformsRejectsMutualInternalRecursion(t *testing.T) {
	functions := []Function{
		{
			Name:     "helper_a",
			Internal: true,
			Lines: []Line{{
				Disassembled: "CALL helper_b<>(SB)",
			}},
		},
		{
			Name:     "helper_b",
			Internal: true,
			Lines: []Line{{
				Disassembled: "CALL helper_a<>(SB)",
			}},
		},
	}

	for _, arch := range []*config.Arch{config.ARM64(), config.AMD64()} {
		t.Run(arch.Name, func(t *testing.T) {
			_, err := ApplyTransforms(arch, functions)
			require.EqualError(t, err, "recursive internal helper call graph: helper_a -> helper_b -> helper_a")
		})
	}
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
