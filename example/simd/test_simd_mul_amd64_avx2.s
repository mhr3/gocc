//go:build !noasm && amd64
// Code generated by gocc -- DO NOT EDIT.

#include "textflag.h"

DATA LCPI0_0<>+0x00(SB)/2, $0x00ff
DATA LCPI0_0<>+0x02(SB)/2, $0x00ff
DATA LCPI0_0<>+0x04(SB)/2, $0x00ff
DATA LCPI0_0<>+0x06(SB)/2, $0x00ff
DATA LCPI0_0<>+0x08(SB)/2, $0x00ff
DATA LCPI0_0<>+0x0a(SB)/2, $0x00ff
DATA LCPI0_0<>+0x0c(SB)/2, $0x00ff
DATA LCPI0_0<>+0x0e(SB)/2, $0x00ff
DATA LCPI0_0<>+0x10(SB)/2, $0x00ff
DATA LCPI0_0<>+0x12(SB)/2, $0x00ff
DATA LCPI0_0<>+0x14(SB)/2, $0x00ff
DATA LCPI0_0<>+0x16(SB)/2, $0x00ff
DATA LCPI0_0<>+0x18(SB)/2, $0x00ff
DATA LCPI0_0<>+0x1a(SB)/2, $0x00ff
DATA LCPI0_0<>+0x1c(SB)/2, $0x00ff
DATA LCPI0_0<>+0x1e(SB)/2, $0x00ff
GLOBL LCPI0_0<>(SB), (RODATA|NOPTR), $32

TEXT ·uint8_simd_mul_avx2(SB), NOSPLIT, $0-32
	MOVQ input1+0(FP), DI
	MOVQ input2+8(FP), SI
	MOVQ output+16(FP), DX
	MOVQ size+24(FP), CX
	NOP                      // <--                                  // push	rbp
	NOP                      // <--                                  // mov	rbp, rsp
	NOP                      // <--                                  // and	rsp, -8
	WORD $0xc985             // TESTL CX, CX                         // test	ecx, ecx
	JLE  LBB0_18             // <--                                  // jle	.LBB0_18
	MOVL CX, R8              // <--                                  // mov	r8d, ecx
	LONG $0x10f88349         // CMPQ $0x10, R8                       // cmp	r8, 16
	JAE  LBB0_3              // <--                                  // jae	.LBB0_3
	WORD $0x3145; BYTE $0xc9 // XORL R9, R9                          // xor	r9d, r9d

LBB0_14:
	WORD $0x2944; BYTE $0xc9 // SUBL R9, CX                          // sub	ecx, r9d
	MOVQ R9, R10             // <--                                  // mov	r10, r9
	WORD $0xf749; BYTE $0xd2 // NOTQ R10                             // not	r10
	WORD $0x014d; BYTE $0xc2 // ADDQ R8, R10                         // add	r10, r8
	LONG $0x03e18348         // ANDQ $0x3, CX                        // and	rcx, 3
	JE   LBB0_16             // <--                                  // je	.LBB0_16

LBB0_15:
	MOVZX 0(SI)(R9*1), AX     // <--                                  // movzx	eax, byte ptr [rsi + r9]
	LONG  $0x0f24f642         // MULB 0(DI)(R9*1)                     // mul	byte ptr [rdi + r9]
	MOVB  AL, 0(DX)(R9*1)     // <--                                  // mov	byte ptr [rdx + r9], al
	WORD  $0xff49; BYTE $0xc1 // INCQ R9                              // inc	r9
	WORD  $0xff48; BYTE $0xc9 // DECQ CX                              // dec	rcx
	JNE   LBB0_15             // <--                                  // jne	.LBB0_15

LBB0_16:
	LONG $0x03fa8349 // CMPQ $0x3, R10                       // cmp	r10, 3
	JB   LBB0_18     // <--                                  // jb	.LBB0_18

