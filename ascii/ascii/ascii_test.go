package ascii

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"unicode"

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
		if res := IndexBit(string(data), 0x80); res != -1 {
			t.Errorf("IndexBit(string%q[%d]) = %d; want %d", data, len(data), res, -1)
		}

		idx := rand.Intn(i)
		data[idx] |= 0x80
		if IsASCII(string(data)) != false {
			t.Errorf("IsASCII(%q) = true; want false", data)
		}
		if res := IndexBit(string(data), 0x80); res != idx {
			t.Errorf("IndexBit(string%q[%d]) = %d; want %d", data, len(data), res, idx)
		}
	}
}

func TestContainsFold(t *testing.T) {
	containsTests := []struct {
		str, substr string
		expected    bool
	}{
		{"abc", "bc", true},
		{"abc", "bcd", false},
		{"abc", "", true},
		{"", "a", false},
		// 2-byte needle
		{"xxxxxx", "01", false},
		{"01xxxx", "01", true},
		{"xx01xx", "01", true},
		{"xxxx01", "01", true},
		{"01xxxxx"[1:], "01", false},
		{"xxxxx01"[:6], "01", false},
		// 3-byte needle
		{"xxxxxxx", "012", false},
		{"012xxxx", "012", true},
		{"xx012xx", "012", true},
		{"xxxx012", "012", true},
		{"012xxxxx"[1:], "012", false},
		{"xxxxx012"[:7], "012", false},
		// 4-byte needle
		{"xxxxxxxx", "0123", false},
		{"0123xxxx", "0123", true},
		{"xx0123xx", "0123", true},
		{"xxxx0123", "0123", true},
		{"0123xxxxx"[1:], "0123", false},
		{"xxxxx0123"[:8], "0123", false},
		// 5-7-byte needle
		{"xxxxxxxxx", "01234", false},
		{"01234xxxx", "01234", true},
		{"xx01234xx", "01234", true},
		{"xxxx01234", "01234", true},
		{"01234xxxxx"[1:], "01234", false},
		{"xxxxx01234"[:9], "01234", false},
		// 8-byte needle
		{"xxxxxxxxxxxx", "01234567", false},
		{"01234567xxxx", "01234567", true},
		{"xx01234567xx", "01234567", true},
		{"xxxx01234567", "01234567", true},
		{"01234567xxxxx"[1:], "01234567", false},
		{"xxxxx01234567"[:12], "01234567", false},
		// 9-15-byte needle
		{"xxxxxxxxxxxxx", "012345678", false},
		{"012345678xxxx", "012345678", true},
		{"xx012345678xx", "012345678", true},
		{"xxxx012345678", "012345678", true},
		{"012345678xxxxx"[1:], "012345678", false},
		{"xxxxx012345678"[:13], "012345678", false},
		// 16-byte needle
		{"xxxxxxxxxxxxxxxxxxxx", "0123456789ABCDEF", false},
		{"0123456789ABCDEFxxxx", "0123456789ABCDEF", true},
		{"xx0123456789ABCDEFxx", "0123456789ABCDEF", true},
		{"xxxx0123456789ABCDEF", "0123456789ABCDEF", true},
		{"0123456789ABCDEFxxxxx"[1:], "0123456789ABCDEF", false},
		{"xxxxx0123456789ABCDEF"[:20], "0123456789ABCDEF", false},
		// 17-31-byte needle
		{"xxxxxxxxxxxxxxxxxxxxx", "0123456789ABCDEFG", false},
		{"0123456789ABCDEFGxxxx", "0123456789ABCDEFG", true},
		{"xx0123456789ABCDEFGxx", "0123456789ABCDEFG", true},
		{"xxxx0123456789ABCDEFG", "0123456789ABCDEFG", true},
		{"0123456789ABCDEFGxxxxx"[1:], "0123456789ABCDEFG", false},
		{"xxxxx0123456789ABCDEFG"[:21], "0123456789ABCDEFG", false},

		// partial match cases
		{"xx01x", "012", false},                             // 3
		{"xx0123x", "01234", false},                         // 5-7
		{"xx01234567x", "012345678", false},                 // 9-15
		{"xx0123456789ABCDEFx", "0123456789ABCDEFG", false}, // 17-31, issue 15679
		// 2 byte needle, 16byte haystack
		{"xxxxxxxxxxxxxxxx", "01", false},
		{"01xxxxxxxxxxxxxx", "01", true},
		{"xx01xxxxxxxxxxxx", "01", true},
		{"xxxxxxxxxxxxx01x", "01", true},
		{"xxxxxxxxxxxxxxx01xxxxxxx", "01", true},
		{"01xxxxxxxxxxxxxxx"[1:], "01", false},
		// 3 byte needle, 32byte haystack
		{"xyyyyyyyyyyyyyyyyxxxxxxxxxxxxxxx", "yyy", true},
		// 5 bytes needle, 21byte haystack
		{"xxxxxxxxxxxxxxxxxxxxx", "01234", false},
		{"01234xxxxxxxxxxxxxxxx", "01234", true},
		{"xx01234xxxxxxxxxxxxxx", "01234", true},
		{"xxxxxxxxxxx01234xxxxx", "01234", true},
		{"xxxxxxxxxxx01x34xxxxx", "01234", false},
		{"0101x340123401234xxxx", "01234", true},
		// fuzzed cases
		{"000", "0\x00", false},
		{"00000000000000000", "0`", false},
		{"0000", "\x00\x00\x00", false},
	}

	for _, ct := range containsTests {
		if ContainsFold(ct.str, ct.substr) != ct.expected {
			t.Errorf("Contains(%s, %s) = %v, want %v",
				ct.str, ct.substr, !ct.expected, ct.expected)
		}
		if idx, goIdx := IndexFold(ct.str, ct.substr), indexFoldGo([]byte(ct.str), []byte(ct.substr)); idx != goIdx {
			t.Errorf("IndexFold(%s, %s) = %v, want %v",
				ct.str, ct.substr, idx, goIdx)
		}
	}
}

