package generated_test

import (
	"testing"

	"github.com/mhr3/gocc/internal/asm/generated"
	"github.com/stretchr/testify/assert"
)

func TestVoidRetFn(t *testing.T) {
	var res int
	generated.Test_fn_4818_0(1, 2, 3, &res)

	assert.Equal(t, 5, res)
}

func TestVoidByteParamsFn(t *testing.T) {
	// The C body has no observable result, but calling it exercises the
	// three-byte-parameter ABI and its void return path.
	generated.Test_fn_111_0(3, 2, 12)
}

func TestByteParamsByteRetFn(t *testing.T) {
	res := generated.Test_fn_111_1(3, 2, 12)

	assert.EqualValues(t, 38, res)
}

func TestByteInt32ParamsByteRetFn(t *testing.T) {
	res := generated.Test_fn_1114_1(3, 2, 12, 4)

	assert.EqualValues(t, 42, res)
}

func TestByteRetFn(t *testing.T) {
	res := generated.Test_fn_481_1(3, 2, 12)

	assert.EqualValues(t, 38, res)
}

func TestInt16RetFn(t *testing.T) {
	res := generated.Test_fn_282_2(3, 2, 12)

	assert.EqualValues(t, 38, res)
}

func TestInt32RetFn(t *testing.T) {
	res := generated.Test_fn_444_4(3, 2, 12)

	assert.EqualValues(t, 38, res)
}

func TestIntFloatMixedFn(t *testing.T) {
	res := generated.Test_fn_44F4F8_4(3, 2, 12, 4)

	assert.EqualValues(t, 42, res)
}

func TestDoubleRetFn(t *testing.T) {
	res := generated.Test_fn_F4F4F8_F8(3, 2, 12)

	assert.EqualValues(t, 38, res)
}

func TestManyParamsFn(t *testing.T) {
	res := generated.Test_fn_888888_8(3, 2, 12, 4, 9, 13)

	assert.Equal(t, 99, res)
}

func TestFloatsFn(t *testing.T) {
	input := []float32{1, 2, 3, 4, 5, 6, 7, 8}
	output := make([]float32, 8)

	generated.Test_fn_sq_floats(input, output)

	assert.Equal(t, []float32{1, 4, 9, 16, 25, 36, 49, 64}, output)
}

// The test_calls cases exercise compiler-emitted helpers translated with
// --with-internal-functions: nested C-ABI calls, tail calls, a helper frame
// above 4 KiB, constant pools and a pointer to a caller's C local.

func refWeightedSum(p []int64, n int64) int64 {
	var s int64
	for i := int64(0); i < n; i++ {
		s += p[i] * (i + 1)
	}
	return s
}

func refMixWithLocals(p []int64, n, k int64) int64 {
	var tmp [8]int64
	for i := int64(0); i < 8; i++ {
		v := int64(0)
		if i < n {
			v = p[i]
		}
		tmp[i] = v + k
	}
	return refWeightedSum(tmp[:], 8) - refWeightedSum(p, n)
}

func TestCallsNested(t *testing.T) {
	p := []int64{3, -7, 11, 5, 9}
	for _, k := range []int64{0, 1, 17, -5} {
		var local [8]int64
		var want int64
		for i := int64(0); i < 8; i++ {
			local[i] = i * k
		}
		for i := range p {
			want += refMixWithLocals(p, int64(len(p)), k+int64(i))
			want ^= refMixWithLocals(local[:], 8, int64(i))
		}
		assert.Equal(t, want, generated.Test_calls_nested(p, k), "k=%d", k)
	}
}

func TestCallsTail(t *testing.T) {
	p := []int64{4, 8, 15, 16, 23, 42}
	triple := func(x int64) int64 { return x*3 + 1 }
	assert.Equal(t, int64(-1), generated.Test_calls_tail(nil, 0))
	assert.Equal(t, triple(4), generated.Test_calls_tail(p, 1))
	var sum int64
	for _, v := range p {
		sum += triple(v)
	}
	assert.Equal(t, sum, generated.Test_calls_tail(p, 2))
	// mode 0 returns through the tail call.
	assert.Equal(t, triple(triple(42)), generated.Test_calls_tail(p, 0))
}

