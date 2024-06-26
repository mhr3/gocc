//go:build !noasm && arm64
// Code generated by gocc devel -- DO NOT EDIT.
//
// Source file         : test_simd_mul.c
// Clang version       : Apple clang version 15.0.0 (clang-1500.3.9.4)
// Target architecture : arm64
// Compiler options    : -march=armv8.5-a+sve2 -mfpu=sve2

#include "textflag.h"

TEXT ·uint8_simd_mul_sve2(SB), NOSPLIT, $0-32
	MOVD input1+0(FP), R0
	MOVD input2+8(FP), R1
	MOVD output+16(FP), R2
	MOVD size+24(FP), R3
	NOP                      // (skipped)                            // stp	x29, x30, [sp, #-16]!
	CMPW $1, R3              // <--                                  // cmp	w3, #1
	NOP                      // (skipped)                            // mov	x29, sp
	BLT  LBB0_5              // <--                                  // b.lt	.LBB0_5
	AND  $4294967295, R3, R8 // <--                                  // and	x8, x3, #0xffffffff
	WORD $0x0460e3ea         // ?                                    // cnth	x10
	CMP  R10, R8             // <--                                  // cmp	x8, x10
	BCS  LBB0_6              // <--                                  // b.hs	.LBB0_6
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
	RET // <--                                  // ret

LBB0_6:
	WORD $0x04bf502b  // ?                                    // rdvl	x11, #1
	MOVD ZR, R9       // <--                                  // mov	x9, xzr
	LSR  $4, R11, R13 // <--                                  // lsr	x13, x11, #4
	SUB  R0, R2, R12  // <--                                  // sub	x12, x2, x0
	LSL  $5, R13, R11 // <--                                  // lsl	x11, x13, #5
	CMP  R11, R12     // <--                                  // cmp	x12, x11
	BCC  LBB0_3       // <--                                  // b.lo	.LBB0_3
	SUB  R1, R2, R12  // <--                                  // sub	x12, x2, x1
	CMP  R11, R12     // <--                                  // cmp	x12, x11
	BCC  LBB0_3       // <--                                  // b.lo	.LBB0_3
	CMP  R11, R8      // <--                                  // cmp	x8, x11
	BCS  LBB0_10      // <--                                  // b.hs	.LBB0_10
	MOVD ZR, R9       // <--                                  // mov	x9, xzr
	JMP  LBB0_14      // <--                                  // b	.LBB0_14

LBB0_10:
	UDIV R11, R8, R9  // <--                                  // udiv	x9, x8, x11
	LSL  $4, R13, R16 // <--                                  // lsl	x16, x13, #4
	MOVD ZR, R12      // <--                                  // mov	x12, xzr
	ADD  R16, R0, R14 // <--                                  // add	x14, x0, x16
	ADD  R16, R1, R15 // <--                                  // add	x15, x1, x16
	ADD  R16, R2, R16 // <--                                  // add	x16, x2, x16
	WORD $0x2518e3e0  // ?                                    // ptrue	p0.b
	MUL  R11, R9, R9  // <--                                  // mul	x9, x9, x11
	SUB  R9, R8, R13  // <--                                  // sub	x13, x8, x9

LBB0_11:
	WORD $0xa40c4000   // ?                                    // ld1b	{ z0.b }, p0/z, [x0, x12]
	WORD $0xa40c4021   // ?                                    // ld1b	{ z1.b }, p0/z, [x1, x12]
	WORD $0xa40c41c2   // ?                                    // ld1b	{ z2.b }, p0/z, [x14, x12]
	WORD $0xa40c41e3   // ?                                    // ld1b	{ z3.b }, p0/z, [x15, x12]
	WORD $0x04206020   // ?                                    // mul	z0.b, z1.b, z0.b
	WORD $0x04226061   // ?                                    // mul	z1.b, z3.b, z2.b
	WORD $0xe40c4040   // ?                                    // st1b	{ z0.b }, p0, [x2, x12]
	WORD $0xe40c4201   // ?                                    // st1b	{ z1.b }, p0, [x16, x12]
	ADD  R11, R12, R12 // <--                                  // add	x12, x12, x11
	CMP  R12, R9       // <--                                  // cmp	x9, x12
	BNE  LBB0_11       // <--                                  // b.ne	.LBB0_11
	CBZ  R13, LBB0_5   // <--                                  // cbz	x13, .LBB0_5
	CMP  R10, R13      // <--                                  // cmp	x13, x10
	BCC  LBB0_3        // <--                                  // b.lo	.LBB0_3

LBB0_14:
	UDIV R10, R8, R11 // <--                                  // udiv	x11, x8, x10
	MOVD R9, R12      // <--                                  // mov	x12, x9
	WORD $0x2558e3e0  // ?                                    // ptrue	p0.h
	MUL  R10, R11, R9 // <--                                  // mul	x9, x11, x10
	SUB  R9, R8, R11  // <--                                  // sub	x11, x8, x9

LBB0_15:
	WORD $0xa42c4000   // ?                                    // ld1b	{ z0.h }, p0/z, [x0, x12]
	WORD $0xa42c4021   // ?                                    // ld1b	{ z1.h }, p0/z, [x1, x12]
	WORD $0x04606020   // ?                                    // mul	z0.h, z1.h, z0.h
	WORD $0xe42c4040   // ?                                    // st1b	{ z0.h }, p0, [x2, x12]
	ADD  R10, R12, R12 // <--                                  // add	x12, x12, x10
	CMP  R12, R9       // <--                                  // cmp	x9, x12
	BNE  LBB0_15       // <--                                  // b.ne	.LBB0_15
	CBNZ R11, LBB0_3   // <--                                  // cbnz	x11, .LBB0_3
	JMP  LBB0_5        // <--                                  // b	.LBB0_5
