#include <stdint.h>

// Fixtures for the way a C frame is fitted into a Go one. Every function
// computes a checksum over the blocks of its input that stack_test.go also
// computes in Go, and each one forces a particular use of the stack. The file
// is translated once per instruction set; the vector types below take the
// widest register of the target, so the same source gives 128-, 256- and
// 512-bit code.

#if defined(__AVX512F__)
#define VW 64
#elif defined(__AVX2__)
#define VW 32
#else
#define VW 16
#endif
#define LANES (VW / 8)

typedef uint64_t vec_t __attribute__((vector_size(VW)));
typedef uint8_t bytes_t __attribute__((vector_size(VW)));
// what a vector load from the input has to use: the data is not aligned
typedef uint64_t unaligned_vec_t __attribute__((vector_size(VW), aligned(1)));

#define LOAD(p) (*(const unaligned_vec_t *)(p))

static inline uint64_t fold(vec_t v) {
    uint64_t r = 0;
    for (int i = 0; i < LANES; i++) r ^= v[i] * (uint64_t)(2 * i + 1);
    return r;
}

// Constants that take a full vector: they do not fit an immediate and no
// lane is a broadcast or an extension of another.
#define K0 0x9e3779b97f4a7c15ull
#define K1 0xbf58476d1ce4e5b9ull
#define K2 0x94d049bb133111ebull
#define K3 0x2545f4914f6cdd1dull
#define K4 0xd6e8feb86659fd93ull
#define K5 0xa0761d6478bd642full
#define K6 0xe7037ed1a0b428dbull
#define K7 0x8ebc6af09c88c6e3ull

#if VW == 64
#define KVEC ((vec_t){K0, K1, K2, K3, K4, K5, K6, K7})
#elif VW == 32
#define KVEC ((vec_t){K0, K1, K2, K3})
#else
#define KVEC ((vec_t){K0, K1})
#endif

// A vector kept in memory: the compiler stores and reloads it with aligned
// moves, which a Go frame cannot give it.
// gocc: Stack_spill(data []byte) uint64
uint64_t stack_spill(const uint8_t *p, uint64_t n, uint64_t cap) {
    volatile vec_t acc = (vec_t){0};
    for (uint64_t i = 0; i + VW <= n; i += VW) {
        acc = acc + LOAD(p + i);
    }
    return fold(acc);
}

// A vector in memory used as an operand. Without VEX the operand has to be
// aligned, and there is no unaligned form of the instruction to use instead.
// gocc: Stack_operand(data []byte) uint64
uint64_t stack_operand(const uint8_t *p, uint64_t n, uint64_t cap) {
    volatile vec_t mask = (vec_t){0};
    vec_t acc = (vec_t){0};
    if (n >= VW) mask = LOAD(p);
    for (uint64_t i = 0; i + VW <= n; i += VW) {
        acc = (acc + LOAD(p + i)) ^ mask;
    }
    return fold(acc);
}

// The same for byte compares and selects, which is where the SSE4 forms
// with an implicit operand come from.
// gocc: Stack_select(data []byte) uint64
uint64_t stack_select(const uint8_t *p, uint64_t n, uint64_t cap) {
    volatile bytes_t limit = (bytes_t){0};
    volatile bytes_t other = (bytes_t)KVEC;
    vec_t acc = (vec_t){0};
    if (n >= VW) limit = (bytes_t)LOAD(p);
    for (uint64_t i = 0; i + VW <= n; i += VW) {
        bytes_t v = (bytes_t)LOAD(p + i);
        bytes_t above = (bytes_t)(v > limit);
        bytes_t picked = (v & above) | (other & ~above);
        acc = acc + (vec_t)picked;
    }
    return fold(acc);
}

// Vector constants, which are read from memory the Go linker lays out.
static const uint64_t table[2 * 8] __attribute__((aligned(64))) = {
    K7, K6, K5, K4, K3, K2, K1, K0, K0, K2, K4, K6, K1, K3, K5, K7,
};

