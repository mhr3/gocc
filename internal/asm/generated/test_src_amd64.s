//go:build !noasm && amd64
// Code generated by gocc -- DO NOT EDIT.

#include "textflag.h"

TEXT ·Test_fn_4818_0(SB), NOSPLIT, $0-32
	MOVL  a+0(FP), DI
	MOVQ  b+8(FP), SI
	MOVB  c+16(FP), DX
	MOVQ  res+24(FP), CX
	NOP                       // (skipped)                            // push	rbp
	NOP                       // (skipped)                            // mov	rbp, rsp
	NOP                       // (skipped)                            // and	rsp, -8
	IMULL DX, DI              // <--                                  // imul	edi, edx
	WORD  $0x6348; BYTE $0xc7 // MOVSXD DI, AX                        // movsxd	rax, edi
	ADDQ  SI, AX              // <--                                  // add	rax, rsi
	MOVQ  AX, 0(CX)           // <--                                  // mov	qword ptr [rcx], rax
	NOP                       // (skipped)                            // mov	rsp, rbp
	NOP                       // (skipped)                            // pop	rbp
	RET                       // <--                                  // ret

TEXT ·Test_fn_111_0(SB), NOSPLIT, $0-3
	MOVB a+0(FP), DI
	MOVB b+1(FP), SI
	MOVB c+2(FP), DX
	NOP              // (skipped)                            // push	rbp
	NOP              // (skipped)                            // mov	rbp, rsp
	NOP              // (skipped)                            // and	rsp, -8
	NOP              // (skipped)                            // mov	rsp, rbp
	NOP              // (skipped)                            // pop	rbp
	RET              // <--                                  // ret

TEXT ·Test_fn_111_1(SB), NOSPLIT, $0-9
	MOVB a+0(FP), DI
	MOVB b+1(FP), SI
	MOVB c+2(FP), DX
	NOP                      // (skipped)                            // push	rbp
	NOP                      // (skipped)                            // mov	rbp, rsp
	NOP                      // (skipped)                            // and	rsp, -8
	MOVL DX, AX              // <--                                  // mov	eax, edx
	MULL DI                  // <--                                  // mul	dil
	WORD $0x0040; BYTE $0xf0 // ADDL SI, AL                          // add	al, sil
	NOP                      // (skipped)                            // mov	rsp, rbp
	NOP                      // (skipped)                            // pop	rbp
	MOVB AX, ret+8(FP)       // <--
	RET                      // <--                                  // ret

TEXT ·Test_fn_1114_1(SB), NOSPLIT, $0-9
	MOVB  a+0(FP), DI
	MOVB  b+1(FP), SI
	MOVB  c+2(FP), DX
	MOVL  d+4(FP), CX
	NOP                   // (skipped)                            // push	rbp
	NOP                   // (skipped)                            // mov	rbp, rsp
	NOP                   // (skipped)                            // and	rsp, -8
	IMULL DX, DI          // <--                                  // imul	edi, edx
	LEAL  0(SI)(CX*1), AX // <--                                  // lea	eax, [rsi + rcx]
	WORD  $0xf801         // ADDL DI, AX                          // add	eax, edi
	NOP                   // (skipped)                            // mov	rsp, rbp
	NOP                   // (skipped)                            // pop	rbp
	MOVB  AX, ret+8(FP)   // <--
	RET                   // <--                                  // ret

TEXT ·Test_fn_282_2(SB), NOSPLIT, $0-26
	MOVW  a+0(FP), DI
	MOVQ  b+8(FP), SI
	MOVW  c+16(FP), DX
	NOP                   // (skipped)                            // push	rbp
	NOP                   // (skipped)                            // mov	rbp, rsp
	NOP                   // (skipped)                            // and	rsp, -8
	IMULL DX, DI          // <--                                  // imul	edi, edx
	LEAL  0(SI)(DI*1), AX // <--                                  // lea	eax, [rsi + rdi]
	NOP                   // (skipped)                            // mov	rsp, rbp
	NOP                   // (skipped)                            // pop	rbp
	MOVW  AX, ret+24(FP)  // <--
	RET                   // <--                                  // ret

TEXT ·Test_fn_481_1(SB), NOSPLIT, $0-25
	MOVL  a+0(FP), DI
	MOVQ  b+8(FP), SI
	MOVB  c+16(FP), DX
	NOP                   // (skipped)                            // push	rbp
	NOP                   // (skipped)                            // mov	rbp, rsp
	NOP                   // (skipped)                            // and	rsp, -8
	IMULL DX, DI          // <--                                  // imul	edi, edx
	LEAL  0(SI)(DI*1), AX // <--                                  // lea	eax, [rsi + rdi]
	NOP                   // (skipped)                            // mov	rsp, rbp
	NOP                   // (skipped)                            // pop	rbp
	MOVB  AX, ret+24(FP)  // <--
	RET                   // <--                                  // ret

