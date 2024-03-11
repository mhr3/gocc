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
	"go/ast"
	"go/parser"
	"go/types"
	"os"
	"reflect"
	"sort"
	"strings"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/kelindar/gocc/internal/asm"
	"modernc.org/cc/v4"
)

var supportedTypes = mapset.NewSet(
	"int64_t", "uint64_t",
	"int32_t", "uint32_t",
	"int16_t", "uint16_t",
	"int8_t", "uint8_t",
	"float", "double",
	"longlong", "unsignedlonglong",
	"long", "unsignedlong",
	"int", "unsignedint",
	"short", "unsignedshort",
	"char", "unsignedchar",
)

var go2c = map[string]asm.Param{
	"int":     {Type: normalizeCType("int64_t")},
	"int8":    {Type: normalizeCType("int8_t")},
	"int16":   {Type: normalizeCType("int16_t")},
	"int32":   {Type: normalizeCType("int32_t")},
	"int64":   {Type: normalizeCType("int64_t")},
	"uint":    {Type: normalizeCType("uint64_t")},
	"uint8":   {Type: normalizeCType("uint8_t")},
	"uint16":  {Type: normalizeCType("uint16_t")},
	"uint32":  {Type: normalizeCType("uint32_t")},
	"uint64":  {Type: normalizeCType("uint64_t")},
	"float32": {Type: "float"},
	"float64": {Type: "double"},
	"string":  {Type: "char", IsPointer: true},
}

var errNoSignature = errors.New("no gocc signature found")

// Parse parse C source file and extracts functions declarations.
func Parse(path string) ([]asm.Function, error) {
	source, err := redactSource(path)
	if err != nil {
		return nil, err
	}

	ccCfg, err := cc.NewConfig("", "")
	if err != nil {
		return nil, fmt.Errorf("gocc: %w", err)
	}
	ast, err := cc.Parse(ccCfg, []cc.Source{
		{Name: "<predefined>", Value: ccCfg.Predefined},
		{Name: "<builtin>", Value: cc.Builtin},
		{Name: path, Value: source},
	})
	if err != nil {
		return nil, fmt.Errorf("gocc: %w", err)
	}

	var functions []asm.Function
	tu := ast.TranslationUnit
	for tu != nil {
		decl := tu.ExternalDeclaration
		tu = tu.TranslationUnit

		if decl == nil || decl.Case != cc.ExternalDeclarationFuncDef {
			continue
		}

		funcDef := decl.FunctionDefinition
		if funcDef.Position().Filename != path {
			continue
		}
		funcResult := funcDef.DeclarationSpecifiers.TypeSpecifier
		funcComment := strings.TrimSpace(string(funcResult.Token.Sep()))
		goName, goSig, err := extractGoSignature(funcComment)
		if err == errNoSignature {
			continue
		} else if err != nil {
			return nil, err
		}

		declarator := funcDef.Declarator
		if declarator != nil {
			funcIdent := declarator.DirectDeclarator
			if funcIdent.Case != cc.DirectDeclaratorFuncParam {
				continue
			}
			if goSig.Results.NumFields() > 0 && funcResult.Case != cc.TypeSpecifierVoid {
				return nil, fmt.Errorf("%s: must return void, not %s", funcIdent.DirectDeclarator.Token.SrcStr(), funcResult.Token.SrcStr())
			}

			function, err := convertFunction(funcIdent, goName)
			if err != nil {
				return nil, err
			}
			if err := checkFunction(function, goSig); err != nil {
				return nil, err
			}
			functions = append(functions, function)
		}
	}

	if len(functions) == 0 {
		return nil, errors.New("gocc: no functions found")
	}

	sort.Slice(functions, func(i, j int) bool {
		return functions[i].Position < functions[j].Position
	})
	return functions, nil
}

