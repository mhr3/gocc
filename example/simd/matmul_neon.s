//go:build !noasm && arm64
// AUTO-GENERATED BY GOCC -- DO NOT EDIT

#include "textflag.h"

TEXT ·f32_axpy(SB), NOSPLIT, $0-28
	MOVD x+0(FP), R0
	MOVD y+8(FP), R1
	MOVD size+16(FP), R2
	MOVW alpha+24(FP), R3
	NOP                   // stp	x29, x30, [sp, #-16]!
	WORD $0xf100105f      // CMP $4, R2;	cmp	x2, #4
	NOP                   // mov	x29, sp
	BCC  LBB0_3           // b.lo	.LBB0_3
	WORD $0x52800068      // MOVW $3, R8;	mov	w8, #3
	WORD $0xaa0003e9      // MOVD R0, R9;	mov	x9, x0
	WORD $0xaa0103ea      // MOVD R1, R10;	mov	x10, x1

LBB0_2:
	WORD $0x3dc00141 // FMOVQ (R10), F1;	ldr	q1, [x10]
	WORD $0x91001108 // ADD $4, R8, R8;	add	x8, x8, #4
	WORD $0x3cc10522 // FMOVQ.P 16(R9), F2;	ldr	q2, [x9], #16
	WORD $0xeb02011f // CMP R2, R8;	cmp	x8, x2
	WORD $0x4f801041 // VFMLA V0.S[0], V2.S4, V1.S4;	fmla	v1.4s, v2.4s, v0.s[0]
	WORD $0x3c810541 // FMOVQ.P F1, 16(R10);	str	q1, [x10], #16
	BCC  LBB0_2      // b.lo	.LBB0_2

LBB0_3:
	WORD $0xf240045f // TST $3, R2;	tst	x2, #0x3
	BEQ  LBB0_7      // b.eq	.LBB0_7
	WORD $0x927ef448 // AND $-4, R2, R8;	and	x8, x2, #0xfffffffffffffffc
	WORD $0xeb02011f // CMP R2, R8;	cmp	x8, x2
	BCS  LBB0_7      // b.hs	.LBB0_7
	WORD $0xd37ef449 // LSL $2, R2, R9;	lsl	x9, x2, #2
	WORD $0xcb080048 // SUB R8, R2, R8;	sub	x8, x2, x8
	WORD $0x927ced2a // AND $-16, R9, R10;	and	x10, x9, #0xfffffffffffffff0
	WORD $0x8b0a0029 // ADD R10, R1, R9;	add	x9, x1, x10
	WORD $0x8b0a000a // ADD R10, R0, R10;	add	x10, x0, x10

LBB0_6:
	WORD $0xbc404541 // FMOVS.P 4(R10), F1;	ldr	s1, [x10], #4
	WORD $0xbd400122 // FMOVS (R9), F2;	ldr	s2, [x9]
	WORD $0xf1000508 // SUBS $1, R8, R8;	subs	x8, x8, #1
	WORD $0x1f000821 // FMADDS F0, F2, F1, F1;	fmadd	s1, s1, s0, s2
	WORD $0xbc004521 // FMOVS.P F1, 4(R9);	str	s1, [x9], #4
	BNE  LBB0_6      // b.ne	.LBB0_6

LBB0_7:
	NOP // ldp	x29, x30, [sp], #16
	RET // ret

TEXT ·f32_matmul(SB), NOSPLIT, $0-32
	MOVD dst+0(FP), R0
	MOVD m+8(FP), R1
	MOVD n+16(FP), R2
	MOVD dims+24(FP), R3
	NOP                   // stp	x29, x30, [sp, #-16]!
	WORD $0xf2403c68      // ANDS $65535, R3, R8;	ands	x8, x3, #0xffff
	NOP                   // mov	x29, sp
	BEQ  LBB1_25          // b.eq	.LBB1_25
	WORD $0xd370fc6a      // LSR $48, R3, R10;	lsr	x10, x3, #48
	WORD $0xf250047f      // TST $844424930131968, R3;	tst	x3, #0x3000000000000
	WORD $0x927e354b      // AND $65532, R10, R11;	and	x11, x10, #0xfffc
	WORD $0xd3507c69      // UBFX $16, R3, $16, R9;	ubfx	x9, x3, #16, #16
	WORD $0xfa4b1140      // CCMP NE, R10, R11, $0;	ccmp	x10, x11, #0, ne
	WORD $0x1a9f97ec      // CSETW HI, R12;	cset	w12, hi
	CBZ  R9, LBB1_25      // cbz	x9, .LBB1_25
	WORD $0xd372fc6d      // LSR $50, R3, R13;	lsr	x13, x3, #50
	CBZ  R13, LBB1_12     // cbz	x13, .LBB1_12
	TBZ  $0, R12, LBB1_19 // tbz	w12, #0, .LBB1_19
	WORD $0xaa1f03ec      // MOVD ZR, R12;	mov	x12, xzr
	WORD $0xd37ef54d      // LSL $2, R10, R13;	lsl	x13, x10, #2

