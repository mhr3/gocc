//go:build !noasm && amd64
// Code generated by gocc -- DO NOT EDIT.

package simd


//go:noescape
func uint8_simd_mul_avx2(input1 *byte, input2 *byte, output *byte, size uint64)
