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
			Name: "no variants",
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
			// Only a function with variants may leave the capacity out.
			Name: "capacity missing without variants",
			Source: `
// gocc: Sum(p []byte) int
long sum(char *p, long p_len)` + body,
			Expected: "too few parameters",
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
