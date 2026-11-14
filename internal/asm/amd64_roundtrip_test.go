package asm

import (
	"bufio"
	"bytes"
	"debug/elf"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/arch/x86/x86asm"
)

// amd64RoundTripCorpus is Intel-syntax input for clang. Every instruction is
// decoded with x86asm and reassembled with cmd/asm. Lines amd64TrustDecoder
// accepts must come back byte-identical (or as the same instruction); the
// rest must be exactly the lines cmd/asm cannot reproduce, so the denylist
// stays minimal.
var amd64RoundTripCorpus = `
mov rax, rbx
mov rax, qword ptr [rsp + 8]
mov qword ptr [rsp + 8], rax
mov rax, 42
mov rax, -1
movabs rax, 0x7ffffffffffffff8
mov qword ptr [rsp], 0
mov eax, ecx
mov eax, dword ptr [rsp + 8]
mov dword ptr [rsp + 8], eax
mov eax, 42
mov word ptr [rsp + 8], ax
mov ax, word ptr [rsp]
mov r8b, cl
mov byte ptr [rsp + 8], al
mov r8, rsp
mov rax, qword ptr [rdi + 8*rcx + 16]
lea rax, [rsp + 8]
lea rax, [rdi + 8*rcx + 16]
lea rsp, [rbp - 8]
lea r13, [r14 - 256]
lea eax, [rcx + 4]
add rax, rcx
add rax, 7
add qword ptr [rsp + 8], rax
add rax, qword ptr [rsp + 8]
add eax, ecx
add eax, 7
add ax, 7
add al, 7
sub rsp, 32
sub eax, ecx
and rsp, -16
and eax, ecx
and edx, 28
and r11b, 0xfe
or rax, 1
or eax, ecx
xor eax, eax
xor eax, ecx
xor bl, cl
xor al, al
adc rax, rcx
sbb rax, rax
cmp rax, 602
cmp rax, qword ptr [rsp + 8]
cmp qword ptr [rsp + 8], 0
cmp byte ptr [rdi], 0x80
cmp eax, ecx
cmp eax, 7
cmp dword ptr [rsp + 8], 7
cmp bl, cl
cmp al, 7
cmp r8b, 12
test rax, rcx
test rax, 1
test rsi, rsi
test eax, eax
test eax, 0x100
test qword ptr [rsp + 8], rax
test byte ptr [rsp + 8], 1
test al, 1
test sil, 0xe0
inc rax
inc eax
dec eax
dec qword ptr [rdi]
neg rax
neg eax
not eax
not rax
imul rax, rcx
imul eax, ecx
imul rax, qword ptr [rsp + 8]
imul rax, rcx, 12
imul rax, rcx, 1000
imul eax, ecx, 12
mul rcx
div rcx
idiv rcx
shl rax, cl
shl eax, 3
shl ax, 5
shl al, cl
shr eax, cl
shr rax, 3
sar rax, 63
sar eax, 1
rol rax, 13
ror eax, 7
rcl rax, 1
shld rax, rdx, cl
shld rax, rdx, 5
shrd rax, rdx, cl
shrd eax, edx, cl
shld eax, edx, cl
bsf rax, rcx
bsr rax, rcx
bt rax, rcx
bts rax, 3
btr qword ptr [rdi], rax
bswap rax
bswap eax
cqo
cdq
cdqe
cwde
cbw
push rbp
push rax
pop rbp
pop rcx
call rax
jmp rax
ret
int3
ud2
pause
lfence
mfence
sfence
prefetcht0 byte ptr [rdi]
clflush byte ptr [rdi]
rdtsc
cpuid
xchg rax, rcx
cmpxchg qword ptr [rdi], rcx
lock cmpxchg qword ptr [rdi], rcx
xadd qword ptr [rdi], rax
lock xadd qword ptr [rdi], rax
rep movsb
rep stosq
movbe eax, dword ptr [rdi]
movnti qword ptr [rdi], rax
crc32 eax, cl
crc32 rax, rcx
crc32 eax, dword ptr [rdi]
movzx eax, byte ptr [rsp + 7]
movzx eax, word ptr [rsp + 6]
movzx eax, cl
movzx r8d, r9b
movsx eax, cl
movsxd rax, ecx
sete al
setne al
seta al
setbe al
setle al
setge al
sets al
setp al
setb al
setae al
setg dl
setl r8b
sete byte ptr [rsp + 8]
cmove rax, rcx
cmovne rax, rcx
cmovb rax, qword ptr [rsp + 8]
cmovae eax, ecx
cmovl r8, r9
cmovg eax, dword ptr [rsp + 8]
cmovs rax, rcx
cmova rax, rcx
cmovbe rax, rcx
cmovle rax, rcx
cmovge rax, rcx
andn rax, rcx, rdx
andn rax, rcx, qword ptr [rsp + 8]
blsi rax, rcx
blsmsk rax, rcx
blsr rax, rcx
bzhi rax, rcx, rdx
mulx rax, rdx, rcx
pdep rax, rcx, rdx
pext rax, rcx, rdx
sarx rax, rcx, rdx
shlx rax, rcx, rdx
shrx rax, rcx, rdx
shrx eax, dword ptr [rsp + 8], edx
rorx rax, rcx, 7
bextr rax, rcx, rdx
tzcnt rax, rcx
lzcnt rax, rcx
popcnt rax, rcx
cvtsi2sd xmm0, rax
cvtsi2sd xmm0, eax
cvtsi2ss xmm0, rax
cvtsi2sd xmm0, qword ptr [rsp + 8]
cvttsd2si rax, xmm0
cvttsd2si eax, xmm0
cvtsd2si rax, xmm0
cvtsd2si eax, xmm0
cvtss2si rax, xmm0
cvtss2si eax, xmm0
cvttss2si rax, xmm0
cvttss2si eax, xmm0
cvtsi2ss xmm0, eax
cvtpd2dq xmm0, xmm1
cvttpd2dq xmm0, xmm1
cvtps2dq xmm0, xmm1
cvtsd2ss xmm0, xmm1
cvtss2sd xmm0, xmm1
cvtdq2pd xmm0, xmm1
cvtdq2ps xmm0, xmm1
cvttps2dq xmm0, xmm1
cvtps2pd xmm0, xmm1
cvtpd2ps xmm0, xmm1
movaps xmm0, xmm1
movaps xmmword ptr [rsp + 16], xmm0
movups xmm0, xmmword ptr [rdi]
movapd xmm0, xmm1
movupd xmmword ptr [rsp], xmm0
movdqa xmm0, xmm1
movdqa xmm0, xmmword ptr [rsp]
movdqa xmmword ptr [rsp + 16], xmm2
movdqu xmm0, xmmword ptr [rdi]
movdqu xmmword ptr [rsp], xmm0
movd xmm0, eax
movd eax, xmm0
movd xmm0, dword ptr [rsp]
movq xmm0, rax
movq rax, xmm0
movq xmm0, qword ptr [rsp]
movq qword ptr [rsp], xmm0
movsd xmm0, qword ptr [rsp + 8]
movsd qword ptr [rsp + 8], xmm0
movss xmm0, dword ptr [rsp + 8]
movss dword ptr [rsi], xmm0
movhps xmm0, qword ptr [rdi]
movlps xmm0, qword ptr [rdi]
movhlps xmm0, xmm1
movlhps xmm0, xmm1
movntdq xmmword ptr [rdi], xmm0
movntps xmmword ptr [rdi], xmm0
paddq xmm0, xmm1
paddq xmm0, xmmword ptr [rsp + 32]
paddd xmm0, xmm1
paddw xmm0, xmm1
paddb xmm0, xmm1
psubd xmm0, xmm1
psubq xmm0, xmm1
pmulld xmm0, xmm1
pmullw xmm0, xmm1
pmuludq xmm0, xmm1
pmaddwd xmm0, xmm1
pshufd xmm1, xmm0, 238
pshufb xmm0, xmm1
pshuflw xmm0, xmm1, 0x1b
pshufhw xmm0, xmm1, 0x1b
punpcklbw xmm0, xmm1
punpcklwd xmm9, xmmword ptr [rsp + 224]
punpckldq xmm9, xmm8
punpcklqdq xmm0, xmm1
punpckhbw xmm0, xmm1
punpckhwd xmm0, xmm1
punpckhdq xmm0, xmm1
punpckhqdq xmm0, xmm1
packuswb xmm0, xmm1
packsswb xmm0, xmm1
packssdw xmm0, xmm1
packusdw xmm0, xmm1
pcmpeqb xmm0, xmm1
pcmpeqw xmm0, xmm1
pcmpeqd xmm0, xmm1
pcmpeqq xmm0, xmm1
pcmpgtb xmm0, xmm1
pcmpgtd xmm0, xmm1
pmovmskb eax, xmm0
pxor xmm0, xmm0
pand xmm0, xmm1
por xmm0, xmm1
pandn xmm0, xmm1
pminub xmm0, xmm1
pmaxub xmm0, xmm1
pminsd xmm0, xmm1
pmaxsd xmm0, xmm1
pminud xmm0, xmm1
pmaxud xmm0, xmm1
pabsb xmm0, xmm1
pabsd xmm0, xmm1
pinsrb xmm0, eax, 3
pextrb eax, xmm0, 3
pinsrw xmm0, eax, 3
pextrw eax, xmm0, 3
pinsrd xmm0, eax, 1
pextrd eax, xmm0, 1
pinsrq xmm0, rax, 1
pextrq rax, xmm0, 1
pblendw xmm0, xmm1, 0xaa
pblendvb xmm0, xmm1, xmm0
palignr xmm0, xmm1, 4
ptest xmm0, xmm1
psrlw xmm0, 4
psrld xmm0, 4
psrlq xmm0, 4
psraw xmm0, 4
psrad xmm0, 4
psllw xmm0, 4
pslld xmm0, 4
psllq xmm0, 4
psrldq xmm0, 8
pslldq xmm0, 8
psrlw xmm0, xmm1
psrlq xmm0, xmmword ptr [rsp]
pmovzxbw xmm0, xmm1
pmovzxbd xmm0, xmm1
pmovzxwd xmm0, xmm1
pmovzxdq xmm0, xmm1
pmovsxbw xmm0, xmm1
pmovsxwd xmm0, xmm1
pmovsxdq xmm0, xmm1
phaddd xmm0, xmm1
phaddw xmm0, xmm1
psadbw xmm0, xmm1
psignd xmm0, xmm1
mulpd xmm1, xmmword ptr [rsp + 80]
mulsd xmm0, xmm1
mulss xmm0, xmm1
mulps xmm0, xmm1
addsd xmm0, xmm1
addps xmm0, xmm1
subps xmm0, xmm1
subsd xmm0, xmm1
divsd xmm0, xmm1
divps xmm0, xmm1
sqrtsd xmm0, xmm1
sqrtps xmm0, xmm1
rsqrtps xmm0, xmm1
andps xmm0, xmm1
andnps xmm0, xmm1
orps xmm0, xmm1
xorps xmm0, xmm0
xorpd xmm0, xmm0
unpcklpd xmm0, xmm1
unpckhps xmm0, xmm1
shufps xmm0, xmm1, 0x1b
shufpd xmm0, xmm1, 1
comisd xmm0, xmm1
ucomisd xmm0, xmm1
ucomiss xmm0, xmm1
cmpltps xmm0, xmm1
cmpeqsd xmm0, xmm1
cmpless xmm0, xmm1
cmpneqpd xmm0, xmm1
roundsd xmm0, xmm1, 9
roundps xmm0, xmm1, 9
maxsd xmm0, xmm1
minps xmm0, xmm1
maxps xmm0, xmm1
blendvps xmm0, xmm1, xmm0
blendps xmm0, xmm1, 5
dpps xmm0, xmm1, 0xff
haddps xmm0, xmm1
movmskps eax, xmm0
movmskpd eax, xmm0
pclmulqdq xmm0, xmm1, 0x11
aesenc xmm0, xmm1
aesenclast xmm0, xmm1
aesdec xmm0, xmm1
aeskeygenassist xmm0, xmm1, 1
sha256rnds2 xmm1, xmm2, xmm0
sha256msg1 xmm1, xmm2
sha1rnds4 xmm1, xmm2, 1
stmxcsr dword ptr [rsp]
ldmxcsr dword ptr [rsp]
vmovdqu ymm0, ymmword ptr [rdi]
vmovdqu ymmword ptr [rsp + 64], ymm0
vmovdqa ymm0, ymm1
vmovdqa xmm0, xmm1
vmovaps ymm0, ymmword ptr [rsp + 32]
vmovaps xmmword ptr [rsp + 16], xmm0
vmovups ymmword ptr [rsp], ymm0
vmovupd ymm0, ymmword ptr [rdi]
vmovd xmm0, eax
vmovd eax, xmm0
vmovq xmm0, rax
vmovq rax, xmm0
vmovq xmm0, qword ptr [rsp]
vmovsd xmm0, qword ptr [rsp + 8]
vmovsd qword ptr [rsp + 8], xmm0
vmovss xmm0, dword ptr [rdi]
vmovss dword ptr [rsp + 8], xmm1
vmovss xmm0, dword ptr [rdi + 4*rcx]
vmovss xmm0, xmm1, xmm2
vmovsd xmm0, xmm1, xmm2
vmovhps xmm0, xmm1, qword ptr [rdi]
vmovlhps xmm0, xmm1, xmm2
vmovddup ymm0, ymm1
vmovshdup ymm0, ymm1
vpaddq ymm0, ymm1, ymmword ptr [rsp + 64]
vpaddd ymm0, ymm1, ymm2
vpaddw ymm0, ymm1, ymm2
vpaddb ymm0, ymm1, ymm2
vpsubb ymm0, ymm1, ymm2
vpsubq ymm0, ymm1, ymm2
vpaddsb ymm0, ymm1, ymm2
vpaddusw ymm0, ymm1, ymm2
vpsubusb ymm0, ymm1, ymm2
vpxor xmm0, xmm0, xmm0
vpxor ymm0, ymm1, ymm2
vpand ymm0, ymm1, ymm2
vpandn ymm0, ymm1, ymm2
vpor ymm0, ymm1, ymm2
vpcmpeqb ymm0, ymm1, ymm2
vpcmpeqw ymm0, ymm1, ymm2
vpcmpeqd ymm0, ymm1, ymm2
vpcmpeqq ymm0, ymm1, ymm2
vpcmpgtb ymm0, ymm1, ymm2
vpcmpgtd ymm0, ymm1, ymm2
vpmovmskb eax, ymm0
vpshufb ymm0, ymm1, ymm2
vpshufd ymm0, ymm1, 0x1b
vpshuflw ymm0, ymm1, 0x1b
vpshufhw ymm0, ymm1, 0x1b
vpermq ymm1, ymm0, 0x1b
vpermd ymm0, ymm1, ymm2
vpermps ymm0, ymm1, ymm2
vpermpd ymm0, ymm1, 0x1b
vperm2i128 ymm0, ymm1, ymm2, 0x21
vperm2f128 ymm0, ymm1, ymm2, 0x21
vpblendd ymm0, ymm1, ymm2, 0xaa
vpblendw ymm0, ymm1, ymm2, 0xaa
vpblendvb ymm0, ymm1, ymm2, ymm3
vblendvps ymm0, ymm1, ymm2, ymm3
vblendps ymm0, ymm1, ymm2, 5
vpbroadcastb ymm0, xmm1
vpbroadcastw ymm0, word ptr [rdi]
vpbroadcastd ymm0, xmm1
vpbroadcastq ymm0, qword ptr [rdi]
vbroadcasti128 ymm0, xmmword ptr [rdi]
vbroadcastf128 ymm0, xmmword ptr [rdi]
vbroadcastss ymm0, xmm1
vbroadcastss ymm0, dword ptr [rdi]
vbroadcastsd ymm0, xmm1
vpsrlw ymm0, ymm1, 4
vpsrld ymm0, ymm1, 4
vpsrlq ymm0, ymm1, 4
vpsraw ymm0, ymm1, 4
vpsrad ymm0, ymm1, 4
vpsllw ymm0, ymm1, 4
vpslld ymm0, ymm1, 4
vpsllq ymm0, ymm1, 4
vpsrldq ymm0, ymm1, 8
vpslldq ymm0, ymm1, 8
vpsrlw ymm0, ymm1, xmm2
vpsrlvd ymm0, ymm1, ymm2
vpsrlvq ymm0, ymm1, ymm2
vpsllvq ymm0, ymm1, ymm2
vpsravd ymm0, ymm1, ymm2
vextracti128 xmm0, ymm1, 1
vinserti128 ymm0, ymm1, xmm2, 1
vextractf128 xmm0, ymm1, 1
vinsertf128 ymm0, ymm1, xmm2, 1
vinsertps xmm0, xmm1, xmm2, 0x10
vinsertps xmm0, xmm1, dword ptr [rdi + 4], 0x10
vextractps eax, xmm0, 1
vpinsrb xmm0, xmm1, eax, 3
vpinsrw xmm0, xmm1, eax, 3
vpinsrd xmm0, xmm1, eax, 1
vpinsrq xmm0, xmm1, rax, 1
vpextrb eax, xmm0, 3
vpextrw eax, xmm0, 3
vpextrd eax, xmm0, 1
vpextrq rax, xmm0, 1
vpunpcklbw ymm0, ymm1, ymm2
vpunpcklwd ymm0, ymm1, ymm2
vpunpckldq ymm0, ymm1, ymm2
vpunpcklqdq ymm0, ymm1, ymm2
vpunpckhbw ymm0, ymm1, ymm2
vpunpckhwd ymm0, ymm1, ymm2
vpunpckhdq ymm0, ymm1, ymm2
vpunpckhqdq ymm0, ymm1, ymm2
vpmovzxbw ymm0, xmm1
vpmovzxbd ymm0, xmm1
vpmovzxwd ymm0, xmm1
vpmovzxdq ymm0, xmm1
vpmovsxbw ymm0, xmm1
vpmovsxwd ymm0, xmm1
vpmovsxdq ymm0, xmm1
vpackuswb ymm0, ymm1, ymm2
vpacksswb ymm0, ymm1, ymm2
vpackusdw ymm0, ymm1, ymm2
vpackssdw ymm0, ymm1, ymm2
vpmaddwd ymm0, ymm1, ymm2
vpmaddubsw ymm0, ymm1, ymm2
vpmulld ymm0, ymm1, ymm2
vpmullw ymm0, ymm1, ymm2
vpmulhw ymm0, ymm1, ymm2
vpmulhuw ymm0, ymm1, ymm2
vpmuludq ymm0, ymm1, ymm2
vpmuldq ymm0, ymm1, ymm2
vpsadbw ymm0, ymm1, ymm2
vpalignr ymm0, ymm1, ymm2, 4
vptest ymm0, ymm1
vpminub ymm0, ymm1, ymm2
vpmaxub ymm0, ymm1, ymm2
vpminsb ymm0, ymm1, ymm2
vpmaxsd ymm0, ymm1, ymm2
vpminud ymm0, ymm1, ymm2
vpmaxuw ymm0, ymm1, ymm2
vpabsb ymm0, ymm1
vpabsd ymm0, ymm1
vphaddd ymm0, ymm1, ymm2
vphaddw ymm0, ymm1, ymm2
vpavgb ymm0, ymm1, ymm2
vpsignb ymm0, ymm1, ymm2
vpgatherdd ymm0, dword ptr [rdi + 4*ymm1], ymm2
vpmaskmovd ymm0, ymm1, ymmword ptr [rdi]
vmaskmovps ymmword ptr [rdi], ymm1, ymm0
vmulsd xmm3, xmm0, xmm1
vaddsd xmm1, xmm2, xmm3
vsubsd xmm0, xmm1, xmm2
vdivsd xmm0, xmm1, xmm2
vmulss xmm0, xmm1, xmm2
vmulps ymm0, ymm1, ymm2
vmulpd ymm0, ymm1, ymm2
vaddps ymm0, ymm1, ymmword ptr [rsp]
vaddpd ymm0, ymm1, ymm2
vsubps ymm0, ymm1, ymm2
vdivps ymm0, ymm1, ymm2
vfmadd231ps ymm0, ymm1, ymm2
vfmadd231ss xmm0, xmm1, dword ptr [rdi]
vfmadd213sd xmm0, xmm1, xmm2
vfmadd132ps ymm0, ymm1, ymm2
vfmsub231ps ymm0, ymm1, ymm2
vfnmadd231pd ymm0, ymm1, ymm2
vfnmsub213ps ymm0, ymm1, ymm2
vsqrtps ymm0, ymm1
vsqrtsd xmm0, xmm1, xmm2
vrsqrtps ymm0, ymm1
vrcpps ymm0, ymm1
vxorps ymm0, ymm0, ymm0
vxorpd ymm0, ymm1, ymm2
vandps ymm0, ymm1, ymm2
vandnpd ymm0, ymm1, ymm2
vorps ymm0, ymm1, ymm2
vshufps ymm0, ymm1, ymm2, 0x1b
vshufpd ymm0, ymm1, ymm2, 5
vunpcklps ymm0, ymm1, ymm2
vunpckhpd ymm0, ymm1, ymm2
vcvtps2pd ymm0, xmm1
vcvtpd2ps xmm0, ymm1
vcvtdq2ps ymm0, ymm1
vcvtdq2pd ymm0, xmm1
vcvtps2dq ymm0, ymm1
vcvttps2dq ymm0, ymm1
vcvttpd2dq xmm0, ymm1
vcvtsi2sd xmm0, xmm1, rax
vcvtsi2sd xmm0, xmm1, eax
vcvtsi2ss xmm0, xmm1, rax
vcvttsd2si rax, xmm0
vcvttsd2si eax, xmm0
vcvttss2si rax, xmm0
vcvtsd2ss xmm0, xmm1, xmm2
vcvtss2sd xmm0, xmm1, xmm2
vcvtph2ps ymm0, xmm1
vcvtps2ph xmm0, ymm1, 0
vroundps ymm0, ymm1, 9
vroundsd xmm0, xmm1, xmm2, 9
vcmpltps ymm0, ymm1, ymm2
vcmpps ymm0, ymm1, ymm2, 0x1e
vcmpeqsd xmm0, xmm1, xmm2
vcmpneqpd ymm0, ymm1, ymm2
vmaxps ymm0, ymm1, ymm2
vminpd ymm0, ymm1, ymm2
vmaxsd xmm0, xmm1, xmm2
vhaddps ymm0, ymm1, ymm2
vdpps ymm0, ymm1, ymm2, 0xff
vmovmskps eax, ymm0
vmovmskpd eax, ymm0
vcomisd xmm0, xmm1
vucomiss xmm0, xmm1
vptest ymm0, ymm1
vtestps ymm0, ymm1
vzeroupper
vzeroall
vpclmulqdq xmm0, xmm1, xmm2, 0x11
vaesenc ymm0, ymm1, ymm2
vaesdeclast xmm0, xmm1, xmm2
vmovdqu64 zmm0, zmmword ptr [rsp + 128]
vmovdqu64 zmmword ptr [rdi], zmm0
vmovdqa64 zmm0, zmm1
vmovdqu8 zmm0, zmmword ptr [rdi]
vmovdqu16 zmm0, zmm1
vmovdqu32 zmm0 {k1}{z}, zmmword ptr [rdi]
vmovdqa32 zmmword ptr [rdi] {k1}, zmm0
vmovups zmm0, zmmword ptr [rdi]
vmovaps zmmword ptr [rsp + 64], zmm0
vpaddq zmm0, zmm1, zmm2
vpaddq zmm0, zmm1, zmmword ptr [rdi + 8*rcx + 64]
vpaddd zmm0, zmm1, zmm2
vpaddd zmm0 {k1}, zmm1, zmm2
vpaddd zmm0 {k1}{z}, zmm1, zmm2
vpaddw zmm0, zmm1, zmm2
vpaddb zmm0, zmm1, zmm2
vpsubq zmm0, zmm1, zmm2
vpxorq zmm0, zmm0, zmm0
vpxord zmm0, zmm1, zmm2
vpandq zmm0, zmm1, zmm2
vpandd zmm0, zmm1, zmm2
vporq zmm0, zmm1, zmm2
vpandnq zmm0, zmm1, zmm2
vpternlogd zmm0, zmm1, zmm2, 0x96
vpternlogq zmm0, zmm1, zmm2, 0x96
vpcmpeqb k1, zmm0, zmm1
vpcmpeqd k1, zmm0, zmm1
vpcmpub k1, zmm0, zmm1, 6
vpcmpd k1, zmm0, zmm1, 1
vpcmpuq k1 {k2}, zmm0, zmm1, 2
vptestmd k1, zmm0, zmm1
vptestnmb k1, zmm0, zmm1
kmovq rax, k1
kmovq k1, rax
kmovd eax, k1
kmovw k1, eax
kmovb k1, k2
kortestq k1, k1
kortestw k1, k2
kandq k1, k2, k3
korq k1, k2, k3
kxorq k1, k2, k3
knotq k1, k2
kshiftrq k1, k2, 8
kunpckdq k1, k2, k3
vpbroadcastq zmm0, rax
vpbroadcastd zmm0, eax
vpbroadcastb zmm0, xmm1
vpbroadcastd zmm0, dword ptr [rdi]
vbroadcastss zmm0, dword ptr [rdi]
vbroadcasti32x4 zmm0, xmmword ptr [rdi]
vbroadcasti64x4 zmm0, ymmword ptr [rdi]
vextracti64x4 ymm0, zmm1, 1
vextracti32x4 xmm0, zmm1, 3
vinserti64x4 zmm0, zmm1, ymm2, 1
vinserti32x4 zmm0, zmm1, xmm2, 3
vpermt2b zmm1, zmm0, zmm2
vpermi2d zmm1, zmm0, zmm2
vpermt2q zmm1, zmm0, zmm2
vpermb zmm0, zmm1, zmm2
vpermd zmm0, zmm1, zmm2
vpermq zmm0, zmm1, 0x1b
vpermw zmm0, zmm1, zmm2
vpshufb zmm0, zmm1, zmm2
vpshufd zmm0, zmm1, 0x1b
vpsrlw zmm0, zmm1, 4
vpsrld zmm0, zmm1, 4
vpsrlq zmm0, zmm1, 4
vpsllq zmm0, zmm1, 4
vpsraq zmm0, zmm1, 4
vpsrldq zmm0, zmm1, 8
vpsrlvd zmm0, zmm1, zmm2
vpsllvq zmm0, zmm1, zmm2
vprolq zmm0, zmm1, 7
vprord zmm0, zmm1, 7
vpminub zmm0, zmm1, zmm2
vpmaxsd zmm0, zmm1, zmm2
vpminuq zmm0, zmm1, zmm2
vpabsq zmm0, zmm1
vpmullw zmm0, zmm1, zmm2
vpmulld zmm0, zmm1, zmm2
vpmullq zmm0, zmm1, zmm2
vpmuludq zmm0, zmm1, zmm2
vpmaddwd zmm0, zmm1, zmm2
vpmaddubsw zmm0, zmm1, zmm2
vpsadbw zmm0, zmm1, zmm2
vpunpcklbw zmm0, zmm1, zmm2
vpunpckhqdq zmm0, zmm1, zmm2
vpackuswb zmm0, zmm1, zmm2
vpackssdw zmm0, zmm1, zmm2
vpalignr zmm0, zmm1, zmm2, 4
valignd zmm0, zmm1, zmm2, 4
valignq zmm0, zmm1, zmm2, 1
vpcompressd zmmword ptr [rdi] {k1}, zmm0
vpcompressq zmm0 {k1}{z}, zmm1
vpexpandd zmm0 {k1}{z}, zmmword ptr [rdi]
vpmovzxbw zmm0, ymm1
vpmovzxbd zmm0, xmm1
vpmovzxdq zmm0, ymm1
vpmovsxwd zmm0, ymm1
vpmovwb ymm0, zmm1
vpmovdb xmm0, zmm1
vpmovqd ymm0, zmm1
vpmovuswb ymm0, zmm1
vpmovm2b zmm0, k1
vpmovb2m k1, zmm0
vpmovd2m k1, zmm0
vpblendmb zmm0 {k1}, zmm1, zmm2
vpblendmd zmm0 {k1}, zmm1, zmm2
vblendmps zmm0 {k1}, zmm1, zmm2
vpconflictd zmm0, zmm1
vplzcntd zmm0, zmm1
vpopcntd zmm0, zmm1
vpopcntb zmm0, zmm1
vpshldq zmm0, zmm1, zmm2, 8
vpshrdvd zmm0, zmm1, zmm2
vpmultishiftqb zmm0, zmm1, zmm2
vpdpbusd zmm0, zmm1, zmm2
vpdpwssd zmm0, zmm1, zmm2
vgf2p8affineqb zmm0, zmm1, zmm2, 0
vgf2p8mulb zmm0, zmm1, zmm2
vfmadd231ps zmm0, zmm1, zmm2
vfmadd213pd zmm0, zmm1, zmm2
vfnmadd231ps zmm0, zmm1, zmm2
vaddps zmm0, zmm1, zmm2
vaddps zmm0, zmm1, dword ptr [rdi]{1to16}
vaddpd zmm0, zmm1, qword ptr [rdi]{1to8}
vmulps zmm0, zmm1, zmm2
vmulpd zmm0 {k1}, zmm1, zmm2
vsubps zmm0, zmm1, zmm2
vdivpd zmm0, zmm1, zmm2
vsqrtps zmm0, zmm1
vrsqrt14ps zmm0, zmm1
vrcp14pd zmm0, zmm1
vmaxps zmm0, zmm1, zmm2
vminpd zmm0, zmm1, zmm2
vxorps zmm0, zmm0, zmm0
vandps zmm0, zmm1, zmm2
vcvtdq2pd zmm0, ymm1
vcvtdq2ps zmm0, zmm1
vcvtps2dq zmm0, zmm1
vcvttps2dq zmm0, zmm1
vcvtps2pd zmm0, ymm1
vcvtpd2ps ymm0, zmm1
vcvtudq2ps zmm0, zmm1
vcvtqq2pd zmm0, zmm1
vcvttpd2qq zmm0, zmm1
vcvtps2ph ymm0, zmm1, 0
vcvtph2ps zmm0, ymm1
vrndscaleps zmm0, zmm1, 9
vreduceps zmm0, zmm1, 9
vscalefps zmm0, zmm1, zmm2
vgetexpps zmm0, zmm1
vfixupimmps zmm0, zmm1, zmm2, 0
vrangeps zmm0, zmm1, zmm2, 0
vcmpps k1, zmm0, zmm1, 0x1e
vcmpltps k1, zmm0, zmm1
vcmppd k1 {k2}, zmm0, zmm1, 0
vfpclassps k1, zmm0, 0x66
vshufps zmm0, zmm1, zmm2, 0x1b
vshuff32x4 zmm0, zmm1, zmm2, 0x4e
vshufi64x2 zmm0, zmm1, zmm2, 0x4e
vunpcklps zmm0, zmm1, zmm2
vmovshdup zmm0, zmm1
vpgatherdd zmm0 {k1}, dword ptr [rdi + 4*zmm1]
vpscatterdd dword ptr [rdi + 4*zmm1] {k1}, zmm0
vpaddd ymm0 {k1}, ymm1, ymm2
vpaddq xmm0 {k1}{z}, xmm1, xmm2
vmovdqu8 ymm0 {k1}, ymmword ptr [rdi]
vpcmpub k1, ymm0, ymm1, 6
vpternlogd ymm0, ymm1, ymm2, 0x96
vpbroadcastb ymm0, eax
vpmovwb xmm0, ymm1
vpermt2b ymm1, ymm0, ymm2
vpermb ymm0, ymm1, ymm2
vpmovm2b ymm0, k1
vpmovb2m k1, ymm0
fld qword ptr [rsp]
fld dword ptr [rsp]
fld tbyte ptr [rsp]
fstp qword ptr [rsp]
fstp tbyte ptr [rsp]
fild qword ptr [rsp]
fistp qword ptr [rsp]
fmul st, st(1)
fmul st(1), st
fmulp st(1), st
fadd st, st(1)
faddp st(1), st
fsubrp st(1), st
fdivrp st(1), st
fxch st(1)
fucomip st, st(1)
fldz
fld1
fabs
fchs
fsqrt
fnstcw word ptr [rsp]
fldcw word ptr [rsp]
`

