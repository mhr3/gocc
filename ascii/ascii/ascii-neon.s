//go:build !noasm && arm64
// Code generated by gocc devel -- DO NOT EDIT.
//
// Source file         : ascii-neon.c
// Clang version       : Apple clang version 16.0.0 (clang-1600.0.26.4)
// Target architecture : arm64
// Compiler options    : [none]

#include "textflag.h"

TEXT ·IsASCII(SB), NOSPLIT, $0-17
	MOVD data+0(FP), R0
	MOVD length+8(FP), R1
	NOP                   // (skipped)                            // stp	x29, x30, [sp, #-16]!
	CMP  $16, R1          // <--                                  // cmp	x1, #16
	NOP                   // (skipped)                            // mov	x29, sp
	BCC  LBB0_7           // <--                                  // b.lo	.LBB0_7
	ADD  R1, R0, R9       // <--                                  // add	x9, x0, x1
	AND  $63, R1, R8      // <--                                  // and	x8, x1, #0x3f
	SUB  R8, R9, R9       // <--                                  // sub	x9, x9, x8
	CMP  R0, R9           // <--                                  // cmp	x9, x0
	BLS  LBB0_4           // <--                                  // b.ls	.LBB0_4

LBB0_2:
	WORD  $0x4c402000            // VLD1 (R0), [V0.B16, V1.B16, V2.B16, V3.B16] // ld1	{ v0.16b, v1.16b, v2.16b, v3.16b }, [x0]
	VORR  V1.B16, V0.B16, V4.B16 // <--                                  // orr	v4.16b, v0.16b, v1.16b
	VORR  V2.B16, V3.B16, V0.B16 // <--                                  // orr	v0.16b, v3.16b, v2.16b
	VORR  V0.B16, V4.B16, V0.B16 // <--                                  // orr	v0.16b, v4.16b, v0.16b
	WORD  $0x4e20a800            // VCMLT $0, V0.B16, V0.B16             // cmlt	v0.16b, v0.16b, #0
	WORD  $0x6eb0a800            // VUMAXV V0.S4, V0                     // umaxv	s0, v0.4s
	FMOVS F0, R10                // <--                                  // fmov	w10, s0
	CBNZW R10, LBB0_12           // <--                                  // cbnz	w10, .LBB0_12
	ADD   $64, R0, R0            // <--                                  // add	x0, x0, #64
	CMP   R9, R0                 // <--                                  // cmp	x0, x9
	BCC   LBB0_2                 // <--                                  // b.lo	.LBB0_2

LBB0_4:
	ADD R8, R0, R8  // <--                                  // add	x8, x0, x8
	AND $15, R1, R1 // <--                                  // and	x1, x1, #0xf
	SUB R1, R8, R8  // <--                                  // sub	x8, x8, x1
	CMP R8, R0      // <--                                  // cmp	x0, x8
	BCS LBB0_7      // <--                                  // b.hs	.LBB0_7

LBB0_5:
	WORD  $0x3dc00000 // FMOVQ (R0), F0                       // ldr	q0, [x0]
	WORD  $0x4e20a800 // VCMLT $0, V0.B16, V0.B16             // cmlt	v0.16b, v0.16b, #0
	WORD  $0x6eb0a800 // VUMAXV V0.S4, V0                     // umaxv	s0, v0.4s
	FMOVS F0, R9      // <--                                  // fmov	w9, s0
	CBNZW R9, LBB0_12 // <--                                  // cbnz	w9, .LBB0_12
	ADD   $16, R0, R0 // <--                                  // add	x0, x0, #16
	CMP   R8, R0      // <--                                  // cmp	x0, x8
	BCC   LBB0_5      // <--                                  // b.lo	.LBB0_5

LBB0_7:
	SUBS $8, R1, R8                    // <--                                  // subs	x8, x1, #8
	BCC  LBB0_10                       // <--                                  // b.lo	.LBB0_10
	WORD $0xf8408409                   // MOVD.P 8(R0), R9                     // ldr	x9, [x0], #8
	AND  $-9187201950435737472, R9, R9 // <--                                  // and	x9, x9, #0x8080808080808080
	CBNZ R9, LBB0_12                   // <--                                  // cbnz	x9, .LBB0_12
	MOVD R8, R1                        // <--                                  // mov	x1, x8

