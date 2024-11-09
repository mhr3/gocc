package ascii

import (
	"strings"
)

func ContainsFold(str, substr string) bool {
	return IndexFold(str, substr) >= 0
}

func IndexFold(str, substr string) int {
	switch l := len(substr); l {
	case 0:
		return 0
	case 1:
		b := substr[0]
		if idx := strings.IndexByte(str, b); idx >= 0 {
			return idx
		} else if isUpper(b) || isLower(b) {
			return strings.IndexByte(str, b^0x20)
		}
		return -1
	case len(str):
		if EqualFold(str, substr) {
			return 0
		}
		return -1
	}

	return index_fold_simd(str, substr)
}

func isUpper(b byte) bool {
	return b >= 'A' && b <= 'Z'
}

func isLower(b byte) bool {
	return b >= 'a' && b <= 'z'
}
