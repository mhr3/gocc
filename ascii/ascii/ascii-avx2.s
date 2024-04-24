//go:build !noasm && amd64
// Code generated by gocc -- DO NOT EDIT.

#include "textflag.h"

TEXT ·isAsciiAvx(SB), NOSPLIT, $0-17
	MOVQ src+0(FP), DI
	MOVQ src_len+8(FP), SI
	NOP                      // (skipped)                            // push	rbp
	NOP                      // (skipped)                            // mov	rbp, rsp
	NOP                      // (skipped)                            // and	rsp, -8
	XORL CX, CX              // <--                                  // xor	ecx, ecx
	CMPQ SI, $0x20           // <--                                  // cmp	rsi, 32
	JB   LBB0_1              // <--                                  // jb	.LBB0_1
	LEAQ -0x20(SI), AX       // <--                                  // lea	rax, [rsi - 32]
	MOVQ AX, R9              // <--                                  // mov	r9, rax
	SHRQ $0x5, R9            // <--                                  // shr	r9, 5
	INCQ R9                  // <--                                  // inc	r9
	WORD $0x8944; BYTE $0xca // MOVL R9, DX                          // mov	edx, r9d
	WORD $0xe283; BYTE $0x07 // ANDL $0x7, DX                        // and	edx, 7
	CMPQ AX, $0xe0           // <--                                  // cmp	rax, 224
	JAE  LBB0_4              // <--                                  // jae	.LBB0_4
	LONG $0xc0eff9c5         // ?                                    // vpxor	xmm0, xmm0, xmm0
	XORL R8, R8              // <--                                  // xor	r8d, r8d
	JMP  LBB0_6              // <--                                  // jmp	.LBB0_6

LBB0_1:
	LONG $0xc0eff9c5 // ?                                    // vpxor	xmm0, xmm0, xmm0
	XORL AX, AX      // <--                                  // xor	eax, eax
	CMPQ AX, SI      // <--                                  // cmp	rax, rsi
	JB   LBB0_11     // <--                                  // jb	.LBB0_11
	JMP  LBB0_29     // <--                                  // jmp	.LBB0_29

LBB0_4:
	ANDQ $-0x8, R9   // <--                                  // and	r9, -8
	LONG $0xc0eff9c5 // ?                                    // vpxor	xmm0, xmm0, xmm0
	XORL R8, R8      // <--                                  // xor	r8d, r8d

LBB0_5:
	LONG $0xeb7da1c4; WORD $0x0704             // ?                                    // vpor	ymm0, ymm0, ymmword ptr [rdi + r8]
	LONG $0xeb7da1c4; WORD $0x0744; BYTE $0x20 // ?                                    // vpor	ymm0, ymm0, ymmword ptr [rdi + r8 + 32]
	LONG $0xeb7da1c4; WORD $0x0744; BYTE $0x40 // ?                                    // vpor	ymm0, ymm0, ymmword ptr [rdi + r8 + 64]
	LONG $0xeb7da1c4; WORD $0x0744; BYTE $0x60 // ?                                    // vpor	ymm0, ymm0, ymmword ptr [rdi + r8 + 96]
	QUAD $0x00800784eb7da1c4; WORD $0x0000     // ?                                    // vpor	ymm0, ymm0, ymmword ptr [rdi + r8 + 128]
	QUAD $0x00a00784eb7da1c4; WORD $0x0000     // ?                                    // vpor	ymm0, ymm0, ymmword ptr [rdi + r8 + 160]
	QUAD $0x00c00784eb7da1c4; WORD $0x0000     // ?                                    // vpor	ymm0, ymm0, ymmword ptr [rdi + r8 + 192]
	QUAD $0x00e00784eb7da1c4; WORD $0x0000     // ?                                    // vpor	ymm0, ymm0, ymmword ptr [rdi + r8 + 224]
	ADDQ $0x100, R8                            // <--                                  // add	r8, 256
	ADDQ $-0x8, R9                             // <--                                  // add	r9, -8
	JNE  LBB0_5                                // <--                                  // jne	.LBB0_5

