//go:build !noasm && amd64
// Code generated by gocc -- DO NOT EDIT.

#include "textflag.h"

TEXT ·f32_axpy(SB), NOSPLIT, $0-28
	MOVQ x+0(FP), DI
	MOVQ y+8(FP), SI
	MOVQ size+16(FP), DX
	MOVL alpha+24(FP), CX
	NOP                          // <--                                  // push	rbp
	NOP                          // <--                                  // mov	rbp, rsp
	NOP                          // <--                                  // and	rsp, -8
	LONG $0x08fa8348             // CMPQ $0x8, DX                        // cmp	rdx, 8
	JB   LBB0_5                  // <--                                  // jb	.LBB0_5
	LONG $0x187de2c4; BYTE $0xc8 // SBBL CL, AL                          // vbroadcastss	ymm1, xmm0
	LONG $0xf84a8d48             // LEAQ -0x8(DX), CX                    // lea	rcx, [rdx - 8]
	MOVQ CX, AX                  // <--                                  // mov	rax, rcx
	LONG $0x03e8c148             // SHRQ $0x3, AX                        // shr	rax, 3
	WORD $0xff48; BYTE $0xc0     // INCQ AX                              // inc	rax
	LONG $0x08f98348             // CMPQ $0x8, CX                        // cmp	rcx, 8
	JAE  LBB0_12                 // <--                                  // jae	.LBB0_12
	WORD $0xc931                 // XORL CX, CX                          // xor	ecx, ecx
	JMP  LBB0_3                  // <--                                  // jmp	.LBB0_3

LBB0_12:
	MOVQ AX, R8      // <--                                  // mov	r8, rax
	LONG $0xfee08349 // ANDQ $-0x2, R8                       // and	r8, -2
	WORD $0xc931     // XORL CX, CX                          // xor	ecx, ecx

LBB0_13:
	LONG $0x1410fcc5; BYTE $0x8f               // ADCB DL, 0(DI)(CX*4)                 // vmovups	ymm2, ymmword ptr [rdi + 4*rcx]
	LONG $0xa875e2c4; WORD $0x8e14             // ?                                    // vfmadd213ps	ymm2, ymm1, ymmword ptr [rsi + 4*rcx]
	LONG $0x1411fcc5; BYTE $0x8e               // ADCL DX, 0(SI)(CX*4)                 // vmovups	ymmword ptr [rsi + 4*rcx], ymm2
	LONG $0x5410fcc5; WORD $0x208f             // ADCB DL, 0x20(DI)(CX*4)              // vmovups	ymm2, ymmword ptr [rdi + 4*rcx + 32]
	LONG $0xa875e2c4; WORD $0x8e54; BYTE $0x20 // ?                                    // vfmadd213ps	ymm2, ymm1, ymmword ptr [rsi + 4*rcx + 32]
	LONG $0x5411fcc5; WORD $0x208e             // ADCL DX, 0x20(SI)(CX*4)              // vmovups	ymmword ptr [rsi + 4*rcx + 32], ymm2
	LONG $0x10c18348                           // ADDQ $0x10, CX                       // add	rcx, 16
	LONG $0xfec08349                           // ADDQ $-0x2, R8                       // add	r8, -2
	JNE  LBB0_13                               // <--                                  // jne	.LBB0_13

LBB0_3:
	WORD $0x01a8                   // TESTL $0x1, AL                       // test	al, 1
	JE   LBB0_5                    // <--                                  // je	.LBB0_5
	LONG $0x1410fcc5; BYTE $0x8f   // ADCB DL, 0(DI)(CX*4)                 // vmovups	ymm2, ymmword ptr [rdi + 4*rcx]
	LONG $0xa86de2c4; WORD $0x8e0c // ?                                    // vfmadd213ps	ymm1, ymm2, ymmword ptr [rsi + 4*rcx]
	LONG $0x0c11fcc5; BYTE $0x8e   // ADCL CX, 0(SI)(CX*4)                 // vmovups	ymmword ptr [rsi + 4*rcx], ymm1

