package main

import (
	"bytes"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// mustGenerate calls generate and returns the output as a string, failing on error.
func mustGenerate(t *testing.T, enums []enumType) string {
	t.Helper()
	var buf bytes.Buffer
	if err := generate(&buf, enums); err != nil {
		t.Fatalf("generate: %v", err)
	}
	return buf.String()
}

// extractFuncBody returns the full source text of a top-level function named funcName
// from src (searching for "func <funcName>"), from opening brace to matching close.
func extractFuncBody(src, funcName string) string {
	start := strings.Index(src, "func "+funcName)
	if start < 0 {
		return ""
	}
	depth := 0
	for i, ch := range src[start:] {
		switch ch {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[start : start+i+1]
			}
		}
	}
	return ""
}

// threeValueEnum is a representative input with one invalid and two valid values.
var threeValueEnum = enumType{
	pkg:            "p",
	typeName:       "myEnum",
	underlyingType: "int",
	values: []enumValue{
		{unexportedConst: "myEnumUnknown", stringName: "unknown", isInvalid: true, iotaIndex: 0},
		{unexportedConst: "myEnumFoo", stringName: "foo", isInvalid: false, iotaIndex: 1},
		{unexportedConst: "myEnumBar", stringName: "bar", isInvalid: false, iotaIndex: 2},
	},
}

func TestGenerate_outputIsValidGo(t *testing.T) {
	t.Parallel()
	src := mustGenerate(t, []enumType{threeValueEnum})
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, "generated.go", src, 0); err != nil {
		t.Fatalf("generated code is not valid Go:\n%v\n\nOutput:\n%s", err, src)
	}
}

func TestGenerate_exportedTypeDeclaration(t *testing.T) {
	t.Parallel()
	src := mustGenerate(t, []enumType{threeValueEnum})
	if !strings.Contains(src, "type MyEnum int") {
		t.Errorf("missing 'type MyEnum int'; output:\n%s", src)
	}
}

func TestGenerate_exportedConstsWithIota(t *testing.T) {
	t.Parallel()
	src := mustGenerate(t, []enumType{threeValueEnum})

	if !strings.Contains(src, "MyEnumUnknown MyEnum = iota") {
		t.Errorf("first const must declare the type and use iota")
	}
	for _, name := range []string{"MyEnumUnknown", "MyEnumFoo", "MyEnumBar"} {
		if !strings.Contains(src, name) {
			t.Errorf("missing exported const %q", name)
		}
	}
}

func TestGenerate_functionDeclarations(t *testing.T) {
	t.Parallel()
	src := mustGenerate(t, []enumType{threeValueEnum})

	want := []string{
		"func (v MyEnum) String()",
		"func (v MyEnum) IsValid()",
		"func ParseMyEnum(",
		"func allMyEnums()",
		"func AllMyEnums()",
		"func ExhaustiveMyEnums(",
		"func (v MyEnum) MarshalJSON()",
		"func (v *MyEnum) UnmarshalJSON(",
		"func (v MyEnum) MarshalText()",
		"func (v *MyEnum) UnmarshalText(",
		"func (v MyEnum) MarshalBinary()",
		"func (v *MyEnum) UnmarshalBinary(",
		"func (v MyEnum) MarshalYAML()",
		"func (v *MyEnum) UnmarshalYAML(",
		"func (v *MyEnum) Scan(",
		"func (v MyEnum) Value()",
	}
	for _, fn := range want {
		if !strings.Contains(src, fn) {
			t.Errorf("missing %q in output", fn)
		}
	}
}

func TestGenerate_namesMapUsesDirectStrings(t *testing.T) {
	t.Parallel()
	src := mustGenerate(t, []enumType{threeValueEnum})

	for _, want := range []string{
		`MyEnumUnknown: "unknown"`,
		`MyEnumFoo: "foo"`,
		`MyEnumBar: "bar"`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("missing names map entry %q", want)
		}
	}
}

