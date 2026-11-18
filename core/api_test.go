// SAGE - Secure Agent Guarantee Engine
// Copyright (C) 2025 SAGE-X-project
//
// This file is part of SAGE.
//
// SAGE is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// SAGE is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with SAGE. If not, see <https://www.gnu.org/licenses/>.

package core

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const modulePath = "github.com/sage-x-project/sage/core"

const secretBytes = modulePath + "/secret.Bytes"

// privateKeyTypes are the types that hold private key material. The core
// must not let any of them, or a secret.Bytes holding a key, reach a caller.
var privateKeyTypes = map[string]bool{
	"crypto/ecdh.PrivateKey":                                  true,
	"crypto/ed25519.PrivateKey":                               true,
	"crypto/ecdsa.PrivateKey":                                 true,
	"crypto/rsa.PrivateKey":                                   true,
	"crypto/mlkem.DecapsulationKey768":                        true,
	"crypto/mlkem.DecapsulationKey1024":                       true,
	"crypto/hpke.PrivateKey":                                  true,
	"filippo.io/edwards25519.Scalar":                          true,
	"github.com/decred/dcrd/dcrec/secp256k1/v4.PrivateKey":    true,
	"github.com/decred/dcrd/dcrec/secp256k1/v4.ModNScalar":    true,
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa.PrivKey": true,
}

// allowedByteResults lists exported methods of key-holding types that may
// return bytes, with the reason. Every entry is reviewed.
var allowedByteResults = map[string]string{
	modulePath + "/crypto.x25519Key.ECDH":   "shared secret; required by crypto/ecdh.KeyExchanger for crypto/hpke",
	modulePath + "/crypto.x25519Key.Public": "public key",
}

// allowedMethodNames return fixed redacted text on every type.
var allowedMethodNames = map[string]bool{"String": true, "GoString": true, "Error": true}

// exportLookup returns an importer lookup that finds compiled export data
// with `go list -export`, so imports resolve exactly as the go command
// builds them in module mode.
func exportLookup(t *testing.T) func(string) (io.ReadCloser, error) {
	t.Helper()
	cache := map[string]string{}
	return func(path string) (io.ReadCloser, error) {
		file, ok := cache[path]
		if !ok {
			out, err := exec.Command("go", "list", "-export", "-f", "{{.Export}}", path).Output()
			if err != nil {
				return nil, fmt.Errorf("go list -export %s: %w", path, err)
			}
			file = strings.TrimSpace(string(out))
			cache[path] = file
		}
		if file == "" {
			return nil, fmt.Errorf("no export data for %s", path)
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		return io.NopCloser(bytes.NewReader(data)), nil
	}
}

// checkPackage type-checks the non-test files of dir as importPath and
// returns every exposure it finds.
func checkPackage(t *testing.T, imp types.Importer, dir, importPath string) []string {
	t.Helper()
	fset := token.NewFileSet()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var files []*ast.File
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}
	if len(files) == 0 {
		return nil
	}
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}}
	pkg, err := (&types.Config{Importer: imp}).Check(importPath, fset, files, info)
	if err != nil {
		t.Fatalf("type-check %s: %v", importPath, err)
	}
	// A defined type such as `type Scalar ecdh.PrivateKey` takes the key's
	// unexported structure as its underlying type, so the walk also follows
	// the type expression each declaration names.
	rhs := map[*types.TypeName]types.Type{}
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			if spec, ok := n.(*ast.TypeSpec); ok {
				if tn, ok := info.Defs[spec.Name].(*types.TypeName); ok {
					rhs[tn] = info.TypeOf(spec.Type)
				}
			}
			return true
		})
	}
	return exposures(pkg, rhs)
}

type walker struct {
	pkg      *types.Package
	rhs      map[*types.TypeName]types.Type
	findings []string
	visited  map[types.Type]bool
}

func (w *walker) report(where, what string) {
	w.findings = append(w.findings, where+": "+what)
}

