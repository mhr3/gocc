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

TEXT ·uint8_simd_mul_avx512(SB), NOSPLIT, $0-32
	MOVQ input1+0(FP), DI
	MOVQ input2+8(FP), SI
	MOVQ output+16(FP), DX
	MOVQ size+24(FP), CX
	NOP                    // (skipped)                            // push	rbp
	NOP                    // (skipped)                            // mov	rbp, rsp
	NOP                    // (skipped)                            // and	rsp, -8
	WORD $0xc985           // TESTL CX, CX                         // test	ecx, ecx
	JLE  LBB0_18           // <--                                  // jle	.LBB0_18
	MOVL CX, R8            // <--                                  // mov	r8d, ecx
	LONG $0x20f88349       // CMPQ $0x20, R8                       // cmp	r8, 32
	JAE  LBB0_3            // <--                                  // jae	.LBB0_3
	XORL R9, R9            // <--                                  // xor	r9d, r9d

LBB0_14:
	WORD $0x2944; BYTE $0xc9 // SUBL R9, CX                          // sub	ecx, r9d
	MOVQ R9, R10             // <--                                  // mov	r10, r9
	NOTQ R10                 // <--                                  // not	r10
	ADDQ R8, R10             // <--                                  // add	r10, r8
	ANDQ $0x3, CX            // <--                                  // and	rcx, 3
	JE   LBB0_16             // <--                                  // je	.LBB0_16

LBB0_15:
	LONG $0x04b60f42; BYTE $0x0e // MOVZX 0(SI)(R9*1), AX                // movzx	eax, byte ptr [rsi + r9]
	MULB 0(DI)(R9*1)             // <--                                  // mul	byte ptr [rdi + r9]
	MOVB AL, 0(DX)(R9*1)         // <--                                  // mov	byte ptr [rdx + r9], al
	INCQ R9                      // <--                                  // inc	r9
	DECQ CX                      // <--                                  // dec	rcx
	JNE  LBB0_15                 // <--                                  // jne	.LBB0_15

LBB0_16:
	LONG $0x03fa8349 // CMPQ $0x3, R10                       // cmp	r10, 3
	JB   LBB0_18     // <--                                  // jb	.LBB0_18

LBB0_17:
	LONG $0x04b60f42; BYTE $0x0e   // MOVZX 0(SI)(R9*1), AX                // movzx	eax, byte ptr [rsi + r9]
	MULB 0(DI)(R9*1)               // <--                                  // mul	byte ptr [rdi + r9]
	MOVB AL, 0(DX)(R9*1)           // <--                                  // mov	byte ptr [rdx + r9], al
	LONG $0x44b60f42; WORD $0x010e // MOVZX 0x1(SI)(R9*1), AX              // movzx	eax, byte ptr [rsi + r9 + 1]
	MULB 0x1(DI)(R9*1)             // <--                                  // mul	byte ptr [rdi + r9 + 1]
	MOVB AL, 0x1(DX)(R9*1)         // <--                                  // mov	byte ptr [rdx + r9 + 1], al
	LONG $0x44b60f42; WORD $0x020e // MOVZX 0x2(SI)(R9*1), AX              // movzx	eax, byte ptr [rsi + r9 + 2]
	MULB 0x2(DI)(R9*1)             // <--                                  // mul	byte ptr [rdi + r9 + 2]
	MOVB AL, 0x2(DX)(R9*1)         // <--                                  // mov	byte ptr [rdx + r9 + 2], al
	LONG $0x44b60f42; WORD $0x030e // MOVZX 0x3(SI)(R9*1), AX              // movzx	eax, byte ptr [rsi + r9 + 3]
	MULB 0x3(DI)(R9*1)             // <--                                  // mul	byte ptr [rdi + r9 + 3]
	MOVB AL, 0x3(DX)(R9*1)         // <--                                  // mov	byte ptr [rdx + r9 + 3], al
	ADDQ $0x4, R9                  // <--                                  // add	r9, 4
	WORD $0x394d; BYTE $0xc8       // CMPQ R9, R8                          // cmp	r8, r9
	JNE  LBB0_17                   // <--                                  // jne	.LBB0_17

