//go:build !noasm && arm64
// Code generated by gocc -- DO NOT EDIT.

#include "textflag.h"

TEXT ·memcmp_sve(SB), NOSPLIT, $0-56
	MOVD x+0(FP), R0
	MOVD x_len+8(FP), R1
	MOVD x_cap+16(FP), R2
	MOVD y+24(FP), R3
	MOVD y_len+32(FP), R4
	MOVD y_cap+40(FP), R5
	NOP                   // (skipped)                            // stp	x29, x30, [sp, #-16]!
	CMP  R4, R1           // <--                                  // cmp	x1, x4
	NOP                   // (skipped)                            // mov	x29, sp
	BNE  LBB0_7           // <--                                  // b.ne	.LBB0_7
	CBZ  R1, LBB0_5       // <--                                  // cbz	x1, .LBB0_5
	MOVD ZR, R8           // <--                                  // mov	x8, xzr

LBB0_3:
	WORD $0x25211d00 // ?                                    // whilelo	p0.b, x8, x1
	WORD $0xa4084001 // ?                                    // ld1b	{ z1.b }, p0/z, [x0, x8]
	WORD $0xa4084060 // ?                                    // ld1b	{ z0.b }, p0/z, [x3, x8]
	WORD $0x2400a031 // ?                                    // cmpne	p1.b, p0/z, z1.b, z0.b
	BNE  LBB0_6      // <--                                  // b.ne	.LBB0_6
	WORD $0x04285028 // ?                                    // addvl	x8, x8, #1
	CMP  R1, R8      // <--                                  // cmp	x8, x1
	BCC  LBB0_3      // <--                                  // b.lo	.LBB0_3

LBB0_5:
	MOVD ZR, R0         // <--                                  // mov	x0, xzr
	NOP                 // (skipped)                            // ldp	x29, x30, [sp], #16
	MOVD R0, ret+48(FP) // <--
	RET                 // <--                                  // ret

LBB0_6:
	WORD $0x25904020  // ?                                    // brkb	p0.b, p0/z, p1.b
	WORD $0x0520a028  // ?                                    // lasta	w8, p0, z1.b
	WORD $0x0520a009  // ?                                    // lasta	w9, p0, z0.b
	ANDW $255, R8, R8 // <--                                  // and	w8, w8, #0xff
	CMPW R9.UXTB, R8  // <--                                  // cmp	w8, w9, uxtb

LBB0_7:
	MOVD $-1, R8        // <--                                  // mov	x8, #-1
	CNEG HS, R8, R0     // <--                                  // cneg	x0, x8, hs
	NOP                 // (skipped)                            // ldp	x29, x30, [sp], #16
	MOVD R0, ret+48(FP) // <--
	RET                 // <--                                  // ret