func TestCallsBigFrame(t *testing.T) {
	p := []int64{1, 2, 3, 4, 5}
	var want int64 = 1
	for i := int64(0); i < 600; i++ {
		if i < int64(len(p)) {
			want += p[i]
		} else {
			want += i
		}
	}
	assert.Equal(t, want, generated.Test_calls_big_frame(p))
}

func TestCallsConsts(t *testing.T) {
	tbl := []float64{1.2345, 2.3456, 3.4567, 4.5678}
	x, n := 2.5, int64(7)
	want := 1.0
	for i := int64(0); i < n; i++ {
		want += x*tbl[i&3] + 0.987654321
	}
	assert.InDelta(t, want, generated.Test_calls_consts(x, n), 1e-9)
}

func TestCallsLocalPointer(t *testing.T) {
	var want int64
	for i := int64(0); i < 32; i++ {
		want += 7 + i*i
	}
	assert.Equal(t, want, generated.Test_calls_local_pointer(7))
}

// The variant tests pass slices whose capacity is neither their length nor
// the value of any other argument: the C functions take a pointer and a
// length, and the capacity in the frame must not reach them.

func TestVariantString(t *testing.T) {
	data := append(make([]byte, 0, 64), "hello"...)
	want := 7
	for _, c := range data {
		want += int(c)
	}

	assert.Equal(t, want, generated.Test_fn_sum_bytes(data, 7))
	assert.Equal(t, want, generated.Test_fn_sum_string("hello", 7))
}

func TestVariantArgumentLayout(t *testing.T) {
	const first, second, k = "gopher", "go!", 7
	a := append(make([]byte, 0, 32), first...)
	b := append(make([]byte, 0, 16), second...)

	want := int(first[len(first)-1]) + 1000*len(first) +
		1000000*int(second[len(second)-1]) + 1000000000*len(second) + 1000000000000*k

	assert.Equal(t, want, generated.Test_fn_layout_strings(first, second, k))
	assert.Equal(t, want, generated.Test_fn_layout_bytes(a, b, k))
	assert.Equal(t, want, generated.Test_fn_layout_bytes_string(a, second, k))
}

func TestVariantOtherSliceAndFloat(t *testing.T) {
	dst := make([]float32, 3, 9)

	generated.Test_fn_scale_bytes(dst, append(make([]byte, 0, 16), "abc"...), 0.5)
	assert.Equal(t, []float32{48.5, 49, 49.5}, dst)
	generated.Test_fn_scale_string(dst, "xyz", 2)
	assert.Equal(t, []float32{240, 242, 244}, dst)
}

func TestCallsVariantGrowsStack(t *testing.T) {
	s := "stack"
	var want int64 = 1
	for i := range 32768 {
		if i < len(s) {
			want += int64(s[i])
		} else {
			want += int64(uint8(i))
		}
	}

	// A new goroutine starts with a stack smaller than the frame of the
	// function the variant jumps to.
	got := make(chan int64)
	go func() { got <- generated.Test_calls_big_frame_string(s) }()
	assert.Equal(t, want, <-got)

	p := append(make([]byte, 0, 4096), s...)
	go func() { got <- generated.Test_calls_big_frame_bytes(p) }()
	assert.Equal(t, want, <-got)
}

func TestOmitCap(t *testing.T) {
	// capacities that are neither a length nor any other argument
	a := append(make([]int64, 0, 32), 1, 2, 3, 4, 5, 7)
	b := append(make([]int32, 0, 16), 9, 8, 5)

	// b is not listed, so its capacity is an argument; that of a is not
	want := 7 + 1000*len(a) + 1000000*5 + 1000000000*len(b) + 1000000000000*cap(b) + 1000000000000000*2
	assert.Equal(t, want, generated.Test_fn_omit_cap_mixed(a, b, 2))

	x := append(make([]byte, 0, 64), "gopher"...)
	y := append(make([]byte, 0, 48), "go!"...)
	z := append(make([]byte, 0, 40), "gocc"...)
	want = int('r') + 1000*len(x) + 1000000*int('!') + 1000000000*len(y) + 1000000000000*int('c') + 1000000000000000*len(z)
	assert.Equal(t, want, generated.Test_fn_omit_cap_three(x, y, z))
}