LBB0_5:
	WORD $0xc2f6; BYTE $0x07       // TESTL $0x7, DL                       // test	dl, 7
	JE   LBB0_11                   // <--                                  // je	.LBB0_11
	MOVQ DX, AX                    // <--                                  // mov	rax, rdx
	LONG $0xf8e08348               // ANDQ $-0x8, AX                       // and	rax, -8
	WORD $0x3948; BYTE $0xd0       // CMPQ DX, AX                          // cmp	rax, rdx
	JAE  LBB0_11                   // <--                                  // jae	.LBB0_11
	MOVQ AX, CX                    // <--                                  // mov	rcx, rax
	WORD $0xf748; BYTE $0xd1       // NOTQ CX                              // not	rcx
	WORD $0xc2f6; BYTE $0x01       // TESTL $0x1, DL                       // test	dl, 1
	JE   LBB0_9                    // <--                                  // je	.LBB0_9
	LONG $0x0c10fac5; BYTE $0x87   // ADCB CL, 0(DI)(AX*4)                 // vmovss	xmm1, dword ptr [rdi + 4*rax]
	LONG $0xa979e2c4; WORD $0x860c // ?                                    // vfmadd213ss	xmm1, xmm0, dword ptr [rsi + 4*rax]
	LONG $0x0c11fac5; BYTE $0x86   // ADCL CX, 0(SI)(AX*4)                 // vmovss	dword ptr [rsi + 4*rax], xmm1
	LONG $0x01c88348               // ORQ $0x1, AX                         // or	rax, 1

LBB0_9:
	WORD $0x0148; BYTE $0xd1 // ADDQ DX, CX                          // add	rcx, rdx
	JE   LBB0_11             // <--                                  // je	.LBB0_11

LBB0_10:
	LONG $0x0c10fac5; BYTE $0x87               // ADCB CL, 0(DI)(AX*4)                 // vmovss	xmm1, dword ptr [rdi + 4*rax]
	LONG $0xa979e2c4; WORD $0x860c             // ?                                    // vfmadd213ss	xmm1, xmm0, dword ptr [rsi + 4*rax]
	LONG $0x0c11fac5; BYTE $0x86               // ADCL CX, 0(SI)(AX*4)                 // vmovss	dword ptr [rsi + 4*rax], xmm1
	LONG $0x4c10fac5; WORD $0x0487             // ADCB CL, 0x4(DI)(AX*4)               // vmovss	xmm1, dword ptr [rdi + 4*rax + 4]
	LONG $0xa979e2c4; WORD $0x864c; BYTE $0x04 // ?                                    // vfmadd213ss	xmm1, xmm0, dword ptr [rsi + 4*rax + 4]
	LONG $0x4c11fac5; WORD $0x0486             // ADCL CX, 0x4(SI)(AX*4)               // vmovss	dword ptr [rsi + 4*rax + 4], xmm1
	LONG $0x02c08348                           // ADDQ $0x2, AX                        // add	rax, 2
	WORD $0x3948; BYTE $0xd0                   // CMPQ DX, AX                          // cmp	rax, rdx
	JB   LBB0_10                               // <--                                  // jb	.LBB0_10

LBB0_11:
	NOP        // <--                                  // mov	rsp, rbp
	NOP        // <--                                  // pop	rbp
	VZEROUPPER // <--                                  // vzeroupper
	RET        // <--                                  // ret

TEXT ·f32_matmul(SB), 0, $288-32
	MOVQ   dst+0(FP), DI
	MOVQ   m+8(FP), SI
	MOVQ   n+16(FP), DX
	MOVQ   dims+24(FP), CX
	NOP                                          // <--                                  // push	rbp
	NOP                                          // <--                                  // mov	rbp, rsp
	MOVQ   R15, 248(SP)                          // <--                                  // push	r15
	MOVQ   R14, 256(SP)                          // <--                                  // push	r14
	MOVQ   R13, 264(SP)                          // <--                                  // push	r13
	MOVQ   R12, 272(SP)                          // <--                                  // push	r12
	MOVQ   BX, 280(SP)                           // <--                                  // push	rbx
	LONG   $0xf8e48348                           // ANDQ $-0x8, SP                       // and	rsp, -8
	NOP                                          // <--                                  // sub	rsp, 248
	MOVQ   DX, 0x8(SP)                           // <--                                  // mov	qword ptr [rsp + 8], rdx
	MOVQ   SI, 0(SP)                             // <--                                  // mov	qword ptr [rsp], rsi
	MOVQ   CX, AX                                // <--                                  // mov	rax, rcx
	LONG   $0xffff2548; WORD $0x0000             // ANDQ $0xffff, AX                     // and	rax, 65535
	MOVQ   AX, 0x18(SP)                          // <--                                  // mov	qword ptr [rsp + 24], rax
	JE     LBB1_27                               // <--                                  // je	.LBB1_27
	MOVQ   CX, AX                                // <--                                  // mov	rax, rcx
	LONG   $0x30e8c148                           // SHRQ $0x30, AX                       // shr	rax, 48
	MOVQ   AX, 0x10(SP)                          // <--                                  // mov	qword ptr [rsp + 16], rax
	JE     LBB1_27                               // <--                                  // je	.LBB1_27
	MOVQ   CX, R13                               // <--                                  // mov	r13, rcx
	LONG   $0x10edc149                           // SHRQ $0x10, R13                      // shr	r13, 16
	MOVZX  R13, DX                               // <--                                  // movzx	edx, r13w
	LONG   $0xf8e58141; WORD $0x00ff; BYTE $0x00 // ANDL $0xfff8, R13                    // and	r13d, 65528
	MOVQ   0x10(SP), SI                          // <--                                  // mov	rsi, qword ptr [rsp + 16]
	LONG   $0x02fe8348                           // CMPQ $0x2, SI                        // cmp	rsi, 2
	MOVL   $0x1, R14                             // <--                                  // mov	r14d, 1
	CMOVAE SI, R14                               // <--                                  // cmovae	r14, rsi
	MOVQ   DX, AX                                // <--                                  // mov	rax, rdx
	WORD   $0x294c; BYTE $0xe8                   // SUBQ R13, AX                         // sub	rax, r13
	MOVQ   DX, R8                                // <--                                  // mov	r8, rdx
	WORD   $0x294d; BYTE $0xe8                   // SUBQ R13, R8                         // sub	r8, r13
	JBE    LBB1_3                                // <--                                  // jbe	.LBB1_3
	LONG   $0x10f88349                           // CMPQ $0x10, R8                       // cmp	r8, 16
	MOVQ   DX, 0x20(SP)                          // <--                                  // mov	qword ptr [rsp + 32], rdx
	JAE    LBB1_11                               // <--                                  // jae	.LBB1_11
	WORD   $0xe9c1; BYTE $0x13                   // SHRL $0x13, CX                       // shr	ecx, 19
	LONG   $0xceaf0f48                           // IMULQ SI, CX                         // imul	rcx, rsi
	LONG   $0x05e1c148                           // SHLQ $0x5, CX                        // shl	rcx, 5
	LONG   $0x244c0148; BYTE $0x08               // ADDQ CX, 0x8(SP)                     // add	qword ptr [rsp + 8], rcx
	QUAD   $0x00000000b5048d48                   // LEAQ 0(SI*4), AX                     // lea	rax, [4*rsi]
	QUAD   $0x00000000950c8d48                   // LEAQ 0(DX*4), CX                     // lea	rcx, [4*rdx]
	WORD   $0xd231                               // XORL DX, DX                          // xor	edx, edx