LBB0_17:
	MOVZX 0(SI)(R9*1), AX         // <--                                  // movzx	eax, byte ptr [rsi + r9]
	LONG  $0x0f24f642             // MULB 0(DI)(R9*1)                     // mul	byte ptr [rdi + r9]
	MOVB  AL, 0(DX)(R9*1)         // <--                                  // mov	byte ptr [rdx + r9], al
	MOVZX 0x1(SI)(R9*1), AX       // <--                                  // movzx	eax, byte ptr [rsi + r9 + 1]
	LONG  $0x0f64f642; BYTE $0x01 // MULB 0x1(DI)(R9*1)                   // mul	byte ptr [rdi + r9 + 1]
	MOVB  AL, 0x1(DX)(R9*1)       // <--                                  // mov	byte ptr [rdx + r9 + 1], al
	MOVZX 0x2(SI)(R9*1), AX       // <--                                  // movzx	eax, byte ptr [rsi + r9 + 2]
	LONG  $0x0f64f642; BYTE $0x02 // MULB 0x2(DI)(R9*1)                   // mul	byte ptr [rdi + r9 + 2]
	MOVB  AL, 0x2(DX)(R9*1)       // <--                                  // mov	byte ptr [rdx + r9 + 2], al
	MOVZX 0x3(SI)(R9*1), AX       // <--                                  // movzx	eax, byte ptr [rsi + r9 + 3]
	LONG  $0x0f64f642; BYTE $0x03 // MULB 0x3(DI)(R9*1)                   // mul	byte ptr [rdi + r9 + 3]
	MOVB  AL, 0x3(DX)(R9*1)       // <--                                  // mov	byte ptr [rdx + r9 + 3], al
	LONG  $0x04c18349             // ADDQ $0x4, R9                        // add	r9, 4
	WORD  $0x394d; BYTE $0xc8     // CMPQ R9, R8                          // cmp	r8, r9
	JNE   LBB0_17                 // <--                                  // jne	.LBB0_17

LBB0_18:
	NOP        // <--                                  // mov	rsp, rbp
	NOP        // <--                                  // pop	rbp
	VZEROUPPER // <--                                  // vzeroupper
	RET        // <--                                  // ret

LBB0_3:
	MOVQ DX, AX                                // <--                                  // mov	rax, rdx
	WORD $0x2948; BYTE $0xf8                   // SUBQ DI, AX                          // sub	rax, rdi
	WORD $0x3145; BYTE $0xc9                   // XORL R9, R9                          // xor	r9d, r9d
	LONG $0x00803d48; WORD $0x0000             // CMPQ $0x80, AX                       // cmp	rax, 128
	JB   LBB0_14                               // <--                                  // jb	.LBB0_14
	MOVQ DX, AX                                // <--                                  // mov	rax, rdx
	WORD $0x2948; BYTE $0xf0                   // SUBQ SI, AX                          // sub	rax, rsi
	LONG $0x00803d48; WORD $0x0000             // CMPQ $0x80, AX                       // cmp	rax, 128
	JB   LBB0_14                               // <--                                  // jb	.LBB0_14
	LONG $0x80f88141; WORD $0x0000; BYTE $0x00 // CMPL $0x80, R8                       // cmp	r8d, 128
	JAE  LBB0_7                                // <--                                  // jae	.LBB0_7
	WORD $0x3145; BYTE $0xc9                   // XORL R9, R9                          // xor	r9d, r9d
	JMP  LBB0_11                               // <--                                  // jmp	.LBB0_11

LBB0_7:
	MOVL    CX, AX              // <--                                  // mov	eax, ecx
	WORD    $0xe083; BYTE $0x7f // ANDL $0x7f, AX                       // and	eax, 127
	MOVQ    R8, R9              // <--                                  // mov	r9, r8
	WORD    $0x2949; BYTE $0xc1 // SUBQ AX, R9                          // sub	r9, rax
	WORD    $0x3145; BYTE $0xd2 // XORL R10, R10                        // xor	r10d, r10d
	VMOVDQA LCPI0_0<>(SB), Y0   // <--                                  // vmovdqa	ymm0, ymmword ptr [rip + .LCPI0_0]

