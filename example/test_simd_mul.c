// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

#include <stdint.h>

// gocc: uint8_simd_mul(input1, input2, output *byte, size uint64)
void uint8_simd_mul(uint8_t *input1, uint8_t *input2, uint8_t *output, uint64_t size) {
    #pragma clang loop vectorize(enable) interleave(enable)
    for (int i = 0; i < (int)size; i++) {
        output[i] = input1[i] * input2[i];
    }
}