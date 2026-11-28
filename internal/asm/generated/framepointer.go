package generated

// FramePointer returns the frame pointer register as its caller has it. It is
// written by hand, for the tests: a Go function saves the frame pointer of its
// caller on the stack and reloads it when it returns, so assembly that writes
// over the saved copy shows as a changed register one return later.
func FramePointer() uintptr
