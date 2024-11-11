#include <stdint.h>
#include <stdbool.h>
#include <arm_neon.h>

// gocc: IsASCII(data string) bool
bool is_ascii(unsigned char *data, uint64_t length)
{
    const uint64_t blockSize = 16; // NEON can process 128 bits (16 bytes) at a time
    const uint64_t ld4BlockSize = blockSize * 4;

    if (length >= blockSize)
    {
        const uint8x16_t msb_mask = vdupq_n_u8(0x80); // Create a mask for the MSB

#ifndef SKIP_LD1x4
        for (const unsigned char *data64_end = (data + length) - (length % ld4BlockSize); data < data64_end; data += ld4BlockSize)
        {
            uint8x16x4_t blocks = vld1q_u8_x4(data);           // Load 64 bytes of data
            blocks.val[0] = vtstq_u8(blocks.val[0], msb_mask); // AND with the mask to isolate MSB
            blocks.val[1] = vtstq_u8(blocks.val[1], msb_mask); // AND with the mask to isolate MSB
            blocks.val[2] = vtstq_u8(blocks.val[2], msb_mask); // AND with the mask to isolate MSB
            blocks.val[3] = vtstq_u8(blocks.val[3], msb_mask); // AND with the mask to isolate MSB

            uint8x16_t result = vorrq_u8(vorrq_u8(blocks.val[0], blocks.val[1]), vorrq_u8(blocks.val[2], blocks.val[3])); // OR the results
            // SHRN can be faster than UMAXV
            if (vget_lane_u64(vshrn_n_u16(result, 4), 0) > 0)
            {
                return false;
            }
        }
        length %= ld4BlockSize;
#endif
        for (const unsigned char *data16_end = (data + length) - (length % blockSize); data < data16_end; data += blockSize)
        {
            uint8x16_t block = vld1q_u8(data);             // Load 16 bytes of data
            uint8x16_t result = vtstq_u8(block, msb_mask); // AND with the mask to isolate MSB
            // SHRN can be faster than UMAXV
            if (vget_lane_u64(vshrn_n_u16(result, 4), 0) > 0)
            {
                return false;
            }
        }
        length %= blockSize;
    }

    if (length & 8)
    {
        uint64_t lo64, hi64;
        uint8_t *data_end = data + length;
        __builtin_memcpy(&lo64, data, sizeof(lo64));
        __builtin_memcpy(&hi64, data_end - 8, sizeof(hi64));

        uint64_t data64 = lo64 | hi64;
        return (data64 & 0x8080808080808080ull) ? false : true;
    }

    if (length & 4)
    {
        uint32_t lo32, hi32;
        uint8_t *data_end = data + length;
        __builtin_memcpy(&lo32, data, sizeof(lo32));
        __builtin_memcpy(&hi32, data_end - 4, sizeof(hi32));

        uint32_t data32 = lo32 | hi32;
        return (data32 & 0x80808080) ? false : true;
    }

    uint32_t data32 = 0;

    if (length != 0)
    {
        // branchless check for 1-3 bytes
        uint8_t *data_end = data + length;
        int idx = length >> 1;
        data32 |= data[0];
        data32 |= data[idx];
        data32 |= data_end[-1];
    }

    return (data32 & 0x80808080) ? false : true;
}

