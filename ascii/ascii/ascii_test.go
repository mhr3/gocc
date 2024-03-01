package ascii

import (
	"fmt"
	"math/rand"
	"testing"

	segAscii "github.com/segmentio/asm/ascii"
)

func makeASCII(n int) []byte {
	data := make([]byte, n)
	for i := range data {
		data[i] = byte(rand.Uint32() & 0x7f)
	}
	return data
}

type ValidTest struct {
	in  string
	exp bool
}

var validTests = []ValidTest{
	{"", true},
	{"a", true},
	{"abc", true},
	{"Ж", false},
	{"ЖЖ", false},
	{"брэд-ЛГТМ", false},
	{"☺☻☹", false},
	{"aa\xe2", false},
	{string([]byte{66, 250}), false},
	{string([]byte{66, 250, 67}), false},
	{"a\uFFFDb", false},
	{string("\xF4\x8F\xBF\xBF"), false},     // U+10FFFF
	{string("\xF4\x90\x80\x80"), false},     // U+10FFFF+1; exp of range
	{string("\xF7\xBF\xBF\xBF"), false},     // 0x1FFFFF; exp of range
	{string("\xFB\xBF\xBF\xBF\xBF"), false}, // 0x3FFFFFF; exp of range
	{string("\xc0\x80"), false},             // U+0000 encoded in two bytes: incorrect
	{string("\xed\xa0\x80"), false},         // U+D800 high surrogate (sic)
	{string("\xed\xbf\xbf"), false},         // U+DFFF low surrogate (sic)
	{"hellowo\xff", false},
	{"hellowor", true},
}

func TestAscii(t *testing.T) {
	for _, vt := range validTests {
		if IsASCII(vt.in) != vt.exp {
			t.Errorf("IsASCII(%q) = %v; want %v", vt.in, !vt.exp, vt.exp)
		}
	}

	for _, vt := range validTests {
		pt := "0123456789ab" + vt.in
		if IsASCII(pt) != vt.exp {
			t.Errorf("IsASCII(%q) = %v; want %v", pt, !vt.exp, vt.exp)
		}
	}
}

func TestFfs(t *testing.T) {
	for i := 4; i < 6400; i++ {
		data := makeASCII(i)
		if IsASCII(string(data)) != true {
			t.Errorf("IsASCII(%q) = false; want true", data)
		}
		if res := IndexNonASCII(string(data)); res != -1 {
			t.Errorf("IndexNonASCII(%q[%d]) = %d; want %d", data, len(data), res, -1)
		}

		idx := rand.Intn(i)
		data[idx] |= 0x80
		if IsASCII(string(data)) != false {
			t.Errorf("IsASCII(%q) = true; want false", data)
		}
		if res := IndexNonASCII(string(data)); res != idx {
			t.Errorf("IndexNonASCII(%q[%d]) = %d; want %d", data, len(data), res, idx)
		}
	}
}

func indexNonAsciiGo(s []byte) int {
	for i, r := range s {
		if r >= 0x80 {
			return i
		}
	}
	return -1
}

func isAsciiGo(s []byte) bool {
	for _, r := range s {
		if r >= 0x80 {
			return false
		}
	}
	return true
}

func BenchmarkAscii(b *testing.B) {
	for _, n := range []int{1, 7, 15, 44, 100, 1000} {
		asciiBuf := makeASCII(n)
		asciiStr := string(asciiBuf)

		b.Run(fmt.Sprintf("go-%d", n), func(b *testing.B) {
			b.SetBytes(int64(len(asciiStr)))
			for i := 0; i < b.N; i++ {
				isAsciiGo(asciiBuf)
			}
		})

		b.Run(fmt.Sprintf("segment-%d", n), func(b *testing.B) {
			b.SetBytes(int64(len(asciiStr)))
			for i := 0; i < b.N; i++ {
				segAscii.ValidString(asciiStr)
			}
		})

		b.Run(fmt.Sprintf("simd-%d", n), func(b *testing.B) {
			b.SetBytes(int64(len(asciiStr)))
			for i := 0; i < b.N; i++ {
				IsASCII(asciiStr)
			}
		})
	}
}
