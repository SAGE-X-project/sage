// Command codegraph builds an AST/type-resolved dependency graph of a Go module.
//
// It loads every package under a module root with go/packages, extracts
// packages, declared symbols (funcs, methods, types, interfaces), and the
// relationships between them (imports, calls, embeds, implements), and writes:
//
//   - graph.json   : machine-readable node/edge list
//   - summary.md   : human-readable metrics (fan-in/out, cycles, interfaces,
//     duplicate-body candidates, dead-code candidates)
//
// Usage:
//
//	go run . -dir ../.. -out ../../docs/refactoring/graph
//	go run . -dir ../.. -out ./out -diff ./out/graph.json   # report delta vs previous run
//
// It is a standalone module so the main SAGE module does not gain a
// golang.org/x/tools dependency.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

// ---- graph model -----------------------------------------------------------

type PackageNode struct {
	Path       string   `json:"path"`
	Name       string   `json:"name"`
	Dir        string   `json:"dir"`
	Kind       string   `json:"kind"` // cmd | internal | pkg | tools | other
	Files      int      `json:"files"`
	TestFiles  int      `json:"test_files"`
	LOC        int      `json:"loc"`
	TestLOC    int      `json:"test_loc"`
	Funcs      int      `json:"funcs"`
	Types      int      `json:"types"`
	Interfaces int      `json:"interfaces"`
	Imports    []string `json:"imports"`          // module-internal imports
	External   []string `json:"external_imports"` // third-party / std imports
}

type SymbolNode struct {
	ID       string `json:"id"` // pkgpath.[Recv.]Name
	Package  string `json:"package"`
	Name     string `json:"name"`
	Kind     string `json:"kind"` // func | method | struct | interface | alias | other
	Receiver string `json:"receiver,omitempty"`
	Exported bool   `json:"exported"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Lines    int    `json:"lines"`
	Sig      string `json:"sig,omitempty"`
	Methods  int    `json:"methods,omitempty"` // for interfaces
}

type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"` // import | call | ref | embeds | implements
}

type Graph struct {
	Module   string        `json:"module"`
	Packages []PackageNode `json:"packages"`
	Symbols  []SymbolNode  `json:"symbols"`
	Edges    []Edge        `json:"edges"`
}

// ---- main ------------------------------------------------------------------

func main() {
	dir := flag.String("dir", ".", "module root directory")
	out := flag.String("out", "./out", "output directory")
	diff := flag.String("diff", "", "previous graph.json to diff against")
	flag.Parse()

	root, err := filepath.Abs(*dir)
	must(err)

	var prev *Graph
	if *diff != "" {
		prev = loadGraph(*diff)
	}

	g := build(root)
	must(os.MkdirAll(*out, 0o755))
	writeJSON(filepath.Join(*out, "graph.json"), g)
	must(os.WriteFile(filepath.Join(*out, "summary.md"), []byte(summarize(g)), 0o644))
	if prev != nil {
		must(os.WriteFile(filepath.Join(*out, "delta.md"), []byte(delta(prev, g)), 0o644))
	}
	fmt.Printf("codegraph: %d packages, %d symbols, %d edges -> %s\n",
		len(g.Packages), len(g.Symbols), len(g.Edges), *out)
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "codegraph:", err)
		os.Exit(1)
	}
}

func loadGraph(path string) *Graph {
	b, err := os.ReadFile(path)
	must(err)
	var g Graph
	must(json.Unmarshal(b, &g))
	return &g
}

func writeJSON(path string, v any) {
	b, err := json.MarshalIndent(v, "", " ")
	must(err)
	must(os.WriteFile(path, b, 0o644))
}

// ---- build -----------------------------------------------------------------

