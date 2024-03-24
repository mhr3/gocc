#!/bin/bash

go run ../cmd/gocc/. -l -o simd matmul_avx2.c --arch avx2 -O3
go run ../cmd/gocc/. -l -o simd matmul_neon.c --arch neon -O3
go run ../cmd/gocc/. -l -o simd test_simd_mul.c --arch avx2 -O3 --suffix _amd64_avx2 --function-suffix _avx2
go run ../cmd/gocc/. -l -o simd test_simd_mul.c --arch amd64 -O3 --suffix _amd64_sse --function-suffix _sse
go run ../cmd/gocc/. -l -o simd test_simd_mul.c --arch arm64 -O3 --suffix _arm64
