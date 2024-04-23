package simd

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/cpu"
)

/*
cpu: 13th Gen Intel(R) Core(TM) i7-13700K
BenchmarkAXPY/std-24         	287770197	         4.048 ns/op	       0 B/op	       0 allocs/op
BenchmarkAXPY/asm-24         	422536102	         2.870 ns/op	       0 B/op	       0 allocs/op
*/
func BenchmarkAXPY(b *testing.B) {
	x := []float32{2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2}
	y := []float32{3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3}

	b.Run("std", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			axpy(x, y, 3)
		}
	})

	b.Run("asm", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			f32_axpy(
				unsafe.Pointer(&x[0]),
				unsafe.Pointer(&y[0]),
				len(x), 3.0,
			)
		}
	})
}

/*
cpu: 13th Gen Intel(R) Core(TM) i7-13700K
BenchmarkMatmul/4x4-std-24         	24242570	        49.69 ns/op	       0 B/op	       0 allocs/op
BenchmarkMatmul/4x4-asm-24         	26667140	        45.19 ns/op	       0 B/op	       0 allocs/op
BenchmarkMatmul/8x8-std-24         	 4545457	       265.2 ns/op	       0 B/op	       0 allocs/op
BenchmarkMatmul/8x8-asm-24         	21428494	        50.40 ns/op	       0 B/op	       0 allocs/op
BenchmarkMatmul/16x16-std-24       	 1000000	      1267 ns/op	       0 B/op	       0 allocs/op
BenchmarkMatmul/16x16-asm-24       	 7017567	       165.8 ns/op	       0 B/op	       0 allocs/op
BenchmarkMatmul/32x32-std-24       	  129031	      9893 ns/op	       0 B/op	       0 allocs/op
BenchmarkMatmul/32x32-asm-24       	 1854714	       623.2 ns/op	       0 B/op	       0 allocs/op
BenchmarkMatmul/64x64-std-24       	   18644	     64486 ns/op	       0 B/op	       0 allocs/op
BenchmarkMatmul/64x64-asm-24       	  510646	      2408 ns/op	       0 B/op	       0 allocs/op
*/
func BenchmarkMatmul(b *testing.B) {
	for _, size := range []int{4, 8, 16, 32, 64, 128, 256, 512} {
		m := newTestMatrix(size, size)
		n := newTestMatrix(size, size)
		o := newTestMatrix(size, size)

		b.Run(fmt.Sprintf("%dx%d-std", size, size), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				matmul(o.Data, m.Data, n.Data, m.Rows, m.Cols, n.Rows, n.Cols)
			}
		})

		b.Run(fmt.Sprintf("%dx%d-asm", size, size), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				f32_matmul(
					unsafe.Pointer(&o.Data[0]), unsafe.Pointer(&m.Data[0]), unsafe.Pointer(&n.Data[0]),
					dimensionsOf(m.Rows, m.Cols, n.Rows, n.Cols),
				)
			}
		})
	}
}

func TestGenericMatmul(t *testing.T) {
	x := []float32{1, 2, 3, 4}
	y := []float32{5, 6, 7, 8}
	o := make([]float32, 4)

	matmul(o, x, y, 2, 2, 2, 2)
	assert.Equal(t, []float32{19, 22, 43, 50}, o)
}

func TestMatmulNative(t *testing.T) {
	x := []float32{1, 2, 3, 4}
	y := []float32{5, 6, 7, 8}
	o := make([]float32, 4)

	if !useAccelerated {
		t.Skip("hardware acceleration is not available")
	}

	f32_matmul(
		unsafe.Pointer(&o[0]), unsafe.Pointer(&x[0]), unsafe.Pointer(&y[0]),
		dimensionsOf(2, 2, 2, 2),
	)

	assert.Equal(t, []float32{19, 22, 43, 50}, o)
}

func TestMatmul(t *testing.T) {
	x := Matrix{Rows: 2, Cols: 2, Data: []float32{1, 2, 3, 4}}
	y := Matrix{Rows: 2, Cols: 2, Data: []float32{5, 6, 7, 8}}
	o := Matrix{Rows: 2, Cols: 2, Data: make([]float32, 4)}

	Matmul(&o, &x, &y)
	assert.Equal(t, []float32{19, 22, 43, 50}, o.Data)
}

func TestUintMul(t *testing.T) {
	checkResult := func(input1, input2, result []uint8) {
		t.Helper()
		expected := make([]uint8, len(input1))
		uint8_mul_go(input1, input2, expected)
		assert.Equal(t, expected, result)
	}

	cases := []struct {
		Name      string
		Fn        func(unsafe.Pointer, unsafe.Pointer, unsafe.Pointer, uint64)
		Supported bool
	}{
		{"simd", uint8_simd_mul, true},
		{"sse", uint8_simd_mul_sse, cpu.X86.HasSSE2},
		{"avx2", uint8_simd_mul_avx2, cpu.X86.HasAVX2},
		{"avx512", uint8_simd_mul_avx512, cpu.X86.HasAVX512F},
		{"neon", uint8_simd_mul_neon, cpu.ARM64.HasASIMD},
		{"sve", uint8_simd_mul_sve, cpu.ARM64.HasSVE},
		{"sve2", uint8_simd_mul_sve2, cpu.ARM64.HasSVE2},
		{"sve-intrinsics", uint8_simd_mul_sve_manual, cpu.ARM64.HasSVE},
	}

	for _, sz := range []int{1, 15, 44, 100, 10000} {
		for _, c := range cases {
			t.Run(fmt.Sprintf("%d-%s", sz, c.Name), func(t *testing.T) {
				if !c.Supported {
					t.Skipf("%s not supported", c.Name)
				}
				input1, input2, dst := prepareTestUint8Input(sz)

				c.Fn(unsafe.Pointer(&input1[0]), unsafe.Pointer(&input2[0]), unsafe.Pointer(&dst[0]), uint64(sz))

				checkResult(input1, input2, dst)
			})
		}
	}
}