// gocc: IndexBit(data string, mask byte) int
int64_t index_bit(unsigned char *data, uint64_t length, uint8_t mask_bit)
{
    const uint64_t blockSize = 16; // NEON can process 128 bits (16 bytes) at a time
    const uint64_t ld4BlockSize = blockSize * 4;

    const unsigned char *data_start = data;
    
    if (length >= blockSize)
    {
        uint8x16_t mask = vdupq_n_u8(mask_bit); // Create a vector mask
#ifndef SKIP_LD1x4
        for (const unsigned char *data64_end = (data + length) - (length % ld4BlockSize); data < data64_end; data += ld4BlockSize)
        {
            uint8x16x4_t blocks = vld1q_u8_x4(data);       // Load 64 bytes of data
            blocks.val[0] = vtstq_u8(blocks.val[0], mask); // AND with the mask
            blocks.val[1] = vtstq_u8(blocks.val[1], mask); // AND with the mask
            blocks.val[2] = vtstq_u8(blocks.val[2], mask); // AND with the mask
            blocks.val[3] = vtstq_u8(blocks.val[3], mask); // AND with the mask

            uint8x16_t result = vorrq_u8(vorrq_u8(blocks.val[0], blocks.val[1]), vorrq_u8(blocks.val[2], blocks.val[3])); // OR the results
            // SHRN can be faster than UMAXV
            if (vget_lane_u64(vshrn_n_u16(result, 4), 0) > 0)
            {
                // now a few operations to find the index of the first set bit
                for (int j = 0; j < 4; j++)
                {
                    uint64_t data64 = vget_lane_u64(vshrn_n_u16(blocks.val[j], 4), 0);
                    if (data64 != 0)
                    {
                        int64_t offset = j*16 + __builtin_ctzll(data64) / 4;
                        return (data - data_start) + offset;
                    }
                }
            }
        }
        length %= ld4BlockSize;
#endif
        for (const unsigned char *data16_end = (data + length) - (length % blockSize); data < data16_end; data += blockSize)
        {
            uint8x16_t block = vld1q_u8(data);         // Load 16 bytes of data
            uint8x16_t result = vtstq_u8(block, mask); // AND with the mask

            // SHRN can be faster than UMAXV
            uint64_t data64 = vget_lane_u64(vshrn_n_u16(result, 4), 0);
            if (data64 > 0)
            {
                int offset = __builtin_ctzll(data64) / 4;
                return (data - data_start) + offset;
            }
        }
        length %= blockSize;
    }

    uint32_t mask32 = mask_bit;
    mask32 |= mask32 << 8;
    mask32 |= mask32 << 16;

    if (length >= 8)
    {
        uint64_t mask64 = mask32;
        mask64 |= mask64 << 32;
        
        uint64_t data64 = *(uint64_t *)(data);
        data64 &= mask64;
        if (data64 != 0)
        {
            return (data - data_start) + __builtin_ctzll(data64) / 8;
        }
        data += 8;
        length -= 8;
    }

    uint32_t data32;

    if (length >= 4)
    {
        data32 = *(uint32_t *)(data);
        data32 &= mask32;
        if (data32 != 0)
        {
            return (data - data_start) + __builtin_ctz(data32) / 8;
        }
        data += 4;
        length -= 4;
    }

    // Handle the remaining bytes (if any)
    switch (length)
    {
    case 3:
        data32 = *(uint16_t *)(data);
        data32 |= data[2] << 16;
        break;
    case 2:
        data32 = *(uint16_t *)(data);
        break;
    case 1:
        data32 = (uint32_t)*data;
        break;
    default:
        data32 = 0;
        break;
    }

    data32 &= mask32;
    if (data32)
    {
        return (data - data_start) + __builtin_ctz(data32) / 8;
    }

    return -1;
}

static uint8_t uppercasingTable[32] = {
    0,32,32,32,32,32,32,32,32,32,32,32,32,32,32,32,
    32,32,32,32,32,32,32,32,32,32,32,0, 0, 0, 0, 0,
};

