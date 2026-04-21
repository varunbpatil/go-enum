package main

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// tempGoFile writes src to a temp file and returns its path.
func tempGoFile(t *testing.T, src string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "input.go")
	if err := os.WriteFile(f, []byte(src), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return f
}

func sortEnums(enums []enumType) {
	sort.Slice(enums, func(i, j int) bool { return enums[i].typeName < enums[j].typeName })
}

func assertEnumsEqual(t *testing.T, got, want []enumType) {
	t.Helper()
	sortEnums(got)
	sortEnums(want)
	if len(got) != len(want) {
		t.Fatalf("enum count: got %d, want %d", len(got), len(want))
	}
	for i := range want {
		assertEnumEqual(t, got[i], want[i])
	}
}

func assertEnumEqual(t *testing.T, got, want enumType) {
	t.Helper()
	if got.pkg != want.pkg {
		t.Errorf("pkg: got %q, want %q", got.pkg, want.pkg)
	}
	if got.typeName != want.typeName {
		t.Errorf("typeName: got %q, want %q", got.typeName, want.typeName)
	}
	if got.underlyingType != want.underlyingType {
		t.Errorf("underlyingType: got %q, want %q", got.underlyingType, want.underlyingType)
	}
	if len(got.values) != len(want.values) {
		t.Fatalf("values len for %q: got %d, want %d\n  got:  %+v\n  want: %+v",
			want.typeName, len(got.values), len(want.values), got.values, want.values)
	}
	for i, wv := range want.values {
		gv := got.values[i]
		if gv.unexportedConst != wv.unexportedConst {
			t.Errorf("values[%d].unexportedConst: got %q, want %q", i, gv.unexportedConst, wv.unexportedConst)
		}
		if gv.stringName != wv.stringName {
			t.Errorf("values[%d].stringName: got %q, want %q", i, gv.stringName, wv.stringName)
		}
		if gv.isInvalid != wv.isInvalid {
			t.Errorf("values[%d].isInvalid: got %v, want %v", i, gv.isInvalid, wv.isInvalid)
		}
		if gv.iotaIndex != wv.iotaIndex {
			t.Errorf("values[%d].iotaIndex: got %d, want %d", i, gv.iotaIndex, wv.iotaIndex)
		}
	}
}

func TestParseFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  string
		want []enumType
	}{
		{
			name: "basic enum with three values",
			src: `package p
type myStatus int
const (
	myStatusUnknown myStatus = iota // invalid unknown
	myStatusActive                  // active
	myStatusInactive                // inactive
)`,
			want: []enumType{{
				pkg: "p", typeName: "myStatus", underlyingType: "int",
				values: []enumValue{
					{unexportedConst: "myStatusUnknown", stringName: "unknown", isInvalid: true, iotaIndex: 0},
					{unexportedConst: "myStatusActive", stringName: "active", isInvalid: false, iotaIndex: 1},
					{unexportedConst: "myStatusInactive", stringName: "inactive", isInvalid: false, iotaIndex: 2},
				},
			}},
		},
		{
			name: "multiple distinct enum types in one file",
			src: `package p
type colorEnum int
type sizeEnum int
const (
	colorEnumRed  colorEnum = iota // red
	colorEnumBlue                  // blue
)
const (
	sizeEnumSmall sizeEnum = iota // small
	sizeEnumLarge                 // large
)`,
			want: []enumType{
				{
					pkg: "p", typeName: "colorEnum", underlyingType: "int",
					values: []enumValue{
						{unexportedConst: "colorEnumRed", stringName: "red", isInvalid: false, iotaIndex: 0},
						{unexportedConst: "colorEnumBlue", stringName: "blue", isInvalid: false, iotaIndex: 1},
					},
				},
				{
					pkg: "p", typeName: "sizeEnum", underlyingType: "int",
					values: []enumValue{
						{unexportedConst: "sizeEnumSmall", stringName: "small", isInvalid: false, iotaIndex: 0},
						{unexportedConst: "sizeEnumLarge", stringName: "large", isInvalid: false, iotaIndex: 1},
					},
				},
			},
		},
		{
			name: "exported type is ignored",
			src: `package p
type MyExported int
const (
	MyExportedFoo MyExported = iota // foo
)`,
			want: nil,
		},
		{
			name: "string underlying type is ignored",
			src: `package p
type myStr string
const (
	myStrFoo myStr = iota
)`,
			want: nil,
		},
		{
			name: "bool underlying type is ignored",
			src: `package p
type myBool bool
const (
	myBoolFoo myBool = iota
)`,
			want: nil,
		},
		{
			name: "const block without iota is ignored",
			src: `package p
type myEnum int
const (
	myEnumFoo myEnum = 1
	myEnumBar myEnum = 2
)`,
			want: nil,
		},
		{
			name: "type declared but no const block returns nothing",
			src: `package p
type myEnum int`,
			want: nil,
		},
		{
			name: "invalid-only comment derives string name from const name",
			src: `package p
type myEnum int
const (
	myEnumUnknown myEnum = iota // invalid
	myEnumFoo                   // foo
)`,
			want: []enumType{{
				pkg: "p", typeName: "myEnum", underlyingType: "int",
				values: []enumValue{
					{unexportedConst: "myEnumUnknown", stringName: "unknown", isInvalid: true, iotaIndex: 0},
					{unexportedConst: "myEnumFoo", stringName: "foo", isInvalid: false, iotaIndex: 1},
				},
			}},
		},
		{
			name: "no comment derives string name from const name",
			src: `package p
type myEnum int
const (
	myEnumFoo myEnum = iota
	myEnumBar
)`,
			want: []enumType{{
				pkg: "p", typeName: "myEnum", underlyingType: "int",
				values: []enumValue{
					{unexportedConst: "myEnumFoo", stringName: "foo", isInvalid: false, iotaIndex: 0},
					{unexportedConst: "myEnumBar", stringName: "bar", isInvalid: false, iotaIndex: 1},
				},
			}},
		},
		{
			name: "int8 underlying type accepted",
			src: `package p
type myEnum int8
const (
	myEnumFoo myEnum = iota // foo
)`,
			want: []enumType{{
				pkg: "p", typeName: "myEnum", underlyingType: "int8",
				values: []enumValue{
					{unexportedConst: "myEnumFoo", stringName: "foo", isInvalid: false, iotaIndex: 0},
				},
			}},
		},
		{
			name: "int64 underlying type accepted",
			src: `package p
type myEnum int64
const (
	myEnumFoo myEnum = iota // foo
)`,
			want: []enumType{{
				pkg: "p", typeName: "myEnum", underlyingType: "int64",
				values: []enumValue{
					{unexportedConst: "myEnumFoo", stringName: "foo", isInvalid: false, iotaIndex: 0},
				},
			}},
		},
		{
			name: "uint underlying type accepted",
			src: `package p
type myEnum uint
const (
	myEnumFoo myEnum = iota // foo
)`,
			want: []enumType{{
				pkg: "p", typeName: "myEnum", underlyingType: "uint",
				values: []enumValue{
					{unexportedConst: "myEnumFoo", stringName: "foo", isInvalid: false, iotaIndex: 0},
				},
			}},
		},
		{
			name: "uint32 underlying type accepted",
			src: `package p
type myEnum uint32
const (
	myEnumFoo myEnum = iota // foo
)`,
			want: []enumType{{
				pkg: "p", typeName: "myEnum", underlyingType: "uint32",
				values: []enumValue{
					{unexportedConst: "myEnumFoo", stringName: "foo", isInvalid: false, iotaIndex: 0},
				},
			}},
		},
		{
			name: "iota indices track correctly across many values",
			src: `package p
type myEnum int
const (
	myEnumA myEnum = iota // a
	myEnumB               // b
	myEnumC               // c
	myEnumD               // d
)`,
			want: []enumType{{
				pkg: "p", typeName: "myEnum", underlyingType: "int",
				values: []enumValue{
					{unexportedConst: "myEnumA", stringName: "a", isInvalid: false, iotaIndex: 0},
					{unexportedConst: "myEnumB", stringName: "b", isInvalid: false, iotaIndex: 1},
					{unexportedConst: "myEnumC", stringName: "c", isInvalid: false, iotaIndex: 2},
					{unexportedConst: "myEnumD", stringName: "d", isInvalid: false, iotaIndex: 3},
				},
			}},
		},
		{
			name: "comment with extra words uses only first word as string name",
			src: `package p
type myEnum int
const (
	myEnumFoo myEnum = iota // foo this is extra and ignored
)`,
			want: []enumType{{
				pkg: "p", typeName: "myEnum", underlyingType: "int",
				values: []enumValue{
					{unexportedConst: "myEnumFoo", stringName: "foo", isInvalid: false, iotaIndex: 0},
				},
			}},
		},
		{
			name: "INVALID keyword is case insensitive",
			src: `package p
type myEnum int
const (
	myEnumFoo myEnum = iota // INVALID foo
)`,
			want: []enumType{{
				pkg: "p", typeName: "myEnum", underlyingType: "int",
				values: []enumValue{
					{unexportedConst: "myEnumFoo", stringName: "foo", isInvalid: true, iotaIndex: 0},
				},
			}},
		},
		{
			name: "all values marked invalid",
			src: `package p
type myEnum int
const (
	myEnumA myEnum = iota // invalid a
	myEnumB               // invalid b
)`,
			want: []enumType{{
				pkg: "p", typeName: "myEnum", underlyingType: "int",
				values: []enumValue{
					{unexportedConst: "myEnumA", stringName: "a", isInvalid: true, iotaIndex: 0},
					{unexportedConst: "myEnumB", stringName: "b", isInvalid: true, iotaIndex: 1},
				},
			}},
		},
		{
			name: "empty file returns no enums",
			src:  `package p`,
			want: nil,
		},
		{
			name: "quoted string comment allows spaces in name",
			src: `package p
type myEnum int
const (
	myEnumBankTransfer myEnum = iota // "bank transfer"
	myEnumCreditCard                 // "credit card"
)`,
			want: []enumType{{
				pkg: "p", typeName: "myEnum", underlyingType: "int",
				values: []enumValue{
					{unexportedConst: "myEnumBankTransfer", stringName: "bank transfer", isInvalid: false, iotaIndex: 0},
					{unexportedConst: "myEnumCreditCard", stringName: "credit card", isInvalid: false, iotaIndex: 1},
				},
			}},
		},
		{
			name: "invalid with quoted string name",
			src: `package p
type myEnum int
const (
	myEnumUnknown myEnum = iota // invalid "not valid"
	myEnumFoo                   // foo
)`,
			want: []enumType{{
				pkg: "p", typeName: "myEnum", underlyingType: "int",
				values: []enumValue{
					{unexportedConst: "myEnumUnknown", stringName: "not valid", isInvalid: true, iotaIndex: 0},
					{unexportedConst: "myEnumFoo", stringName: "foo", isInvalid: false, iotaIndex: 1},
				},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := tempGoFile(t, tt.src)
			got, err := parseFile(f)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.want == nil {
				if len(got) != 0 {
					t.Errorf("expected no enums, got %d: %+v", len(got), got)
				}
				return
			}
			assertEnumsEqual(t, got, tt.want)
		})
	}
}

