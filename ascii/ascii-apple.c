#include <stdint.h>
#include <stdbool.h>
#include <arm_neon.h>

// gocc: IsASCII(data string) bool
bool is_ascii(unsigned char *data, uint64_t length)
{
    const uint64_t blockSize = 16; // NEON can process 128 bits (16 bytes) at a time
    const uint64_t ld4BlockSize = blockSize * 4;

    if (length >= 8)
    {
        uint8x16_t msb_mask = vdupq_n_u8(0x80); // Create a mask for the MSB

#ifndef SKIP_LD1x4
        for (const unsigned char *data_bound = (data + length) - (length % ld4BlockSize); data < data_bound; data += ld4BlockSize)
        {
            uint8x16x4_t blocks = vld1q_u8_x4(data);           // Load 64 bytes of data
            blocks.val[0] = vandq_u8(blocks.val[0], msb_mask); // AND with the mask to isolate MSB
            blocks.val[1] = vandq_u8(blocks.val[1], msb_mask); // AND with the mask to isolate MSB
            blocks.val[2] = vandq_u8(blocks.val[2], msb_mask); // AND with the mask to isolate MSB
            blocks.val[3] = vandq_u8(blocks.val[3], msb_mask); // AND with the mask to isolate MSB

            uint8x16_t result = vorrq_u8(vorrq_u8(blocks.val[0], blocks.val[1]), vorrq_u8(blocks.val[2], blocks.val[3])); // OR the results
            // Check if there's any set bit (u32 is faster than u8)
            if (vmaxvq_u32(result) > 0)
            {
                return false;
            }
        }
        length %= ld4BlockSize;
#endif
        for (const unsigned char *data_bound = (data + length) - (length % blockSize); data < data_bound; data += blockSize)
        {
            uint8x16_t block = vld1q_u8(data);             // Load 16 bytes of data
            uint8x16_t result = vandq_u8(block, msb_mask); // AND with the mask to isolate MSB
            // Check if there's any set bit (u32 is faster than u8)
            if (vmaxvq_u32(result) > 0)
            {
                return false;
            }
        }
        length %= blockSize;

        if (length >= 8)
        {
            uint64_t block = *(uint64_t *)(data);
            if (block & 0x8080808080808080ull)
            {
                return false;
            }
            data += 8;
            length -= 8;
        }
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

static uint8_t indexesTable[16] = {
    1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16};

// gocc: IndexNonASCII(data string) int
int64_t index_nonascii(const unsigned char *data, const uint64_t length)
{
    const uint64_t blockSize = 16; // NEON can process 128 bits (16 bytes) at a time

    uint64_t i = 0;

    if (length >= 8)
    {
        uint8x16_t msb_mask = vdupq_n_u8(0x80); // Create a mask for the MSB
        // uint64x2_t loader = vdupq_n_u64(0x0807060504030201ull);
        // loader = vsetq_lane_u64(0x100f0e0d0c0b0a09ull, loader, 1);
        uint8x16_t indexes = vld1q_u8(indexesTable);

        for (; i + blockSize <= length; i += blockSize)
        {
            uint8x16_t block = vld1q_u8(data + i);         // Load 16 bytes of data
            uint8x16_t result = vandq_u8(block, msb_mask); // AND with the mask to isolate MSB
            // Check if there's any set bit (u32 is faster than u8)
            if (vmaxvq_u32(result) > 0)
            {
                // now a few operations to find the index of the first set bit
                result = vshrq_n_u8(result, 7); // Shift the MSB to the LSB
                // multiply the result by the indexes
                result = vmulq_u8(result, indexes);
                // we'll use minv, so need to invert zeroes
                uint8x16_t mmask = vceqzq_u8(result);
                uint8x16_t multiplied = vorrq_u8(result, mmask);

                return vminvq_u8(multiplied) + i - 1;
            }
        }

        if (i + 8 <= length)
        {
            // same as above, but for 8 bytes
            uint8x8_t block = vld1_u8(data + i);
            uint8x8_t result = vand_u8(block, vget_low_u8(msb_mask));
            if (vmaxv_u16(result) > 0) // faster than u8
            {
                result = vshr_n_u8(result, 7);
                result = vmul_u8(result, vget_low_u8(indexes));
                uint8x8_t mmask = vceqz_u8(result);
                uint8x8_t multiplied = vorr_u8(result, mmask);
                return vminv_u8(multiplied) + i - 1;
            }
            i += 8;
        }
    }

    if (i + 4 <= length)
    {
        uint32_t data32 = *(uint32_t *)(data + i);
        data32 &= 0x80808080;
        if (data32 != 0)
        {
            return i + __builtin_ctz(data32) / 8;
        }
        i += 4;
    }

    // Handle the remaining bytes (if any)
    for (; i < length; i++)
    {
        if (data[i] & 0x80)
        {
            return i;
        }
    }

    return -1;
}

static uint8_t uppercasingTable[32] = {
    0,
    32,
    32,
    32,
    32,
    32,
    32,
    32,
    32,
    32,
    32,
    32,
    32,
    32,
    32,
    32,
    32,
    32,
    32,
    32,
    32,
    32,
    32,
    32,
    32,
    32,
    32,
    0,
    0,
    0,
    0,
    0,
};

// gocc: EqualFold(a, b string) bool
bool equal_fold(unsigned char *a, uint64_t a_len, unsigned char *b, uint64_t b_len)
{
    const uint64_t blockSize = 16; // NEON can process 128 bits (16 bytes) at a time

    if (a_len != b_len)
    {
        return false;
    }

    uint64_t length = a_len;
    uint8x16x2_t table = vld1q_u8_x2(uppercasingTable);
    uint8x16_t shift = vdupq_n_u8(0x60);

    if (length >= 8)
    {
        for (const unsigned char *data_bound = (a + length) - (length % blockSize); a < data_bound; a += blockSize, b += blockSize)
        {
            uint8x16_t a_data = vld1q_u8(a); // Load 16 bytes of data
            uint8x16_t b_data = vld1q_u8(b); // Load 16 bytes of data

            a_data = vsubq_u8(a_data, shift);
            b_data = vsubq_u8(b_data, shift);

            uint8x16_t a_upper = vqtbl2q_u8(table, a_data);
            uint8x16_t b_upper = vqtbl2q_u8(table, b_data);

            a_data = vsubq_u8(a_data, a_upper);
            b_data = vsubq_u8(b_data, b_upper);

            // we should shift the data back, but we just need to compare, so we can skip that
            uint8x16_t result = vceqq_u8(a_data, b_data);

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
            b_data = vsub_u8(b_data, vget_low_u8(shift));

            uint8x8_t a_upper = vqtbl2_u8(table, a_data);
            uint8x8_t b_upper = vqtbl2_u8(table, b_data);

            a_data = vsub_u8(a_data, a_upper);
            b_data = vsub_u8(b_data, b_upper);

            // we should shift the data back, but we just need to compare, so we can skip that
            uint8x8_t result = vceq_u8(a_data, b_data);

            // Check if there's any 0 bytes (u16 is faster than u8)
            if (vminv_u16(result) != 0xFFFF)
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
    b_data = vsub_u8(b_data, vget_low_u8(shift));

    uint8x8_t a_upper = vqtbl2_u8(table, a_data);
    uint8x8_t b_upper = vqtbl2_u8(table, b_data);

    a_data = vsub_u8(a_data, a_upper);
    b_data = vsub_u8(b_data, b_upper);

    // we should shift the data back, but we just need to compare, so we can skip that
    uint8x8_t result = vceq_u8(a_data, b_data);

    // Check if there's any 0 bytes (u16 is faster than u8)
    if (vminv_u16(result) != 0xFFFF)
    {
        return false;
    }

    return true;
}