LBB0_10:
	MOVW $2155905152, R8 // <--                                  // mov	w8, #-2139062144
	SUBS $4, R1, R9      // <--                                  // subs	x9, x1, #4
	BCC  LBB0_14         // <--                                  // b.lo	.LBB0_14
	WORD $0xb940000a     // MOVWU (R0), R10                      // ldr	w10, [x0]
	TSTW R8, R10         // <--                                  // tst	w10, w8
	BEQ  LBB0_13         // <--                                  // b.eq	.LBB0_13

LBB0_12:
	MOVW ZR, R0         // <--                                  // mov	w0, wzr
	NOP                 // (skipped)                            // ldp	x29, x30, [sp], #16
	MOVB R0, ret+16(FP) // <--
	RET                 // <--                                  // ret

LBB0_13:
	ADD  $4, R0, R0 // <--                                  // add	x0, x0, #4
	MOVD R9, R1     // <--                                  // mov	x1, x9

LBB0_14:
	CMP   $1, R1          // <--                                  // cmp	x1, #1
	BEQ   LBB0_18         // <--                                  // b.eq	.LBB0_18
	CMP   $2, R1          // <--                                  // cmp	x1, #2
	BEQ   LBB0_19         // <--                                  // b.eq	.LBB0_19
	CMP   $3, R1          // <--                                  // cmp	x1, #3
	BNE   LBB0_20         // <--                                  // b.ne	.LBB0_20
	WORD  $0x79400009     // MOVHU (R0), R9                       // ldrh	w9, [x0]
	WORD  $0x3940080a     // MOVBU 2(R0), R10                     // ldrb	w10, [x0, #2]
	ORRW  R10<<16, R9, R9 // <--                                  // orr	w9, w9, w10, lsl #16
	TSTW  R8, R9          // <--                                  // tst	w9, w8
	CSETW EQ, R0          // <--                                  // cset	w0, eq
	NOP                   // (skipped)                            // ldp	x29, x30, [sp], #16
	MOVB  R0, ret+16(FP)  // <--
	RET                   // <--                                  // ret

LBB0_18:
	WORD  $0x39400009    // MOVBU (R0), R9                       // ldrb	w9, [x0]
	TSTW  R8, R9         // <--                                  // tst	w9, w8
	CSETW EQ, R0         // <--                                  // cset	w0, eq
	NOP                  // (skipped)                            // ldp	x29, x30, [sp], #16
	MOVB  R0, ret+16(FP) // <--
	RET                  // <--                                  // ret

LBB0_19:
	WORD  $0x79400009    // MOVHU (R0), R9                       // ldrh	w9, [x0]
	TSTW  R8, R9         // <--                                  // tst	w9, w8
	CSETW EQ, R0         // <--                                  // cset	w0, eq
	NOP                  // (skipped)                            // ldp	x29, x30, [sp], #16
	MOVB  R0, ret+16(FP) // <--
	RET                  // <--                                  // ret

LBB0_20:
	TSTW  R8, ZR         // <--                                  // tst	wzr, w8
	CSETW EQ, R0         // <--                                  // cset	w0, eq
	NOP                  // (skipped)                            // ldp	x29, x30, [sp], #16
	MOVB  R0, ret+16(FP) // <--
	RET                  // <--                                  // ret

TEXT ·IndexBit(SB), NOSPLIT, $0-32
	MOVD data+0(FP), R0
	MOVD length+8(FP), R1
	MOVB mask_bit+16(FP), R2
	NOP                      // (skipped)                            // stp	x29, x30, [sp, #-16]!
	CMP  $16, R1             // <--                                  // cmp	x1, #16
	NOP                      // (skipped)                            // mov	x29, sp
	BCC  LBB1_14             // <--                                  // b.lo	.LBB1_14
	ADD  R1, R0, R8          // <--                                  // add	x8, x0, x1
	AND  $63, R1, R10        // <--                                  // and	x10, x1, #0x3f
	SUB  R10, R8, R11        // <--                                  // sub	x11, x8, x10
	MOVD R0, R8              // <--                                  // mov	x8, x0
	VDUP R2, V0.B16          // <--                                  // dup	v0.16b, w2
	CMP  R0, R11             // <--                                  // cmp	x11, x0
	BLS  LBB1_18             // <--                                  // b.ls	.LBB1_18
	MOVW $16, R9             // <--                                  // mov	w9, #16
	MOVD R0, R8              // <--                                  // mov	x8, x0
	JMP  LBB1_4              // <--                                  // b	.LBB1_4