LBB1_18:
	MOVQ R14, R15                  // <--                                  // mov	r15, r14
	MOVQ DX, SI                    // <--                                  // mov	rsi, rdx
	LONG $0x74af0f48; WORD $0x1024 // IMULQ 0x10(SP), SI                   // imul	rsi, qword ptr [rsp + 16]
	MOVQ 0x8(SP), R11              // <--                                  // mov	r11, qword ptr [rsp + 8]
	WORD $0x3145; BYTE $0xc0       // XORL R8, R8                          // xor	r8d, r8d
	MOVQ 0(SP), BX                 // <--                                  // mov	rbx, qword ptr [rsp]
	MOVQ 0x20(SP), R14             // <--                                  // mov	r14, qword ptr [rsp + 32]

LBB1_19:
	LONG $0xc057f8c5 // ?                                    // vxorps	xmm0, xmm0, xmm0
	MOVQ R11, R9     // <--                                  // mov	r9, r11
	MOVQ R13, R10    // <--                                  // mov	r10, r13

LBB1_20:
	LONG $0x107ac1c4; BYTE $0x09   // ADCB CL, 0(CX)                       // vmovss	xmm1, dword ptr [r9]
	LONG $0xb971a2c4; WORD $0x9304 // ?                                    // vfmadd231ss	xmm0, xmm1, dword ptr [rbx + 4*r10]
	WORD $0xff49; BYTE $0xc2       // INCQ R10                             // inc	r10
	WORD $0x0149; BYTE $0xc1       // ADDQ AX, R9                          // add	r9, rax
	WORD $0x394d; BYTE $0xd6       // CMPQ R10, R14                        // cmp	r14, r10
	JNE  LBB1_20                   // <--                                  // jne	.LBB1_20
	LONG $0x300c8d4d               // LEAQ 0(R8)(SI*1), R9                 // lea	r9, [r8 + rsi]
	LONG $0x117aa1c4; WORD $0x8f04 // ADCL AX, 0(DI)(R9*4)                 // vmovss	dword ptr [rdi + 4*r9], xmm0
	WORD $0xff49; BYTE $0xc0       // INCQ R8                              // inc	r8
	LONG $0x04c38349               // ADDQ $0x4, R11                       // add	r11, 4
	WORD $0x394d; BYTE $0xf8       // CMPQ R15, R8                         // cmp	r8, r15
	JNE  LBB1_19                   // <--                                  // jne	.LBB1_19
	WORD $0xff48; BYTE $0xc2       // INCQ DX                              // inc	rdx
	WORD $0x0148; BYTE $0xcb       // ADDQ CX, BX                          // add	rbx, rcx
	MOVQ BX, 0(SP)                 // <--                                  // mov	qword ptr [rsp], rbx
	LONG $0x24543b48; BYTE $0x18   // CMPQ 0x18(SP), DX                    // cmp	rdx, qword ptr [rsp + 24]
	MOVQ R15, R14                  // <--                                  // mov	r14, r15
	JNE  LBB1_18                   // <--                                  // jne	.LBB1_18