func build(root string) *Graph {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax |
			packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedModule,
		Dir:   root,
		Tests: false,
	}
	pkgs, err := packages.Load(cfg, "./...")
	must(err)

	g := &Graph{}
	if len(pkgs) > 0 && pkgs[0].Module != nil {
		g.Module = pkgs[0].Module.Path
	}
	modPrefix := g.Module + "/"
	isInternal := func(p string) bool { return p == g.Module || strings.HasPrefix(p, modPrefix) }

	// symbol id lookup for *types.Object -> id
	objID := map[types.Object]string{}
	funcNodes := map[string]*SymbolNode{}
	var namedTypes []*types.Named
	var ifaces []*types.Named

	for _, p := range pkgs {
		if len(p.Errors) > 0 {
			for _, e := range p.Errors {
				fmt.Fprintln(os.Stderr, "warn:", e)
			}
		}
		if !isInternal(p.PkgPath) {
			continue
		}
		rel, _ := filepath.Rel(root, pkgDir(p))
		pn := PackageNode{Path: p.PkgPath, Name: p.Name, Dir: rel, Kind: pkgKind(rel)}
		for _, f := range p.GoFiles {
			pn.Files++
			pn.LOC += countLines(f)
		}
		// test files are not loaded (Tests=false); count them from disk
		matches, _ := filepath.Glob(filepath.Join(pkgDir(p), "*_test.go"))
		for _, f := range matches {
			pn.TestFiles++
			pn.TestLOC += countLines(f)
		}
		for imp := range p.Imports {
			if isInternal(imp) {
				pn.Imports = append(pn.Imports, imp)
				g.Edges = append(g.Edges, Edge{From: p.PkgPath, To: imp, Kind: "import"})
			} else {
				pn.External = append(pn.External, imp)
			}
		}
		sort.Strings(pn.Imports)
		sort.Strings(pn.External)

		for _, file := range p.Syntax {
			fname, _ := filepath.Rel(root, p.Fset.Position(file.Pos()).Filename)
			if strings.HasPrefix(fname, "..") { // cgo-generated files live in the build cache
				continue
			}
			for _, decl := range file.Decls {
				switch d := decl.(type) {
				case *ast.FuncDecl:
					obj := p.TypesInfo.Defs[d.Name]
					if obj == nil {
						continue
					}
					fn := obj.(*types.Func)
					sn := SymbolNode{
						Package:  p.PkgPath,
						Name:     d.Name.Name,
						Kind:     "func",
						Exported: d.Name.IsExported(),
						File:     fname,
						Line:     p.Fset.Position(d.Pos()).Line,
						Lines:    p.Fset.Position(d.End()).Line - p.Fset.Position(d.Pos()).Line + 1,
						Sig:      types.TypeString(fn.Type(), types.RelativeTo(p.Types)),
					}
					if d.Recv != nil && len(d.Recv.List) > 0 {
						sn.Kind = "method"
						sn.Receiver = recvName(d.Recv.List[0].Type)
						sn.ID = p.PkgPath + "." + sn.Receiver + "." + sn.Name
					} else {
						sn.ID = p.PkgPath + "." + sn.Name
					}
					objID[fn] = sn.ID
					g.Symbols = append(g.Symbols, sn)
					funcNodes[sn.ID] = &g.Symbols[len(g.Symbols)-1]
					pn.Funcs++
				case *ast.GenDecl:
					if d.Tok != token.TYPE {
						continue
					}
					for _, spec := range d.Specs {
						ts := spec.(*ast.TypeSpec)
						obj := p.TypesInfo.Defs[ts.Name]
						if obj == nil {
							continue
						}
						sn := SymbolNode{
							ID:       p.PkgPath + "." + ts.Name.Name,
							Package:  p.PkgPath,
							Name:     ts.Name.Name,
							Exported: ts.Name.IsExported(),
							File:     fname,
							Line:     p.Fset.Position(ts.Pos()).Line,
							Lines:    p.Fset.Position(ts.End()).Line - p.Fset.Position(ts.Pos()).Line + 1,
						}
						switch ut := ts.Type.(type) {
						case *ast.StructType:
							sn.Kind = "struct"
							for _, f := range ut.Fields.List {
								if len(f.Names) == 0 { // embedded
									if t := p.TypesInfo.TypeOf(f.Type); t != nil {
										if nt, ok := deref(t).(*types.Named); ok && nt.Obj().Pkg() != nil && isInternal(nt.Obj().Pkg().Path()) {
											g.Edges = append(g.Edges, Edge{From: sn.ID, To: nt.Obj().Pkg().Path() + "." + nt.Obj().Name(), Kind: "embeds"})
										}
									}
								}
							}
						case *ast.InterfaceType:
							sn.Kind = "interface"
							sn.Methods = ut.Methods.NumFields()
							pn.Interfaces++
						default:
							if ts.Assign.IsValid() {
								sn.Kind = "alias"
							} else {
								sn.Kind = "other"
							}
						}
						if nt, ok := obj.Type().(*types.Named); ok {
							namedTypes = append(namedTypes, nt)
							if sn.Kind == "interface" && sn.Methods > 0 {
								ifaces = append(ifaces, nt)
							}
						}
						g.Symbols = append(g.Symbols, sn)
						pn.Types++
					}
				}
			}
		}
		g.Packages = append(g.Packages, pn)
	}

	// call edges (second pass: needs objID complete)
	seen := map[Edge]bool{}
	for _, p := range pkgs {
		if !isInternal(p.PkgPath) {
			continue
		}
		for _, file := range p.Syntax {
			for _, decl := range file.Decls {
				fd, ok := decl.(*ast.FuncDecl)
				if !ok || fd.Body == nil {
					continue
				}
				from := objID[p.TypesInfo.Defs[fd.Name]]
				if from == "" {
					continue
				}
				// idents in call position -> "call"; other references to funcs -> "ref"
				callPos := map[*ast.Ident]bool{}
				ast.Inspect(fd.Body, func(n ast.Node) bool {
					if call, ok := n.(*ast.CallExpr); ok {
						switch f := call.Fun.(type) {
						case *ast.Ident:
							callPos[f] = true
						case *ast.SelectorExpr:
							callPos[f.Sel] = true
						}
					}
					return true
				})
				ast.Inspect(fd.Body, func(n ast.Node) bool {
					id, ok := n.(*ast.Ident)
					if !ok {
						return true
					}
					fn, ok := p.TypesInfo.Uses[id].(*types.Func)
					if !ok {
						return true
					}
					to := objID[fn]
					if to == "" {
						// interface method: build an id when the interface is module-internal
						if sig, ok := fn.Type().(*types.Signature); ok && sig.Recv() != nil {
							if nt, ok := deref(sig.Recv().Type()).(*types.Named); ok && nt.Obj().Pkg() != nil && isInternal(nt.Obj().Pkg().Path()) {
								if _, isIface := nt.Underlying().(*types.Interface); isIface {
									to = nt.Obj().Pkg().Path() + "." + nt.Obj().Name() + "." + fn.Name()
								}
							}
						}
					}
					if to == "" || to == from {
						return true
					}
					kind := "ref"
					if callPos[id] {
						kind = "call"
					}
					e := Edge{From: from, To: to, Kind: kind}
					if !seen[e] {
						seen[e] = true
						g.Edges = append(g.Edges, e)
					}
					return true
				})
			}
		}
	}

	// references from package-level var/const initialisers (e.g. cobra RunE: runX)
	for _, p := range pkgs {
		if !isInternal(p.PkgPath) {
			continue
		}
		for _, file := range p.Syntax {
			for _, decl := range file.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok || (gd.Tok != token.VAR && gd.Tok != token.CONST) {
					continue
				}
				for _, spec := range gd.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok || len(vs.Names) == 0 {
						continue
					}
					from := p.PkgPath + "." + vs.Names[0].Name
					for _, v := range vs.Values {
						ast.Inspect(v, func(n ast.Node) bool {
							id, ok := n.(*ast.Ident)
							if !ok {
								return true
							}
							fn, ok := p.TypesInfo.Uses[id].(*types.Func)
							if !ok {
								return true
							}
							if to := objID[fn]; to != "" {
								e := Edge{From: from, To: to, Kind: "ref"}
								if !seen[e] {
									seen[e] = true
									g.Edges = append(g.Edges, e)
								}
							}
							return true
						})
					}
				}
			}
		}
	}

	// implements edges
	for _, nt := range namedTypes {
		if _, isIface := nt.Underlying().(*types.Interface); isIface {
			continue
		}
		for _, in := range ifaces {
			iface := in.Underlying().(*types.Interface)
			if types.Implements(nt, iface) || types.Implements(types.NewPointer(nt), iface) {
				g.Edges = append(g.Edges, Edge{
					From: nt.Obj().Pkg().Path() + "." + nt.Obj().Name(),
					To:   in.Obj().Pkg().Path() + "." + in.Obj().Name(),
					Kind: "implements",
				})
			}
		}
	}

	// duplicate-body hashes stored in a side table on Sig? keep separate: computed in summary via re-parse is costly,
	// so compute here and stash into a global.
	dupExact, dupStruct = hashBodies(pkgs, isInternal, objID)

	sort.Slice(g.Packages, func(i, j int) bool { return g.Packages[i].Path < g.Packages[j].Path })
	sort.Slice(g.Symbols, func(i, j int) bool { return g.Symbols[i].ID < g.Symbols[j].ID })
	sort.Slice(g.Edges, func(i, j int) bool {
		a, b := g.Edges[i], g.Edges[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.From != b.From {
			return a.From < b.From
		}
		return a.To < b.To
	})
	return g
}