LBB1_3:
	ADD $64, R8, R8 // <--                                  // add	x8, x8, #64
	CMP R11, R8     // <--                                  // cmp	x8, x11
	BCS LBB1_18     // <--                                  // b.hs	.LBB1_18

LBB1_4:
	WORD  $0x4c402101            // VLD1 (R8), [V1.B16, V2.B16, V3.B16, V4.B16] // ld1	{ v1.16b, v2.16b, v3.16b, v4.16b }, [x8]
	VORR  V1.B16, V2.B16, V5.B16 // <--                                  // orr	v5.16b, v2.16b, v1.16b
	VORR  V4.B16, V3.B16, V6.B16 // <--                                  // orr	v6.16b, v3.16b, v4.16b
	VORR  V6.B16, V5.B16, V5.B16 // <--                                  // orr	v5.16b, v5.16b, v6.16b
	WORD  $0x4e208ca5            // VCMTST V0.B16, V5.B16, V5.B16        // cmtst	v5.16b, v5.16b, v0.16b
	WORD  $0x6eb0a8a5            // VUMAXV V5.S4, V5                     // umaxv	s5, v5.4s
	FMOVS F5, R12                // <--                                  // fmov	w12, s5
	CBZW  R12, LBB1_3            // <--                                  // cbz	w12, .LBB1_3
	WORD  $0x4e208c25            // VCMTST V0.B16, V1.B16, V5.B16        // cmtst	v5.16b, v1.16b, v0.16b
	FMOVD F5, R12                // <--                                  // fmov	x12, d5
	CBNZ  R12, LBB1_37           // <--                                  // cbnz	x12, .LBB1_37
	WORD  $0x4e183cac            // VMOV V5.D[1], R12                    // mov	x12, v5.d[1]
	CBNZ  R12, LBB1_38           // <--                                  // cbnz	x12, .LBB1_38
	WORD  $0x4e208c45            // VCMTST V0.B16, V2.B16, V5.B16        // cmtst	v5.16b, v2.16b, v0.16b
	FMOVD F5, R12                // <--                                  // fmov	x12, d5
	CBNZ  R12, LBB1_44           // <--                                  // cbnz	x12, .LBB1_44
	WORD  $0x4e183cac            // VMOV V5.D[1], R12                    // mov	x12, v5.d[1]
	CBNZ  R12, LBB1_42           // <--                                  // cbnz	x12, .LBB1_42
	WORD  $0x4e208c65            // VCMTST V0.B16, V3.B16, V5.B16        // cmtst	v5.16b, v3.16b, v0.16b
	FMOVD F5, R12                // <--                                  // fmov	x12, d5
	CBNZ  R12, LBB1_40           // <--                                  // cbnz	x12, .LBB1_40
	WORD  $0x4e183cac            // VMOV V5.D[1], R12                    // mov	x12, v5.d[1]
	CBNZ  R12, LBB1_41           // <--                                  // cbnz	x12, .LBB1_41
	WORD  $0x4e208c81            // VCMTST V0.B16, V4.B16, V1.B16        // cmtst	v1.16b, v4.16b, v0.16b
	FMOVD F1, R12                // <--                                  // fmov	x12, d1
	CBNZ  R12, LBB1_43           // <--                                  // cbnz	x12, .LBB1_43
	WORD  $0x4e183c2c            // VMOV V1.D[1], R12                    // mov	x12, v1.d[1]
	CBZ   R12, LBB1_3            // <--                                  // cbz	x12, .LBB1_3
	MOVW  $48, R9                // <--                                  // mov	w9, #48
	JMP   LBB1_42                // <--                                  // b	.LBB1_42

LBB1_14:
	MOVD R0, R8 // <--                                  // mov	x8, x0