func TestMemcmp(t *testing.T) {
	sizes := []int{1, 15, 44, 100, 10000, 64 * 1024}
	for _, size := range sizes {
		t.Run(fmt.Sprintf("%d", size), func(t *testing.T) {
			if !cpu.ARM64.HasSVE {
				t.Skip("SVE not supported")
			}
			input1, input2, _ := prepareTestUint8Input(size)

			resSve := memcmp_sve(input1, input2)
			resGo := bytes.Compare(input1, input2)

			assert.Equal(t, resGo, resSve)

			input1[rand.Intn(size)] = byte(rand.Int())

			resSve = memcmp_sve(input1, input2)
			resGo = bytes.Compare(input1, input2)

			assert.Equal(t, resGo, resSve)
		})
	}
}

func prepareTestUint8Input(sz int) ([]uint8, []uint8, []uint8) {
	input1 := make([]uint8, sz)
	input2 := make([]uint8, sz)
	dst := make([]uint8, sz)

	for i := 0; i < sz; i++ {
		input1[i] = uint8(i)
		input2[i] = uint8(i)
	}

	return input1, input2, dst
}

// newTestMatrix creates a new matrix
func newTestMatrix(r, c int) *Matrix {
	mx := NewMatrix(r, c, nil)
	for i := 0; i < len(mx.Data); i++ {
		mx.Data[i] = 2
	}
	return &mx
}

func uint8_mul_go(input1, input2, output []uint8) {
	for i := range input1 {
		output[i] = input1[i] * input2[i]
	}
}

func BenchmarkUintMul(b *testing.B) {
	cases := []struct {
		Name      string
		Fn        func(unsafe.Pointer, unsafe.Pointer, unsafe.Pointer, uint64)
		Supported bool
	}{
		//{"simd", uint8_simd_mul, true},
		{"sse", uint8_simd_mul_sse, cpu.X86.HasSSE2},
		{"avx2", uint8_simd_mul_avx2, cpu.X86.HasAVX2},
		{"avx512", uint8_simd_mul_avx512, cpu.X86.HasAVX512F},
		{"neon", uint8_simd_mul_neon, cpu.ARM64.HasASIMD},
		{"sve", uint8_simd_mul_sve, cpu.ARM64.HasSVE},
		{"sve2", uint8_simd_mul_sve2, cpu.ARM64.HasSVE2},
		{"sve-intrinsics", uint8_simd_mul_sve_manual, cpu.ARM64.HasSVE},
	}

	sizes := []int{1, 15, 44, 100, 10000, 64 * 1024}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("go-%d", size), func(b *testing.B) {
			input1, input2, output := prepareTestUint8Input(size)

			startTime := time.Now()
			for i := 0; i < b.N; i++ {
				uint8_mul_go(input1, input2, output)
			}
			b.ReportMetric(float64(b.N*size)/float64(time.Since(startTime).Microseconds()), "MB/s")
		})

		for _, c := range cases {
			b.Run(fmt.Sprintf("%s-%d", c.Name, size), func(b *testing.B) {
				if !c.Supported {
					b.Skipf("%s not supported", c.Name)
				}

				input1, input2, output := prepareTestUint8Input(size)

				startTime := time.Now()
				for i := 0; i < b.N; i++ {
					c.Fn(unsafe.Pointer(&input1[0]), unsafe.Pointer(&input2[0]), unsafe.Pointer(&output[0]), uint64(size))
				}
				b.ReportMetric(float64(b.N*size)/float64(time.Since(startTime).Microseconds()), "MB/s")
			})
		}
	}
}

func BenchmarkMemcpy(b *testing.B) {
	sizes := []int{1, 15, 44, 100, 10000, 64 * 1024}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("go-%d", size), func(b *testing.B) {
			input1, input2, _ := prepareTestUint8Input(size)

			startTime := time.Now()
			for i := 0; i < b.N; i++ {
				bytes.Compare(input1, input2)
			}
			b.ReportMetric(float64(b.N*size)/float64(time.Since(startTime).Microseconds()), "MB/s")
		})

		b.Run(fmt.Sprintf("sve-%d", size), func(b *testing.B) {
			if !cpu.ARM64.HasSVE {
				b.Skip("SVE not supported")
			}
			input1, input2, _ := prepareTestUint8Input(size)

			startTime := time.Now()
			for i := 0; i < b.N; i++ {
				memcmp_sve(input1, input2)
			}
			b.ReportMetric(float64(b.N*size)/float64(time.Since(startTime).Microseconds()), "MB/s")
		})
	}
}
