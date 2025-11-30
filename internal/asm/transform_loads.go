package asm

import (
	"fmt"
	"strings"

	"github.com/mhr3/gocc/internal/config"
	"golang.org/x/arch/arm64/arm64asm"
)

func rewriteLoadAmd64(arch *config.Arch, _ Function, line Line, combined string, lines []Line) {
	reParams := getRegexpParams(arch.DataLoad, combined)
	// FIXME: this is extremely fragile
	op := arch.MovInstr[8]
	register := reParams["register"]
	symbol := reParams["var"]

	// sigh, why oh why do we have to do this?
	switch instr := reParams["instr"]; {
	case instr == "MOVDQA":
		op = "MOVO"
	case instr == "MOVDQU":
		op = "MOVOU"
	default:
		op = instr
	}

	if strings.HasPrefix(register, "X") {
		// the disassembler gets this wrong sometimes
		switch {
		case strings.Contains(line.Assembly, "xmm"):
			// all good
		case strings.Contains(line.Assembly, "ymm"):
			register = "Y" + register[1:]
		case strings.Contains(line.Assembly, "zmm"):
			register = "Z" + register[1:] // does go even support this?
		}
	}

	rewritten := fmt.Sprintf("%s %s<>(SB), %s", op, symbol, register)
	lines[0].Disassembled = rewritten
	lines[0].Binary = nil
}

func rewriteLoadArm64(arch *config.Arch, _ Function, _ Line, combined string, lines []Line) {
	// processes ADRP instructions
	reParams := getRegexpParams(arch.DataLoad, combined)

	op := arch.MovInstr[8]
	register := reParams["register"]
	symbol := reParams["var"]

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
