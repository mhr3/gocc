#!/bin/bash

#go run ../cmd/gocc/. -l -o utf8 valid-sse.c --arch amd64 -O3
go run ../cmd/gocc/. -l -o utf8 range-neon.c --arch arm64 -O3
go run ../cmd/gocc/. -l -o utf8 lemire-neon.c --arch apple -O3
