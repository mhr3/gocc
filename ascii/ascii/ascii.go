package ascii

import "math/bits"

func indexBitGo(s []byte, mask byte) int {
	s = s[:len(s):len(s)]

	mask32 := uint32(mask)
	mask32 |= mask32 << 8
	mask32 |= mask32 << 16

	pos := 0
	// use all go tricks to make this fast
	for ; len(s) >= 8; pos, s = pos+8, s[8:] {
		_ = s[7]
		first32 := uint32(s[0]) | uint32(s[1])<<8 | uint32(s[2])<<16 | uint32(s[3])<<24
		second32 := uint32(s[4]) | uint32(s[5])<<8 | uint32(s[6])<<16 | uint32(s[7])<<24
		if (first32|second32)&mask32 != 0 {
			first32 &= mask32
			if first32 != 0 {
				return pos + bits.TrailingZeros32(first32)/8
			}
			second32 &= mask32
			return pos + 4 + bits.TrailingZeros32(second32)/8
		}
	}

	for i, b := range s {
		if b&mask != 0 {
			return pos + i
		}
	}
	return -1
}

func isAsciiGo(s []byte) bool {
	return indexBitGo(s, 0x80) == -1
}

// based on https://graphics.stanford.edu/~seander/bithacks.html#HasBetweenInWord
func hasLowercaseAsciiByte(x uint64) uint64 {
	const mult = ^uint64(0) / 255
	const m, n = 'a' - 1, 'z' + 1

	A := mult * (127 + n)
	B := x & (mult * 127)
	C := ^x
	D := mult * (127 - m)
	return (A - B) & C & (B + D) & (mult * 128)
}

func asciiFoldWord(x uint64) uint64 {
	mask := hasLowercaseAsciiByte(x)
	mask >>= 2
	return x - mask
}

func equalFoldGo(a, b string) bool {
	if len(a) != len(b) {
		return false
	}

	for len(a) >= 8 {
		_ = a[7]
		_ = b[7]

		// the compiler should be able to optimize this to two 64-bit loads
		a64 := uint64(a[0]) | uint64(a[1])<<8 | uint64(a[2])<<16 | uint64(a[3])<<24 |
			uint64(a[4])<<32 | uint64(a[5])<<40 | uint64(a[6])<<48 | uint64(a[7])<<56
		b64 := uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
			uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56

		if a64 != b64 {
			a64 = asciiFoldWord(a64)
			b64 = asciiFoldWord(b64)
			if a64 != b64 {
				return false
			}
		}
		a = a[8:]
		b = b[8:]
	}

	var a0, a1, b0, b1 uint32
	switch len(a) {
	case 7:
		fallthrough
	case 6:
		fallthrough
	case 5:
		// get the data using four 32-bit loads
		_, _ = a[3], b[3]
		a0 = uint32(a[0]) | uint32(a[1])<<8 | uint32(a[2])<<16 | uint32(a[3])<<24
		b0 = uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24

		idx := len(a) - 4
		a, b = a[idx:], b[idx:]
		_, _ = a[3], b[3]
		a1 = uint32(a[0]) | uint32(a[1])<<8 | uint32(a[2])<<16 | uint32(a[3])<<24
		b1 = uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
	case 4:
		_ = b[3]
		a0 = uint32(a[0]) | uint32(a[1])<<8 | uint32(a[2])<<16 | uint32(a[3])<<24
		b0 = uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
	case 3:
		a0 = uint32(a[0]) | uint32(a[1])<<8 | uint32(a[2])<<16
		_ = b[2]
		b0 = uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16
	case 2:
		a0 = uint32(a[0]) | uint32(a[1])<<8
		_ = b[1]
		b0 = uint32(b[0]) | uint32(b[1])<<8
	case 1:
		a0 = uint32(a[0])
		b0 = uint32(b[0])
	case 0:
		return true
	}

	a64 := uint64(a0) | uint64(a1)<<32
	b64 := uint64(b0) | uint64(b1)<<32
	if a64 == b64 {
		return true
	}

	return asciiFoldWord(a64) == asciiFoldWord(b64)
}

func indexFoldGo[T string | []byte](s T, substr T) int {
	if len(substr) == 0 {
		return 0
	} else if len(substr) > len(s) {
		return -1
	}

	first := substr[0]
	complement := first
	if first >= 'A' && first <= 'Z' {
		complement += 0x20
	} else if first >= 'a' && first <= 'z' {
		complement -= 0x20
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		b := byte(s[i])
		if b == first || b == complement {
			prefix := s[i:]
			if equalFoldGo(string(prefix[:len(substr)]), string(substr)) {
				return i
			}
		}
	}
	return -1
}
