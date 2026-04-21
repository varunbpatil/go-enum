package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// behaviorTestSrc is placed in a temp module alongside generated enum code.
// It exercises every method and function the generator produces.
const behaviorTestSrc = `package p

import (
	"database/sql/driver"
	"encoding/json"
	"testing"
)

// stringerHelper lets us test the fmt.Stringer branch in ParseMyEnum.
type stringerHelper struct{ s string }
func (sh stringerHelper) String() string { return sh.s }

// Compile-time interface assertions.
var _ driver.Valuer = MyEnum(0)

func TestString(t *testing.T) {
	tests := []struct {
		v    MyEnum
		want string
	}{
		{MyEnumUnknown, "unknown"},
		{MyEnumFoo, "foo"},
		{MyEnumBar, "bar"},
		{MyEnum(99), "MyEnum(99)"},
	}
	for _, tt := range tests {
		if got := tt.v.String(); got != tt.want {
			t.Errorf("MyEnum(%d).String() = %q, want %q", int(tt.v), got, tt.want)
		}
	}
}

func TestIsValid(t *testing.T) {
	if MyEnumUnknown.IsValid() {
		t.Error("MyEnumUnknown should not be valid")
	}
	if !MyEnumFoo.IsValid() {
		t.Error("MyEnumFoo should be valid")
	}
	if !MyEnumBar.IsValid() {
		t.Error("MyEnumBar should be valid")
	}
	if MyEnum(99).IsValid() {
		t.Error("undefined value 99 should not be valid")
	}
}

func TestParseMyEnum(t *testing.T) {
	t.Run("string name", func(t *testing.T) {
		v, err := ParseMyEnum("foo")
		if err != nil || v != MyEnumFoo {
			t.Errorf("ParseMyEnum(\"foo\") = %v, %v; want MyEnumFoo, nil", v, err)
		}
	})

	t.Run("byte slice", func(t *testing.T) {
		v, err := ParseMyEnum([]byte("bar"))
		if err != nil || v != MyEnumBar {
			t.Errorf("ParseMyEnum([]byte(\"bar\")) = %v, %v; want MyEnumBar, nil", v, err)
		}
	})

	t.Run("passthrough same type", func(t *testing.T) {
		v, err := ParseMyEnum(MyEnumBar)
		if err != nil || v != MyEnumBar {
			t.Errorf("ParseMyEnum(MyEnumBar) = %v, %v; want MyEnumBar, nil", v, err)
		}
	})

	t.Run("fmt.Stringer", func(t *testing.T) {
		v, err := ParseMyEnum(stringerHelper{"foo"})
		if err != nil || v != MyEnumFoo {
			t.Errorf("ParseMyEnum(Stringer{\"foo\"}) = %v, %v; want MyEnumFoo, nil", v, err)
		}
	})

	t.Run("int valid", func(t *testing.T) {
		v, err := ParseMyEnum(int(1))
		if err != nil || v != MyEnumFoo {
			t.Errorf("ParseMyEnum(int(1)) = %v, %v; want MyEnumFoo, nil", v, err)
		}
	})

	t.Run("int64 valid", func(t *testing.T) {
		v, err := ParseMyEnum(int64(2))
		if err != nil || v != MyEnumBar {
			t.Errorf("ParseMyEnum(int64(2)) = %v, %v; want MyEnumBar, nil", v, err)
		}
	})

	t.Run("float64 whole number", func(t *testing.T) {
		v, err := ParseMyEnum(float64(1))
		if err != nil || v != MyEnumFoo {
			t.Errorf("ParseMyEnum(float64(1)) = %v, %v; want MyEnumFoo, nil", v, err)
		}
	})

	t.Run("invalid string returns error", func(t *testing.T) {
		_, err := ParseMyEnum("doesnotexist")
		if err == nil {
			t.Error("ParseMyEnum(unknown string) should return error")
		}
	})

	t.Run("int matching invalid value returns error", func(t *testing.T) {
		_, err := ParseMyEnum(int(0)) // 0 = unknown, IsValid=false
		if err == nil {
			t.Error("ParseMyEnum(0) should return error since value is invalid")
		}
	})

	t.Run("int out of range returns error", func(t *testing.T) {
		_, err := ParseMyEnum(int(99))
		if err == nil {
			t.Error("ParseMyEnum(99) should return error")
		}
	})

	t.Run("float with fractional part returns error", func(t *testing.T) {
		_, err := ParseMyEnum(float64(1.5))
		if err == nil {
			t.Error("ParseMyEnum(1.5) should return error (non-integer)")
		}
	})

	t.Run("unsupported type returns error", func(t *testing.T) {
		_, err := ParseMyEnum(struct{}{})
		if err == nil {
			t.Error("ParseMyEnum(struct{}{}) should return error")
		}
	})
}

func TestAllMyEnums(t *testing.T) {
	var got []MyEnum
	for v := range AllMyEnums() {
		got = append(got, v)
	}
	want := []MyEnum{MyEnumFoo, MyEnumBar}
	if len(got) != len(want) {
		t.Fatalf("AllMyEnums() yielded %d items, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("AllMyEnums()[%d] = %v, want %v", i, got[i], w)
		}
	}
}

func TestAllMyEnums_excludesInvalidValues(t *testing.T) {
	for v := range AllMyEnums() {
		if !v.IsValid() {
			t.Errorf("AllMyEnums() yielded invalid value %v", v)
		}
	}
}

func TestExhaustiveMyEnums(t *testing.T) {
	var got []MyEnum
	ExhaustiveMyEnums(func(v MyEnum) {
		got = append(got, v)
	})
	if len(got) != 2 {
		t.Fatalf("ExhaustiveMyEnums called f %d times, want 2", len(got))
	}
	if got[0] != MyEnumFoo || got[1] != MyEnumBar {
		t.Errorf("ExhaustiveMyEnums values = %v, want [MyEnumFoo, MyEnumBar]", got)
	}
}

func TestMarshalJSON(t *testing.T) {
	b, err := json.Marshal(MyEnumFoo)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if string(b) != "\"foo\"" {
		t.Errorf("MarshalJSON = %s, want \"foo\"", b)
	}
}

func TestUnmarshalJSON(t *testing.T) {
	var v MyEnum
	if err := json.Unmarshal([]byte("\"bar\""), &v); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if v != MyEnumBar {
		t.Errorf("UnmarshalJSON = %v, want MyEnumBar", v)
	}
}

func TestMarshalJSON_roundtrip(t *testing.T) {
	for _, orig := range []MyEnum{MyEnumFoo, MyEnumBar} {
		b, err := json.Marshal(orig)
		if err != nil {
			t.Fatalf("Marshal %v: %v", orig, err)
		}
		var got MyEnum
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("Unmarshal %v: %v", orig, err)
		}
		if got != orig {
			t.Errorf("JSON roundtrip: got %v, want %v", got, orig)
		}
	}
}

func TestUnmarshalJSON_invalidReturnsError(t *testing.T) {
	var v MyEnum
	if err := json.Unmarshal([]byte("\"doesnotexist\""), &v); err == nil {
		t.Error("UnmarshalJSON invalid value should return error")
	}
}

func TestMarshalText(t *testing.T) {
	b, err := MyEnumFoo.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}
	if string(b) != "foo" {
		t.Errorf("MarshalText = %q, want \"foo\"", b)
	}
}

func TestUnmarshalText(t *testing.T) {
	var v MyEnum
	if err := v.UnmarshalText([]byte("bar")); err != nil {
		t.Fatalf("UnmarshalText: %v", err)
	}
	if v != MyEnumBar {
		t.Errorf("UnmarshalText = %v, want MyEnumBar", v)
	}
}

func TestMarshalText_roundtrip(t *testing.T) {
	for _, orig := range []MyEnum{MyEnumFoo, MyEnumBar} {
		b, err := orig.MarshalText()
		if err != nil {
			t.Fatalf("MarshalText %v: %v", orig, err)
		}
		var got MyEnum
		if err := got.UnmarshalText(b); err != nil {
			t.Fatalf("UnmarshalText %v: %v", orig, err)
		}
		if got != orig {
			t.Errorf("Text roundtrip: got %v, want %v", got, orig)
		}
	}
}

func TestMarshalBinary(t *testing.T) {
	b, err := MyEnumFoo.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if string(b) != "foo" {
		t.Errorf("MarshalBinary = %q, want \"foo\"", b)
	}
}

func TestUnmarshalBinary(t *testing.T) {
	var v MyEnum
	if err := v.UnmarshalBinary([]byte("bar")); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if v != MyEnumBar {
		t.Errorf("UnmarshalBinary = %v, want MyEnumBar", v)
	}
}

func TestUnmarshalBinary_invalidReturnsError(t *testing.T) {
	var v MyEnum
	if err := v.UnmarshalBinary([]byte("doesnotexist")); err == nil {
		t.Error("UnmarshalBinary invalid value should return error")
	}
}

func TestMarshalBinary_roundtrip(t *testing.T) {
	for _, orig := range []MyEnum{MyEnumFoo, MyEnumBar} {
		b, err := orig.MarshalBinary()
		if err != nil {
			t.Fatalf("MarshalBinary %v: %v", orig, err)
		}
		var got MyEnum
		if err := got.UnmarshalBinary(b); err != nil {
			t.Fatalf("UnmarshalBinary %v: %v", orig, err)
		}
		if got != orig {
			t.Errorf("Binary roundtrip: got %v, want %v", got, orig)
		}
	}
}

func TestMarshalYAML(t *testing.T) {
	v, err := MyEnumFoo.MarshalYAML()
	if err != nil {
		t.Fatalf("MarshalYAML: %v", err)
	}
	s, ok := v.(string)
	if !ok {
		t.Fatalf("MarshalYAML returned %T, want string", v)
	}
	if s != "foo" {
		t.Errorf("MarshalYAML = %q, want \"foo\"", s)
	}
}

func TestUnmarshalYAML(t *testing.T) {
	var got MyEnum
	err := got.UnmarshalYAML(func(dst interface{}) error {
		*dst.(*string) = "bar"
		return nil
	})
	if err != nil {
		t.Fatalf("UnmarshalYAML: %v", err)
	}
	if got != MyEnumBar {
		t.Errorf("UnmarshalYAML = %v, want MyEnumBar", got)
	}
}

func TestUnmarshalYAML_invalidReturnsError(t *testing.T) {
	var v MyEnum
	err := v.UnmarshalYAML(func(dst interface{}) error {
		*dst.(*string) = "doesnotexist"
		return nil
	})
	if err == nil {
		t.Error("UnmarshalYAML invalid value should return error")
	}
}

func TestSQLValue(t *testing.T) {
	val, err := MyEnumFoo.Value()
	if err != nil {
		t.Fatalf("Value(): %v", err)
	}
	s, ok := val.(string)
	if !ok || s != "foo" {
		t.Errorf("Value() = %v (%T), want \"foo\" (string)", val, val)
	}
}

func TestSQLScan_string(t *testing.T) {
	var v MyEnum
	if err := v.Scan("bar"); err != nil {
		t.Fatalf("Scan(string): %v", err)
	}
	if v != MyEnumBar {
		t.Errorf("Scan(\"bar\") = %v, want MyEnumBar", v)
	}
}

func TestSQLScan_bytes(t *testing.T) {
	var v MyEnum
	if err := v.Scan([]byte("foo")); err != nil {
		t.Fatalf("Scan([]byte): %v", err)
	}
	if v != MyEnumFoo {
		t.Errorf("Scan([]byte(\"foo\")) = %v, want MyEnumFoo", v)
	}
}

func TestSQLScan_invalidReturnsError(t *testing.T) {
	var v MyEnum
	if err := v.Scan("doesnotexist"); err == nil {
		t.Error("Scan(invalid) should return error")
	}
}

func TestSQLRoundtrip(t *testing.T) {
	for _, orig := range []MyEnum{MyEnumFoo, MyEnumBar} {
		val, err := orig.Value()
		if err != nil {
			t.Fatalf("Value %v: %v", orig, err)
		}
		var got MyEnum
		if err := got.Scan(val); err != nil {
			t.Fatalf("Scan %v: %v", orig, err)
		}
		if got != orig {
			t.Errorf("SQL roundtrip: got %v, want %v", got, orig)
		}
	}
}
`

