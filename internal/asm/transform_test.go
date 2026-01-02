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
					{Assembly: "adrp\tx9, lengthTable", Disassembled: "ADRP 0(PC), R9"},
					{Assembly: "add	x9, x9, :lo12:shuffleTable", Disassembled: "ADD $0, R9, R9", Binary: binaryFromHex("29 01 00 91")},
				},
			},
			ExpectedLines: []Line{
				{Disassembled: "MOVD $lengthTable<>(SB), R9", Binary: nil},
				{Disassembled: "NOP", Binary: nil},
			},
		},
		{
			Name: "arm64-jump-misdecoded",
			Cfg:  config.ARM64(),
			Func: Function{
				Lines: []Line{
					{Assembly: "b\t.LBB2_14", Disassembled: "JMP encodingShuffleTable_0124(SB)"},
				},
			},
			ExpectedLines: []Line{
				{Disassembled: "JMP LBB2_14", Binary: nil},
			},
		},
		{
			Name: "arm64-jump-misdecoded-2",
			Cfg:  config.ARM64(),
			Func: Function{
				Lines: []Line{
					{Assembly: "b.eq\t.LBB7_10", Disassembled: "BEQ shuffleTable_1234(SB)"},
				},
			},
			ExpectedLines: []Line{
				{Disassembled: "BEQ LBB7_10", Binary: nil},
			},
		},
		{
			Name: "arm64-jump-cbz",
			Cfg:  config.ARM64(),
			Func: Function{
				Lines: []Line{
					{Assembly: "cbz\tx8, .LBB2_23", Disassembled: "CBZ R8, 38(PC)"},
				},
			},
			ExpectedLines: []Line{
				{Disassembled: "CBZ R8, LBB2_23", Binary: nil},
			},
		},
		{
			Name: "arm64-jump-tbz",
			Cfg:  config.ARM64(),
			Func: Function{
				Lines: []Line{
					{Assembly: "tbz\tw12, #0, .LBB1_19", Disassembled: "TBZ $0, R12, 67(PC)"},
				},
			},
			ExpectedLines: []Line{
				{Disassembled: "TBZ $0, R12, LBB1_19", Binary: nil},
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
		{
			Name: "amd64-vpand-2registers",
			Cfg:  config.AMD64(),
			Func: Function{
				Lines: []Line{
					{Assembly: "vpand\tymm3, ymm7, ymmword ptr [rip + .LCPI0_0]", Disassembled: "VPAND 0(IP), Y7, Y3"},
				},
			},
			ExpectedLines: []Line{
				{Disassembled: "VPAND LCPI0_0<>(SB), Y7, Y3", Binary: nil},
			},
		},
		{
			Name: "amd64-vpsubusb-2registers",
			Cfg:  config.AMD64(),
			Func: Function{
				Lines: []Line{
					{Assembly: "vpsubusb\tymm4, ymm4, ymmword ptr [rip + .LCPI0_3]", Disassembled: "VPSUBUSB 0(IP), Y4, Y4"},
				},
			},
			ExpectedLines: []Line{
				{Disassembled: "VPSUBUSB LCPI0_3<>(SB), Y4, Y4", Binary: nil},
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
