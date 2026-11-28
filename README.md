<p align="center">
<img width="250" height="110" src=".github/logo.png" border="0" alt="mhr3/gocc">
<br>
<img src="https://img.shields.io/github/go-mod/go-version/mhr3/gocc" alt="Go Version">
<a href="https://opensource.org/license/apache-2-0/"><img src="https://img.shields.io/badge/License-Apache-blue.svg" alt="License"></a>
</p>

## GOCC: Compile C to Go Assembly

This utility transpiles C code to Go assembly. It uses the LLVM toolchain to compile C code to assembly and machine code and generates Go assembly from it, as well as the corresponding Go stubs. This is useful for certain features such as using intrinsics, which are not supported by the Go ecosystem.
Generated Go assembly uses Plan 9 mnemonics wherever `cmd/asm` reassembles them to the same machine
code, making the output more suitable for tuning or as a starting point. Instructions `cmd/asm` cannot
express (x87, `SETcc`/`CMOVcc`, `MOVZX`/`MOVSX`, some legacy SSE names, ...) are emitted as raw bytes
with the decoded instruction in a comment; the list is verified against the real assembler by
`TestAmd64DenylistRoundTrips`.

## Features

- Only requires `clang` and `objdump` to be installed in order to compile.
- Auto-detects the appropriate version of `clang` and `objdump` to use.
- Supports cross-compilation.
- Annotated C functions will have their Go stubs auto-generated.
- Compiler-emitted helper functions are supported with `--with-internal-functions`.
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

### Slices without their capacity

A slice is three parameters in C. A function that has no use for the capacity can leave it out
for the slices it lists in a `gocc-omit-cap` directive:

```c
// gocc: Dot(a []float32, b []float32) float32
// gocc-omit-cap: a, b
float dot(const float *a, uint64_t a_len, const float *b, uint64_t b_len) {
  ...
}
```

The listed slices are a pointer and a length in C; the others keep their capacity. Nothing
changes for the Go caller: the capacity still has its word in the arguments, the generated
code does not load it. This matters where argument registers are short. amd64 has six for
integers and pointers, which three slices fill without their capacities and two fill with them.

### String and `[]byte` variants of a function

Go code often needs the same routine for strings and for byte slices. A `gocc-variant` directive
adds a second Go signature to a function, without compiling the C code twice:

```c
// gocc: ValidString(data string) bool
// gocc-variant: Valid(data []byte) bool
bool ascii_valid(const unsigned char *data, uint64_t len) {
  ...
}
```

The signatures of a function may differ only in taking a parameter as a `string` or as a `[]byte`.
Either form can be on the `gocc:` line, a function can have several variants, and they may mix the
two, as in `IndexString(haystack string, needle []byte)`. One of the signatures has to take
`[]byte` for every parameter that any of them takes as `[]byte`.

The C function takes a pointer and a length for each string and each slice: a function with
variants has no capacity parameters, for any of its slices. For the example above gocc generates
(comments omitted):

```go
//go:noescape
func Valid(data []byte) bool

func ValidString(data string) bool {
	return validStringInternal(data, 0)
}

//go:noescape
func validStringInternal(data string, _ int) bool
```

```
TEXT ·Valid(SB), NOSPLIT, $0-25
	MOVD data_base+0(FP), R0
	MOVD data_len+8(FP), R1
	...

TEXT ·validStringInternal(SB), NOSPLIT|NOFRAME, $0-25
	JMP ·Valid(SB)
```

The assembly is declared with the `[]byte` form and loads the pointer and the length; the
capacity word of the frame is skipped. The blank `int` gives a string the size of a slice, so
both entry points have the same frame and the string one is a single jump. Calling through it
costs one more stored word per string and that jump.

## Setting up locally

Building gocc requires Go 1.27 or newer: `golang.org/x/arch` needs 1.26 for its AVX/AVX-512
decoder, and the set of instructions emitted as Plan 9 text rather than raw bytes was verified
against the Go 1.27 assembler. The generated assembly itself has no such requirement.

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

## Compiling on another machine

gocc compiles on the machine it runs on. To have every machine use the compiler of one of them, or
to generate code on a machine without clang, run the web server on the machine that has the
toolchain:

```bash
GOCC_TOKEN=some-secret go run github.com/mhr3/gocc/cmd/webserver@latest --listen :8080
```

and point gocc at it from the others, with the `--server` flag or the environment:

```bash
export GOCC_SERVER=http://that-machine:8080 GOCC_TOKEN=some-secret
gocc csrc/ascii_neon.c -p ascii -o ./ -a arm64 -O3
```

The client sends the source together with the files it includes with quotes (`#include "x.h"`),
as far as they exist next to the including file, and writes the generated files as a local run
would. All options work the same; `--with-os-tag` uses the operating system of the client.
`-l` compiles locally even when `GOCC_SERVER` is set.

The server is for machines and networks you trust. A client picks the compiler options and the
source, and with them can make the compiler read and write files as the user running the server.
The token keeps strangers out, it does not make the server safe to offer to them. Without
`--listen` it only accepts connections from its own machine, which is enough behind an SSH tunnel
(`ssh -L 8080:localhost:8080 that-machine`). `GET /health` reports its compiler.

## Limitations

This tool does not support most of the C features, it's not a replacement for C/Go. If you are using this for production code, make sure to test the generated code thoroughly. Also, this is not meant to be general-purpose tool, but rather a tool for solving my own problems of speeding up certain routines.

- Only supports C code that can be compiled by `clang`.
- Does not support C++ code or templates for now.
- Calls to external C libraries are not supported. Internal helper calls (including
  tail calls) require `--with-internal-functions`; recursive helpers and helpers
  reachable from multiple exported functions are rejected, as are helpers that
  take arguments on the stack (more than 8 integer arguments on arm64, 6 on amd64).
  On arm64 a helper is emitted as part of the function that calls it, so that the
  Go runtime can unwind the stack from inside it; a CPU profile shows its time
  under that function. Taking the address of a helper is not supported there.
- No dynamic stack allocation: VLAs and `alloca` are rejected. Go only keeps the
  amd64 stack 8-byte aligned; aligned vector moves to and from the stack are
  rewritten to unaligned ones. A function whose code still depends on the
  alignment of its frame realigns SP inside it: one with a legacy SSE instruction
  that takes its operand from the stack, or one that passes the address of a
  local to code using aligned accesses, such as a helper taking a `__m128i *`.
  That makes it SPWRITE for the Go toolchain, so tracebacks, CPU profiles
  included, end at it. Compile with AVX to avoid the first case. Over-aligned
  locals are only 16-byte aligned on arm64.
- No dynamic memory allocation.
- Currently limited to 6 arguments per function.
- Function-local `static const` tables are supported on arm64; on amd64 clang's
  Intel-syntax assembler cannot re-assemble the dotted `fn.table` symbol it emits,
  so use file-scope `static const` tables instead.

## Resources

The ideas of this are built on top of others who have done similar things, such as

- [c2goasm](https://github.com/minio/c2goasm) and [asm2plan9s](https://github.com/minio/asm2plan9s) by Minio
- [gorse/goat by Gorse](https://github.com/gorse-io/gorse/tree/master/cmd/goat)
- [A Primer on Go Assembly](https://github.com/teh-cmc/go-internals/blob/master/chapter1_assembly_primer/README.md)
- [Go Function in Assembly](https://github.com/golang/go/files/447163/GoFunctionsInAssembly.pdf)
- [Stack frame layout on x86-64](http://eli.thegreenplace.net/2011/09/06/stack-frame-layout-on-x86-64)
- [Compiler Explorer (interactive)](https://go.godbolt.org/)
