// Copyright 2022 gorse Project Authors
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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/mhr3/gocc/internal/config"
)

// Compiler represents a C/C++ compiler.
type Compiler struct {
	arch    *config.Arch
	clang   string
	version string
}

var arm64X29Register = regexp.MustCompile(`\b([xw])29\b`)
var arm64X20Register = regexp.MustCompile(`\b([xw])20\b`)

func (c *Compiler) remapArm64FrameRegister(assembly string, args []string) error {
	if c.arch.Name != "arm64" || !slicesContains(args, "-fomit-frame-pointer") ||
		!slicesContains(args, "-mno-stackrealign") {
		return nil
	}
	contents, err := os.ReadFile(assembly)
	if err != nil {
		return err
	}
	// Go owns R29 for its frame chain. Clang's ARM64 backend can still use X29
	// as a general callee-saved register after omitting the C frame pointer, so
	// move that allocation to X25. Callers that enable this rewrite reserve X25
	// from Clang, leaving it available as a normal (non-Go-reserved) register.
	contents = arm64X29Register.ReplaceAll(contents, []byte("${1}25"))
	// cmd/asm uses R20 while establishing larger ARM64 frames, before the C
	// body has a chance to preserve the incoming C-ABI value. X26 is also
	// reserved by callers of this mode, so move Clang's X20 allocation there.
	contents = arm64X20Register.ReplaceAll(contents, []byte("${1}26"))
	return os.WriteFile(assembly, contents, 0o644)
}

func slicesContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// NewCompiler creates a new compiler.
func NewCompiler(arch *config.Arch) (*Compiler, error) {
	var version string

	clang, err := config.FindClang()
	if err != nil {
		return nil, err
	}

	versionOutput, err := runCommand(clang, "--version")
	if err != nil {
		return nil, err
	}

	if parts := strings.SplitN(versionOutput, "\n", 2); len(parts) > 0 {
		version = strings.TrimSpace(parts[0])
	}

	return &Compiler{
		arch:    arch,
		clang:   clang,
		version: version,
	}, nil
}

func (c *Compiler) Version() string {
	return c.version
}

// compile compiles the C source file to assembly and then to object.
func (c *Compiler) Compile(source, assembly, object string, compilerArgs ...string) error {
	defaults := []string{
		"-mno-red-zone",
		"-mstackrealign",
		"-mllvm",
		"-inline-threshold=1000",
		"-fno-asynchronous-unwind-tables",
		"-fno-exceptions",
		"-fno-rtti",
		"-fno-jump-tables",
		"-ffast-math",
		"-Wno-unused-command-line-argument",
	}
	// User options come last so they can intentionally override gocc's
	// conservative defaults (for example -mno-stackrealign on ARM64).
	args := append(defaults, c.arch.ClangFlags...)
	args = append(args, compilerArgs...)
	// Generated functions cannot rely on a hosted C runtime. Keep this after
	// user options so -fhosted cannot accidentally enable libc assumptions.
	args = append(args, "-ffreestanding")

	compileOutput, err := runCommandAndLog(c.clang, append([]string{"-S", "-c", source, "-o", assembly}, args...)...)
	// Compile to assembly first
	if err != nil {
		return err
	}
	if compileOutput != "" {
		fmt.Fprintln(os.Stderr, compileOutput)
	}
	if err := c.remapArm64FrameRegister(assembly, args); err != nil {
		return err
	}

	// Use clang to compile to object
	objOutput, err := runCommandAndLog(c.clang, append([]string{"-c", assembly, "-o", object}, args...)...)
	if err != nil {
		return err
	}
	if objOutput != "" {
		fmt.Fprintln(os.Stderr, objOutput)
	}

	return nil
}

// runCommandAndLog runs a command and extract its output.
func runCommandAndLog(name string, args ...string) (string, error) {
	cmd := fmt.Sprintf("%s %s", name, strings.Join(args, " "))
	fmt.Printf("Running %q\n", cmd)

	return runCommand(name, args...)
}

func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err == nil {
		return string(output), nil
	}

	switch {
	case output != nil:
		return "", errors.New(string(output))
	default:
		return "", err
	}
}