// redactSource removes code from the source and only leaves function declarations.
// This is done to avoid parsing errors when the source is not compatible with the compiler.
func redactSource(path string) (string, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var src strings.Builder
	src.WriteString("#define __STDC_HOSTED__ 1\n")
	src.WriteString("#define uint64_t unsigned long long\n")
	src.WriteString("#define uint32_t unsigned int\n")
	src.WriteString("#define uint16_t unsigned short\n")
	src.WriteString("#define uint8_t unsigned char\n")
	src.WriteString("#define int64_t long long\n")
	src.WriteString("#define int32_t int\n")
	src.WriteString("#define int16_t short\n")
	src.WriteString("#define int8_t char\n")

	var clauseCount int
	for _, line := range strings.Split(string(bytes), "\n") {
		switch {
		case strings.HasPrefix(line, "#"):
			continue
		case strings.HasPrefix(line, "//"):
			// keep comments
			src.WriteString(line)
			src.WriteRune('\n')
		case strings.Contains(line, "{"):
			if clauseCount == 0 {
				src.WriteString(line[:strings.Index(line, "{")+1])
				src.WriteString("\n // removed for compatibility\n")
			}
			clauseCount++
		case strings.Contains(line, "}"):
			clauseCount--
			if clauseCount == 0 {
				src.WriteString(line[strings.Index(line, "}"):])
				src.WriteRune('\n')
			}
		default:
			continue
		}
	}

	return src.String(), nil
}

func normalizeCType(t string) string {
	switch t {
	case "uint64_t":
		return "unsignedlonglong"
	case "uint32_t":
		return "unsignedint"
	case "uint16_t":
		return "unsignedshort"
	case "uint8_t":
		return "unsignedchar"
	case "int64_t":
		return "longlong"
	case "int32_t":
		return "int"
	case "int16_t":
		return "short"
	case "int8_t":
		return "char"
	default:
		return t
	}
}

// convertFunction extracts the function definition from cc.DirectDeclarator.
func convertFunction(declarator *cc.DirectDeclarator, comment string) (asm.Function, error) {
	params, err := convertFunctionParameters(declarator.ParameterTypeList.ParameterList)
	if err != nil {
		return asm.Function{}, err
	}

	return asm.Function{
		Name:     declarator.DirectDeclarator.Token.SrcStr(),
		Position: declarator.Position().Line,
		Params:   params,
	}, nil
}

func extractGoSignature(comment string) (string, *ast.FuncType, error) {
	lines := strings.Split(comment, "\n")
	for _, line := range lines {
		if !strings.Contains(line, "gocc:") {
			continue
		}
		parts := strings.SplitN(line, "gocc:", 2)

		goExpr, err := parser.ParseExpr("interface {" + parts[1] + "}")
		if err != nil {
			return "", nil, err
		}
		if interfaceType, ok := goExpr.(*ast.InterfaceType); !ok {
			return "", nil, errors.New("gocc: invalid signature")
		} else if len(interfaceType.Methods.List) != 1 {
			return "", nil, errors.New("gocc: invalid signature")
		} else {
			method := interfaceType.Methods.List[0]
			return method.Names[0].String(), method.Type.(*ast.FuncType), nil
		}
	}

	return "", nil, errNoSignature
}

