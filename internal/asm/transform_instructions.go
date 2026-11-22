package asm

import (
	"regexp"
	"strings"

	"github.com/mhr3/gocc/internal/config"
)

// amd64UnassemblableMnemonics lists Plan 9 mnemonics produced by x86asm's
// GoSyntax that cmd/asm does not accept or encodes differently. For these
// the compiler's machine code is kept and emitted as raw bytes. Everything
// else round-trips through cmd/asm to the same instruction.
//
// The list was established with Go 1.27 (cmd/asm) and golang.org/x/arch at
// 3ddaabb; cmd/asm's mnemonic set changes between releases (it accepts
// SETNE but not SETE, for instance), so TestAmd64DenylistRoundTrips checks
// both halves of the split against the assembler of the Go toolchain that
// builds gocc, which is what will assemble the generated code.
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

// arm64AssemblableMnemonics lists Plan 9 mnemonics produced by arm64asm's
// GoSyntax that cmd/asm reassembles to the same encoding, for the operand
// shapes compiler output uses. Unlike amd64 this is an allowlist: cmd/asm
// covers a fraction of NEON, arm64asm decodes no SVE at all, and GoSyntax
// spells several scalar/element/immediate forms identically to vector forms
// that cmd/asm cannot encode. Everything else is emitted as raw bytes with
// the decoded instruction in a comment.
//
// Established with Go 1.27 (cmd/asm) and golang.org/x/arch at 3ddaabb;
// TestArm64AllowlistRoundTrips checks it against the toolchain building gocc.
var arm64AssemblableMnemonics = map[string]bool{
	// integer
	"ADD": true, "ADDS": true, "ADDW": true, "ADDSW": true, "SUB": true, "SUBS": true, "SUBW": true, "SUBSW": true,
	"NEG": true, "NEGW": true, "NEGS": true, "NEGSW": true, "MUL": true, "MULW": true, "MADD": true, "MADDW": true,
	"MSUB": true, "MSUBW": true, "SMULL": true, "UMULL": true, "SMULH": true, "UMULH": true, "SMADDL": true, "UMADDL": true,
	"SDIV": true, "SDIVW": true, "UDIV": true, "UDIVW": true,
	"AND": true, "ANDW": true, "ANDS": true, "ANDSW": true, "ORR": true, "ORRW": true, "ORN": true, "ORNW": true,
	"EOR": true, "EORW": true, "EON": true, "EONW": true, "BIC": true, "BICW": true, "BICS": true, "BICSW": true,
	"TST": true, "TSTW": true, "MVN": true, "MVNW": true,
	"LSL": true, "LSLW": true, "LSR": true, "LSRW": true, "ASR": true, "ASRW": true, "ROR": true, "RORW": true,
	"UBFX": true, "UBFXW": true, "SBFX": true, "SBFXW": true, "UBFIZ": true, "UBFIZW": true, "SBFIZ": true, "SBFIZW": true,
	"BFI": true, "BFIW": true, "BFXIL": true, "BFXILW": true, "EXTR": true, "EXTRW": true,
	"SXTB": true, "SXTBW": true, "SXTH": true, "SXTHW": true, "SXTW": true, "UXTBW": true, "UXTHW": true,
	"CLS": true, "CLSW": true, "CLZ": true, "CLZW": true, "RBIT": true, "RBITW": true,
	"REV": true, "REVW": true, "REV16": true, "REV16W": true, "REV32": true,
	"CMP": true, "CMPW": true, "CMN": true, "CMNW": true, "CCMP": true, "CCMPW": true, "CCMN": true, "CCMNW": true,
	"CSEL": true, "CSELW": true, "CSINC": true, "CSINCW": true, "CSINV": true, "CSINVW": true, "CSNEG": true, "CSNEGW": true,
	"CSET": true, "CSETW": true, "CSETM": true, "CSETMW": true, "CINC": true, "CINCW": true, "CINV": true, "CINVW": true,
	"CNEG": true, "CNEGW": true,
	// MOVZ/MOVN decode as MOVD/MOVW with an immediate. Register MOVW is not
	// safe (see arm64TrustDecoder).
	"MOVD": true, "MOVW": true, "MOVK": true, "MOVKW": true,
	// loads and stores (fixed-offset, pre/post-indexed and register-offset forms)
	"MOVB": true, "MOVB.P": true, "MOVB.W": true, "MOVBU": true, "MOVBU.P": true, "MOVBU.W": true,
	"MOVH": true, "MOVH.P": true, "MOVH.W": true, "MOVHU": true, "MOVHU.P": true, "MOVHU.W": true,
	"MOVWU": true, "MOVWU.P": true, "MOVWU.W": true, "MOVD.P": true, "MOVD.W": true,
	"LDP": true, "LDP.P": true, "LDP.W": true, "LDPW": true, "LDPW.P": true, "LDPW.W": true,
	"STP": true, "STP.P": true, "STP.W": true, "STPW": true, "STPW.P": true, "STPW.W": true,
	"FMOVS": true, "FMOVS.P": true, "FMOVS.W": true, "FMOVD": true, "FMOVD.P": true, "FMOVD.W": true,
	"FMOVQ": true, "FMOVQ.P": true, "FMOVQ.W": true,
	"FLDPS": true, "FLDPD": true, "FLDPQ": true, "FSTPS": true, "FSTPD": true, "FSTPQ": true,
	"LDAR": true, "LDARW": true, "LDAXR": true, "LDAXRW": true, "STLR": true, "STLRW": true, "STLXR": true, "STLXRW": true,
	"PRFM": true,
	// control flow, system
	"RET": true, "CALL": true, "JMP": true, "NOOP": true, "BRK": true, "HLT": true, "DMB": true, "DSB": true, "ISB": true,
	"CRC32B": true, "CRC32H": true, "CRC32W": true, "CRC32X": true, "CRC32CB": true, "CRC32CH": true, "CRC32CW": true, "CRC32CX": true,
	// scalar floating point
	"FADDS": true, "FADDD": true, "FSUBS": true, "FSUBD": true, "FMULS": true, "FMULD": true, "FDIVS": true, "FDIVD": true,
	"FNMULS": true, "FNMULD": true, "FMADDS": true, "FMADDD": true, "FMSUBS": true, "FMSUBD": true,
	"FNMADDS": true, "FNMADDD": true, "FNMSUBS": true, "FNMSUBD": true, "FSQRTS": true, "FSQRTD": true,
	"FABSS": true, "FABSD": true, "FNEGS": true, "FNEGD": true, "FMAXS": true, "FMAXD": true, "FMINS": true, "FMIND": true,
	"FMAXNMS": true, "FMAXNMD": true, "FMINNMS": true, "FMINNMD": true,
	"FCMPS": true, "FCMPD": true, "FCMPES": true, "FCMPED": true, "FCCMPS": true, "FCCMPD": true, "FCSELS": true, "FCSELD": true,
	"FCVTSD": true, "FCVTDS": true, "FCVTHS": true, "FCVTSH": true, "FCVTHD": true, "FCVTDH": true,
	"FCVTZSS": true, "FCVTZSD": true, "FCVTZSSW": true, "FCVTZSDW": true, "FCVTZUS": true, "FCVTZUD": true, "FCVTZUSW": true, "FCVTZUDW": true,
	"SCVTFS": true, "SCVTFD": true, "SCVTFWS": true, "SCVTFWD": true, "UCVTFS": true, "UCVTFD": true, "UCVTFWS": true, "UCVTFWD": true,
	"FRINTAS": true, "FRINTAD": true, "FRINTIS": true, "FRINTID": true, "FRINTMS": true, "FRINTMD": true, "FRINTNS": true, "FRINTND": true,
	"FRINTPS": true, "FRINTPD": true, "FRINTXS": true, "FRINTXD": true, "FRINTZS": true, "FRINTZD": true,
	// NEON
	"VADD": true, "VSUB": true, "VMUL": true, "VMLA": true, "VMLS": true, "VNEG": true, "VABS": true,
	"VAND": true, "VORR": true, "VORN": true, "VEOR": true, "VBIC": true, "VBSL": true, "VBIT": true, "VBIF": true,
	"VMOV": true, "VMOVI": true, "VDUP": true, "VEXT": true, "VTBL": true, "VTBX": true,
	"VCNT": true, "VCLS": true, "VCLZ": true, "VRBIT": true, "VREV16": true, "VREV32": true, "VREV64": true,
	"VSHL": true, "VUSHR": true, "VSSHR": true, "VUSHL": true, "VSSHL": true, "VSLI": true, "VSRI": true, "VUSRA": true, "VSRSHR": true,
	"VUSHLL": true, "VUSHLL2": true, "VUXTL": true, "VUXTL2": true, "VSXTL": true, "VSXTL2": true,
	"VSHRN": true, "VSHRN2": true, "VXTN": true, "VXTN2": true, "VUQXTN": true, "VUQXTN2": true, "VSQXTN": true, "VSQXTN2": true,
	"VSQXTUN": true, "VSQXTUN2": true,
	"VCMEQ": true, "VCMHI": true, "VCMHS": true, "VCMGT": true, "VCMGE": true, "VCMLT": true, "VCMLE": true, "VCMTST": true,
	"VUMAX": true, "VUMIN": true, "VSMAX": true, "VSMIN": true, "VUMAXV": true, "VUMINV": true, "VSMAXV": true, "VSMINV": true,
	"VUMAXP": true, "VUMINP": true, "VSMAXP": true, "VSMINP": true, "VADDV": true, "VADDP": true, "VUADDLV": true,
	"VUADDW": true, "VUADDW2": true, "VUHADD": true, "VURHADD": true, "VSHADD": true, "VUQADD": true, "VSQADD": true, "VUQSUB": true, "VSQSUB": true,
	"VUMULL": true, "VUMULL2": true, "VSMULL": true, "VSMULL2": true, "VUMLAL": true, "VUMLAL2": true, "VSMLAL": true, "VSMLAL2": true,
	"VPMULL": true, "VPMULL2": true,
	"VZIP1": true, "VZIP2": true, "VUZP1": true, "VUZP2": true, "VTRN1": true, "VTRN2": true,
	"VFADDP": true, "VFMAXP": true, "VFMLA": true, "VFMLS": true, "VFCMEQ": true, "VFCMGE": true, "VFCMGT": true,
	"VFCVTL": true, "VFCVTL2": true, "VFCVTN": true, "VFCVTN2": true,
	"VLD1": true, "VLD1.P": true, "VLD1R": true, "VLD1R.P": true, "VLD2": true, "VLD2.P": true, "VLD2R": true, "VLD2R.P": true,
	"VLD3": true, "VLD3.P": true, "VLD3R": true, "VLD3R.P": true, "VLD4": true, "VLD4.P": true, "VLD4R": true, "VLD4R.P": true,
	"VST1": true, "VST1.P": true,
	"AESE": true, "AESD": true, "AESMC": true, "AESIMC": true,
	"SHA1C": true, "SHA1H": true, "SHA1M": true, "SHA1P": true, "SHA1SU0": true, "SHA1SU1": true,
	"SHA256H": true, "SHA256H2": true, "SHA256SU0": true, "SHA256SU1": true,
}

