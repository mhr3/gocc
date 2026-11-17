#!/bin/bash
# Regenerates the assembly checked into this directory.
#
#   gen.sh [arch...]    arch is amd64 or arm64; both when none is given
#
# Translating for an architecture other than the host's needs that
# architecture's C library headers.
set -euo pipefail
cd "$(dirname "$0")"

gocc() {
	go run ../../../cmd/gocc "$@" -l -O3
}

archs=("$@")
if [ ${#archs[@]} -eq 0 ]; then
	archs=(amd64 arm64)
fi

for arch in "${archs[@]}"; do
	case "$arch" in
	amd64 | arm64) ;;
	*)
		echo "gen.sh: unknown architecture $arch" >&2
		exit 1
		;;
	esac

	gocc testdata/test_src.c --arch "$arch" --suffix "_$arch"
	gocc testdata/test_calls.c --arch "$arch" --suffix "_$arch" --with-internal-functions

	# test_stack.c sizes its vectors by the instruction set, so it is
	# translated for every one of them.
	if [ "$arch" = amd64 ]; then
		gocc testdata/test_stack.c --arch amd64 --suffix _amd64_sse2 --function-suffix _sse2 --with-internal-functions
		gocc testdata/test_stack.c --arch amd64 --suffix _amd64_sse4 --function-suffix _sse4 --with-internal-functions -msse4.2
		gocc testdata/test_stack.c --arch avx2 --suffix _amd64_avx2 --function-suffix _avx2 --with-internal-functions
		gocc testdata/test_stack.c --arch avx512 --suffix _amd64_avx512 --function-suffix _avx512 --with-internal-functions -mavx512bw
	else
		gocc testdata/test_stack.c --arch arm64 --suffix _arm64 --function-suffix _neon --with-internal-functions
	fi
done
