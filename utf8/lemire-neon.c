// Adapted from https://github.com/lemire/fastvalidate-utf-8

#include <stddef.h>
#include <stdint.h>
#include <arm_neon.h>

#define MIN(a, b) ((a) > (b) ? (b) : (a))

/*
 * legal utf-8 byte sequence
 * http://www.unicode.org/versions/Unicode6.0.0/ch03.pdf - page 94
 *
 *  Code Points        1st       2s       3s       4s
 * U+0000..U+007F     00..7F
 * U+0080..U+07FF     C2..DF   80..BF
 * U+0800..U+0FFF     E0       A0..BF   80..BF
 * U+1000..U+CFFF     E1..EC   80..BF   80..BF
 * U+D000..U+D7FF     ED       80..9F   80..BF
 * U+E000..U+FFFF     EE..EF   80..BF   80..BF
 * U+10000..U+3FFFF   F0       90..BF   80..BF   80..BF
 * U+40000..U+FFFFF   F1..F3   80..BF   80..BF   80..BF
 * U+100000..U+10FFFF F4       80..8F   80..BF   80..BF
 *
 */

// all byte values must be no larger than 0xF4
static inline void checkSmallerThan0xF4(int8x16_t current_bytes,
                                        int8x16_t *has_error) {
  // unsigned, saturates to 0 below max
  *has_error = vorrq_s8(*has_error,
          vreinterpretq_s8_u8(vqsubq_u8(vreinterpretq_u8_s8(current_bytes), vdupq_n_u8(0xF4))));
}

static const int8_t _nibbles[] = {
  1, 1, 1, 1, 1, 1, 1, 1, // 0xxx (ASCII)
  0, 0, 0, 0,             // 10xx (continuation)
  2, 2,                   // 110x
  3,                      // 1110
  4, // 1111, next should be 0 (not checked here)
};

static inline int8x16_t continuationLengths(int8x16_t high_nibbles) {
  return vqtbl1q_s8(vld1q_s8(_nibbles), vreinterpretq_u8_s8(high_nibbles));
}

static inline int8x16_t carryContinuations(int8x16_t initial_lengths,
                                         int8x16_t previous_carries) {

  int8x16_t right1 =
     vreinterpretq_s8_u8(vqsubq_u8(vreinterpretq_u8_s8(vextq_s8(previous_carries, initial_lengths, 16 - 1)),
                    vdupq_n_u8(1)));
  int8x16_t sum = vaddq_s8(initial_lengths, right1);

  int8x16_t right2 = vreinterpretq_s8_u8(vqsubq_u8(vreinterpretq_u8_s8(vextq_s8(previous_carries, sum, 16 - 2)),
                                 vdupq_n_u8(2)));
  return vaddq_s8(sum, right2);
}

static inline void checkContinuations(int8x16_t initial_lengths, int8x16_t carries,
                                      int8x16_t *has_error) {

  // overlap || underlap
  // carry > length && length > 0 || !(carry > length) && !(length > 0)
  // (carries > length) == (lengths > 0)
  uint8x16_t overunder =
      vceqq_u8(vcgtq_s8(carries, initial_lengths),
                     vcgtq_s8(initial_lengths, vdupq_n_s8(0)));

  *has_error = vorrq_s8(*has_error, vreinterpretq_s8_u8(overunder));
}

// when 0xED is found, next byte must be no larger than 0x9F
// when 0xF4 is found, next byte must be no larger than 0x8F
// next byte must be continuation, ie sign bit is set, so signed < is ok
static inline void checkFirstContinuationMax(int8x16_t current_bytes,
                                             int8x16_t off1_current_bytes,
                                             int8x16_t *has_error) {
  uint8x16_t maskED = vceqq_s8(off1_current_bytes, vdupq_n_s8(0xED));
  uint8x16_t maskF4 = vceqq_s8(off1_current_bytes, vdupq_n_s8(0xF4));

  uint8x16_t badfollowED =
      vandq_u8(vcgtq_s8(current_bytes, vdupq_n_s8(0x9F)), maskED);
  uint8x16_t badfollowF4 =
      vandq_u8(vcgtq_s8(current_bytes, vdupq_n_s8(0x8F)), maskF4);

  *has_error = vorrq_s8(*has_error, vreinterpretq_s8_u8(vorrq_u8(badfollowED, badfollowF4)));
}

static const int8_t _initial_mins[] = {
  -128, -128, -128, -128, -128, -128, -128, -128, -128, -128,
  -128, -128, // 10xx => false
  0xC2, -128, // 110x
  0xE1,       // 1110
  0xF1,
};

static const int8_t _second_mins[] = {
  -128, -128, -128, -128, -128, -128, -128, -128, -128, -128,
  -128, -128, // 10xx => false
  127, 127,   // 110x => true
  0xA0,       // 1110
  0x90,
};

