//go:build !noasm && amd64
// Code generated by gocc devel -- DO NOT EDIT.
//
// Source file         : test_src.c
// Clang version       : Apple clang version 16.0.0 (clang-1600.0.26.3)
// Target architecture : amd64
// Compiler options    : [none]

#include "textflag.h"

TEXT ·Test_fn_4818_0(SB), NOSPLIT, $0-32
	MOVLQZX a+0(FP), DI
	MOVQ    b+8(FP), SI
	MOVBQZX c+16(FP), DX
	MOVQ    res+24(FP), CX
	NOP                         // (skipped)                            // push	rbp
	NOP                         // (skipped)                            // mov	rbp, rsp
	NOP                         // (skipped)                            // and	rsp, -8
	WORD    $0xaf0f; BYTE $0xfa // IMULL DX, DI                         // imul	edi, edx
	WORD    $0x6348; BYTE $0xc7 // MOVSXD DI, AX                        // movsxd	rax, edi
	ADDQ    SI, AX              // <--                                  // add	rax, rsi
	MOVQ    AX, 0(CX)           // <--                                  // mov	qword ptr [rcx], rax
	NOP                         // (skipped)                            // mov	rsp, rbp
	NOP                         // (skipped)                            // pop	rbp
	RET                         // <--                                  // ret

TEXT ·Test_fn_111_0(SB), NOSPLIT, $0-3
	MOVBQZX a+0(FP), DI
	MOVBQZX b+1(FP), SI
	MOVBQZX c+2(FP), DX
	NOP                 // (skipped)                            // push	rbp
	NOP                 // (skipped)                            // mov	rbp, rsp
	NOP                 // (skipped)                            // and	rsp, -8
	NOP                 // (skipped)                            // mov	rsp, rbp
	NOP                 // (skipped)                            // pop	rbp
	RET                 // <--                                  // ret

TEXT ·Test_fn_111_1(SB), NOSPLIT, $0-9
	MOVBQZX a+0(FP), DI
	MOVBQZX b+1(FP), SI
	MOVBQZX c+2(FP), DX
	NOP                         // (skipped)                            // push	rbp
	NOP                         // (skipped)                            // mov	rbp, rsp
	NOP                         // (skipped)                            // and	rsp, -8
	WORD    $0xd089             // MOVL DX, AX                          // mov	eax, edx
	WORD    $0xf640; BYTE $0xe7 // MULL DI                              // mul	dil
	WORD    $0x0040; BYTE $0xf0 // ADDL SI, AL                          // add	al, sil
	NOP                         // (skipped)                            // mov	rsp, rbp
	NOP                         // (skipped)                            // pop	rbp
	MOVB    AX, ret+8(FP)       // <--
	RET                         // <--                                  // ret

TEXT ·Test_fn_1114_1(SB), NOSPLIT, $0-9
	MOVBQZX a+0(FP), DI
	MOVBQZX b+1(FP), SI
	MOVBQZX c+2(FP), DX
	MOVLQZX d+4(FP), CX
	NOP                         // (skipped)                            // push	rbp
	NOP                         // (skipped)                            // mov	rbp, rsp
	NOP                         // (skipped)                            // and	rsp, -8
	WORD    $0xaf0f; BYTE $0xfa // IMULL DX, DI                         // imul	edi, edx
	WORD    $0x048d; BYTE $0x0e // LEAL 0(SI)(CX*1), AX                 // lea	eax, [rsi + rcx]
	WORD    $0xf801             // ADDL DI, AX                          // add	eax, edi
	NOP                         // (skipped)                            // mov	rsp, rbp
	NOP                         // (skipped)                            // pop	rbp
	MOVB    AX, ret+8(FP)       // <--
	RET                         // <--                                  // ret

