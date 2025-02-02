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
	"strings"

	"github.com/mhr3/gocc/internal/config"
)

// Compiler represents a C/C++ compiler.
type Compiler struct {
	arch    *config.Arch
	clang   string
	version string
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
func (c *Compiler) Compile(source, assembly, object string, args ...string) error {
	args = append(args,
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
	)
	args = append(args, c.arch.ClangFlags...)

	compileOutput, err := runCommandAndLog(c.clang, append([]string{"-S", "-c", source, "-o", assembly}, args...)...)
	// Compile to assembly first
	if err != nil {
		return err
	}
	if compileOutput != "" {
		fmt.Fprintln(os.Stderr, compileOutput)
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
