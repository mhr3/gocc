package ascii

import (
	"strings"
)

func ContainsFold(str, substr string) bool {
	n := len(substr)
	switch n {
	case 0:
		return true
	case 1:
		b := substr[0]
		if strings.IndexByte(str, b) >= 0 {
			return true
		} else if b >= 'A' && b <= 'Z' {
			return strings.IndexByte(str, b+0x20) >= 0
		} else if b >= 'a' && b <= 'z' {
			return strings.IndexByte(str, b-0x20) >= 0
		}
		return false
	case len(str):
		return EqualFold(str, substr)
	}

	return contains_fold(str, substr)
}
