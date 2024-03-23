#include <stdint.h>

// gocc: Test_fn_4818_0(a int32, b int, c int8, res *int)
void test_fn_fn_4818_0(int a, long b, char c, long *res) {
    *res = (c*a) + b;

    return;
}

// gocc: Test_fn_111_0(a, b, c byte)
void test_fn_111_0(uint8_t a, uint8_t b, uint8_t c) {
    (c*a) + b;
}

// gocc: Test_fn_111_1(a, b, c byte) byte
uint8_t test_fn_111_1(uint8_t a, uint8_t b, uint8_t c) {
    return (c*a) + b;
}

// gocc: Test_fn_1114_1(a, b, c byte, d int32) byte
uint8_t test_fn_1114_1(uint8_t a, uint8_t b, uint8_t c, int d) {
    return (c*a) + b + d;
}

// gocc: Test_fn_481_1(a int32, b int, c int8) byte
uint8_t test_fn_481_1(int a, long b, char c) {
    return (c*a) + b;
}

// gocc: Test_fn_444_4(a int32, b int32, c int32) int32
int test_fn_444_4(int a, int b, int c) {
    return (c*a) + b;
}

// gocc: Test_fn_888888_8(a,b,c,d,e,f int) int
long test_fn_888888_8(long a, long b, long c, long d, long e, long f) {
    return (c*a) + b + (f*d) + e;
}

// gocc: Test_fn_sq_floats(input []float32, output []float32)
void test_fn_sq_floats(float *input, long input_len, long input_cap, float *output, long output_len, long output_cap) {
    for (int i = 0; i < input_len; i++) {
        output[i] = input[i] * input[i];
    }
}