// amd64Instruction is one corpus line after clang assembled it.
type amd64Instruction struct {
	intel string
	code  []byte
	plan9 string
}

func assembleAmd64Corpus(t *testing.T, dir string) []amd64Instruction {
	t.Helper()
	var lines []string
	for _, l := range strings.Split(amd64RoundTripCorpus, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	// One instruction per line, each followed by a 16-byte marker so the text
	// section can be split without a second disassembler.
	const marker = "\t.byte 0x90,0x90,0x90,0x90,0x90,0x90,0x90,0x90,0xcc,0xcc,0xcc,0xcc,0xcc,0xcc,0xcc,0xcc\n"
	var src strings.Builder
	src.WriteString(".intel_syntax noprefix\n.text\n")
	for _, l := range lines {
		src.WriteString(l + "\n" + marker)
	}
	asmPath := filepath.Join(dir, "corpus.s")
	objPath := filepath.Join(dir, "corpus.o")
	require.NoError(t, os.WriteFile(asmPath, []byte(src.String()), 0o644))
	out, err := exec.Command("clang", "-c", asmPath, "-o", objPath, "--target=x86_64-linux-gnu",
		"-march=x86-64", "-mavx512f", "-mavx512bw", "-mavx512vl", "-mavx512dq", "-mavx512vbmi", "-mavx512vbmi2",
		"-mavx512vnni", "-mavx512bitalg", "-mavx512vpopcntdq", "-mavx512cd", "-mgfni", "-mbmi", "-mbmi2",
		"-mf16c", "-mfma", "-mvaes", "-mvpclmulqdq", "-msha", "-maes", "-mpclmul", "-mmovbe", "-mlzcnt", "-mpopcnt").CombinedOutput()
	require.NoError(t, err, "clang: %s", out)

	f, err := elf.Open(objPath)
	require.NoError(t, err)
	defer f.Close()
	text, err := f.Section(".text").Data()
	require.NoError(t, err)

	sep := []byte{0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0xcc, 0xcc, 0xcc, 0xcc, 0xcc, 0xcc, 0xcc, 0xcc}
	chunks := bytes.Split(text, sep)
	require.Len(t, chunks, len(lines)+1, "marker split mismatch")

	var insts []amd64Instruction
	for i, l := range lines {
		code := chunks[i]
		inst, err := x86asm.Decode(code, 64)
		require.NoError(t, err, "x86asm cannot decode %q (%x)", l, code)
		require.Equal(t, len(code), inst.Len, "%q decoded with the wrong length", l)
		insts = append(insts, amd64Instruction{intel: l, code: code, plan9: x86asm.GoSyntax(inst, 0, nil)})
	}
	return insts
}

// goAssemble runs cmd/asm on a Plan 9 instruction and returns its encoding.
func goAssemble(t *testing.T, dir, goroot, plan9 string) ([]byte, error) {
	t.Helper()
	src := filepath.Join(dir, "g.s")
	obj := filepath.Join(dir, "g.o")
	require.NoError(t, os.WriteFile(src, []byte("#include \"textflag.h\"\nTEXT ·f(SB), NOSPLIT|NOFRAME, $0-0\n\t"+plan9+"\n"), 0o644))
	cmd := exec.Command("go", "tool", "asm", "-S", "-I", filepath.Join(goroot, "pkg", "include"), "-o", obj, src)
	cmd.Env = append(os.Environ(), "GOARCH=amd64", "GOOS=linux")
	out, err := cmd.CombinedOutput()
	if err != nil {
		for _, l := range strings.Split(string(out), "\n") {
			if strings.Contains(l, "g.s:3") || strings.Contains(l, "invalid instruction") {
				return nil, fmt.Errorf("%s", strings.TrimSpace(l))
			}
		}
		return nil, fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	var code []byte
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) < 2 || !strings.HasPrefix(f[0], "0x") || len(f[1]) != 2 {
			continue
		}
		for _, h := range f[1:] {
			b, err := hex.DecodeString(h)
			if err != nil || len(h) != 2 {
				break
			}
			code = append(code, b...)
		}
	}
	return code, nil
}

