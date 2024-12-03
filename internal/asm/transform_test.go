package asm

import (
	"strings"
	"testing"

	"github.com/mhr3/gocc/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var binaryIsSet = []string{}

func TestDataLoadRewrite(t *testing.T) {
	testCases := []struct {
		Name          string
		Cfg           *config.Arch
		Func          Function
		ExpectedLines []Line
	}{
		{
			Name: "arm64-adrp",
			Cfg:  config.ARM64(),
			Func: Function{
				Lines: []Line{
					{Assembly: "adrp\tx10, lengthTable", Disassembled: "ADRP 0(PC), R10"},
					{Assembly: "add	x10, x10, :lo12:shuffleTable", Disassembled: "ADD $0, R10, R10"},
				},
			},
			ExpectedLines: []Line{
				{Disassembled: "MOVD $lengthTable<>(SB), R10", Binary: nil},
				{Disassembled: "ADD $0, R10, R10", Binary: nil},
			},
		},
		{
			Name: "amd64-movo",
			Cfg:  config.AMD64(),
			Func: Function{
				Lines: []Line{
					{Assembly: "movdqa\txmm1, xmmword ptr [rip + .LCPI0_0]", Disassembled: "MOVDQA 0(IP), X1"},
				},
			},
			ExpectedLines: []Line{
				{Disassembled: "MOVO LCPI0_0<>(SB), X1", Binary: nil},
			},
		},
		{
			Name: "amd64-lea",
			Cfg:  config.AMD64(),
			Func: Function{
				Lines: []Line{
					{Assembly: "lea\tr9, [rip + shuf_lut]", Disassembled: "LEAQ 0(IP), R9"},
				},
			},
			ExpectedLines: []Line{
				{Disassembled: "LEAQ shuf_lut<>(SB), R9", Binary: nil},
			},
		},
		{
			Name: "amd64-lea-rbx",
			Cfg:  config.AMD64(),
			Func: Function{
				Lines: []Line{
					{Assembly: "lea\trbx, [rip + lengthTable_1234]", Disassembled: "LEAQ 0(IP), BX"},
				},
			},
			ExpectedLines: []Line{
				{Disassembled: "LEAQ lengthTable_1234<>(SB), BX", Binary: nil},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			fn := rewriteJumpsAndLoads(tc.Cfg, tc.Func)

			require.Len(t, fn.Lines, len(tc.ExpectedLines))
			for i, l := range tc.ExpectedLines {
				assert.Equal(t, l.Disassembled, fn.Lines[i].Disassembled)
				if l.Binary == nil {
					assert.Empty(t, fn.Lines[i].Binary)
				} else {
					assert.NotEmpty(t, fn.Lines[i].Binary)
				}
			}
		})
	}
}

func TestTransformInstructions(t *testing.T) {
	testCases := []struct {
		Name          string
		Cfg           *config.Arch
		Func          Function
		ExpectedLines []Line
	}{
		{
			Name: "amd64-cmpl-cl",
			Cfg:  config.AMD64(),
			Func: Function{
				Lines: []Line{
					{Assembly: "cmp	cl, 8", Disassembled: "CMPL CL, $0x8", Binary: binaryFromHex("80f908")},
				},
			},
			ExpectedLines: []Line{
				{Disassembled: "CMPL CL, $0x8", Binary: binaryIsSet}, // doesn't compile, must use binary
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			fn := removeBinaryInstructions(tc.Cfg, tc.Func)

			require.Len(t, fn.Lines, len(tc.ExpectedLines))
			for i, l := range tc.ExpectedLines {
				assert.Equal(t, l.Disassembled, fn.Lines[i].Disassembled)
				if l.Binary == nil {
					assert.Empty(t, fn.Lines[i].Binary)
				} else {
					assert.NotEmpty(t, fn.Lines[i].Binary)
				}
			}
		})
	}
}

func binaryFromHex(bin string) []string {
	bin = strings.ReplaceAll(bin, " ", "")
	// split every two bytes
	var res []string
	for i := 0; i < len(bin); i += 2 {
		res = append(res, bin[i:i+2])
	}
	return res
}
