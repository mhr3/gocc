//go:build !noasm && amd64
// Code generated by gocc -- DO NOT EDIT.

package ascii


//go:noescape
func isAsciiSse(src string) bool
