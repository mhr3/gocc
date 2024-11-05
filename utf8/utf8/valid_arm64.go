package utf8

import "unicode/utf8"

func Valid(s string) bool {
	if len(s) < 16 {
		return utf8.ValidString(s)
	}

	return utf8_valid_range(s)
}
