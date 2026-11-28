#include "textflag.h"

// func FramePointer() uintptr
TEXT ·FramePointer(SB), NOSPLIT, $0-8
	MOVQ BP, ret+0(FP)
	RET
