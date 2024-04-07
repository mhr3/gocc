#!/bin/bash

go run ../cmd/gocc/. -l -o simd matmul_avx2.c --arch avx2 -O3
go run ../cmd/gocc/. -l -o simd matmul_neon.c --arch neon -O3

go run ../cmd/gocc/. -l -o simd test_simd_mul.c --arch avx2 -O3 --suffix _amd64_avx2 --function-suffix _avx2
go run ../cmd/gocc/. -l -o simd test_simd_mul.c --arch avx512 -O3 --suffix _amd64_avx512 --function-suffix _avx512
go run ../cmd/gocc/. -l -o simd test_simd_mul.c --arch amd64 -O3 --suffix _amd64_sse --function-suffix _sse
go run ../cmd/gocc/. -l -o simd test_simd_mul.c --arch neon -O3 --suffix _arm64_neon --function-suffix _neon
go run ../cmd/gocc/. -l -o simd test_simd_mul.c --arch sve -O3 --suffix _arm64_sve --function-suffix _sve
go run ../cmd/gocc/. -l -o simd test_simd_mul_sve.c --arch sve -O3

go run ../cmd/gocc/. -l -o simd memcmp_sve.c --arch sve -O3