// gocc: Stack_consts(data []byte) uint64
uint64_t stack_consts(const uint8_t *p, uint64_t n, uint64_t cap) {
    vec_t acc = (vec_t){0};
    for (uint64_t i = 0; i + VW <= n; i += VW) {
        // the table is declared aligned, so these are aligned loads
        vec_t row = *(const vec_t *)&table[(i / VW % 2) * 8];
        acc = (acc + LOAD(p + i)) ^ KVEC ^ row;
    }
    return fold(acc);
}

// A helper that is handed a pointer to vectors in its caller's frame. It
// only knows their type, which says they are aligned.
__attribute__((noinline)) static void mix(vec_t *state, const uint8_t *p, uint64_t n) {
    for (uint64_t i = 0; i + VW <= n; i += VW) {
        state[0] = state[0] + LOAD(p + i);
        state[1] = state[1] ^ state[0];
    }
}

// gocc: Stack_pointer(data []byte) uint64
uint64_t stack_pointer(const uint8_t *p, uint64_t n, uint64_t cap) {
    vec_t state[2];
    state[0] = (vec_t){0};
    state[1] = KVEC;
    mix(state, p, n);
    return fold(state[0]) + fold(state[1]);
}

// Vectors in the frame picked by an index that depends on the data, so that
// the compiler has to address them through a register.
// gocc: Stack_walk(data []byte) uint64
uint64_t stack_walk(const uint8_t *p, uint64_t n, uint64_t cap) {
    vec_t ring[8];
    for (int i = 0; i < 8; i++) ring[i] = KVEC * (uint64_t)(i + 1);
    for (uint64_t i = 0; i + VW <= n; i += VW) {
        vec_t *slot = &ring[(p[i] ^ i / VW) & 7];
        *slot = *slot + LOAD(p + i);
    }
    vec_t acc = (vec_t){0};
    for (int i = 0; i < 8; i++) acc = acc ^ ring[i];
    return fold(acc);
}

// Locals the compiler is asked to align more than its registers need, for
// which it realigns its own frame. The helper writes through the pointer,
// which keeps the compiler from passing the vectors in registers instead.
__attribute__((noinline)) static void absorb(vec_t *block, const uint8_t *p) {
    block[0] = block[0] + LOAD(p);
    block[1] = block[1] ^ block[0] ^ LOAD(p + VW);
}

// gocc: Stack_overaligned(data []byte) uint64
uint64_t stack_overaligned(const uint8_t *p, uint64_t n, uint64_t cap) {
    vec_t block[2] __attribute__((aligned(64)));
    block[0] = KVEC;
    block[1] = (vec_t){0};
    for (uint64_t i = 0; i + 2 * VW <= n; i += 2 * VW) {
        absorb(block, p + i);
    }
    return fold(block[0]) ^ fold(block[1]);
}

// A frame much larger than a page, written from one end to the other, with
// more values alive across the calls than there are registers to keep them.
__attribute__((noinline)) static uint64_t scramble(uint64_t a, uint64_t b) {
    volatile uint64_t t = a * K1;
    return (t ^ (t >> 29)) + b;
}

// gocc: Stack_frame(data []byte) uint64
uint64_t stack_frame(const uint8_t *p, uint64_t n, uint64_t cap) {
    volatile uint8_t buf[6000];
    for (int i = 0; i < 6000; i++) buf[i] = (uint8_t)(i * 7);
    for (uint64_t i = 0; i < n; i++) {
        buf[i % 6000] ^= p[i];
        buf[5999 - i % 6000] += p[i];
    }

    uint64_t a = scramble(buf[0], n), b = scramble(buf[5999], a), c = scramble(buf[1], b);
    uint64_t d = scramble(buf[5998], c), e = scramble(buf[2], d), f = scramble(buf[5997], e);
    uint64_t g = scramble(buf[3], f), h = scramble(buf[5996], g), k = scramble(buf[4], h);
    uint64_t sum = a ^ b * 3 ^ c * 5 ^ d * 7 ^ e * 11 ^ f * 13 ^ g * 17 ^ h * 19 ^ k * 23;
    for (int i = 0; i < 6000; i++) sum = sum * 31 + buf[i];
    return sum;
}