LBB0_18:
	NOP        // (skipped)                            // mov	rsp, rbp
	NOP        // (skipped)                            // pop	rbp
	VZEROUPPER // <--                                  // vzeroupper
	RET        // <--                                  // ret

LBB0_3:
	MOVQ DX, AX      // <--                                  // mov	rax, rdx
	SUBQ DI, AX      // <--                                  // sub	rax, rdi
	XORL R9, R9      // <--                                  // xor	r9d, r9d
	LONG $0x40f88348 // CMPQ $0x40, AX                       // cmp	rax, 64
	JB   LBB0_14     // <--                                  // jb	.LBB0_14
	MOVQ DX, AX      // <--                                  // mov	rax, rdx
	SUBQ SI, AX      // <--                                  // sub	rax, rsi
	LONG $0x40f88348 // CMPQ $0x40, AX                       // cmp	rax, 64
	JB   LBB0_14     // <--                                  // jb	.LBB0_14
	LONG $0x40f88341 // CMPL $0x40, R8                       // cmp	r8d, 64
	JAE  LBB0_7      // <--                                  // jae	.LBB0_7
	XORL R9, R9      // <--                                  // xor	r9d, r9d
	JMP  LBB0_11     // <--                                  // jmp	.LBB0_11

LBB0_7:
	MOVL    CX, AX            // <--                                  // mov	eax, ecx
	ANDL    $0x3f, AX         // <--                                  // and	eax, 63
	MOVQ    R8, R9            // <--                                  // mov	r9, r8
	SUBQ    AX, R9            // <--                                  // sub	r9, rax
	XORL    R10, R10          // <--                                  // xor	r10d, r10d
	VMOVDQA LCPI0_0<>(SB), Y0 // <--                                  // vmovdqa	ymm0, ymmword ptr [rip + .LCPI0_0]

LBB0_8:
	LONG $0x6f7ea1c4; WORD $0x170c             // VMOVDQU 0(DI)(R10*1), X1             // vmovdqu	ymm1, ymmword ptr [rdi + r10]
	LONG $0x6f7ea1c4; WORD $0x1754; BYTE $0x20 // VMOVDQU 0x20(DI)(R10*1), X2          // vmovdqu	ymm2, ymmword ptr [rdi + r10 + 32]
	LONG $0xd968f5c5                           // ?                                    // vpunpckhbw	ymm3, ymm1, ymm1
	LONG $0x6f7ea1c4; WORD $0x1624             // VMOVDQU 0(SI)(R10*1), X4             // vmovdqu	ymm4, ymmword ptr [rsi + r10]
	LONG $0x6f7ea1c4; WORD $0x166c; BYTE $0x20 // VMOVDQU 0x20(SI)(R10*1), X5          // vmovdqu	ymm5, ymmword ptr [rsi + r10 + 32]
	LONG $0xf468ddc5                           // ?                                    // vpunpckhbw	ymm6, ymm4, ymm4
	LONG $0xdbd5cdc5                           // ?                                    // vpmullw	ymm3, ymm6, ymm3
	LONG $0xd8dbe5c5                           // FCMOVNU F0, F0                       // vpand	ymm3, ymm3, ymm0
	LONG $0xc960f5c5                           // ?                                    // vpunpcklbw	ymm1, ymm1, ymm1
	LONG $0xe460ddc5                           // ?                                    // vpunpcklbw	ymm4, ymm4, ymm4
	LONG $0xc9d5ddc5                           // ?                                    // vpmullw	ymm1, ymm4, ymm1
	LONG $0xc8dbf5c5                           // FCMOVNE F0, F0                       // vpand	ymm1, ymm1, ymm0
	LONG $0xcb67f5c5                           // LRET                                 // vpackuswb	ymm1, ymm1, ymm3
	LONG $0xda68edc5                           // ?                                    // vpunpckhbw	ymm3, ymm2, ymm2
	LONG $0xe568d5c5                           // ?                                    // vpunpckhbw	ymm4, ymm5, ymm5
	LONG $0xdbd5ddc5                           // ?                                    // vpmullw	ymm3, ymm4, ymm3
	LONG $0xd8dbe5c5                           // FCMOVNU F0, F0                       // vpand	ymm3, ymm3, ymm0
	LONG $0xd260edc5                           // ?                                    // vpunpcklbw	ymm2, ymm2, ymm2
	LONG $0xe560d5c5                           // ?                                    // vpunpcklbw	ymm4, ymm5, ymm5
	LONG $0xd2d5ddc5                           // ?                                    // vpmullw	ymm2, ymm4, ymm2
	LONG $0xd0dbedc5                           // FCMOVNBE F0, F0                      // vpand	ymm2, ymm2, ymm0
	LONG $0xd367edc5                           // ?                                    // vpackuswb	ymm2, ymm2, ymm3
	LONG $0x7f7ea1c4; WORD $0x1254; BYTE $0x20 // VMOVDQU X2, 0x20(DX)(R10*1)          // vmovdqu	ymmword ptr [rdx + r10 + 32], ymm2
	LONG $0x7f7ea1c4; WORD $0x120c             // VMOVDQU X1, 0(DX)(R10*1)             // vmovdqu	ymmword ptr [rdx + r10], ymm1
	ADDQ $0x40, R10                            // <--                                  // add	r10, 64
	WORD $0x394d; BYTE $0xd1                   // CMPQ R10, R9                         // cmp	r9, r10
	JNE  LBB0_8                                // <--                                  // jne	.LBB0_8
	WORD $0x8548; BYTE $0xc0                   // TESTQ AX, AX                         // test	rax, rax
	JE   LBB0_18                               // <--                                  // je	.LBB0_18
	WORD $0xf883; BYTE $0x20                   // CMPL $0x20, AX                       // cmp	eax, 32
	JB   LBB0_14                               // <--                                  // jb	.LBB0_14

