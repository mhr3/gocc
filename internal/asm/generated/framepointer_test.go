package generated_test

import (
	"testing"
	"unsafe"

	"github.com/mhr3/gocc/internal/asm/generated"
)

// framePointerKept reports whether the frame pointer survives call, which has
// to be the Go function that calls into the assembly.
//
// A Go function saves the frame pointer of its caller on the stack, in the
// 8 bytes below its own frame on arm64, and reloads it when it returns. Those
// bytes are the top of what the assembler allocates for the function it
// calls. Assembly that writes there changes nothing for itself or its caller;
// the caller's caller gets the wrong register back, and whatever walks the
// frame pointers from below it (the execution tracer, a block or mutex
// profile, perf) follows it.
//
//go:noinline
func framePointerKept(call func()) bool {
	// relative to a local: the stack may move during the call
	var mark byte
	before := generated.FramePointer() - uintptr(unsafe.Pointer(&mark))
	call()
	return generated.FramePointer()-uintptr(unsafe.Pointer(&mark)) == before
}

func TestCallsKeepFramePointer(t *testing.T) {
	p := []int64{3, -7, 11, 5, 9}
	text := make([]byte, 700)
	for i := range text {
		text[i] = byte(i)
	}

	for _, c := range []struct {
		name string
		call func()
	}{
		{"nested", func() { generated.Test_calls_nested(p, 3) }},
		{"tail, mode 0", func() { generated.Test_calls_tail(p, 0) }},
		{"tail, mode 1", func() { generated.Test_calls_tail(p, 1) }},
		{"tail, mode 2", func() { generated.Test_calls_tail(p, 2) }},
		{"big frame", func() { generated.Test_calls_big_frame(p) }},
		{"consts", func() { generated.Test_calls_consts(2.5, 7) }},
		{"local pointer", func() { generated.Test_calls_local_pointer(7) }},
		{"big frame, bytes", func() { generated.Test_calls_big_frame_bytes(text) }},
		{"big frame, string", func() { generated.Test_calls_big_frame_string(string(text)) }},
	} {
		if !framePointerKept(c.call) {
			t.Errorf("%s: the frame pointer of the calling function was written over", c.name)
		}
	}
}
