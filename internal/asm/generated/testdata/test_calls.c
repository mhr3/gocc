#include <stdint.h>

// Compiler-emitted helpers: these are only reachable through the annotated
// functions below and are translated with --with-internal-functions.

__attribute__((noinline)) static int64_t weighted_sum(const int64_t *p, int64_t n) {
    int64_t s = 0;
    for (int64_t i = 0; i < n; i++) s += p[i] * (i + 1);
    return s;
}

__attribute__((noinline)) static int64_t mix_with_locals(const int64_t *p, int64_t n, int64_t k) {
    int64_t tmp[8];
    for (int i = 0; i < 8; i++) tmp[i] = (i < n ? p[i] : 0) + k;
    return weighted_sum(tmp, 8) - weighted_sum(p, n);
}

// gocc: Test_calls_nested(p []int64, k int64) int64
int64_t test_calls_nested(const int64_t *p, uint64_t n, uint64_t cap, int64_t k) {
    int64_t local[8];
    int64_t acc = 0;
    for (uint64_t i = 0; i < 8; i++) local[i] = (int64_t)i * k;
    for (uint64_t i = 0; i < n; i++) {
        acc += mix_with_locals(p, (int64_t)n, k + (int64_t)i);
        acc ^= mix_with_locals(local, 8, (int64_t)i);
    }
    return acc;
}

__attribute__((noinline)) static int64_t triple_plus_one(int64_t x) {
    return x * 3 + 1;
}

// Returns through a tail call on every path except the early return.
// gocc: Test_calls_tail(p []int64, mode int64) int64
int64_t test_calls_tail(const int64_t *p, uint64_t n, uint64_t cap, int64_t mode) {
    int64_t buf[16];
    if (n == 0) return -1;
    for (uint64_t i = 0; i < 16 && i < n; i++) buf[i] = triple_plus_one(p[i]);
    if (mode == 1) return buf[0];
    if (mode == 2) {
        int64_t s = 0;
        for (int i = 0; i < 16 && (uint64_t)i < n; i++) s += buf[i];
        return s;
    }
    return triple_plus_one(buf[n > 16 ? 15 : n - 1]);
}

__attribute__((noinline)) static int64_t big_frame_helper(const int64_t *p, int64_t n) {
    // 4800 bytes of locals: ARM64 allocates this with a shifted immediate.
    volatile int64_t buf[600];
    for (int64_t i = 0; i < 600; i++) buf[i] = i < n ? p[i] : i;
    int64_t s = 0;
    for (int64_t i = 0; i < 600; i++) s += buf[i];
    return s;
}

// gocc: Test_calls_big_frame(p []int64) int64
int64_t test_calls_big_frame(const int64_t *p, uint64_t n, uint64_t cap) {
    return big_frame_helper(p, (int64_t)n) + 1;
}

__attribute__((noinline)) static double scale_table(double x, int64_t n) {
    static const double tbl[4] = {1.2345, 2.3456, 3.4567, 4.5678};
    double s = 0.0;
    for (int64_t i = 0; i < n; i++) s += x * tbl[i & 3] + 0.987654321;
    return s;
}

// gocc: Test_calls_consts(x float64, n int64) float64
double test_calls_consts(double x, int64_t n) {
    return scale_table(x, n) + 1.0;
}

__attribute__((noinline)) static void fill_squares(int64_t *out, int64_t n, int64_t seed) {
    for (int64_t i = 0; i < n; i++) out[i] = seed + i * i;
}

// A pointer to the caller's C local is handed to the helper; the summation
// loop is vectorized and reads the local back through SSE/NEON loads.
// gocc: Test_calls_local_pointer(seed int64) int64
int64_t test_calls_local_pointer(int64_t seed) {
    int64_t buf[32];
    fill_squares(buf, 32, seed);
    int64_t s = 0;
    for (int i = 0; i < 32; i++) s += buf[i];
    return s;
}
