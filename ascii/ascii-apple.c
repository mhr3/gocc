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
