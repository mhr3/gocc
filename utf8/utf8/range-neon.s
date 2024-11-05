//go:build !noasm && arm64
// Code generated by gocc devel -- DO NOT EDIT.
//
// Source file         : range-neon.c
// Clang version       : Apple clang version 16.0.0 (clang-1600.0.26.4)
// Target architecture : arm64
// Compiler options    : [none]

#include "textflag.h"

DATA LCPI0_0<>+0x00(SB)/1, $0x00
DATA LCPI0_0<>+0x01(SB)/1, $0x00
DATA LCPI0_0<>+0x02(SB)/1, $0x00
DATA LCPI0_0<>+0x03(SB)/1, $0x00
DATA LCPI0_0<>+0x04(SB)/1, $0x00
DATA LCPI0_0<>+0x05(SB)/1, $0x00
DATA LCPI0_0<>+0x06(SB)/1, $0x00
DATA LCPI0_0<>+0x07(SB)/1, $0x00
DATA LCPI0_0<>+0x08(SB)/1, $0x00
DATA LCPI0_0<>+0x09(SB)/1, $0x00
DATA LCPI0_0<>+0x0a(SB)/1, $0x00
DATA LCPI0_0<>+0x0b(SB)/1, $0x00
DATA LCPI0_0<>+0x0c(SB)/1, $0x01
DATA LCPI0_0<>+0x0d(SB)/1, $0x01
DATA LCPI0_0<>+0x0e(SB)/1, $0x02
DATA LCPI0_0<>+0x0f(SB)/1, $0x03
GLOBL LCPI0_0<>(SB), (RODATA|NOPTR), $16

DATA LCPI0_1<>+0x00(SB)/1, $0x00
DATA LCPI0_1<>+0x01(SB)/1, $0x00
DATA LCPI0_1<>+0x02(SB)/1, $0x00
DATA LCPI0_1<>+0x03(SB)/1, $0x00
DATA LCPI0_1<>+0x04(SB)/1, $0x00
DATA LCPI0_1<>+0x05(SB)/1, $0x00
DATA LCPI0_1<>+0x06(SB)/1, $0x00
DATA LCPI0_1<>+0x07(SB)/1, $0x00
DATA LCPI0_1<>+0x08(SB)/1, $0x00
DATA LCPI0_1<>+0x09(SB)/1, $0x00
DATA LCPI0_1<>+0x0a(SB)/1, $0x00
DATA LCPI0_1<>+0x0b(SB)/1, $0x00
DATA LCPI0_1<>+0x0c(SB)/1, $0x08
DATA LCPI0_1<>+0x0d(SB)/1, $0x08
DATA LCPI0_1<>+0x0e(SB)/1, $0x08
DATA LCPI0_1<>+0x0f(SB)/1, $0x08
GLOBL LCPI0_1<>(SB), (RODATA|NOPTR), $16

DATA LCPI0_2<>+0x00(SB)/1, $0x00
DATA LCPI0_2<>+0x01(SB)/1, $0x80
DATA LCPI0_2<>+0x02(SB)/1, $0x80
DATA LCPI0_2<>+0x03(SB)/1, $0x80
DATA LCPI0_2<>+0x04(SB)/1, $0xa0
DATA LCPI0_2<>+0x05(SB)/1, $0x80
DATA LCPI0_2<>+0x06(SB)/1, $0x90
DATA LCPI0_2<>+0x07(SB)/1, $0x80
DATA LCPI0_2<>+0x08(SB)/1, $0xc2
DATA LCPI0_2<>+0x09(SB)/1, $0xff
DATA LCPI0_2<>+0x0a(SB)/1, $0xff
DATA LCPI0_2<>+0x0b(SB)/1, $0xff
DATA LCPI0_2<>+0x0c(SB)/1, $0xff
DATA LCPI0_2<>+0x0d(SB)/1, $0xff
DATA LCPI0_2<>+0x0e(SB)/1, $0xff
DATA LCPI0_2<>+0x0f(SB)/1, $0xff
GLOBL LCPI0_2<>(SB), (RODATA|NOPTR), $16

