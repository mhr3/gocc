package asm

import (
	"strings"

	"github.com/mhr3/gocc/internal/config"
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
				case strings.HasPrefix(inst, "CVT"):
					// nope
				case strings.HasPrefix(inst, "SET"):
					// nope
				case strings.HasPrefix(inst, "PSR"):
					// nope
				case strings.HasSuffix(dInst, "L"):
					// the toolchain is so bad with these, skip
					// unless it's CMPL or XORL
					if dInst == "CMPL" || dInst == "XORL" {
						line.Binary = nil
					}
				case inst == "ADD" || inst == "SUB" || inst == "AND" || inst == "OR":
					line.Binary = nil
				case inst == "TEST":
					// operands can be reversed, skip
				case inst == dInst:
					if strings.Contains(line.Assembly, "xmm") || strings.Contains(line.Assembly, "ymm") || strings.Contains(line.Assembly, "zmm") {
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
					// we'll trust the disassembler, unless it's MOVL
					if strings.HasPrefix(dInst, "MOVL") {
						break
					}
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
					skip := false
					switch inst {
					case "FMUL":
						// no such instruction ???
						skip = true
					case "MRS":
						skip = true
					}

					if skip {
						break
					}
					line.Binary = nil
				case strings.HasPrefix(dInst, inst) && len(dInst) == len(inst)+1:
					switch dInst {
					// denylist
					case "LDURBW":
					default:
						line.Binary = nil
					}
				case strings.HasPrefix(dInst, "V") && dInst[1:] == inst:
					switch dInst {
					// allowlist
					case "VLD1", "VLD1R", "VDUP", "VMOV", "VTBL", "VADD", "VSUB", "VAND", "VORR", "VEOR", "VEXT", "VCNT", "VSHL", "VSHR", "VCMTST", "VCMEQ":
						line.Binary = nil
					}
				}
			}
		}

		function.Lines[i] = line
	}

	return function
}
