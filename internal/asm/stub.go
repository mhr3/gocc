package asm

import (
	"fmt"
	"os"
	"strings"

	"github.com/mhr3/gocc/internal/config"
)

// GenerateGoStubs generates Go stubs for the functions.
func GenerateGoStubs(arch *config.Arch, pkg, output string, functions []Function) error {
	var builder strings.Builder

	builder.WriteString(arch.BuildTags)
	builder.WriteString("\n// Code generated by gocc -- DO NOT EDIT.\n\n")
	fmt.Fprintf(&builder, "package %s\n\n", pkg)

	usesUnsafe := false
	for _, function := range functions {
		if function.GoFunc.Expr == nil {
			continue
		}
		function.GoFunc.ForEachParam(func(_, typ string) {
			if typ == "unsafe.Pointer" {
				usesUnsafe = true
			}
		})
	}
	if usesUnsafe {
		builder.WriteString("import \"unsafe\"\n")
	}

	for _, function := range functions {
		builder.WriteString(function.String())
	}

	// write file
	f, err := os.Create(output)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		if err = f.Close(); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}(f)
	_, err = f.WriteString(builder.String())
	return err
}
