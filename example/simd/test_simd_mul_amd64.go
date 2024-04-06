//go:build !noasm && amd64

package simd

import (
	"unsafe"

	"golang.org/x/sys/cpu"
)

var (
	useAVX2   = cpu.X86.HasAVX2
	useAVX512 = cpu.X86.HasAVX512
)

func uint8_simd_mul(input1 unsafe.Pointer, input2 unsafe.Pointer, output unsafe.Pointer, size uint64) {
	if useAVX512 {
		uint8_simd_mul_avx512(input1, input2, output, size)
	} else if useAVX2 {
		uint8_simd_mul_avx2(input1, input2, output, size)
	} else {
		uint8_simd_mul_sse(input1, input2, output, size)
	}
}

// just to make compiling tests easier
func uint8_simd_mul_sve(input1 unsafe.Pointer, input2 unsafe.Pointer, output unsafe.Pointer, size uint64) {
	panic("not implemented")
}

func uint8_simd_mul_sve_manual(input1 unsafe.Pointer, input2 unsafe.Pointer, output unsafe.Pointer, size uint64) {
	panic("not implemented")
}

func uint8_simd_mul_neon(input1 unsafe.Pointer, input2 unsafe.Pointer, output unsafe.Pointer, size uint64) {
	panic("not implemented")
}
