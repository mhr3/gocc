package generated_test

import (
	"testing"

	"github.com/kelindar/gocc/internal/asm/generated"
	"github.com/stretchr/testify/assert"
)

func TestVoidRetFn(t *testing.T) {
	var res int
	generated.Test_fn_4818_0(1, 2, 3, &res)

	assert.Equal(t, 5, res)
}

func TestByteRetFn(t *testing.T) {
	res := generated.Test_fn_481_1(3, 2, 12)

	assert.EqualValues(t, 38, res)
}

func TestInt32RetFn(t *testing.T) {
	res := generated.Test_fn_444_4(3, 2, 12)

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