LBB1_5:
	WORD $0x9b097d8f // MUL R9, R12, R15;	mul	x15, x12, x9
	WORD $0xaa1f03ee // MOVD ZR, R14;	mov	x14, xzr
	WORD $0xaa0203f0 // MOVD R2, R16;	mov	x16, x2

LBB1_6:
	WORD $0x8b0f01d1 // ADD R15, R14, R17;	add	x17, x14, x15
	WORD $0xaa1003f2 // MOVD R16, R18;	mov	x18, x16
	WORD $0x52800063 // MOVW $3, R3;	mov	w3, #3
	WORD $0xbc717820 // FMOVS (R1)(R17<<2), F0;	ldr	s0, [x1, x17, lsl  #2]
	WORD $0xaa0003f1 // MOVD R0, R17;	mov	x17, x0

LBB1_7:
	WORD $0x3dc00221 // FMOVQ (R17), F1;	ldr	q1, [x17]
	WORD $0x91001063 // ADD $4, R3, R3;	add	x3, x3, #4
	WORD $0x3cc10642 // FMOVQ.P 16(R18), F2;	ldr	q2, [x18], #16
	WORD $0xeb0a007f // CMP R10, R3;	cmp	x3, x10
	WORD $0x4f801041 // VFMLA V0.S[0], V2.S4, V1.S4;	fmla	v1.4s, v2.4s, v0.s[0]
	WORD $0x3c810621 // FMOVQ.P F1, 16(R17);	str	q1, [x17], #16
	BCC  LBB1_7      // b.lo	.LBB1_7
	WORD $0xaa0b03f1 // MOVD R11, R17;	mov	x17, x11

LBB1_9:
	WORD $0xd37ef632 // LSL $2, R17, R18;	lsl	x18, x17, #2
	WORD $0x91000631 // ADD $1, R17, R17;	add	x17, x17, #1
	WORD $0xeb11015f // CMP R17, R10;	cmp	x10, x17
	WORD $0xbc726a01 // FMOVS (R16)(R18), F1;	ldr	s1, [x16, x18]
	WORD $0xbc726802 // FMOVS (R0)(R18), F2;	ldr	s2, [x0, x18]
	WORD $0x1f000821 // FMADDS F0, F2, F1, F1;	fmadd	s1, s1, s0, s2
	WORD $0xbc326801 // FMOVS F1, (R0)(R18);	str	s1, [x0, x18]
	BNE  LBB1_9      // b.ne	.LBB1_9
	WORD $0x910005ce // ADD $1, R14, R14;	add	x14, x14, #1
	WORD $0x8b0d0210 // ADD R13, R16, R16;	add	x16, x16, x13
	WORD $0xeb0901df // CMP R9, R14;	cmp	x14, x9
	BNE  LBB1_6      // b.ne	.LBB1_6
	WORD $0x9100058c // ADD $1, R12, R12;	add	x12, x12, #1
	WORD $0x8b0d0000 // ADD R13, R0, R0;	add	x0, x0, x13
	WORD $0xeb08019f // CMP R8, R12;	cmp	x12, x8
	BNE  LBB1_5      // b.ne	.LBB1_5
	JMP  LBB1_25     // b	.LBB1_25

LBB1_12:
	CBZW R12, LBB1_25 // cbz	w12, .LBB1_25
	WORD $0xd36efc6d  // LSR $46, R3, R13;	lsr	x13, x3, #46
	WORD $0xaa1f03ec  // MOVD ZR, R12;	mov	x12, xzr
	WORD $0x927c35ae  // AND $262128, R13, R14;	and	x14, x13, #0x3fff0
	WORD $0xcb0b014b  // SUB R11, R10, R11;	sub	x11, x10, x11
	WORD $0x8b0e000d  // ADD R14, R0, R13;	add	x13, x0, x14
	WORD $0xd37ef54a  // LSL $2, R10, R10;	lsl	x10, x10, #2
	WORD $0x8b0e004e  // ADD R14, R2, R14;	add	x14, x2, x14

LBB1_14:
	WORD $0x9b097d90 // MUL R9, R12, R16;	mul	x16, x12, x9
	WORD $0xaa1f03ef // MOVD ZR, R15;	mov	x15, xzr
	WORD $0xaa0e03f1 // MOVD R14, R17;	mov	x17, x14

