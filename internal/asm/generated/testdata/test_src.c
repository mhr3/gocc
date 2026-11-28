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

// gocc: Test_fn_282_2(a int16, b int, c int16) int16
short test_fn_282_2(short a, long b, short c) {
    return (c*a) + b;
}

// gocc: Test_fn_481_1(a int32, b int, c int8) byte
uint8_t test_fn_481_1(int a, long b, char c) {
    return (c*a) + b;
}

// gocc: Test_fn_444_4(a int32, b int32, c int32) int32
int test_fn_444_4(int a, int b, int c) {
    return (c*a) + b;
}

// gocc: Test_fn_44F4F8_4(a int32, b int32, c float32, d float64) int32
int test_fn_44F4F8_4(int a, int b, float c, double d) {
    return (c*a) + b + d;
}

// gocc: Test_fn_F4F4F8_F8(a, b float32, c float64) float64
double test_fn_F4F4F8_F8(float a, float b, double c) {
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

// Functions with variants: the C code takes a pointer and a length for every
// string and slice, and is entered with either. Each has a parameter after
// the first of them, which is read from the wrong place if the capacity
// word of the frame is not skipped.

// The string form is the main signature here and the []byte one the variant.
// gocc: Test_fn_sum_string(data string, bias int) int
// gocc-variant: Test_fn_sum_bytes(data []byte, bias int) int
long test_fn_sum(const unsigned char *data, long data_len, long bias) {
    long sum = bias;
    for (long i = 0; i < data_len; i++) {
        sum += data[i];
    }
    return sum;
}

// Two strings and an integer, each of the five C arguments at its own weight
// in the result: one that is loaded from the wrong word of the frame, or
// swapped with another, changes it.
// gocc: Test_fn_layout_strings(a, b string, k int) int
// gocc-variant: Test_fn_layout_bytes(a, b []byte, k int) int
// gocc-variant: Test_fn_layout_bytes_string(a []byte, b string, k int) int
long test_fn_layout(const unsigned char *a, long a_len, const unsigned char *b, long b_len, long k) {
    return a[a_len - 1] + 1000 * a_len + 1000000 * b[b_len - 1] + 1000000000 * b_len + 1000000000000 * k;
}

// A slice of another type, a floating-point argument and no result.
// gocc: Test_fn_scale_bytes(dst []float32, src []byte, factor float32)
// gocc-variant: Test_fn_scale_string(dst []float32, src string, factor float32)
void test_fn_scale(float *dst, long dst_len, const unsigned char *src, long src_len, float factor) {
    for (long i = 0; i < src_len && i < dst_len; i++) {
        dst[i] = src[i] * factor;
    }
}

// Slices listed in gocc-omit-cap are a pointer and a length in C; the
// others keep their capacity. As above, each argument has its own weight in
// the result, so one loaded from the wrong word of the frame changes it.
// gocc: Test_fn_omit_cap_mixed(a []int64, b []int32, k int) int
// gocc-omit-cap: a
long test_fn_omit_cap_mixed(const long *a, long a_len, const int *b, long b_len, long b_cap, long k) {
    return a[a_len - 1] + 1000 * a_len + 1000000 * b[b_len - 1] + 1000000000 * b_len +
           1000000000000 * b_cap + 1000000000000000 * k;
}

// Three slices are six arguments without their capacities, which is all the
// integer argument registers of amd64, and nine with them.
// gocc: Test_fn_omit_cap_three(a, b, c []byte) int
// gocc-omit-cap: a, b, c
long test_fn_omit_cap_three(const unsigned char *a, long a_len, const unsigned char *b, long b_len,
                            const unsigned char *c, long c_len) {
    return a[a_len - 1] + 1000 * a_len + 1000000 * b[b_len - 1] + 1000000000 * b_len +
           1000000000000 * c[c_len - 1] + 1000000000000000 * c_len;
}