// TestAmd64DenylistRoundTrips checks amd64TrustDecoder against the real
// toolchain: every trusted corpus line must reassemble to the same
// instruction, and every denied line must be one cmd/asm really gets wrong,
// so that the denylist cannot silently grow stale in either direction.
func TestAmd64DenylistRoundTrips(t *testing.T) {
	if _, err := exec.LookPath("clang"); err != nil {
		t.Skip("clang not installed")
	}
	goroot, err := exec.Command("go", "env", "GOROOT").Output()
	require.NoError(t, err)
	dir := t.TempDir()
	insts := assembleAmd64Corpus(t, dir)

	deniedMnemonics := make(map[string]bool)
	deniedPrefixes := make(map[string]bool)
	for _, inst := range insts {
		inst := inst
		t.Run(inst.intel, func(t *testing.T) {
			trusted := amd64TrustDecoder(Line{Assembly: inst.intel, Disassembled: inst.plan9, Binary: hexBytes(inst.code)})
			got, err := goAssemble(t, dir, strings.TrimSpace(string(goroot)), inst.plan9)
			same := err == nil && sameAmd64Instruction(inst.code, got)
			mnemonic, _, _ := strings.Cut(inst.plan9, " ")
			switch {
			case trusted && err != nil:
				t.Errorf("trusted %q (%s) but cmd/asm rejects it: %v", inst.plan9, inst.intel, err)
			case trusted && !same:
				t.Errorf("trusted %q (%s) but cmd/asm encodes %x instead of %x", inst.plan9, inst.intel, got, inst.code)
			case !trusted && same && deniedByPrefix(mnemonic):
				// cmd/asm accepts some Intel condition names (SETNE, SETA) but
				// not others (SETE, SETB); the family is denied as a whole.
			case !trusted && same:
				t.Errorf("denied %q (%s) although cmd/asm reassembles it correctly; drop it from the denylist", inst.plan9, inst.intel)
			case !trusted && deniedByPrefix(mnemonic):
				deniedPrefixes[prefixOf(mnemonic)] = true
			case !trusted:
				deniedMnemonics[mnemonic] = true
			}
		})
	}
	for mnemonic := range amd64UnassemblableMnemonics {
		if !deniedMnemonics[mnemonic] {
			t.Errorf("denylist entry %q is not exercised by the corpus", mnemonic)
		}
	}
	for _, prefix := range amd64UnassemblablePrefixes {
		if !deniedPrefixes[prefix] {
			t.Errorf("denylist prefix %q has no failing instruction in the corpus", prefix)
		}
	}
}