LBB1_15:
	WORD $0x8b1001f2 // ADD R16, R15, R18;	add	x18, x15, x16
	WORD $0xaa0d03e0 // MOVD R13, R0;	mov	x0, x13
	WORD $0xaa0b03e2 // MOVD R11, R2;	mov	x2, x11
	WORD $0xbc727820 // FMOVS (R1)(R18<<2), F0;	ldr	s0, [x1, x18, lsl  #2]
	WORD $0xaa1103f2 // MOVD R17, R18;	mov	x18, x17

LBB1_16:
	WORD $0xbc404641 // FMOVS.P 4(R18), F1;	ldr	s1, [x18], #4
	WORD $0xbd400002 // FMOVS (R0), F2;	ldr	s2, [x0]
	WORD $0xf1000442 // SUBS $1, R2, R2;	subs	x2, x2, #1
	WORD $0x1f000821 // FMADDS F0, F2, F1, F1;	fmadd	s1, s1, s0, s2
	WORD $0xbc004401 // FMOVS.P F1, 4(R0);	str	s1, [x0], #4
	BNE  LBB1_16     // b.ne	.LBB1_16
	WORD $0x910005ef // ADD $1, R15, R15;	add	x15, x15, #1
	WORD $0x8b0a0231 // ADD R10, R17, R17;	add	x17, x17, x10
	WORD $0xeb0901ff // CMP R9, R15;	cmp	x15, x9
	BNE  LBB1_15     // b.ne	.LBB1_15
	WORD $0x9100058c // ADD $1, R12, R12;	add	x12, x12, #1
	WORD $0x8b0a01ad // ADD R10, R13, R13;	add	x13, x13, x10
	WORD $0xeb08019f // CMP R8, R12;	cmp	x12, x8
	BNE  LBB1_14     // b.ne	.LBB1_14
	JMP  LBB1_25     // b	.LBB1_25

LBB1_19:
	WORD $0xaa1f03eb // MOVD ZR, R11;	mov	x11, xzr
	WORD $0xd37ef54c // LSL $2, R10, R12;	lsl	x12, x10, #2

LBB1_20:
	WORD $0x9b097d6e // MUL R9, R11, R14;	mul	x14, x11, x9
	WORD $0xaa1f03ed // MOVD ZR, R13;	mov	x13, xzr
	WORD $0xaa0203ef // MOVD R2, R15;	mov	x15, x2

LBB1_21:
	WORD $0x8b0e01b0 // ADD R14, R13, R16;	add	x16, x13, x14
	WORD $0xaa0f03f1 // MOVD R15, R17;	mov	x17, x15
	WORD $0x52800072 // MOVW $3, R18;	mov	w18, #3
	WORD $0x8b100830 // ADD R16<<2, R1, R16;	add	x16, x1, x16, lsl #2
	WORD $0x4d40ca00 // VLD1R (R16), [V0.S4];	ld1r	{ v0.4s }, [x16]
	WORD $0xaa0003f0 // MOVD R0, R16;	mov	x16, x0

LBB1_22:
	WORD $0x3dc00201 // FMOVQ (R16), F1;	ldr	q1, [x16]
	WORD $0x91001252 // ADD $4, R18, R18;	add	x18, x18, #4
	WORD $0x3cc10622 // FMOVQ.P 16(R17), F2;	ldr	q2, [x17], #16
	WORD $0xeb0a025f // CMP R10, R18;	cmp	x18, x10
	WORD $0x4e20cc41 // VFMLA V0.S4, V2.S4, V1.S4;	fmla	v1.4s, v2.4s, v0.4s
	WORD $0x3c810601 // FMOVQ.P F1, 16(R16);	str	q1, [x16], #16
	BCC  LBB1_22     // b.lo	.LBB1_22
	WORD $0x910005ad // ADD $1, R13, R13;	add	x13, x13, #1
	WORD $0x8b0c01ef // ADD R12, R15, R15;	add	x15, x15, x12
	WORD $0xeb0901bf // CMP R9, R13;	cmp	x13, x9
	BNE  LBB1_21     // b.ne	.LBB1_21
	WORD $0x9100056b // ADD $1, R11, R11;	add	x11, x11, #1
	WORD $0x8b0c0000 // ADD R12, R0, R0;	add	x0, x0, x12
	WORD $0xeb08017f // CMP R8, R11;	cmp	x11, x8
	BNE  LBB1_20     // b.ne	.LBB1_20

LBB1_25:
	NOP // ldp	x29, x30, [sp], #16
	RET // ret