func TestEqualFold(t *testing.T) {
	equalFoldTests := []struct {
		s, t string
		out  bool
	}{
		{"", "", true},
		{"abc", "abc", true},
		{"ABcd", "ABcd", true},
		{"123abc", "123ABC", true},
		{"abc", "xyz", false},
		{"abc", "XYZ", false},
		{"abcdefghijk", "abcdefghijX", false},
		{"1", "2", false},
		{"utf-8", "US-ASCII", false},
		{"hello", "Hello", true},
		{"oh hello there!!", "oh hello there!!", true},
		{"oh hello there!!", "oh HELLO there!!", true},
		{"oh hello there!!", "oh HELLO there !", false},
		{"oh hello there!! friend!", "oh HELLO there!! FRIEND!", true},
	}

	for _, tt := range equalFoldTests {
		if out := EqualFold(tt.s, tt.t); out != tt.out {
			t.Errorf("EqualFold(%#q, %#q) = %v, want %v", tt.s, tt.t, out, tt.out)
		}
		if out := EqualFold(tt.t, tt.s); out != tt.out {
			t.Errorf("EqualFold(%#q, %#q) = %v, want %v", tt.t, tt.s, out, tt.out)
		}
	}
}

func indexBitGo(s []byte, mask byte) int {
	s = s[:len(s):len(s)]

	mask32 := uint32(mask)
	mask32 |= mask32 << 8
	mask32 |= mask32 << 16

	// use all go tricks to make this fast
	for len(s) >= 8 {
		first32 := uint32(s[0]) | uint32(s[1])<<8 | uint32(s[2])<<16 | uint32(s[3])<<24
		second32 := uint32(s[4]) | uint32(s[5])<<8 | uint32(s[6])<<16 | uint32(s[7])<<24
		if (first32|second32)&mask32 != 0 {
			break
		}
		s = s[8:]
	}

	for i, b := range s {
		if b&mask != 0 {
			return i
		}
	}
	return -1
}

func isAsciiGo(s []byte) bool {
	return indexBitGo(s, 0x80) == -1
}

