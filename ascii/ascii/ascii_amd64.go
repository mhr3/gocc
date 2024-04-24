//go:build !noasm && amd64

package ascii

import "golang.org/x/sys/cpu"

var useAVX2 = cpu.X86.HasAVX2

func IsASCII(s string) bool {
	if useAVX2 {
		return isAsciiAvx(s)
	}
	return isAsciiSse(s)
}

func IndexNonASCII(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return i
		}
	}
	return -1
}