static inline bool equal_fold_core(unsigned char *a, unsigned char *b, uint64_t length,
    const uint8x16x2_t table, const uint8x16_t shift)
{
    const uint64_t blockSize = 16; // NEON can process 128 bits (16 bytes) at a time

    for (const unsigned char *data_bound = (a + length) - (length % blockSize); a < data_bound; a += blockSize, b += blockSize)
    {
        uint8x16_t a_data = vld1q_u8(a); // Load 16 bytes of data
        uint8x16_t b_data = vld1q_u8(b); // Load 16 bytes of data

        a_data = vsubq_u8(a_data, shift);
        a_data = vsubq_u8(a_data, vqtbl2q_u8(table, a_data));

        b_data = vsubq_u8(b_data, shift);
        b_data = vsubq_u8(b_data, vqtbl2q_u8(table, b_data));

        // we should shift the data back, but we just need to compare, so we can skip that
        const uint8x16_t result = vceqq_u8(a_data, b_data);

        // SHRN can be faster than UMAXV
        if (vget_lane_u64(vshrn_n_u16(result, 4), 0) != ~0ull)
        {
            return false;
        }
    }
    length %= blockSize;

    // same as above, but with just half the vector register
    if (length >= 8)
    {
        uint8x8_t a_data = vld1_u8(a);
        uint8x8_t b_data = vld1_u8(b);

        a_data = vsub_u8(a_data, vget_low_u8(shift));
        a_data = vsub_u8(a_data, vqtbl2_u8(table, a_data));

        b_data = vsub_u8(b_data, vget_low_u8(shift));
        b_data = vsub_u8(b_data, vqtbl2_u8(table, b_data));

        // we should shift the data back, but we just need to compare, so we can skip that
        const uint8x8_t result = vceq_u8(a_data, b_data);

        // Check if there's any 0 bytes
        if (vget_lane_u64(result, 0) != ~0ull)
        {
            return false;
        }
        a += 8;
        b += 8;
        length %= 8;
    }

    if (length == 0) {
        return true;
    }

    uint64_t a_data64 = 0;
    uint64_t b_data64 = 0;

    if (length >= 4)
    {
        a_data64 = *(uint32_t *)(a);
        a += 4;
        b_data64 = *(uint32_t *)(b);
        b += 4;
        length -= 4;
    }

    // FIXME: this is reordering the bytes, though does it for both a and b, so it's fine
    switch (length)
    {
    case 3:
        a_data64 <<= 24;
        a_data64 |= *(uint16_t *)(a) << 8;
        a_data64 |= a[2];
        // same for b
        b_data64 <<= 24;
        b_data64 |= *(uint16_t *)(b) << 8;
        b_data64 |= b[2];
        break;
    case 2:
        a_data64 <<= 16;
        a_data64 |= *(uint16_t *)(a);
        // same for b
        b_data64 <<= 16;
        b_data64 |= *(uint16_t *)(b);
        break;
    case 1:
        a_data64 <<= 8;
        a_data64 |= *a;
        // same for b
        b_data64 <<= 8;
        b_data64 |= *b;
        break;
    }

    uint8x8_t a_data = vcreate_u8(a_data64);
    uint8x8_t b_data = vcreate_u8(b_data64);

    a_data = vsub_u8(a_data, vget_low_u8(shift));
    a_data = vsub_u8(a_data, vqtbl2_u8(table, a_data));

    b_data = vsub_u8(b_data, vget_low_u8(shift));
    b_data = vsub_u8(b_data, vqtbl2_u8(table, b_data));

    // we should shift the data back, but we just need to compare, so we can skip that
    const uint8x8_t result = vceq_u8(a_data, b_data);

    // Check if there's any 0 bytes
    if (vget_lane_u64(result, 0) != ~0ull)
    {
        return false;
    }

    return true;
}

// gocc: EqualFold(a, b string) bool
bool equal_fold(unsigned char *a, uint64_t a_len, unsigned char *b, uint64_t b_len)
{
    if (a_len != b_len)
    {
        return false;
    }

    const uint8x16x2_t table = vld1q_u8_x2(uppercasingTable);
    const uint8x16_t shift = vdupq_n_u8(0x60);

    return equal_fold_core(a, b, a_len, table, shift);
}

// loads up to 16 bytes of data into a 128-bit register
static inline uint8x16_t load_data16(const unsigned char *src, int64_t len) {
    if (len >= 16) {
        return vld1q_u8(src);
    } else if (len == 8) {
        return vcombine_u64(vld1_u64((uint64_t *)src), vcreate_u64(0));
    } else if (len <= 0) {
        return vdupq_n_u8(0);
    }

    const uint64_t orig_len = len;
    uint64_t data64 = 0;
    uint64_t data64lo;

    if (len & 8) {
        data64lo = *(uint64_t *)(src);
        src += 8;
        len -= 8;
    }

    if (len & 4) {
        data64 = *(uint32_t *)(src);
        src += 4;
        len -= 4;
    }

    uint64_t tmp;
    switch (len) {
    case 3:
        tmp = *(uint16_t *)(src);
        int shift = 8 * (orig_len & 4);
        data64 |= tmp << shift;
        tmp = src[2];
        data64 |= tmp << (16 + shift);
        break;
    case 2:
        tmp = *(uint16_t *)(src);
        data64 |= tmp << (8 * (orig_len & 4));
        break;
    case 1:
        tmp = *src;
        data64 |= tmp << (8 * (orig_len & 4));
        break;
    }

    if (orig_len < 8) {
        return vcombine_u64(vcreate_u64(data64), vcreate_u64(0));
    }
    return vcombine_u64(vcreate_u64(data64lo), vcreate_u64(data64));
}