// map off1_hibits => error condition
// hibits     off1    cur
// C       => < C2 && true
// E       => < E1 && < A0
// F       => < F1 && < 90
// else      false && false
static inline void checkOverlong(int8x16_t current_bytes,
                                 int8x16_t off1_current_bytes, int8x16_t hibits,
                                 int8x16_t previous_hibits, int8x16_t *has_error) {
  int8x16_t off1_hibits = vextq_s8(previous_hibits, hibits, 16 - 1);
  int8x16_t initial_mins = vqtbl1q_s8(vld1q_s8(_initial_mins), vreinterpretq_u8_s8(off1_hibits));

  uint8x16_t initial_under = vcgtq_s8(initial_mins, off1_current_bytes);

  int8x16_t second_mins = vqtbl1q_s8(vld1q_s8(_second_mins), vreinterpretq_u8_s8(off1_hibits));
  uint8x16_t second_under = vcgtq_s8(second_mins, current_bytes);
  *has_error =
     vorrq_s8(*has_error, vreinterpretq_s8_u8(vandq_u8(initial_under, second_under)));
}

struct processed_utf_bytes {
  int8x16_t rawbytes;
  int8x16_t high_nibbles;
  int8x16_t carried_continuations;
};

static inline void count_nibbles(int8x16_t bytes,
                                 struct processed_utf_bytes *answer) {
  answer->rawbytes = bytes;
  answer->high_nibbles =
    vreinterpretq_s8_u8(vshrq_n_u8(vreinterpretq_u8_s8(bytes), 4));
}

// check whether the current bytes are valid UTF-8
// at the end of the function, previous gets updated
static inline struct processed_utf_bytes
checkUTF8Bytes(int8x16_t current_bytes, struct processed_utf_bytes *previous,
               int8x16_t *has_error) {
  struct processed_utf_bytes pb;
  count_nibbles(current_bytes, &pb);

  checkSmallerThan0xF4(current_bytes, has_error);

  int8x16_t initial_lengths = continuationLengths(pb.high_nibbles);

  pb.carried_continuations =
      carryContinuations(initial_lengths, previous->carried_continuations);

  checkContinuations(initial_lengths, pb.carried_continuations, has_error);

  int8x16_t off1_current_bytes =
    vextq_s8(previous->rawbytes, pb.rawbytes, 16 - 1);
  checkFirstContinuationMax(current_bytes, off1_current_bytes, has_error);

  checkOverlong(current_bytes, off1_current_bytes, pb.high_nibbles,
                previous->high_nibbles, has_error);
  return pb;
}

static const int8_t _verror[] = {9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 1};

static inline uint64_t load_data(const unsigned char *src, int64_t len) {
  switch (len) {
    case 8:
      return *(uint64_t *)src;
    case 7:
      return *(uint32_t *)src | ((uint64_t)(*(uint16_t *)(src+4)) << 32) | ((uint64_t)src[6] << 48);
    case 6:
      return *(uint32_t *)src | ((uint64_t)(*(uint16_t *)(src+4)) << 32);
    case 5:
      return *(uint32_t *)src | ((uint64_t)src[4] << 32);
    case 4:
      return *(uint32_t *)src;
    case 3:
      return *(uint16_t *)src | ((uint64_t)src[2] << 16);
    case 2:
      return *(uint16_t *)src;
    case 1:
      return *src;
  }

  return 0;
}

// gocc: valid_lemire(src string) int
/* Return 0 on success, -1 on error */
int64_t utf8_lemire(const unsigned char *src, int64_t len) {
  size_t i = 0;
  int8x16_t has_error = vdupq_n_s8(0);
  struct processed_utf_bytes previous = {.rawbytes = vdupq_n_s8(0),
                                         .high_nibbles = vdupq_n_s8(0),
                                         .carried_continuations =
                                             vdupq_n_s8(0)};
  if (len >= 16) {
    for (; i <= len - 16; i += 16) {
      int8x16_t current_bytes = vld1q_s8((int8_t*)(src + i));
      previous = checkUTF8Bytes(current_bytes, &previous, &has_error);
    }
  }

  // last part
  if (i < len) {
#ifdef USE_MEMCPY
    unsigned long long buffer[2] = {0, 0};
    memcpy(buffer, src + i, len - i);
#else
    unsigned long long buffer[2];
    buffer[0] = load_data(src + i, MIN(8, len - i));
    buffer[1] = load_data(src + i + 8, len - i - 8);
#endif
    int8x16_t current_bytes = vld1q_s8((int8_t *)buffer);
    previous = checkUTF8Bytes(current_bytes, &previous, &has_error);
  } else {
    has_error =
        vorrq_s8(vreinterpretq_s8_u8(vcgtq_s8(previous.carried_continuations,
                                    vld1q_s8(_verror))),
                     has_error);
  }

  return vmaxvq_u8(vreinterpretq_u8_s8(has_error)) == 0 ? 0 : -1;
}