func TestParseFile_exportedTypeIsSkipped(t *testing.T) {
	t.Parallel()
	f := tempGoFile(t, `package p
type OrderStatus int
const (
	OrderStatusPending OrderStatus = iota
	OrderStatusConfirmed
)
`)
	enums, err := parseFile(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(enums) != 0 {
		t.Errorf("expected exported type to be skipped, got %d enums", len(enums))
	}
}

func TestParseFile_exportedConstsAreSkipped(t *testing.T) {
	t.Parallel()
	f := tempGoFile(t, `package p
type orderStatus int
const (
	OrderStatusPending orderStatus = iota
	OrderStatusConfirmed
)
`)
	enums, err := parseFile(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(enums) != 0 {
		t.Errorf("expected type with exported consts to be skipped, got %d enums", len(enums))
	}
}

func TestParseFile_syntaxError(t *testing.T) {
	t.Parallel()
	f := tempGoFile(t, `package p
this is not {{{ valid go`)
	_, err := parseFile(f)
	if err == nil {
		t.Fatal("expected error for invalid Go syntax, got nil")
	}
}

func TestParseFile_nonexistentFile(t *testing.T) {
	t.Parallel()
	_, err := parseFile("/nonexistent/path/does_not_exist.go")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestParseComment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		comment     string
		constName   string
		typeName    string
		wantName    string
		wantInvalid bool
	}{
		{
			name:        "plain string name",
			comment:     "filterOnly",
			constName:   "myEnumFilterOnly",
			typeName:    "myEnum",
			wantName:    "filterOnly",
			wantInvalid: false,
		},
		{
			name:        "invalid prefix with name",
			comment:     "invalid unknown",
			constName:   "myEnumUnknown",
			typeName:    "myEnum",
			wantName:    "unknown",
			wantInvalid: true,
		},
		{
			name:        "invalid with no following name derives from const",
			comment:     "invalid",
			constName:   "myEnumUnknown",
			typeName:    "myEnum",
			wantName:    "unknown",
			wantInvalid: true,
		},
		{
			name:        "empty comment derives from const",
			comment:     "",
			constName:   "myEnumFoo",
			typeName:    "myEnum",
			wantName:    "foo",
			wantInvalid: false,
		},
		{
			name:        "INVALID uppercase is case-insensitive",
			comment:     "INVALID foo",
			constName:   "myEnumFoo",
			typeName:    "myEnum",
			wantName:    "foo",
			wantInvalid: true,
		},
		{
			name:        "extra words after name are ignored",
			comment:     "foo extra words",
			constName:   "myEnumFoo",
			typeName:    "myEnum",
			wantName:    "foo",
			wantInvalid: false,
		},
		{
			name:        "invalid with extra words uses only first word after invalid",
			comment:     "invalid foo extra words",
			constName:   "myEnumFoo",
			typeName:    "myEnum",
			wantName:    "foo",
			wantInvalid: true,
		},
		{
			name:        "const name equals type name falls back to full const name lowercased",
			comment:     "",
			constName:   "myEnum",
			typeName:    "myEnum",
			wantName:    "myEnum",
			wantInvalid: false,
		},
		{
			name:        "const with no type prefix in name",
			comment:     "",
			constName:   "unrelatedName",
			typeName:    "myEnum",
			wantName:    "unrelatedName",
			wantInvalid: false,
		},
		{
			name:        "quoted string with spaces",
			comment:     `"filter only"`,
			constName:   "myEnumFilterOnly",
			typeName:    "myEnum",
			wantName:    "filter only",
			wantInvalid: false,
		},
		{
			name:        "invalid with quoted string",
			comment:     `invalid "not valid"`,
			constName:   "myEnumUnknown",
			typeName:    "myEnum",
			wantName:    "not valid",
			wantInvalid: true,
		},
		{
			name:        "quoted string with extra content after closing quote ignored",
			comment:     `"bank transfer" extra`,
			constName:   "myEnumBankTransfer",
			typeName:    "myEnum",
			wantName:    "bank transfer",
			wantInvalid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotName, gotInvalid := parseComment(tt.comment, tt.constName, tt.typeName)
			if gotName != tt.wantName {
				t.Errorf("name: got %q, want %q", gotName, tt.wantName)
			}
			if gotInvalid != tt.wantInvalid {
				t.Errorf("isInvalid: got %v, want %v", gotInvalid, tt.wantInvalid)
			}
		})
	}
}

func TestExportName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		want string
	}{
		{"myEnum", "MyEnum"},
		{"status", "Status"},
		{"myEnumFoo", "MyEnumFoo"},
		{"a", "A"},
		{"ABC", "ABC"},
		{"already", "Already"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			if got := exportName(tt.in); got != tt.want {
				t.Errorf("exportName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
