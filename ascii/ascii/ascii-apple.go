//go:build !noasm && arm64 && enabled
// Code generated by gocc -- DO NOT EDIT.

package ascii


//go:noescape
func IndexNonASCII(data string) int

//go:noescape
func IsASCII(data string) bool
