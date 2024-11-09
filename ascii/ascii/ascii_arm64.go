package ascii

func ContainsFold(str, substr string) bool {
	return IndexFold(str, substr) >= 0
}

func IndexFold(str, substr string) int {
	switch l := len(substr); l {
	case 0:
		return 0
	case len(str):
		if EqualFold(str, substr) {
			return 0
		}
		return -1
	}

	return index_fold_simd(str, substr)
}

/*
func isUpper(b byte) bool {
	return b >= 'A' && b <= 'Z'
}

func isLower(b byte) bool {
	return b >= 'a' && b <= 'z'
}
*/