func checkFunction(function asm.Function, goSig *ast.FuncType) error {
	checkParam := func(idx int, expectedParam asm.Param) error {
		if idx >= len(function.Params) {
			return fmt.Errorf("%s: too few parameters, missing %s", function.Name, expectedParam.CTypeStr())
		}

		param := function.Params[idx]
		if expectedParam.IsPointer {
			if !param.IsPointer {
				return fmt.Errorf("%s: expected pointer, got %v", function.Name, param.CTypeStr())
			}
		} else if param.IsPointer {
			return fmt.Errorf("%s: expected value, got pointer", function.Name)
		}
		if param.Type != expectedParam.Type {
			got := normalizeCType(param.Type)
			// ignore the "unsigned" prefix
			if !strings.HasSuffix(got, expectedParam.Type) {
				return fmt.Errorf("%s: expected %v, got %v", function.Name, expectedParam.CTypeStr(), param.CTypeStr())
			}
		}

		return nil
	}

	j := 0
	for _, goParam := range goSig.Params.List {
		goParamTypeName := types.ExprString(goParam.Type)
		for range goParam.Names {
			switch goParamTypeName {
			case "int", "int8", "int16", "int32", "int64",
				"uint", "uint8", "uint16", "uint32", "uint64",
				"float32", "float64":
				if err := checkParam(j, go2c[goParamTypeName]); err != nil {
					return err
				}
				j++
			case "string":
				if err := checkParam(j, go2c[goParamTypeName]); err != nil {
					return err
				}
				if err := checkParam(j+1, go2c["int"]); err != nil {
					return err
				}
				j += 2
			case "unsafe.Pointer":
				if err := checkParam(j, asm.Param{IsPointer: true}); err != nil {
					return err
				}
				j++
			default:
				if strings.HasPrefix(goParamTypeName, "[]") {
					if err := checkParam(j, asm.Param{IsPointer: true}); err != nil {
						return err
					}
					if err := checkParam(j+1, go2c["int"]); err != nil {
						return err
					}
					if err := checkParam(j+2, go2c["int"]); err != nil {
						return err
					}
					j += 3
					continue
				}
				return fmt.Errorf("gocc: unsupported type: %v", goParamTypeName)
			}
		}
	}

	if goSig.Results != nil {
		for _, goRet := range goSig.Results.List {
			goRetTypeName := types.ExprString(goRet.Type)
			switch goRetTypeName {
			case "int", "int8", "int16", "int32", "int64",
				"uint", "uint8", "uint16", "uint32", "uint64",
				"float32", "float64":
				p := go2c[goRetTypeName]
				p.IsPointer = true
				if err := checkParam(j, p); err != nil {
					return err
				}
				j++
			case "unsafe.Pointer":
				if err := checkParam(j, asm.Param{IsPointer: true}); err != nil {
					return err
				}
				j++
			default:
				return fmt.Errorf("%s: unsupported return type: %v", function.Name, goRetTypeName)
			}
		}
	}

	if j < len(function.Params) {
		return fmt.Errorf("%s: too many parameters, unexpected %s", function.Name, function.Params[j].CTypeStr())
	}

	return nil
}

// convertFunctionParameters extracts function parameters from cc.ParameterList.
func convertFunctionParameters(params *cc.ParameterList) ([]asm.Param, error) {
	declaration := params.ParameterDeclaration
	isPointer := declaration.Declarator.Pointer != nil
	paramName := declaration.Declarator.DirectDeclarator.Token.SrcStr()
	paramType := typeOf(declaration.DeclarationSpecifiers)

	if !isPointer && !supportedTypes.Contains(paramType) {
		position := declaration.Position()
		return nil, fmt.Errorf("gocc: [%v] unsupported type: %v",
			position.Filename, paramType)
	}

	paramNames := []asm.Param{{
		Name:      paramName,
		Type:      paramType,
		IsPointer: isPointer,
	}}

	if params.ParameterList != nil {
		if nextParamNames, err := convertFunctionParameters(params.ParameterList); err != nil {
			return nil, err
		} else {
			paramNames = append(paramNames, nextParamNames...)
		}
	}
	return paramNames, nil
}

// typeOf returns the type of the given value, recursively.
func typeOf(v any) string {
	if rv := reflect.ValueOf(v); rv.Kind() == reflect.Ptr && rv.IsNil() {
		return ""
	}

	switch s := v.(type) {
	case *cc.TypeQualifier:
		return s.Token.SrcStr()
	case *cc.TypeSpecifier:
		return s.Token.SrcStr()
	case *cc.DeclarationSpecifiers:
		var result string
		switch s.Case {
		case cc.DeclarationSpecifiersTypeQual:
			result += typeOf(s.TypeSpecifier)
			result += typeOf(s.DeclarationSpecifiers)
		case cc.DeclarationSpecifiersTypeSpec:
			result += typeOf(s.TypeSpecifier)
			result += typeOf(s.DeclarationSpecifiers)
		default:
			panic(fmt.Sprintf("gocc: unexpected specifiers case: %v", s.Case))
		}
		return result
	default:
		panic(fmt.Sprintf("gocc: unexpected specifier type: %T", v))
	}
}