func deniedByPrefix(mnemonic string) bool { return prefixOf(mnemonic) != "" }

func prefixOf(mnemonic string) string {
	for _, prefix := range amd64UnassemblablePrefixes {
		if strings.HasPrefix(mnemonic, prefix) {
			return prefix
		}
	}
	return ""
}

func hexBytes(code []byte) []string {
	out := make([]string, len(code))
	for i, b := range code {
		out[i] = fmt.Sprintf("%02x", b)
	}
	return out
}

// sameAmd64Instruction accepts byte-identical output or a different
// encoding of the same instruction (cmd/asm may pick a shorter form, e.g.
// the 32-bit immediate move for MOVQ $imm when it zero-extends identically).
func sameAmd64Instruction(want, got []byte) bool {
	if bytes.Equal(want, got) {
		return true
	}
	wi, err1 := x86asm.Decode(want, 64)
	gi, err2 := x86asm.Decode(got, 64)
	if err1 != nil || err2 != nil {
		return false
	}
	if x86asm.GNUSyntax(wi, 0, nil) == x86asm.GNUSyntax(gi, 0, nil) {
		return true
	}
	// mov $imm32, %eax zero-extends into %rax: same architectural effect as
	// mov $imm32, %rax with a non-negative immediate.
	if wi.Op == x86asm.MOV && gi.Op == x86asm.MOV {
		wimm, ok1 := wi.Args[1].(x86asm.Imm)
		gimm, ok2 := gi.Args[1].(x86asm.Imm)
		wr, ok3 := wi.Args[0].(x86asm.Reg)
		gr, ok4 := gi.Args[0].(x86asm.Reg)
		if ok1 && ok2 && ok3 && ok4 && uint32(wimm) == uint32(gimm) && wimm >= 0 && wimm <= 0xffffffff &&
			x86asm.RAX <= wr && wr <= x86asm.R15 && gr == wr-x86asm.RAX+x86asm.EAX {
			return true
		}
	}
	return false
}