LBB1_27:
	NOP               // <--                                  // lea	rsp, [rbp - 40]
	MOVQ 280(SP), BX  // <--                                  // pop	rbx
	MOVQ 272(SP), R12 // <--                                  // pop	r12
	MOVQ 264(SP), R13 // <--                                  // pop	r13
	MOVQ 256(SP), R14 // <--                                  // pop	r14
	MOVQ 248(SP), R15 // <--                                  // pop	r15
	NOP               // <--                                  // pop	rbp
	VZEROUPPER        // <--                                  // vzeroupper
	RET               // <--                                  // ret

LBB1_3:
	LONG $0xff468d49         // LEAQ -0x1(R14), AX                   // lea	rax, [r14 - 1]
	MOVL R14, CX             // <--                                  // mov	ecx, r14d
	WORD $0xe183; BYTE $0x03 // ANDL $0x3, CX                        // and	ecx, 3
	LONG $0xfce68341         // ANDL $-0x4, R14                      // and	r14d, -4
	LONG $0x0c578d48         // LEAQ 0xc(DI), DX                     // lea	rdx, [rdi + 12]
	LONG $0x02e6c148         // SHLQ $0x2, SI                        // shl	rsi, 2
	MOVQ SI, R10             // <--                                  // mov	r10, rsi
	WORD $0xf631             // XORL SI, SI                          // xor	esi, esi
	LONG $0xc057f8c5         // ?                                    // vxorps	xmm0, xmm0, xmm0
	JMP  LBB1_4              // <--                                  // jmp	.LBB1_4

LBB1_9:
	WORD $0xff48; BYTE $0xc6     // INCQ SI                              // inc	rsi
	WORD $0x014c; BYTE $0xda     // ADDQ R11, DX                         // add	rdx, r11
	WORD $0x014c; BYTE $0xdf     // ADDQ R11, DI                         // add	rdi, r11
	LONG $0x24743b48; BYTE $0x18 // CMPQ 0x18(SP), SI                    // cmp	rsi, qword ptr [rsp + 24]
	JE   LBB1_27                 // <--                                  // je	.LBB1_27

LBB1_4:
	WORD $0x3145; BYTE $0xc0 // XORL R8, R8                          // xor	r8d, r8d
	MOVQ R10, R11            // <--                                  // mov	r11, r10
	LONG $0x03f88348         // CMPQ $0x3, AX                        // cmp	rax, 3
	JB   LBB1_6              // <--                                  // jb	.LBB1_6

LBB1_5:
	LONG $0x1178a1c4; WORD $0x8244; BYTE $0xf4 // ADCL AX, -0xc(DX)(R8*4)              // vmovups	xmmword ptr [rdx + 4*r8 - 12], xmm0
	LONG $0x04c08349                           // ADDQ $0x4, R8                        // add	r8, 4
	WORD $0x394d; BYTE $0xc6                   // CMPQ R8, R14                         // cmp	r14, r8
	JNE  LBB1_5                                // <--                                  // jne	.LBB1_5

LBB1_6:
	WORD $0x8548; BYTE $0xc9 // TESTQ CX, CX                         // test	rcx, rcx
	JE   LBB1_9              // <--                                  // je	.LBB1_9
	LONG $0x870c8d4e         // LEAQ 0(DI)(R8*4), R9                 // lea	r9, [rdi + 4*r8]
	WORD $0x3145; BYTE $0xc0 // XORL R8, R8                          // xor	r8d, r8d

LBB1_8:
	MOVL $0x0, 0(R9)(R8*4)   // <--                                  // mov	dword ptr [r9 + 4*r8], 0
	WORD $0xff49; BYTE $0xc0 // INCQ R8                              // inc	r8
	WORD $0x394c; BYTE $0xc1 // CMPQ R8, CX                          // cmp	rcx, r8
	JNE  LBB1_8              // <--                                  // jne	.LBB1_8
	JMP  LBB1_9              // <--                                  // jmp	.LBB1_9