LBB1_15:
	ANDW $255, R2, R9    // <--                                  // and	w9, w2, #0xff
	MOVW $16843009, R10  // <--                                  // mov	w10, #16843009
	MULW R10, R9, R9     // <--                                  // mul	w9, w9, w10
	SUBS $8, R1, R10     // <--                                  // subs	x10, x1, #8
	BCC  LBB1_22         // <--                                  // b.lo	.LBB1_22
	WORD $0xf940010b     // MOVD (R8), R11                       // ldr	x11, [x8]
	ORR  R9<<32, R9, R12 // <--                                  // orr	x12, x9, x9, lsl #32
	ANDS R12, R11, R11   // <--                                  // ands	x11, x11, x12
	BEQ  LBB1_21         // <--                                  // b.eq	.LBB1_21
	RBIT R11, R9         // <--                                  // rbit	x9, x11
	SUB  R0, R8, R8      // <--                                  // sub	x8, x8, x0
	CLZ  R9, R9          // <--                                  // clz	x9, x9
	ADD  R9>>3, R8, R0   // <--                                  // add	x0, x8, x9, lsr #3
	NOP                  // (skipped)                            // ldp	x29, x30, [sp], #16
	MOVD R0, ret+24(FP)  // <--
	RET                  // <--                                  // ret

LBB1_18:
	ADD R10, R8, R9 // <--                                  // add	x9, x8, x10
	AND $15, R1, R1 // <--                                  // and	x1, x1, #0xf
	SUB R1, R9, R9  // <--                                  // sub	x9, x9, x1
	CMP R9, R8      // <--                                  // cmp	x8, x9
	BCS LBB1_15     // <--                                  // b.hs	.LBB1_15

LBB1_19:
	WORD  $0x3dc00101  // FMOVQ (R8), F1                       // ldr	q1, [x8]
	WORD  $0x4e208c21  // VCMTST V0.B16, V1.B16, V1.B16        // cmtst	v1.16b, v1.16b, v0.16b
	WORD  $0x6eb0a822  // VUMAXV V1.S4, V2                     // umaxv	s2, v1.4s
	FMOVS F2, R10      // <--                                  // fmov	w10, s2
	CBNZW R10, LBB1_35 // <--                                  // cbnz	w10, .LBB1_35
	ADD   $16, R8, R8  // <--                                  // add	x8, x8, #16
	CMP   R9, R8       // <--                                  // cmp	x8, x9
	BCC   LBB1_19      // <--                                  // b.lo	.LBB1_19
	JMP   LBB1_15      // <--                                  // b	.LBB1_15

LBB1_21:
	ADD  $8, R8, R8 // <--                                  // add	x8, x8, #8
	MOVD R10, R1    // <--                                  // mov	x1, x10

LBB1_22:
	SUBS  $4, R1, R10    // <--                                  // subs	x10, x1, #4
	BCC   LBB1_26        // <--                                  // b.lo	.LBB1_26
	WORD  $0xb940010b    // MOVWU (R8), R11                      // ldr	w11, [x8]
	ANDSW R9, R11, R11   // <--                                  // ands	w11, w11, w9
	BEQ   LBB1_25        // <--                                  // b.eq	.LBB1_25
	RBITW R11, R9        // <--                                  // rbit	w9, w11
	CLZW  R9, R9         // <--                                  // clz	w9, w9
	SUB   R0, R8, R8     // <--                                  // sub	x8, x8, x0
	LSRW  $3, R9, R9     // <--                                  // lsr	w9, w9, #3
	ADD   R9, R8, R0     // <--                                  // add	x0, x8, x9
	NOP                  // (skipped)                            // ldp	x29, x30, [sp], #16
	MOVD  R0, ret+24(FP) // <--
	RET                  // <--                                  // ret

LBB1_25:
	ADD  $4, R8, R8 // <--                                  // add	x8, x8, #4
	MOVD R10, R1    // <--                                  // mov	x1, x10

LBB1_26:
	CMP   $1, R1            // <--                                  // cmp	x1, #1
	BEQ   LBB1_30           // <--                                  // b.eq	.LBB1_30
	CMP   $2, R1            // <--                                  // cmp	x1, #2
	BEQ   LBB1_31           // <--                                  // b.eq	.LBB1_31
	CMP   $3, R1            // <--                                  // cmp	x1, #3
	BNE   LBB1_33           // <--                                  // b.ne	.LBB1_33
	WORD  $0x7940010a       // MOVHU (R8), R10                      // ldrh	w10, [x8]
	WORD  $0x3940090b       // MOVBU 2(R8), R11                     // ldrb	w11, [x8, #2]
	ORRW  R11<<16, R10, R10 // <--                                  // orr	w10, w10, w11, lsl #16
	ANDSW R9, R10, R9       // <--                                  // ands	w9, w10, w9
	BNE   LBB1_32           // <--                                  // b.ne	.LBB1_32
	JMP   LBB1_34           // <--                                  // b	.LBB1_34