TEXT ·Test_fn_444_4(SB), NOSPLIT, $0-20
	MOVL  a+0(FP), DI
	MOVL  b+4(FP), SI
	MOVL  c+8(FP), DX
	NOP                   // (skipped)                            // push	rbp
	NOP                   // (skipped)                            // mov	rbp, rsp
	NOP                   // (skipped)                            // and	rsp, -8
	IMULL DX, DI          // <--                                  // imul	edi, edx
	LEAL  0(DI)(SI*1), AX // <--                                  // lea	eax, [rdi + rsi]
	NOP                   // (skipped)                            // mov	rsp, rbp
	NOP                   // (skipped)                            // pop	rbp
	MOVL  AX, ret+16(FP)  // <--
	RET                   // <--                                  // ret

TEXT ·Test_fn_888888_8(SB), NOSPLIT, $0-56
	MOVQ  a+0(FP), DI
	MOVQ  b+8(FP), SI
	MOVQ  c+16(FP), DX
	MOVQ  d+24(FP), CX
	MOVQ  e+32(FP), R8
	MOVQ  f+40(FP), R9
	NOP                   // (skipped)                            // push	rbp
	NOP                   // (skipped)                            // mov	rbp, rsp
	NOP                   // (skipped)                            // and	rsp, -8
	IMULQ DX, DI          // <--                                  // imul	rdi, rdx
	LEAQ  0(DI)(SI*1), AX // <--                                  // lea	rax, [rdi + rsi]
	IMULQ R9, CX          // <--                                  // imul	rcx, r9
	ADDQ  R8, AX          // <--                                  // add	rax, r8
	ADDQ  CX, AX          // <--                                  // add	rax, rcx
	NOP                   // (skipped)                            // mov	rsp, rbp
	NOP                   // (skipped)                            // pop	rbp
	MOVQ  AX, ret+48(FP)  // <--
	RET                   // <--                                  // ret

TEXT ·Test_fn_sq_floats(SB), NOSPLIT, $0-48
	MOVQ input+0(FP), DI
	MOVQ input_len+8(FP), SI
	MOVQ input_cap+16(FP), DX
	MOVQ output+24(FP), CX
	MOVQ output_len+32(FP), R8
	MOVQ output_cap+40(FP), R9
	NOP                        // (skipped)                            // push	rbp
	NOP                        // (skipped)                            // mov	rbp, rsp
	NOP                        // (skipped)                            // and	rsp, -8
	WORD $0x8548; BYTE $0xf6   // TESTQ SI, SI                         // test	rsi, rsi
	JLE  LBB8_14               // <--                                  // jle	.LBB8_14
	XORL AX, AX                // <--                                  // xor	eax, eax
	LONG $0x08fe8348           // CMPQ $0x8, SI                        // cmp	rsi, 8
	JB   LBB8_10               // <--                                  // jb	.LBB8_10
	MOVQ CX, DX                // <--                                  // mov	rdx, rcx
	SUBQ DI, DX                // <--                                  // sub	rdx, rdi
	LONG $0x20fa8348           // CMPQ $test_fn_111_0(SB), DX          // cmp	rdx, 32
	JB   LBB8_10               // <--                                  // jb	.LBB8_10
	LEAQ -0x8(SI), AX          // <--                                  // lea	rax, [rsi - 8]
	MOVQ AX, R8                // <--                                  // mov	r8, rax
	SHRQ $0x3, R8              // <--                                  // shr	r8, 3
	INCQ R8                    // <--                                  // inc	r8
	LONG $0x08f88348           // CMPQ $0x8, AX                        // cmp	rax, 8
	JAE  LBB8_5                // <--                                  // jae	.LBB8_5
	XORL DX, DX                // <--                                  // xor	edx, edx
	JMP  LBB8_7                // <--                                  // jmp	.LBB8_7

LBB8_5:
	MOVQ R8, AX    // <--                                  // mov	rax, r8
	ANDQ $-0x2, AX // <--                                  // and	rax, -2
	XORL DX, DX    // <--                                  // xor	edx, edx

LBB8_6:
	MOVUPS 0(DI)(DX*4), X0    // <--                                  // movups	xmm0, xmmword ptr [rdi + 4*rdx]
	MOVUPS 0x10(DI)(DX*4), X1 // <--                                  // movups	xmm1, xmmword ptr [rdi + 4*rdx + 16]
	MULPS  X0, X0             // <--                                  // mulps	xmm0, xmm0
	MULPS  X1, X1             // <--                                  // mulps	xmm1, xmm1
	MOVUPS X0, 0(CX)(DX*4)    // <--                                  // movups	xmmword ptr [rcx + 4*rdx], xmm0
	MOVUPS X1, 0x10(CX)(DX*4) // <--                                  // movups	xmmword ptr [rcx + 4*rdx + 16], xmm1
	MOVUPS 0x20(DI)(DX*4), X0 // <--                                  // movups	xmm0, xmmword ptr [rdi + 4*rdx + 32]
	MOVUPS 0x30(DI)(DX*4), X1 // <--                                  // movups	xmm1, xmmword ptr [rdi + 4*rdx + 48]
	MULPS  X0, X0             // <--                                  // mulps	xmm0, xmm0
	MULPS  X1, X1             // <--                                  // mulps	xmm1, xmm1
	MOVUPS X0, 0x20(CX)(DX*4) // <--                                  // movups	xmmword ptr [rcx + 4*rdx + 32], xmm0
	MOVUPS X1, 0x30(CX)(DX*4) // <--                                  // movups	xmmword ptr [rcx + 4*rdx + 48], xmm1
	ADDQ   $0x10, DX          // <--                                  // add	rdx, 16
	ADDQ   $-0x2, AX          // <--                                  // add	rax, -2
	JNE    LBB8_6             // <--                                  // jne	.LBB8_6

