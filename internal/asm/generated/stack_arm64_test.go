package generated_test

import "github.com/mhr3/gocc/internal/asm/generated"

// Go keeps the stack pointer 16-byte aligned.
const stackPlacements = 4

const variantNEON stackVariantID = iota

var stackVariants = []stackVariant{
	{id: variantNEON, name: "neon", width: 16, available: true},
}

// stackKernel calls a fixture. The calls are direct ones so that the compiler
// sees the argument does not escape and leaves the caller's copy of the
// input on the stack. It has no stack check of its own, so that a stack that
// moves between the caller and the fixture was moved by the fixture.
//
//go:nosplit
func stackKernel(_ stackVariantID, fixture stackFixtureID, data []byte) uint64 {
	switch fixture {
	case fixtureSpill:
		return generated.Stack_spill_neon(data)
	case fixtureOperand:
		return generated.Stack_operand_neon(data)
	case fixtureSelect:
		return generated.Stack_select_neon(data)
	case fixtureConsts:
		return generated.Stack_consts_neon(data)
	case fixturePointer:
		return generated.Stack_pointer_neon(data)
	case fixtureWalk:
		return generated.Stack_walk_neon(data)
	case fixtureOveraligned:
		return generated.Stack_overaligned_neon(data)
	case fixtureFrame:
		return generated.Stack_frame_neon(data)
	}
	panic("no such fixture")
}
