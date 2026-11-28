package generated_test

import (
	"golang.org/x/sys/cpu"

	"github.com/mhr3/gocc/internal/asm/generated"
)

// Go keeps the stack pointer 8-byte aligned.
const stackPlacements = 8

const (
	variantSSE2 stackVariantID = iota
	variantSSE4
	variantAVX2
	variantAVX512
)

var stackVariants = []stackVariant{
	{id: variantSSE2, name: "sse2", width: 16, available: true},
	{id: variantSSE4, name: "sse4", width: 16, available: cpu.X86.HasSSE42},
	{id: variantAVX2, name: "avx2", width: 32, available: cpu.X86.HasAVX2},
	{id: variantAVX512, name: "avx512", width: 64, available: cpu.X86.HasAVX512F && cpu.X86.HasAVX512DQ && cpu.X86.HasAVX512BW},
}

// stackKernel calls a fixture. The calls are direct ones so that the compiler
// sees the argument does not escape and leaves the caller's copy of the
// input on the stack. It has no stack check of its own, so that a stack that
// moves between the caller and the fixture was moved by the fixture. It is a
// frame of its own, which is where a fixture that writes over the saved frame
// pointer does so: stackLeaf sees it once this function has returned.
//
//go:nosplit
//go:noinline
func stackKernel(variant stackVariantID, fixture stackFixtureID, data []byte) uint64 {
	switch variant {
	case variantSSE2:
		switch fixture {
		case fixtureSpill:
			return generated.Stack_spill_sse2(data)
		case fixtureOperand:
			return generated.Stack_operand_sse2(data)
		case fixtureSelect:
			return generated.Stack_select_sse2(data)
		case fixtureConsts:
			return generated.Stack_consts_sse2(data)
		case fixturePointer:
			return generated.Stack_pointer_sse2(data)
		case fixtureWalk:
			return generated.Stack_walk_sse2(data)
		case fixtureOveraligned:
			return generated.Stack_overaligned_sse2(data)
		case fixtureFrame:
			return generated.Stack_frame_sse2(data)
		}
	case variantSSE4:
		switch fixture {
		case fixtureSpill:
			return generated.Stack_spill_sse4(data)
		case fixtureOperand:
			return generated.Stack_operand_sse4(data)
		case fixtureSelect:
			return generated.Stack_select_sse4(data)
		case fixtureConsts:
			return generated.Stack_consts_sse4(data)
		case fixturePointer:
			return generated.Stack_pointer_sse4(data)
		case fixtureWalk:
			return generated.Stack_walk_sse4(data)
		case fixtureOveraligned:
			return generated.Stack_overaligned_sse4(data)
		case fixtureFrame:
			return generated.Stack_frame_sse4(data)
		}
	case variantAVX2:
		switch fixture {
		case fixtureSpill:
			return generated.Stack_spill_avx2(data)
		case fixtureOperand:
			return generated.Stack_operand_avx2(data)
		case fixtureSelect:
			return generated.Stack_select_avx2(data)
		case fixtureConsts:
			return generated.Stack_consts_avx2(data)
		case fixturePointer:
			return generated.Stack_pointer_avx2(data)
		case fixtureWalk:
			return generated.Stack_walk_avx2(data)
		case fixtureOveraligned:
			return generated.Stack_overaligned_avx2(data)
		case fixtureFrame:
			return generated.Stack_frame_avx2(data)
		}
	case variantAVX512:
		switch fixture {
		case fixtureSpill:
			return generated.Stack_spill_avx512(data)
		case fixtureOperand:
			return generated.Stack_operand_avx512(data)
		case fixtureSelect:
			return generated.Stack_select_avx512(data)
		case fixtureConsts:
			return generated.Stack_consts_avx512(data)
		case fixturePointer:
			return generated.Stack_pointer_avx512(data)
		case fixtureWalk:
			return generated.Stack_walk_avx512(data)
		case fixtureOveraligned:
			return generated.Stack_overaligned_avx512(data)
		case fixtureFrame:
			return generated.Stack_frame_avx512(data)
		}
	}
	panic("no such fixture")
}
