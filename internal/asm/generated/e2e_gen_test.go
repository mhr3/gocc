package generated_test

//go:generate go run ../../../cmd/gocc testdata/test_src.c -l --suffix _amd64 --arch amd64 -O3 --package generated
//go:generate go run ../../../cmd/gocc testdata/test_src.c -l --suffix _arm64 --arch arm64 -O3 --package generated
