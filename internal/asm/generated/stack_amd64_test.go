package generated_test

import (
	"golang.org/x/sys/cpu"

	"github.com/mhr3/gocc/internal/asm/generated"
)

// Go keeps the stack pointer 8-byte aligned.
const stackPlacements = 8

var stackVariants = []stackVariant{
	{name: "sse2", width: 16, available: true},
	{name: "sse4", width: 16, available: cpu.X86.HasSSE42},
	{name: "avx2", width: 32, available: cpu.X86.HasAVX2},
	{name: "avx512", width: 64, available: cpu.X86.HasAVX512F && cpu.X86.HasAVX512DQ && cpu.X86.HasAVX512BW},
}

// stackKernel calls a fixture. The calls are direct ones so that the compiler
// sees the argument does not escape and leaves the caller's copy of the
// input on the stack. It has no stack check of its own, so that a stack that
// moves between the caller and the fixture was moved by the fixture.
//
//go:nosplit
func stackKernel(variant, fixture int, data []byte) uint64 {
	switch variant {
	case 0:
		switch fixture {
		case 0:
			return generated.Stack_spill_sse2(data)
		case 1:
			return generated.Stack_operand_sse2(data)
		case 2:
			return generated.Stack_select_sse2(data)
		case 3:
			return generated.Stack_consts_sse2(data)
		case 4:
			return generated.Stack_pointer_sse2(data)
		case 5:
			return generated.Stack_walk_sse2(data)
		case 6:
			return generated.Stack_overaligned_sse2(data)
		case 7:
			return generated.Stack_frame_sse2(data)
		}
	case 1:
		switch fixture {
		case 0:
			return generated.Stack_spill_sse4(data)
		case 1:
			return generated.Stack_operand_sse4(data)
		case 2:
			return generated.Stack_select_sse4(data)
		case 3:
			return generated.Stack_consts_sse4(data)
		case 4:
			return generated.Stack_pointer_sse4(data)
		case 5:
			return generated.Stack_walk_sse4(data)
		case 6:
			return generated.Stack_overaligned_sse4(data)
		case 7:
			return generated.Stack_frame_sse4(data)
		}
	case 2:
		switch fixture {
		case 0:
			return generated.Stack_spill_avx2(data)
		case 1:
			return generated.Stack_operand_avx2(data)
		case 2:
			return generated.Stack_select_avx2(data)
		case 3:
			return generated.Stack_consts_avx2(data)
		case 4:
			return generated.Stack_pointer_avx2(data)
		case 5:
			return generated.Stack_walk_avx2(data)
		case 6:
			return generated.Stack_overaligned_avx2(data)
		case 7:
			return generated.Stack_frame_avx2(data)
		}
	case 3:
		switch fixture {
		case 0:
			return generated.Stack_spill_avx512(data)
		case 1:
			return generated.Stack_operand_avx512(data)
		case 2:
			return generated.Stack_select_avx512(data)
		case 3:
			return generated.Stack_consts_avx512(data)
		case 4:
			return generated.Stack_pointer_avx512(data)
		case 5:
			return generated.Stack_walk_avx512(data)
		case 6:
			return generated.Stack_overaligned_avx512(data)
		case 7:
			return generated.Stack_frame_avx512(data)
		}
	}
	panic("no such fixture")
}
