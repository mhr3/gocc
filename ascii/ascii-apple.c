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
        uint8x16_t msb_mask = vdupq_n_u8(0x80); // Create a mask for the MSB

#ifndef SKIP_LD1x4
        for (const unsigned char *data64_end = (data + length) - (length % ld4BlockSize); data < data64_end; data += ld4BlockSize)
        {
            uint8x16x4_t blocks = vld1q_u8_x4(data);           // Load 64 bytes of data
            blocks.val[0] = vtstq_u8(blocks.val[0], msb_mask); // AND with the mask to isolate MSB
            blocks.val[1] = vtstq_u8(blocks.val[1], msb_mask); // AND with the mask to isolate MSB
            blocks.val[2] = vtstq_u8(blocks.val[2], msb_mask); // AND with the mask to isolate MSB
            blocks.val[3] = vtstq_u8(blocks.val[3], msb_mask); // AND with the mask to isolate MSB

            uint8x16_t result = vorrq_u8(vorrq_u8(blocks.val[0], blocks.val[1]), vorrq_u8(blocks.val[2], blocks.val[3])); // OR the results
            // Check if there's any set bit (u32 is faster than u8)
            if (vmaxvq_u32(result) > 0)
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
            // Check if there's any set bit (u32 is faster than u8)
            if (vmaxvq_u32(result) > 0)
            {
                return false;
            }
        }
        length %= blockSize;
    }

    if (length >= 8)
    {
        uint64_t data64 = *(uint64_t *)(data);
        if (data64 & 0x8080808080808080ull)
        {
            return false;
        }
        data += 8;
        length -= 8;
    }

    uint32_t data32;

    if (length >= 4)
    {
        data32 = *(uint32_t *)(data);
        if ((data32 & 0x80808080) != 0)
        {
            return false;
        }
        data += 4;
        length -= 4;
    }

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
            // Check if there's any set bit (u32 is faster than u8)
            if (vmaxvq_u32(result) > 0)
            {
                // now a few operations to find the index of the first set bit
                for (int j = 0; j < 4; j++)
                {
                    uint64_t data64lo = vgetq_lane_u64(vreinterpretq_u64_u8(blocks.val[j]), 0);
                    uint64_t data64hi = vgetq_lane_u64(vreinterpretq_u64_u8(blocks.val[j]), 1);
                    if (data64lo != 0)
                    {
                        int64_t offset = j*16 + __builtin_ctzll(data64lo) / 8;
                        return (data - data_start) + offset;
                    }
                    else if (data64hi != 0)
                    {
                        int64_t offset = j*16 + 8 + __builtin_ctzll(data64hi) / 8;
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
            // Check if there's any set bit (u32 is faster than u8)
            if (vmaxvq_u32(result) > 0)
            {
                // now a few operations to find the index of the first set bit
                uint64_t data64lo = vgetq_lane_u64(vreinterpretq_u64_u8(result), 0);
                uint64_t data64hi = vgetq_lane_u64(vreinterpretq_u64_u8(result), 1);
                int64_t offset = 0;
                if (data64lo != 0)
                {
                    offset += __builtin_ctzll(data64lo) / 8;
                }
                else
                {
                    offset += 8 + __builtin_ctzll(data64hi) / 8;
                }
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

    if (length >= 8)
    {
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

            // Check if there's any 0 bytes (u32 is faster than u8)
            if (vminvq_u32(result) != 0xFFFFFFFF)
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
            if (vget_lane_u64(result, 0) != 0xFFFFFFFFFFFFFFFFull)
            {
                return false;
            }
            a += 8;
            b += 8;
            length -= 8;
        }
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
        a_data64 <<= 16;
        a_data64 |= *(uint16_t *)(a);
        a_data64 <<= 8;
        a_data64 |= a[2];
        // same for b
        b_data64 <<= 16;
        b_data64 |= *(uint16_t *)(b);
        b_data64 <<= 8;
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

    // Check if there's any 0 bytes (u16 is faster than u8)
    if (vget_lane_u64(result, 0) != 0xFFFFFFFFFFFFFFFFull)
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
    } else if (len <= 0) {
        return vdupq_n_u8(0);
    }

    const int64_t orig_len = len;
    uint64_t data64 = 0;
    uint64_t data64lo;

    if (len & 8) {
        data64 = *(uint64_t *)(src);
        src += 8;
        len -= 8;
    }

    if (len == 0) {
        return vcombine_u64(vcreate_u64(data64), vcreate_u64(0));
    } else {
        data64lo = data64;
        data64 = 0;
    }

    if (len & 4) {
        data64 = *(uint32_t *)(src);
        src += 4;
        len -= 4;
    }
    if (len & 2) {
        uint64_t tmp = *(uint16_t *)(src);
        data64 |= tmp << (8 * (orig_len & 4));
        src += 2;
        len -= 2;
    }
    if (len & 1) {
        uint64_t tmp = *src;
        data64 |= tmp << (8 * (orig_len & 6));
    }

    if (orig_len < 8) {
        return vcombine_u64(vcreate_u64(data64), vcreate_u64(0));
    }
    return vcombine_u64(vcreate_u64(data64lo), vcreate_u64(data64));
}

// gocc: contains_fold(a, b string) bool
bool contains_fold(unsigned char *haystack, uint64_t haystack_len, unsigned char *needle, uint64_t needle_len)
{
    const uint64_t blockSize = 16; // NEON can process 128 bits (16 bytes) at a time

    if (haystack_len < needle_len)
    {
        return false;
    }

    const uint64_t checked_len = haystack_len - needle_len;
    const uint8x16x2_t table = vld1q_u8_x2(uppercasingTable);
    const uint8x16_t shift = vdupq_n_u8(0x60);

    // load the first 2 bytes of the needle
    uint8x16_t needle8 = vreinterpretq_u8_u16(vld1q_dup_u16((uint16_t *)needle));
    needle8 = vsubq_u8(needle8, shift);
    needle8 = vsubq_u8(needle8, vqtbl2q_u8(table, needle8));
    // the needle is lowercased and shifted
    const uint16x8_t first16 = vreinterpretq_u16_u8(needle8);

    if (needle_len == 2)
    {
        // we're checking two bytes at a time, so we can't advance by full 16 bytes
        const uint64_t advance = blockSize-1;
        uint64_t curr_pos = 0;

        for (const unsigned char *data_bound = haystack + checked_len; haystack <= data_bound; haystack += advance, curr_pos += advance)
        {
            uint8x16_t data = load_data16(haystack, haystack_len - curr_pos);

            data = vsubq_u8(data, shift);
            data = vsubq_u8(data, vqtbl2q_u8(table, data));

            // operating on shifted data
            const uint16x8_t res1 = vceqq_u16(data, first16);
            data = vextq_u8(data, vdupq_n_u8(0), 1);
            const uint16x8_t res2 = vceqq_u16(data, first16);

            const uint16x8_t combined = vorrq_u16(vshrq_n_u16(res1, 8), vshlq_n_u16(res2, 8));
            const uint8x8_t narrowed = vshrn_n_u16(combined, 4);

            uint64_t data64 = vget_lane_u64(narrowed, 0);
            if (data64)
            {
                const int pos = __builtin_ctzll(data64) / 4;
                // 15th byte is never valid, we made up a 0 on that position
                if (pos < 15 && haystack+pos <= data_bound) return true;
            }
        }

        return false;
    }

    // load the last 2 bytes of the needle
    needle8 = vreinterpretq_u8_u16(vld1q_dup_u16((uint16_t *)(needle + needle_len - 2)));
    needle8 = vsubq_u8(needle8, shift);
    needle8 = vsubq_u8(needle8, vqtbl2q_u8(table, needle8));
    const uint16x8_t last16 = vreinterpretq_u16_u8(needle8);

    const uint64_t advance = blockSize-1;
    uint64_t curr_pos = 0;

    for (const unsigned char *data_bound = haystack + checked_len; haystack <= data_bound; haystack += advance, curr_pos += advance)
    {
        const int64_t data_len = haystack_len - curr_pos;
        uint8x16_t data = load_data16(haystack, data_len);
        uint8x16_t data_end = load_data16(haystack + needle_len - 2, data_len - needle_len + 2);

        data = vsubq_u8(data, shift);
        data = vsubq_u8(data, vqtbl2q_u8(table, data));

        data_end = vsubq_u8(data_end, shift);
        data_end = vsubq_u8(data_end, vqtbl2q_u8(table, data_end));

        // operating on shifted data
        const uint16x8_t res1 = vandq_u16(vceqq_u16(data, first16), vceqq_u16(data_end, last16));

        data = vextq_u8(data, vdupq_n_u8(0), 1);
        data_end = vextq_u8(data_end, vdupq_n_u8(0), 1);

        const uint16x8_t res2 = vandq_u16(vceqq_u16(data, first16), vceqq_u16(data_end, last16));

        const uint16x8_t combined = vorrq_u16(vshrq_n_u16(res1, 8), vshlq_n_u16(res2, 8));
        const uint8x8_t narrowed = vshrn_n_u16(combined, 4);

        uint64_t data64 = vget_lane_u64(narrowed, 0);
        while (data64)
        {
            const int pos = __builtin_ctzll(data64) / 4;
            // 15th byte is never valid, we made up a 0 on that position
            if (pos < 15 && haystack+pos <= data_bound && equal_fold_core(haystack + pos, needle, needle_len, table, shift)) return true;
            // clear the byte we just checked
            data64 &= ~(0xFull << (pos * 4));
        }
    }

    return false;
}