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

// The aliases enter the same code with strings in place of byte slices; the
// capacity word of such an argument then holds its length.
// gocc: Test_fn_sum_bytes(data []byte, bias int) int
// gocc-alias: Test_fn_sum_string(data string, bias int) int
long test_fn_sum_bytes(unsigned char *data, long data_len, long data_cap, long bias) {
    long sum = bias + data_cap * 1000003;
    for (long i = 0; i < data_len; i++) {
        sum += data[i];
    }
    return sum;
}

// gocc: Test_fn_mismatch_bytes(a, b []byte) int
// gocc-alias: Test_fn_mismatch_strings(a, b string) int
// gocc-alias: Test_fn_mismatch_bytes_string(a []byte, b string) int
long test_fn_mismatch_bytes(unsigned char *a, long a_len, long a_cap, unsigned char *b, long b_len, long b_cap) {
    long n = a_len < b_len ? a_len : b_len;
    long i = 0;
    while (i < n && a[i] == b[i]) {
        i++;
    }
    return i + a_cap * 1000 + b_cap * 1000000;
}

// gocc: Test_fn_shift_bytes(dst []byte, src []byte)
// gocc-alias: Test_fn_shift_string(dst []byte, src string)
void test_fn_shift_bytes(unsigned char *dst, long dst_len, long dst_cap, unsigned char *src, long src_len, long src_cap) {
    for (long i = 0; i < src_len && i < dst_len; i++) {
        dst[i] = src[i] + (unsigned char)src_cap;
    }
}