LBB0_6:
	ANDQ $-0x20, AX          // <--                                  // and	rax, -32
	WORD $0x8548; BYTE $0xd2 // TESTQ DX, DX                         // test	rdx, rdx
	JE   LBB0_9              // <--                                  // je	.LBB0_9
	ADDQ DI, R8              // <--                                  // add	r8, rdi
	SHLQ $0x5, DX            // <--                                  // shl	rdx, 5
	XORL R9, R9              // <--                                  // xor	r9d, r9d

LBB0_8:
	LONG $0xeb7d81c4; WORD $0x0804 // ?                                    // vpor	ymm0, ymm0, ymmword ptr [r8 + r9]
	ADDQ $0x20, R9                 // <--                                  // add	r9, 32
	CMPQ DX, R9                    // <--                                  // cmp	rdx, r9
	JNE  LBB0_8                    // <--                                  // jne	.LBB0_8

LBB0_9:
	ADDQ $0x20, AX // <--                                  // add	rax, 32
	CMPQ AX, SI    // <--                                  // cmp	rax, rsi
	JAE  LBB0_29   // <--                                  // jae	.LBB0_29

LBB0_11:
	MOVQ SI, CX    // <--                                  // mov	rcx, rsi
	SUBQ AX, CX    // <--                                  // sub	rcx, rax
	CMPQ CX, $0x10 // <--                                  // cmp	rcx, 16
	JAE  LBB0_13   // <--                                  // jae	.LBB0_13
	XORL DX, DX    // <--                                  // xor	edx, edx
	JMP  LBB0_27   // <--                                  // jmp	.LBB0_27

LBB0_13:
	CMPQ CX, $0x80 // <--                                  // cmp	rcx, 128
	JAE  LBB0_18   // <--                                  // jae	.LBB0_18
	XORL DX, DX    // <--                                  // xor	edx, edx
	XORL R9, R9    // <--                                  // xor	r9d, r9d
	JMP  LBB0_15   // <--                                  // jmp	.LBB0_15

LBB0_18:
	LEAQ -0x80(CX), R8 // <--                                  // lea	r8, [rcx - 128]
	MOVQ R8, DX        // <--                                  // mov	rdx, r8
	SHRQ $0x7, DX      // <--                                  // shr	rdx, 7
	INCQ DX            // <--                                  // inc	rdx
	CMPQ R8, $0x80     // <--                                  // cmp	r8, 128
	JAE  LBB0_20       // <--                                  // jae	.LBB0_20
	LONG $0xc9eff1c5   // ?                                    // vpxor	xmm1, xmm1, xmm1
	XORL R8, R8        // <--                                  // xor	r8d, r8d
	LONG $0xd2efe9c5   // ?                                    // vpxor	xmm2, xmm2, xmm2
	LONG $0xdbefe1c5   // ?                                    // vpxor	xmm3, xmm3, xmm3
	LONG $0xe4efd9c5   // ?                                    // vpxor	xmm4, xmm4, xmm4
	JMP  LBB0_22       // <--                                  // jmp	.LBB0_22

LBB0_20:
	MOVQ DX, R9           // <--                                  // mov	r9, rdx
	ANDQ $-0x2, R9        // <--                                  // and	r9, -2
	LEAQ 0(AX)(DI*1), R10 // <--                                  // lea	r10, [rax + rdi]
	ADDQ $0xe0, R10       // <--                                  // add	r10, 224
	LONG $0xc9eff1c5      // ?                                    // vpxor	xmm1, xmm1, xmm1
	XORL R8, R8           // <--                                  // xor	r8d, r8d
	LONG $0xd2efe9c5      // ?                                    // vpxor	xmm2, xmm2, xmm2
	LONG $0xdbefe1c5      // ?                                    // vpxor	xmm3, xmm3, xmm3
	LONG $0xe4efd9c5      // ?                                    // vpxor	xmm4, xmm4, xmm4

