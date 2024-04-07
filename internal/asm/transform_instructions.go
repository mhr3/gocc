package asm

import (
	"strings"

	"github.com/kelindar/gocc/internal/config"
)

func removeBinaryInstructionsAmd64(_ *config.Arch, function Function) Function {
	for i := 0; i < len(function.Lines); i++ {
		line := function.Lines[i]

		if line.Disassembled != "" && len(line.Binary) > 0 {
			asmParts := strings.Fields(line.Assembly)
			if len(asmParts) > 0 {
				// do the instructions match?
				inst := strings.ToUpper(asmParts[0])
				dInst := line.Disassembled
				dIdx := strings.IndexByte(line.Disassembled, ' ')
				if dIdx > 0 {
					dInst = line.Disassembled[:dIdx]
				}

				switch {
				case strings.Contains(line.Disassembled, "(SB)"):
					// definitely not
				case inst == "ADD" || inst == "SUB":
					// only some variants are ok
					if len(dInst) == len(inst)+1 && strings.HasPrefix(dInst, inst) && strings.HasSuffix(dInst, "Q") {
						line.Binary = nil
					}
				case inst == "CMP" || inst == "TEST":
					// operands can be reversed, skip
				case inst == dInst:
					if strings.Contains(line.Assembly, "ymm") || strings.Contains(line.Assembly, "zmm") {
						// disassembler gets this wrong
						break
					} else if inst == "FMUL" {
						break
					} else if strings.HasPrefix(inst, "CMOV") {
						break
					} else if strings.HasPrefix(inst, "MOVDQ") {
						// should be disassembled as MOVO and MOVOU
						break
					} else if strings.HasPrefix(inst, "MOVSX") || strings.HasPrefix(inst, "MOVZX") {
						break
					}
					// otherwise trust the disassembler
					line.Binary = nil
				case inst == "MOV" && strings.HasPrefix(dInst, "MOV"):
					// we'll trust the disassembler
					line.Binary = nil
				case strings.HasPrefix(dInst, inst) && len(dInst) == len(inst)+1:
					// we'll trust the disassembler
					line.Binary = nil
				}
			}
		}

		function.Lines[i] = line
	}

	return function
}

func removeBinaryInstructionsArm64(_ *config.Arch, function Function) Function {
	for i := 0; i < len(function.Lines); i++ {
		line := function.Lines[i]

		if line.Disassembled != "" && len(line.Binary) > 0 {
			asmParts := strings.Fields(line.Assembly)
			if len(asmParts) > 0 {
				// do the instructions match?
				inst := strings.ToUpper(asmParts[0])
				dInst := line.Disassembled
				dIdx := strings.IndexByte(line.Disassembled, ' ')
				if dIdx > 0 {
					dInst = line.Disassembled[:dIdx]
				}

				switch {
				case strings.Contains(line.Disassembled, "(SB)"):
					// definitely not
				case inst == dInst:
					if inst == "FMUL" {
						// no such instruction ???
						break
					}
					line.Binary = nil
				case strings.HasPrefix(dInst, inst) && len(dInst) == len(inst)+1:
					line.Binary = nil
				}
			}
		}

		function.Lines[i] = line
	}

	return function
}