LBB0_8:
	LONG $0x6f7ea1c4; WORD $0x171c             // VMOVDQU 0(DI)(R10*1), X3             // vmovdqu	ymm3, ymmword ptr [rdi + r10]
	LONG $0x6f7ea1c4; WORD $0x1764; BYTE $0x20 // VMOVDQU 0x20(DI)(R10*1), X4          // vmovdqu	ymm4, ymmword ptr [rdi + r10 + 32]
	LONG $0x6f7ea1c4; WORD $0x176c; BYTE $0x40 // VMOVDQU 0x40(DI)(R10*1), X5          // vmovdqu	ymm5, ymmword ptr [rdi + r10 + 64]
	LONG $0x6f7ea1c4; WORD $0x174c; BYTE $0x60 // VMOVDQU 0x60(DI)(R10*1), X1          // vmovdqu	ymm1, ymmword ptr [rdi + r10 + 96]
	LONG $0x6f7ea1c4; WORD $0x1634             // VMOVDQU 0(SI)(R10*1), X6             // vmovdqu	ymm6, ymmword ptr [rsi + r10]
	LONG $0x6f7ea1c4; WORD $0x167c; BYTE $0x20 // VMOVDQU 0x20(SI)(R10*1), X7          // vmovdqu	ymm7, ymmword ptr [rsi + r10 + 32]
	LONG $0x6f7e21c4; WORD $0x1644; BYTE $0x40 // VMOVDQU 0x40(SI)(R10*1), X8          // vmovdqu	ymm8, ymmword ptr [rsi + r10 + 64]
	LONG $0x6f7ea1c4; WORD $0x1654; BYTE $0x60 // VMOVDQU 0x60(SI)(R10*1), X2          // vmovdqu	ymm2, ymmword ptr [rsi + r10 + 96]
	LONG $0xcb6865c5                           // ?                                    // vpunpckhbw	ymm9, ymm3, ymm3
	LONG $0xd6684dc5                           // ?                                    // vpunpckhbw	ymm10, ymm6, ymm6
	LONG $0xd52d41c4; BYTE $0xc9               // ?                                    // vpmullw	ymm9, ymm10, ymm9
	LONG $0xc8db35c5                           // FCMOVNE F0, F0                       // vpand	ymm9, ymm9, ymm0
	LONG $0xdb60e5c5                           // ?                                    // vpunpcklbw	ymm3, ymm3, ymm3
	LONG $0xf660cdc5                           // ?                                    // vpunpcklbw	ymm6, ymm6, ymm6
	LONG $0xdbd5cdc5                           // ?                                    // vpmullw	ymm3, ymm6, ymm3
	LONG $0xd8dbe5c5                           // FCMOVNU F0, F0                       // vpand	ymm3, ymm3, ymm0
	LONG $0x6765c1c4; BYTE $0xd9               // ?                                    // vpackuswb	ymm3, ymm3, ymm9
	LONG $0xf468ddc5                           // ?                                    // vpunpckhbw	ymm6, ymm4, ymm4
	LONG $0xcf6845c5                           // ?                                    // vpunpckhbw	ymm9, ymm7, ymm7
	LONG $0xf6d5b5c5                           // ?                                    // vpmullw	ymm6, ymm9, ymm6
	LONG $0xf0dbcdc5                           // FCOMI F0, F0                         // vpand	ymm6, ymm6, ymm0
	LONG $0xe460ddc5                           // ?                                    // vpunpcklbw	ymm4, ymm4, ymm4
	LONG $0xff60c5c5                           // ?                                    // vpunpcklbw	ymm7, ymm7, ymm7
	LONG $0xe4d5c5c5                           // ?                                    // vpmullw	ymm4, ymm7, ymm4
	LONG $0xe0dbddc5                           // ?                                    // vpand	ymm4, ymm4, ymm0
	LONG $0xe667ddc5                           // ?                                    // vpackuswb	ymm4, ymm4, ymm6
	LONG $0xf568d5c5                           // ?                                    // vpunpckhbw	ymm6, ymm5, ymm5
	LONG $0x683dc1c4; BYTE $0xf8               // ?                                    // vpunpckhbw	ymm7, ymm8, ymm8
	LONG $0xf6d5c5c5                           // ?                                    // vpmullw	ymm6, ymm7, ymm6
	LONG $0xf0dbcdc5                           // FCOMI F0, F0                         // vpand	ymm6, ymm6, ymm0
	LONG $0xed60d5c5                           // ?                                    // vpunpcklbw	ymm5, ymm5, ymm5
	LONG $0x603dc1c4; BYTE $0xf8               // ?                                    // vpunpcklbw	ymm7, ymm8, ymm8
	LONG $0xedd5c5c5                           // ?                                    // vpmullw	ymm5, ymm7, ymm5
	LONG $0xe8dbd5c5                           // FUCOMI F0, F0                        // vpand	ymm5, ymm5, ymm0
	LONG $0xee67d5c5                           // OUTL AL, DX                          // vpackuswb	ymm5, ymm5, ymm6
	LONG $0xf168f5c5                           // ?                                    // vpunpckhbw	ymm6, ymm1, ymm1
	LONG $0xfa68edc5                           // ?                                    // vpunpckhbw	ymm7, ymm2, ymm2
	LONG $0xf6d5c5c5                           // ?                                    // vpmullw	ymm6, ymm7, ymm6
	LONG $0xf0dbcdc5                           // FCOMI F0, F0                         // vpand	ymm6, ymm6, ymm0
	LONG $0xc960f5c5                           // ?                                    // vpunpcklbw	ymm1, ymm1, ymm1
	LONG $0xd260edc5                           // ?                                    // vpunpcklbw	ymm2, ymm2, ymm2
	LONG $0xc9d5edc5                           // ?                                    // vpmullw	ymm1, ymm2, ymm1
	LONG $0xc8dbf5c5                           // FCMOVNE F0, F0                       // vpand	ymm1, ymm1, ymm0
	LONG $0xce67f5c5                           // ?                                    // vpackuswb	ymm1, ymm1, ymm6
	LONG $0x7f7ea1c4; WORD $0x121c             // VMOVDQU X3, 0(DX)(R10*1)             // vmovdqu	ymmword ptr [rdx + r10], ymm3
	LONG $0x7f7ea1c4; WORD $0x1264; BYTE $0x20 // VMOVDQU X4, 0x20(DX)(R10*1)          // vmovdqu	ymmword ptr [rdx + r10 + 32], ymm4
	LONG $0x7f7ea1c4; WORD $0x126c; BYTE $0x40 // VMOVDQU X5, 0x40(DX)(R10*1)          // vmovdqu	ymmword ptr [rdx + r10 + 64], ymm5
	LONG $0x7f7ea1c4; WORD $0x124c; BYTE $0x60 // VMOVDQU X1, 0x60(DX)(R10*1)          // vmovdqu	ymmword ptr [rdx + r10 + 96], ymm1
	LONG $0x80ea8349                           // SUBQ $-0x80, R10                     // sub	r10, -128
	WORD $0x394d; BYTE $0xd1                   // CMPQ R10, R9                         // cmp	r9, r10
	JNE  LBB0_8                                // <--                                  // jne	.LBB0_8
	WORD $0x8548; BYTE $0xc0                   // TESTQ AX, AX                         // test	rax, rax
	JE   LBB0_18                               // <--                                  // je	.LBB0_18
	WORD $0xf883; BYTE $0x10                   // CMPL $0x10, AX                       // cmp	eax, 16
	JB   LBB0_14                               // <--                                  // jb	.LBB0_14

