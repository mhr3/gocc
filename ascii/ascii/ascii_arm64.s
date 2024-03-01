//go:build !noasm && darwin && arm64 && manual

#include "textflag.h"

TEXT ·index_nonascii(SB),NOSPLIT,$0-24
	MOVD data_base+0(FP), R0
	MOVD data_len+8(FP), R1
	MOVD $ret+16(FP), R2
	MOVD $0, R8
	VMOVI $128, V0.B16

LBB0_1:
	ADD $16, R8, R9
	CMP R1, R9
	BHI LBB0_4
	WORD $0x3ce86801 // ldr	q1, [x0, x8]
	VAND V0.B16, V1.B16, V2.B16
	WORD $0x6e30a842 // VUMAXV V2.B16, V2
	WORD $0x1e26004a // fmov	w10, s2
	MOVD R9, R8
	CBZ R10, LBB0_1
	SUB $16, R9, R8
	WORD $0x6f07e7e0 // VMOVI $-1, V0.D2
	WORD $0x4e203420 // VCMGT V0.B16, V1.B16, V0.B16

    VMOVQ $0x0807060504030201, $0x100F0E0D0C0B0A09, V1
	VORR V1.B16, V0.B16, V0.B16
	WORD $0x6e31a800 // VUMINV V0.B16, V0
	WORD $0x1e260009 // fmov	w9, s0
	B LBB0_7

LBB0_4:
	ORR $8, R8, R9
	CMP R1, R9
	BHI LBB0_10
	WORD $0xfc686800 // ldr	d0, [x0, x8]
	VMOVI $128, V1.B8
	VAND V1.B8, V0.B8, V1.B8
	WORD $0x2e30a821 // VUMAXV V1.B8, V1
	WORD $0x1e26002a // fmov	w10, s1
	CBZW R10, LBB0_9
	WORD $0x6f07e7e1 // VMOVI $-1, V1.D2
	WORD $0x0e213400 // VCMGT V1.B8, V0.B8, V0.B8

    VMOVD $0x0807060504030201, V1
	VORR V1.B8, V0.B8, V0.B8
	WORD $0x2e31a800 // VUMINV V0.B8, V0
	WORD $0x1e260009 // fmov	w9, s0

LBB0_7:
	ADD R9, R8, R8
	SUB $1, R8, R8

LBB0_8:
	MOVD R8, (R2)
	RET

LBB0_9:
	MOVD R9, R8

LBB0_10:
	CMP R1, R8
	BHS LBB0_13

LBB0_11:
	WORD $0x38e86809 // ldrsb	w9, [x0, x8]
	TBNZ $31, R9, LBB0_8
	ADD $1, R8, R8
	CMP R8, R1
	BNE LBB0_11

LBB0_13:
	MOVD $-1, R8
	MOVD R8, (R2)
	RET

TEXT ·is_ascii(SB), $0-32
	MOVD data+0(FP), R0
	MOVD length+8(FP), R1
	MOVD $ret+16(FP), R2
	WORD $0xa9bf7bfd      // stp	x29, x30, [sp, #-16]!
	WORD $0x910003fd      // mov	x29, sp
	WORD $0xd2800008      // mov	x8, #0
	WORD $0x4f04e400      // movi.16b	v0, #128

LBB1_1:
	WORD $0x91004109 // add	x9, x8, #16
	WORD $0xeb01013f // cmp	x9, x1
	WORD $0x54000108 // b.hi	LBB1_3
	WORD $0x3ce86801 // ldr	q1, [x0, x8]
	WORD $0x4e201c21 // and.16b	v1, v1, v0
	WORD $0x6e30a821 // umaxv.16b	b1, v1
	WORD $0x1e26002a // fmov	w10, s1
	WORD $0xaa0903e8 // mov	x8, x9
	WORD $0x34ffff0a // cbz	w10, LBB1_1
	WORD $0x1400000a // b	LBB1_5

LBB1_3:
	WORD $0xb27d0109 // orr	x9, x8, #0x8
	WORD $0xeb01013f // cmp	x9, x1
	WORD $0x54000188 // b.hi	LBB1_8
	WORD $0xfc686800 // ldr	d0, [x0, x8]
	WORD $0x0f04e401 // movi.8b	v1, #128
	WORD $0x0e211c00 // and.8b	v0, v0, v1
	WORD $0x2e30a800 // umaxv.8b	b0, v0
	WORD $0x1e260008 // fmov	w8, s0
	WORD $0x340000a8 // cbz	w8, LBB1_7

LBB1_5:
	WORD $0xd2800008 // mov	x8, #0

LBB1_6:
	WORD $0xf9000048 // str	x8, [x2]
	WORD $0xa8c17bfd // ldp	x29, x30, [sp], #16
	WORD $0xd65f03c0 // ret

LBB1_7:
	WORD $0xaa0903e8 // mov	x8, x9

LBB1_8:
	WORD $0xeb01011f // cmp	x8, x1
	WORD $0x54000142 // b.hs	LBB1_12
	WORD $0xcb080029 // sub	x9, x1, x8
	WORD $0x8b08000a // add	x10, x0, x8
	WORD $0x52800028 // mov	w8, #1

LBB1_10:
	WORD $0x39c0014b // ldrsb	w11, [x10]
	WORD $0x37fffeab // tbnz	w11, #31, LBB1_5
	WORD $0x9100054a // add	x10, x10, #1
	WORD $0xf1000529 // subs	x9, x9, #1
	WORD $0x54ffff81 // b.ne	LBB1_10
	WORD $0x17fffff2 // b	LBB1_6

LBB1_12:
	WORD $0x52800028 // mov	w8, #1
	WORD $0xf9000048 // str	x8, [x2]
	WORD $0xa8c17bfd // ldp	x29, x30, [sp], #16
	WORD $0xd65f03c0 // ret
