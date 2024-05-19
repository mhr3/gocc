#include <stdint.h>
#include <stdbool.h>
#include <arm_neon.h>

// gocc: IndexNonASCII(data string) int
int64_t index_nonascii(const unsigned char *data, const uint64_t length){
    const uint64_t blockSize = 16; // NEON can process 128 bits (16 bytes) at a time

    uint64_t i = 0;

    if (length >= 8)
    {
        uint8x16_t msb_mask = vdupq_n_u8(0x80); // Create a mask for the MSB
        uint64x2_t loader = vdupq_n_u64(0x0807060504030201ull);
        loader = vsetq_lane_u64(0x100f0e0d0c0b0a09ull, loader, 1);
        uint8x16_t indexes = vreinterpretq_u8_u64(loader);

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

// gocc: IsASCII(data string) bool
bool is_ascii(unsigned char *data, uint64_t length){
    const uint64_t blockSize = 16; // NEON can process 128 bits (16 bytes) at a time

    if (length >= 8)
    {
        uint8x16_t msb_mask = vdupq_n_u8(0x80); // Create a mask for the MSB

        for (const unsigned char *neon_end = (data + length) - (length % blockSize); data < neon_end; data += blockSize)
        {
            uint8x16_t block = vld1q_u8(data);         // Load 16 bytes of data
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
            uint64_t block = *(uint64_t*)(data);
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
