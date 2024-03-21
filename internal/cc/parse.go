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
	"byte":    {Type: normalizeCType("uint8_t")},
	"float32": {Type: "float"},
	"float64": {Type: "double"},
	"string":  {Type: "char", IsPointer: true},
	"bool":    {Type: "bool"},
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
		var (
			allTypeSpecs []cc.TypeSpecifierCase
			funcComment  string
			funcTypeSpec *cc.TypeSpecifier
		)
		declSpec := funcDef.DeclarationSpecifiers
		for declSpec != nil {
			switch declSpec.Case {
			case cc.DeclarationSpecifiersTypeSpec:
				funcTypeSpec = declSpec.TypeSpecifier
				if funcComment == "" {
					funcComment = strings.TrimSpace(string(declSpec.TypeSpecifier.Token.Sep()))
				}
				allTypeSpecs = append(allTypeSpecs, declSpec.TypeSpecifier.Case)
			case cc.DeclarationSpecifiersTypeQual:
				if funcComment == "" {
					funcComment = strings.TrimSpace(string(declSpec.TypeQualifier.Token.Sep()))
				}
			case cc.DeclarationSpecifiersStorage:
				if funcComment == "" {
					funcComment = strings.TrimSpace(string(declSpec.StorageClassSpecifier.Token.Sep()))
				}
			}
			declSpec = declSpec.DeclarationSpecifiers
		}
		if funcTypeSpec == nil {
			continue
		}

		goFunc, err := extractGoSignature(funcComment)
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
			if goFunc.NumResults() > 0 {
				if funcTypeSpec.Case == cc.TypeSpecifierVoid {
					var expectedRet string
					goFunc.ForEachResult(func(_, typ string) {
						expectedRet = typ
					})
					return nil, fmt.Errorf("%s: function cannot return void, expected %s", funcIdent.DirectDeclarator.Token.SrcStr(), expectedRet)
				}
				// check whether the type actually matches
			}

			function, err := convertFunction(funcIdent, extractCReturnType(allTypeSpecs), goFunc)
			if err != nil {
				return nil, err
			}
			if err := checkFunction(function); err != nil {
				return nil, err
			}
			functions = append(functions, function)
		}
	}

	if len(functions) == 0 {
		return nil, fmt.Errorf("gocc: %s no function annotations found", path)
	}

	sort.Slice(functions, func(i, j int) bool {
		return functions[i].Position < functions[j].Position
	})
	return functions, nil
}

func extractCReturnType(specs []cc.TypeSpecifierCase) string {
	var ret string
	for _, spec := range specs {
		s := strings.TrimPrefix(spec.String(), "TypeSpecifier")
		s = strings.ToLower(s)
		ret += s
	}
	return ret
}

// redactSource removes code from the source and only leaves function declarations.
// This is done to avoid parsing errors when the source is not compatible with the compiler.
func redactSource(path string) (string, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var redacted strings.Builder
	redacted.WriteString("#define __STDC_HOSTED__ 1\n")
	redacted.WriteString("#define uint64_t unsigned long long\n")
	redacted.WriteString("#define uint32_t unsigned int\n")
	redacted.WriteString("#define uint16_t unsigned short\n")
	redacted.WriteString("#define uint8_t unsigned char\n")
	redacted.WriteString("#define int64_t long long\n")
	redacted.WriteString("#define int32_t int\n")
	redacted.WriteString("#define int16_t short\n")
	redacted.WriteString("#define int8_t char\n")
	redacted.WriteString("#define bool _Bool\n")

	var clauseCount int

	lines := strings.Split(string(src), "\n")
	for i, l := range lines {
		line := strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(line, "#"):
			continue
		case strings.HasPrefix(line, "//") && clauseCount == 0:
			// keep comments
			redacted.WriteString(line)
			redacted.WriteRune('\n')
		case strings.Contains(line, "{"):
			if clauseCount == 0 {
				bracketIdx := strings.Index(line, "{")
				if bracketIdx == 0 {
					// just try including the previous line
					redacted.WriteString(lines[i-1])
				}
				redacted.WriteString(line[:bracketIdx+1])
				redacted.WriteString("\n // removed for compatibility\n")
			}
			clauseCount++
		case strings.Contains(line, "}"):
			clauseCount--
			if clauseCount == 0 {
				redacted.WriteString(line[strings.Index(line, "}"):])
				redacted.WriteRune('\n')
			}
		default:
			continue
		}
	}

	return redacted.String(), nil
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
func convertFunction(declarator *cc.DirectDeclarator, returnType string, goFunc asm.GoFunction) (asm.Function, error) {
	params, err := convertFunctionParameters(declarator.ParameterTypeList.ParameterList)
	if err != nil {
		return asm.Function{}, err
	}

	var retParam *asm.Param
	if returnType != "void" {
		retParam = &asm.Param{
			Type: returnType,
			Name: "ret",
		}
	}

	return asm.Function{
		Name:     declarator.DirectDeclarator.Token.SrcStr(),
		Position: declarator.Position().Line,
		Params:   params,
		Ret:      retParam,
		GoFunc:   goFunc,
	}, nil
}