DATA LCPI0_3<>+0x00(SB)/1, $0x7f
DATA LCPI0_3<>+0x01(SB)/1, $0xbf
DATA LCPI0_3<>+0x02(SB)/1, $0xbf
DATA LCPI0_3<>+0x03(SB)/1, $0xbf
DATA LCPI0_3<>+0x04(SB)/1, $0xbf
DATA LCPI0_3<>+0x05(SB)/1, $0x9f
DATA LCPI0_3<>+0x06(SB)/1, $0xbf
DATA LCPI0_3<>+0x07(SB)/1, $0x8f
DATA LCPI0_3<>+0x08(SB)/1, $0xf4
DATA LCPI0_3<>+0x09(SB)/1, $0x00
DATA LCPI0_3<>+0x0a(SB)/1, $0x00
DATA LCPI0_3<>+0x0b(SB)/1, $0x00
DATA LCPI0_3<>+0x0c(SB)/1, $0x00
DATA LCPI0_3<>+0x0d(SB)/1, $0x00
DATA LCPI0_3<>+0x0e(SB)/1, $0x00
DATA LCPI0_3<>+0x0f(SB)/1, $0x00
GLOBL LCPI0_3<>(SB), (RODATA|NOPTR), $16

DATA _range_adjust_tbl<>+0x00(SB)/8, $0x0000000000000302
DATA _range_adjust_tbl<>+0x08(SB)/8, $0x0000000000000400
DATA _range_adjust_tbl<>+0x10(SB)/8, $0x0000000000000000
DATA _range_adjust_tbl<>+0x18(SB)/8, $0x0000000000030000
GLOBL _range_adjust_tbl<>(SB), (RODATA|NOPTR), $32

TEXT ·utf8_valid_range(SB), NOSPLIT, $0-17
	MOVD data+0(FP), R0
	MOVD len+8(FP), R1
	NOP                               // (skipped)                            // stp	x29, x30, [sp, #-16]!
	MOVD $_range_adjust_tbl<>(SB), R8 // <--                                  // adrp	x8, _range_adjust_tbl
	ADD  $0, R8, R8                   // <--                                  // add	x8, x8, :lo12:_range_adjust_tbl
	MOVD $LCPI0_1<>(SB), R9           // <--                                  // adrp	x9, .LCPI0_1
	MOVW $3, R10                      // <--                                  // mov	w10, #3
	WORD $0x6f00e402                  // VMOVI $0, V2.D2                      // movi	v2.2d, #0000000000000000
	NOP                               // (skipped)                            // mov	x29, sp
	WORD $0x4c408100                  // VLD2 (R8), [V0.B16, V1.B16]          // ld2	{ v0.16b, v1.16b }, [x8]
	MOVD $LCPI0_0<>(SB), R8           // <--                                  // adrp	x8, .LCPI0_0
	WORD $0x4f00e427                  // VMOVI $1, V7.B16                     // movi	v7.16b, #1
	WORD $0x4f00e450                  // VMOVI $2, V16.B16                    // movi	v16.16b, #2
	WORD $0x4f01e411                  // VMOVI $32, V17.B16                   // movi	v17.16b, #32
	WORD $0x3dc00104                  // FMOVQ (R8), F4                       // ldr	q4, [x8, :lo12:.LCPI0_0]
	MOVD $LCPI0_2<>(SB), R8           // <--                                  // adrp	x8, .LCPI0_2
	WORD $0x3dc00126                  // FMOVQ (R9), F6                       // ldr	q6, [x9, :lo12:.LCPI0_1]
	MOVD $LCPI0_3<>(SB), R9           // <--                                  // adrp	x9, .LCPI0_3
	WORD $0x6f00e412                  // VMOVI $0, V18.D2                     // movi	v18.2d, #0000000000000000
	WORD $0x3dc00103                  // FMOVQ (R8), F3                       // ldr	q3, [x8, :lo12:.LCPI0_2]
	WORD $0x3dc00125                  // FMOVQ (R9), F5                       // ldr	q5, [x9, :lo12:.LCPI0_3]
	MOVW $3221225471, R9              // <--                                  // mov	w9, #-1073741825

LBB0_1:
	CMP  $16, R1     // <--                                  // cmp	x1, #16
	BLT  LBB0_6      // <--                                  // b.lt	.LBB0_6
	WORD $0x6f00e414 // VMOVI $0, V20.D2                     // movi	v20.2d, #0000000000000000
	WORD $0x6f00e413 // VMOVI $0, V19.D2                     // movi	v19.2d, #0000000000000000
	JMP  LBB0_4      // <--                                  // b	.LBB0_4

LBB0_3:
	ADD $16, R1, R8 // <--                                  // add	x8, x1, #16
	CMP $31, R8     // <--                                  // cmp	x8, #31
	BLS LBB0_7      // <--                                  // b.ls	.LBB0_7

