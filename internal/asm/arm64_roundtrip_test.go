package asm

import (
	"debug/elf"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/arch/arm64/arm64asm"
)

// arm64RoundTripCorpus is GNU-syntax input for clang, one instruction per
// line. It covers every mnemonic in arm64AssemblableMnemonics with the
// operand shapes compiler output uses, the shapes arm64TrustDecoder has to
// reject, and a sample of what arm64asm cannot decode at all (SVE, newer
// NEON extensions), which must stay raw bytes.
var arm64RoundTripCorpus = `
ldrsb x0, [x1, #1]!
ldrsh x0, [x1, #2]!
ldrsb x0, [x1], #1
ldrsh x0, [x1], #2
ldrsw x0, [x1], #4
ldrsw x0, [x1, #4]!
ldrh w0, [x1, #2]!
ldrsh w0, [x1, #2]!
ldrsb w0, [x1, #1]!
ldr s0, [x0], #4
smlal2 v0.4s, v1.8h, v2.8h
ubfx w0, w1, #8, #4
negs x0, x1
rev16 x0, x1
lsl w0, w1, #3
sminp v0.4s, v1.4s, v2.4s
mul w0, w1, w2
sbfiz x0, x1, #8, #4
stp s0, s1, [x0]
madd w0, w1, w2, w3
sxth w0, w1
sub w0, w1, w2
bics x0, x1, x2
ld3r {v0.16b, v1.16b, v2.16b}, [x0], #3
orr w0, w1, w2
ubfiz w0, w1, #8, #4
neg w0, w1
msub w0, w1, w2, w3
umaddl x0, w1, w2, x3
smaddl x0, w1, w2, x3
udiv x0, x1, x2
sdiv w0, w1, w2
adds w0, w1, w2
subs w0, w1, #1
ands w0, w1, w2
orn w0, w1, w2
eon w0, w1, w2
eor w0, w1, w2
bic w0, w1, w2
mvn w0, w1
asr w0, w1, #3
ror w0, w1, #3
sbfx w0, w1, #8, #4
bfxil w0, w1, #8, #4
extr w0, w1, w2, #8
sxtb w0, w1
uxth x0, w1
cls w0, w1
rbit w0, w1
csinc w0, w1, w2, ne
csinv w0, w1, w2, ge
csneg w0, w1, w2, lt
csetm w0, ne
cinc w0, w1, ne
cinv w0, w1, eq
cneg w0, w1, lt
ccmp w0, w1, #0, eq
ccmn w0, #1, #4, ne
movz x0, #0x1234
movz w0, #0x1234
movk w0, #0x5678, lsl #16
movn w0, #0x1234
ldr w0, [x1, #4]!
str w0, [x1, #4]!
ldr s0, [x0, #4]!
str d0, [x0, #8]!
stp w0, w1, [x2, #-8]!
stp w0, w1, [x2], #8
ldp w0, w1, [x2], #8
ldp w0, w1, [x2, #8]!
stlr w0, [x1]
ldar w0, [x1]
ldaxr w0, [x1]
stlxr w2, w0, [x1]
crc32h w0, w1, w2
crc32ch w0, w1, w2
crc32cw w0, w1, w2
fsub s0, s1, s2
fdiv s0, s1, s2
fnmul s0, s1, s2
fmadd s0, s1, s2, s3
fmsub s0, s1, s2, s3
fnmadd s0, s1, s2, s3
fnmsub d0, d1, d2, d3
fsqrt s0, s1
fabs s0, s1
fneg s0, s1
fmax s0, s1, s2
fmin d0, d1, d2
fmaxnm s0, s1, s2
fminnm s0, s1, s2
fcmp s0, s1
fcmpe d0, d1
fccmp s0, s1, #0, eq
fcsel s0, s1, s2, eq
fcvt d0, h1
fcvtzs w0, d1
fcvtzs x0, s1
fcvtzu x0, s1
fcvtzu w0, d1
ucvtf d0, w1
ucvtf s0, x1
frintz s0, s1
frintm s0, s1
frintp d0, d1
frintn s0, s1
frinta s0, s1
frintx s0, s1
frinti s0, s1
cmlt v0.8h, v1.8h, #0
uaddlv h0, v1.16b
sqxtn2 v0.16b, v1.8h
smaxv s0, v1.4s
sminv b0, v1.16b
ld4r {v0.16b, v1.16b, v2.16b, v3.16b}, [x0]
ld3r {v0.16b, v1.16b, v2.16b}, [x0]
ld2r {v0.4s, v1.4s}, [x0], #8
ld3 {v0.16b, v1.16b, v2.16b}, [x0], #48
ld4r {v0.4s, v1.4s, v2.4s, v3.4s}, [x0], #16
ushll2 v0.8h, v1.16b, #1
ushll2 v0.4s, v1.8h, #2
add x0, x1, x2
add x0, x1, #16
add x0, x1, x2, lsl #3
add w0, w1, w2
add x0, x1, w2, uxtb
add x0, x1, w2, uxtw
add x0, sp, #16
adds x0, x1, x2
sub x0, x1, x2
sub x0, x1, #16
sub sp, sp, #32
sub sp, sp, #1, lsl #12
sub x0, x1, x2, asr #2
subs x0, x1, #1
neg x0, x1
negs w0, w1
mul x0, x1, x2
madd x0, x1, x2, x3
msub x0, x1, x2, x3
smull x0, w1, w2
umull x0, w1, w2
umulh x0, x1, x2
smulh x0, x1, x2
sdiv x0, x1, x2
udiv w0, w1, w2
and x0, x1, x2
and x0, x1, #0xff
and w0, w1, #0xfffffffe
ands x0, x1, x2
orr x0, x1, x2
orr x0, x1, #0x10
orn x0, x1, x2
eor x0, x1, x2
eon x0, x1, x2
bic x0, x1, x2
bics w0, w1, w2
tst x0, x1
tst w0, #1
mvn x0, x1
lsl x0, x1, #3
lsl x0, x1, x2
lsr x0, x1, #3
lsr w0, w1, w2
asr x0, x1, #63
ror x0, x1, #7
ror x0, x1, x2
ubfx x0, x1, #8, #4
sbfx x0, x1, #8, #4
ubfiz x0, x1, #8, #4
sbfiz w0, w1, #8, #4
bfi x0, x1, #8, #4
bfxil x0, x1, #8, #4
bfi w0, w1, #8, #4
extr x0, x1, x2, #8
uxtb w0, w1
uxth w0, w1
sxtb x0, w1
sxth x0, w1
sxtw x0, w1
cls x0, x1
clz x0, x1
clz w0, w1
rbit x0, x1
rev x0, x1
rev w0, w1
rev16 w0, w1
rev32 x0, x1
cmp x0, x1
cmp x0, #1
cmp w0, #4
cmp x0, x1, lsl #2
cmp x0, w1, uxtb
cmn x0, #65
cmn w0, w1
ccmp x0, x1, #0, eq
ccmp x0, #1, #4, ne
ccmn x0, x1, #0, eq
csel x0, x1, x2, eq
csel w0, w1, w2, lo
csinc x0, x1, x2, ne
csinv x0, x1, x2, ge
csneg x0, x1, x2, lt
cset x0, eq
cset w0, hs
csetm x0, ne
cinc x0, x1, ne
cinv x0, x1, eq
cneg x0, x1, lt
mov x0, x1
mov w0, w1
mov x0, #42
mov w0, #6
mov x0, #-1
mov x0, #0x10000
mov x0, #0xffff0000
mov x0, xzr
mov x0, sp
mov sp, x0
movz x0, #0x1234, lsl #16
movk x0, #0x5678, lsl #32
movn x0, #0x1234
ldr x0, [x1]
ldr x0, [x1, #16]
ldr x0, [x1, #-16]
ldr x0, [x1, x2]
ldr x0, [x1, x2, lsl #3]
ldr x0, [x1, w2, uxtw #3]
ldr x0, [x1, w2, sxtw]
ldr x0, [x1], #8
ldr x0, [x1, #8]!
ldr w0, [x1, #4]
ldr w0, [x1, x2, lsl #2]
ldr w0, [x1], #4
ldrb w0, [x1]
ldrb w0, [x1, #1]
ldrb w0, [x1, x2]
ldrb w0, [x1], #1
ldrb w0, [x1, #1]!
ldrh w0, [x1, #2]
ldrh w0, [x1, x2, lsl #1]
ldrh w0, [x1], #2
ldrsb x0, [x1]
ldrsb w0, [x1, #1]
ldrsh x0, [x1, #2]
ldrsh w0, [x1]
ldrsw x0, [x1, #4]
ldrsw x0, [x1, x2, lsl #2]
ldur x0, [x1, #-8]
ldur w0, [x1, #-4]
ldurb w0, [x1, #-1]
ldurh w0, [x1, #-2]
ldursb x0, [x1, #-1]
ldursw x0, [x1, #-4]
ldp x0, x1, [sp]
ldp x0, x1, [sp, #16]
ldp x0, x1, [sp], #16
ldp x0, x1, [sp, #-16]!
ldp w0, w1, [x2, #8]
ldpsw x0, x1, [x2]
ldp d0, d1, [sp, #16]
ldp q0, q1, [x0]
ldp q0, q1, [x0, #32]
ldp s0, s1, [x0]
ldnp x0, x1, [x2]
str x0, [x1]
str x0, [x1, #16]
str x0, [x1, x2]
str x0, [x1, x2, lsl #3]
str x0, [x1], #8
str x0, [x1, #8]!
str w0, [x1, #4]
str w0, [x1, x2, lsl #2]
strb w0, [x1]
strb w0, [x1, #1]
strb w0, [x1, x2]
strb w0, [x1], #1
strh w0, [x1, #2]
strh w0, [x1], #2
stur x0, [x1, #-8]
sturb w0, [x1, #-1]
sturh w0, [x1, #-2]
stp x0, x1, [sp]
stp x0, x1, [sp, #16]
stp x0, x1, [sp, #-16]!
stp x0, x1, [sp], #16
stp w0, w1, [x2, #8]
stp d0, d1, [sp, #16]
stp q0, q1, [x0]
stp q0, q1, [x0, #32]
stp xzr, xzr, [sp, #16]
str xzr, [sp]
str wzr, [x0]
ldr s0, [x0]
ldr s0, [x0, #4]
ldr s0, [x0, x1, lsl #2]
ldr d0, [x0]
ldr d0, [x0, #8]
ldr d0, [x0, x1, lsl #3]
ldr d0, [x0], #8
ldr q0, [x0]
ldr q0, [x0, #16]
ldr q0, [x0, x1]
ldr q0, [x0, x1, lsl #4]
ldr q0, [x0], #16
ldr q0, [x0, #16]!
ldr b0, [x0]
ldr h0, [x0]
ldur q0, [x0, #-16]
ldur d0, [x0, #-8]
str s0, [x0]
str d0, [x0, #8]
str q0, [x0]
str q0, [x0, #16]
str q0, [x0, x1]
str q0, [x0], #16
str b0, [x0]
str h0, [x0, #2]
stur q0, [x0, #-16]
ldar x0, [x1]
ldaxr x0, [x1]
stlr x0, [x1]
stlxr w2, x0, [x1]
ldadd x0, x1, [x2]
ldaddal x0, x1, [x2]
swp x0, x1, [x2]
cas x0, x1, [x2]
casal x0, x1, [x2]
ldset x0, x1, [x2]
ldclr x0, x1, [x2]
ldeor x0, x1, [x2]
prfm pldl1keep, [x0]
prfm pldl1strm, [x0, #64]
dmb ish
dsb sy
isb
nop
brk #1
hlt #0
ret
blr x0
br x0
crc32b w0, w1, w2
crc32w w0, w1, w2
crc32x w0, w1, x2
crc32cb w0, w1, w2
crc32cx w0, w1, x2
mrs x0, tpidr_el0
mrs x0, fpcr
msr fpcr, x0
mrs x0, nzcv
fmov d0, x0
fmov x0, d0
fmov s0, w0
fmov w0, s0
fmov d0, d1
fmov s0, s1
fmov d0, #1.0
fmov s0, #0.5
fmov d0, xzr
fmov v0.d[1], x0
fmov x0, v0.d[1]
fadd d0, d1, d2
fadd s0, s1, s2
fsub d0, d1, d2
fmul d0, d1, d2
fmul s0, s1, s2
fdiv d0, d1, d2
fnmul d0, d1, d2
fmadd d0, d1, d2, d3
fmsub d0, d1, d2, d3
fnmadd d0, d1, d2, d3
fnmsub s0, s1, s2, s3
fsqrt d0, d1
fabs d0, d1
fneg d0, d1
fmax d0, d1, d2
fmin s0, s1, s2
fmaxnm d0, d1, d2
fminnm d0, d1, d2
fcmp d0, d1
fcmp d0, #0.0
fcmpe s0, s1
fccmp d0, d1, #0, eq
fcsel d0, d1, d2, eq
fcvt d0, s1
fcvt s0, d1
fcvt h0, s1
fcvt s0, h1
fcvtzs x0, d1
fcvtzs w0, s1
fcvtzs w0, d1
fcvtzu x0, d1
fcvtzu w0, s1
fcvtas x0, d1
fcvtms x0, d1
fcvtps x0, d1
fcvtns w0, s1
scvtf d0, x1
scvtf s0, w1
scvtf d0, w1
scvtf s0, x1
ucvtf d0, x1
ucvtf s0, w1
frintz d0, d1
frintm d0, d1
frintp s0, s1
frintn d0, d1
frinta d0, d1
frintx d0, d1
frinti d0, d1
movi v0.2d, #0
movi v0.16b, #1
movi v0.16b, #0
movi v31.16b, #0xff
movi v0.8b, #0x80
movi v7.8b, #15
movi v0.4h, #1
movi v0.2s, #1, lsl #24
movi v0.4s, #1, msl #8
movi d0, #0xff00ff00ff00ff00
movi v0.4s, #1
movi v0.4s, #1, lsl #8
movi v0.8h, #1
movi v0.2d, #0xffffffffffffffff
movi d0, #0
mvni v0.4s, #1
fmov v0.4s, #1.0
fmov v0.2d, #1.0
dup v0.16b, w0
dup v0.4s, w0
dup v0.2d, x0
dup v0.16b, v1.b[3]
dup v0.4s, v1.s[1]
dup v0.2d, v1.d[1]
dup s0, v1.s[1]
dup d0, v1.d[1]
mov v0.16b, v1.16b
mov v0.s[1], w0
mov v0.d[1], x0
mov v0.b[3], w0
mov v0.h[3], w0
mov w0, v0.s[1]
mov x0, v0.d[1]
mov v0.s[1], v1.s[2]
mov v0.d[0], v1.d[1]
umov w0, v0.b[3]
umov w0, v0.h[3]
umov w0, v0.s[1]
umov x0, v0.d[1]
smov w0, v0.b[3]
smov w0, v0.h[3]
smov x0, v0.b[3]
smov x0, v0.s[1]
ins v0.s[1], w0
ins v0.d[1], v1.d[0]
ld1 {v0.16b}, [x0]
ld1 {v0.16b}, [x0], #16
ld1 {v0.16b}, [x0], x1
ld1 {v0.4s}, [x0]
ld1 {v0.2d}, [x0], #16
ld1 {v0.8h}, [x0]
ld1 {v0.16b, v1.16b}, [x0]
ld1 {v0.16b, v1.16b}, [x0], #32
ld1 {v0.16b, v1.16b, v2.16b, v3.16b}, [x0], #64
ld1 {v0.4s, v1.4s}, [x0], x1
ld1 {v0.8b}, [x0]
ld1 {v0.b}[3], [x0]
ld1 {v0.s}[1], [x0]
ld1 {v0.d}[1], [x0]
ld1 {v0.d}[1], [x0], #8
ld1r {v0.16b}, [x0]
ld1r {v0.4s}, [x0]
ld1r {v0.2d}, [x0]
ld1r {v0.8h}, [x0]
ld1r {v0.16b}, [x0], #1
ld1r {v0.4s}, [x0], #4
ld1r {v0.8b}, [x0]
ld2 {v0.16b, v1.16b}, [x0]
ld2 {v0.4s, v1.4s}, [x0], #32
ld2 {v0.b, v1.b}[3], [x0]
ld2r {v0.16b, v1.16b}, [x0]
ld3 {v0.16b, v1.16b, v2.16b}, [x0]
ld4 {v0.16b, v1.16b, v2.16b, v3.16b}, [x0]
ld4 {v0.4s, v1.4s, v2.4s, v3.4s}, [x0], #64
st1 {v0.16b}, [x0]
st1 {v0.16b}, [x0], #16
st1 {v0.16b}, [x0], x1
st1 {v0.4s}, [x0]
st1 {v0.2d}, [x0], #16
st1 {v0.16b, v1.16b}, [x0]
st1 {v0.16b, v1.16b}, [x0], #32
st1 {v0.16b, v1.16b, v2.16b, v3.16b}, [x0], #64
st1 {v0.8b}, [x0]
st1 {v0.b}[3], [x0]
st1 {v0.s}[1], [x0]
st1 {v0.d}[1], [x0]
st1 {v0.d}[1], [x0], #8
st2 {v0.16b, v1.16b}, [x0]
st2 {v0.4s, v1.4s}, [x0], #32
st3 {v0.16b, v1.16b, v2.16b}, [x0]
st4 {v0.16b, v1.16b, v2.16b, v3.16b}, [x0]
st4 {v0.4s, v1.4s, v2.4s, v3.4s}, [x0], #64
add v0.16b, v1.16b, v2.16b
add v0.8h, v1.8h, v2.8h
add v0.4s, v1.4s, v2.4s
add v0.2d, v1.2d, v2.2d
add v0.8b, v1.8b, v2.8b
add d0, d1, d2
sub v0.16b, v1.16b, v2.16b
sub v0.4s, v1.4s, v2.4s
sub v0.2d, v1.2d, v2.2d
mul v0.16b, v1.16b, v2.16b
mul v0.8h, v1.8h, v2.8h
mul v0.4s, v1.4s, v2.4s
mul v0.4s, v1.4s, v2.s[1]
mla v0.4s, v1.4s, v2.4s
mls v0.4s, v1.4s, v2.4s
mla v0.4s, v1.4s, v2.s[1]
umull v0.8h, v1.8b, v2.8b
umull2 v0.8h, v1.16b, v2.16b
smull v0.4s, v1.4h, v2.4h
smull2 v0.2d, v1.4s, v2.4s
umlal v0.8h, v1.8b, v2.8b
umlal2 v0.4s, v1.8h, v2.8h
smlal v0.4s, v1.4h, v2.4h
pmull v0.8h, v1.8b, v2.8b
pmull2 v0.8h, v1.16b, v2.16b
pmull v0.1q, v1.1d, v2.1d
sdot v0.4s, v1.16b, v2.16b
udot v0.4s, v1.16b, v2.16b
udot v0.4s, v1.16b, v2.4b[0]
smmla v0.4s, v1.16b, v2.16b
ummla v0.4s, v1.16b, v2.16b
usdot v0.4s, v1.16b, v2.16b
and v0.16b, v1.16b, v2.16b
and v0.8b, v1.8b, v2.8b
orr v0.16b, v1.16b, v2.16b
orr v0.4s, #1
orr v0.8h, #1, lsl #8
orn v0.16b, v1.16b, v2.16b
eor v0.16b, v1.16b, v2.16b
bic v0.16b, v1.16b, v2.16b
bic v0.4s, #1
bic v0.8h, #1, lsl #8
bsl v0.16b, v1.16b, v2.16b
bit v0.16b, v1.16b, v2.16b
bif v0.16b, v1.16b, v2.16b
not v0.16b, v1.16b
mvn v0.16b, v1.16b
neg v0.4s, v1.4s
neg v0.16b, v1.16b
abs v0.4s, v1.4s
abs v0.16b, v1.16b
cnt v0.16b, v1.16b
cnt v0.8b, v1.8b
clz v0.4s, v1.4s
cls v0.16b, v1.16b
rbit v0.16b, v1.16b
rev16 v0.16b, v1.16b
rev32 v0.16b, v1.16b
rev32 v0.8h, v1.8h
rev64 v0.4s, v1.4s
rev64 v0.16b, v1.16b
ext v0.16b, v1.16b, v2.16b, #8
ext v0.16b, v1.16b, v2.16b, #1
ext v0.8b, v1.8b, v2.8b, #4
tbl v0.16b, {v1.16b}, v2.16b
tbl v0.16b, {v1.16b, v2.16b}, v3.16b
tbl v0.8b, {v1.16b}, v2.8b
tbl v0.16b, {v1.16b, v2.16b, v3.16b, v4.16b}, v5.16b
tbx v0.16b, {v1.16b}, v2.16b
tbx v0.16b, {v1.16b, v2.16b}, v3.16b
shl v0.16b, v1.16b, #1
shl v0.8h, v1.8h, #4
shl v0.4s, v1.4s, #4
shl v0.2d, v1.2d, #4
shl d0, d1, #4
ushr v0.16b, v1.16b, #4
ushr v0.8h, v1.8h, #4
ushr v0.4s, v1.4s, #4
ushr v0.2d, v1.2d, #4
ushr d0, d1, #4
sshr v0.16b, v1.16b, #7
sshr v0.8h, v1.8h, #15
sshr v0.4s, v1.4s, #31
sshr v0.2d, v1.2d, #63
sshr d0, d1, #63
sshl v0.16b, v1.16b, v2.16b
sshl v0.4s, v1.4s, v2.4s
ushl v0.16b, v1.16b, v2.16b
ushl v0.2d, v1.2d, v2.2d
ushl d0, d1, d2
srshl v0.4s, v1.4s, v2.4s
urshl v0.4s, v1.4s, v2.4s
sli v0.16b, v1.16b, #4
sri v0.16b, v1.16b, #4
sli v0.4s, v1.4s, #4
usra v0.4s, v1.4s, #4
ssra v0.4s, v1.4s, #4
usra v0.16b, v1.16b, #7
ursra v0.4s, v1.4s, #4
srsra v0.4s, v1.4s, #4
urshr v0.4s, v1.4s, #4
srshr v0.4s, v1.4s, #4
shll v0.8h, v1.8b, #8
shll2 v0.8h, v1.16b, #8
ushll v0.8h, v1.8b, #0
ushll v0.8h, v1.8b, #1
ushll2 v0.8h, v1.16b, #0
ushll v0.4s, v1.4h, #0
ushll2 v0.4s, v1.8h, #0
ushll v0.2d, v1.2s, #0
sshll v0.8h, v1.8b, #0
sshll2 v0.8h, v1.16b, #0
sshll v0.4s, v1.4h, #0
sshll2 v0.2d, v1.4s, #0
uxtl v0.8h, v1.8b
uxtl2 v0.8h, v1.16b
sxtl v0.4s, v1.4h
sxtl2 v0.2d, v1.4s
shrn v0.8b, v1.8h, #8
shrn2 v0.16b, v1.8h, #8
shrn v0.4h, v1.4s, #16
rshrn v0.8b, v1.8h, #8
uqshrn v0.8b, v1.8h, #8
sqshrn v0.8b, v1.8h, #8
sqshrun v0.8b, v1.8h, #8
uqrshrn v0.8b, v1.8h, #8
uqshrn2 v0.16b, v1.8h, #8
xtn v0.8b, v1.8h
xtn2 v0.16b, v1.8h
xtn v0.4h, v1.4s
xtn2 v0.8h, v1.4s
xtn v0.2s, v1.2d
uqxtn v0.8b, v1.8h
uqxtn2 v0.16b, v1.8h
sqxtn v0.8b, v1.8h
sqxtun v0.8b, v1.8h
sqxtun2 v0.16b, v1.8h
cmeq v0.16b, v1.16b, v2.16b
cmeq v0.16b, v1.16b, #0
cmeq v0.4s, v1.4s, v2.4s
cmeq v0.4s, v1.4s, #0
cmeq v0.2d, v1.2d, v2.2d
cmeq v0.8h, v1.8h, #0
cmeq v0.8b, v1.8b, v2.8b
cmeq d0, d1, d2
cmhi v0.16b, v1.16b, v2.16b
cmhi v0.4s, v1.4s, v2.4s
cmhs v0.16b, v1.16b, v2.16b
cmhs v0.4s, v1.4s, v2.4s
cmgt v0.16b, v1.16b, v2.16b
cmgt v0.16b, v1.16b, #0
cmgt v0.4s, v1.4s, v2.4s
cmge v0.16b, v1.16b, v2.16b
cmge v0.4s, v1.4s, #0
cmlt v0.16b, v1.16b, #0
cmlt v0.4s, v1.4s, #0
cmle v0.16b, v1.16b, #0
cmtst v0.16b, v1.16b, v2.16b
cmtst v0.4s, v1.4s, v2.4s
cmtst v0.2d, v1.2d, v2.2d
umax v0.16b, v1.16b, v2.16b
umax v0.4s, v1.4s, v2.4s
umin v0.16b, v1.16b, v2.16b
umin v0.8h, v1.8h, v2.8h
smax v0.16b, v1.16b, v2.16b
smax v0.4s, v1.4s, v2.4s
smin v0.16b, v1.16b, v2.16b
smin v0.4s, v1.4s, v2.4s
umaxv b0, v1.16b
umaxv h0, v1.8h
umaxv s0, v1.4s
uminv b0, v1.16b
uminv s0, v1.4s
smaxv b0, v1.16b
sminv s0, v1.4s
umaxp v0.16b, v1.16b, v2.16b
uminp v0.16b, v1.16b, v2.16b
smaxp v0.4s, v1.4s, v2.4s
addv b0, v1.16b
addv h0, v1.8h
addv s0, v1.4s
addp v0.16b, v1.16b, v2.16b
addp v0.4s, v1.4s, v2.4s
addp v0.2d, v1.2d, v2.2d
addp d0, v1.2d
uaddlv h0, v1.16b
uaddlv s0, v1.8h
uaddlv d0, v1.4s
saddlv h0, v1.16b
uaddlp v0.8h, v1.16b
uaddlp v0.4s, v1.8h
uaddlp v0.2d, v1.4s
sadalp v0.8h, v1.16b
uadalp v0.8h, v1.16b
uadalp v0.4s, v1.8h
uaddl v0.8h, v1.8b, v2.8b
uaddl2 v0.8h, v1.16b, v2.16b
saddl v0.4s, v1.4h, v2.4h
uaddw v0.8h, v1.8h, v2.8b
uaddw2 v0.8h, v1.8h, v2.16b
usubl v0.8h, v1.8b, v2.8b
uabd v0.16b, v1.16b, v2.16b
sabd v0.4s, v1.4s, v2.4s
uabdl v0.8h, v1.8b, v2.8b
uaba v0.16b, v1.16b, v2.16b
uabal v0.8h, v1.8b, v2.8b
uhadd v0.16b, v1.16b, v2.16b
shadd v0.4s, v1.4s, v2.4s
urhadd v0.16b, v1.16b, v2.16b
uhsub v0.16b, v1.16b, v2.16b
uqadd v0.16b, v1.16b, v2.16b
sqadd v0.4s, v1.4s, v2.4s
uqsub v0.16b, v1.16b, v2.16b
sqsub v0.8h, v1.8h, v2.8h
sqdmulh v0.4s, v1.4s, v2.4s
sqrdmulh v0.8h, v1.8h, v2.8h
sqrdmulh v0.4s, v1.4s, v2.s[1]
sqrdmlah v0.4s, v1.4s, v2.4s
zip1 v0.16b, v1.16b, v2.16b
zip2 v0.16b, v1.16b, v2.16b
zip1 v0.8h, v1.8h, v2.8h
zip2 v0.4s, v1.4s, v2.4s
zip1 v0.2d, v1.2d, v2.2d
zip1 v0.8b, v1.8b, v2.8b
uzp1 v0.16b, v1.16b, v2.16b
uzp2 v0.16b, v1.16b, v2.16b
uzp1 v0.8h, v1.8h, v2.8h
uzp2 v0.4s, v1.4s, v2.4s
uzp1 v0.2d, v1.2d, v2.2d
trn1 v0.16b, v1.16b, v2.16b
trn2 v0.16b, v1.16b, v2.16b
trn1 v0.4s, v1.4s, v2.4s
trn2 v0.2d, v1.2d, v2.2d
fadd v0.4s, v1.4s, v2.4s
fadd v0.2d, v1.2d, v2.2d
fadd v0.2s, v1.2s, v2.2s
fsub v0.4s, v1.4s, v2.4s
fsub v0.2d, v1.2d, v2.2d
fmul v0.4s, v1.4s, v2.4s
fmul v0.2d, v1.2d, v2.2d
fmul v0.4s, v1.4s, v2.s[1]
fmul v0.2d, v1.2d, v2.d[1]
fmul s0, s1, v2.s[1]
fmul d0, d1, v2.d[1]
fdiv v0.4s, v1.4s, v2.4s
fdiv v0.2d, v1.2d, v2.2d
fmla v0.4s, v1.4s, v2.4s
fmla v0.2d, v1.2d, v2.2d
fmla v0.4s, v1.4s, v2.s[0]
fmla v0.4s, v1.4s, v2.s[3]
fmla v0.2d, v1.2d, v2.d[1]
fmla s0, s1, v2.s[1]
fmls v0.4s, v1.4s, v2.4s
fmls v0.4s, v1.4s, v2.s[1]
fneg v0.4s, v1.4s
fneg v0.2d, v1.2d
fabs v0.4s, v1.4s
fsqrt v0.4s, v1.4s
fsqrt v0.2d, v1.2d
fmax v0.4s, v1.4s, v2.4s
fmin v0.4s, v1.4s, v2.4s
fmaxnm v0.4s, v1.4s, v2.4s
fminnm v0.2d, v1.2d, v2.2d
fmaxv s0, v1.4s
fminv s0, v1.4s
fmaxnmv s0, v1.4s
faddp v0.4s, v1.4s, v2.4s
faddp v0.2d, v1.2d, v2.2d
faddp s0, v1.2s
faddp d0, v1.2d
fmaxp v0.4s, v1.4s, v2.4s
fcmeq v0.4s, v1.4s, v2.4s
fcmeq v0.4s, v1.4s, #0.0
fcmgt v0.4s, v1.4s, v2.4s
fcmgt v0.2d, v1.2d, #0.0
fcmge v0.4s, v1.4s, v2.4s
fcmlt v0.4s, v1.4s, #0.0
fcmle v0.4s, v1.4s, #0.0
facgt v0.4s, v1.4s, v2.4s
facge v0.4s, v1.4s, v2.4s
fabd v0.4s, v1.4s, v2.4s
frecpe v0.4s, v1.4s
frecps v0.4s, v1.4s, v2.4s
frsqrte v0.4s, v1.4s
frsqrts v0.4s, v1.4s, v2.4s
frecpe s0, s1
frsqrte d0, d1
frintm v0.4s, v1.4s
frintz v0.4s, v1.4s
frintp v0.2d, v1.2d
frintn v0.4s, v1.4s
frinta v0.4s, v1.4s
frintx v0.4s, v1.4s
frinti v0.4s, v1.4s
fcvtzs v0.4s, v1.4s
fcvtzs v0.2d, v1.2d
fcvtzu v0.4s, v1.4s
fcvtzs v0.4s, v1.4s, #16
fcvtns v0.4s, v1.4s
fcvtms v0.4s, v1.4s
fcvtps v0.4s, v1.4s
fcvtas v0.4s, v1.4s
fcvtau v0.4s, v1.4s
scvtf v0.4s, v1.4s
scvtf v0.2d, v1.2d
ucvtf v0.4s, v1.4s
ucvtf v0.2d, v1.2d
scvtf v0.4s, v1.4s, #16
fcvtl v0.4s, v1.4h
fcvtl2 v0.4s, v1.8h
fcvtl v0.2d, v1.2s
fcvtl2 v0.2d, v1.4s
fcvtn v0.4h, v1.4s
fcvtn2 v0.8h, v1.4s
fcvtn v0.2s, v1.2d
fcvtn2 v0.4s, v1.2d
fcvtxn v0.2s, v1.2d
bfcvtn v0.4h, v1.4s
bfdot v0.4s, v1.8h, v2.8h
bfmmla v0.4s, v1.8h, v2.8h
fmlal v0.4s, v1.4h, v2.4h
fmlal2 v0.4s, v1.4h, v2.4h
fadd v0.8h, v1.8h, v2.8h
fmul v0.8h, v1.8h, v2.8h
fmla v0.8h, v1.8h, v2.8h
fcvt h0, d1
aese v0.16b, v1.16b
aesd v0.16b, v1.16b
aesmc v0.16b, v1.16b
aesimc v0.16b, v1.16b
sha1c q0, s1, v2.4s
sha1h s0, s1
sha1m q0, s1, v2.4s
sha1p q0, s1, v2.4s
sha1su0 v0.4s, v1.4s, v2.4s
sha1su1 v0.4s, v1.4s
sha256h q0, q1, v2.4s
sha256h2 q0, q1, v2.4s
sha256su0 v0.4s, v1.4s
sha256su1 v0.4s, v1.4s, v2.4s
sha512h q0, q1, v2.2d
sha512h2 q0, q1, v2.2d
sha512su0 v0.2d, v1.2d
sha512su1 v0.2d, v1.2d, v2.2d
eor3 v0.16b, v1.16b, v2.16b, v3.16b
bcax v0.16b, v1.16b, v2.16b, v3.16b
rax1 v0.2d, v1.2d, v2.2d
xar v0.2d, v1.2d, v2.2d, #8
sm3partw1 v0.4s, v1.4s, v2.4s
sm4e v0.4s, v1.4s
fcadd v0.4s, v1.4s, v2.4s, #90
fcmla v0.4s, v1.4s, v2.4s, #90
ptrue p0.b
ptrue p0.h
ptrue p0.s
ptrue p0.d
ptrue p0.b, vl16
pfalse p0.b
ptest p0, p1.b
whilelo p0.b, x0, x1
whilelo p0.s, w0, w1
whilelt p0.b, x0, x1
whilele p0.b, x0, x1
whilels p0.d, x0, x1
rdvl x0, #1
rdvl x0, #-1
cntb x0
cnth x0
cntw x0
cntd x0
cntb x0, all, mul #2
incb x0
incd x0, all, mul #2
incw x0
cntp x0, p0, p1.b
addvl x0, x1, #1
addpl x0, x1, #1
ld1b {z0.b}, p0/z, [x0]
ld1b {z0.b}, p0/z, [x0, x1]
ld1b {z0.b}, p0/z, [x0, #1, mul vl]
ld1b {z0.h}, p0/z, [x0, x1]
ld1b {z0.s}, p0/z, [x0]
ld1b {z0.d}, p0/z, [x0, x1]
ld1sb {z0.h}, p0/z, [x0, x1]
ld1h {z0.h}, p0/z, [x0]
ld1h {z0.h}, p0/z, [x0, x1, lsl #1]
ld1h {z0.s}, p0/z, [x0, x1, lsl #1]
ld1w {z0.s}, p0/z, [x0]
ld1w {z0.s}, p0/z, [x0, x1, lsl #2]
ld1w {z0.d}, p0/z, [x0]
ld1d {z0.d}, p0/z, [x0]
ld1d {z0.d}, p0/z, [x0, x1, lsl #3]
ld1d {z0.d}, p0/z, [x0, #-1, mul vl]
ld1rb {z0.b}, p0/z, [x0]
ld1rw {z0.s}, p0/z, [x0]
ld1rd {z0.d}, p0/z, [x0]
ld1rqb {z0.b}, p0/z, [x0]
ld1rqw {z0.s}, p0/z, [x0]
ld1w {z0.s}, p0/z, [z1.s, #4]
ld1d {z0.d}, p0/z, [x0, z1.d, lsl #3]
ld1w {z0.s}, p0/z, [x0, z1.s, uxtw #2]
ldff1b {z0.b}, p0/z, [x0, x1]
ldnf1b {z0.b}, p0/z, [x0]
ldnt1b {z0.b}, p0/z, [x0, x1]
ld2b {z0.b, z1.b}, p0/z, [x0]
ld2w {z0.s, z1.s}, p0/z, [x0, x1, lsl #2]
ld4b {z0.b, z1.b, z2.b, z3.b}, p0/z, [x0]
ldr z0, [x0]
ldr z0, [x0, #1, mul vl]
ldr p0, [x0]
st1b {z0.b}, p0, [x0]
st1b {z0.b}, p0, [x0, x1]
st1b {z0.h}, p0, [x0, x1]
st1b {z0.s}, p0, [x0]
st1b {z0.b}, p0, [x0, #1, mul vl]
st1h {z0.h}, p0, [x0, x1, lsl #1]
st1h {z0.s}, p0, [x0]
st1w {z0.s}, p0, [x0]
st1w {z0.s}, p0, [x0, x1, lsl #2]
st1d {z0.d}, p0, [x0]
st1d {z0.d}, p0, [x0, x1, lsl #3]
st1w {z0.s}, p0, [x0, z1.s, uxtw #2]
st1d {z0.d}, p0, [x0, z1.d, lsl #3]
stnt1b {z0.b}, p0, [x0, x1]
st2b {z0.b, z1.b}, p0, [x0]
str z0, [x0]
str z0, [x0, #1, mul vl]
str p0, [x0]
mov z0.b, #0
mov z0.b, #1
mov z0.s, #1
mov z0.d, #-1
mov z0.b, w0
mov z0.s, w0
mov z0.d, x0
mov z0.s, p0/m, z1.s
dup z0.b, w0
dup z0.b, #0
dup z0.b, z1.b[3]
dupm z0.s, #0x1
sel z0.b, p0, z1.b, z2.b
sel p0.b, p1, p2.b, p3.b
cpy z0.b, p0/m, w0
cpy z0.b, p0/z, #1
fmov z0.s, #1.0
fmov z0.d, #0.0
fdup z0.s, #1.0
index z0.b, #0, #1
index z0.s, w0, #1
index z0.d, x0, x1
add z0.b, z1.b, z2.b
add z0.s, z1.s, z2.s
add z0.d, z1.d, z2.d
add z0.b, z0.b, #1
add z0.s, p0/m, z0.s, z1.s
sub z0.b, z1.b, z2.b
sub z0.s, p0/m, z0.s, z1.s
sub z0.b, z0.b, #1
subr z0.s, p0/m, z0.s, z1.s
subr z0.b, z0.b, #1
mul z0.b, p0/m, z0.b, z1.b
mul z0.s, p0/m, z0.s, z1.s
mul z0.s, z0.s, #3
mul z0.s, z1.s, z2.s
mad z0.s, p0/m, z1.s, z2.s
mla z0.s, p0/m, z1.s, z2.s
mls z0.s, p0/m, z1.s, z2.s
msb z0.s, p0/m, z1.s, z2.s
smulh z0.s, p0/m, z0.s, z1.s
umulh z0.b, p0/m, z0.b, z1.b
sdiv z0.s, p0/m, z0.s, z1.s
udiv z0.d, p0/m, z0.d, z1.d
smullb z0.h, z1.b, z2.b
smullt z0.h, z1.b, z2.b
umullb z0.s, z1.h, z2.h
umullt z0.s, z1.h, z2.h
umlalb z0.s, z1.h, z2.h
umlalt z0.s, z1.h, z2.h
sdot z0.s, z1.b, z2.b
udot z0.s, z1.b, z2.b
udot z0.d, z1.h, z2.h
sdot z0.s, z1.b, z2.b[1]
usdot z0.s, z1.b, z2.b
smmla z0.s, z1.b, z2.b
and z0.d, z1.d, z2.d
and z0.b, p0/m, z0.b, z1.b
and z0.b, z0.b, #0xf
and p0.b, p1/z, p2.b, p3.b
ands p0.b, p1/z, p2.b, p3.b
orr z0.d, z1.d, z2.d
orr z0.b, z0.b, #0xf
orr z0.b, p0/m, z0.b, z1.b
orr p0.b, p1/z, p2.b, p3.b
orn p0.b, p1/z, p2.b, p3.b
eor z0.d, z1.d, z2.d
eor z0.b, z0.b, #0xf
eor z0.b, p0/m, z0.b, z1.b
eor p0.b, p1/z, p2.b, p3.b
eors p0.b, p1/z, p2.b, p3.b
bic z0.d, z1.d, z2.d
bic z0.b, p0/m, z0.b, z1.b
bic p0.b, p1/z, p2.b, p3.b
bics p0.b, p1/z, p2.b, p3.b
nand p0.b, p1/z, p2.b, p3.b
nor p0.b, p1/z, p2.b, p3.b
not z0.b, p0/m, z1.b
not p0.b, p1/z, p2.b
nots p0.b, p1/z, p2.b
bsl z0.d, z0.d, z1.d, z2.d
bsl1n z0.d, z0.d, z1.d, z2.d
nbsl z0.d, z0.d, z1.d, z2.d
eor3 z0.d, z0.d, z1.d, z2.d
bcax z0.d, z0.d, z1.d, z2.d
xar z0.b, z0.b, z1.b, #1
neg z0.s, p0/m, z1.s
abs z0.s, p0/m, z1.s
cnt z0.b, p0/m, z1.b
cnt z0.s, p0/m, z1.s
clz z0.s, p0/m, z1.s
cls z0.b, p0/m, z1.b
rbit z0.b, p0/m, z1.b
revb z0.h, p0/m, z1.h
revh z0.s, p0/m, z1.s
revw z0.d, p0/m, z1.d
rev z0.b, z1.b
rev p0.b, p1.b
lsl z0.b, z1.b, #1
lsl z0.s, z1.s, #4
lsl z0.s, p0/m, z0.s, z1.s
lsl z0.s, p0/m, z0.s, #4
lsl z0.b, z1.b, z2.d
lsr z0.b, z1.b, #1
lsr z0.s, z1.s, #4
lsr z0.s, p0/m, z0.s, z1.s
lsr z0.s, p0/m, z0.s, #4
asr z0.s, z1.s, #4
asr z0.s, p0/m, z0.s, z1.s
asr z0.b, p0/m, z0.b, #7
asrd z0.s, p0/m, z0.s, #1
lslr z0.s, p0/m, z0.s, z1.s
sli z0.b, z1.b, #4
sri z0.b, z1.b, #4
usra z0.s, z1.s, #4
ssra z0.s, z1.s, #4
shrnb z0.b, z1.h, #8
shrnt z0.b, z1.h, #8
rshrnb z0.b, z1.h, #8
uqshrnb z0.b, z1.h, #8
uqshrnt z0.b, z1.h, #8
sqxtnb z0.b, z1.h
sqxtnt z0.b, z1.h
uqxtnb z0.b, z1.h
uqxtnt z0.b, z1.h
sqxtunb z0.b, z1.h
sqxtunt z0.b, z1.h
sunpklo z0.h, z1.b
sunpkhi z0.h, z1.b
uunpklo z0.h, z1.b
uunpkhi z0.h, z1.b
uunpklo z0.s, z1.h
uunpkhi z0.d, z1.s
punpklo p0.h, p1.b
punpkhi p0.h, p1.b
sxtb z0.h, p0/m, z1.h
sxtb z0.s, p0/m, z1.s
sxth z0.s, p0/m, z1.s
sxtw z0.d, p0/m, z1.d
uxtb z0.h, p0/m, z1.h
uxth z0.s, p0/m, z1.s
uxtw z0.d, p0/m, z1.d
cmpeq p0.b, p1/z, z0.b, z1.b
cmpeq p0.b, p1/z, z0.b, #0
cmpeq p0.s, p1/z, z0.s, #1
cmpeq p0.b, p1/z, z0.b, z1.d
cmpne p0.b, p1/z, z0.b, z1.b
cmpne p0.b, p1/z, z0.b, #0
cmpne p0.s, p1/z, z0.s, z1.d
cmpgt p0.s, p1/z, z0.s, z1.s
cmpgt p0.b, p1/z, z0.b, #0
cmpge p0.s, p1/z, z0.s, z1.s
cmpge p0.b, p1/z, z0.b, #0
cmplt p0.s, p1/z, z0.s, #0
cmple p0.b, p1/z, z0.b, #0
cmphi p0.b, p1/z, z0.b, z1.b
cmphi p0.b, p1/z, z0.b, #1
cmphs p0.b, p1/z, z0.b, z1.b
cmphs p0.b, p1/z, z0.b, #1
cmplo p0.b, p1/z, z0.b, #1
cmpls p0.b, p1/z, z0.b, #1
cmplt p0.b, p1/z, z0.b, z1.d
cmphi p0.b, p1/z, z0.b, z1.d
cmphs p0.b, p1/z, z0.b, z1.d
cmplo p0.b, p1/z, z0.b, z1.d
cmpls p0.b, p1/z, z0.b, z1.d
match p0.b, p1/z, z0.b, z1.b
nmatch p0.b, p1/z, z0.b, z1.b
histseg z0.b, z1.b, z2.b
umax z0.b, p0/m, z0.b, z1.b
umax z0.s, p0/m, z0.s, z1.s
umax z0.b, z0.b, #1
umin z0.b, p0/m, z0.b, z1.b
umin z0.b, z0.b, #1
smax z0.s, p0/m, z0.s, z1.s
smax z0.s, z0.s, #1
smin z0.s, p0/m, z0.s, z1.s
smin z0.s, z0.s, #-1
umaxv b0, p0, z1.b
umaxv s0, p0, z1.s
uminv b0, p0, z1.b
smaxv s0, p0, z1.s
sminv s0, p0, z1.s
uaddv d0, p0, z1.b
uaddv d0, p0, z1.s
saddv d0, p0, z1.b
andv b0, p0, z1.b
orv b0, p0, z1.b
eorv s0, p0, z1.s
uabd z0.b, p0/m, z0.b, z1.b
sabd z0.s, p0/m, z0.s, z1.s
uaddlb z0.h, z1.b, z2.b
uaddlt z0.h, z1.b, z2.b
saddlb z0.s, z1.h, z2.h
uaddwb z0.h, z1.h, z2.b
uaddwt z0.h, z1.h, z2.b
uabalb z0.h, z1.b, z2.b
uabalt z0.h, z1.b, z2.b
uadalp z0.h, p0/m, z1.b
sadalp z0.s, p0/m, z1.h
uhadd z0.b, p0/m, z0.b, z1.b
urhadd z0.b, p0/m, z0.b, z1.b
uqadd z0.b, z1.b, z2.b
uqadd z0.b, z0.b, #1
sqadd z0.s, z1.s, z2.s
uqsub z0.b, z1.b, z2.b
sqsub z0.s, z1.s, z2.s
sqdmulh z0.s, z1.s, z2.s
sqrdmulh z0.h, z1.h, z2.h
zip1 z0.b, z1.b, z2.b
zip2 z0.b, z1.b, z2.b
zip1 z0.s, z1.s, z2.s
zip2 z0.d, z1.d, z2.d
zip1 p0.b, p1.b, p2.b
zip2 p0.h, p1.h, p2.h
uzp1 z0.b, z1.b, z2.b
uzp2 z0.b, z1.b, z2.b
uzp1 z0.s, z1.s, z2.s
uzp2 z0.d, z1.d, z2.d
uzp1 p0.b, p1.b, p2.b
uzp2 p0.b, p1.b, p2.b
trn1 z0.b, z1.b, z2.b
trn2 z0.s, z1.s, z2.s
trn1 p0.b, p1.b, p2.b
tbl z0.b, {z1.b}, z2.b
tbl z0.b, {z1.b, z2.b}, z3.b
tbl z0.s, {z1.s}, z2.s
tbx z0.b, z1.b, z2.b
ext z0.b, z0.b, z1.b, #1
ext z0.b, z0.b, z1.b, #16
ext z0.b, {z1.b, z2.b}, #1
splice z0.b, p0, z0.b, z1.b
splice z0.b, p0, {z1.b, z2.b}
compact z0.s, p0, z1.s
lasta w0, p0, z1.b
lasta x0, p0, z1.d
lastb w0, p0, z1.b
lastb b0, p0, z1.b
lasta s0, p0, z1.s
clasta w0, p0, w0, z1.b
clastb z0.b, p0, z0.b, z1.b
brka p0.b, p1/z, p2.b
brkb p0.b, p1/z, p2.b
brkas p0.b, p1/z, p2.b
brkbs p0.b, p1/z, p2.b
brkn p0.b, p1/z, p2.b, p0.b
brkns p0.b, p1/z, p2.b, p0.b
brkpa p0.b, p1/z, p2.b, p3.b
brkpb p0.b, p1/z, p2.b, p3.b
pfirst p0.b, p1, p0.b
pnext p0.b, p1, p0.b
trn1 p0.d, p1.d, p2.d
fadd z0.s, z1.s, z2.s
fadd z0.d, z1.d, z2.d
fadd z0.s, p0/m, z0.s, z1.s
fadd z0.s, p0/m, z0.s, #1.0
fsub z0.s, z1.s, z2.s
fsub z0.s, p0/m, z0.s, z1.s
fsubr z0.s, p0/m, z0.s, z1.s
fmul z0.s, z1.s, z2.s
fmul z0.d, z1.d, z2.d
fmul z0.s, p0/m, z0.s, z1.s
fmul z0.s, p0/m, z0.s, #2.0
fmul z0.s, z1.s, z2.s[1]
fmul z0.d, z1.d, z2.d[1]
fdiv z0.s, p0/m, z0.s, z1.s
fdivr z0.s, p0/m, z0.s, z1.s
fmla z0.s, p0/m, z1.s, z2.s
fmla z0.d, p0/m, z1.d, z2.d
fmla z0.s, z1.s, z2.s[1]
fmla z0.d, z1.d, z2.d[1]
fmls z0.s, p0/m, z1.s, z2.s
fmls z0.s, z1.s, z2.s[1]
fmad z0.s, p0/m, z1.s, z2.s
fmsb z0.s, p0/m, z1.s, z2.s
fnmla z0.s, p0/m, z1.s, z2.s
fnmls z0.s, p0/m, z1.s, z2.s
fnmad z0.s, p0/m, z1.s, z2.s
fneg z0.s, p0/m, z1.s
fabs z0.s, p0/m, z1.s
fsqrt z0.s, p0/m, z1.s
fsqrt z0.d, p0/m, z1.d
fmax z0.s, p0/m, z0.s, z1.s
fmin z0.s, p0/m, z0.s, z1.s
fmaxnm z0.s, p0/m, z0.s, z1.s
fminnm z0.d, p0/m, z0.d, z1.d
fmax z0.s, p0/m, z0.s, #1.0
fmaxv s0, p0, z1.s
fminv s0, p0, z1.s
fmaxnmv s0, p0, z1.s
faddv s0, p0, z1.s
faddv d0, p0, z1.d
fadda s0, p0, s0, z1.s
fcmeq p0.s, p1/z, z0.s, z1.s
fcmeq p0.s, p1/z, z0.s, #0.0
fcmne p0.s, p1/z, z0.s, z1.s
fcmgt p0.s, p1/z, z0.s, z1.s
fcmgt p0.s, p1/z, z0.s, #0.0
fcmge p0.s, p1/z, z0.s, z1.s
fcmlt p0.s, p1/z, z0.s, #0.0
fcmle p0.s, p1/z, z0.s, #0.0
fcmuo p0.s, p1/z, z0.s, z1.s
facgt p0.s, p1/z, z0.s, z1.s
facge p0.s, p1/z, z0.s, z1.s
fabd z0.s, p0/m, z0.s, z1.s
frecpe z0.s, z1.s
frecps z0.s, z1.s, z2.s
frsqrte z0.s, z1.s
frsqrts z0.s, z1.s, z2.s
frecpx z0.s, p0/m, z1.s
frintm z0.s, p0/m, z1.s
frintz z0.s, p0/m, z1.s
frintp z0.s, p0/m, z1.s
frintn z0.s, p0/m, z1.s
frinta z0.s, p0/m, z1.s
frintx z0.s, p0/m, z1.s
frinti z0.s, p0/m, z1.s
fcvtzs z0.s, p0/m, z1.s
fcvtzs z0.d, p0/m, z1.d
fcvtzs z0.d, p0/m, z1.s
fcvtzs z0.s, p0/m, z1.d
fcvtzu z0.s, p0/m, z1.s
scvtf z0.s, p0/m, z1.s
scvtf z0.d, p0/m, z1.d
scvtf z0.d, p0/m, z1.s
scvtf z0.s, p0/m, z1.d
ucvtf z0.s, p0/m, z1.s
fcvt z0.s, p0/m, z1.d
fcvt z0.d, p0/m, z1.s
fcvt z0.h, p0/m, z1.s
fcvt z0.s, p0/m, z1.h
fcvtnt z0.s, p0/m, z1.d
fcvtlt z0.d, p0/m, z1.s
fexpa z0.s, z1.s
ftmad z0.s, z0.s, z1.s, #1
ftsmul z0.s, z1.s, z2.s
ftssel z0.s, z1.s, z2.s
fscale z0.s, p0/m, z0.s, z1.s
fcadd z0.s, p0/m, z0.s, z1.s, #90
fcmla z0.s, p0/m, z1.s, z2.s, #90
bfdot z0.s, z1.h, z2.h
bfmmla z0.s, z1.h, z2.h
bfcvt z0.h, p0/m, z1.s
bfcvtnt z0.h, p0/m, z1.s
fmlalb z0.s, z1.h, z2.h
fmlalt z0.s, z1.h, z2.h
fadd z0.h, z1.h, z2.h
fmla z0.h, p0/m, z1.h, z2.h
histcnt z0.s, p0/z, z1.s, z2.s
adclb z0.s, z1.s, z2.s
adclt z0.s, z1.s, z2.s
sbclb z0.s, z1.s, z2.s
eorbt z0.b, z1.b, z2.b
eortb z0.b, z1.b, z2.b
pmullb z0.h, z1.b, z2.b
pmullt z0.h, z1.b, z2.b
pmul z0.b, z1.b, z2.b
cadd z0.s, z0.s, z1.s, #90
sqcadd z0.s, z0.s, z1.s, #90
cmla z0.s, z1.s, z2.s, #90
sqrdcmlah z0.s, z1.s, z2.s, #90
cdot z0.s, z1.b, z2.b, #90
saba z0.s, z1.s, z2.s
uaba z0.b, z1.b, z2.b
sabalb z0.h, z1.b, z2.b
sabalt z0.h, z1.b, z2.b
addhnb z0.b, z1.h, z2.h
addhnt z0.b, z1.h, z2.h
raddhnb z0.b, z1.h, z2.h
subhnb z0.b, z1.h, z2.h
rsubhnt z0.b, z1.h, z2.h
addp z0.b, p0/m, z0.b, z1.b
smaxp z0.s, p0/m, z0.s, z1.s
uminp z0.b, p0/m, z0.b, z1.b
faddp z0.s, p0/m, z0.s, z1.s
fmaxp z0.s, p0/m, z0.s, z1.s
urshl z0.b, p0/m, z0.b, z1.b
srshl z0.s, p0/m, z0.s, z1.s
urshr z0.b, p0/m, z0.b, #1
srshr z0.s, p0/m, z0.s, #1
sqshl z0.b, p0/m, z0.b, #1
uqshl z0.b, p0/m, z0.b, #1
sqshlu z0.b, p0/m, z0.b, #1
urshr z0.s, p0/m, z0.s, #1
ursra z0.s, z1.s, #1
srsra z0.s, z1.s, #1
sqabs z0.s, p0/m, z1.s
sqneg z0.s, p0/m, z1.s
urecpe z0.s, p0/m, z1.s
ursqrte z0.s, p0/m, z1.s
sqdmullb z0.s, z1.h, z2.h
sqdmullt z0.s, z1.h, z2.h
sqdmlalb z0.s, z1.h, z2.h
sqdmlalt z0.s, z1.h, z2.h
sqdmlslb z0.s, z1.h, z2.h
ushllb z0.h, z1.b, #0
ushllt z0.h, z1.b, #0
sshllb z0.h, z1.b, #0
sshllt z0.s, z1.h, #0
sqshrunb z0.b, z1.h, #8
sqshrunt z0.b, z1.h, #8
sqrshrunb z0.b, z1.h, #8
sqshrnb z0.b, z1.h, #8
sqrshrnt z0.b, z1.h, #8
uqrshrnb z0.b, z1.h, #8
uqrshrnt z0.b, z1.h, #8
tbl z0.b, {z1.b, z2.b}, z3.b
tbx z0.b, z1.b, z2.b
whilege p0.b, x0, x1
whilegt p0.b, x0, x1
whilehi p0.b, x0, x1
whilehs p0.b, x0, x1
whilerw p0.b, x0, x1
whilewr p0.b, x0, x1
setffr
rdffr p0.b
rdffr p0.b, p1/z
rdffrs p0.b, p1/z
wrffr p0.b
ldff1h {z0.h}, p0/z, [x0, x1, lsl #1]
ldff1w {z0.s}, p0/z, [x0]
ldff1d {z0.d}, p0/z, [x0]
ldnt1h {z0.h}, p0/z, [x0]
ldnt1w {z0.s}, p0/z, [x0, x1, lsl #2]
stnt1w {z0.s}, p0, [x0, x1, lsl #2]
prfb pldl1keep, p0, [x0]
prfw pldl1keep, p0, [x0, #1, mul vl]
prfd pldl1keep, p0, [x0, x1, lsl #3]
decb x0
decw x0
decd x0, all, mul #2
decp x0, p0.b
incp x0, p0.b
incp z0.s, p0.s
uqincb w0
uqincw x0
sqincb x0, w0
uqdecb x0
sqdecb x0, w0
uqdecp x0, p0.b
sqdecp x0, p0.s
cntp x0, p0, p1.s
insr z0.b, w0
insr z0.s, s0
insr z0.d, x0
insr z0.d, d0
sunpklo z0.s, z1.h
dup z0.b, z1.b[0]
dup z0.s, z1.s[1]
dup z0.d, z1.d[1]
dup z0.h, #1
dup z0.h, #1, lsl #8
dup z0.s, #1, lsl #8
mov z0.s, #1, lsl #8
mov z0.h, p0/m, #1, lsl #8
mov z0.b, b0
mov z0.h, h0
mov z0.h, w0
mov p0.b, p1.b
mov p0.b, p1/z, p2.b
mov p0.b, p1/m, p2.b
movs p0.b, p1.b
movs p0.b, p1/z, p2.b
rev z0.s, z1.s
revb z0.d, p0/m, z1.d
cpy z0.s, p0/m, s0
cpy z0.d, p0/m, d0
fcpy z0.s, p0/m, #1.0
fmov z0.s, p0/m, #1.0
fmov z0.d, p0/m, #0.0
`

