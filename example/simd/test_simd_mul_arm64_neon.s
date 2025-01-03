//go:build !noasm && arm64
// Code generated by gocc devel -- DO NOT EDIT.
//
// Source file         : test_simd_mul.c
// Clang version       : Apple clang version 16.0.0 (clang-1600.0.26.4)
// Target architecture : arm64
// Compiler options    : [none]

#include "textflag.h"

TEXT ·uint8_simd_mul_neon(SB), NOSPLIT, $0-32
	MOVD input1+0(FP), R0
	MOVD input2+8(FP), R1
	MOVD output+16(FP), R2
	MOVD size+24(FP), R3
	CMPW $1, R3              // <--                                  // cmp	w3, #1
	BLT  LBB0_6              // <--                                  // b.lt	.LBB0_6
	NOP                      // (skipped)                            // stp	x29, x30, [sp, #-16]!
	AND  $4294967295, R3, R8 // <--                                  // and	x8, x3, #0xffffffff
	NOP                      // (skipped)                            // mov	x29, sp
	CMP  $8, R8              // <--                                  // cmp	x8, #8
	BCS  LBB0_7              // <--                                  // b.hs	.LBB0_7
	MOVD ZR, R9              // <--                                  // mov	x9, xzr

LBB0_3:
	ADD R9, R2, R10 // <--                                  // add	x10, x2, x9
	ADD R9, R1, R11 // <--                                  // add	x11, x1, x9
	ADD R9, R0, R12 // <--                                  // add	x12, x0, x9
	SUB R9, R8, R8  // <--                                  // sub	x8, x8, x9

LBB0_4:
	WORD $0x38401589 // MOVBU.P 1(R12), R9                   // ldrb	w9, [x12], #1
	WORD $0x3840156d // MOVBU.P 1(R11), R13                  // ldrb	w13, [x11], #1
	SUBS $1, R8, R8  // <--                                  // subs	x8, x8, #1
	MULW R9, R13, R9 // <--                                  // mul	w9, w13, w9
	WORD $0x38001549 // MOVB.P R9, 1(R10)                    // strb	w9, [x10], #1
	BNE  LBB0_4      // <--                                  // b.ne	.LBB0_4

LBB0_5:
	NOP // (skipped)                            // ldp	x29, x30, [sp], #16

LBB0_6:
	RET // <--                                  // ret

LBB0_7:
	MOVD ZR, R9      // <--                                  // mov	x9, xzr
	SUB  R0, R2, R10 // <--                                  // sub	x10, x2, x0
	CMP  $32, R10    // <--                                  // cmp	x10, #32
	BCC  LBB0_3      // <--                                  // b.lo	.LBB0_3
	SUB  R1, R2, R10 // <--                                  // sub	x10, x2, x1
	CMP  $32, R10    // <--                                  // cmp	x10, #32
	BCC  LBB0_3      // <--                                  // b.lo	.LBB0_3
	CMP  $32, R8     // <--                                  // cmp	x8, #32
	BCS  LBB0_11     // <--                                  // b.hs	.LBB0_11
	MOVD ZR, R9      // <--                                  // mov	x9, xzr
	JMP  LBB0_15     // <--                                  // b	.LBB0_15

LBB0_11:
	AND  $31, R3, R10 // <--                                  // and	x10, x3, #0x1f
	ADD  $16, R0, R11 // <--                                  // add	x11, x0, #16
	SUB  R10, R8, R9  // <--                                  // sub	x9, x8, x10
	ADD  $16, R1, R12 // <--                                  // add	x12, x1, #16
	ADD  $16, R2, R13 // <--                                  // add	x13, x2, #16
	MOVD R9, R14      // <--                                  // mov	x14, x9

LBB0_12:
	WORD $0xad7f8560   // FLDPQ -16(R11), (F0, F1)             // ldp	q0, q1, [x11, #-16]
	ADD  $32, R11, R11 // <--                                  // add	x11, x11, #32
	SUBS $32, R14, R14 // <--                                  // subs	x14, x14, #32
	WORD $0xad7f8d82   // FLDPQ -16(R12), (F2, F3)             // ldp	q2, q3, [x12, #-16]
	ADD  $32, R12, R12 // <--                                  // add	x12, x12, #32
	WORD $0x4e209c40   // VMUL V0.B16, V2.B16, V0.B16          // mul	v0.16b, v2.16b, v0.16b
	WORD $0x4e219c61   // VMUL V1.B16, V3.B16, V1.B16          // mul	v1.16b, v3.16b, v1.16b
	WORD $0xad3f85a0   // FSTPQ (F0, F1), -16(R13)             // stp	q0, q1, [x13, #-16]
	ADD  $32, R13, R13 // <--                                  // add	x13, x13, #32
	BNE  LBB0_12       // <--                                  // b.ne	.LBB0_12
	CBZ  R10, LBB0_5   // <--                                  // cbz	x10, .LBB0_5
	CMP  $8, R10       // <--                                  // cmp	x10, #8
	BCC  LBB0_3        // <--                                  // b.lo	.LBB0_3

LBB0_15:
	AND $7, R3, R10  // <--                                  // and	x10, x3, #0x7
	ADD R9, R0, R11  // <--                                  // add	x11, x0, x9
	ADD R10, R9, R14 // <--                                  // add	x14, x9, x10
	ADD R9, R1, R12  // <--                                  // add	x12, x1, x9
	ADD R9, R2, R13  // <--                                  // add	x13, x2, x9
	SUB R10, R8, R9  // <--                                  // sub	x9, x8, x10
	SUB R8, R14, R14 // <--                                  // sub	x14, x14, x8

LBB0_16:
	WORD $0xfc408560  // FMOVD.P 8(R11), F0                   // ldr	d0, [x11], #8
	WORD $0xfc408581  // FMOVD.P 8(R12), F1                   // ldr	d1, [x12], #8
	ADDS $8, R14, R14 // <--                                  // adds	x14, x14, #8
	WORD $0x0e209c20  // VMUL V0.B8, V1.B8, V0.B8             // mul	v0.8b, v1.8b, v0.8b
	WORD $0xfc0085a0  // FMOVD.P F0, 8(R13)                   // str	d0, [x13], #8
	BNE  LBB0_16      // <--                                  // b.ne	.LBB0_16
	CBNZ R10, LBB0_3  // <--                                  // cbnz	x10, .LBB0_3
	JMP  LBB0_5       // <--                                  // b	.LBB0_5
