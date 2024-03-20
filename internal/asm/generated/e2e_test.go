package generated_test

import (
	"testing"

	"github.com/kelindar/gocc/internal/asm/generated"
	"github.com/stretchr/testify/assert"
)

func TestVoidRetFn(t *testing.T) {
	var res int
	generated.Test_fn_void_ret(1, 2, 3, &res)

	assert.Equal(t, 5, res)
}

func TestByteRetFn(t *testing.T) {
	res := generated.Test_fn_byte_ret(3, 2, 12)

	assert.EqualValues(t, 38, res)
}