LBB0_21:
	QUAD $0xff20028ceb7581c4; WORD $0xffff     // ?                                    // vpor	ymm1, ymm1, ymmword ptr [r10 + r8 - 224]
	QUAD $0xff400294eb6d81c4; WORD $0xffff     // ?                                    // vpor	ymm2, ymm2, ymmword ptr [r10 + r8 - 192]
	QUAD $0xff60029ceb6581c4; WORD $0xffff     // ?                                    // vpor	ymm3, ymm3, ymmword ptr [r10 + r8 - 160]
	LONG $0xeb5d81c4; WORD $0x0264; BYTE $0x80 // ?                                    // vpor	ymm4, ymm4, ymmword ptr [r10 + r8 - 128]
	LONG $0xeb7581c4; WORD $0x024c; BYTE $0xa0 // ?                                    // vpor	ymm1, ymm1, ymmword ptr [r10 + r8 - 96]
	LONG $0xeb6d81c4; WORD $0x0254; BYTE $0xc0 // ?                                    // vpor	ymm2, ymm2, ymmword ptr [r10 + r8 - 64]
	LONG $0xeb6581c4; WORD $0x025c; BYTE $0xe0 // ?                                    // vpor	ymm3, ymm3, ymmword ptr [r10 + r8 - 32]
	LONG $0xeb5d81c4; WORD $0x0224             // ?                                    // vpor	ymm4, ymm4, ymmword ptr [r10 + r8]
	ADDQ $0x100, R8                            // <--                                  // add	r8, 256
	ADDQ $-0x2, R9                             // <--                                  // add	r9, -2
	JNE  LBB0_21                               // <--                                  // jne	.LBB0_21

LBB0_22:
	MOVQ CX, R9                                // <--                                  // mov	r9, rcx
	ANDQ $-0x80, R9                            // <--                                  // and	r9, -128
	WORD $0xc2f6; BYTE $0x01                   // TESTL $0x1, DL                       // test	dl, 1
	JE   LBB0_24                               // <--                                  // je	.LBB0_24
	ADDQ AX, R8                                // <--                                  // add	r8, rax
	LONG $0xeb75a1c4; WORD $0x070c             // ?                                    // vpor	ymm1, ymm1, ymmword ptr [rdi + r8]
	LONG $0xeb6da1c4; WORD $0x0754; BYTE $0x20 // ?                                    // vpor	ymm2, ymm2, ymmword ptr [rdi + r8 + 32]
	LONG $0xeb65a1c4; WORD $0x075c; BYTE $0x40 // ?                                    // vpor	ymm3, ymm3, ymmword ptr [rdi + r8 + 64]
	LONG $0xeb5da1c4; WORD $0x0764; BYTE $0x60 // ?                                    // vpor	ymm4, ymm4, ymmword ptr [rdi + r8 + 96]

LBB0_24:
	LONG $0xd4ebedc5               // JMP 0x1bf                            // vpor	ymm2, ymm2, ymm4
	LONG $0xcbebf5c5               // JMP 0x1ba                            // vpor	ymm1, ymm1, ymm3
	LONG $0xcaebf5c5               // JMP 0x1bd                            // vpor	ymm1, ymm1, ymm2
	LONG $0x397de3c4; WORD $0x01ca // ?                                    // vextracti128	xmm2, ymm1, 1
	LONG $0xcaebf1c5               // JMP 0x1c7                            // vpor	xmm1, xmm1, xmm2
	LONG $0xd170f9c5; BYTE $0xee   // ?                                    // vpshufd	xmm2, xmm1, 238
	LONG $0xcaebf1c5               // JMP 0x1c7                            // vpor	xmm1, xmm1, xmm2
	LONG $0xd170f9c5; BYTE $0x55   // ?                                    // vpshufd	xmm2, xmm1, 85
	LONG $0xcaebf1c5               // JMP 0x1c7                            // vpor	xmm1, xmm1, xmm2
	LONG $0xd172e9c5; BYTE $0x10   // ?                                    // vpsrld	xmm2, xmm1, 16
	LONG $0xcaebf1c5               // JMP 0x1c7                            // vpor	xmm1, xmm1, xmm2
	LONG $0xd171e9c5; BYTE $0x08   // ?                                    // vpsrlw	xmm2, xmm1, 8
	LONG $0xcaebf1c5               // JMP 0x1c7                            // vpor	xmm1, xmm1, xmm2
	LONG $0xca7ef9c5               // JLE 0x1ef                            // vmovd	edx, xmm1
	CMPQ CX, R9                    // <--                                  // cmp	rcx, r9
	JE   LBB0_28                   // <--                                  // je	.LBB0_28
	WORD $0xc1f6; BYTE $0x70       // TESTL $0x70, CL                      // test	cl, 112
	JE   LBB0_26                   // <--                                  // je	.LBB0_26

