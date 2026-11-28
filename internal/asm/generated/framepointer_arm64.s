#include "textflag.h"

// func FramePointer() uintptr
TEXT ·FramePointer(SB), NOSPLIT|NOFRAME, $0-8
	MOVD R29, ret+0(FP)
	RET