// loads up to 16 bytes of data into a 128-bit register
// requires stack space for 2 64-bit integers, gocc has trouble with that
static inline uint8x16_t load_data16_v2(const unsigned char *src, int64_t len) {
    if (len >= 16) {
        return vld1q_u8(src);
    } else if (len <= 0) {
        return vdupq_n_u8(0);
    }

    const uint64_t orig_len = len;
    const uint8_t *src_end = src + len;
    uint64_t buf[2] = {0};

    uint8_t *dst = (uint8_t *)&buf[0];
    uint8_t *dst_end = dst + len;

    if (len & 8) {
        // Copy 8-15 bytes when the 4th bit of length is set (length >= 8)
        uint64_t lo64, hi64;
        __builtin_memcpy(&lo64, src, sizeof(lo64));
        __builtin_memcpy(&hi64, src_end - 8, sizeof(hi64));
        __builtin_memcpy(dst, &lo64, sizeof(lo64));
        __builtin_memcpy(dst_end - 8, &hi64, sizeof(hi64));

        return vld1q_u8(dst);
    }

    if (len & 4) {
        // Copy 4-7 bytes when the 3rd bit of length is set (length >= 4)
        uint32_t lo32, hi32;
        __builtin_memcpy(&lo32, src, sizeof(lo32));
        __builtin_memcpy(&hi32, src_end - 4, sizeof(hi32));
        __builtin_memcpy(dst, &lo32, sizeof(lo32));
        __builtin_memcpy(dst_end - 4, &hi32, sizeof(hi32));

        return vld1q_u8(dst);
    }

    // Copy 1-3 bytes
    int idx = len >> 1;
    dst[0] = src[0];
    dst[idx] = src[idx];
    dst_end[-1] = src_end[-1];

    return vld1q_u8(dst);
}

static inline int64_t index_fold_1_byte_needle(unsigned char *haystack, uint64_t haystack_len,
    uint8_t needle, const uint8x16x2_t table)
{
    const uint64_t blockSize = 16; // NEON can process 128 bits (16 bytes) at a time
    const uint8x16_t shift = vdupq_n_u8(0x60);

    if (needle >= 'a' && needle <= 'z') needle -= 32;
    // the needle is uppercased and shifted
    const uint8x16_t searched = vdupq_n_u8(needle-0x60);

    uint64_t curr_pos = 0;

    for (const unsigned char *data_bound = haystack + haystack_len; haystack < data_bound; haystack += blockSize, curr_pos += blockSize)
    {
        uint8x16_t data = load_data16(haystack, haystack_len - curr_pos);

        data = vsubq_u8(data, shift);
        data = vsubq_u8(data, vqtbl2q_u8(table, data));

        // operating on shifted data
        const uint8x16_t res = vceqq_u8(data, searched);
        const uint8x8_t narrowed = vshrn_n_u16(res, 4);

        uint64_t data64 = vget_lane_u64(narrowed, 0);
        if (data64)
        {
            const int pos = (__builtin_ctzll(data64) / 4);
            if (haystack+pos >= data_bound) return -1;
            return curr_pos + pos;
        }
    }

    return -1;
}

static inline uint16x8_t index_fold_prepare_comparer(const uint16_t* needle, const uint8x16_t shift, const uint8x16x2_t table)
{
    uint8x16_t needle8vec = vreinterpretq_u8_u16(vld1q_dup_u16(needle));
    needle8vec = vsubq_u8(needle8vec, shift);
    needle8vec = vsubq_u8(needle8vec, vqtbl2q_u8(table, needle8vec));
    return vreinterpretq_u16_u8(needle8vec);
}

static inline int64_t index_fold_2_byte_needle(unsigned char *haystack, uint64_t haystack_len,
    const uint16_t* needle, const uint8x16x2_t table)
{
    const uint64_t blockSize = 16; // NEON can process 128 bits (16 bytes) at a time
    const uint64_t checked_len = haystack_len - 2;
    const uint8x16_t shift = vdupq_n_u8(0x60);

    const uint16x8_t searched = index_fold_prepare_comparer(needle, shift, table);

    uint8x16_t prev_data = vdupq_n_u8(0);
    uint64_t curr_pos = 0;

    for (const unsigned char *data_bound = haystack + checked_len + 1; haystack <= data_bound; haystack += blockSize, curr_pos += blockSize)
    {
        uint8x16_t data = load_data16(haystack, haystack_len - curr_pos);

        data = vsubq_u8(data, shift);
        data = vsubq_u8(data, vqtbl2q_u8(table, data));

        // operating on shifted data
        const uint16x8_t res1 = vceqq_u16(data, searched);
        const uint16x8_t prev = vextq_u8(prev_data, data, 15);
        const uint16x8_t res2 = vceqq_u16(prev, searched);
        prev_data = data;

        const uint16x8_t combined = vorrq_u16(vshlq_n_u16(res1, 8), vshrq_n_u16(res2, 8));
        const uint8x8_t narrowed = vshrn_n_u16(combined, 4);

        // these represent positions: [-1, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14]
        uint64_t data64 = vget_lane_u64(narrowed, 0);
        // clear the 0th byte on the first iteration, we made up a 0 on that position
        if (data64 && curr_pos == 0) data64 &= ~(0xF);
        if (data64)
        {
            const int pos = (__builtin_ctzll(data64) / 4) - 1;
            if (haystack+pos >= data_bound) return -1;
            return curr_pos + pos;
        }
    }

    return -1;
}