LBB1_11:
	MOVQ R8, 0x98(SP)        // <--                                  // mov	qword ptr [rsp + 152], r8
	LONG $0xf0e08348         // ANDQ $-0x10, AX                      // and	rax, -16
	MOVQ AX, 0x28(SP)        // <--                                  // mov	qword ptr [rsp + 40], rax
	WORD $0xe9c1; BYTE $0x13 // SHRL $0x13, CX                       // shr	ecx, 19
	MOVQ SI, AX              // <--                                  // mov	rax, rsi
	LONG $0xc1af0f48         // IMULQ CX, AX                         // imul	rax, rcx
	LONG $0x05e0c148         // SHLQ $0x5, AX                        // shl	rax, 5
	MOVQ AX, 0xe8(SP)        // <--                                  // mov	qword ptr [rsp + 232], rax
	MOVQ SI, AX              // <--                                  // mov	rax, rsi
	LONG $0x06e0c148         // SHLQ $0x6, AX                        // shl	rax, 6
	MOVQ AX, 0xe0(SP)        // <--                                  // mov	qword ptr [rsp + 224], rax
	LONG $0x05e1c148         // SHLQ $0x5, CX                        // shl	rcx, 5
	MOVQ 0(SP), AX           // <--                                  // mov	rax, qword ptr [rsp]
	WORD $0x0148; BYTE $0xc8 // ADDQ CX, AX                          // add	rax, rcx
	LONG $0x20c08348         // ADDQ $0x20, AX                       // add	rax, 32
	MOVQ AX, 0x38(SP)        // <--                                  // mov	qword ptr [rsp + 56], rax
	LONG $0x0f458d49         // LEAQ 0xf(R13), AX                    // lea	rax, [r13 + 15]
	LONG $0xc6af0f48         // IMULQ SI, AX                         // imul	rax, rsi
	MOVQ AX, 0xd8(SP)        // <--                                  // mov	qword ptr [rsp + 216], rax
	LONG $0x0e4d8d4d         // LEAQ 0xe(R13), R9                    // lea	r9, [r13 + 14]
	LONG $0xceaf0f4c         // IMULQ SI, R9                         // imul	r9, rsi
	LONG $0x0d458d4d         // LEAQ 0xd(R13), R8                    // lea	r8, [r13 + 13]
	LONG $0xc6af0f4c         // IMULQ SI, R8                         // imul	r8, rsi
	LONG $0x0c5d8d49         // LEAQ 0xc(R13), BX                    // lea	rbx, [r13 + 12]
	LONG $0xdeaf0f48         // IMULQ SI, BX                         // imul	rbx, rsi
	LONG $0x0b558d4d         // LEAQ 0xb(R13), R10                   // lea	r10, [r13 + 11]
	LONG $0xd6af0f4c         // IMULQ SI, R10                        // imul	r10, rsi
	LONG $0x0a7d8d4d         // LEAQ 0xa(R13), R15                   // lea	r15, [r13 + 10]
	LONG $0xfeaf0f4c         // IMULQ SI, R15                        // imul	r15, rsi
	LONG $0x09458d49         // LEAQ 0x9(R13), AX                    // lea	rax, [r13 + 9]
	LONG $0xc6af0f48         // IMULQ SI, AX                         // imul	rax, rsi
	MOVQ AX, 0x90(SP)        // <--                                  // mov	qword ptr [rsp + 144], rax
	LONG $0x08658d4d         // LEAQ 0x8(R13), R12                   // lea	r12, [r13 + 8]
	LONG $0xe6af0f4c         // IMULQ SI, R12                        // imul	r12, rsi
	LONG $0x075d8d4d         // LEAQ 0x7(R13), R11                   // lea	r11, [r13 + 7]
	LONG $0xdeaf0f4c         // IMULQ SI, R11                        // imul	r11, rsi
	MOVQ SI, CX              // <--                                  // mov	rcx, rsi
	LONG $0x06758d49         // LEAQ 0x6(R13), SI                    // lea	rsi, [r13 + 6]
	LONG $0xf1af0f48         // IMULQ CX, SI                         // imul	rsi, rcx
	LONG $0x05458d49         // LEAQ 0x5(R13), AX                    // lea	rax, [r13 + 5]
	LONG $0xc1af0f48         // IMULQ CX, AX                         // imul	rax, rcx
	MOVQ AX, 0x88(SP)        // <--                                  // mov	qword ptr [rsp + 136], rax
	LONG $0x04458d49         // LEAQ 0x4(R13), AX                    // lea	rax, [r13 + 4]
	LONG $0xc1af0f48         // IMULQ CX, AX                         // imul	rax, rcx
	MOVQ AX, 0x80(SP)        // <--                                  // mov	qword ptr [rsp + 128], rax
	LONG $0x03458d49         // LEAQ 0x3(R13), AX                    // lea	rax, [r13 + 3]
	LONG $0xc1af0f48         // IMULQ CX, AX                         // imul	rax, rcx
	MOVQ AX, 0x78(SP)        // <--                                  // mov	qword ptr [rsp + 120], rax
	LONG $0x02458d49         // LEAQ 0x2(R13), AX                    // lea	rax, [r13 + 2]
	LONG $0xc1af0f48         // IMULQ CX, AX                         // imul	rax, rcx
	MOVQ AX, 0x70(SP)        // <--                                  // mov	qword ptr [rsp + 112], rax
	MOVQ 0x28(SP), AX        // <--                                  // mov	rax, qword ptr [rsp + 40]
	LONG $0x28148d4a         // LEAQ 0(AX)(R13*1), DX                // lea	rdx, [rax + r13]
	WORD $0xff49; BYTE $0xc5 // INCQ R13                             // inc	r13
	LONG $0xe9af0f4c         // IMULQ CX, R13                        // imul	r13, rcx
	MOVQ CX, AX              // <--                                  // mov	rax, rcx
	MOVQ DX, 0x68(SP)        // <--                                  // mov	qword ptr [rsp + 104], rdx
	LONG $0xc2af0f48         // IMULQ DX, AX                         // imul	rax, rdx
	MOVQ 0x8(SP), DX         // <--                                  // mov	rdx, qword ptr [rsp + 8]
	LONG $0x82048d48         // LEAQ 0(DX)(AX*4), AX                 // lea	rax, [rdx + 4*rax]
	MOVQ AX, 0x50(SP)        // <--                                  // mov	qword ptr [rsp + 80], rax
	MOVQ 0x20(SP), AX        // <--                                  // mov	rax, qword ptr [rsp + 32]
	QUAD $0x0000000085048d48 // LEAQ 0(AX*4), AX                     // lea	rax, [4*rax]
	MOVQ AX, 0x48(SP)        // <--                                  // mov	qword ptr [rsp + 72], rax
	QUAD $0x000000008d048d48 // LEAQ 0(CX*4), AX                     // lea	rax, [4*rcx]
	MOVQ AX, 0x60(SP)        // <--                                  // mov	qword ptr [rsp + 96], rax
	WORD $0xc931             // XORL CX, CX                          // xor	ecx, ecx
	MOVQ R14, 0xa0(SP)       // <--                                  // mov	qword ptr [rsp + 160], r14
	MOVQ R13, 0xf0(SP)       // <--                                  // mov	qword ptr [rsp + 240], r13
	MOVQ DI, 0x30(SP)        // <--                                  // mov	qword ptr [rsp + 48], rdi
	MOVQ R8, 0xd0(SP)        // <--                                  // mov	qword ptr [rsp + 208], r8
	MOVQ R9, 0xc8(SP)        // <--                                  // mov	qword ptr [rsp + 200], r9
	MOVQ BX, 0xc0(SP)        // <--                                  // mov	qword ptr [rsp + 192], rbx
	MOVQ 0x90(SP), BX        // <--                                  // mov	rbx, qword ptr [rsp + 144]
	MOVQ 0x70(SP), R9        // <--                                  // mov	r9, qword ptr [rsp + 112]
	JMP  LBB1_12             // <--                                  // jmp	.LBB1_12