LBB0_15:
	MOVQ CX, R8              // <--                                  // mov	r8, rcx
	ANDQ $-0x10, R8          // <--                                  // and	r8, -16
	WORD $0xb60f; BYTE $0xd2 // MOVZX DL, DX                         // movzx	edx, dl
	LONG $0xca6ef9c5         // ?                                    // vmovd	xmm1, edx
	LEAQ 0(DI)(AX*1), DX     // <--                                  // lea	rdx, [rdi + rax]
	ADDQ R8, AX              // <--                                  // add	rax, r8

LBB0_16:
	LONG $0xeb71a1c4; WORD $0x0a0c // ?                                    // vpor	xmm1, xmm1, xmmword ptr [rdx + r9]
	ADDQ $0x10, R9                 // <--                                  // add	r9, 16
	CMPQ R8, R9                    // <--                                  // cmp	r8, r9
	JNE  LBB0_16                   // <--                                  // jne	.LBB0_16
	LONG $0xd170f9c5; BYTE $0xee   // ?                                    // vpshufd	xmm2, xmm1, 238
	LONG $0xcaebf1c5               // JMP 0x1c7                            // vpor	xmm1, xmm1, xmm2
	LONG $0xd170f9c5; BYTE $0x55   // ?                                    // vpshufd	xmm2, xmm1, 85
	LONG $0xcaebf1c5               // JMP 0x1c7                            // vpor	xmm1, xmm1, xmm2
	LONG $0xd172e9c5; BYTE $0x10   // ?                                    // vpsrld	xmm2, xmm1, 16
	LONG $0xcaebf1c5               // JMP 0x1c7                            // vpor	xmm1, xmm1, xmm2
	LONG $0xd171e9c5; BYTE $0x08   // ?                                    // vpsrlw	xmm2, xmm1, 8
	LONG $0xcaebf1c5               // JMP 0x1c7                            // vpor	xmm1, xmm1, xmm2
	LONG $0xca7ef9c5               // JLE 0x1ef                            // vmovd	edx, xmm1
	CMPQ CX, R8                    // <--                                  // cmp	rcx, r8
	JNE  LBB0_27                   // <--                                  // jne	.LBB0_27
	JMP  LBB0_28                   // <--                                  // jmp	.LBB0_28

LBB0_26:
	ADDQ R9, AX // <--                                  // add	rax, r9

LBB0_27:
	WORD $0x140a; BYTE $0x07 // ORB 0(DI)(AX*1), DL                  // or	dl, byte ptr [rdi + rax]
	INCQ AX                  // <--                                  // inc	rax
	CMPQ SI, AX              // <--                                  // cmp	rsi, rax
	JNE  LBB0_27             // <--                                  // jne	.LBB0_27

LBB0_28:
	WORD $0xe280; BYTE $0x80 // ANDL $0x80, DL                       // and	dl, -128
	WORD $0xb60f; BYTE $0xca // MOVZX DL, CX                         // movzx	ecx, dl

LBB0_29:
	LONG $0xc0d7fdc5         // ?                                    // vpmovmskb	eax, ymm0
	WORD $0xc809             // ORL CX, AX                           // or	eax, ecx
	WORD $0x940f; BYTE $0xc0 // SETE AL                              // sete	al
	NOP                      // (skipped)                            // mov	rsp, rbp
	NOP                      // (skipped)                            // pop	rbp
	VZEROUPPER               // <--                                  // vzeroupper
	MOVB AX, ret+16(FP)      // <--
	RET                      // <--                                  // ret
