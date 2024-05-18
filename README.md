<p align="center">
<img width="250" height="110" src=".github/logo.png" border="0" alt="mhr3/gocc">
<br>
<img src="https://img.shields.io/github/go-mod/go-version/mhr3/gocc" alt="Go Version">
<a href="https://opensource.org/license/apache-2-0/"><img src="https://img.shields.io/badge/License-Apache-blue.svg" alt="License"></a>
</p>

## GOCC: Compile C to Go Assembly

This utility transpiles C code to Go assembly. It uses the LLVM toolchain to compile C code to assembly and machine code and generates Go assembly from it, as well as the corresponding Go stubs. This is useful for certain features such as using intrinsics, which are not supported by the Go ecosystem.
Generated Go assembly will try to use at little binary codes as possible, making the output more suitable
for tuning, or to give you a starting point, but do note that not everything can be expressed in Plan9
assembly plus the disassembly process is sometimes buggy (which is the reason why some instructions will
still use binary).

## Features

- Only requires `clang` and `objdump` to be installed in order to compile.
- Auto-detects the appropriate version of `clang` and `objdump` to use.
- Supports cross-compilation.
- Annotated C functions will have their Go stubs auto-generated.
- Automatically formats go assembly using `asmfmt`.

## Annotating C functions

Only functions with gocc annotation will be compiled into Go functions, note that there's some limitations
for parameters - for example when using Go slices, 3 parameters in C have to be used (base pointer, length
and capacity). The tool will check whether the C signature is compatible with the go signature.

An example annotation:

```c
// gocc: simd_fn(input string) int64
int64_t simd_fn(char *input1, uint64_t input1_len) {
  ...
}
```

## Setting up locally

Before you use gocc, you need to install the LLVM toolchain. On Ubuntu, you can do it with the following commands (LLVM 15 was used the most during development of gocc):

```bash
sudo apt install build-essential
bash -c "$(wget -O - https://apt.llvm.org/llvm.sh)"
```

For cross-compilation you will also need to install the appropriate toolchain. For example, for ARM Linux:

```bash
sudo apt install -qy binutils-aarch64-linux-gnu gcc-aarch64-linux-gnu g++-aarch64-linux-gnu
```

On macOS, you'll need to install Xcode, which includes clang and other build essentials:

```bash
xcode-select --install
```

The `example` folder includes matrix multiplication using intrinsics compiled for amd64 and arm64, as well
as an example that only uses clang's auto-vectorization. See the `example/gen.sh` script to see how gocc
can be invoked.

## Limitations

This tool does not support most of the C features, it's not a replacement for C/Go. If you are using this for production code, make sure to test the generated code thoroughly. Also, this is not meant to be general-purpose tool, but rather a tool for solving my own problems of speeding up certain routines.

- Only supports C code that can be compiled by `clang`.
- Does not support C++ code or templates for now.
- Does not support call statements, thus requires you to inline your C functions.
- No dynamic memory allocation.
- Currently limited to 6 arguments per function.

## Resources

The ideas of this are built on top of others who have done similar things, such as

- [c2goasm](https://github.com/minio/c2goasm) and [asm2plan9s](https://github.com/minio/asm2plan9s) by Minio
- [gorse/goat by Gorse](https://github.com/gorse-io/gorse/tree/master/cmd/goat)
- [A Primer on Go Assembly](https://github.com/teh-cmc/go-internals/blob/master/chapter1_assembly_primer/README.md)
- [Go Function in Assembly](https://github.com/golang/go/files/447163/GoFunctionsInAssembly.pdf)
- [Stack frame layout on x86-64](http://eli.thegreenplace.net/2011/09/06/stack-frame-layout-on-x86-64)
- [Compiler Explorer (interactive)](https://go.godbolt.org/)
