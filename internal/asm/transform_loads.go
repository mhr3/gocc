package asm

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mhr3/gocc/internal/config"
	"golang.org/x/arch/arm64/arm64asm"
)

// amd64RIPOperand matches the RIP-relative memory operand in x86asm's Plan 9
// text. Legacy encodings print as N(IP); for VEX/EVEX encodings x86asm drops
// the RIP base and prints the displacement alone (golang/go#83116), so a bare
// number operand is accepted as well.
var amd64RIPOperand = regexp.MustCompile(`(?:^|, )(-?(?:0x[0-9a-fA-F]+|[0-9]+)(?:\(IP\))?)(?:, |$)`)

// rewriteLoadAmd64 replaces a RIP-relative memory operand with a reference
// to the constant the compiler placed there. Only that operand is touched:
// the decoder's Plan 9 text keeps every other operand, including implicit
// ones such as the X0 mask of PBLENDVB and immediates.
func rewriteLoadAmd64(arch *config.Arch, _ Function, line Line, combined string, lines []Line) {
	reParams := getRegexpParams(arch.DataLoad, combined)
	symbol := DataSymbolName(reParams["var"])

	op, operands, _ := strings.Cut(line.Disassembled, " ")
	switch op {
	case "MOVDQA":
		op = "MOVO"
	case "MOVDQU":
		op = "MOVOU"
	case "MOVSD_XMM":
		// x86asm distinguishes the SSE move from the string instruction.
		op = "MOVSD"
	}
	match := amd64RIPOperand.FindStringSubmatchIndex(operands)
	if match == nil {
		// Leave the line alone; amd64TrustDecoder keeps its encoding and the
		// decoded text stays in the comment for inspection.
		return
	}
	operands = operands[:match[2]] + symbol + "<>(SB)" + operands[match[3]:]

	lines[0].Disassembled = op + " " + operands
	lines[0].Binary = nil
}

func rewriteLoadArm64(arch *config.Arch, _ Function, _ Line, combined string, lines []Line) {
	// processes ADRP instructions
	reParams := getRegexpParams(arch.DataLoad, combined)

	op := arch.MovInstr[8]
	register := reParams["register"]
	symbol := DataSymbolName(reParams["var"])

	// addrMode := "$" // absolute addressing
	rewritten := fmt.Sprintf("%s $%s<>(SB), %s", op, symbol, register)
	lines[0].Disassembled = rewritten
	lines[0].Binary = nil

	if len(lines) > 1 {
		nextLine := lines[1]

		inst := decodeArm64Line(nextLine)
		switch inst.Op {
		case arm64asm.ADD:
			if len(inst.Args) < 3 {
				return
			}
			if arg, ok := inst.Args[2].(arm64asm.ImmShift); ok && arg.String() == "#0x0" {
				// darwin uses PAGEOFF
				if strings.Contains(nextLine.Assembly, "@PAGEOFF") || strings.Contains(nextLine.Assembly, ":lo") {
					lines[1].Disassembled = "NOP"
					lines[1].Binary = nil
				}
			}
		default:
			return
		}
	}
}
