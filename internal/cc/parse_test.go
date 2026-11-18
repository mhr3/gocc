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
	path := filepath.Join(t.TempDir(), "alias.c")
	require.NoError(t, os.WriteFile(path, []byte(source), 0o644))
	return path
}

func TestParseAlias(t *testing.T) {
	const body = " {\n    return 0;\n}\n"

	t.Run("pads only the replaced parameters", func(t *testing.T) {
		functions, err := Parse(writeSnippet(t, `
// gocc-alias: IndexString(haystack string, needle []byte, from int) int
// gocc: Index(haystack, needle []byte, from int) int
// gocc-alias: IndexStrings(haystack, needle string, from int) int
long index(char *h, long h_len, long h_cap, char *n, long n_len, long n_cap, long from)`+body))
		require.NoError(t, err)
		require.Len(t, functions, 1)

		aliases := functions[0].Aliases
		require.Len(t, aliases, 2)
		assert.Equal(t, "IndexString", aliases[0].Name)
		assert.Equal(t, []bool{true, false, false}, aliases[0].Padded)
		assert.Equal(t, "IndexStrings", aliases[1].Name)
		assert.Equal(t, []bool{true, true, false}, aliases[1].Padded)
	})

	rejected := []struct {
		Name     string
		Source   string
		Expected string
	}{
		{
			Name: "slice of another type",
			Source: `
// gocc: Sum(p []int64) int
// gocc-alias: SumString(p string) int
long sum(long *p, long p_len, long p_cap)` + body,
			Expected: `param "p": expected []int64, got string`,
		},
		{
			Name: "string to slice",
			Source: `
// gocc: LenString(s string) int
// gocc-alias: Len(s []byte) int
long len(char *s, long s_len)` + body,
			Expected: `param "s": expected string, got []byte`,
		},
		{
			Name: "other parameter changed",
			Source: `
// gocc: Sum(p []byte, bias int) int
// gocc-alias: SumString(p string, bias int32) int
long sum(char *p, long p_len, long p_cap, long bias)` + body,
			Expected: `param "bias": expected int, got int32`,
		},
		{
			Name: "parameter count",
			Source: `
// gocc: Sum(p []byte, bias int) int
// gocc-alias: SumString(p string) int
long sum(char *p, long p_len, long p_cap, long bias)` + body,
			Expected: "expected 2 parameters, got 1",
		},
		{
			Name: "result changed",
			Source: `
// gocc: Sum(p []byte) int
// gocc-alias: SumString(p string) int32
long sum(char *p, long p_len, long p_cap)` + body,
			Expected: "results differ",
		},
		{
			Name: "nothing replaced",
			Source: `
// gocc: Sum(p []byte) int
// gocc-alias: Total(p []byte) int
long sum(char *p, long p_len, long p_cap)` + body,
			Expected: "no []byte parameter",
		},
		{
			// The alias must not end up on the next annotated function.
			Name: "function without a signature",
			Source: `
// gocc-alias: SumString(p string) int
long sum(char *p, long p_len, long p_cap)` + body + `
// gocc: Total(p []byte) int
long total(char *p, long p_len, long p_cap)` + body,
			Expected: "line 2: gocc-alias on a function without a gocc signature",
		},
	}
	for _, tc := range rejected {
		t.Run(tc.Name, func(t *testing.T) {
			_, err := Parse(writeSnippet(t, tc.Source))
			assert.ErrorContains(t, err, tc.Expected)
		})
	}
}