LBB1_26:
	MOVQ 0x58(SP), CX            // <--                                  // mov	rcx, qword ptr [rsp + 88]
	WORD $0xff48; BYTE $0xc1     // INCQ CX                              // inc	rcx
	MOVQ 0x48(SP), AX            // <--                                  // mov	rax, qword ptr [rsp + 72]
	LONG $0x24440148; BYTE $0x38 // ADDQ AX, 0x38(SP)                    // add	qword ptr [rsp + 56], rax
	LONG $0x24040148             // ADDQ AX, 0(SP)                       // add	qword ptr [rsp], rax
	LONG $0x244c3b48; BYTE $0x18 // CMPQ 0x18(SP), CX                    // cmp	rcx, qword ptr [rsp + 24]
	JE   LBB1_27                 // <--                                  // je	.LBB1_27

LBB1_12:
	MOVQ CX, 0x58(SP)              // <--                                  // mov	qword ptr [rsp + 88], rcx
	LONG $0x4caf0f48; WORD $0x1024 // IMULQ 0x10(SP), CX                   // imul	rcx, qword ptr [rsp + 16]
	MOVQ CX, 0xa8(SP)              // <--                                  // mov	qword ptr [rsp + 168], rcx
	MOVQ 0x50(SP), AX              // <--                                  // mov	rax, qword ptr [rsp + 80]
	MOVQ AX, 0x40(SP)              // <--                                  // mov	qword ptr [rsp + 64], rax
	MOVQ 0x8(SP), AX               // <--                                  // mov	rax, qword ptr [rsp + 8]
	WORD $0xc931                   // XORL CX, CX                          // xor	ecx, ecx
	JMP  LBB1_13                   // <--                                  // jmp	.LBB1_13

LBB1_16:
	MOVQ 0x30(SP), DI // <--                                  // mov	rdi, qword ptr [rsp + 48]