LBB8_7:
	MOVQ   SI, AX             // <--                                  // mov	rax, rsi
	ANDQ   $-0x8, AX          // <--                                  // and	rax, -8
	LONG   $0x01c0f641        // TESTL $0x1, R8                       // test	r8b, 1
	JE     LBB8_9             // <--                                  // je	.LBB8_9
	MOVUPS 0(DI)(DX*4), X0    // <--                                  // movups	xmm0, xmmword ptr [rdi + 4*rdx]
	MOVUPS 0x10(DI)(DX*4), X1 // <--                                  // movups	xmm1, xmmword ptr [rdi + 4*rdx + 16]
	MULPS  X0, X0             // <--                                  // mulps	xmm0, xmm0
	MULPS  X1, X1             // <--                                  // mulps	xmm1, xmm1
	MOVUPS X0, 0(CX)(DX*4)    // <--                                  // movups	xmmword ptr [rcx + 4*rdx], xmm0
	MOVUPS X1, 0x10(CX)(DX*4) // <--                                  // movups	xmmword ptr [rcx + 4*rdx + 16], xmm1

LBB8_9:
	WORD $0x3948; BYTE $0xf0 // CMPQ SI, AX                          // cmp	rax, rsi
	JE   LBB8_14             // <--                                  // je	.LBB8_14

LBB8_10:
	MOVQ AX, DX   // <--                                  // mov	rdx, rax
	NOTQ DX       // <--                                  // not	rdx
	ADDQ SI, DX   // <--                                  // add	rdx, rsi
	MOVQ SI, R8   // <--                                  // mov	r8, rsi
	ANDQ $0x3, R8 // <--                                  // and	r8, 3
	JE   LBB8_12  // <--                                  // je	.LBB8_12

LBB8_11:
	MOVSS 0(DI)(AX*4), X0 // <--                                  // movss	xmm0, dword ptr [rdi + 4*rax]
	MULSS X0, X0          // <--                                  // mulss	xmm0, xmm0
	MOVSS X0, 0(CX)(AX*4) // <--                                  // movss	dword ptr [rcx + 4*rax], xmm0
	INCQ  AX              // <--                                  // inc	rax
	DECQ  R8              // <--                                  // dec	r8
	JNE   LBB8_11         // <--                                  // jne	.LBB8_11

LBB8_12:
	LONG $0x03fa8348 // CMPQ $0x3, DX                        // cmp	rdx, 3
	JB   LBB8_14     // <--                                  // jb	.LBB8_14

LBB8_13:
	MOVSS 0(DI)(AX*4), X0     // <--                                  // movss	xmm0, dword ptr [rdi + 4*rax]
	MULSS X0, X0              // <--                                  // mulss	xmm0, xmm0
	MOVSS X0, 0(CX)(AX*4)     // <--                                  // movss	dword ptr [rcx + 4*rax], xmm0
	MOVSS 0x4(DI)(AX*4), X0   // <--                                  // movss	xmm0, dword ptr [rdi + 4*rax + 4]
	MULSS X0, X0              // <--                                  // mulss	xmm0, xmm0
	MOVSS X0, 0x4(CX)(AX*4)   // <--                                  // movss	dword ptr [rcx + 4*rax + 4], xmm0
	MOVSS 0x8(DI)(AX*4), X0   // <--                                  // movss	xmm0, dword ptr [rdi + 4*rax + 8]
	MULSS X0, X0              // <--                                  // mulss	xmm0, xmm0
	MOVSS X0, 0x8(CX)(AX*4)   // <--                                  // movss	dword ptr [rcx + 4*rax + 8], xmm0
	MOVSS 0xc(DI)(AX*4), X0   // <--                                  // movss	xmm0, dword ptr [rdi + 4*rax + 12]
	MULSS X0, X0              // <--                                  // mulss	xmm0, xmm0
	MOVSS X0, 0xc(CX)(AX*4)   // <--                                  // movss	dword ptr [rcx + 4*rax + 12], xmm0
	ADDQ  $0x4, AX            // <--                                  // add	rax, 4
	WORD  $0x3948; BYTE $0xc6 // CMPQ AX, SI                          // cmp	rsi, rax
	JNE   LBB8_13             // <--                                  // jne	.LBB8_13

LBB8_14:
	NOP // (skipped)                            // mov	rsp, rbp
	NOP // (skipped)                            // pop	rbp
	RET // <--                                  // ret
