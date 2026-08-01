package gocc

import (
	"testing"

	"github.com/mhr3/gocc/internal/config"
	"github.com/stretchr/testify/assert"
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
