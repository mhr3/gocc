package gocc

import (
	"testing"

	"github.com/mhr3/gocc/internal/asm"
	"github.com/mhr3/gocc/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInternalFunctionCompilerOptions(t *testing.T) {
	tests := []struct {
		name string
		arch *config.Arch
		want []string
	}{
		{
			name: "amd64",
			arch: config.AMD64(),
			want: []string{"-O3", "-mno-stackrealign", "-fomit-frame-pointer"},
		},
		{
			name: "arm64",
			arch: config.ARM64(),
			want: []string{"-O3", "-ffixed-x25", "-ffixed-x26", "-mno-stackrealign", "-fomit-frame-pointer"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := []string{"-O3"}
			translator := &Local{
				Arch:                  tt.arch,
				Options:               base,
				WithInternalFunctions: true,
			}

			assert.Equal(t, tt.want, translator.compilerOptions())
			assert.Equal(t, []string{"-O3"}, base)
		})
	}
}

func TestInternalFunctionCompilerOptionsAreNotDuplicated(t *testing.T) {
	translator := &Local{
		Arch: config.AMD64(),
		Options: []string{
			"-mno-stackrealign",
			"-fomit-frame-pointer",
		},
		WithInternalFunctions: true,
	}

	assert.Equal(t, translator.Options, translator.compilerOptions())
}

func TestInternalFunctionCallsRequireExplicitOptIn(t *testing.T) {
	functions := []asm.Function{
		{
			Name:  "entry",
			Lines: []asm.Line{{Disassembled: "CALL helper<>(SB)"}},
		},
		{
			Name:     "helper",
			Internal: true,
		},
	}

	err := validateInternalFunctionOptIn(functions, false)
	require.EqualError(t, err,
		`C function "entry" calls "helper"; pass --with-internal-functions to enable C helper calls`)
	require.NoError(t, validateInternalFunctionOptIn(functions, true))
}

func TestInternalFunctionOptInNotRequiredWithoutEmittedCalls(t *testing.T) {
	functions := []asm.Function{{Name: "entry", Lines: []asm.Line{{Disassembled: "RET"}}}}

	require.NoError(t, validateInternalFunctionOptIn(functions, false))
}
