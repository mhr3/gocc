//go:build !noasm && arm64
// Code generated by gocc -- DO NOT EDIT.

package simd


//go:noescape
func uint8_simd_mul_sve_manual(input1 *byte, input2 *byte, output *byte, size uint64)
