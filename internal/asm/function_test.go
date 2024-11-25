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
	"encoding/binary"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mhr3/gocc/internal/config"
)

func TestParamSizes(t *testing.T) {
	testCases := []struct {
		Name         string
		Fn           Function
		ExpectedSize int
	}{
		{
			Name: "3bytes+void",
			Fn: Function{Params: []Param{
				{Type: "char"},
				{Type: "char"},
				{Type: "char"},
			}},
			ExpectedSize: 3,
		},
		{
			Name: "3bytes+ret",
			Fn: Function{Params: []Param{
				{Type: "char"},
				{Type: "char"},
				{Type: "char"},
			},
				Ret: &Param{Type: "char"},
			},
			ExpectedSize: 8,
		},
		{
			Name: "3bytes+int",
			Fn: Function{Params: []Param{
				{Type: "char"},
				{Type: "char"},
				{Type: "char"},
				{Type: "int"},
			}},
			ExpectedSize: 8,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			sz, _ := tc.Fn.ParamsSize(nil)
			assert.Equal(t, tc.ExpectedSize, sz)
		})
	}
}

func TestLineWord(t *testing.T) {
	line := Line{
		Assembly: "vaddps 0x40(%rdx,%rax,4),%ymm3,%ymm3",
		Binary:   []string{"c5", "e4", "58", "5c", "82", "40"},
	}
	assert.Equal(t, "\tLONG $0x5c58e4c5; WORD $0x4082\t// ?                                    // vaddps 0x40(%rdx,%rax,4),%ymm3,%ymm3\n",
		line.Compile(nil))
}

func TestLineByte(t *testing.T) {
	line := Line{
		Assembly:     "ret",
		Disassembled: "RET",
	}
	assert.Equal(t, "\tRET\t// <--                                  // ret\n", line.Compile(nil))
}

func TestLineLabel(t *testing.T) {
	line := Line{
		Labels:       []string{"label"},
		Assembly:     "ret",
		Disassembled: "RET",
	}
	assert.Equal(t, "label:\n\tRET\t// <--                                  // ret\n", line.Compile(nil))
}

func TestLineJumpAMD(t *testing.T) {
	t.Skip()
	line := Line{
		Assembly:     "jmp .LBB0_2",
		Binary:       []string{"e9", "13", "01", "00", "00"},
		Disassembled: "JMP 0x123",
	}
	assert.Equal(t, "\tJMP LBB0_2\n",
		line.Compile(config.AMD64()))
}

func TestLineARM(t *testing.T) {
	t.Skip()
	line := Line{
		Assembly: "mov x29, sp",
		Binary:   []string{"fd", "03", "00", "91"},
	}
	assert.Equal(t, "\tWORD $0x910003fd\t// mov x29, sp\n",
		line.Compile(config.ARM64()))
}

func TestParseConst(t *testing.T) {
	testCases := []struct {
		Name        string
		Const       string
		ExpectedVal string
	}{
		{
			Name:        "zero",
			Const:       `	.zero	16,255`,
			ExpectedVal: "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF",
		},
		{
			Name:        "byte",
			Const:       `	.byte	255`,
			ExpectedVal: "FF",
		},
		{
			Name:        "hword",
			Const:       `	.hword  30`,
			ExpectedVal: "1E00",
		},
		{
			Name:        "int",
			Const:       `	.int	42`,
			ExpectedVal: "2A000000",
		},
		{
			Name:        "ascii",
			Const:       `	.ascii	"\000\377\377\377\001\377\377\377\002\377\377\377\003\377\377\377"`,
			ExpectedVal: "00FFFFFF01FFFFFF02FFFFFF03FFFFFF",
		},
		{
			Name:        "asciz",
			Const:       `	.asciz	"\002\003\000\000\000\000\000\000\000\004\000\000\000\000\000\000\000\000\000\000\000\000\000\000\000\000\003\000\000\000\000"`,
			ExpectedVal: "0203000000000000000400000000000000000000000000000000030000000000",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			for _, arch := range []*config.Arch{config.AMD64(), config.ARM64()} {
				require.True(t, arch.Const.MatchString(tc.Const))

				lines := parseConst(arch, tc.Const)
				assert.NotEmpty(t, lines)

				l := 0
				for _, line := range lines {
					l += line.Size
				}
				assert.Len(t, tc.ExpectedVal, l*2)
				actualVal := linesToHex(lines)
				assert.Equal(t, tc.ExpectedVal, actualVal)
			}
		})
	}
}
func linesToHex(lines []ConstLine) string {
	var val string
	var slc [8]byte
	buf := slc[:]
	for _, line := range lines {
		buf = buf[:8]
		switch line.Size {
		case 1:
			buf[0] = byte(line.Value)
			buf = buf[:1]
		case 2:
			binary.LittleEndian.PutUint16(buf, uint16(line.Value))
			buf = buf[:2]
		case 4:
			binary.LittleEndian.PutUint32(buf, uint32(line.Value))
			buf = buf[:4]
		case 8:
			binary.LittleEndian.PutUint64(buf, uint64(line.Value))
			buf = buf[:8]
		}

		val += hex.EncodeToString(buf)
	}
	return strings.ToUpper(val)
}
