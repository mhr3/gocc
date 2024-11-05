package utf8

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

var valid1k = bytes.Repeat([]byte("0123456789日本語日本語日本語日abcdefghijklmnopqrstuvwx"), 16)
var valid1M = bytes.Repeat(valid1k, 1024)
var someutf8 = []byte("\xF4\x8F\xBF\xBF")

type byteRange struct {
	Low  byte
	High byte
}

func one(b byte) byteRange {
	return byteRange{b, b}
}

func genExamples(current string, ranges []byteRange) []string {
	if len(ranges) == 0 {
		return []string{string(current)}
	}
	r := ranges[0]
	var all []string

	elements := []byte{r.Low, r.High}

	mid := (r.High + r.Low) / 2
	if mid != r.Low && mid != r.High {
		elements = append(elements, mid)
	}

	for _, x := range elements {
		s := current + string(x)
		all = append(all, genExamples(s, ranges[1:])...)
		if x == r.High {
			break
		}
	}
	return all
}

func TestValid(t *testing.T) {
	var examples = []string{
		// Tests copied from the stdlib
		"",
		"a",
		"abc",
		"Ж",
		"ЖЖ",
		"брэд-ЛГТМ",
		"☺☻☹",

		// overlong
		"\xE0\x80",
		// unfinished continuation
		"aa\xE2",

		string([]byte{66, 250}),

		string([]byte{66, 250, 67}),

		"a\uFFFDb",

		"\xF4\x8F\xBF\xBF", // U+10FFFF

		"\xF4\x90\x80\x80", // U+10FFFF+1; out of range
		"\xF7\xBF\xBF\xBF", // 0x1FFFFF; out of range

		"\xFB\xBF\xBF\xBF\xBF", // 0x3FFFFFF; out of range

		"\xc0\x80",     // U+0000 encoded in two bytes: incorrect
		"\xed\xa0\x80", // U+D800 high surrogate (sic)
		"\xed\xbf\xbf", // U+DFFF low surrogate (sic)

		// valid at boundary
		strings.Repeat("a", 32+28) + "☺☻☹",
		strings.Repeat("a", 32+29) + "☺☻☹",
		strings.Repeat("a", 32+30) + "☺☻☹",
		strings.Repeat("a", 32+31) + "☺☻☹",
		// invalid at boundary
		strings.Repeat("a", 32+31) + "\xE2a",
		strings.Repeat("a", 14) + "☺" + strings.Repeat("a", 13) + "\xE2",

		// same inputs as benchmarks
		"0123456789",
		"日本語日本語日本語日",
		"\xF4\x8F\xBF\xBF",

		// bugs found with fuzzing
		"0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000\xc60",
		"000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000\xc300",
		"߀0000000000000000000000000000訨",
		"0000000000000000000000000000000˂00000000000000000000000000000000",
	}

	any := byteRange{0, 0xFF}
	ascii := byteRange{0, 0x7F}
	cont := byteRange{0x80, 0xBF}

	rangesToTest := [][]byteRange{
		{one(0x20), ascii, ascii, ascii},

		// 2-byte sequences
		{one(0xC2)},
		{one(0xC2), ascii},
		{one(0xC2), cont},
		{one(0xC2), {0xC0, 0xFF}},
		{one(0xC2), cont, cont},
		{one(0xC2), cont, cont, cont},

		// 3-byte sequences
		{one(0xE1)},
		{one(0xE1), cont},
		{one(0xE1), cont, cont},
		{one(0xE1), cont, cont, ascii},
		{one(0xE1), cont, ascii},
		{one(0xE1), cont, cont, cont},

		// 4-byte sequences
		{one(0xF1)},
		{one(0xF1), cont},
		{one(0xF1), cont, cont},
		{one(0xF1), cont, cont, cont},
		{one(0xF1), cont, cont, ascii},
		{one(0xF1), cont, cont, cont, ascii},

		// overlong
		{{0xC0, 0xC1}, any},
		{{0xC0, 0xC1}, any, any},
		{{0xC0, 0xC1}, any, any, any},
		{one(0xE0), {0x0, 0x9F}, cont},
		{one(0xE0), {0xA0, 0xBF}, cont},
	}

	for _, r := range rangesToTest {
		examples = append(examples, genExamples("", r)...)
	}

	for _, i := range []int{300, 316} {
		d := bytes.Repeat(someutf8, i/len(someutf8))
		examples = append(examples, string(d))
	}

	for _, tt := range examples {
		t.Run(tt, func(t *testing.T) {
			expected := utf8.ValidString(tt)
			assert.Equal(t, expected, Valid(tt))
		})
	}
}

var ascii100000 = strings.Repeat("0123456789", 10000)
var longStringMostlyASCII string // ~100KB, ~97% ASCII
var longStringJapanese string    // ~100KB, non-ASCII

func init() {
	const japanese = "日本語日本語日本語日"
	var b strings.Builder
	for i := 0; b.Len() < 100_000; i++ {
		if i%100 == 0 {
			b.WriteString(japanese)
		} else {
			b.WriteString("0123456789")
		}
	}
	longStringMostlyASCII = b.String()
	longStringJapanese = strings.Repeat(japanese, 100_000/len(japanese))
}

func BenchmarkValidStringTenASCIIChars(b *testing.B) {
	b.Run("std", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			utf8.ValidString("0123456789")
		}
	})

	b.Run("simd", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Valid("0123456789")
		}
	})
}

func BenchmarkValidString100KASCIIChars(b *testing.B) {
	b.Run("std", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			utf8.ValidString(ascii100000)
		}
	})

	b.Run("simd", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Valid(ascii100000)
		}
	})
}

func BenchmarkValidStringTenJapaneseChars(b *testing.B) {
	b.Run("std", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			utf8.ValidString("日本語日本語日本語日")
		}
	})

	b.Run("simd", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Valid("日本語日本語日本語日")
		}
	})
}

func BenchmarkValidStringLongMostlyASCII(b *testing.B) {
	b.Run("std", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			utf8.ValidString(longStringMostlyASCII)
		}
	})

	b.Run("simd", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Valid(longStringMostlyASCII)
		}
	})
}

func BenchmarkValidStringLongJapanese(b *testing.B) {
	b.Run("std", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			utf8.ValidString(longStringJapanese)
		}
	})

	b.Run("simd", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Valid(longStringJapanese)
		}
	})
}

func BenchmarkInvalidStringLong(b *testing.B) {
	invalidLongString := "\xe2" + longStringMostlyASCII

	b.Run("std", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			utf8.ValidString(invalidLongString)
		}
	})

	b.Run("simd", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Valid(invalidLongString)
		}
	})
}