LBB1_30:
	WORD  $0x3940010a // MOVBU (R8), R10                      // ldrb	w10, [x8]
	ANDSW R9, R10, R9 // <--                                  // ands	w9, w10, w9
	BNE   LBB1_32     // <--                                  // b.ne	.LBB1_32
	JMP   LBB1_34     // <--                                  // b	.LBB1_34

LBB1_31:
	WORD  $0x7940010a // MOVHU (R8), R10                      // ldrh	w10, [x8]
	ANDSW R9, R10, R9 // <--                                  // ands	w9, w10, w9
	BEQ   LBB1_34     // <--                                  // b.eq	.LBB1_34

LBB1_32:
	RBITW R9, R9         // <--                                  // rbit	w9, w9
	CLZW  R9, R9         // <--                                  // clz	w9, w9
	SUB   R0, R8, R8     // <--                                  // sub	x8, x8, x0
	LSRW  $3, R9, R9     // <--                                  // lsr	w9, w9, #3
	ADD   R9, R8, R0     // <--                                  // add	x0, x8, x9
	NOP                  // (skipped)                            // ldp	x29, x30, [sp], #16
	MOVD  R0, ret+24(FP) // <--
	RET                  // <--                                  // ret

LBB1_33:
	MOVW  ZR, R10    // <--                                  // mov	w10, wzr
	ANDSW R9, ZR, R9 // <--                                  // ands	w9, wzr, w9
	BNE   LBB1_32    // <--                                  // b.ne	.LBB1_32

LBB1_34:
	MOVD $-1, R0        // <--                                  // mov	x0, #-1
	NOP                 // (skipped)                            // ldp	x29, x30, [sp], #16
	MOVD R0, ret+24(FP) // <--
	RET                 // <--                                  // ret

LBB1_35:
	FMOVD F1, R9         // <--                                  // fmov	x9, d1
	CBZ   R9, LBB1_39    // <--                                  // cbz	x9, .LBB1_39
	RBIT  R9, R9         // <--                                  // rbit	x9, x9
	SUB   R0, R8, R8     // <--                                  // sub	x8, x8, x0
	CLZ   R9, R9         // <--                                  // clz	x9, x9
	LSR   $3, R9, R9     // <--                                  // lsr	x9, x9, #3
	ADD   R9, R8, R0     // <--                                  // add	x0, x8, x9
	NOP                  // (skipped)                            // ldp	x29, x30, [sp], #16
	MOVD  R0, ret+24(FP) // <--
	RET                  // <--                                  // ret

LBB1_37:
	MOVD ZR, R9  // <--                                  // mov	x9, xzr
	JMP  LBB1_44 // <--                                  // b	.LBB1_44

LBB1_38:
	MOVD ZR, R9  // <--                                  // mov	x9, xzr
	JMP  LBB1_42 // <--                                  // b	.LBB1_42

LBB1_39:
	WORD $0x4e183c29     // VMOV V1.D[1], R9                     // mov	x9, v1.d[1]
	SUB  R0, R8, R8      // <--                                  // sub	x8, x8, x0
	RBIT R9, R9          // <--                                  // rbit	x9, x9
	CLZ  R9, R9          // <--                                  // clz	x9, x9
	UBFX $3, R9, $29, R9 // <--                                  // ubfx	x9, x9, #3, #29
	ADDW $8, R9, R9      // <--                                  // add	w9, w9, #8
	ADD  R9, R8, R0      // <--                                  // add	x0, x8, x9
	NOP                  // (skipped)                            // ldp	x29, x30, [sp], #16
	MOVD R0, ret+24(FP)  // <--
	RET                  // <--                                  // ret

LBB1_40:
	MOVW $32, R9 // <--                                  // mov	w9, #32
	JMP  LBB1_44 // <--                                  // b	.LBB1_44

LBB1_41:
	MOVW $32, R9 // <--                                  // mov	w9, #32

