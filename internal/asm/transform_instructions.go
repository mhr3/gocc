package asm

import (
	"strings"

	"github.com/mhr3/gocc/internal/config"
)

// amd64UnassemblableMnemonics lists Plan 9 mnemonics produced by x86asm's
// GoSyntax that cmd/asm does not accept or encodes differently. For these
// the compiler's machine code is kept and emitted as raw bytes. Everything
// else round-trips through cmd/asm to the same instruction; see
// TestAmd64DenylistRoundTrips, which checks both halves against the real
// assembler.
var amd64UnassemblableMnemonics = map[string]string{
	// x87: cmd/asm names loads/stores and arithmetic with operand widths
	// (FMOVD, FMULD, FADDDP, ...), which GoSyntax does not emit.
	"FLD": "FMOVD/FMOVF/FMOVX", "FSTP": "FMOVDP/FMOVFP/FMOVXP", "FILD": "FMOVL/FMOVQ", "FISTP": "FMOVLP/FMOVQP",
	"FMUL": "FMULD", "FMULP": "FMULDP", "FADD": "FADDD", "FADDP": "FADDDP",
	"FSUBRP": "FSUBRDP", "FDIVRP": "FDIVRDP", "FXCH": "FXCHD", "FUCOMIP": "FUCOMIP F0, F1 (operand order)",
	"FNSTCW": "FSTCW",
	// Integer: cmd/asm wants operand-size suffixes or different names.
	"BSWAP": "BSWAPQ/BSWAPL", "CRC32": "CRC32B/W/L/Q", "MOVBE": "MOVBELL/MOVBEQQ",
	"TZCNT": "TZCNTL/Q", "LZCNT": "LZCNTL/Q",
	"SHLDQ": "SHLQ CL, DX, AX", "SHRDQ": "SHRQ CL, DX, AX", "SHLDL": "SHLL ...", "SHRDL": "SHRL ...",
	"MOVZX": "MOVBLZX/MOVBQZX/MOVWLZX/MOVWQZX", "MOVSX": "MOVBLSX/MOVBQSX/MOVWLSX/MOVWQSX", "MOVSXD": "MOVLQSX",
	"INT": "INT $3 encodes as CD 03, not the INT3 opcode",
	// Conversions: cmd/asm uses S{L,Q} / P{L,S,D} naming (CVTSQ2SD, CVTTSD2SL, CVTPL2PD).
	"CVTSI2SDL": "CVTSL2SD", "CVTSI2SDQ": "CVTSQ2SD", "CVTSI2SSL": "CVTSL2SS", "CVTSI2SSQ": "CVTSQ2SS",
	"CVTTSD2SIL": "CVTTSD2SL", "CVTTSD2SIQ": "CVTTSD2SQ", "CVTTSS2SIL": "CVTTSS2SL", "CVTTSS2SIQ": "CVTTSS2SQ",
	"CVTSD2SIL": "CVTSD2SL", "CVTSD2SIQ": "CVTSD2SQ", "CVTSS2SIL": "CVTSS2SL", "CVTSS2SIQ": "CVTSS2SQ",
	"CVTDQ2PD": "CVTPL2PD", "CVTDQ2PS": "CVTPL2PS", "CVTPD2DQ": "CVTPD2PL", "CVTPS2DQ": "CVTPS2PL",
	"CVTTPD2DQ": "CVTTPD2PL", "CVTTPS2DQ": "CVTTPS2PL",
	// SSE: legacy names cmd/asm never adopted.
	"MOVDQA": "MOVO", "MOVDQU": "MOVOU", "MOVSD_XMM": "MOVSD", "CMPSD_XMM": "CMPSD",
	"PUNPCKLWD": "PUNPCKLWL", "PUNPCKLDQ": "PUNPCKLLQ", "PUNPCKHWD": "PUNPCKHWL", "PUNPCKHDQ": "PUNPCKHLQ",
	"PMULUDQ": "PMULULQ", "PMADDWD": "PMADDWL", "PSRLD": "PSRLL", "PSRAD": "PSRAL", "PSLLD": "PSLLL",
	"PSUBD": "PSUBL", "PCMPEQD": "PCMPEQL", "PCMPGTD": "PCMPGTL", "PACKSSDW": "PACKSSLW",
	// Predicated compares take the immediate last in cmd/asm.
	"CMPPS": "CMPPS X1, X0, $imm", "CMPPD": "CMPPD X1, X0, $imm", "CMPSS": "CMPSS X1, X0, $imm",
}

// amd64UnassemblablePrefixes covers whole families with a different naming
// scheme in cmd/asm.
var amd64UnassemblablePrefixes = []string{
	// SETE -> SETEQ, SETB -> SETCS, ... cmd/asm happens to accept a few of
	// the Intel condition names (SETNE, SETA); deny the family rather than
	// depend on which aliases exist in a given Go release.
	"SET",
	"CMOV", // CMOVE -> CMOVQEQ, needs an operand-size infix
	"LOCK", // GoSyntax prints "LOCK XADDQ ..." on one line; cmd/asm wants "LOCK; XADDQ ..."
	"REP",  // same, and the string instructions below it are unassemblable anyway
}

// amd64TrustDecoder reports whether cmd/asm reassembles the decoder's Plan 9
// text to the same instruction, so the compiler encoding can be dropped.
func amd64TrustDecoder(line Line) bool {
	disassembled := line.Disassembled
	if disassembled == "" || strings.Contains(disassembled, "(SB)") {
		// Symbol references need relocation; rewriteLoadAmd64 already decided
		// how they are spelled.
		return false
	}
	mnemonic, operands, _ := strings.Cut(disassembled, " ")
	if _, ok := amd64UnassemblableMnemonics[mnemonic]; ok {
		return false
	}
	for _, prefix := range amd64UnassemblablePrefixes {
		if strings.HasPrefix(mnemonic, prefix) {
			return false
		}
	}
	switch {
	case strings.Contains(operands, "(IP)"):
		// A RIP-relative operand that rewriteLoadAmd64 did not recognise.
		return false
	case strings.HasPrefix(mnemonic, "IMUL") && strings.Count(operands, ",") == 2:
		// Three-operand immediate form is IMUL3Q in cmd/asm.
		return false
	case mnemonic == "MOVD" && (strings.Contains(line.Assembly, "xmm") || strings.Contains(line.Assembly, "ymm")):
		// GoSyntax prints MOVD for both movd (32-bit, MOVL in cmd/asm) and
		// movq (64-bit) GPR<->XMM moves; cmd/asm's MOVD is the 64-bit one.
		return false
	}
	return true
}

func removeBinaryInstructionsAmd64(_ *config.Arch, function Function) Function {
	for i := range function.Lines {
		line := &function.Lines[i]
		if len(line.Binary) > 0 && amd64TrustDecoder(*line) {
			line.Binary = nil
		}
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
					// denylist: cmd/asm has no spelling for these
					case "LDURBW", "SMOVW", "SMOVD":
					default:
						line.Binary = nil
					}
				case strings.HasPrefix(dInst, "V") && dInst[1:] == inst:
					if strings.Contains(line.Assembly, "#0") {
						break
					}
					// some variants of VLD1R don't work, do not add below
					switch dInst {
					// allowlist
					case "VLD1", "VDUP", "VMOV", "VTBL", "VADD", "VSUB", "VAND", "VORR", "VEOR", "VEXT", "VCNT", "VSHL", "VSHR", "VCMTST", "VCMEQ":
						line.Binary = nil
					}
				}
			}
		}

		function.Lines[i] = line
	}

	return function
}