var dupExact, dupStruct map[string][]string

// hashBodies returns exact-body and structural-body hash groups (only groups with >1 member).
func hashBodies(pkgs []*packages.Package, isInternal func(string) bool, objID map[types.Object]string) (map[string][]string, map[string][]string) {
	exact := map[string][]string{}
	structural := map[string][]string{}
	for _, p := range pkgs {
		if !isInternal(p.PkgPath) {
			continue
		}
		for _, file := range p.Syntax {
			for _, decl := range file.Decls {
				fd, ok := decl.(*ast.FuncDecl)
				if !ok || fd.Body == nil {
					continue
				}
				id := objID[p.TypesInfo.Defs[fd.Name]]
				lines := p.Fset.Position(fd.End()).Line - p.Fset.Position(fd.Pos()).Line
				if lines < 8 { // ignore trivial bodies
					continue
				}
				var sb strings.Builder
				_ = printer.Fprint(&sb, p.Fset, fd.Body)
				exact[sum(sb.String())] = append(exact[sum(sb.String())], id)

				var tokens []string
				ast.Inspect(fd.Body, func(n ast.Node) bool {
					switch x := n.(type) {
					case *ast.Ident:
						tokens = append(tokens, "I")
					case *ast.BasicLit:
						tokens = append(tokens, "L"+x.Kind.String())
					case nil:
					default:
						tokens = append(tokens, fmt.Sprintf("%T", n))
					}
					return true
				})
				structural[sum(strings.Join(tokens, ","))] = append(structural[sum(strings.Join(tokens, ","))], id)
			}
		}
	}
	prune := func(m map[string][]string) map[string][]string {
		out := map[string][]string{}
		for k, v := range m {
			if len(v) > 1 {
				sort.Strings(v)
				out[k] = v
			}
		}
		return out
	}
	return prune(exact), prune(structural)
}