LBB1_42:
	RBIT R12, R10       // <--                                  // rbit	x10, x12
	SUB  R0, R8, R8     // <--                                  // sub	x8, x8, x0
	CLZ  R10, R10       // <--                                  // clz	x10, x10
	ORR  R10>>3, R9, R9 // <--                                  // orr	x9, x9, x10, lsr #3
	ORR  $8, R9, R9     // <--                                  // orr	x9, x9, #0x8
	ADD  R9, R8, R0     // <--                                  // add	x0, x8, x9
	NOP                 // (skipped)                            // ldp	x29, x30, [sp], #16
	MOVD R0, ret+24(FP) // <--
	RET                 // <--                                  // ret

LBB1_43:
	MOVW $48, R9 // <--                                  // mov	w9, #48

LBB1_44:
	RBIT R12, R10       // <--                                  // rbit	x10, x12
	SUB  R0, R8, R8     // <--                                  // sub	x8, x8, x0
	CLZ  R10, R10       // <--                                  // clz	x10, x10
	ORR  R10>>3, R9, R9 // <--                                  // orr	x9, x9, x10, lsr #3
	ADD  R9, R8, R0     // <--                                  // add	x0, x8, x9
	NOP                 // (skipped)                            // ldp	x29, x30, [sp], #16
	MOVD R0, ret+24(FP) // <--
	RET                 // <--                                  // ret

DATA uppercasingTable<>+0x00(SB)/8, $0x2020202020202000
DATA uppercasingTable<>+0x08(SB)/8, $0x2020202020202020
DATA uppercasingTable<>+0x10(SB)/8, $0x2020202020202020
DATA uppercasingTable<>+0x18(SB)/8, $0x0000000000202020
GLOBL uppercasingTable<>(SB), (RODATA|NOPTR), $32

TEXT ·EqualFold(SB), NOSPLIT, $0-33
	MOVD a+0(FP), R0
	MOVD a_len+8(FP), R1
	MOVD b+16(FP), R2
	MOVD b_len+24(FP), R3
	CMP  R3, R1                      // <--                                  // cmp	x1, x3
	BNE  LBB2_9                      // <--                                  // b.ne	.LBB2_9
	NOP                              // (skipped)                            // stp	x29, x30, [sp, #-16]!
	MOVD $uppercasingTable<>(SB), R8 // <--                                  // adrp	x8, uppercasingTable
	ADD  $0, R8, R8                  // <--                                  // add	x8, x8, :lo12:uppercasingTable
	CMP  $8, R1                      // <--                                  // cmp	x1, #8
	NOP                              // (skipped)                            // mov	x29, sp
	WORD $0x4c40a100                 // VLD1 (R8), [V0.B16, V1.B16]          // ld1	{ v0.16b, v1.16b }, [x8]
	BCC  LBB2_11                     // <--                                  // b.lo	.LBB2_11
	ADD  R1, R0, R8                  // <--                                  // add	x8, x0, x1
	AND  $15, R1, R1                 // <--                                  // and	x1, x1, #0xf
	SUB  R1, R8, R8                  // <--                                  // sub	x8, x8, x1
	CMP  R0, R8                      // <--                                  // cmp	x8, x0
	BLS  LBB2_6                      // <--                                  // b.ls	.LBB2_6
	WORD $0x4f05e402                 // VMOVI $160, V2.B16                   // movi	v2.16b, #160

LBB2_4:
	WORD  $0x3dc00003                      // FMOVQ (R0), F3                       // ldr	q3, [x0]
	WORD  $0x3dc00044                      // FMOVQ (R2), F4                       // ldr	q4, [x2]
	VADD  V2.B16, V3.B16, V3.B16           // <--                                  // add	v3.16b, v3.16b, v2.16b
	VADD  V2.B16, V4.B16, V4.B16           // <--                                  // add	v4.16b, v4.16b, v2.16b
	VTBL  V3.B16, [V0.B16, V1.B16], V5.B16 // <--                                  // tbl	v5.16b, { v0.16b, v1.16b }, v3.16b
	VTBL  V4.B16, [V0.B16, V1.B16], V6.B16 // <--                                  // tbl	v6.16b, { v0.16b, v1.16b }, v4.16b
	VSUB  V5.B16, V3.B16, V3.B16           // <--                                  // sub	v3.16b, v3.16b, v5.16b
	VSUB  V6.B16, V4.B16, V4.B16           // <--                                  // sub	v4.16b, v4.16b, v6.16b
	WORD  $0x6e248c63                      // VCMEQ V4.B16, V3.B16, V3.B16         // cmeq	v3.16b, v3.16b, v4.16b
	WORD  $0x6eb1a863                      // VUMINV V3.S4, V3                     // uminv	s3, v3.4s
	FMOVS F3, R9                           // <--                                  // fmov	w9, s3
	CMNW  $1, R9                           // <--                                  // cmn	w9, #1
	BNE   LBB2_8                           // <--                                  // b.ne	.LBB2_8
	ADD   $16, R0, R0                      // <--                                  // add	x0, x0, #16
	ADD   $16, R2, R2                      // <--                                  // add	x2, x2, #16
	CMP   R8, R0                           // <--                                  // cmp	x0, x8
	BCC   LBB2_4                           // <--                                  // b.lo	.LBB2_4