LBB0_11:
	MOVQ    R9, AX              // <--                                  // mov	rax, r9
	MOVL    CX, R10             // <--                                  // mov	r10d, ecx
	LONG    $0x0fe28341         // ANDL $0xf, R10                       // and	r10d, 15
	MOVQ    R8, R9              // <--                                  // mov	r9, r8
	WORD    $0x294d; BYTE $0xd1 // SUBQ R10, R9                         // sub	r9, r10
	VMOVDQA LCPI0_0<>(SB), Y0   // <--                                  // vmovdqa	ymm0, ymmword ptr [rip + .LCPI0_0]

LBB0_12:
	LONG    $0x307de2c4; WORD $0x070c // XORB CL, 0(DI)(AX*1)                 // vpmovzxbw	ymm1, xmmword ptr [rdi + rax]
	LONG    $0x307de2c4; WORD $0x0614 // XORB DL, 0(SI)(AX*1)                 // vpmovzxbw	ymm2, xmmword ptr [rsi + rax]
	LONG    $0xc9d5edc5               // ?                                    // vpmullw	ymm1, ymm2, ymm1
	LONG    $0xc8dbf5c5               // FCMOVNE F0, F0                       // vpand	ymm1, ymm1, ymm0
	LONG    $0x397de3c4; WORD $0x01ca // ?                                    // vextracti128	xmm2, ymm1, 1
	LONG    $0xca67f1c5               // ?                                    // vpackuswb	xmm1, xmm1, xmm2
	VMOVDQU X1, 0(DX)(AX*1)           // <--                                  // vmovdqu	xmmword ptr [rdx + rax], xmm1
	LONG    $0x10c08348               // ADDQ $0x10, AX                       // add	rax, 16
	WORD    $0x3949; BYTE $0xc1       // CMPQ AX, R9                          // cmp	r9, rax
	JNE     LBB0_12                   // <--                                  // jne	.LBB0_12
	WORD    $0x854d; BYTE $0xd2       // TESTQ R10, R10                       // test	r10, r10
	JNE     LBB0_14                   // <--                                  // jne	.LBB0_14
	JMP     LBB0_18                   // <--                                  // jmp	.LBB0_18
