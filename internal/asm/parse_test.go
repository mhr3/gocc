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

package asm

import (
	"os"
	"testing"

	"github.com/mhr3/gocc/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestParseAssembly(t *testing.T) {
	fn, err := ParseAssembly(config.AMD64(), "../../fixtures/test_avx.s")
	assert.NoError(t, err)
	assert.Len(t, fn, 1)
	assert.Len(t, fn[0].Consts, 1)
	assert.Len(t, fn[0].Lines, 135)
	for _, line := range fn[0].Lines {
		assert.NotEmpty(t, line.Assembly)
		assert.Empty(t, line.Binary)
	}
}

func TestParseClangObjectDumpAmd64(t *testing.T) {
	fn, err := ParseAssembly(config.AMD64(), "../../fixtures/test_avx.s")
	assert.NoError(t, err)

	dump, err := os.ReadFile("../../fixtures/test_avx.o.txt")
	assert.NoError(t, err)

	assert.NoError(t, ParseClangObjectDump(config.AMD64(), string(dump), fn, nil))
	assert.Len(t, fn, 1)
	assert.Len(t, fn[0].Consts, 1)
	assert.Len(t, fn[0].Lines, 135)
	for _, line := range fn[0].Lines {
		assert.NotEmpty(t, line.Assembly)
		assert.NotEmpty(t, line.Binary)
	}
}

func TestParseClangObjectDumpArm64(t *testing.T) {
	fn, err := ParseAssembly(config.ARM64(), "../../fixtures/test_neon.s")
	assert.NoError(t, err)

	dump, err := os.ReadFile("../../fixtures/test_neon.o.txt")
	assert.NoError(t, err)

	assert.NoError(t, ParseClangObjectDump(config.ARM64(), string(dump), fn, nil))
	assert.Len(t, fn, 1)
	assert.Len(t, fn[0].Consts, 0)
	assert.Len(t, fn[0].Lines, 65)
	for _, line := range fn[0].Lines {
		assert.NotEmpty(t, line.Assembly)
		assert.NotEmpty(t, line.Binary)
	}
}

func TestParseGoObjectDump(t *testing.T) {
	t.Skip("Skipping test")
	fn, err := ParseAssembly(config.AMD64(), "../../fixtures/test_avx.s")
	assert.NoError(t, err)

	dump, err := os.ReadFile("../../fixtures/test_avx.o.go-txt")
	assert.NoError(t, err)

	assert.NoError(t, ParseGoObjectDump(config.AMD64(), string(dump), fn))
	assert.Len(t, fn, 1)
	assert.Len(t, fn[0].Consts, 1)
	assert.Len(t, fn[0].Lines, 135)
	for _, line := range fn[0].Lines {
		assert.NotEmpty(t, line.Assembly)
		assert.NotEmpty(t, line.Binary)
	}
}