LBB1_25:
	MOVQ 0xa8(SP), AX              // <--                                  // mov	rax, qword ptr [rsp + 168]
	MOVQ 0xb0(SP), CX              // <--                                  // mov	rcx, qword ptr [rsp + 176]
	WORD $0x0148; BYTE $0xc8       // ADDQ CX, AX                          // add	rax, rcx
	LONG $0x0411fac5; BYTE $0x87   // ADCL AX, 0(DI)(AX*4)                 // vmovss	dword ptr [rdi + 4*rax], xmm0
	WORD $0xff48; BYTE $0xc1       // INCQ CX                              // inc	rcx
	MOVQ 0xb8(SP), AX              // <--                                  // mov	rax, qword ptr [rsp + 184]
	LONG $0x04c08348               // ADDQ $0x4, AX                        // add	rax, 4
	LONG $0x24448348; WORD $0x0440 // ADDQ $0x4, 0x40(SP)                  // add	qword ptr [rsp + 64], 4
	MOVQ 0xa0(SP), R14             // <--                                  // mov	r14, qword ptr [rsp + 160]
	WORD $0x394c; BYTE $0xf1       // CMPQ R14, CX                         // cmp	rcx, r14
	JE   LBB1_26                   // <--                                  // je	.LBB1_26

LBB1_13:
	MOVQ CX, 0xb0(SP)  // <--                                  // mov	qword ptr [rsp + 176], rcx
	LONG $0xc057f8c5   // ?                                    // vxorps	xmm0, xmm0, xmm0
	MOVQ 0x38(SP), R13 // <--                                  // mov	r13, qword ptr [rsp + 56]
	MOVQ AX, 0xb8(SP)  // <--                                  // mov	qword ptr [rsp + 184], rax
	MOVQ 0x28(SP), DX  // <--                                  // mov	rdx, qword ptr [rsp + 40]
	LONG $0xc957f0c5   // ?                                    // vxorps	xmm1, xmm1, xmm1
	MOVQ 0x88(SP), R14 // <--                                  // mov	r14, qword ptr [rsp + 136]
	MOVQ 0x80(SP), DI  // <--                                  // mov	rdi, qword ptr [rsp + 128]
	MOVQ 0x78(SP), CX  // <--                                  // mov	rcx, qword ptr [rsp + 120]

