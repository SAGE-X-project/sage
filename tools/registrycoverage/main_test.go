package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDeclarationOnlyRequiresTypesWithoutExecutableDeclarations(t *testing.T) {
	for _, test := range []struct {
		name   string
		source string
		valid  bool
	}{
		{"types", "package registry011\ntype A struct{ Value int }; type B = A", true},
		{"imports", "package registry011\nimport t \"time\"\ntype A struct{ Timeout t.Duration }", true},
		{"generic", "package registry011\ntype Box[T any] struct{ Value T }", true},
		{"function-type", "package registry011\ntype Callback func(int) error", true},
		{"function", "package registry011\nfunc F() {}", false},
		{"method", "package registry011\ntype A int\nfunc (A) F() {}", false},
		{"init", "package registry011\nfunc init() {}", false},
		{"variable", "package registry011\ntype A int\nvar Value int", false},
		{"initializer", "package registry011\ntype A int\nvar Value = 1", false},
		{"constant", "package registry011\ntype A int\nconst Value = 1", false},
		{"cgo", "package registry011\nimport \"C\"\ntype A int", false},
		{"blank-import", "package registry011\nimport _ \"time\"\ntype A int", false},
		{"dot-import", "package registry011\nimport . \"time\"\ntype A int", false},
		{"directive", "//go:build linux\n\npackage registry011\ntype A int", false},
		{"legacy-directive", "// +build linux\n\npackage registry011\ntype A int", false},
		{"compact-legacy-directive", "//+build linux\n\npackage registry011\ntype A int", false},
		{"tab-legacy-directive", "// +build\tlinux\n\npackage registry011\ntype A int", false},
		{"leading-tab-legacy-directive", "//\t+build linux\n\npackage registry011\ntype A int", false},
		{"line-directive", "//line other.go:1\npackage registry011\ntype A int", false},
		{"embedded-expression", "package registry011\ntype A [len([1]func(){func(){}})]int", false},
		{"receive-expression", "package registry011\ntype A [<-values]int", false},
		{"empty", "package registry011", false},
		{"malformed", "package registry011\ntype", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			file, err := declarationOnly("declarations.go", []byte(test.source))
			if (err == nil) != test.valid {
				t.Fatalf("classification mismatch: valid=%t err=%v", test.valid, err)
			}
			if test.valid && (file.Name != "declarations.go" || file.Package != "registry011" || file.Types <= 0 || len(file.SHA256) != 64 || file.Kind != declarationOnlyKind) {
				t.Fatal("declaration inventory is not bound to the source file")
			}
		})
	}
}

func TestDeclarationInventoryCommandIsCompleteAndRejectsPartialResults(t *testing.T) {
	directory := t.TempDir()
	for name, source := range map[string]string{
		"a.go":   "package registry011\ntype A int",
		"z.go":   "package registry011\ntype Z string",
		"bad.go": "package registry011\nfunc F() {}",
	} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var output bytes.Buffer
	if err := run([]string{"--", directory, "z.go", "a.go"}, &output); err != nil {
		t.Fatal(err)
	}
	var files []declarationFile
	if err := json.Unmarshal(output.Bytes(), &files); err != nil || len(files) != 2 {
		t.Fatalf("complete inventory missing: %v", err)
	}
	if files[0].Name != "a.go" || files[1].Name != "z.go" || files[0].SHA256 == files[1].SHA256 {
		t.Fatal("inventory order or per-file byte binding changed")
	}
	for _, test := range []struct {
		name string
		args []string
	}{
		{"no-arguments", nil},
		{"separator-only", []string{"--"}},
		{"no-files", []string{directory}},
		{"duplicate", []string{directory, "a.go", "a.go"}},
		{"missing", []string{directory, "a.go", "missing.go"}},
		{"executable", []string{directory, "a.go", "bad.go"}},
		{"parent", []string{directory, "../a.go"}},
		{"nested", []string{directory, "nested/a.go"}},
		{"backslash", []string{directory, "nested\\a.go"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var result bytes.Buffer
			if err := run(test.args, &result); err == nil || result.Len() != 0 {
				t.Fatal("invalid inventory emitted an accepted partial result")
			}
		})
	}
	if err := os.Mkdir(filepath.Join(directory, "directory.go"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("a.go", filepath.Join(directory, "link.go")); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"directory.go", "link.go"} {
		var result bytes.Buffer
		if err := run([]string{directory, name}, &result); err == nil || result.Len() != 0 {
			t.Fatal("nonregular source emitted a classification")
		}
	}
}