static inline uint64_t index_fold_process_block(uint8x16_t data, uint8x16_t data_end, const uint16x8_t first2, const uint16x8_t last2,
    const uint8x16x2_t table, const uint8x16_t shift, uint8x16_t* prev_data, uint8x16_t* prev_data_end)
{
    data = vsubq_u8(data, shift);
    data = vsubq_u8(data, vqtbl2q_u8(table, data));

    data_end = vsubq_u8(data_end, shift);
    data_end = vsubq_u8(data_end, vqtbl2q_u8(table, data_end));

    // operating on shifted data
    const uint16x8_t res1 = vandq_u16(vceqq_u16(data, first2), vceqq_u16(data_end, last2));
    const uint16x8_t prev = vextq_u8(*prev_data, data, 15);
    const uint16x8_t prev_end = vextq_u8(*prev_data_end, data_end, 15);
    const uint16x8_t res2 = vandq_u16(vceqq_u16(prev, first2), vceqq_u16(prev_end, last2));
    *prev_data = data;
    *prev_data_end = data_end;

    const uint16x8_t combined = vorrq_u16(vshlq_n_u16(res1, 8), vshrq_n_u16(res2, 8));
    const uint8x8_t narrowed = vshrn_n_u16(combined, 4);

    // these represent positions: [-1, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14]
    return vget_lane_u64(narrowed, 0);
}

// gocc: IndexFold(a, b string) int
int64_t index_fold(unsigned char *haystack, const uint64_t haystack_len, unsigned char *needle, const uint64_t needle_len)
{
    const uint64_t blockSize = 16; // NEON can process 128 bits (16 bytes) at a time

    const uint64_t checked_len = haystack_len - needle_len;
    const uint8x16x2_t table = vld1q_u8_x2(uppercasingTable);
    const uint8x16_t shift = vdupq_n_u8(0x60);

    if (haystack_len < needle_len) return -1;
    if (needle_len == 0) return 0;
    if (haystack_len == needle_len) {
        return equal_fold_core(haystack, needle, needle_len, table, shift) ? 0 : -1;
    }

    switch (needle_len)
    {
    case 1:
        // special case for 1-byte needles
        return index_fold_1_byte_needle(haystack, haystack_len, *(uint8_t *)needle, table);
    case 2:
        // special case for 2-byte needles, no need for two loads
        return index_fold_2_byte_needle(haystack, haystack_len, (uint16_t *)needle, table);
    }

    // load the first 2 bytes of the needle
    const uint16x8_t first2 = index_fold_prepare_comparer((uint16_t *)needle, shift, table);
    // load the last 2 bytes of the needle
    const uint16x8_t last2 = index_fold_prepare_comparer((uint16_t *)(needle + needle_len - 2), shift, table);

    uint8x16_t prev_data = vdupq_n_u8(0);
    uint8x16_t prev_data_end = vdupq_n_u8(0);
    uint64_t curr_pos = 0;

    for (const unsigned char *data_bound = haystack + checked_len + 1; haystack <= data_bound; haystack += blockSize, curr_pos += blockSize)
    {
        const int64_t data_len = haystack_len - curr_pos;
        uint8x16_t data = load_data16(haystack, data_len);
        uint8x16_t data_end = load_data16(haystack + needle_len - 2, data_len - needle_len + 2);

        uint64_t data64 = index_fold_process_block(data, data_end, first2, last2, table, shift, &prev_data, &prev_data_end);
        while (data64)
        {
            int pos = __builtin_ctzll(data64) / 4;
            // clear this byte position
            data64 &= ~(0xFull << (pos * 4));
            // disregard 0th byte on the first iteration, we made up a 0 on that position
            if (curr_pos == 0 && pos == 0) continue;
            pos--;

            if (haystack+pos < data_bound && equal_fold_core(haystack+pos, needle, needle_len, table, shift))
            {
                return curr_pos + pos;
            }
        }
    }

    return -1;
}