LBB0_4:
	WORD  $0x4eb31e75                        // VMOV V19.B16, V21.B16                // mov	v21.16b, v19.16b
	SUB   $16, R1, R1                        // <--                                  // sub	x1, x1, #16
	WORD  $0x3cc10413                        // FMOVQ.P 16(R0), F19                  // ldr	q19, [x0], #16
	WORD  $0x4eb41e96                        // VMOV V20.B16, V22.B16                // mov	v22.16b, v20.16b
	TST   $112, R1                           // <--                                  // tst	x1, #0x70
	WORD  $0x6f0c0677                        // VUSHR $4, V19.B16, V23.B16           // ushr	v23.16b, v19.16b, #4
	VEXT  $15, V19.B16, V21.B16, V21.B16     // <--                                  // ext	v21.16b, v21.16b, v19.16b, #15
	VTBL  V23.B16, [V4.B16], V20.B16         // <--                                  // tbl	v20.16b, { v4.16b }, v23.16b
	VTBL  V23.B16, [V6.B16], V23.B16         // <--                                  // tbl	v23.16b, { v6.16b }, v23.16b
	VADD  V17.B16, V21.B16, V21.B16          // <--                                  // add	v21.16b, v21.16b, v17.16b
	VEXT  $14, V20.B16, V22.B16, V24.B16     // <--                                  // ext	v24.16b, v22.16b, v20.16b, #14
	VEXT  $13, V20.B16, V22.B16, V25.B16     // <--                                  // ext	v25.16b, v22.16b, v20.16b, #13
	VEXT  $15, V20.B16, V22.B16, V22.B16     // <--                                  // ext	v22.16b, v22.16b, v20.16b, #15
	VTBL  V21.B16, [V0.B16, V1.B16], V21.B16 // <--                                  // tbl	v21.16b, { v0.16b, v1.16b }, v21.16b
	WORD  $0x6e272f18                        // VUQSUB V7.B16, V24.B16, V24.B16      // uqsub	v24.16b, v24.16b, v7.16b
	WORD  $0x6e302f39                        // VUQSUB V16.B16, V25.B16, V25.B16     // uqsub	v25.16b, v25.16b, v16.16b
	VORR  V23.B16, V22.B16, V22.B16          // <--                                  // orr	v22.16b, v22.16b, v23.16b
	VORR  V24.B16, V22.B16, V22.B16          // <--                                  // orr	v22.16b, v22.16b, v24.16b
	VORR  V25.B16, V22.B16, V22.B16          // <--                                  // orr	v22.16b, v22.16b, v25.16b
	VADD  V21.B16, V22.B16, V21.B16          // <--                                  // add	v21.16b, v22.16b, v21.16b
	VTBL  V21.B16, [V3.B16], V22.B16         // <--                                  // tbl	v22.16b, { v3.16b }, v21.16b
	VTBL  V21.B16, [V5.B16], V21.B16         // <--                                  // tbl	v21.16b, { v5.16b }, v21.16b
	WORD  $0x6e3336d6                        // VCMHI V19.B16, V22.B16, V22.B16      // cmhi	v22.16b, v22.16b, v19.16b
	WORD  $0x6e353675                        // VCMHI V21.B16, V19.B16, V21.B16      // cmhi	v21.16b, v19.16b, v21.16b
	VORR  V22.B16, V2.B16, V2.B16            // <--                                  // orr	v2.16b, v2.16b, v22.16b
	VORR  V21.B16, V18.B16, V18.B16          // <--                                  // orr	v18.16b, v18.16b, v21.16b
	BNE   LBB0_3                             // <--                                  // b.ne	.LBB0_3
	VORR  V2.B16, V18.B16, V21.B16           // <--                                  // orr	v21.16b, v18.16b, v2.16b
	WORD  $0x6eb0aab5                        // VUMAXV V21.S4, V21                   // umaxv	s21, v21.4s
	FMOVS F21, R8                            // <--                                  // fmov	w8, s21
	CBZW  R8, LBB0_3                         // <--                                  // cbz	w8, .LBB0_3
	JMP   LBB0_17                            // <--                                  // b	.LBB0_17

LBB0_6:
	WORD $0x6f00e413 // VMOVI $0, V19.D2                     // movi	v19.2d, #0000000000000000

LBB0_7:
	VORR  V2.B16, V18.B16, V2.B16 // <--                                  // orr	v2.16b, v18.16b, v2.16b
	WORD  $0x6eb0a854             // VUMAXV V2.S4, V20                    // umaxv	s20, v2.4s
	FMOVS F20, R11                // <--                                  // fmov	w11, s20
	CMPW  $0, R11                 // <--                                  // cmp	w11, #0
	CSETW EQ, R8                  // <--                                  // cset	w8, eq
	CBNZW R11, LBB0_18            // <--                                  // cbnz	w11, .LBB0_18
	WORD  $0x0e1c3e68             // VMOV V19.S[3], R8                    // mov	w8, v19.s[3]
	CMPW  R9, R8                  // <--                                  // cmp	w8, w9
	BLE   LBB0_10                 // <--                                  // b.le	.LBB0_10
	MOVW  $1, R8                  // <--                                  // mov	w8, #1
	ADDS  R1, R8, R1              // <--                                  // adds	x1, x8, x1
	BNE   LBB0_13                 // <--                                  // b.ne	.LBB0_13
	JMP   LBB0_19                 // <--                                  // b	.LBB0_19

