//go:build !noasm && arm64
// Code generated by gocc -- DO NOT EDIT.

#include "textflag.h"

TEXT ·f32_axpy(SB), NOSPLIT, $0-28
	MOVD  x+0(FP), R0
	MOVD  y+8(FP), R1
	MOVD  size+16(FP), R2
	FMOVS alpha+24(FP), F0
	NOP                    // (skipped)                            // stp	x29, x30, [sp, #-16]!
	CMP   $4, R2           // <--                                  // cmp	x2, #4
	NOP                    // (skipped)                            // mov	x29, sp
	BCC   LBB0_3           // <--                                  // b.lo	.LBB0_3
	MOVW  $3, R8           // <--                                  // mov	w8, #3
	MOVD  R0, R9           // <--                                  // mov	x9, x0
	MOVD  R1, R10          // <--                                  // mov	x10, x1

LBB0_2:
	WORD $0x3dc00141 // FMOVQ (R10), F1                      // ldr	q1, [x10]
	ADD  $4, R8, R8  // <--                                  // add	x8, x8, #4
	WORD $0x3cc10522 // FMOVQ.P 16(R9), F2                   // ldr	q2, [x9], #16
	CMP  R2, R8      // <--                                  // cmp	x8, x2
	WORD $0x4f801041 // VFMLA V0.S[0], V2.S4, V1.S4          // fmla	v1.4s, v2.4s, v0.s[0]
	WORD $0x3c810541 // FMOVQ.P F1, 16(R10)                  // str	q1, [x10], #16
	BCC  LBB0_2      // <--                                  // b.lo	.LBB0_2

LBB0_3:
	TST $3, R2        // <--                                  // tst	x2, #0x3
	BEQ LBB0_7        // <--                                  // b.eq	.LBB0_7
	AND $-4, R2, R8   // <--                                  // and	x8, x2, #0xfffffffffffffffc
	CMP R2, R8        // <--                                  // cmp	x8, x2
	BCS LBB0_7        // <--                                  // b.hs	.LBB0_7
	LSL $2, R2, R9    // <--                                  // lsl	x9, x2, #2
	SUB R8, R2, R8    // <--                                  // sub	x8, x2, x8
	AND $-16, R9, R10 // <--                                  // and	x10, x9, #0xfffffffffffffff0
	ADD R10, R1, R9   // <--                                  // add	x9, x1, x10
	ADD R10, R0, R10  // <--                                  // add	x10, x0, x10

LBB0_6:
	WORD   $0xbc404541    // FMOVS.P 4(R10), F1                   // ldr	s1, [x10], #4
	WORD   $0xbd400122    // FMOVS (R9), F2                       // ldr	s2, [x9]
	SUBS   $1, R8, R8     // <--                                  // subs	x8, x8, #1
	FMADDS F0, F2, F1, F1 // <--                                  // fmadd	s1, s1, s0, s2
	WORD   $0xbc004521    // FMOVS.P F1, 4(R9)                    // str	s1, [x9], #4
	BNE    LBB0_6         // <--                                  // b.ne	.LBB0_6

LBB0_7:
	NOP // (skipped)                            // ldp	x29, x30, [sp], #16
	RET // <--                                  // ret

TEXT ·f32_matmul(SB), NOSPLIT, $0-32
	MOVD  dst+0(FP), R0
	MOVD  m+8(FP), R1
	MOVD  n+16(FP), R2
	MOVD  dims+24(FP), R3
	NOP                        // (skipped)                            // stp	x29, x30, [sp, #-16]!
	ANDS  $65535, R3, R8       // <--                                  // ands	x8, x3, #0xffff
	NOP                        // (skipped)                            // mov	x29, sp
	BEQ   LBB1_25              // <--                                  // b.eq	.LBB1_25
	LSR   $48, R3, R10         // <--                                  // lsr	x10, x3, #48
	TST   $844424930131968, R3 // <--                                  // tst	x3, #0x3000000000000
	AND   $65532, R10, R11     // <--                                  // and	x11, x10, #0xfffc
	UBFX  $16, R3, $16, R9     // <--                                  // ubfx	x9, x3, #16, #16
	CCMP  NE, R10, R11, $0     // <--                                  // ccmp	x10, x11, #0, ne
	CSETW HI, R12              // <--                                  // cset	w12, hi
	CBZ   R9, LBB1_25          // <--                                  // cbz	x9, .LBB1_25
	LSR   $50, R3, R13         // <--                                  // lsr	x13, x3, #50
	CBZ   R13, LBB1_12         // <--                                  // cbz	x13, .LBB1_12
	TBZ   $0, R12, LBB1_19     // <--                                  // tbz	w12, #0, .LBB1_19
	MOVD  ZR, R12              // <--                                  // mov	x12, xzr
	LSL   $2, R10, R13         // <--                                  // lsl	x13, x10, #2

