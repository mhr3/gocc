//go:build !noasm && arm64
// Code generated by gocc -- DO NOT EDIT.

package simd

import "unsafe"

//go:noescape
func f32_axpy(x unsafe.Pointer, y unsafe.Pointer, size int, alpha float32)

//go:noescape
func f32_matmul(dst unsafe.Pointer, m unsafe.Pointer, n unsafe.Pointer, dims uint64)