LBB0_10:
	CMPW R8<<8, R9  // <--                                  // cmp	w9, w8, lsl #8
	BGE  LBB0_12    // <--                                  // b.ge	.LBB0_12
	MOVW $2, R8     // <--                                  // mov	w8, #2
	ADDS R1, R8, R1 // <--                                  // adds	x1, x8, x1
	BNE  LBB0_13    // <--                                  // b.ne	.LBB0_13
	JMP  LBB0_19    // <--                                  // b	.LBB0_19

LBB0_12:
	CMPW R8<<16, R9      // <--                                  // cmp	w9, w8, lsl #16
	CSEL LT, R10, ZR, R8 // <--                                  // csel	x8, x10, xzr, lt
	ADDS R1, R8, R1      // <--                                  // adds	x1, x8, x1
	BEQ  LBB0_19         // <--                                  // b.eq	.LBB0_19

LBB0_13:
	SUB  R8, R0, R0       // <--                                  // sub	x0, x0, x8
	CMP  $15, R1          // <--                                  // cmp	x1, #15
	BGT  LBB0_1           // <--                                  // b.gt	.LBB0_1
	TBNZ $3, R1, LBB0_20  // <--                                  // tbnz	w1, #3, .LBB0_20
	MOVD ZR, R8           // <--                                  // mov	x8, xzr
	MOVD R1, R10          // <--                                  // mov	x10, x1
	TBZ  $2, R10, LBB0_21 // <--                                  // tbz	w10, #2, .LBB0_21

LBB0_16:
	WORD $0xb8404409      // MOVWU.P 4(R0), R9                    // ldr	w9, [x0], #4
	SUB  $4, R10, R10     // <--                                  // sub	x10, x10, #4
	TBNZ $1, R10, LBB0_22 // <--                                  // tbnz	w10, #1, .LBB0_22
	JMP  LBB0_23          // <--                                  // b	.LBB0_23

LBB0_17:
	MOVW ZR, R8 // <--                                  // mov	w8, wzr

LBB0_18:
	MOVW R8, R0         // <--                                  // mov	w0, w8
	NOP                 // (skipped)                            // ldp	x29, x30, [sp], #16
	MOVB R0, ret+16(FP) // <--
	RET                 // <--                                  // ret

LBB0_19:
	MOVW $1, R8         // <--                                  // mov	w8, #1
	MOVW R8, R0         // <--                                  // mov	w0, w8
	NOP                 // (skipped)                            // ldp	x29, x30, [sp], #16
	MOVB R0, ret+16(FP) // <--
	RET                 // <--                                  // ret

LBB0_20:
	WORD $0xf8408408      // MOVD.P 8(R0), R8                     // ldr	x8, [x0], #8
	SUB  $8, R1, R10      // <--                                  // sub	x10, x1, #8
	TBNZ $2, R10, LBB0_16 // <--                                  // tbnz	w10, #2, .LBB0_16

LBB0_21:
	MOVD ZR, R9           // <--                                  // mov	x9, xzr
	TBZ  $1, R10, LBB0_23 // <--                                  // tbz	w10, #1, .LBB0_23

LBB0_22:
	LSLW $3, R1, R12   // <--                                  // lsl	w12, w1, #3
	SUB  $2, R10, R10  // <--                                  // sub	x10, x10, #2
	WORD $0x7840240b   // MOVHU.P 2(R0), R11                   // ldrh	w11, [x0], #2
	AND  $32, R12, R12 // <--                                  // and	x12, x12, #0x20
	LSL  R12, R11, R11 // <--                                  // lsl	x11, x11, x12
	ORR  R9, R11, R9   // <--                                  // orr	x9, x11, x9

LBB0_23:
	TBZ  $0, R10, LBB0_25 // <--                                  // tbz	w10, #0, .LBB0_25
	LSLW $3, R1, R10      // <--                                  // lsl	w10, w1, #3
	WORD $0x3940000b      // MOVBU (R0), R11                      // ldrb	w11, [x0]
	AND  $48, R10, R10    // <--                                  // and	x10, x10, #0x30
	LSL  R10, R11, R10    // <--                                  // lsl	x10, x11, x10
	ORR  R9, R10, R9      // <--                                  // orr	x9, x10, x9

