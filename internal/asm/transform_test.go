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
					// VEX encodings: x86asm prints the RIP operand as a bare displacement.
					{Assembly: "vpand\tymm3, ymm7, ymmword ptr [rip + .LCPI0_0]", Disassembled: "VPAND 0x8, Y7, Y3"},
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
					{Assembly: "vpsubusb\tymm4, ymm4, ymmword ptr [rip + .LCPI0_3]", Disassembled: "VPSUBUSB 0x10, Y4, Y4"},
				},
			},
			ExpectedLines: []Line{
				{Disassembled: "VPSUBUSB LCPI0_3<>(SB), Y4, Y4", Binary: nil},
			},
		},
		{
			Name: "amd64-vpblendd-immediate",
			Cfg:  config.AMD64(),
			Func: Function{
				Lines: []Line{
					{Assembly: "vpblendd\tymm0, ymm0, ymmword ptr [rip + .LCPI0_27], 252", Disassembled: "VPBLENDD $0xfc, 0x20, Y0, Y0"},
				},
			},
			ExpectedLines: []Line{
				{Disassembled: "VPBLENDD $0xfc, LCPI0_27<>(SB), Y0, Y0", Binary: nil},
			},
		},
		{
			// The implicit X0 mask operand must survive the rewrite.
			Name: "amd64-pblendvb-implicit-mask",
			Cfg:  config.AMD64(),
			Func: Function{
				Lines: []Line{
					{Assembly: "pblendvb\txmm4, xmmword ptr [rip + .LCPI2_5], xmm0", Disassembled: "PBLENDVB X0, 0xc(IP), X4"},
					{Assembly: "blendvps\txmm4, xmmword ptr [rip + .LCPI2_6], xmm0", Disassembled: "BLENDVPS X0, 0x1c(IP), X4"},
				},
			},
			ExpectedLines: []Line{
				{Disassembled: "PBLENDVB X0, LCPI2_5<>(SB), X4", Binary: nil},
				{Disassembled: "BLENDVPS X0, LCPI2_6<>(SB), X4", Binary: nil},
			},
		},
		{
			Name: "amd64-vinserti128-mixed-width",
			Cfg:  config.AMD64(),
			Func: Function{
				Lines: []Line{
					{Assembly: "vinserti128\tymm0, ymm1, xmmword ptr [rip + .LCPI0_1], 1", Disassembled: "VINSERTI128 $0x1, 0x10, Y1, Y0"},
				},
			},
			ExpectedLines: []Line{
				{Disassembled: "VINSERTI128 $0x1, LCPI0_1<>(SB), Y1, Y0", Binary: nil},
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
			Name: "amd64-trusts-decoder-by-default",
			Cfg:  config.AMD64(),
			Func: Function{
				Lines: []Line{
					{Assembly: "cmp	cl, 8", Disassembled: "CMPB CL, $0x8", Binary: binaryFromHex("80f908")},
					{Assembly: "test	eax, eax", Disassembled: "TESTL AX, AX", Binary: binaryFromHex("85c0")},
					{Assembly: "vpaddq	ymm0, ymm1, ymm2", Disassembled: "VPADDQ Y2, Y1, Y0", Binary: binaryFromHex("c5f5d4c2")},
					{Assembly: "shlx	rax, rcx, rdx", Disassembled: "SHLXQ DX, CX, AX", Binary: binaryFromHex("c4e2e9f7c1")},
				},
			},
			ExpectedLines: []Line{
				{Disassembled: "CMPB CL, $0x8"},
				{Disassembled: "TESTL AX, AX"},
				{Disassembled: "VPADDQ Y2, Y1, Y0"},
				{Disassembled: "SHLXQ DX, CX, AX"},
			},
		},
		{
			Name: "amd64-keeps-unassemblable-encodings",
			Cfg:  config.AMD64(),
			Func: Function{
				Lines: []Line{
					{Assembly: "sete	al", Disassembled: "SETE AL", Binary: binaryFromHex("0f94c0")},
					{Assembly: "cmove	rax, rcx", Disassembled: "CMOVE CX, AX", Binary: binaryFromHex("480f44c1")},
					{Assembly: "movd	xmm0, eax", Disassembled: "MOVD AX, X0", Binary: binaryFromHex("660f6ec0")},
					{Assembly: "imul	rax, rcx, 12", Disassembled: "IMULQ $0xc, CX, AX", Binary: binaryFromHex("486bc10c")},
					{Assembly: "lea	rax, [rip + 16]", Disassembled: "LEAQ 0x10(IP), AX", Binary: binaryFromHex("488d0510000000")},
					{Assembly: "vpand	ymm0, ymm1, ymmword ptr [rip + .LCPI0_0]", Disassembled: "VPAND LCPI0_0<>(SB), Y1, Y0", Binary: binaryFromHex("c5f5db05")},
					{Assembly: "movdqa	xmm0, xmm1", Disassembled: "MOVDQA X1, X0", Binary: binaryFromHex("660f6fc1")},
				},
			},
			ExpectedLines: []Line{
				{Disassembled: "SETE AL", Binary: binaryIsSet},
				{Disassembled: "CMOVE CX, AX", Binary: binaryIsSet},
				{Disassembled: "MOVD AX, X0", Binary: binaryIsSet},
				{Disassembled: "IMULQ $0xc, CX, AX", Binary: binaryIsSet},
				{Disassembled: "LEAQ 0x10(IP), AX", Binary: binaryIsSet},
				{Disassembled: "VPAND LCPI0_0<>(SB), Y1, Y0", Binary: binaryIsSet},
				{Disassembled: "MOVDQA X1, X0", Binary: binaryIsSet},
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
