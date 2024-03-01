#include <stdint.h>
#include <stdbool.h>
#include <x86intrin.h>

// gocc: IsASCII(src string) bool
bool is_ascii_sse(const char *src, uint64_t len)
{
  __m128i ma;

  uint64_t i = 0;
  while ((i + 16) <= len)
  {
    ma = _mm_loadu_si128((const __m128i *)(src + i));
    if (_mm_movemask_epi8(ma)) {
      return false;
    }

    i += 16;
  }

  char tail_acc = 0;
  for (; i < len; i++)
  {
    tail_acc |= src[i];
  }
  return (tail_acc & 0x80) ? false : true;
}