var arm64ElementOperand = regexp.MustCompile(`\bV[0-9]+\.[BHSD]\[`)
var arm64ScalarVectorSource = regexp.MustCompile(`^(?:\$[^,]+, )?V[0-9]+,`)
var arm64ScalarVectorDest = regexp.MustCompile(`, V[0-9]+$`)
var arm64RegisterOffset = regexp.MustCompile(`\(R[0-9]+\)\(R[0-9]+`)

// arm64TrustDecoder reports whether cmd/asm reassembles the decoder's Plan 9
// text to the same encoding, so the compiler encoding can be dropped.
func arm64TrustDecoder(line Line) bool {
	disassembled := line.Disassembled
	if disassembled == "" || strings.Contains(disassembled, "(SB)") {
		return false
	}
	mnemonic, operands, _ := strings.Cut(disassembled, " ")
	if !arm64AssemblableMnemonics[mnemonic] {
		return false
	}
	switch {
	case mnemonic == "MOVW" && !strings.HasPrefix(operands, "$"):
		// C's mov w0, w1 zero-extends into x0; cmd/asm's MOVW is a
		// sign-extending move (sxtw). Clang relies on the zero extension,
		// so register MOVW must keep its encoding.
		return false
	case strings.HasPrefix(mnemonic, "V") && arm64ElementOperand.MatchString(operands):
		// Element sources (fmla v0.4s, v1.4s, v2.s[1]) have no Plan 9 form
		// for arithmetic; VMOV/VDUP element moves do work.
		return mnemonic == "VMOV" || mnemonic == "VDUP"
	case strings.HasPrefix(mnemonic, "V") && arm64ScalarVectorSource.MatchString(operands):
		// Scalar NEON (shl d0, d1, #4 -> VSHL $4, V1, V0) is spelled like a
		// vector operand without an arrangement; cmd/asm rejects it. Vector
		// reductions (VADDV V1.B16, V0) have a scalar destination only.
		return false
	case mnemonic == "VADDP" && arm64ScalarVectorDest.MatchString(operands):
		// addp d0, v1.2d: the scalar pairwise form is not encodable, unlike
		// the reductions whose scalar destination cmd/asm accepts.
		return false
	case strings.HasPrefix(mnemonic, "V") && strings.HasPrefix(operands, "$"):
		// Immediate forms are assemblable for shifts and zero compares;
		// ORR/BIC with immediate and FCM* against zero are not.
		switch mnemonic {
		case "VSHL", "VUSHR", "VSSHR", "VSLI", "VSRI", "VUSRA", "VSRSHR", "VUSHLL", "VUSHLL2", "VSHRN", "VSHRN2", "VEXT",
			"VCMEQ", "VCMGT", "VCMGE", "VCMLT", "VCMLE":
			return true
		case "VMOVI":
			// cmd/asm only has the byte form (movi v0.16b, #imm8). It rejects
			// the wider elements, their shifted immediates and the 64-bit
			// byte mask, all of which GoSyntax spells the same way.
			return strings.HasSuffix(operands, ".B8") || strings.HasSuffix(operands, ".B16")
		}
		return false
	case (mnemonic == "VFCVTN" || mnemonic == "VFCVTN2" || mnemonic == "VFCVTL" || mnemonic == "VFCVTL2") && strings.Contains(operands, ".H"):
		// cmd/asm encodes the half-precision forms as the single->double ones.
		return false
	case (mnemonic == "FMULS" || mnemonic == "FMULD") && arm64ElementOperand.MatchString(operands):
		return false
	case strings.HasPrefix(mnemonic, "FMOV") && arm64RegisterOffset.MatchString(operands):
		// FMOVQ (R0)(R1), F0: register-offset 128-bit loads are not encodable.
		return mnemonic != "FMOVQ"
	case strings.HasPrefix(mnemonic, "MRS") || strings.HasPrefix(mnemonic, "MSR"):
		return false
	}
	return true
}

func removeBinaryInstructionsArm64(_ *config.Arch, function Function) Function {
	for i := range function.Lines {
		line := &function.Lines[i]
		if len(line.Binary) > 0 && arm64TrustDecoder(*line) {
			line.Binary = nil
		}
	}
	return function
}