LBB0_11:
	MOVQ    R9, AX            // <--                                  // mov	rax, r9
	MOVL    CX, R10           // <--                                  // mov	r10d, ecx
	ANDL    $0x1f, R10        // <--                                  // and	r10d, 31
	MOVQ    R8, R9            // <--                                  // mov	r9, r8
	SUBQ    R10, R9           // <--                                  // sub	r9, r10
	VMOVDQA LCPI0_0<>(SB), Y0 // <--                                  // vmovdqa	ymm0, ymmword ptr [rip + .LCPI0_0]

LBB0_12:
	LONG $0x0c6ffec5; BYTE $0x07 // VMOVDQU 0(DI)(AX*1), X1              // vmovdqu	ymm1, ymmword ptr [rdi + rax]
	LONG $0x146ffec5; BYTE $0x06 // VMOVDQU 0(SI)(AX*1), X2              // vmovdqu	ymm2, ymmword ptr [rsi + rax]
	LONG $0xd968f5c5             // ?                                    // vpunpckhbw	ymm3, ymm1, ymm1
	LONG $0xe268edc5             // ?                                    // vpunpckhbw	ymm4, ymm2, ymm2
	LONG $0xdbd5ddc5             // ?                                    // vpmullw	ymm3, ymm4, ymm3
	LONG $0xd8dbe5c5             // FCMOVNU F0, F0                       // vpand	ymm3, ymm3, ymm0
	LONG $0xc960f5c5             // ?                                    // vpunpcklbw	ymm1, ymm1, ymm1
	LONG $0xd260edc5             // ?                                    // vpunpcklbw	ymm2, ymm2, ymm2
	LONG $0xc9d5edc5             // ?                                    // vpmullw	ymm1, ymm2, ymm1
	LONG $0xc8dbf5c5             // FCMOVNE F0, F0                       // vpand	ymm1, ymm1, ymm0
	LONG $0xcb67f5c5             // LRET                                 // vpackuswb	ymm1, ymm1, ymm3
	LONG $0x0c7ffec5; BYTE $0x02 // VMOVDQU X1, 0(DX)(AX*1)              // vmovdqu	ymmword ptr [rdx + rax], ymm1
	ADDQ $0x20, AX               // <--                                  // add	rax, 32
	WORD $0x3949; BYTE $0xc1     // CMPQ AX, R9                          // cmp	r9, rax
	JNE  LBB0_12                 // <--                                  // jne	.LBB0_12
	WORD $0x854d; BYTE $0xd2     // TESTQ R10, R10                       // test	r10, r10
	JNE  LBB0_14                 // <--                                  // jne	.LBB0_14
	JMP  LBB0_18                 // <--                                  // jmp	.LBB0_18
