//go:build !noasm && amd64

package simd

import (
	"unsafe"

	"golang.org/x/sys/cpu"
)

var useAVX2 = cpu.X86.HasAVX2

func uint8_simd_mul(input1 unsafe.Pointer, input2 unsafe.Pointer, output unsafe.Pointer, size uint64) {
	if useAVX2 {
		uint8_simd_mul_avx2(input1, input2, output, size)
		return
	}
	uint8_simd_mul_sse(input1, input2, output, size)
}