LBB2_6:
	SUBS  $8, R1, R8                     // <--                                  // subs	x8, x1, #8
	BCC   LBB2_11                        // <--                                  // b.lo	.LBB2_11
	WORD  $0x0f05e403                    // VMOVI $160, V3.B8                    // movi	v3.8b, #160
	WORD  $0xfc408402                    // FMOVD.P 8(R0), F2                    // ldr	d2, [x0], #8
	WORD  $0xfc408444                    // FMOVD.P 8(R2), F4                    // ldr	d4, [x2], #8
	VADD  V3.B8, V2.B8, V2.B8            // <--                                  // add	v2.8b, v2.8b, v3.8b
	VADD  V3.B8, V4.B8, V3.B8            // <--                                  // add	v3.8b, v4.8b, v3.8b
	VTBL  V2.B8, [V0.B16, V1.B16], V4.B8 // <--                                  // tbl	v4.8b, { v0.16b, v1.16b }, v2.8b
	VTBL  V3.B8, [V0.B16, V1.B16], V5.B8 // <--                                  // tbl	v5.8b, { v0.16b, v1.16b }, v3.8b
	VSUB  V4.B8, V2.B8, V2.B8            // <--                                  // sub	v2.8b, v2.8b, v4.8b
	VSUB  V5.B8, V3.B8, V3.B8            // <--                                  // sub	v3.8b, v3.8b, v5.8b
	WORD  $0x2e238c42                    // VCMEQ V3.B8, V2.B8, V2.B8            // cmeq	v2.8b, v2.8b, v3.8b
	FMOVD F2, R9                         // <--                                  // fmov	x9, d2
	CMN   $1, R9                         // <--                                  // cmn	x9, #1
	BEQ   LBB2_10                        // <--                                  // b.eq	.LBB2_10

LBB2_8:
	MOVW ZR, R0         // <--                                  // mov	w0, wzr
	NOP                 // (skipped)                            // ldp	x29, x30, [sp], #16
	MOVB R0, ret+32(FP) // <--
	RET                 // <--                                  // ret

LBB2_9:
	MOVW ZR, R0         // <--                                  // mov	w0, wzr
	MOVB R0, ret+32(FP) // <--
	RET                 // <--                                  // ret

LBB2_10:
	MOVD R8, R1 // <--                                  // mov	x1, x8

LBB2_11:
	CBZ  R1, LBB2_17 // <--                                  // cbz	x1, .LBB2_17
	SUBS $4, R1, R10 // <--                                  // subs	x10, x1, #4
	BCC  LBB2_18     // <--                                  // b.lo	.LBB2_18
	WORD $0xb8404408 // MOVWU.P 4(R0), R8                    // ldr	w8, [x0], #4
	WORD $0xb8404449 // MOVWU.P 4(R2), R9                    // ldr	w9, [x2], #4
	MOVD R10, R1     // <--                                  // mov	x1, x10
	CMP  $1, R10     // <--                                  // cmp	x10, #1
	BEQ  LBB2_19     // <--                                  // b.eq	.LBB2_19