func sum(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:8])
}

func pkgDir(p *packages.Package) string {
	if len(p.GoFiles) > 0 {
		return filepath.Dir(p.GoFiles[0])
	}
	if len(p.OtherFiles) > 0 {
		return filepath.Dir(p.OtherFiles[0])
	}
	return ""
}

func pkgKind(rel string) string {
	switch {
	case rel == "" || rel == ".":
		return "root"
	case strings.HasPrefix(rel, "cmd/"):
		return "cmd"
	case strings.HasPrefix(rel, "internal/"):
		return "internal"
	case strings.HasPrefix(rel, "pkg/"):
		return "pkg"
	case strings.HasPrefix(rel, "tools/"):
		return "tools"
	case strings.HasPrefix(rel, "examples/"):
		return "examples"
	case strings.HasPrefix(rel, "tests/"):
		return "tests"
	case strings.HasPrefix(rel, "contracts/"):
		return "contracts"
	case strings.HasPrefix(rel, "lib/"):
		return "lib"
	}
	return "other"
}

func recvName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return recvName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return recvName(t.X)
	case *ast.IndexListExpr:
		return recvName(t.X)
	}
	return "?"
}

func deref(t types.Type) types.Type {
	if p, ok := t.(*types.Pointer); ok {
		return p.Elem()
	}
	return t
}

