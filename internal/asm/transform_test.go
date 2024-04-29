package asm

import (
	"testing"

	"github.com/kelindar/gocc/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestDataLoadRewrite(t *testing.T) {
	fn := Function{
		Lines: []Line{
			{Assembly: "adrp\tx10, lengthTable", Disassembled: "ADRP 0(PC), R10"},
			{Assembly: "add	x10, x10, :lo12:shuffleTable", Disassembled: "ADD $0, R10, R10"},
		},
	}

	fn = rewriteJumpsAndLoads(config.ARM64(), fn)

	assert.Equal(t, "MOVD $lengthTable<>(SB), R10", fn.Lines[0].Disassembled)
	assert.Empty(t, fn.Lines[0].Binary)
	assert.Equal(t, "ADD $0, R10, R10", fn.Lines[1].Disassembled)
	assert.Empty(t, fn.Lines[1].Binary)
}