func TestGenerate_allSliceExcludesInvalidValues(t *testing.T) {
	t.Parallel()
	src := mustGenerate(t, []enumType{threeValueEnum})

	body := extractFuncBody(src, "allMyEnums()")
	if body == "" {
		t.Fatal("allMyEnums function not found in output")
	}
	if strings.Contains(body, "MyEnumUnknown") {
		t.Error("allMyEnums should not include invalid value MyEnumUnknown")
	}
	if !strings.Contains(body, "MyEnumFoo") {
		t.Error("allMyEnums should include valid value MyEnumFoo")
	}
	if !strings.Contains(body, "MyEnumBar") {
		t.Error("allMyEnums should include valid value MyEnumBar")
	}
}

func TestGenerate_validMapMarksInvalidFalse(t *testing.T) {
	t.Parallel()
	src := mustGenerate(t, []enumType{threeValueEnum})

	if !strings.Contains(src, "MyEnumUnknown: false") {
		t.Error("invalid value should be false in valid map")
	}
	if !strings.Contains(src, "MyEnumFoo: true") {
		t.Error("valid value should be true in valid map")
	}
	if !strings.Contains(src, "MyEnumBar: true") {
		t.Error("valid value should be true in valid map")
	}
}

func TestGenerate_compileTimeCheckUsesUnexportedConsts(t *testing.T) {
	t.Parallel()
	src := mustGenerate(t, []enumType{threeValueEnum})

	if !strings.Contains(src, "var x [3]struct{}") {
		t.Error("compile-time check array size should equal total value count")
	}
	// index 0: no subtraction
	if !strings.Contains(src, "_ = x[myEnumUnknown]") {
		t.Error("compile-time check for index 0 should not subtract")
	}
	// subsequent indices subtract their position
	if !strings.Contains(src, "_ = x[myEnumFoo-1]") {
		t.Error("compile-time check for index 1 should subtract 1")
	}
	if !strings.Contains(src, "_ = x[myEnumBar-2]") {
		t.Error("compile-time check for index 2 should subtract 2")
	}
}

func TestGenerate_invalidSentinelIsFirstValue(t *testing.T) {
	t.Parallel()
	src := mustGenerate(t, []enumType{threeValueEnum})

	// ParseMyEnum error paths return the first (invalid) const
	count := strings.Count(src, "return MyEnumUnknown,")
	if count < 2 {
		t.Errorf("expected at least 2 error-return paths using MyEnumUnknown, found %d", count)
	}
}

func TestGenerate_int64UnderlyingType(t *testing.T) {
	t.Parallel()
	e := enumType{
		pkg:            "p",
		typeName:       "myEnum",
		underlyingType: "int64",
		values: []enumValue{
			{unexportedConst: "myEnumFoo", stringName: "foo", isInvalid: false, iotaIndex: 0},
		},
	}
	src := mustGenerate(t, []enumType{e})

	if !strings.Contains(src, "type MyEnum int64") {
		t.Error("expected 'type MyEnum int64'")
	}
	// String() should cast to int64, not int
	if !strings.Contains(src, "int64(v)") {
		t.Error("String() should use int64 cast matching underlying type")
	}
}

func TestGenerate_multipleEnumsProduceValidGo(t *testing.T) {
	t.Parallel()
	enums := []enumType{
		{
			pkg: "p", typeName: "colorEnum", underlyingType: "int",
			values: []enumValue{
				{unexportedConst: "colorEnumRed", stringName: "red", isInvalid: false, iotaIndex: 0},
			},
		},
		{
			pkg: "p", typeName: "sizeEnum", underlyingType: "int",
			values: []enumValue{
				{unexportedConst: "sizeEnumSmall", stringName: "small", isInvalid: false, iotaIndex: 0},
			},
		},
	}
	src := mustGenerate(t, enums)

	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, "generated.go", src, 0); err != nil {
		t.Fatalf("generated code for multiple enums is not valid Go: %v\n\n%s", err, src)
	}
	for _, name := range []string{"ColorEnum", "SizeEnum", "ParseColorEnum", "ParseSizeEnum"} {
		if !strings.Contains(src, name) {
			t.Errorf("missing %q in multi-enum output", name)
		}
	}
}

