#include <stdint.h>
#include <stdbool.h>
#include <x86intrin.h>

// The function returns true (1) if all chars passed in src are
// 7-bit values (0x00..0x7F). Otherwise, it returns false (0).
// gocc: IsASCII(src string) bool
bool is_ascii_avx(const char *src, uint64_t src_len)
{
  uint64_t i = 0;
  __m256i has_error = _mm256_setzero_si256();
  if (src_len >= 32)
  {
    for (; i <= src_len - 32; i += 32)
    {
      __m256i current_bytes = _mm256_loadu_si256((const __m256i *)(src + i));
      has_error = _mm256_or_si256(has_error, current_bytes);
    }
  }
  int error_mask = _mm256_movemask_epi8(has_error);

  char tail_has_error = 0;
  for (; i < src_len; i++)
  {
    tail_has_error |= src[i];
  }
  error_mask |= (tail_has_error & 0x80);

  return !error_mask;
}