func countLines(path string) int {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	return strings.Count(string(b), "\n")
}

// ---- summary ---------------------------------------------------------------

func summarize(g *Graph) string {
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }

	pkgByPath := map[string]*PackageNode{}
	for i := range g.Packages {
		pkgByPath[g.Packages[i].Path] = &g.Packages[i]
	}
	fanIn := map[string]int{}
	fanOut := map[string]int{}
	callIn := map[string]int{}
	callOut := map[string]int{}
	impls := map[string][]string{}
	for _, e := range g.Edges {
		switch e.Kind {
		case "import":
			fanOut[e.From]++
			fanIn[e.To]++
		case "call", "ref":
			callOut[e.From]++
			callIn[e.To]++
		case "implements":
			impls[e.To] = append(impls[e.To], e.From)
		}
	}

	totLOC, totTest, totFuncs, totTypes := 0, 0, 0, 0
	for _, p := range g.Packages {
		totLOC += p.LOC
		totTest += p.TestLOC
		totFuncs += p.Funcs
		totTypes += p.Types
	}
	w("# Code Graph Summary\n\n")
	w("Module: `%s`\n\n", g.Module)
	w("| Metric | Value |\n|---|---|\n")
	w("| Packages | %d |\n| Symbols | %d |\n| Edges | %d |\n", len(g.Packages), len(g.Symbols), len(g.Edges))
	w("| Non-test LOC | %d |\n| Test LOC | %d |\n| Funcs+methods | %d |\n| Types | %d |\n\n", totLOC, totTest, totFuncs, totTypes)

	w("## Packages\n\n")
	w("| Package | Kind | Files | LOC | Test LOC | Funcs | Types | Ifaces | Fan-in | Fan-out | Ext deps |\n|---|---|---|---|---|---|---|---|---|---|---|\n")
	for _, p := range g.Packages {
		w("| %s | %s | %d | %d | %d | %d | %d | %d | %d | %d | %d |\n",
			short(g.Module, p.Path), p.Kind, p.Files, p.LOC, p.TestLOC, p.Funcs, p.Types, p.Interfaces,
			fanIn[p.Path], fanOut[p.Path], len(p.External))
	}

	w("\n## Package import graph (module-internal)\n\n```mermaid\ngraph LR\n")
	for _, e := range g.Edges {
		if e.Kind == "import" {
			w("  %s --> %s\n", mid(g.Module, e.From), mid(g.Module, e.To))
		}
	}
	w("```\n\n")

	w("## Import cycles (SCC size > 1)\n\n")
	cycles := scc(g)
	if len(cycles) == 0 {
		w("None.\n\n")
	} else {
		for _, c := range cycles {
			w("- %s\n", strings.Join(mapShort(g.Module, c), " <-> "))
		}
		w("\n")
	}

	w("## Layer violations\n\n")
	w("Rule: pkg must not import cmd/internal; internal must not import cmd; examples/tests/tools must not be imported by pkg/internal/cmd.\n\n")
	viol := 0
	for _, e := range g.Edges {
		if e.Kind != "import" {
			continue
		}
		fk, tk := pkgByPath[e.From].Kind, pkgByPath[e.To].Kind
		bad := false
		switch fk {
		case "pkg":
			bad = tk == "cmd" || tk == "internal" || tk == "examples" || tk == "tests" || tk == "tools"
		case "internal":
			bad = tk == "cmd" || tk == "examples" || tk == "tests" || tk == "tools"
		case "cmd":
			bad = tk == "examples" || tk == "tests" || tk == "tools"
		}
		if bad {
			viol++
			w("- %s (%s) -> %s (%s)\n", short(g.Module, e.From), fk, short(g.Module, e.To), tk)
		}
	}
	if viol == 0 {
		w("None.\n")
	}
	w("\n")

	w("## Interfaces and implementers\n\n| Interface | Methods | Implementers (module-internal) |\n|---|---|---|\n")
	for _, s := range g.Symbols {
		if s.Kind != "interface" || s.Methods == 0 {
			continue
		}
		im := impls[s.ID]
		sort.Strings(im)
		w("| %s | %d | %s |\n", short(g.Module, s.ID), s.Methods, strings.Join(mapShort(g.Module, im), ", "))
	}

	w("\n## Most-called functions (top 25 by internal call fan-in)\n\n| Function | Callers |\n|---|---|\n")
	type kv struct {
		k string
		v int
	}
	var top []kv
	for k, v := range callIn {
		top = append(top, kv{k, v})
	}
	sort.Slice(top, func(i, j int) bool {
		if top[i].v != top[j].v {
			return top[i].v > top[j].v
		}
		return top[i].k < top[j].k
	})
	for i, t := range top {
		if i >= 25 {
			break
		}
		w("| %s | %d |\n", short(g.Module, t.k), t.v)
	}

	w("\n## Largest functions (top 25 by lines)\n\n| Function | Lines | File |\n|---|---|---|\n")
	var fs []SymbolNode
	for _, s := range g.Symbols {
		if s.Kind == "func" || s.Kind == "method" {
			fs = append(fs, s)
		}
	}
	sort.Slice(fs, func(i, j int) bool { return fs[i].Lines > fs[j].Lines })
	for i, s := range fs {
		if i >= 25 {
			break
		}
		w("| %s | %d | %s:%d |\n", short(g.Module, s.ID), s.Lines, s.File, s.Line)
	}

	w("\n## Duplicate function bodies\n\n")
	w("### Exact duplicates (identical body text, >= 8 lines)\n\n")
	writeGroups(&b, g.Module, dupExact)
	w("\n### Structural duplicates (identical AST shape ignoring identifiers/literals, >= 8 lines)\n\n")
	writeGroups(&b, g.Module, dupStruct)

	w("\n## Dead-code candidates (unexported funcs/methods with no internal callers or references)\n\n")
	n := 0
	for _, s := range g.Symbols {
		if (s.Kind != "func" && s.Kind != "method") || s.Exported || callIn[s.ID] > 0 {
			continue
		}
		if s.Name == "init" || s.Name == "main" || strings.HasPrefix(s.Name, "Test") || strings.HasPrefix(s.Name, "Benchmark") {
			continue
		}
		// methods may satisfy interfaces dynamically; only flag plain funcs and
		// methods that are not named like interface methods we know.
		n++
		w("- %s (%s:%d)\n", short(g.Module, s.ID), s.File, s.Line)
	}
	if n == 0 {
		w("None.\n")
	}
	return b.String()
}