func extractGoSignature(comment string) (asm.GoFunction, error) {
	lines := strings.Split(comment, "\n")
	for _, line := range lines {
		if !strings.Contains(line, "gocc:") {
			continue
		}
		parts := strings.SplitN(line, "gocc:", 2)

		goExpr, err := parser.ParseExpr("interface {" + parts[1] + "}")
		if err != nil {
			return asm.GoFunction{}, err
		}
		if interfaceType, ok := goExpr.(*ast.InterfaceType); !ok {
			return asm.GoFunction{}, errors.New("gocc: invalid signature")
		} else if len(interfaceType.Methods.List) != 1 {
			return asm.GoFunction{}, errors.New("gocc: invalid signature")
		} else {
			method := interfaceType.Methods.List[0]
			return asm.GoFunction{Name: method.Names[0].String(), Expr: method.Type.(*ast.FuncType)}, nil
		}
	}

	return asm.GoFunction{}, errNoSignature
}

func checkFunction(function asm.Function) error {
	checkParam := func(idx int, expectedParam asm.Param) error {
		if idx >= len(function.Params) {
			return fmt.Errorf("%s: too few parameters, missing %s", function.Name, expectedParam.CTypeStr())
		}

		param := function.Params[idx]
		if expectedParam.IsPointer {
			if !param.IsPointer {
				return fmt.Errorf("%s: param %q: expected pointer, got %v", function.Name, param.Name, param.CTypeStr())
			}
		} else if param.IsPointer {
			return fmt.Errorf("%s: param %q: expected value, got pointer", function.Name, param.Name)
		}
		if param.Size() != expectedParam.Size() {
			got := normalizeCType(param.Type)
			// ignore the "unsigned" prefix
			if !strings.HasSuffix(got, expectedParam.Type) {
				return fmt.Errorf("%s: param %q: expected %v, got %v", function.Name, param.Name, expectedParam.CTypeStr(), param.CTypeStr())
			}
		}

		return nil
	}

	j := 0
	for _, goParam := range function.GoFunc.Expr.Params.List {
		goParamTypeName := types.ExprString(goParam.Type)
		for range goParam.Names {
			switch goParamTypeName {
			case "int", "int8", "int16", "int32", "int64",
				"uint", "uint8", "uint16", "uint32", "uint64",
				"byte", "char", "bool",
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
				if strings.HasPrefix(goParamTypeName, "*") {
					if err := checkParam(j, asm.Param{IsPointer: true}); err != nil {
						return err
					}
					j++
					continue
				}
				return fmt.Errorf("gocc: unsupported type: %v", goParamTypeName)
			}
		}
	}

	if function.GoFunc.Expr.Results != nil {
		if function.GoFunc.NumResults() > 1 {
			return fmt.Errorf("%s: multiple return values are not supported", function.Name)
		}
		for _, goRet := range function.GoFunc.Expr.Results.List {
			goRetTypeName := types.ExprString(goRet.Type)
			switch goRetTypeName {
			case "int", "int8", "int16", "int32", "int64",
				"uint", "uint8", "uint16", "uint32", "uint64",
				"float32", "float64",
				"byte", "bool":
				p := go2c[goRetTypeName]

				// check the return type
				if function.Ret == nil {
					return fmt.Errorf("%s: missing return type, expected %s", function.Name, p.CTypeStr())
				}

				if p.Size() != function.Ret.Size() {
					return fmt.Errorf("%s: invalid return type, expected %s, got %s", function.Name, p.Type, function.Ret.Type)
				}
			default:
				return fmt.Errorf("%s: unsupported return type: %v", function.Name, goRetTypeName)
			}
		}
	}

	if j < len(function.Params) {
		extraParam := function.Params[j]
		return fmt.Errorf("%s: too many parameters, unexpected %q", function.Name, extraParam.CString())
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
