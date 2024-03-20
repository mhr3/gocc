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