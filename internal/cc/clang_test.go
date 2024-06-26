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

// find looks for a particular executable in the system
func find(versions []string) (string, error) {
	for _, v := range versions {
		if _, err := exec.LookPath(v); err == nil {
			return v, nil
		}
	}

	return "", fmt.Errorf("gocc: '%s' executable not found)", versions[0])
}
