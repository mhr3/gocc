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
			{Assembly: "pushq\trbp", Disassembled: "PUSHQ BP"},
			{Assembly: "mov\trbp, rsp", Disassembled: "MOVQ SP, BP"},
			{Assembly: "push\trbx", Disassembled: "PUSHQ BX"},
			{Assembly: "and\trsp, -8", Disassembled: "ANDQ $-0x8, SP"},
			{Assembly: "mov\teax, 16", Disassembled: "MOVL $0x10, AX"},
			{Assembly: "lea\trsp, [rbp - 8]", Disassembled: "LEAQ -0x8(BP), SP"},
			{Assembly: "pop\trbx", Disassembled: "POPQ BX"},
			{Assembly: "pop\trbp", Disassembled: "POPQ BP"},
			{Assembly: "ret", Disassembled: "RET"},
		},
	}

	modified := checkStackAmd64(config.AMD64(), testFn)

	require.Equal(t, 8, modified.LocalsSize)

	require.Len(t, modified.Lines, 9)
	assert.Equal(t, "NOP", modified.Lines[0].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[1].Disassembled)
	assert.True(t, strings.HasPrefix(modified.Lines[2].Disassembled, "MOV"))
	assert.Equal(t, testFn.Lines[4].Disassembled, modified.Lines[4].Disassembled)
	assert.Equal(t, "NOP", modified.Lines[5].Disassembled)
	assert.True(t, strings.HasPrefix(modified.Lines[6].Disassembled, "MOV"))
	assert.Equal(t, "NOP", modified.Lines[7].Disassembled)
	assert.Equal(t, "RET", modified.Lines[8].Disassembled)
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
	assert.Equal(t, "NOP", modified.Lines[4].Disassembled)
	assert.True(t, strings.HasPrefix(modified.Lines[2].Disassembled, "STP"))
	assert.True(t, strings.HasPrefix(modified.Lines[5].Disassembled, "STP"))
	assert.True(t, strings.HasPrefix(modified.Lines[6].Disassembled, "STP"))
	assert.True(t, strings.HasPrefix(modified.Lines[12].Disassembled, "LDP"))
	assert.Equal(t, "NOP", modified.Lines[13].Disassembled)
	assert.Equal(t, "RET", modified.Lines[14].Disassembled)
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
