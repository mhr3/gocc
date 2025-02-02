//go:build !noasm && arm64
// Code generated by gocc devel -- DO NOT EDIT.
//
// Source file         : test_simd_mul.c
// Clang version       : Apple clang version 16.0.0 (clang-1600.0.26.4)
// Target architecture : arm64
// Compiler options    : -march=armv8.2-a+sve

#include "textflag.h"

TEXT ·uint8_simd_mul_sve(SB), NOSPLIT, $0-32
	MOVD input1+0(FP), R0
	MOVD input2+8(FP), R1
	MOVD output+16(FP), R2
	MOVD size+24(FP), R3
	CMPW $1, R3              // <--                                  // cmp	w3, #1
	BLT  LBB0_17             // <--                                  // b.lt	.LBB0_17
	NOP                      // (skipped)                            // stp	x29, x30, [sp, #-16]!
	AND  $4294967295, R3, R8 // <--                                  // and	x8, x3, #0xffffffff
	WORD $0x0460e3ea         // ?                                    // cnth	x10
	CMP  R10, R8             // <--                                  // cmp	x8, x10
	NOP                      // (skipped)                            // mov	x29, sp
	BCS  LBB0_3              // <--                                  // b.hs	.LBB0_3
	MOVD ZR, R9              // <--                                  // mov	x9, xzr
	JMP  LBB0_14             // <--                                  // b	.LBB0_14

LBB0_3:
	WORD $0x04bf502b  // ?                                    // rdvl	x11, #1
	MOVD ZR, R9       // <--                                  // mov	x9, xzr
	LSR  $4, R11, R11 // <--                                  // lsr	x11, x11, #4
	SUB  R0, R2, R13  // <--                                  // sub	x13, x2, x0
	LSL  $5, R11, R12 // <--                                  // lsl	x12, x11, #5
	CMP  R12, R13     // <--                                  // cmp	x13, x12
	BCC  LBB0_14      // <--                                  // b.lo	.LBB0_14
	SUB  R1, R2, R13  // <--                                  // sub	x13, x2, x1
	CMP  R12, R13     // <--                                  // cmp	x13, x12
	BCC  LBB0_14      // <--                                  // b.lo	.LBB0_14
	CMP  R12, R8      // <--                                  // cmp	x8, x12
	BCS  LBB0_10      // <--                                  // b.hs	.LBB0_10
	MOVD ZR, R9       // <--                                  // mov	x9, xzr

LBB0_7:
	NEG  R11<<3, R12 // <--                                  // neg	x12, x11, lsl #3
	MOVD R9, R11     // <--                                  // mov	x11, x9
	AND  R12, R8, R9 // <--                                  // and	x9, x8, x12
	WORD $0x2558e3e0 // ?                                    // ptrue	p0.h

LBB0_8:
	WORD $0xa42b4000   // ?                                    // ld1b	{ z0.h }, p0/z, [x0, x11]
	WORD $0xa42b4021   // ?                                    // ld1b	{ z1.h }, p0/z, [x1, x11]
	WORD $0x04500020   // ?                                    // mul	z0.h, p0/m, z0.h, z1.h
	WORD $0xe42b4040   // ?                                    // st1b	{ z0.h }, p0, [x2, x11]
	ADD  R10, R11, R11 // <--                                  // add	x11, x11, x10
	CMP  R11, R9       // <--                                  // cmp	x9, x11
	BNE  LBB0_8        // <--                                  // b.ne	.LBB0_8
	CMP  R9, R8        // <--                                  // cmp	x8, x9
	BNE  LBB0_14       // <--                                  // b.ne	.LBB0_14
	JMP  LBB0_16       // <--                                  // b	.LBB0_16

LBB0_10:
	NEG  R11<<5, R9   // <--                                  // neg	x9, x11, lsl #5
	LSL  $4, R11, R16 // <--                                  // lsl	x16, x11, #4
	MOVD ZR, R13      // <--                                  // mov	x13, xzr
	AND  R9, R8, R9   // <--                                  // and	x9, x8, x9
	ADD  R16, R0, R14 // <--                                  // add	x14, x0, x16
	ADD  R16, R1, R15 // <--                                  // add	x15, x1, x16
	ADD  R16, R2, R16 // <--                                  // add	x16, x2, x16
	WORD $0x2518e3e0  // ?                                    // ptrue	p0.b

LBB0_11:
	WORD $0xa40d4000   // ?                                    // ld1b	{ z0.b }, p0/z, [x0, x13]
	WORD $0xa40d4021   // ?                                    // ld1b	{ z1.b }, p0/z, [x1, x13]
	WORD $0xa40d41c2   // ?                                    // ld1b	{ z2.b }, p0/z, [x14, x13]
	WORD $0xa40d41e3   // ?                                    // ld1b	{ z3.b }, p0/z, [x15, x13]
	WORD $0x04100020   // ?                                    // mul	z0.b, p0/m, z0.b, z1.b
	WORD $0x0420bc61   // ?                                    // movprfx	z1, z3
	WORD $0x04100041   // ?                                    // mul	z1.b, p0/m, z1.b, z2.b
	WORD $0xe40d4040   // ?                                    // st1b	{ z0.b }, p0, [x2, x13]
	WORD $0xe40d4201   // ?                                    // st1b	{ z1.b }, p0, [x16, x13]
	ADD  R12, R13, R13 // <--                                  // add	x13, x13, x12
	CMP  R13, R9       // <--                                  // cmp	x9, x13
	BNE  LBB0_11       // <--                                  // b.ne	.LBB0_11
	SUBS R9, R8, R12   // <--                                  // subs	x12, x8, x9
	BEQ  LBB0_16       // <--                                  // b.eq	.LBB0_16
	CMP  R10, R12      // <--                                  // cmp	x12, x10
	BCS  LBB0_7        // <--                                  // b.hs	.LBB0_7

LBB0_14:
	ADD R9, R2, R10 // <--                                  // add	x10, x2, x9
	ADD R9, R1, R11 // <--                                  // add	x11, x1, x9
	ADD R9, R0, R12 // <--                                  // add	x12, x0, x9
	SUB R9, R8, R8  // <--                                  // sub	x8, x8, x9

LBB0_15:
	WORD $0x38401589 // MOVBU.P 1(R12), R9                   // ldrb	w9, [x12], #1
	WORD $0x3840156d // MOVBU.P 1(R11), R13                  // ldrb	w13, [x11], #1
	SUBS $1, R8, R8  // <--                                  // subs	x8, x8, #1
	MULW R9, R13, R9 // <--                                  // mul	w9, w13, w9
	WORD $0x38001549 // MOVB.P R9, 1(R10)                    // strb	w9, [x10], #1
	BNE  LBB0_15     // <--                                  // b.ne	.LBB0_15

LBB0_16:
	NOP // (skipped)                            // ldp	x29, x30, [sp], #16

LBB0_17:
	RET // <--                                  // ret
