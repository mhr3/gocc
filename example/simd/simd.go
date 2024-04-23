package simd

import (
	"bytes"
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/cpu"
)

var (
	avx2           = cpu.X86.HasAVX2 && cpu.X86.HasFMA
	neon           = cpu.ARM64.HasASIMD
	useAccelerated = avx2 || neon
)

var (
	errZeroLength        = errors.New("mat: zero length in matrix dimension")
	errNegativeDimension = errors.New("mat: negative dimension")
	errShape             = errors.New("mat: dimension mismatch")
)

// Matmul multiplies matrix M by N and writes the result into dst
func Matmul(dst, m, n *Matrix) {
	switch {
	case useAccelerated:
		f32_matmul(unsafe.SliceData(dst.Data), unsafe.SliceData(m.Data), unsafe.SliceData(n.Data),
			dimensionsOf(m.Rows, m.Cols, n.Rows, n.Cols),
		)
	default:
		matmul(dst.Data, m.Data, n.Data, m.Rows, m.Cols, n.Rows, n.Cols)
	}
}

func dimensionsOf(mr, mc, nr, nc int) (v uint64) {
	v |= uint64(mr) << 0
	v |= uint64(mc) << 16
	v |= uint64(nr) << 32
	v |= uint64(nc) << 48
	return v
}

// ---------------------------------- Matrix ----------------------------------

// Matrix represents a Matrix using the conventional storage scheme.
type Matrix struct {
	Data []float32 `json:"data"`
	Rows int       `json:"rows"`
	Cols int       `json:"cols"`
}

// NewMatrix creates a new dense matrix
func NewMatrix(r, c int, data []float32) Matrix {
	if r <= 0 || c <= 0 {
		if r == 0 || c == 0 {
			panic(errZeroLength)
		}
		panic(errNegativeDimension)
	}

	if data != nil && r*c != len(data) {
		panic(errShape)
	}

	if data == nil {
		data = make([]float32, r*c)
	}

	return Matrix{
		Rows: r,
		Cols: c,
		Data: data,
	}
}

// String returns a string representation of the matrix
func (mx *Matrix) String() string {
	if mx == nil {
		return "nil"
	}

	var buf bytes.Buffer
	for i := 0; i < mx.Rows; i++ {
		buf.WriteString("[")
		for j := 0; j < mx.Cols; j++ {
			buf.WriteString(fmt.Sprintf("%g", mx.Data[i*mx.Cols+j]))
			if j < mx.Cols-1 {
				buf.WriteString(", ")
			}
		}
		buf.WriteString("]")
		if i < mx.Rows-1 {
			buf.WriteString("")
		}
	}
	return buf.String()
}
