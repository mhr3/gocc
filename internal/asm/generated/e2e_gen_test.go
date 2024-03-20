package generated_test

//go:generate go run ../../../cmd/gocc testdata/test_amd64.c -l --arch avx2 -O3 --package generated
//go:generate go run ../../../cmd/gocc testdata/test_arm64.c -l --arch apple -O3 --package generated
