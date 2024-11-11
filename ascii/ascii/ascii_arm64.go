package ascii

func ContainsFold(str, substr string) bool {
	return IndexFold(str, substr) >= 0
}

/*
func isUpper(b byte) bool {
	return b >= 'A' && b <= 'Z'
}

func isLower(b byte) bool {
	return b >= 'a' && b <= 'z'
}
*/
