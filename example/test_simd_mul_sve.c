// Copyright (c) Michal Hruby and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

#include <stdint.h>
#include <arm_sve.h>

// gocc: uint8_simd_mul_sve_manual(input1, input2, output *byte, size uint64)
void uint8_simd_mul_sve_manual(uint8_t *input1, uint8_t *input2, uint8_t *output, uint64_t size) {
    uint64_t i;

    svbool_t predicate;
    svuint8_t xseg;
    svuint8_t yseg;
  
    // get the vector length being used, so we know how to increment the loop (1)
    uint64_t numVals = svlen_u8(xseg);
  
    for (i=0; i<size; i+=numVals) { // (2)
        // set predicate based off loop counter (3)
        predicate = svwhilelt_b8_u64(i, size);

        // load in a vectors worth of x and y values (4)
        xseg = svld1_u8(predicate, input1+i); // ld1w for x
        yseg = svld1_u8(predicate, input2+i); // ld1w for y
  
        yseg = svmul_u8_m(predicate, xseg, yseg);
  
        svst1_u8(predicate, output+i, yseg); // st1w for y <-y+a*x
    }
}