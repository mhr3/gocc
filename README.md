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

### String variants of `[]byte` functions

Go code often needs the same routine for byte slices and for strings. A function taking `[]byte`
can get a string variant with a `gocc-alias` directive, without compiling the C code twice:

```c
// gocc: Valid(data []byte) bool
// gocc-alias: ValidString(data string) bool
bool ascii_valid(const unsigned char *data, uint64_t len, uint64_t cap) {
  ...
}
```

The alias must have the signature of the function with one or more `[]byte` parameters replaced
by `string`; a function can have several aliases. For the example above gocc generates (comments
omitted):

```go
//go:noescape
func Valid(data []byte) bool

func ValidString(data string) bool {
	return validStringInternal(data, len(data))
}

//go:noescape
func validStringInternal(data string, _ int) bool
```

```
TEXT ·validStringInternal(SB), NOSPLIT|NOFRAME, $0-25
	JMP ·Valid(SB)
```

The blank `int` gives the string the size of a slice, so both entry points have the same frame
and the alias is a single jump. The C function receives the length of the string as its
capacity. The direction is fixed: the C function has to implement the `[]byte` version,
because a slice cannot be passed in the frame of a function taking a string.

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

## Limitations

This tool does not support most of the C features, it's not a replacement for C/Go. If you are using this for production code, make sure to test the generated code thoroughly. Also, this is not meant to be general-purpose tool, but rather a tool for solving my own problems of speeding up certain routines.

- Only supports C code that can be compiled by `clang`.
- Does not support C++ code or templates for now.
- Calls to external C libraries are not supported. Internal helper calls (including
  tail calls) require `--with-internal-functions`; recursive helpers and helpers
  reachable from multiple exported functions are rejected, as are helpers that
  take arguments on the stack (more than 8 integer arguments on arm64, 6 on amd64).
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