LBB2_14:
	CMP  $2, R1         // <--                                  // cmp	x1, #2
	BEQ  LBB2_20        // <--                                  // b.eq	.LBB2_20
	CMP  $3, R1         // <--                                  // cmp	x1, #3
	BNE  LBB2_21        // <--                                  // b.ne	.LBB2_21
	WORD $0x7940000a    // MOVHU (R0), R10                      // ldrh	w10, [x0]
	LSL  $24, R8, R8    // <--                                  // lsl	x8, x8, #24
	WORD $0x7940004c    // MOVHU (R2), R12                      // ldrh	w12, [x2]
	LSL  $24, R9, R9    // <--                                  // lsl	x9, x9, #24
	WORD $0x3940080b    // MOVBU 2(R0), R11                     // ldrb	w11, [x0, #2]
	WORD $0x3940084d    // MOVBU 2(R2), R13                     // ldrb	w13, [x2, #2]
	ORR  R10<<8, R8, R8 // <--                                  // orr	x8, x8, x10, lsl #8
	ORR  R12<<8, R9, R9 // <--                                  // orr	x9, x9, x12, lsl #8
	ORR  R11, R8, R8    // <--                                  // orr	x8, x8, x11
	ORR  R13, R9, R9    // <--                                  // orr	x9, x9, x13
	JMP  LBB2_21        // <--                                  // b	.LBB2_21

LBB2_17:
	MOVW $1, R0         // <--                                  // mov	w0, #1
	NOP                 // (skipped)                            // ldp	x29, x30, [sp], #16
	MOVB R0, ret+32(FP) // <--
	RET                 // <--                                  // ret

LBB2_18:
	MOVD ZR, R9  // <--                                  // mov	x9, xzr
	MOVD ZR, R8  // <--                                  // mov	x8, xzr
	CMP  $1, R1  // <--                                  // cmp	x1, #1
	BNE  LBB2_14 // <--                                  // b.ne	.LBB2_14

LBB2_19:
	WORD $0x3940000a    // MOVBU (R0), R10                      // ldrb	w10, [x0]
	WORD $0x3940004b    // MOVBU (R2), R11                      // ldrb	w11, [x2]
	ORR  R8<<8, R10, R8 // <--                                  // orr	x8, x10, x8, lsl #8
	ORR  R9<<8, R11, R9 // <--                                  // orr	x9, x11, x9, lsl #8
	JMP  LBB2_21        // <--                                  // b	.LBB2_21

LBB2_20:
	WORD $0x7940000a     // MOVHU (R0), R10                      // ldrh	w10, [x0]
	WORD $0x7940004b     // MOVHU (R2), R11                      // ldrh	w11, [x2]
	ORR  R8<<16, R10, R8 // <--                                  // orr	x8, x10, x8, lsl #16
	ORR  R9<<16, R11, R9 // <--                                  // orr	x9, x11, x9, lsl #16

LBB2_21:
	WORD  $0x0f05e402                    // VMOVI $160, V2.B8                    // movi	v2.8b, #160
	FMOVD R8, F3                         // <--                                  // fmov	d3, x8
	FMOVD R9, F4                         // <--                                  // fmov	d4, x9
	VADD  V2.B8, V3.B8, V3.B8            // <--                                  // add	v3.8b, v3.8b, v2.8b
	VADD  V2.B8, V4.B8, V2.B8            // <--                                  // add	v2.8b, v4.8b, v2.8b
	VTBL  V3.B8, [V0.B16, V1.B16], V4.B8 // <--                                  // tbl	v4.8b, { v0.16b, v1.16b }, v3.8b
	VTBL  V2.B8, [V0.B16, V1.B16], V0.B8 // <--                                  // tbl	v0.8b, { v0.16b, v1.16b }, v2.8b
	VSUB  V4.B8, V3.B8, V1.B8            // <--                                  // sub	v1.8b, v3.8b, v4.8b
	VSUB  V0.B8, V2.B8, V0.B8            // <--                                  // sub	v0.8b, v2.8b, v0.8b
	WORD  $0x2e208c20                    // VCMEQ V0.B8, V1.B8, V0.B8            // cmeq	v0.8b, v1.8b, v0.8b
	FMOVD F0, R8                         // <--                                  // fmov	x8, d0
	CMN   $1, R8                         // <--                                  // cmn	x8, #1
	CSETW EQ, R0                         // <--                                  // cset	w0, eq
	NOP                                  // (skipped)                            // ldp	x29, x30, [sp], #16
	MOVB  R0, ret+32(FP)                 // <--
	RET                                  // <--                                  // ret
