#!/bin/bash

go run ../cmd/gocc/. matmul_avx.c -l --arch avx2 -O3 -o simd --package simd
go run ../cmd/gocc/. matmul_neon.c -l --arch neon -O3 -o simd --package simd
go run ../cmd/gocc/. test_simd_mul.c -l --arch avx2 -O3 -o simd --suffix _avx2 --package simd
go run ../cmd/gocc/. test_simd_mul.c -l --arch arm64 -O3 -o simd --suffix _arm64 --package simd