TEXT ·Test_fn_282_2(SB), NOSPLIT, $0-26
	MOVWQZX a+0(FP), DI
	MOVQ    b+8(FP), SI
	MOVWQZX c+16(FP), DX
	NOP                         // (skipped)                            // push	rbp
	NOP                         // (skipped)                            // mov	rbp, rsp
	NOP                         // (skipped)                            // and	rsp, -8
	WORD    $0xaf0f; BYTE $0xfa // IMULL DX, DI                         // imul	edi, edx
	WORD    $0x048d; BYTE $0x3e // LEAL 0(SI)(DI*1), AX                 // lea	eax, [rsi + rdi]
	NOP                         // (skipped)                            // mov	rsp, rbp
	NOP                         // (skipped)                            // pop	rbp
	MOVW    AX, ret+24(FP)      // <--
	RET                         // <--                                  // ret

TEXT ·Test_fn_481_1(SB), NOSPLIT, $0-25
	MOVLQZX a+0(FP), DI
	MOVQ    b+8(FP), SI
	MOVBQZX c+16(FP), DX
	NOP                         // (skipped)                            // push	rbp
	NOP                         // (skipped)                            // mov	rbp, rsp
	NOP                         // (skipped)                            // and	rsp, -8
	WORD    $0xaf0f; BYTE $0xfa // IMULL DX, DI                         // imul	edi, edx
	WORD    $0x048d; BYTE $0x3e // LEAL 0(SI)(DI*1), AX                 // lea	eax, [rsi + rdi]
	NOP                         // (skipped)                            // mov	rsp, rbp
	NOP                         // (skipped)                            // pop	rbp
	MOVB    AX, ret+24(FP)      // <--
	RET                         // <--                                  // ret

TEXT ·Test_fn_444_4(SB), NOSPLIT, $0-20
	MOVLQZX a+0(FP), DI
	MOVLQZX b+4(FP), SI
	MOVLQZX c+8(FP), DX
	NOP                         // (skipped)                            // push	rbp
	NOP                         // (skipped)                            // mov	rbp, rsp
	NOP                         // (skipped)                            // and	rsp, -8
	WORD    $0xaf0f; BYTE $0xfa // IMULL DX, DI                         // imul	edi, edx
	WORD    $0x048d; BYTE $0x37 // LEAL 0(DI)(SI*1), AX                 // lea	eax, [rdi + rsi]
	NOP                         // (skipped)                            // mov	rsp, rbp
	NOP                         // (skipped)                            // pop	rbp
	MOVL    AX, ret+16(FP)      // <--
	RET                         // <--                                  // ret

TEXT ·Test_fn_44F4F8_4(SB), NOSPLIT, $0-28
	MOVLQZX a+0(FP), DI
	MOVLQZX b+4(FP), SI
	MOVSS   c+8(FP), X0
	MOVSD   d+16(FP), X1
	NOP                         // (skipped)                            // push	rbp
	NOP                         // (skipped)                            // mov	rbp, rsp
	NOP                         // (skipped)                            // and	rsp, -8
	LONG    $0xd72a0ff3         // CVTSI2SSL DI, X2                     // cvtsi2ss	xmm2, edi
	LONG    $0xde2a0ff3         // CVTSI2SSL SI, X3                     // cvtsi2ss	xmm3, esi
	LONG    $0xd0590ff3         // MULSS X0, X2                         // mulss	xmm2, xmm0
	LONG    $0xda580ff3         // ADDSS X2, X3                         // addss	xmm3, xmm2
	WORD    $0x570f; BYTE $0xc0 // XORPS X0, X0                         // xorps	xmm0, xmm0
	LONG    $0xc35a0ff3         // CVTSS2SD X3, X0                      // cvtss2sd	xmm0, xmm3
	LONG    $0xc1580ff2         // ADDSD X1, X0                         // addsd	xmm0, xmm1
	LONG    $0xc02c0ff2         // CVTTSD2SIL X0, AX                    // cvttsd2si	eax, xmm0
	NOP                         // (skipped)                            // mov	rsp, rbp
	NOP                         // (skipped)                            // pop	rbp
	MOVL    AX, ret+24(FP)      // <--
	RET                         // <--                                  // ret

