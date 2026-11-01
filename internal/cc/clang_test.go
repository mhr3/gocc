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
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"testing"

	"github.com/mhr3/gocc/internal/config"
	"github.com/stretchr/testify/assert"
)

const testSource = "../../example/matmul_avx2.c"

func TestCompiler(t *testing.T) {
	echo, err := find([]string{"echo", "cmd /c echo", "cmd"})
	assert.NoError(t, err)
	assert.NotEmpty(t, echo)

	arch, err := config.For(runtime.GOARCH)
	assert.NoError(t, err)

	compiler := Compiler{
		clang: echo,
		arch:  arch,
	}

	assert.NoError(t, compiler.Compile(testSource, "temp.s", "temp.o"))
}

func TestRemapArm64FrameRegisterRequiresReservedRegisters(t *testing.T) {
	const source = "\tmov\tx29, sp\n\tadd\tx20, x25, #1\n"
	compiler := Compiler{arch: config.ARM64()}

	write := func() string {
		path := t.TempDir() + "/remap.s"
		assert.NoError(t, os.WriteFile(path, []byte(source), 0o644))
		return path
	}

	// Omitting the frame pointer alone must not rewrite X29 into a register
	// that Clang may still have allocated itself.
	path := write()
	assert.NoError(t, compiler.remapArm64FrameRegister(path, []string{"-fomit-frame-pointer", "-mno-stackrealign"}))
	contents, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, source, string(contents))

	path = write()
	assert.NoError(t, compiler.remapArm64FrameRegister(path, arm64RegisterRemapFlags))
	contents, err = os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, "\tmov\tx25, sp\n\tadd\tx26, x25, #1\n", string(contents))
}

// find looks for a particular executable in the system
func find(versions []string) (string, error) {
	for _, v := range versions {
		if _, err := exec.LookPath(v); err == nil {
			return v, nil
		}
	}

	return "", fmt.Errorf("gocc: '%s' executable not found)", versions[0])
}