// exposures lists every way a private key type is reachable from the
// exported API of pkg, and every exported method of a key-holding type that
// returns bytes without a reviewed reason.
func exposures(pkg *types.Package, rhs map[*types.TypeName]types.Type) []string {
	w := &walker{pkg: pkg, rhs: rhs}
	scope := pkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		where := pkg.Path() + "." + name
		if obj.Exported() {
			w.visited = map[types.Type]bool{}
			if tn, isType := obj.(*types.TypeName); isType {
				w.walk(where, obj.Type().Underlying())
				if r := rhs[tn]; r != nil {
					w.walk(where, r)
				}
			} else {
				w.walk(where, obj.Type())
			}
			if fn, ok := obj.(*types.Func); ok && pkg.Path() != modulePath+"/secret" {
				w.checkSecretResults(where, fn.Type().(*types.Signature))
			}
		}
		tn, ok := obj.(*types.TypeName)
		if !ok {
			continue
		}
		named, ok := types.Unalias(tn.Type()).(*types.Named)
		if !ok {
			continue
		}
		holder := pkg.Path() != modulePath+"/secret" && holdsKey(named.Underlying(), map[types.Type]bool{})
		for i := 0; i < named.NumMethods(); i++ {
			m := named.Method(i)
			if !m.Exported() {
				continue
			}
			mwhere := pkg.Path() + "." + name + "." + m.Name()
			w.visited = map[types.Type]bool{}
			w.walk(mwhere, m.Type())
			if holder && !allowedMethodNames[m.Name()] {
				if _, ok := allowedByteResults[mwhere]; !ok {
					w.checkByteResults(mwhere, m.Type().(*types.Signature))
				}
			}
		}
	}
	sort.Strings(w.findings)
	return w.findings
}

func typeKey(n *types.Named) string {
	o := n.Obj()
	if o.Pkg() == nil {
		return o.Name()
	}
	return o.Pkg().Path() + "." + o.Name()
}

// walk reports a private key type reachable from t through exported
// structure: struct fields (exported or embedded), elements, signature
// parameters and results, interface methods and type parameter constraints.
// Named types of this module are followed; others stop at their name.
func (w *walker) walk(where string, t types.Type) {
	t = types.Unalias(t)
	if t == nil || w.visited[t] {
		return
	}
	w.visited[t] = true
	switch t := t.(type) {
	case *types.Named:
		key := typeKey(t)
		if privateKeyTypes[key] {
			w.report(where, "exposes "+key)
			return
		}
		if args := t.TypeArgs(); args != nil {
			for i := 0; i < args.Len(); i++ {
				w.walk(where, args.At(i))
			}
		}
		if t.Obj().Pkg() != nil && strings.HasPrefix(t.Obj().Pkg().Path(), modulePath) {
			w.walk(where, t.Underlying())
			if r := w.rhs[t.Obj()]; r != nil {
				w.walk(where, r)
			}
			for i := 0; i < t.NumMethods(); i++ {
				if m := t.Method(i); m.Exported() {
					w.walk(where+"."+m.Name(), m.Type())
				}
			}
		}
	case *types.Pointer:
		w.walk(where, t.Elem())
	case *types.Slice:
		w.walk(where, t.Elem())
	case *types.Array:
		w.walk(where, t.Elem())
	case *types.Map:
		w.walk(where, t.Key())
		w.walk(where, t.Elem())
	case *types.Chan:
		w.walk(where, t.Elem())
	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			if f := t.Field(i); f.Exported() || f.Embedded() {
				w.walk(where+"."+f.Name(), f.Type())
			}
		}
	case *types.Signature:
		for _, tuple := range []*types.Tuple{t.Params(), t.Results()} {
			for i := 0; i < tuple.Len(); i++ {
				w.walk(where, tuple.At(i).Type())
			}
		}
		if tp := t.TypeParams(); tp != nil {
			for i := 0; i < tp.Len(); i++ {
				w.walk(where, tp.At(i).Constraint())
			}
		}
	case *types.Interface:
		for i := 0; i < t.NumMethods(); i++ {
			if m := t.Method(i); m.Exported() {
				w.walk(where+"."+m.Name(), m.Type())
			}
		}
		for i := 0; i < t.NumEmbeddeds(); i++ {
			w.walk(where, t.EmbeddedType(i))
		}
	case *types.Union:
		for i := 0; i < t.Len(); i++ {
			w.walk(where, t.Term(i).Type())
		}
	case *types.TypeParam:
		w.walk(where, t.Constraint())
	}
}