TEXT ·Test_fn_F4F4F8_F8(SB), NOSPLIT, $0-24
	MOVSS a+0(FP), X0
	MOVSS b+4(FP), X1
	MOVSD c+8(FP), X2
	NOP                       // (skipped)                            // push	rbp
	NOP                       // (skipped)                            // mov	rbp, rsp
	NOP                       // (skipped)                            // and	rsp, -8
	LONG  $0xd85a0ff3         // CVTSS2SD X0, X3                      // cvtss2sd	xmm3, xmm0
	LONG  $0xda590ff2         // MULSD X2, X3                         // mulsd	xmm3, xmm2
	WORD  $0x570f; BYTE $0xc0 // XORPS X0, X0                         // xorps	xmm0, xmm0
	LONG  $0xc15a0ff3         // CVTSS2SD X1, X0                      // cvtss2sd	xmm0, xmm1
	LONG  $0xc3580ff2         // ADDSD X3, X0                         // addsd	xmm0, xmm3
	NOP                       // (skipped)                            // mov	rsp, rbp
	NOP                       // (skipped)                            // pop	rbp
	MOVSD X0, ret+16(FP)      // <--
	RET                       // <--                                  // ret

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
	IMULQ R9, CX          // <--                                  // imul	rcx, r9
	LEAQ  0(SI)(R8*1), AX // <--                                  // lea	rax, [rsi + r8]
	ADDQ  DI, AX          // <--                                  // add	rax, rdi
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
	WORD $0x8548; BYTE $0xf6   // TESTQ SI, SI                         // test	rsi, rsi
	JLE  LBB10_11              // <--                                  // jle	.LBB10_11
	NOP                        // (skipped)                            // push	rbp
	NOP                        // (skipped)                            // mov	rbp, rsp
	NOP                        // (skipped)                            // and	rsp, -8
	XORL AX, AX                // <--                                  // xor	eax, eax
	CMPQ SI, $0x8              // <--                                  // cmp	rsi, 8
	JB   LBB10_6               // <--                                  // jb	.LBB10_6
	MOVQ CX, DX                // <--                                  // mov	rdx, rcx
	SUBQ DI, DX                // <--                                  // sub	rdx, rdi
	LONG $0x20fa8348           // CMPQ DX, $test_fn_111_0(SB)          // cmp	rdx, 32
	JB   LBB10_6               // <--                                  // jb	.LBB10_6
	MOVQ SI, AX                // <--                                  // mov	rax, rsi
	ANDQ $-0x8, AX             // <--                                  // and	rax, -8
	XORL DX, DX                // <--                                  // xor	edx, edx

LBB10_4:
	LONG $0x9704100f             // MOVUPS 0(DI)(DX*4), X0               // movups	xmm0, xmmword ptr [rdi + 4*rdx]
	LONG $0x974c100f; BYTE $0x10 // MOVUPS 0x10(DI)(DX*4), X1            // movups	xmm1, xmmword ptr [rdi + 4*rdx + 16]
	WORD $0x590f; BYTE $0xc0     // MULPS X0, X0                         // mulps	xmm0, xmm0
	WORD $0x590f; BYTE $0xc9     // MULPS X1, X1                         // mulps	xmm1, xmm1
	LONG $0x9104110f             // MOVUPS X0, 0(CX)(DX*4)               // movups	xmmword ptr [rcx + 4*rdx], xmm0
	LONG $0x914c110f; BYTE $0x10 // MOVUPS X1, 0x10(CX)(DX*4)            // movups	xmmword ptr [rcx + 4*rdx + 16], xmm1
	ADDQ $0x8, DX                // <--                                  // add	rdx, 8
	CMPQ AX, DX                  // <--                                  // cmp	rax, rdx
	JNE  LBB10_4                 // <--                                  // jne	.LBB10_4
	CMPQ AX, SI                  // <--                                  // cmp	rax, rsi
	JE   LBB10_10                // <--                                  // je	.LBB10_10

