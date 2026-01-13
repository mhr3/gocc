	.section	.rodata.cst16,"aM",@progbits,16
	.p2align	4, 0x0                          // -- Begin function test_func
.LCPI0_0:
	.xword	32                              // 0x20
	.xword	48                              // 0x30
.LCPI0_1:
	.byte	1                               // 0x1
	.byte	4                               // 0x4
	.byte	16                              // 0x10
	.byte	64                              // 0x40
	.byte	1                               // 0x1
	.byte	4                               // 0x4
	.byte	16                              // 0x10
	.byte	64                              // 0x40
	.byte	1                               // 0x1
	.byte	4                               // 0x4
	.byte	16                              // 0x10
	.byte	64                              // 0x40
	.byte	1                               // 0x1
	.byte	4                               // 0x4
	.byte	16                              // 0x10
	.byte	64                              // 0x40
	.text
	.globl	test_func
	.p2align	2
	.type	test_func,@function
test_func:                              // @test_func
// %bb.0:
	mov	x0, xzr
	ret
.Lfunc_end0:
	.size	test_func, .Lfunc_end0-test_func