LBB1_5:
	MUL  R9, R12, R15 // <--                                  // mul	x15, x12, x9
	MOVD ZR, R14      // <--                                  // mov	x14, xzr
	MOVD R2, R16      // <--                                  // mov	x16, x2

LBB1_6:
	ADD  R15, R14, R17 // <--                                  // add	x17, x14, x15
	MOVD R16, R3       // <--                                  // mov	x3, x16
	MOVW $3, R4        // <--                                  // mov	w4, #3
	WORD $0xbc717820   // FMOVS (R1)(R17<<2), F0               // ldr	s0, [x1, x17, lsl #2]
	MOVD R0, R17       // <--                                  // mov	x17, x0

LBB1_7:
	WORD $0x3dc00221 // FMOVQ (R17), F1                      // ldr	q1, [x17]
	ADD  $4, R4, R4  // <--                                  // add	x4, x4, #4
	WORD $0x3cc10462 // FMOVQ.P 16(R3), F2                   // ldr	q2, [x3], #16
	CMP  R10, R4     // <--                                  // cmp	x4, x10
	WORD $0x4f801041 // VFMLA V0.S[0], V2.S4, V1.S4          // fmla	v1.4s, v2.4s, v0.s[0]
	WORD $0x3c810621 // FMOVQ.P F1, 16(R17)                  // str	q1, [x17], #16
	BCC  LBB1_7      // <--                                  // b.lo	.LBB1_7
	MOVD R11, R17    // <--                                  // mov	x17, x11

LBB1_9:
	LSL    $2, R17, R3    // <--                                  // lsl	x3, x17, #2
	ADD    $1, R17, R17   // <--                                  // add	x17, x17, #1
	CMP    R17, R10       // <--                                  // cmp	x10, x17
	WORD   $0xbc636a01    // FMOVS (R16)(R3), F1                  // ldr	s1, [x16, x3]
	WORD   $0xbc636802    // FMOVS (R0)(R3), F2                   // ldr	s2, [x0, x3]
	FMADDS F0, F2, F1, F1 // <--                                  // fmadd	s1, s1, s0, s2
	WORD   $0xbc236801    // FMOVS F1, (R0)(R3)                   // str	s1, [x0, x3]
	BNE    LBB1_9         // <--                                  // b.ne	.LBB1_9
	ADD    $1, R14, R14   // <--                                  // add	x14, x14, #1
	ADD    R13, R16, R16  // <--                                  // add	x16, x16, x13
	CMP    R9, R14        // <--                                  // cmp	x14, x9
	BNE    LBB1_6         // <--                                  // b.ne	.LBB1_6
	ADD    $1, R12, R12   // <--                                  // add	x12, x12, #1
	ADD    R13, R0, R0    // <--                                  // add	x0, x0, x13
	CMP    R8, R12        // <--                                  // cmp	x12, x8
	BNE    LBB1_5         // <--                                  // b.ne	.LBB1_5
	JMP    LBB1_25        // <--                                  // b	.LBB1_25

LBB1_12:
	CBZW R12, LBB1_25      // <--                                  // cbz	w12, .LBB1_25
	LSR  $46, R3, R13      // <--                                  // lsr	x13, x3, #46
	MOVD ZR, R12           // <--                                  // mov	x12, xzr
	AND  $262128, R13, R14 // <--                                  // and	x14, x13, #0x3fff0
	SUB  R11, R10, R11     // <--                                  // sub	x11, x10, x11
	ADD  R14, R0, R13      // <--                                  // add	x13, x0, x14
	LSL  $2, R10, R10      // <--                                  // lsl	x10, x10, #2
	ADD  R14, R2, R14      // <--                                  // add	x14, x2, x14

LBB1_14:
	MUL  R9, R12, R16 // <--                                  // mul	x16, x12, x9
	MOVD ZR, R15      // <--                                  // mov	x15, xzr
	MOVD R14, R17     // <--                                  // mov	x17, x14