// holdsKey reports whether t contains, through any field, a private key
// type or a secret.Bytes.
func holdsKey(t types.Type, seen map[types.Type]bool) bool {
	t = types.Unalias(t)
	if seen[t] {
		return false
	}
	seen[t] = true
	switch t := t.(type) {
	case *types.Named:
		key := typeKey(t)
		if privateKeyTypes[key] || key == secretBytes {
			return true
		}
		if t.Obj().Pkg() != nil && strings.HasPrefix(t.Obj().Pkg().Path(), modulePath) {
			return holdsKey(t.Underlying(), seen)
		}
	case *types.Pointer:
		return holdsKey(t.Elem(), seen)
	case *types.Slice:
		return holdsKey(t.Elem(), seen)
	case *types.Array:
		return holdsKey(t.Elem(), seen)
	case *types.Map:
		return holdsKey(t.Elem(), seen)
	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			if holdsKey(t.Field(i).Type(), seen) {
				return true
			}
		}
	}
	return false
}

func isBytes(t types.Type) bool {
	t = types.Unalias(t)
	if n, ok := t.(*types.Named); ok && typeKey(n) == secretBytes {
		return true
	}
	var elem types.Type
	switch u := t.Underlying().(type) {
	case *types.Slice:
		elem = u.Elem()
	case *types.Array:
		elem = u.Elem()
	case *types.Basic:
		return u.Kind() == types.String
	default:
		return false
	}
	b, ok := elem.Underlying().(*types.Basic)
	return ok && b.Kind() == types.Byte
}

// checkByteResults flags results of bytes, byte arrays, strings or
// secret.Bytes on a key-holding type: such a method can hand out the key.
func (w *walker) checkByteResults(where string, sig *types.Signature) {
	for i := 0; i < sig.Results().Len(); i++ {
		if isBytes(sig.Results().At(i).Type()) {
			w.report(where, "key-holding type returns bytes")
		}
	}
	for i := 0; i < sig.Params().Len(); i++ {
		if p, ok := types.Unalias(sig.Params().At(i).Type()).(*types.Pointer); ok && isBytes(p.Elem()) {
			w.report(where, "key-holding type writes bytes into a parameter")
		}
	}
}

// checkSecretResults flags exported functions outside package secret that
// return a secret.Bytes; each needs a reviewed reason.
func (w *walker) checkSecretResults(where string, sig *types.Signature) {
	for i := 0; i < sig.Results().Len(); i++ {
		if n, ok := types.Unalias(sig.Results().At(i).Type()).(*types.Named); ok && typeKey(n) == secretBytes {
			if _, allowed := allowedByteResults[where]; !allowed {
				w.report(where, "returns a secret.Bytes")
			}
		}
	}
}

// TestNoExportedPrivateKey type-checks every package of the module and
// fails when its exported API exposes a private key (spec 00-overview.md
// §7.3; design 4.5: the core never hands out a private key).
func TestNoExportedPrivateKey(t *testing.T) {
	imp := importer.ForCompiler(token.NewFileSet(), "gc", exportLookup(t))
	var findings []string
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if path != "." && (d.Name() == "testdata" || strings.HasPrefix(d.Name(), ".")) {
			return filepath.SkipDir
		}
		importPath := modulePath
		if path != "." {
			importPath += "/" + filepath.ToSlash(path)
		}
		findings = append(findings, checkPackage(t, imp, path, importPath)...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		t.Error(f)
	}
}

// TestCheckerFindsKnownExposures runs the checker on a fixture package
// holding one violation per known shape, so a checker that stops working
// fails here.
func TestCheckerFindsKnownExposures(t *testing.T) {
	imp := importer.ForCompiler(token.NewFileSet(), "gc", exportLookup(t))
	findings := checkPackage(t, imp, "testdata/apileak", modulePath+"/testdata/apileak")
	joined := strings.Join(findings, "\n")
	for _, want := range []string{
		"apileak.Direct:",
		"apileak.Holder.PrivateKey:",
		"apileak.Alias:",
		"apileak.Defined:",
		"apileak.Getter:",
		"apileak.Visit:",
		"apileak.Load:",
		"apileak.Exposer.Seed:",
		"apileak.DecapKey:",
		"apileak.Generic:",
		"apileak.Field.Key:",
		"apileak.Slice:",
		"apileak.holder.Bytes: key-holding type returns bytes",
		"apileak.holder.Scalar: key-holding type returns bytes",
		"apileak.holder.Into: key-holding type writes bytes",
		"apileak.secretHolder.Raw: key-holding type returns bytes",
		"apileak.Export: returns a secret.Bytes",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("checker missed %q; findings:\n%s", want, joined)
		}
	}
	for _, clean := range []string{"apileak.Clean", "apileak.holder.String", "apileak.holder.Public:"} {
		if strings.Contains(joined, clean) {
			t.Errorf("checker flagged allowed %q", clean)
		}
	}
}