// enumInputSrc is the source file the generator will parse.
const enumInputSrc = `package p

type myEnum int

const (
	myEnumUnknown myEnum = iota // invalid unknown
	myEnumFoo                   // foo
	myEnumBar                   // bar
)
`

func TestIntegration_generatedBehavior(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go binary not in PATH")
	}

	dir := t.TempDir()

	writeFile := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	writeFile("go.mod", "module p\ngo 1.23\n")
	writeFile("enum.go", enumInputSrc)

	// Generate the enum file using the same logic as the CLI.
	enums, err := parseFile(filepath.Join(dir, "enum.go"))
	if err != nil {
		t.Fatalf("parseFile: %v", err)
	}
	if len(enums) == 0 {
		t.Fatal("parseFile returned no enums from test input")
	}

	genFile, err := os.Create(filepath.Join(dir, "enum_enum.go"))
	if err != nil {
		t.Fatalf("create generated file: %v", err)
	}
	generateErr := generate(genFile, enums)
	if err := genFile.Close(); err != nil {
		t.Fatalf("close generated file: %v", err)
	}
	if generateErr != nil {
		t.Fatalf("generate: %v", generateErr)
	}

	writeFile("behavior_test.go", behaviorTestSrc)

	cmd := exec.Command("go", "test", "-v", "-count=1", "./...")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test failed:\n%s", out)
	}
}
