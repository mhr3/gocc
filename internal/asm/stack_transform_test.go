package asm

import (
	"strings"
	"testing"

	"github.com/kelindar/gocc/internal/config"
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
