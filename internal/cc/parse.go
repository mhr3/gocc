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
	"slices"
	"sort"
	"strings"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/mhr3/gocc/internal/asm"
	"modernc.org/cc/v4"
)

const (
	goccComment        = "gocc:"
	goccVariantComment = "gocc-variant:"
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
	source, err := extractAnnotatedSignatures(path)
	if err != nil {
		return nil, fmt.Errorf("gocc: unable to process file: %w", err)
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
		return nil, fmt.Errorf("gocc: cannot parse function signatures: %w", err)
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
			allTypeSpecs     []cc.TypeSpecifierCase
			funcComment      string
			funcTypeSpec     *cc.TypeSpecifier
			funcStorageClass *cc.StorageClassSpecifier
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
				funcStorageClass = declSpec.StorageClassSpecifier
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
		variantSignatures, err := extractGoVariants(funcComment)
		if err != nil {
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
				// the return type check is done in checkFunction
			} else if funcTypeSpec.Case != cc.TypeSpecifierVoid {
				// the C function is returning a value, but the Go function is not
				return nil, fmt.Errorf("%s: go function is missing a return type", goFunc.Name)
			} else if funcStorageClass != nil && funcStorageClass.Case == cc.StorageClassSpecifierStatic {
				return nil, fmt.Errorf("%s: function cannot be static", funcIdent.DirectDeclarator.Token.SrcStr())
			}

			cName := funcIdent.DirectDeclarator.Token.SrcStr()
			goFunc, variants, err := resolveVariants(cName, goFunc, variantSignatures)
			if err != nil {
				return nil, err
			}
			function, err := convertFunction(funcIdent, extractCReturnType(allTypeSpecs), goFunc)
			if err != nil {
				return nil, err
			}
			function.Variants = variants
			if err := checkFunction(&function); err != nil {
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

// extractAnnotatedSignatures removes code from the source and only leaves function declarations.
// This is done to avoid parsing errors when the source is not compatible with the compiler.
func extractAnnotatedSignatures(path string) (string, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var redacted strings.Builder
	redacted.WriteString("#define __STDC_HOSTED__ 1\n")
	redacted.WriteString("#define size_t unsigned long long\n")
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
	var isAnnotated bool
	// variantLine is the line of a gocc-variant directive waiting for the
	// function it belongs to. A variant of a function without a gocc
	// signature must be rejected here: the function is dropped below while
	// its comments are kept, so the directive would attach itself to the
	// next annotated one.
	var variantLine int
	claimVariant := func() error {
		if clauseCount != 0 {
			return nil
		}
		if variantLine != 0 && !isAnnotated {
			return fmt.Errorf("line %d: gocc-variant on a function without a gocc signature", variantLine)
		}
		variantLine = 0
		return nil
	}

	lines := strings.Split(string(src), "\n")
	for i, l := range lines {
		line := strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(line, "#"):
			continue
		case strings.HasPrefix(line, "//") && clauseCount == 0:
			// keep comments
			if strings.Contains(line, goccComment) {
				isAnnotated = true
			} else if strings.Contains(line, goccVariantComment) {
				// part of the annotation, on either side of the gocc line
				variantLine = i + 1
			} else if isAnnotated {
				isAnnotated = false
			}

			redacted.WriteString(line)
			redacted.WriteRune('\n')
		case strings.Contains(line, "{") && strings.Contains(line, "}"):
			if err := claimVariant(); err != nil {
				return "", err
			}
			clauseCount += strings.Count(line, "{") - strings.Count(line, "}")
		case strings.Contains(line, "{"):
			if err := claimVariant(); err != nil {
				return "", err
			}
			if clauseCount == 0 && isAnnotated {
				startLine := i
				bracketIdx := strings.Index(line, "(")
				for bracketIdx == -1 && startLine > 0 {
					startLine--
					bracketIdx = strings.Index(lines[startLine], "(")
				}
				// possibly include the previous lines
				for j := startLine; j < i; j++ {
					redacted.WriteString(strings.TrimSpace(lines[j]))
					redacted.WriteRune(' ')
				}
				braceIdx := strings.Index(line, "{")
				redacted.WriteString(line[:braceIdx+1])
				redacted.WriteString("\n // removed for compatibility\n")
			}
			braceIdx := strings.Index(line, "{")
			closeBraceIdx := strings.Index(line, "}")
			if closeBraceIdx == -1 || closeBraceIdx < braceIdx {
				clauseCount++
			}
		case strings.Contains(line, "}"):
			clauseCount--
			if clauseCount == 0 && isAnnotated {
				redacted.WriteString(line[strings.Index(line, "}"):])
				redacted.WriteRune('\n')

				isAnnotated = false
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
	var params []asm.Param

	if declarator.ParameterTypeList != nil {
		var err error
		params, err = convertFunctionParameters(declarator.ParameterTypeList.ParameterList)
		if err != nil {
			return asm.Function{}, err
		}
	}

	var retParam *asm.Param
	if returnType != "void" {
		retParam = &asm.Param{
			Type: returnType,
			Name: "ret",
		}
	}

	declPos := declarator.Position()

	return asm.Function{
		Name:       declarator.DirectDeclarator.Token.SrcStr(),
		SourcePath: declPos.Filename,
		Position:   declPos.Line,
		Params:     params,
		Ret:        retParam,
		GoFunc:     goFunc,
	}, nil
}

// parseGoSignature parses "Name(params) results" as written in a directive.
func parseGoSignature(signature string) (asm.GoFunction, error) {
	goExpr, err := parser.ParseExpr("interface {" + signature + "}")
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

func extractGoSignature(comment string) (asm.GoFunction, error) {
	lines := strings.Split(comment, "\n")
	for _, line := range lines {
		if !strings.Contains(line, goccComment) {
			continue
		}
		parts := strings.SplitN(line, goccComment, 2)
		return parseGoSignature(parts[1])
	}

	return asm.GoFunction{}, errNoSignature
}

// extractGoVariants returns the signatures of the gocc-variant directives.
func extractGoVariants(comment string) ([]asm.GoFunction, error) {
	var variants []asm.GoFunction
	for _, line := range strings.Split(comment, "\n") {
		_, signature, found := strings.Cut(line, goccVariantComment)
		if !found {
			continue
		}
		variant, err := parseGoSignature(signature)
		if err != nil {
			return nil, err
		}
		variants = append(variants, variant)
	}
	return variants, nil
}

func isByteSlice(goType string) bool {
	return goType == "[]byte" || goType == "[]uint8"
}

// isText reports whether a parameter of this type may be a string in one
// signature of a function and a []byte in another.
func isText(goType string) bool {
	return goType == "string" || isByteSlice(goType)
}

// resolveVariants checks the signatures of a function that has gocc-variant
// directives and decides which of them the assembly is declared with.
//
// The signatures may differ only in taking a parameter as a string or as a
// []byte. The assembly gets the frame of the widest of them, so it is
// declared with the one that takes []byte wherever any of them does. The
// others are entered through padded entry points (see asm.GoVariant), and
// are returned with the parameters that need padding marked.
func resolveVariants(cName string, primary asm.GoFunction, others []asm.GoFunction) (asm.GoFunction, []asm.GoVariant, error) {
	if len(others) == 0 {
		return primary, nil, nil
	}
	fail := func(format string, args ...any) (asm.GoFunction, []asm.GoVariant, error) {
		return asm.GoFunction{}, nil, fmt.Errorf("%s: %s", cName, fmt.Sprintf(format, args...))
	}

	signatures := append([]asm.GoFunction{primary}, others...)
	var names, results [][]string
	types := make([][]string, len(signatures))
	for i := range signatures {
		var paramNames, resultTypes []string
		signatures[i].ForEachParam(func(name, typ string) {
			paramNames = append(paramNames, name)
			types[i] = append(types[i], typ)
		})
		signatures[i].ForEachResult(func(_, typ string) { resultTypes = append(resultTypes, typ) })
		names, results = append(names, paramNames), append(results, resultTypes)
	}

	// wide: some signature takes the parameter as a []byte and another as a
	// string, so its place in the frame is that of the []byte.
	wide := make([]bool, len(types[0]))
	for i := 1; i < len(signatures); i++ {
		variant := signatures[i].Name
		switch {
		case len(types[i]) != len(types[0]):
			return fail("variant %s: expected %d parameters, got %d", variant, len(types[0]), len(types[i]))
		case !slices.Equal(results[i], results[0]):
			return fail("variant %s: results differ from those of %s", variant, primary.Name)
		case slices.Equal(types[i], types[0]):
			return fail("variant %s: takes no parameter of %s as a string or a []byte instead", variant, primary.Name)
		}
		for p, typ := range types[i] {
			want := types[0][p]
			switch {
			case typ == want || (isByteSlice(typ) && isByteSlice(want)):
			case isText(typ) && isText(want):
				wide[p] = true
			case isText(want):
				return fail("variant %s: param %q: expected string or []byte, got %s", variant, names[i][p], typ)
			default:
				return fail("variant %s: param %q: expected %s, got %s", variant, names[i][p], want, typ)
			}
		}
	}

	takesBytes := func(i int) bool {
		for p := range wide {
			if wide[p] && !isByteSlice(types[i][p]) {
				return false
			}
		}
		return true
	}
	declared := -1
	for i := range signatures {
		if takesBytes(i) {
			declared = i
			break
		}
	}
	if declared < 0 {
		var wideNames []string
		for p := range wide {
			if wide[p] {
				wideNames = append(wideNames, names[0][p])
			}
		}
		return fail("one of the signatures has to take all of %s as []byte", strings.Join(wideNames, ", "))
	}

	var variants []asm.GoVariant
	for i := range signatures {
		if i == declared {
			continue
		}
		padded := make([]bool, len(wide))
		for p := range wide {
			// A wrapper passes the arguments on by name.
			if name := names[i][p]; name == "" || name == "_" {
				return fail("%s: parameter %d has to be named", signatures[i].Name, p+1)
			}
			padded[p] = wide[p] && types[i][p] == "string"
		}
		variants = append(variants, asm.GoVariant{GoFunction: signatures[i], Padded: padded})
	}
	return signatures[declared], variants, nil
}

// cParamCount returns the number of C parameters a Go signature stands for.
func cParamCount(goFunc *asm.GoFunction, capacities bool) int {
	count := 0
	goFunc.ForEachParam(func(_, typ string) {
		switch {
		case typ == "string":
			count += 2
		case strings.HasPrefix(typ, "[]") && capacities:
			count += 3
		case strings.HasPrefix(typ, "[]"):
			count += 2
		default:
			count++
		}
	})
	return count
}

// checkFunction verifies that the C signature matches the annotated Go one
// and records, for every C parameter, the name of its Go argument slot.
//
// A slice is three C parameters: pointer, length and capacity. A function
// with variants takes all of its slices as a pointer and a length instead,
// since a string in their place has no capacity to pass. The capacity keeps
// its word in the frame, which is recorded as a gap after the length.
func checkFunction(function *asm.Function) error {
	withoutCapacities := len(function.Variants) > 0
	if withoutCapacities && len(function.Params) != cParamCount(&function.GoFunc, false) &&
		len(function.Params) == cParamCount(&function.GoFunc, true) {
		return fmt.Errorf("%s: a function with variants takes a slice as a pointer and a length, without the capacity", function.Name)
	}

	checkParam := func(idx int, expectedParam asm.Param, goName string) error {
		if idx >= len(function.Params) {
			return fmt.Errorf("%s: too few parameters, missing %s", function.Name, expectedParam.CTypeStr())
		}

		param := function.Params[idx]
		function.Params[idx].GoName = goName
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
		for _, ident := range goParam.Names {
			name := ident.Name
			switch goParamTypeName {
			case "int", "int8", "int16", "int32", "int64",
				"uint", "uint8", "uint16", "uint32", "uint64",
				"byte", "char", "bool",
				"float32", "float64":
				if err := checkParam(j, go2c[goParamTypeName], name); err != nil {
					return err
				}
				j++
			case "string":
				if err := checkParam(j, go2c[goParamTypeName], name+"_base"); err != nil {
					return err
				}
				if err := checkParam(j+1, go2c["int"], name+"_len"); err != nil {
					return err
				}
				j += 2
			case "unsafe.Pointer":
				if err := checkParam(j, asm.Param{IsPointer: true}, name); err != nil {
					return err
				}
				j++
			default:
				if strings.HasPrefix(goParamTypeName, "[]") {
					if err := checkParam(j, asm.Param{IsPointer: true}, name+"_base"); err != nil {
						return err
					}
					if err := checkParam(j+1, go2c["int"], name+"_len"); err != nil {
						return err
					}
					if withoutCapacities {
						capacity := go2c["int"]
						function.Params[j+1].FrameGap = capacity.Size()
						j += 2
						continue
					}
					if err := checkParam(j+2, go2c["int"], name+"_cap"); err != nil {
						return err
					}
					j += 3
					continue
				}
				if strings.HasPrefix(goParamTypeName, "*") {
					if err := checkParam(j, asm.Param{IsPointer: true}, name); err != nil {
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
