package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"strings"
	"unicode"
)

type enumType struct {
	pkg            string
	typeName       string
	underlyingType string
	values         []enumValue
}

type enumValue struct {
	unexportedConst string
	stringName      string
	isInvalid       bool
	iotaIndex       int
}

func parseFile(filename string) ([]enumType, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filename, err)
	}

	// collect unexported int-like type declarations
	intTypes := map[string]string{} // typeName -> underlyingType
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name == nil {
				continue
			}
			name := ts.Name.Name
			if ast.IsExported(name) {
				continue
			}
			ident, ok := ts.Type.(*ast.Ident)
			if !ok {
				continue
			}
			underlying := ident.Name
			if !isIntType(underlying) {
				continue
			}
			intTypes[name] = underlying
		}
	}

	if len(intTypes) == 0 {
		return nil, nil
	}

	// collect const blocks using those types with iota
	typeValues := map[string][]enumValue{}
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}

		var currentType string
		iotaIdx := 0
		hasIota := false

		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			// detect type change
			if vs.Type != nil {
				if ident, ok := vs.Type.(*ast.Ident); ok {
					if _, known := intTypes[ident.Name]; known {
						currentType = ident.Name
						iotaIdx = 0
					} else {
						currentType = ""
					}
				}
			}

			// check if first value uses iota
			if currentType != "" && !hasIota && len(vs.Values) > 0 {
				if isIota(vs.Values[0]) {
					hasIota = true
				}
			}

			if currentType == "" || !hasIota {
				iotaIdx++
				continue
			}

			for _, nameIdent := range vs.Names {
				constName := nameIdent.Name
				if ast.IsExported(constName) {
					log.Printf("skipping type %s: const %s is already exported; go-enum requires unexported const names", currentType, constName)
					delete(intTypes, currentType)
					currentType = ""
					break
				}
				comment := extractComment(vs.Comment)
				stringName, invalid := parseComment(comment, constName, currentType)

				typeValues[currentType] = append(typeValues[currentType], enumValue{
					unexportedConst: constName,
					stringName:      stringName,
					isInvalid:       invalid,
					iotaIndex:       iotaIdx,
				})
			}
			iotaIdx++
		}
	}

	var result []enumType
	for typeName, underlying := range intTypes {
		vals, ok := typeValues[typeName]
		if !ok || len(vals) == 0 {
			continue
		}
		result = append(result, enumType{
			pkg:            f.Name.Name,
			typeName:       typeName,
			underlyingType: underlying,
			values:         vals,
		})
	}
	return result, nil
}

func isIntType(s string) bool {
	switch s {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64":
		return true
	}
	return false
}

func isIota(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "iota"
}

func extractComment(cg *ast.CommentGroup) string {
	if cg == nil {
		return ""
	}
	var parts []string
	for _, c := range cg.List {
		text := strings.TrimPrefix(c.Text, "//")
		text = strings.TrimSpace(text)
		if text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, " ")
}

// parseComment returns (stringName, isInvalid).
// Comment formats:
//
//	filterOnly          → "filterOnly"
//	"filter only"       → "filter only"   (quoted: spaces allowed)
//	invalid filterOnly  → "filterOnly", invalid
//	invalid "not valid" → "not valid",   invalid
//	invalid             → derived,       invalid
//	(empty)             → derived
//
// Derived name: strip typeName prefix from constName, lowercase first rune.
func parseComment(comment, constName, typeName string) (string, bool) {
	comment = strings.TrimSpace(comment)
	isInvalid := false

	// strip leading "invalid" keyword (case-insensitive)
	lower := strings.ToLower(comment)
	if rest, ok := strings.CutPrefix(lower, "invalid"); ok {
		if rest == "" || rest[0] == ' ' {
			isInvalid = true
			comment = strings.TrimSpace(comment[len("invalid"):])
		}
	}

	if comment == "" {
		return deriveName(constName, typeName), isInvalid
	}

	// quoted string: collect everything between the first pair of double-quotes
	if comment[0] == '"' {
		if end := strings.Index(comment[1:], `"`); end >= 0 {
			return comment[1 : end+1], isInvalid
		}
	}

	// plain token: first word only
	return strings.Fields(comment)[0], isInvalid
}

func deriveName(constName, typeName string) string {
	name := strings.TrimPrefix(constName, typeName)
	if name == "" {
		name = constName
	}
	runes := []rune(name)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}
