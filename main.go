package main

import (
	"bytes"
	"fmt"
	"go/format"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: go-enum <file.go> [...]")
		os.Exit(1)
	}

	for _, inputFile := range os.Args[1:] {
		if err := process(inputFile); err != nil {
			log.Fatalf("%s: %v", inputFile, err)
		}
	}
}

func process(inputFile string) error {
	enums, err := parseFile(inputFile)
	if err != nil {
		return err
	}
	if len(enums) == 0 {
		return nil
	}

	ext := filepath.Ext(inputFile)
	outputFile := strings.TrimSuffix(inputFile, ext) + ".enums.go"

	var buf bytes.Buffer
	if err := generate(&buf, enums); err != nil {
		return err
	}
	src, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("format generated source: %w", err)
	}

	f, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	_, err = f.Write(src)
	return err
}