LBB10_6:
	MOVQ AX, DX   // <--                                  // mov	rdx, rax
	NOTQ DX       // <--                                  // not	rdx
	ADDQ SI, DX   // <--                                  // add	rdx, rsi
	MOVQ SI, R8   // <--                                  // mov	r8, rsi
	ANDQ $0x3, R8 // <--                                  // and	r8, 3
	JE   LBB10_8  // <--                                  // je	.LBB10_8

LBB10_7:
	LONG $0x04100ff3; BYTE $0x87 // MOVSS 0(DI)(AX*4), X0                // movss	xmm0, dword ptr [rdi + 4*rax]
	LONG $0xc0590ff3             // MULSS X0, X0                         // mulss	xmm0, xmm0
	LONG $0x04110ff3; BYTE $0x81 // MOVSS X0, 0(CX)(AX*4)                // movss	dword ptr [rcx + 4*rax], xmm0
	INCQ AX                      // <--                                  // inc	rax
	DECQ R8                      // <--                                  // dec	r8
	JNE  LBB10_7                 // <--                                  // jne	.LBB10_7

LBB10_8:
	CMPQ DX, $0x3 // <--                                  // cmp	rdx, 3
	JB   LBB10_10 // <--                                  // jb	.LBB10_10

LBB10_9:
	LONG $0x04100ff3; BYTE $0x87   // MOVSS 0(DI)(AX*4), X0                // movss	xmm0, dword ptr [rdi + 4*rax]
	LONG $0xc0590ff3               // MULSS X0, X0                         // mulss	xmm0, xmm0
	LONG $0x04110ff3; BYTE $0x81   // MOVSS X0, 0(CX)(AX*4)                // movss	dword ptr [rcx + 4*rax], xmm0
	LONG $0x44100ff3; WORD $0x0487 // MOVSS 0x4(DI)(AX*4), X0              // movss	xmm0, dword ptr [rdi + 4*rax + 4]
	LONG $0xc0590ff3               // MULSS X0, X0                         // mulss	xmm0, xmm0
	LONG $0x44110ff3; WORD $0x0481 // MOVSS X0, 0x4(CX)(AX*4)              // movss	dword ptr [rcx + 4*rax + 4], xmm0
	LONG $0x44100ff3; WORD $0x0887 // MOVSS 0x8(DI)(AX*4), X0              // movss	xmm0, dword ptr [rdi + 4*rax + 8]
	LONG $0xc0590ff3               // MULSS X0, X0                         // mulss	xmm0, xmm0
	LONG $0x44110ff3; WORD $0x0881 // MOVSS X0, 0x8(CX)(AX*4)              // movss	dword ptr [rcx + 4*rax + 8], xmm0
	LONG $0x44100ff3; WORD $0x0c87 // MOVSS 0xc(DI)(AX*4), X0              // movss	xmm0, dword ptr [rdi + 4*rax + 12]
	LONG $0xc0590ff3               // MULSS X0, X0                         // mulss	xmm0, xmm0
	LONG $0x44110ff3; WORD $0x0c81 // MOVSS X0, 0xc(CX)(AX*4)              // movss	dword ptr [rcx + 4*rax + 12], xmm0
	ADDQ $0x4, AX                  // <--                                  // add	rax, 4
	CMPQ SI, AX                    // <--                                  // cmp	rsi, rax
	JNE  LBB10_9                   // <--                                  // jne	.LBB10_9

LBB10_10:
	NOP // (skipped)                            // mov	rsp, rbp
	NOP // (skipped)                            // pop	rbp

LBB10_11:
	RET // <--                                  // ret