func writeGroups(b *strings.Builder, mod string, groups map[string][]string) {
	if len(groups) == 0 {
		b.WriteString("None.\n")
		return
	}
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return groups[keys[i]][0] < groups[keys[j]][0] })
	for _, k := range keys {
		fmt.Fprintf(b, "- %s\n", strings.Join(mapShort(mod, groups[k]), ", "))
	}
}

func short(mod, id string) string { return strings.TrimPrefix(id, mod+"/") }
func mid(mod, id string) string {
	return strings.NewReplacer("/", "_", ".", "_", "-", "_").Replace(short(mod, id))
}
func mapShort(mod string, in []string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = short(mod, s)
	}
	return out
}

// scc returns strongly connected components of size > 1 in the package import graph (Tarjan).
func scc(g *Graph) [][]string {
	adj := map[string][]string{}
	for _, e := range g.Edges {
		if e.Kind == "import" {
			adj[e.From] = append(adj[e.From], e.To)
		}
	}
	index := 0
	idx := map[string]int{}
	low := map[string]int{}
	onStack := map[string]bool{}
	var stack []string
	var out [][]string
	var strong func(v string)
	strong = func(v string) {
		idx[v], low[v] = index, index
		index++
		stack = append(stack, v)
		onStack[v] = true
		for _, w := range adj[v] {
			if _, ok := idx[w]; !ok {
				strong(w)
				low[v] = min(low[v], low[w])
			} else if onStack[w] {
				low[v] = min(low[v], idx[w])
			}
		}
		if low[v] == idx[v] {
			var comp []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				comp = append(comp, w)
				if w == v {
					break
				}
			}
			if len(comp) > 1 {
				sort.Strings(comp)
				out = append(out, comp)
			}
		}
	}
	for _, p := range g.Packages {
		if _, ok := idx[p.Path]; !ok {
			strong(p.Path)
		}
	}
	return out
}