LBB1_15:
	ADD  R16, R15, R0 // <--                                  // add	x0, x15, x16
	MOVD R13, R2      // <--                                  // mov	x2, x13
	MOVD R11, R3      // <--                                  // mov	x3, x11
	WORD $0xbc607820  // FMOVS (R1)(R0<<2), F0                // ldr	s0, [x1, x0, lsl #2]
	MOVD R17, R0      // <--                                  // mov	x0, x17

LBB1_16:
	WORD   $0xbc404401    // FMOVS.P 4(R0), F1                    // ldr	s1, [x0], #4
	WORD   $0xbd400042    // FMOVS (R2), F2                       // ldr	s2, [x2]
	SUBS   $1, R3, R3     // <--                                  // subs	x3, x3, #1
	FMADDS F0, F2, F1, F1 // <--                                  // fmadd	s1, s1, s0, s2
	WORD   $0xbc004441    // FMOVS.P F1, 4(R2)                    // str	s1, [x2], #4
	BNE    LBB1_16        // <--                                  // b.ne	.LBB1_16
	ADD    $1, R15, R15   // <--                                  // add	x15, x15, #1
	ADD    R10, R17, R17  // <--                                  // add	x17, x17, x10
	CMP    R9, R15        // <--                                  // cmp	x15, x9
	BNE    LBB1_15        // <--                                  // b.ne	.LBB1_15
	ADD    $1, R12, R12   // <--                                  // add	x12, x12, #1
	ADD    R10, R13, R13  // <--                                  // add	x13, x13, x10
	CMP    R8, R12        // <--                                  // cmp	x12, x8
	BNE    LBB1_14        // <--                                  // b.ne	.LBB1_14
	JMP    LBB1_25        // <--                                  // b	.LBB1_25

LBB1_19:
	MOVD ZR, R11      // <--                                  // mov	x11, xzr
	LSL  $2, R10, R12 // <--                                  // lsl	x12, x10, #2

LBB1_20:
	MUL  R9, R11, R14 // <--                                  // mul	x14, x11, x9
	MOVD ZR, R13      // <--                                  // mov	x13, xzr
	MOVD R2, R15      // <--                                  // mov	x15, x2

LBB1_21:
	ADD  R14, R13, R16   // <--                                  // add	x16, x13, x14
	MOVD R15, R17        // <--                                  // mov	x17, x15
	MOVW $3, R3          // <--                                  // mov	w3, #3
	ADD  R16<<2, R1, R16 // <--                                  // add	x16, x1, x16, lsl #2
	WORD $0x4d40ca00     // VLD1R (R16), [V0.S4]                 // ld1r	{ v0.4s }, [x16]
	MOVD R0, R16         // <--                                  // mov	x16, x0

LBB1_22:
	WORD $0x3dc00201   // FMOVQ (R16), F1                      // ldr	q1, [x16]
	ADD  $4, R3, R3    // <--                                  // add	x3, x3, #4
	WORD $0x3cc10622   // FMOVQ.P 16(R17), F2                  // ldr	q2, [x17], #16
	CMP  R10, R3       // <--                                  // cmp	x3, x10
	WORD $0x4e20cc41   // VFMLA V0.S4, V2.S4, V1.S4            // fmla	v1.4s, v2.4s, v0.4s
	WORD $0x3c810601   // FMOVQ.P F1, 16(R16)                  // str	q1, [x16], #16
	BCC  LBB1_22       // <--                                  // b.lo	.LBB1_22
	ADD  $1, R13, R13  // <--                                  // add	x13, x13, #1
	ADD  R12, R15, R15 // <--                                  // add	x15, x15, x12
	CMP  R9, R13       // <--                                  // cmp	x13, x9
	BNE  LBB1_21       // <--                                  // b.ne	.LBB1_21
	ADD  $1, R11, R11  // <--                                  // add	x11, x11, #1
	ADD  R12, R0, R0   // <--                                  // add	x0, x0, x12
	CMP  R8, R11       // <--                                  // cmp	x11, x8
	BNE  LBB1_20       // <--                                  // b.ne	.LBB1_20

LBB1_25:
	NOP // (skipped)                            // ldp	x29, x30, [sp], #16
	RET // <--                                  // ret
