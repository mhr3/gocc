package utf8

import "unicode/utf8"

func Valid(s string) bool {
	if len(s) < 16 {
		return utf8.ValidString(s)
	}

	res := utf8_range(s)
	if res < 0 {
		return false
	}
	if res == 0 {
		return true
	}

	// Only need to check the last res bytes.
	s = s[len(s)-res:]
	return utf8.ValidString(s)
}
