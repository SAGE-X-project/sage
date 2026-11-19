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

package suites

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"testing"
	"testing/fstest"

	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/core/vectors"
)

var update = flag.Bool("update", false, "rewrite testdata/<suite>.json from the suites")

const (
	spec      = "0.9.0-draft.1.1"
	generator = "sage-core-vectors 0.9.0-draft.1.1"
	// The published files (sage-spec a69e901 v1/vectors), pinned by
	// vectors.TestPublishedFiles.
	published          = "../testdata/v1"
	publishedGenerator = "sage-vectors 1.5.2"
	publishedJCSDesc   = "RFC 8785 JSON Canonicalization Scheme as applied to HPKE response envelopes and A2A agent-card proofs."
)

func runner(t *testing.T, s vectors.Suite, gen string) *vectors.Runner {
	t.Helper()
	r, err := vectors.New(gen, []vectors.Suite{s}, core.WithSpecVersion(spec))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// TestJCSReproducesPublished regenerates the published jcs.json from the
// cases that existed in draft.1 and requires identical bytes.
func TestJCSReproducesPublished(t *testing.T) {
	s := JCS()
	s.Description = publishedJCSDesc
	s.Cases = s.Cases[:4]
	files, err := runner(t, s, publishedGenerator).Generate(nil)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(published + "/jcs.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(files["jcs.json"], want) {
		t.Fatalf("regenerated jcs.json differs from the published file:\n%s", files["jcs.json"])
	}
}

// TestSuiteFiles checks testdata/<suite>.json with the runner and requires
// that regeneration reproduces it byte for byte. Run with -update to
// rewrite the files after a deliberate change.
func TestSuiteFiles(t *testing.T) {
	crypto, err := Crypto()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []vectors.Suite{JCS(), crypto} {
		r := runner(t, s, generator)
		name := vectors.FileName(s.Name)
		// Verify-mode outputs come from the previous file: this suite's own
		// file once it exists, the published one before that, so a new file
		// keeps the published randomised signatures.
		prev := os.DirFS("testdata")
		if _, err := os.Stat("testdata/" + name); errors.Is(err, os.ErrNotExist) {
			prev = os.DirFS(published)
		}
		files, err := r.Generate(prev)
		if err != nil {
			t.Fatal(err)
		}
		if *update {
			if err := os.WriteFile("testdata/"+name, files[name], 0o644); err != nil {
				t.Fatal(err)
			}
		}
		stored, err := os.ReadFile("testdata/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(stored, files[name]) {
			t.Errorf("%s: regeneration changes the file; run go test -update", name)
		}
		if err := r.Check(fstest.MapFS{name: {Data: stored}}); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// TestRejectInputsAreStrict makes sure a reject vector cannot pass for the
// wrong reason, such as an input map the case cannot read.
func TestRejectInputsAreStrict(t *testing.T) {
	for _, in := range []map[string]any{
		{},
		{"json": "{}", "json_hex": "7b7d"},
		{"json": 1},
		{"json_hex": "7B7D"},
		{"json_hex": "7"},
	} {
		if err := rejectSigned(in); err == nil {
			t.Errorf("%v accepted", in)
		} else if r := (*core.Rejection)(nil); errors.As(err, &r) {
			t.Errorf("%v: unreadable input must not be a rejection", in)
		}
	}
	if err := rejectSigned(map[string]any{"json": cardJSON}); err != nil {
		t.Fatalf("the base card must be accepted: %v", err)
	}
	if err := rejectSigned(map[string]any{"json": hpkeResponseJSON}); err != nil {
		t.Fatalf("the base HPKE response must be accepted: %v", err)
	}
}

func isRejection(err error) bool {
	var r *core.Rejection
	return errors.As(err, &r)
}
