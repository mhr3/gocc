#!/bin/bash

go run ../cmd/gocc/. -l -o ascii ascii-avx2.c --arch avx2 -O3
go run ../cmd/gocc/. -l -o ascii ascii-sse.c --arch amd64 -O3
go run ../cmd/gocc/. -l -o ascii ascii-neon.c --arch neon -O3
#go run ../cmd/gocc/. -l -o ascii ascii-sve.c --arch sve -O3
#go run ../cmd/gocc/. -l -o ascii ascii-apple.c --arch apple -O3
