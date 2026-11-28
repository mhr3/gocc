// Copyright 2023 Roman Atachiants
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	fn, err := Parse(testSource)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(fn))
}

func TestParseSplit(t *testing.T) {
	fn, err := Parse("testdata/split_fn.c")
	assert.NoError(t, err)
	assert.Len(t, fn, 1)
}

func TestParseWithStatic(t *testing.T) {
	fn, err := Parse("testdata/static_fn.c")
	assert.Error(t, err)
	assert.Nil(t, fn)
}

// writeSnippet stores C source in a temporary file and returns its path.
func writeSnippet(t *testing.T, source string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "snippet.c")
	require.NoError(t, os.WriteFile(path, []byte(source), 0o644))
	return path
}

func TestParseVariants(t *testing.T) {
	const body = " {\n    return 0;\n}\n"

	type variant struct {
		Name   string
		Padded []bool
	}
	accepted := []struct {
		Name     string
		Source   string
		Declared string // the signature the assembly is declared with
		Variants []variant
		Slots    []string // the frame slot each C parameter is loaded from
		Offsets  []int
		ArgsSize int
	}{
		{
			// The frame is that of the form taking every parameter as []byte,
			// whichever directive names it.
			Name: "declared with the []byte form",
			Source: `
// gocc-variant: IndexString(haystack string, needle []byte, from int) int
// gocc: IndexStrings(haystack, needle string, from int) int
// gocc-variant: Index(haystack, needle []byte, from int) int
long index(char *h, long h_len, char *n, long n_len, long from)` + body,
			Declared: "Index",
			Variants: []variant{{"IndexStrings", []bool{true, true, false}}, {"IndexString", []bool{true, false, false}}},
			Slots:    []string{"haystack_base", "haystack_len", "needle_base", "needle_len", "from"},
			Offsets:  []int{0, 8, 24, 32, 48},
			ArgsSize: 56,
		},
		{
			// A parameter that is a string in every signature has no capacity
			// word, and a slice of another type loses its C capacity too.
			Name: "strings and other slices",
			Source: `
// gocc: Score(weights []float32, text []byte, sep string) int
// gocc-variant: ScoreString(weights []float32, text string, sep string) int
long score(float *w, long w_len, char *t, long t_len, char *s, long s_len)` + body,
			Declared: "Score",
			Variants: []variant{{"ScoreString", []bool{false, true, false}}},
			Slots:    []string{"weights_base", "weights_len", "text_base", "text_len", "sep_base", "sep_len"},
			Offsets:  []int{0, 8, 24, 32, 48, 56},
			ArgsSize: 64,
		},
		{
			// Only the listed slices lose their C capacity. The directive
			// counts on either side of the signature.
			Name: "gocc-omit-cap",
			Source: `
// gocc-omit-cap: a, c
// gocc: Mix(a []float32, b []int64, c []byte, k int) int
long mix(float *a, long a_len, long *b, long b_len, long b_cap, char *c, long c_len, long k)` + body,
			Declared: "Mix",
			Slots:    []string{"a_base", "a_len", "b_base", "b_len", "b_cap", "c_base", "c_len", "k"},
			Offsets:  []int{0, 8, 24, 32, 40, 48, 56, 72},
			ArgsSize: 80,
		},
		{
			Name: "several directives",
			Source: `
// gocc: Dot(a, b, c []int64) int
// gocc-omit-cap: a
// gocc-omit-cap: b c
long dot(long *a, long a_len, long *b, long b_len, long *c, long c_len)` + body,
			Declared: "Dot",
			Slots:    []string{"a_base", "a_len", "b_base", "b_len", "c_base", "c_len"},
			Offsets:  []int{0, 8, 24, 32, 48, 56},
			ArgsSize: 72,
		},
		{
			// A function with variants has no capacities already.
			Name: "gocc-omit-cap with variants",
			Source: `
// gocc: SumString(p string, bias int) int
// gocc-variant: Sum(p []byte, bias int) int
// gocc-omit-cap: p
long sum(char *p, long p_len, long bias)` + body,
			Declared: "Sum",
			Variants: []variant{{"SumString", []bool{true, false}}},
			Slots:    []string{"p_base", "p_len", "bias"},
			Offsets:  []int{0, 8, 24},
			ArgsSize: 32,
		},
		{
			Name: "no directive",
			Source: `
// gocc: Sum(p []byte, bias int) int
long sum(char *p, long p_len, long p_cap, long bias)` + body,
			Declared: "Sum",
			Slots:    []string{"p_base", "p_len", "p_cap", "bias"},
			Offsets:  []int{0, 8, 16, 24},
			ArgsSize: 32,
		},
	}
	for _, tc := range accepted {
		t.Run(tc.Name, func(t *testing.T) {
			functions, err := Parse(writeSnippet(t, tc.Source))
			require.NoError(t, err)
			require.Len(t, functions, 1)
			function := functions[0]

			assert.Equal(t, tc.Declared, function.GoFunc.Name)
			var variants []variant
			for _, v := range function.Variants {
				variants = append(variants, variant{v.Name, v.Padded})
			}
			assert.Equal(t, tc.Variants, variants)

			var slots []string
			for i := range function.Params {
				slots = append(slots, function.Params[i].FrameName())
			}
			assert.Equal(t, tc.Slots, slots)
			argsSize, offsets := function.ParamsSize(nil)
			assert.Equal(t, tc.Offsets, offsets)
			assert.Equal(t, tc.ArgsSize, argsSize)
		})
	}

	rejected := []struct {
		Name     string
		Source   string
		Expected string
	}{
		{
			Name: "slice of another type",
			Source: `
// gocc: Sum(p []int64) int
// gocc-variant: SumString(p string) int
long sum(long *p, long p_len)` + body,
			Expected: `param "p": expected []int64, got string`,
		},
		{
			Name: "other parameter changed",
			Source: `
// gocc: Sum(p []byte, bias int) int
// gocc-variant: SumString(p string, bias int32) int
long sum(char *p, long p_len, long bias)` + body,
			Expected: `param "bias": expected int, got int32`,
		},
		{
			Name: "parameter count",
			Source: `
// gocc: Sum(p []byte, bias int) int
// gocc-variant: SumString(p string) int
long sum(char *p, long p_len, long bias)` + body,
			Expected: "expected 2 parameters, got 1",
		},
		{
			Name: "result changed",
			Source: `
// gocc: Sum(p []byte) int
// gocc-variant: SumString(p string) int32
long sum(char *p, long p_len)` + body,
			Expected: "results differ",
		},
		{
			Name: "nothing replaced",
			Source: `
// gocc: Sum(p []byte) int
// gocc-variant: Total(p []byte) int
long sum(char *p, long p_len)` + body,
			Expected: "takes no parameter of Sum as a string or a []byte instead",
		},
		{
			// Neither frame can hold the arguments of the other.
			Name: "no signature with every []byte",
			Source: `
// gocc: Left(a string, b []byte) int
// gocc-variant: Right(a []byte, b string) int
long both(char *a, long a_len, char *b, long b_len)` + body,
			Expected: "one of the signatures has to take all of a, b as []byte",
		},
		{
			Name: "C function takes the capacity",
			Source: `
// gocc: Sum(p []byte, bias int) int
// gocc-variant: SumString(p string, bias int) int
long sum(char *p, long p_len, long p_cap, long bias)` + body,
			Expected: "takes a slice as a pointer and a length, without the capacity",
		},
		{
			Name: "capacity missing without a directive",
			Source: `
// gocc: Sum(p []byte) int
long sum(char *p, long p_len)` + body,
			Expected: "too few parameters: a slice is a pointer, a length and a capacity; list it in gocc-omit-cap",
		},
		{
			Name: "capacity missing for a slice that is not listed",
			Source: `
// gocc: Mix(a []int64, b []int64) int
// gocc-omit-cap: a
long mix(long *a, long a_len, long *b, long b_len)` + body,
			Expected: "too few parameters",
		},
		{
			Name: "listed and the C function takes the capacity",
			Source: `
// gocc: Sum(p []int64, bias int) int
// gocc-omit-cap: p
long sum(long *p, long p_len, long p_cap, long bias)` + body,
			Expected: "the C function takes the capacity of a slice listed in gocc-omit-cap",
		},
		{
			Name: "listed parameter that does not exist",
			Source: `
// gocc: Sum(p []int64) int
// gocc-omit-cap: q
long sum(long *p, long p_len)` + body,
			Expected: `gocc-omit-cap: Sum has no parameter "q"`,
		},
		{
			Name: "listed parameter that is not a slice",
			Source: `
// gocc: Find(p []int64, s string) int
// gocc-omit-cap: p, s
long find(long *p, long p_len, char *s, long s_len)` + body,
			Expected: `gocc-omit-cap: param "s" is a string, not a slice`,
		},
		{
			Name: "empty list",
			Source: `
// gocc: Sum(p []int64) int
// gocc-omit-cap:
long sum(long *p, long p_len)` + body,
			Expected: "sum: gocc-omit-cap names no parameter",
		},
		{
			Name: "gocc-omit-cap on a function without a signature",
			Source: `
// gocc-omit-cap: p
long sum(long *p, long p_len)` + body + `
// gocc: Total(p []int64) int
long total(long *p, long p_len, long p_cap)` + body,
			Expected: "line 2: gocc-omit-cap on a function without a gocc signature",
		},
		{
			Name: "unnamed parameter of a wrapper",
			Source: `
// gocc: Sum(p []byte) int
// gocc-variant: SumString(_ string) int
long sum(char *p, long p_len)` + body,
			Expected: "SumString: parameter 1 has to be named",
		},
		{
			// The variant must not end up on the next annotated function.
			Name: "function without a signature",
			Source: `
// gocc-variant: SumString(p string) int
long sum(char *p, long p_len)` + body + `
// gocc: Total(p []byte) int
long total(char *p, long p_len, long p_cap)` + body,
			Expected: "line 2: gocc-variant on a function without a gocc signature",
		},
	}
	for _, tc := range rejected {
		t.Run(tc.Name, func(t *testing.T) {
			_, err := Parse(writeSnippet(t, tc.Source))
			assert.ErrorContains(t, err, tc.Expected)
		})
	}
}