// ---- delta -----------------------------------------------------------------

func delta(prev, cur *Graph) string {
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }
	w("# Code Graph Delta\n\n")
	ps, pe := setOf(prev), edgeSet(prev)
	cs, ce := setOf(cur), edgeSet(cur)
	w("| Metric | Before | After |\n|---|---|---|\n")
	w("| Packages | %d | %d |\n| Symbols | %d | %d |\n| Edges | %d | %d |\n\n",
		len(prev.Packages), len(cur.Packages), len(prev.Symbols), len(cur.Symbols), len(prev.Edges), len(cur.Edges))
	w("## Added symbols\n\n")
	list(&b, diffKeys(cs, ps), cur.Module)
	w("\n## Removed symbols\n\n")
	list(&b, diffKeys(ps, cs), cur.Module)
	w("\n## Added edges\n\n")
	list(&b, diffKeys(ce, pe), cur.Module)
	w("\n## Removed edges\n\n")
	list(&b, diffKeys(pe, ce), cur.Module)
	// external dependency changes
	w("\n## External import changes\n\n")
	pext, cext := extSet(prev), extSet(cur)
	added, removed := diffKeys(cext, pext), diffKeys(pext, cext)
	if len(added)+len(removed) == 0 {
		w("None.\n")
	}
	for _, a := range added {
		w("- + %s\n", a)
	}
	for _, r := range removed {
		w("- - %s\n", r)
	}
	return b.String()
}

func setOf(g *Graph) map[string]bool {
	m := map[string]bool{}
	for _, s := range g.Symbols {
		m[s.ID] = true
	}
	return m
}
func edgeSet(g *Graph) map[string]bool {
	m := map[string]bool{}
	for _, e := range g.Edges {
		m[e.Kind+": "+e.From+" -> "+e.To] = true
	}
	return m
}
func extSet(g *Graph) map[string]bool {
	m := map[string]bool{}
	for _, p := range g.Packages {
		for _, e := range p.External {
			m[e] = true
		}
	}
	return m
}
func diffKeys(a, b map[string]bool) []string {
	var out []string
	for k := range a {
		if !b[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}
func list(b *strings.Builder, items []string, mod string) {
	if len(items) == 0 {
		b.WriteString("None.\n")
		return
	}
	for _, it := range items {
		fmt.Fprintf(b, "- %s\n", strings.ReplaceAll(it, mod+"/", ""))
	}
}