func indexFoldGo(s []byte, substr []byte) int {
	if len(substr) == 0 {
		return 0
	} else if len(substr) > len(s) {
		return -1
	}

	first := substr[0]
	complement := first
	if first >= 'A' && first <= 'Z' {
		complement += 0x20
	} else if first >= 'a' && first <= 'z' {
		complement -= 0x20
	}

	for i, b := range s[:len(s)-len(substr)+1] {
		if b == first || b == complement {
			prefix := s[i:]
			if len(prefix) < len(substr) {
				continue
			}
			if bytes.EqualFold(prefix[:len(substr)], substr) {
				return i
			}
		}
	}
	return -1
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

func BenchmarkIndexBit(b *testing.B) {
	for _, n := range []int{1, 7, 15, 44, 100, 1000} {
		asciiBuf := makeASCII(n)
		idx := rand.Intn(n)
		asciiBuf[idx] |= 0x80

		asciiStr := string(asciiBuf)

		b.Run(fmt.Sprintf("go-%d", n), func(b *testing.B) {
			b.SetBytes(int64(len(asciiStr)))
			for i := 0; i < b.N; i++ {
				indexBitGo(asciiBuf, 0x80)
			}
		})

		b.Run(fmt.Sprintf("simd-%d", n), func(b *testing.B) {
			b.SetBytes(int64(len(asciiStr)))
			for i := 0; i < b.N; i++ {
				IndexBit(asciiStr, 0x80)
			}
		})
	}
}

func BenchmarkAsciiEqualFold(b *testing.B) {
	for _, n := range []int{1, 7, 15, 44, 100, 1000} {
		asciiBuf := makeASCII(n)
		s1 := string(asciiBuf)

		// try to flip as least one byte
		for k := 0; k < 3; k++ {
			idx := rand.Intn(n)
			if unicode.IsUpper(rune(asciiBuf[idx])) {
				asciiBuf[idx] = byte(unicode.ToLower(rune(asciiBuf[idx])))
			} else if unicode.IsLower(rune(asciiBuf[idx])) {
				asciiBuf[idx] = byte(unicode.ToUpper(rune(asciiBuf[idx])))
			}
		}
		s2 := string(asciiBuf)

		b.Run(fmt.Sprintf("go-%d", n), func(b *testing.B) {
			b.SetBytes(int64(len(s1)))
			for i := 0; i < b.N; i++ {
				strings.EqualFold(s1, s2)
			}
		})

		b.Run(fmt.Sprintf("segment-%d", n), func(b *testing.B) {
			b.SetBytes(int64(len(s1)))
			for i := 0; i < b.N; i++ {
				segAscii.EqualFoldString(s1, s2)
			}
		})

		b.Run(fmt.Sprintf("simd-%d", n), func(b *testing.B) {
			b.SetBytes(int64(len(s1)))
			for i := 0; i < b.N; i++ {
				EqualFold(s1, s2)
			}
		})
	}
}

func BenchmarkAsciiIndexFold(b *testing.B) {
	for _, n := range []int{1, 7, 15, 44, 100, 1000} {
		asciiBuf := makeASCII(n)
		s1 := string(asciiBuf)

		// try to flip as least one byte
		for k := 0; k < 3; k++ {
			idx := rand.Intn(n)
			if unicode.IsUpper(rune(asciiBuf[idx])) {
				asciiBuf[idx] = byte(unicode.ToLower(rune(asciiBuf[idx])))
			} else if unicode.IsLower(rune(asciiBuf[idx])) {
				asciiBuf[idx] = byte(unicode.ToUpper(rune(asciiBuf[idx])))
			}
		}

		s2 := string(asciiBuf[rand.Intn(n):])

		b1 := []byte(s1)
		b2 := []byte(s2)

		b.Run(fmt.Sprintf("go-%d", n), func(b *testing.B) {
			b.SetBytes(int64(len(s1)))
			for i := 0; i < b.N; i++ {
				indexFoldGo(b1, b2)
			}
		})

		b.Run(fmt.Sprintf("simd-%d", n), func(b *testing.B) {
			b.SetBytes(int64(len(s1)))
			for i := 0; i < b.N; i++ {
				IndexFold(s1, s2)
			}
		})
	}
}

func FuzzEqualFold(f *testing.F) {
	f.Add("01234567", "01234567")
	f.Add("abcd", "ABCD")
	f.Add("EqualFold", "equalFold")

	f.Fuzz(func(t *testing.T, in1, in2 string) {
		if !IsASCII(in1) || !IsASCII(in2) {
			t.Skip()
		}

		res := EqualFold(in1, in2)
		if res != strings.EqualFold(in1, in2) {
			t.Fatalf("EqualFold(%q, %q) = %v; want %v", in1, in2, res, strings.EqualFold(in1, in2))
		}
	})
}

func FuzzIndexFold(f *testing.F) {
	f.Add("01234567", "01234567")
	f.Add("abcdefghijklmnopqrstuvwxyz01234567890", "klmno")
	f.Add("abcdefghijklmnopqrstuvwxyz01234567890", "12")
	f.Add("abcdefghABCDEFGH01234567890", "H")
	f.Add("000000000000000B0", "B0")
	f.Add("EqualFold", "fold")
	f.Add("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor...", " ELIT")

	f.Fuzz(func(t *testing.T, istr, isubstr string) {
		if !IsASCII(isubstr) {
			t.Skip()
		}

		res := IndexFold(istr, isubstr)

		if goRes := indexFoldGo([]byte(istr), []byte(isubstr)); res != goRes {
			t.Fatalf("IndexFold(%q, %q) = %v; want %v", istr, isubstr, res, goRes)
		}
	})
}
