/*
 * Copied from https://github.com/cyb70289/utf8
 */
#include <stdint.h>
#include <stdbool.h>
#include <arm_neon.h>

/*
 * Map high nibble of "First Byte" to legal character length minus 1
 * 0x00 ~ 0xBF --> 0
 * 0xC0 ~ 0xDF --> 1
 * 0xE0 ~ 0xEF --> 2
 * 0xF0 ~ 0xFF --> 3
 */
static const uint8_t _first_len_tbl[] = {
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 2, 3,
};

/* Map "First Byte" to 8-th item of range table (0xC2 ~ 0xF4) */
static const uint8_t _first_range_tbl[] = {
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 8, 8, 8, 8,
};

/*
 * Range table, map range index to min and max values
 * Index 0    : 00 ~ 7F (First Byte, ascii)
 * Index 1,2,3: 80 ~ BF (Second, Third, Fourth Byte)
 * Index 4    : A0 ~ BF (Second Byte after E0)
 * Index 5    : 80 ~ 9F (Second Byte after ED)
 * Index 6    : 90 ~ BF (Second Byte after F0)
 * Index 7    : 80 ~ 8F (Second Byte after F4)
 * Index 8    : C2 ~ F4 (First Byte, non ascii)
 * Index 9~15 : illegal: u >= 255 && u <= 0
 */
static const uint8_t _range_min_tbl[] = {
    0x00, 0x80, 0x80, 0x80, 0xA0, 0x80, 0x90, 0x80,
    0xC2, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
};
static const uint8_t _range_max_tbl[] = {
    0x7F, 0xBF, 0xBF, 0xBF, 0xBF, 0x9F, 0xBF, 0x8F,
    0xF4, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
};

/*
 * This table is for fast handling four special First Bytes(E0,ED,F0,F4), after
 * which the Second Byte are not 80~BF. It contains "range index adjustment".
 * - The idea is to minus byte with E0, use the result(0~31) as the index to
 *   lookup the "range index adjustment". Then add the adjustment to original
 *   range index to get the correct range.
 * - Range index adjustment
 *   +------------+---------------+------------------+----------------+
 *   | First Byte | original range| range adjustment | adjusted range |
 *   +------------+---------------+------------------+----------------+
 *   | E0         | 2             | 2                | 4              |
 *   +------------+---------------+------------------+----------------+
 *   | ED         | 2             | 3                | 5              |
 *   +------------+---------------+------------------+----------------+
 *   | F0         | 3             | 3                | 6              |
 *   +------------+---------------+------------------+----------------+
 *   | F4         | 4             | 4                | 8              |
 *   +------------+---------------+------------------+----------------+
 * - Below is a uint8x16x2 table, data is interleaved in NEON register. So I'm
 *   putting it vertically. 1st column is for E0~EF, 2nd column for F0~FF.
 */
static const uint8_t _range_adjust_tbl[] = {
    /* index -> 0~15  16~31 <- index */
    /*  E0 -> */ 2,     3, /* <- F0  */
                 0,     0,
                 0,     0,
                 0,     0,
                 0,     4, /* <- F4  */
                 0,     0,
                 0,     0,
                 0,     0,
                 0,     0,
                 0,     0,
                 0,     0,
                 0,     0,
                 0,     0,
    /*  ED -> */ 3,     0,
                 0,     0,
                 0,     0,
};

