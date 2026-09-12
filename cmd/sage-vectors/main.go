// sage-vectors generates and checks the SAGE protocol test vectors published
// in the sage-spec repository.
//
//	sage-vectors gen   -dir ../sage-spec/vectors   # write (or refresh) every suite
//	sage-vectors check -dir ../sage-spec/vectors   # verify this implementation against them
//	sage-vectors list                               # print suites and vector names
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sage-x-project/sage/pkg/vectors"
	"github.com/sage-x-project/sage/pkg/version"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	fs := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	dir := fs.String("dir", "vectors", "directory holding one JSON file per suite")
	_ = fs.Parse(os.Args[2:])

	var err error
	switch os.Args[1] {
	case "gen", "generate":
		err = vectors.Generate(*dir)
		if err == nil {
			fmt.Printf("wrote %d suites to %s (spec %s, sage %s)\n", len(vectors.Suites()), *dir, vectors.SpecVersion, version.Version)
		}
	case "check", "verify":
		err = vectors.Check(*dir)
		if err == nil {
			n := 0
			for _, s := range vectors.Suites() {
				n += len(s.Cases)
			}
			fmt.Printf("ok: %d vectors in %d suites verified against sage %s\n", n, len(vectors.Suites()), version.Version)
		}
	case "list":
		for _, s := range vectors.Suites() {
			fmt.Printf("%s\n", s.Name)
			for _, c := range s.Cases {
				fmt.Printf("  %-32s %s\n", c.Name, c.Mode)
			}
		}
	case "version":
		fmt.Printf("sage-vectors %s (spec %s)\n", version.Version, vectors.SpecVersion)
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: sage-vectors <gen|check|list|version> [-dir DIR]")
}
