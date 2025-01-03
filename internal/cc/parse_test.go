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
	"testing"

	"github.com/stretchr/testify/assert"
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
