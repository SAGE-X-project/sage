// Command registrycoverage identifies declaration-only inventory additions.
// It does not execute source files or certify package initialization or resources.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const declarationOnlyKind = "IMPORT_TYPE_DECLARATIONS_ONLY"

type declarationFile struct {
	Name    string `json:"name"`
	SHA256  string `json:"sha256"`
	Package string `json:"package"`
	Types   int    `json:"types"`
	Kind    string `json:"kind"`
}

func declarationOnly(name string, raw []byte) (declarationFile, error) {
	var empty declarationFile
	if filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
		return empty, errors.New("only production Go files can be classified")
	}
	file, err := parser.ParseFile(token.NewFileSet(), name, raw, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return empty, err
	}
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if strings.HasPrefix(comment.Text, "//go:") || strings.HasPrefix(comment.Text, "//line ") ||
				strings.HasPrefix(comment.Text, "/*line ") || constraint.IsPlusBuild(comment.Text) {
				return empty, errors.New("compiler directives require separate inventory review")
			}
		}
	}
	types := 0
	for _, declaration := range file.Decls {
		group, ok := declaration.(*ast.GenDecl)
		if !ok {
			return empty, errors.New("function or method declarations require executable coverage")
		}
		switch group.Tok {
		case token.TYPE:
			types += len(group.Specs)
		case token.IMPORT:
			for _, specification := range group.Specs {
				item, ok := specification.(*ast.ImportSpec)
				if !ok {
					return empty, errors.New("invalid import declaration")
				}
				path, err := strconv.Unquote(item.Path.Value)
				if err != nil || path == "C" || (item.Name != nil && (item.Name.Name == "_" || item.Name.Name == ".")) {
					return empty, errors.New("cgo, blank or dot imports require separate inventory review")
				}
			}
		default:
			return empty, errors.New("value declarations require separate inventory review")
		}
	}
	if types == 0 {
		return empty, errors.New("no type declarations")
	}
	expression := false
	ast.Inspect(file, func(node ast.Node) bool {
		switch value := node.(type) {
		case *ast.CallExpr, *ast.FuncLit:
			expression = true
		case *ast.UnaryExpr:
			expression = expression || value.Op == token.ARROW
		}
		return !expression
	})
	if expression {
		return empty, errors.New("value expressions require separate inventory review")
	}
	digest := sha256.Sum256(raw)
	return declarationFile{Name: name, SHA256: hex.EncodeToString(digest[:]), Package: file.Name.Name, Types: types, Kind: declarationOnlyKind}, nil
}

func collect(root *os.Root, names []string) ([]declarationFile, error) {
	files := make([]declarationFile, 0, len(names))
	seen := make(map[string]bool)
	for _, name := range names {
		if name == "." || name == ".." || filepath.Base(name) != name || strings.ContainsAny(name, "/\\") || seen[name] {
			return nil, errors.New("source names must be unique basenames")
		}
		seen[name] = true
		info, err := root.Lstat(name)
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, errors.New("source must be a regular file")
		}
		raw, err := root.ReadFile(name)
		if err != nil {
			return nil, err
		}
		file, err := declarationOnly(name, raw)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		files = append(files, file)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	return files, nil
}

func run(args []string, output io.Writer) error {
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if len(args) < 2 {
		return errors.New("usage: registrycoverage directory source.go [source.go ...]")
	}
	root, err := os.OpenRoot(args[0])
	if err != nil {
		return err
	}
	files, readErr := collect(root, args[1:])
	closeErr := root.Close()
	if readErr != nil {
		return readErr
	}
	if closeErr != nil {
		return closeErr
	}
	return json.NewEncoder(output).Encode(files)
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