static inline void utf8_range_process_block(const uint8x16_t input,
    uint8x16_t *prev_input, uint8x16_t *prev_first_len,
    uint8x16_t *error1, uint8x16_t *error2,
    const uint8x16_t first_len_tbl, const uint8x16_t first_range_tbl,
    const uint8x16_t const_1, const uint8x16_t const_2, const uint8x16_t const_e0,
    const uint8x16x2_t range_adjust_tbl, const uint8x16_t range_min_tbl, const uint8x16_t range_max_tbl)
{
    /* high_nibbles = input >> 4 */
    const uint8x16_t high_nibbles = vshrq_n_u8(input, 4);

    /* first_len = legal character length minus 1 */
    /* 0 for 00~7F, 1 for C0~DF, 2 for E0~EF, 3 for F0~FF */
    /* first_len = first_len_tbl[high_nibbles] */
    const uint8x16_t first_len =
        vqtbl1q_u8(first_len_tbl, high_nibbles);

    /* First Byte: set range index to 8 for bytes within 0xC0 ~ 0xFF */
    /* range = first_range_tbl[high_nibbles] */
    uint8x16_t range = vqtbl1q_u8(first_range_tbl, high_nibbles);

    /* Second Byte: set range index to first_len */
    /* 0 for 00~7F, 1 for C0~DF, 2 for E0~EF, 3 for F0~FF */
    /* range |= (first_len, prev_first_len) << 1 byte */
    range = vorrq_u8(range, vextq_u8(*prev_first_len, first_len, 15));

    /* Third Byte: set range index to saturate_sub(first_len, 1) */
    /* 0 for 00~7F, 0 for C0~DF, 1 for E0~EF, 2 for F0~FF */
    uint8x16_t tmp1, tmp2;
    /* tmp1 = (first_len, prev_first_len) << 2 bytes */
    tmp1 = vextq_u8(*prev_first_len, first_len, 14);
    /* tmp1 = saturate_sub(tmp1, 1) */
    tmp1 = vqsubq_u8(tmp1, const_1);
    /* range |= tmp1 */
    range = vorrq_u8(range, tmp1);

    /* Fourth Byte: set range index to saturate_sub(first_len, 2) */
    /* 0 for 00~7F, 0 for C0~DF, 0 for E0~EF, 1 for F0~FF */
    /* tmp2 = (first_len, prev_first_len) << 3 bytes */
    tmp2 = vextq_u8(*prev_first_len, first_len, 13);
    /* tmp2 = saturate_sub(tmp2, 2) */
    tmp2 = vqsubq_u8(tmp2, const_2);
    /* range |= tmp2 */
    range = vorrq_u8(range, tmp2);

    /*
     * Now we have below range indices caluclated
     * Correct cases:
     * - 8 for C0~FF
     * - 3 for 1st byte after F0~FF
     * - 2 for 1st byte after E0~EF or 2nd byte after F0~FF
     * - 1 for 1st byte after C0~DF or 2nd byte after E0~EF or
     *         3rd byte after F0~FF
     * - 0 for others
     * Error cases:
     *   9,10,11 if non ascii First Byte overlaps
     *   E.g., F1 80 C2 90 --> 8 3 10 2, where 10 indicates error
     */

    /* Adjust Second Byte range for special First Bytes(E0,ED,F0,F4) */
    /* See _range_adjust_tbl[] definition for details */
    /* Overlaps lead to index 9~15, which are illegal in range table */
    uint8x16_t shift1 = vextq_u8(*prev_input, input, 15);
    uint8x16_t pos = vsubq_u8(shift1, const_e0);
    range = vaddq_u8(range, vqtbl2q_u8(range_adjust_tbl, pos));

    /* Load min and max values per calculated range index */
    uint8x16_t minv = vqtbl1q_u8(range_min_tbl, range);
    uint8x16_t maxv = vqtbl1q_u8(range_max_tbl, range);

    /* Check value range */
    *error1 = vorrq_u8(*error1, vcltq_u8(input, minv));
    *error2 = vorrq_u8(*error2, vcgtq_u8(input, maxv));

    *prev_first_len = first_len;
    *prev_input = input;
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

// gocc: utf8_valid_range(src string) bool
bool utf8_valid_range(const unsigned char *data, int64_t len)
{
    uint8x16_t prev_input = vdupq_n_u8(0);
    uint8x16_t prev_first_len = vdupq_n_u8(0);

    /* Cached tables */
    const uint8x16_t first_len_tbl = vld1q_u8(_first_len_tbl);
    const uint8x16_t first_range_tbl = vld1q_u8(_first_range_tbl);
    const uint8x16_t range_min_tbl = vld1q_u8(_range_min_tbl);
    const uint8x16_t range_max_tbl = vld1q_u8(_range_max_tbl);
    const uint8x16x2_t range_adjust_tbl = vld2q_u8(_range_adjust_tbl);

    /* Cached values */
    const uint8x16_t const_1 = vdupq_n_u8(1);
    const uint8x16_t const_2 = vdupq_n_u8(2);
    const uint8x16_t const_e0 = vdupq_n_u8(0xE0);

    /* We use two error registers to remove a dependency. */
    uint8x16_t error1 = vdupq_n_u8(0);
    uint8x16_t error2 = vdupq_n_u8(0);

main_loop:
    while (len >= 16) {
        const uint8x16_t input = vld1q_u8(data);

        utf8_range_process_block(input,
            &prev_input, &prev_first_len, &error1, &error2,
            first_len_tbl, first_range_tbl, const_1, const_2, const_e0,
            range_adjust_tbl, range_min_tbl, range_max_tbl
        );

        data += 16;
        len -= 16;

        /* Perform error check every now and then */
        if (len % 128 < 16) {
            const uint8x16_t error = vorrq_u8(error1, error2);

            if (vmaxvq_u32(error)) return false;
        }
    }
    /* Merge our error counters together */
    error1 = vorrq_u8(error1, error2);

    if (vmaxvq_u32(error1)) return false;

    /* Find previous token (not 80~BF) */
    uint32_t token4 = vgetq_lane_u32(vreinterpretq_u32_u8(prev_input), 3);

    const int8_t *token = (const int8_t *)&token4;
    int lookahead = 0;
    if (token[3] > (int8_t)0xBF)
        lookahead = 1;
    else if (token[2] > (int8_t)0xBF)
        lookahead = 2;
    else if (token[1] > (int8_t)0xBF)
        lookahead = 3;

    data -= lookahead;
    len += lookahead;

    if (len == 0) return true;

    prev_input = vdupq_n_u8(0);
    prev_first_len = vdupq_n_u8(0);

    // The lookahead might cause us having to look at more than 16 bytes
    if (len >= 16) {
        goto main_loop;
    }

    // Handle the remaining bytes (must be <= 16)
    const uint8x16_t input = load_data16(data, len);

    utf8_range_process_block(input,
        &prev_input, &prev_first_len, &error1, &error2,
        first_len_tbl, first_range_tbl, const_1, const_2, const_e0,
        range_adjust_tbl, range_min_tbl, range_max_tbl
    );

    error1 = vorrq_u8(error1, error2);

    return vmaxvq_u8(error1) ? false : true;
}
