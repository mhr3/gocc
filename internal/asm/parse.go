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

package asm

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/kelindar/gocc/internal/config"
)

type Plan9Decoder interface {
	DecodeInstruction(symName string, binary []string) (string, error)
}

// ParseAssembly parses the assembly file and returns a list of functions
func ParseAssembly(arch *config.Arch, path string) ([]Function, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		if err = file.Close(); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}(file)

	var (
		functions    = make([]Function, 0, 8)
		current      *Function
		consts       []Const
		constant     *Const
		functionName string
		labelName    string
	)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		switch {

		// Handle constant lines and attach them to the current label
		case arch.Const.MatchString(line):
			constant.Lines = append(constant.Lines, parseConst(arch, line))

		// Skip attributes and comment lines
		case arch.Attribute.MatchString(line):
			continue
		case arch.Comment.MatchString(line):
			continue

		// Handle assembly labels. We could potentially have multiple labels per line if
		// compiler decides to generate no-op instructions.
		case arch.SourceLabel.MatchString(line):
			labelName = strings.Split(line, ":")[0]
			labelName = strings.TrimLeft(labelName, ".")
			// If we have a constant, attach it to the current function
			if constant != nil && len(constant.Lines) > 0 {
				consts = append(consts, *constant)
			}
			constant = &Const{Label: labelName} // reset the current constant
			switch {
			case current == nil: // No function yet
			case len(current.Lines) == 0:
				current.Lines = append(current.Lines, Line{Labels: []string{labelName}})
			case current.Lines[len(current.Lines)-1].Assembly == "": // Previous line was a label
				current.Lines[len(current.Lines)-1].Labels = append(current.Lines[len(current.Lines)-1].Labels, labelName)
			default:
				current.Lines = append(current.Lines, Line{Labels: []string{labelName}})
			}

		// Handle assembly function name
		case arch.Function.MatchString(line):
			functionName = strings.SplitN(line, ":", 2)[0]
			functions = append(functions, Function{
				Name:  functionName,
				Lines: make([]Line, 0),
			})
			current = &functions[len(functions)-1]
			labelName = "" // Reset current label

			// If we have a constant, attach it to the current function
			if constant != nil && len(constant.Lines) > 0 {
				consts = append(consts, *constant)
				constant = nil
				current.Consts = append(current.Consts, consts...)
			}

		// Handle assembly instructions
		case arch.Code.MatchString(line):
			code := strings.Split(line, arch.CommentCh)[0]
			code = strings.TrimSpace(code)
			if labelName == "" {
				current.Lines = append(current.Lines, Line{Assembly: code})
			} else {
				current.Lines[len(current.Lines)-1].Assembly = code
				labelName = ""
			}
		}
	}

	if err = scanner.Err(); err != nil {
		return nil, err
	}
	return functions, nil
}

// ParseClangObjectDump parses the output of objdump file and returns a list of functions
func ParseClangObjectDump(arch *config.Arch, dump string, functions []Function, dec Plan9Decoder) error {
	var (
		functionName string
		functionIdx  int
		current      *Function
		lineNumber   int
	)

	for i, line := range strings.Split(dump, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case arch.Symbol.MatchString(line):
			functionName = strings.Split(line, "<")[1]
			functionName = strings.Split(functionName, ">")[0]
			current = &functions[functionIdx]
			lineNumber = 0
			functionIdx++
		case arch.Data.MatchString(line):
			data := strings.SplitN(line, ":", 2)[1]
			data = strings.TrimSpace(data)
			parts := strings.SplitN(data, "\t", 2)
			splits := strings.Split(strings.TrimSpace(parts[0]), " ")
			if len(parts) < 2 {
				return fmt.Errorf("failed to parse objdump line %d: instruction %q (%q)", i, data, line)
			}
			var (
				binary   []string
				assembly = parts[1]
			)

			for _, s := range splits {
				// If the binary representation is not separated with spaces, split it
				switch {
				case len(s) > 2:
					// Iterate backwards
					for i := len(s) - 2; i >= 0; i -= 2 {
						binary = append(binary, s[i:i+2])
					}
				default:
					binary = append(binary, s)
				}
			}

			switch {
			case assembly == "":
				return fmt.Errorf("try to increase --insn-width of objdump")
			case strings.HasPrefix(assembly, "nop"):
				continue
			case assembly == "xchg   %ax,%ax":
				continue
			case strings.HasPrefix(assembly, "cs nopw"):
				continue
			case lineNumber >= len(current.Lines):
				return fmt.Errorf("%d: unexpected objectdump line: %s, please compare assembly with objdump output", i, line)
			}

			if dec != nil {
				p9asm, err := dec.DecodeInstruction(functionName, binary)
				if err != nil {
					return fmt.Errorf("cannot decode instruction %q: %v", data, err)
				}
				current.Lines[lineNumber].Disassembled = p9asm
			}

			current.Lines[lineNumber].Binary = binary
			lineNumber++
		}
	}
	return nil
}

/*
func containsLabel(arch *config.Arch, line string) bool {
	parts := whitespaceRe.Split(line, -1)
	for _, part := range parts {
		if arch.SourceLabel.MatchString(part) {
			return true
		}
	}
	return false
}
*/

// ParseGoObjectDump parses the output of objdump file and returns a list of functions
func ParseGoObjectDump(arch *config.Arch, dump string, functions []Function) error {
	var (
		functionName string
		current      *Function
		lineNumber   int
	)

	symbolRe := regexp.MustCompile(`^TEXT\s+(.*)+[(]SB[)]\s*$`)
	dataRe := regexp.MustCompile(`^\s*([:]\d+)\s+(0x[0-9a-f]+)\s+([0-9a-f]+)\s+([?]|\w+.*)$`)

	for i, line := range strings.Split(dump, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case symbolRe.MatchString(line):
			m := symbolRe.FindStringSubmatch(line)
			functionName = m[1]
			functionIdx := slices.IndexFunc(functions, func(fn Function) bool {
				return fn.Name == functionName
			})
			if functionIdx == -1 {
				current = nil
				continue
			}
			current = &functions[functionIdx]
			lineNumber = 0
		case dataRe.MatchString(line):
			if current == nil {
				continue
			}
			// matches in dataRe:
			// 1: ??
			// 2: address
			// 3: binary
			// 4: assembly
			m := dataRe.FindStringSubmatch(line)

			binHex := m[3]
			assembly := m[4]

			// wait what, this should be independent of the instruction set
			switch {
			case assembly == "" || assembly == "?":
				return fmt.Errorf("objectdump failure on line: %d, please compare assembly with objdump output", i)
			case lineNumber >= len(current.Lines):
				return fmt.Errorf("%d: unexpected objectdump line: %s, please compare assembly with objdump output", i, line)
			}

			binary := []string{}
			// split the binary representation into bytes
			if len(binHex) > 2 {
				// Iterate backwards
				for i := len(binHex) - 2; i >= 0; i -= 2 {
					binary = append(binary, binHex[i:i+2])
				}
			} else {
				binary = append(binary, binHex)
			}

			curLine := &current.Lines[lineNumber]
			curLine.Binary = binary
			curLine.Disassembled = assembly

			lineNumber++
		}
	}
	return nil
}
