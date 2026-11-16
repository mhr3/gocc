package generated_test

import "github.com/mhr3/gocc/internal/asm/generated"

// Go keeps the stack pointer 16-byte aligned.
const stackPlacements = 4

var stackVariants = []stackVariant{
	{name: "neon", width: 16, available: true},
}

// stackKernel calls a fixture. The calls are direct ones so that the compiler
// sees the argument does not escape and leaves the caller's copy of the
// input on the stack. It has no stack check of its own, so that a stack that
// moves between the caller and the fixture was moved by the fixture.
//
//go:nosplit
func stackKernel(_, fixture int, data []byte) uint64 {
	switch fixture {
	case 0:
		return generated.Stack_spill_neon(data)
	case 1:
		return generated.Stack_operand_neon(data)
	case 2:
		return generated.Stack_select_neon(data)
	case 3:
		return generated.Stack_consts_neon(data)
	case 4:
		return generated.Stack_pointer_neon(data)
	case 5:
		return generated.Stack_walk_neon(data)
	case 6:
		return generated.Stack_overaligned_neon(data)
	case 7:
		return generated.Stack_frame_neon(data)
	}
	panic("no such fixture")
}
