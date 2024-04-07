// Copyright (c) Michal Hruby and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

#include <stdint.h>
#include <arm_sve.h>

// gocc: memcmp_sve(x, y []byte) int
long memcmp_sve(uint8_t *x, uint64_t x_len, uint64_t x_cap, uint8_t *y, uint64_t y_len, uint64_t y_cap) {
    svbool_t predicate;
    svbool_t res;
    svuint8_t xseg;
    svuint8_t yseg;

    if (x_len != y_len) {
        return x_len < y_len ? -1 : 1;
    }

    uint64_t i = 0;
    uint64_t size = x_len < y_len ? x_len : y_len;

    while (i < size) {
        predicate = svwhilelt_b8_u64(i, size);
        // load in a vectors worth of x and y values
        xseg = svld1_u8(predicate, x+i); // ld1w for x
        yseg = svld1_u8(predicate, y+i); // ld1w for y
  
        res = svcmpne_u8(predicate, xseg, yseg);
        if (svptest_any(predicate, res)) {
            res = svbrkb_z(predicate, res);
            uint8_t a = svlasta_u8(res, xseg);
            uint8_t b = svlasta_u8(res, yseg);
            return a < b ? -1 : 1;
        }
  
        i+=svcntb();
    }

    return 0;
}