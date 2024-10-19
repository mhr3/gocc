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
package gocc

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/mhr3/gocc/internal/asm"
	"github.com/mhr3/gocc/internal/cc"
	"github.com/mhr3/gocc/internal/config"
)

// Local translates a C file to Go assembly
type Local struct {
	Arch       *config.Arch
	Clang      *cc.Compiler
	ObjDump    *cc.Disassembler
	Source     string
	FuncSuffix string
	Assembly   string
	Object     string
	GoAssembly string
	GoStub     string
	Package    string
	Options    []string
}

// NewLocal creates a new translator that uses locally installed toolchain
func NewLocal(arch *config.Arch, source, outputDir, suffix, functionSuffix, packageName string, options ...string) (*Local, error) {
	sourceExt := filepath.Ext(source)
	noExtSourcePath := source[:len(source)-len(sourceExt)]
	noExtSourceBase := filepath.Base(noExtSourcePath)
	clang, err := cc.NewCompiler(arch)
	if err != nil {
		return nil, err
	}

	objdump, err := cc.NewDisassembler(arch)
	if err != nil {
		return nil, err
	}

	// If package name is not provided, use the directory name of the output
	if packageName == "" {
		packageName = filepath.Base(outputDir)
	}

	return &Local{
		Arch:       arch,
		Clang:      clang,
		ObjDump:    objdump,
		Source:     source,
		FuncSuffix: functionSuffix,
		Assembly:   fmt.Sprintf("%s.s", noExtSourcePath),
		Object:     fmt.Sprintf("%s.o", noExtSourcePath),
		GoAssembly: filepath.Join(outputDir, fmt.Sprintf("%s%s.s", noExtSourceBase, suffix)),
		GoStub:     filepath.Join(outputDir, fmt.Sprintf("%s%s.go", noExtSourceBase, suffix)),
		Package:    packageName,
		Options:    options,
	}, nil
}

// Translate translates the source file to Go assembly
func (t *Local) Translate() error {
	functions, err := cc.Parse(t.Source)
	if err != nil {
		return err
	}

	// Compile the source file to assembly
	if err := t.Clang.Compile(t.Source, t.Assembly, t.Object, t.Options...); err != nil {
		return err
	}

	// Disassemble the object file and extract machine code
	assembly, err := t.ObjDump.Disassemble(t.Assembly, t.Object)
	if err != nil {
		return err
	}

	foundMapping := false
	// Map the machine code to the assembly one
	for _, v := range assembly {
		assemblyName := v.Name
		idx := slices.IndexFunc(functions, func(cFn asm.Function) bool {
			return assemblyName == cFn.Name
		})
		if idx == -1 {
			// try one more time without the underscore prefix
			assemblyName := strings.TrimPrefix(assemblyName, "_")
			idx = slices.IndexFunc(functions, func(cFn asm.Function) bool {
				return assemblyName == cFn.Name
			})
		}
		if idx == -1 {
			continue
		}
		foundMapping = true

		functions[idx].Consts = v.Consts
		functions[idx].Lines = v.Lines
	}

	_ = t.Close()

	if !foundMapping {
		return errors.New("cannot find mapping to machine code")
	}

	// apply the function suffix, this needs to be done after the mapping
	for i := range functions {
		functions[i].GoFunc.Name += t.FuncSuffix
	}

	// Generate the Go stubs for the functions
	if err := asm.GenerateGoStubs(t.Arch, t.Package, t.GoStub, functions); err != nil {
		return err
	}

	meta := asm.GeneratorMeta{
		ClangVersion: t.Clang.Version(),
		Options:      t.Options,
	}

	return asm.GenerateFile(t.Arch, t.GoAssembly, t.Source, functions, meta)
}

// Output returns the output files as a web result
func (t *Local) Output() (*WebResult, error) {
	goFile, err := os.ReadFile(t.GoStub)
	if err != nil {
		return nil, err
	}

	asmFile, err := os.ReadFile(t.GoAssembly)
	if err != nil {
		return nil, err
	}

	return &WebResult{
		Asm: File{
			Name: t.GoAssembly,
			Body: asmFile,
		},
		Go: File{
			Name: t.GoStub,
			Body: goFile,
		},
	}, nil
}

// Cleanup cleans up the temporary files
func (t *Local) Close() error {
	return errors.Join(
		os.Remove(t.Assembly),
		os.Remove(t.Object),
	)
}