LBB1_14:
	LONG $0x1410fac5; BYTE $0xb8               // ADCB DL, 0(AX)(DI*4)                 // vmovss	xmm2, dword ptr [rax + 4*rdi]
	LONG $0x2169a3c4; WORD $0xb014; BYTE $0x10 // ?                                    // vinsertps	xmm2, xmm2, dword ptr [rax + 4*r14], 16
	LONG $0x2169e3c4; WORD $0xb014; BYTE $0x20 // ?                                    // vinsertps	xmm2, xmm2, dword ptr [rax + 4*rsi], 32
	LONG $0x2169a3c4; WORD $0x9814; BYTE $0x30 // ?                                    // vinsertps	xmm2, xmm2, dword ptr [rax + 4*r11], 48
	MOVQ 0xe8(SP), R8                          // <--                                  // mov	r8, qword ptr [rsp + 232]
	LONG $0x107aa1c4; WORD $0x001c             // ADCB BL, 0(AX)(R8*1)                 // vmovss	xmm3, dword ptr [rax + r8]
	MOVQ 0xf0(SP), R8                          // <--                                  // mov	r8, qword ptr [rsp + 240]
	LONG $0x2161a3c4; WORD $0x801c; BYTE $0x10 // ?                                    // vinsertps	xmm3, xmm3, dword ptr [rax + 4*r8], 16
	LONG $0x2161a3c4; WORD $0x881c; BYTE $0x20 // ?                                    // vinsertps	xmm3, xmm3, dword ptr [rax + 4*r9], 32
	LONG $0x2161e3c4; WORD $0x881c; BYTE $0x30 // ?                                    // vinsertps	xmm3, xmm3, dword ptr [rax + 4*rcx], 48
	MOVQ 0xc0(SP), R8                          // <--                                  // mov	r8, qword ptr [rsp + 192]
	LONG $0x107aa1c4; WORD $0x8024             // ADCB AH, 0(AX)(R8*4)                 // vmovss	xmm4, dword ptr [rax + 4*r8]
	MOVQ 0xd0(SP), R8                          // <--                                  // mov	r8, qword ptr [rsp + 208]
	LONG $0x2159a3c4; WORD $0x8024; BYTE $0x10 // ?                                    // vinsertps	xmm4, xmm4, dword ptr [rax + 4*r8], 16
	MOVQ 0xc8(SP), R8                          // <--                                  // mov	r8, qword ptr [rsp + 200]
	LONG $0x2159a3c4; WORD $0x8024; BYTE $0x20 // ?                                    // vinsertps	xmm4, xmm4, dword ptr [rax + 4*r8], 32
	MOVQ 0xd8(SP), R8                          // <--                                  // mov	r8, qword ptr [rsp + 216]
	LONG $0x2159a3c4; WORD $0x8024; BYTE $0x30 // ?                                    // vinsertps	xmm4, xmm4, dword ptr [rax + 4*r8], 48
	LONG $0x107aa1c4; WORD $0xa02c             // ADCB CH, 0(AX)(R12*4)                // vmovss	xmm5, dword ptr [rax + 4*r12]
	LONG $0x2151e3c4; WORD $0x982c; BYTE $0x10 // ?                                    // vinsertps	xmm5, xmm5, dword ptr [rax + 4*rbx], 16
	LONG $0x2151a3c4; WORD $0xb82c; BYTE $0x20 // ?                                    // vinsertps	xmm5, xmm5, dword ptr [rax + 4*r15], 32
	LONG $0x1865e3c4; WORD $0x01d2             // ?                                    // vinsertf128	ymm2, ymm3, xmm2, 1
	LONG $0x2151a3c4; WORD $0x901c; BYTE $0x30 // ?                                    // vinsertps	xmm3, xmm5, dword ptr [rax + 4*r10], 48
	LONG $0x1865e3c4; WORD $0x01dc             // ?                                    // vinsertf128	ymm3, ymm3, xmm4, 1
	LONG $0xb86dc2c4; WORD $0xe045             // ?                                    // vfmadd231ps	ymm0, ymm2, ymmword ptr [r13 - 32]
	LONG $0xb865c2c4; WORD $0x004d             // ?                                    // vfmadd231ps	ymm1, ymm3, ymmword ptr [r13]
	QUAD $0x000000e024840348                   // ADDQ 0xe0(SP), AX                    // add	rax, qword ptr [rsp + 224]
	LONG $0x40c58349                           // ADDQ $0x40, R13                      // add	r13, 64
	LONG $0xf0c28348                           // ADDQ $-0x10, DX                      // add	rdx, -16
	JNE  LBB1_14                               // <--                                  // jne	.LBB1_14
	LONG $0xc058f4c5                           // ?                                    // vaddps	ymm0, ymm1, ymm0
	LONG $0x197de3c4; WORD $0x01c1             // ?                                    // vextractf128	xmm1, ymm0, 1
	LONG $0xc158f8c5                           // ?                                    // vaddps	xmm0, xmm0, xmm1
	LONG $0x0579e3c4; WORD $0x01c8             // ?                                    // vpermilpd	xmm1, xmm0, 1
	LONG $0xc158f8c5                           // ?                                    // vaddps	xmm0, xmm0, xmm1
	LONG $0xc816fac5                           // ?                                    // vmovshdup	xmm1, xmm0
	LONG $0xc158fac5                           // ?                                    // vaddss	xmm0, xmm0, xmm1
	MOVQ 0x28(SP), AX                          // <--                                  // mov	rax, qword ptr [rsp + 40]
	QUAD $0x0000009824843948                   // CMPQ AX, 0x98(SP)                    // cmp	qword ptr [rsp + 152], rax
	JE   LBB1_16                               // <--                                  // je	.LBB1_16
	MOVQ 0x40(SP), AX                          // <--                                  // mov	rax, qword ptr [rsp + 64]
	MOVQ 0x68(SP), DX                          // <--                                  // mov	rdx, qword ptr [rsp + 104]
	MOVQ 0x30(SP), DI                          // <--                                  // mov	rdi, qword ptr [rsp + 48]
	MOVQ 0(SP), R13                            // <--                                  // mov	r13, qword ptr [rsp]
	MOVQ 0x20(SP), R14                         // <--                                  // mov	r14, qword ptr [rsp + 32]
	MOVQ 0x60(SP), CX                          // <--                                  // mov	rcx, qword ptr [rsp + 96]

LBB1_24:
	LONG $0x0810fac5                           // ADCB CL, 0(AX)                       // vmovss	xmm1, dword ptr [rax]
	LONG $0xb971c2c4; WORD $0x9544; BYTE $0x00 // ?                                    // vfmadd231ss	xmm0, xmm1, dword ptr [r13 + 4*rdx]
	WORD $0xff48; BYTE $0xc2                   // INCQ DX                              // inc	rdx
	WORD $0x0148; BYTE $0xc8                   // ADDQ CX, AX                          // add	rax, rcx
	WORD $0x3949; BYTE $0xd6                   // CMPQ DX, R14                         // cmp	r14, rdx
	JNE  LBB1_24                               // <--                                  // jne	.LBB1_24
	JMP  LBB1_25                               // <--                                  // jmp	.LBB1_25