func TestGenerate_noInvalidValues(t *testing.T) {
	t.Parallel()
	// When no value is marked invalid, all appear in allSlice and first is the error sentinel.
	e := enumType{
		pkg:            "p",
		typeName:       "myEnum",
		underlyingType: "int",
		values: []enumValue{
			{unexportedConst: "myEnumFoo", stringName: "foo", isInvalid: false, iotaIndex: 0},
			{unexportedConst: "myEnumBar", stringName: "bar", isInvalid: false, iotaIndex: 1},
		},
	}
	src := mustGenerate(t, []enumType{e})

	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, "generated.go", src, 0); err != nil {
		t.Fatalf("generated code not valid Go: %v", err)
	}
	body := extractFuncBody(src, "allMyEnums()")
	if !strings.Contains(body, "MyEnumFoo") || !strings.Contains(body, "MyEnumBar") {
		t.Error("when no value is invalid, allSlice should include all values")
	}
}

func TestGenerate_singleValue(t *testing.T) {
	t.Parallel()
	e := enumType{
		pkg:            "p",
		typeName:       "myEnum",
		underlyingType: "int",
		values: []enumValue{
			{unexportedConst: "myEnumOnly", stringName: "only", isInvalid: false, iotaIndex: 0},
		},
	}
	src := mustGenerate(t, []enumType{e})

	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, "generated.go", src, 0); err != nil {
		t.Fatalf("single-value enum generated invalid Go: %v", err)
	}
	// compile-time check for index 0 has no subtraction
	if !strings.Contains(src, "_ = x[myEnumOnly]") {
		t.Error("single-value compile-time check should not subtract")
	}
}

func TestGenerate_nilEnumsProducesNoOutput(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := generate(&buf, nil); err != nil {
		t.Fatalf("generate(nil): %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output for nil enums, got %d bytes", buf.Len())
	}
}

func TestGenerate_emptyEnumSliceProducesNoOutput(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := generate(&buf, []enumType{}); err != nil {
		t.Fatalf("generate([]): %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output for empty enum slice, got %d bytes", buf.Len())
	}
}

func TestGenerate_packageNameInOutput(t *testing.T) {
	t.Parallel()
	e := enumType{
		pkg:            "mypkg",
		typeName:       "myEnum",
		underlyingType: "int",
		values: []enumValue{
			{unexportedConst: "myEnumFoo", stringName: "foo", isInvalid: false, iotaIndex: 0},
		},
	}
	src := mustGenerate(t, []enumType{e})

	if !strings.Contains(src, "package mypkg") {
		t.Error("output should contain the correct package declaration")
	}
}

func TestPluralize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// default: add "s"
		{name: "simple noun", input: "Flag", want: "Flags"},
		{name: "lowercase", input: "color", want: "colors"},
		// ends in s → add "es"
		{name: "ends in s", input: "OrderStatus", want: "OrderStatuses"},
		{name: "ends in s lowercase", input: "status", want: "statuses"},
		// ends in x → add "es"
		{name: "ends in x", input: "Box", want: "Boxes"},
		// ends in z → add "es"
		{name: "ends in z", input: "Buzz", want: "Buzzes"},
		// ends in ch → add "es"
		{name: "ends in ch", input: "Church", want: "Churches"},
		// ends in sh → add "es"
		{name: "ends in sh", input: "Wish", want: "Wishes"},
		// ends in consonant+y → replace y with ies
		{name: "ends in consonant+y", input: "Category", want: "Categories"},
		{name: "ends in consonant+y lowercase", input: "city", want: "cities"},
		// ends in vowel+y → add "s" (not ies)
		{name: "ends in vowel+y", input: "Day", want: "Days"},
		{name: "ends in vowel+y key", input: "Key", want: "Keys"},
		// empty string
		{name: "empty", input: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := pluralize(tt.input)
			if got != tt.want {
				t.Errorf("pluralize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGenerate_doNotEditHeader(t *testing.T) {
	t.Parallel()
	src := mustGenerate(t, []enumType{threeValueEnum})
	if !strings.HasPrefix(src, "// Code generated by go-enum. DO NOT EDIT.") {
		t.Error("output should start with the do-not-edit header")
	}
}