LBB0_25:
	CMP   $7, R1                            // <--                                  // cmp	x1, #7
	CSEL  GT, R8, R9, R8                    // <--                                  // csel	x8, x8, x9, gt
	CSEL  GT, R9, ZR, R9                    // <--                                  // csel	x9, x9, xzr, gt
	WORD  $0x6f00e410                       // VMOVI $0, V16.D2                     // movi	v16.2d, #0000000000000000
	WORD  $0x4f01e412                       // VMOVI $32, V18.B16                   // movi	v18.16b, #32
	FMOVD R8, F7                            // <--                                  // fmov	d7, x8
	WORD  $0x4f00e431                       // VMOVI $1, V17.B16                    // movi	v17.16b, #1
	WORD  $0x4f00e455                       // VMOVI $2, V21.B16                    // movi	v21.16b, #2
	WORD  $0x4e181d27                       // VMOV R9, V7.D[1]                     // mov	v7.d[1], x9
	WORD  $0x6f0c04f3                       // VUSHR $4, V7.B16, V19.B16            // ushr	v19.16b, v7.16b, #4
	VEXT  $15, V7.B16, V16.B16, V20.B16     // <--                                  // ext	v20.16b, v16.16b, v7.16b, #15
	VTBL  V19.B16, [V4.B16], V4.B16         // <--                                  // tbl	v4.16b, { v4.16b }, v19.16b
	VTBL  V19.B16, [V6.B16], V6.B16         // <--                                  // tbl	v6.16b, { v6.16b }, v19.16b
	VADD  V18.B16, V20.B16, V18.B16         // <--                                  // add	v18.16b, v20.16b, v18.16b
	VEXT  $14, V4.B16, V16.B16, V19.B16     // <--                                  // ext	v19.16b, v16.16b, v4.16b, #14
	VEXT  $13, V4.B16, V16.B16, V20.B16     // <--                                  // ext	v20.16b, v16.16b, v4.16b, #13
	VEXT  $15, V4.B16, V16.B16, V4.B16      // <--                                  // ext	v4.16b, v16.16b, v4.16b, #15
	VTBL  V18.B16, [V0.B16, V1.B16], V0.B16 // <--                                  // tbl	v0.16b, { v0.16b, v1.16b }, v18.16b
	WORD  $0x6e312e61                       // VUQSUB V17.B16, V19.B16, V1.B16      // uqsub	v1.16b, v19.16b, v17.16b
	WORD  $0x6e352e90                       // VUQSUB V21.B16, V20.B16, V16.B16     // uqsub	v16.16b, v20.16b, v21.16b
	VORR  V6.B16, V4.B16, V4.B16            // <--                                  // orr	v4.16b, v4.16b, v6.16b
	VORR  V1.B16, V4.B16, V1.B16            // <--                                  // orr	v1.16b, v4.16b, v1.16b
	VORR  V16.B16, V1.B16, V1.B16           // <--                                  // orr	v1.16b, v1.16b, v16.16b
	VADD  V0.B16, V1.B16, V0.B16            // <--                                  // add	v0.16b, v1.16b, v0.16b
	VTBL  V0.B16, [V3.B16], V1.B16          // <--                                  // tbl	v1.16b, { v3.16b }, v0.16b
	VTBL  V0.B16, [V5.B16], V0.B16          // <--                                  // tbl	v0.16b, { v5.16b }, v0.16b
	WORD  $0x6e273421                       // VCMHI V7.B16, V1.B16, V1.B16         // cmhi	v1.16b, v1.16b, v7.16b
	WORD  $0x6e2034e0                       // VCMHI V0.B16, V7.B16, V0.B16         // cmhi	v0.16b, v7.16b, v0.16b
	VORR  V1.B16, V0.B16, V0.B16            // <--                                  // orr	v0.16b, v0.16b, v1.16b
	VORR  V0.B16, V2.B16, V0.B16            // <--                                  // orr	v0.16b, v2.16b, v0.16b
	WORD  $0x6e30a800                       // VUMAXV V0.B16, V0                    // umaxv	b0, v0.16b
	FMOVS F0, R8                            // <--                                  // fmov	w8, s0
	CMPW  $0, R8                            // <--                                  // cmp	w8, #0
	CSETW EQ, R8                            // <--                                  // cset	w8, eq
	MOVW  R8, R0                            // <--                                  // mov	w0, w8
	NOP                                     // (skipped)                            // ldp	x29, x30, [sp], #16
	MOVB  R0, ret+16(FP)                    // <--
	RET                                     // <--                                  // ret
