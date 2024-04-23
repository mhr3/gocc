	.text
	.file	"test_simd_mul.c"
	.globl	uint8_simd_mul                  // -- Begin function uint8_simd_mul
	.p2align	2
	.type	uint8_simd_mul,@function
uint8_simd_mul:                         // @uint8_simd_mul
// %bb.0:
	stp	x29, x30, [sp, #-16]!           // 16-byte Folded Spill
	cmp	w3, #1
	mov	x29, sp
	b.lt	.LBB0_5
// %bb.1:
	and	x8, x3, #0xffffffff
	cmp	x8, #8
	b.hs	.LBB0_6
// %bb.2:
	mov	x9, xzr
.LBB0_3:
	add	x10, x2, x9
	add	x11, x1, x9
	add	x12, x0, x9
	sub	x8, x8, x9
.LBB0_4:                                // =>This Inner Loop Header: Depth=1
	ldrb	w9, [x12], #1
	ldrb	w13, [x11], #1
	subs	x8, x8, #1
	mul	w9, w13, w9
	strb	w9, [x10], #1
	b.ne	.LBB0_4
.LBB0_5:
	ldp	x29, x30, [sp], #16             // 16-byte Folded Reload
	ret
.LBB0_6:
	mov	x9, xzr
	sub	x10, x2, x0
	cmp	x10, #32
	b.lo	.LBB0_3
// %bb.7:
	sub	x10, x2, x1
	cmp	x10, #32
	b.lo	.LBB0_3
// %bb.8:
	cmp	x8, #32
	b.hs	.LBB0_10
// %bb.9:
	mov	x9, xzr
	b	.LBB0_14
.LBB0_10:
	and	x10, x3, #0x1f
	add	x11, x0, #16
	sub	x9, x8, x10
	add	x12, x1, #16
	add	x13, x2, #16
	mov	x14, x9
.LBB0_11:                               // =>This Inner Loop Header: Depth=1
	ldp	q0, q1, [x11, #-16]
	add	x11, x11, #32
	subs	x14, x14, #32
	ldp	q2, q3, [x12, #-16]
	add	x12, x12, #32
	mul	v0.16b, v2.16b, v0.16b
	mul	v1.16b, v3.16b, v1.16b
	stp	q0, q1, [x13, #-16]
	add	x13, x13, #32
	b.ne	.LBB0_11
// %bb.12:
	cbz	x10, .LBB0_5
// %bb.13:
	cmp	x10, #8
	b.lo	.LBB0_3
.LBB0_14:
	and	x10, x3, #0x7
	add	x11, x0, x9
	add	x14, x9, x10
	add	x12, x1, x9
	add	x13, x2, x9
	sub	x9, x8, x10
	sub	x14, x14, x8
.LBB0_15:                               // =>This Inner Loop Header: Depth=1
	ldr	d0, [x11], #8
	ldr	d1, [x12], #8
	adds	x14, x14, #8
	mul	v0.8b, v1.8b, v0.8b
	str	d0, [x13], #8
	b.ne	.LBB0_15
// %bb.16:
	cbnz	x10, .LBB0_3
	b	.LBB0_5
.Lfunc_end0:
	.size	uint8_simd_mul, .Lfunc_end0-uint8_simd_mul
                                        // -- End function
	.ident	"Apple clang version 15.0.0 (clang-1500.3.9.4)"
	.section	".note.GNU-stack","",@progbits
	.addrsig