type arm64Instruction struct {
	gnu   string
	code  []byte
	plan9 string // empty when arm64asm cannot decode the instruction
}

func assembleArm64Corpus(t *testing.T, dir string) []arm64Instruction {
	t.Helper()
	var lines []string
	for _, l := range strings.Split(arm64RoundTripCorpus, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	asmPath := filepath.Join(dir, "corpus.s")
	objPath := filepath.Join(dir, "corpus.o")
	require.NoError(t, os.WriteFile(asmPath, []byte(".text\n"+strings.Join(lines, "\n")+"\n"), 0o644))
	out, err := exec.Command("clang", "-c", asmPath, "-o", objPath, "--target=aarch64-linux-gnu",
		"-march=armv8.6-a+sve2+crypto+sha3+dotprod+fp16+bf16+i8mm+crc+lse").CombinedOutput()
	require.NoError(t, err, "clang: %s", out)

	f, err := elf.Open(objPath)
	require.NoError(t, err)
	defer f.Close()
	text, err := f.Section(".text").Data()
	require.NoError(t, err)
	require.Len(t, text, 4*len(lines), "every AArch64 instruction is one word")

	var insts []arm64Instruction
	for i, l := range lines {
		code := text[4*i : 4*i+4]
		inst := arm64Instruction{gnu: l, code: code}
		if decoded, err := arm64asm.Decode(code); err == nil {
			inst.plan9 = arm64asm.GoSyntax(decoded, 0, nil, nil)
		}
		insts = append(insts, inst)
	}
	return insts
}

// TestArm64AllowlistRoundTrips checks arm64TrustDecoder against the real
// toolchain: every trusted corpus line must reassemble byte-identically,
// and every allowlisted mnemonic must be exercised by at least one trusted
// line. Lines that are kept as bytes although they would round-trip are
// reported as a count, not a failure: the allowlist is deliberately
// conservative for shapes that GoSyntax spells ambiguously.
func TestArm64AllowlistRoundTrips(t *testing.T) {
	if _, err := exec.LookPath("clang"); err != nil {
		t.Skip("clang not installed")
	}
	goroot, err := exec.Command("go", "env", "GOROOT").Output()
	require.NoError(t, err)
	dir := t.TempDir()
	insts := assembleArm64Corpus(t, dir)

	trustedMnemonics := make(map[string]bool)
	undecodable, keptButFine := 0, 0
	for _, inst := range insts {
		inst := inst
		if inst.plan9 == "" {
			undecodable++
			continue
		}
		t.Run(inst.gnu, func(t *testing.T) {
			line := Line{Assembly: inst.gnu, Disassembled: inst.plan9, Binary: hexBytes(inst.code)}
			trusted := arm64TrustDecoder(line)
			got, err := goAssemble(t, dir, strings.TrimSpace(string(goroot)), "arm64", inst.plan9)
			same := err == nil && len(got) >= 4 && sameArm64Instruction(inst.code, got[:4])
			mnemonic, _, _ := strings.Cut(inst.plan9, " ")
			switch {
			case trusted && err != nil:
				t.Errorf("trusted %q (%s) but cmd/asm rejects it: %v", inst.plan9, inst.gnu, err)
			case trusted && !same:
				t.Errorf("trusted %q (%s) but cmd/asm encodes %08x instead of %08x", inst.plan9, inst.gnu,
					binary.LittleEndian.Uint32(got), binary.LittleEndian.Uint32(inst.code))
			case trusted:
				trustedMnemonics[mnemonic] = true
			case same:
				keptButFine++
			}
		})
	}
	for mnemonic := range arm64AssemblableMnemonics {
		if !trustedMnemonics[mnemonic] {
			t.Errorf("allowlist entry %q is not exercised by a trusted corpus line", mnemonic)
		}
	}
	t.Logf("%d instructions: %d not decodable by arm64asm, %d kept as bytes although cmd/asm would reassemble them",
		len(insts), undecodable, keptButFine)
}

// sameArm64Instruction accepts byte-identical output or, for register
// immediates, a different encoding with the same architectural effect:
// cmd/asm prefers ORR-with-immediate over MOVZ/MOVN when both apply.
func sameArm64Instruction(want, got []byte) bool {
	if binary.LittleEndian.Uint32(want) == binary.LittleEndian.Uint32(got) {
		return true
	}
	wi, err1 := arm64asm.Decode(want)
	gi, err2 := arm64asm.Decode(got)
	if err1 != nil || err2 != nil {
		return false
	}
	if arm64asm.GNUSyntax(wi) == arm64asm.GNUSyntax(gi) {
		return true
	}
	// arm64asm reports MOVZ/MOVN as MOV; the ORR alias prints as ORRW/ORR.
	return wi.Op == arm64asm.MOV && (gi.Op == arm64asm.MOV || gi.Op == arm64asm.ORR)
}
