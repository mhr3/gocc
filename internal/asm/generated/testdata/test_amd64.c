#include <stdint.h>

// gocc: Test_fn_void_ret(a int32, b int, c int8, res *int)
void test_fn_void_ret(int a, long b, char c, long *res) {
    *res = (c*a) + b;

    return;
}

// gocc: Test_fn_byte_ret(a int32, b int, c int8) byte
uint8_t test_fn_byte_ret(int a, long b, char c) {
    return (c*a) + b;
}

// gocc: Test_fn_6params(a,b,c,d,e,f int) int
int64_t test_fn_6params(long a, long b, long c, long d, long e, long f) {
    return (c*a) + b + (f*d) + e;
}