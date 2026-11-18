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

package vectors

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/core/obs"
)

const spec = string(core.V0_9_0_Draft1_1)

func sum(in map[string]any) (map[string]any, error) {
	msg, _ := in["message"].(string)
	h := sha256.Sum256([]byte(msg))
	return map[string]any{"sha256": hex.EncodeToString(h[:])}, nil
}

// nonceCase produces a random nonce and verifies only its shape, like a
// randomised signature.
func nonceProduce(map[string]any) (map[string]any, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return map[string]any{"nonce": hex.EncodeToString(b)}, nil
}

func nonceVerify(_, out map[string]any) error {
	n := out["nonce"].(string) // panics on a hostile type, as legacy cases do
	if b, err := hex.DecodeString(n); err != nil || len(b) != 8 {
		return errors.New("bad nonce")
	}
	return nil
}

func rejectEmpty(in map[string]any) error {
	if msg, _ := in["message"].(string); msg == "" {
		return core.Reject(obs.ReasonMalformed, obs.Known("empty message"))
	}
	return nil
}

func toySuite(reject func(map[string]any) error) Suite {
	return Suite{
		Name:        "toy",
		Description: "runner self-test",
		Cases: []Case{
			{Name: "sum", Mode: ModeDeterministic, Input: map[string]any{"message": "abc"}, Produce: sum},
			{Name: "nonce", Mode: ModeVerify, Input: map[string]any{"n": 1}, Produce: nonceProduce, Verify: nonceVerify},
			{Name: "empty", Mode: ModeReject, Description: "rejected at S1", Input: map[string]any{"message": ""},
				Reject: reject, Reason: obs.ReasonMalformed},
		},
	}
}

func newRunner(t *testing.T, suites ...Suite) *Runner {
	t.Helper()
	r, err := New("sage-vectors test", suites, core.WithSpecVersion(spec))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func generate(t *testing.T, r *Runner, prev fstest.MapFS) fstest.MapFS {
	t.Helper()
	var files map[string][]byte
	var err error
	if prev == nil {
		files, err = r.Generate(nil)
	} else {
		files, err = r.Generate(prev)
	}
	if err != nil {
		t.Fatal(err)
	}
	out := fstest.MapFS{}
	for name, data := range files {
		out[name] = &fstest.MapFile{Data: data}
	}
	return out
}

func TestGenerateThenCheck(t *testing.T) {
	r := newRunner(t, toySuite(rejectEmpty))
	fsys := generate(t, r, nil)
	if err := r.Check(fsys); err != nil {
		t.Fatal(err)
	}
	data := string(fsys["toy.json"].Data)
	for _, want := range []string{`"spec_version": "` + spec + `"`, `"mode": "reject"`, `"code": "malformed"`, `"reject": true`} {
		if !strings.Contains(data, want) {
			t.Errorf("generated file lacks %s", want)
		}
	}
	again := generate(t, r, fsys)
	if !bytes.Equal(again["toy.json"].Data, fsys["toy.json"].Data) {
		t.Error("regeneration changed a verify-mode output that still verifies")
	}
}

func TestRejectVectorAcceptedFailsTheRun(t *testing.T) {
	fsys := generate(t, newRunner(t, toySuite(rejectEmpty)), nil)
	cases := map[string]func(map[string]any) error{
		"accepted":      func(map[string]any) error { return nil },
		"plain error":   func(map[string]any) error { return errors.New("boom") },
		"wrong reason":  func(map[string]any) error { return core.Reject(obs.ReasonDigest, obs.Detail{}) },
		"wrapped wrong": func(map[string]any) error { return errors.Join(core.Reject(obs.ReasonStale, obs.Detail{})) },
	}
	for name, impl := range cases {
		t.Run(name, func(t *testing.T) {
			r := newRunner(t, toySuite(impl))
			if err := r.Check(fsys); err == nil {
				t.Fatal("check passed")
			}
			if _, err := r.Generate(nil); err == nil {
				t.Fatal("generate wrote a reject vector the implementation does not reject")
			}
		})
	}
	wrapped := newRunner(t, toySuite(func(in map[string]any) error {
		return errors.Join(errors.New("context"), rejectEmpty(in))
	}))
	if err := wrapped.Check(fsys); err != nil {
		t.Fatalf("a wrapped rejection must pass: %v", err)
	}
}

func replace(t *testing.T, fsys fstest.MapFS, old, new string) fstest.MapFS {
	t.Helper()
	data := string(fsys["toy.json"].Data)
	if !strings.Contains(data, old) {
		t.Fatalf("file lacks %q", old)
	}
	return fstest.MapFS{"toy.json": &fstest.MapFile{Data: []byte(strings.Replace(data, old, new, 1))}}
}

func TestCheckRejectsTamperedFiles(t *testing.T) {
	r := newRunner(t, toySuite(rejectEmpty))
	good := generate(t, r, nil)
	digest := sha256.Sum256([]byte("abc"))
	hexDigest := hex.EncodeToString(digest[:])
	tampered := map[string]fstest.MapFS{
		"spec version":      replace(t, good, `"`+spec+`"`, `"0.9.0-draft.1"`),
		"suite field":       replace(t, good, `"suite": "toy"`, `"suite": "other"`),
		"output":            replace(t, good, hexDigest, strings.Repeat("0", 64)),
		"input":             replace(t, good, `"message": "abc"`, `"message": "abd"`),
		"mode":              replace(t, good, `"mode": "reject"`, `"mode": "verify"`),
		"reject code":       replace(t, good, `"code": "malformed"`, `"code": "digest"`),
		"reject false":      replace(t, good, `"reject": true`, `"reject": false`),
		"reject extra":      replace(t, good, `"reject": true`, `"reject": true, "stage": "S1"`),
		"unknown field":     replace(t, good, `"suite": "toy"`, `"suite": "toy", "extra": 1`),
		"duplicate member":  replace(t, good, `"suite": "toy"`, `"suite": "toy", "suite": "toy"`),
		"nested duplicate":  replace(t, good, `"message": "abc"`, `"message": "abc", "message": "abc"`),
		"unknown vector":    replace(t, good, `"name": "sum"`, `"name": "sum2"`),
		"duplicate vector":  replace(t, good, `"name": "nonce"`, `"name": "sum"`),
		"trailing data":     {"toy.json": &fstest.MapFile{Data: append(append([]byte(nil), good["toy.json"].Data...), []byte("{}")...)}},
		"missing file":      {},
		"too large":         {"toy.json": &fstest.MapFile{Data: bytes.Repeat([]byte(" "), MaxFileBytes+1)}},
		"too deep":          {"toy.json": &fstest.MapFile{Data: []byte(strings.Repeat("[", maxDepth+1) + strings.Repeat("]", maxDepth+1))}},
		"verify not verify": replace(t, good, `"nonce": "`, `"nonce": "zz`),
		"case variant top":  replace(t, good, `"suite": "toy"`, `"suite": "other", "Suite": "toy"`),
		"fold variant top":  replace(t, good, `"suite": "toy"`, `"suite": "other", "\u017fuite": "toy"`),
		"case variant mode": replace(t, good, `"mode": "reject"`, `"mode": "reject", "Mode": "reject"`),
		"case variant out":  replace(t, good, `"output": {`+"\n"+`        "sha256"`, `"OUTPUT": {"sha256": "x"}, "output": {`+"\n"+`        "sha256"`),
		"number as float":   replace(t, good, `"n": 1`, `"n": 1.0`),
		"number exponent":   replace(t, good, `"n": 1`, `"n": 1e0`),
		"verify type":       replace(t, good, `"nonce": "`+nonceOf(t, good)+`"`, `"nonce": 1`),
		"invalid utf8":      {"toy.json": &fstest.MapFile{Data: bytes.Replace(good["toy.json"].Data, []byte("runner"), []byte("runn\xffr"), 1)}},
		"null input":        replace(t, good, `"input": {`+"\n"+`        "message": "abc"`+"\n"+`      }`, `"input": null`),
		"not an object":     {"toy.json": &fstest.MapFile{Data: []byte(`[]`)}},
	}
	for name, fsys := range tampered {
		t.Run(name, func(t *testing.T) {
			if err := r.Check(fsys); err == nil {
				t.Fatal("check passed")
			}
		})
	}
}

func nonceOf(t *testing.T, fsys fstest.MapFS) string {
	t.Helper()
	f, err := decodeFile(fsys["toy.json"].Data)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range f.Vectors {
		if v.Name == "nonce" {
			return v.Output["nonce"].(string)
		}
	}
	t.Fatal("no nonce vector")
	return ""
}

func TestRejectNeedsExactlyOneRejection(t *testing.T) {
	fsys := generate(t, newRunner(t, toySuite(rejectEmpty)), nil)
	two := newRunner(t, toySuite(func(in map[string]any) error {
		return errors.Join(rejectEmpty(in), core.Reject(obs.ReasonDigest, obs.Detail{}))
	}))
	if err := two.Check(fsys); err == nil {
		t.Fatal("two rejections accepted")
	}
}

func TestCallerCannotChangeInputsAfterNew(t *testing.T) {
	s := toySuite(rejectEmpty)
	r := newRunner(t, s)
	fsys := generate(t, r, nil)
	s.Cases[0].Input["message"] = "zzz"
	r.Suites()[0].Cases[0].Input["message"] = "zzz"
	if err := r.Check(fsys); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateRefusesDamagedPrevious(t *testing.T) {
	r := newRunner(t, toySuite(rejectEmpty))
	good := generate(t, r, nil)
	bad := fstest.MapFS{"toy.json": &fstest.MapFile{Data: append(append([]byte(nil), good["toy.json"].Data...), 'x')}}
	if _, err := r.Generate(bad); err == nil {
		t.Fatal("damaged previous file ignored")
	}
	if _, err := r.Generate(fstest.MapFS{}); err != nil {
		t.Fatalf("absent previous file: %v", err)
	}
}

func TestGenerateRefusesUnreadableOutput(t *testing.T) {
	deep := any("x")
	for i := 0; i < maxDepth; i++ {
		deep = []any{deep}
	}
	s := Suite{Name: "deep", Cases: []Case{{Name: "a", Mode: ModeDeterministic, Input: map[string]any{"a": 1},
		Produce: func(map[string]any) (map[string]any, error) { return map[string]any{"v": deep}, nil }}}}
	if _, err := newRunner(t, s).Generate(nil); err == nil {
		t.Fatal("generated a file Check cannot read")
	}
}

func TestNumbersKeepTheirText(t *testing.T) {
	s := Suite{Name: "num", Cases: []Case{{Name: "a", Mode: ModeDeterministic, Input: map[string]any{"n": int64(9007199254740993)},
		Produce: func(in map[string]any) (map[string]any, error) { return map[string]any{"n": in["n"]}, nil }}}}
	r := newRunner(t, s)
	fsys := generate(t, r, nil)
	if err := r.Check(fsys); err != nil {
		t.Fatal(err)
	}
	data := string(fsys["num.json"].Data)
	if !strings.Contains(data, "9007199254740993") {
		t.Fatalf("integer lost precision: %s", data)
	}
	lossy := fstest.MapFS{"num.json": &fstest.MapFile{Data: []byte(strings.ReplaceAll(data, "9007199254740993", "9007199254740992.0"))}}
	if err := r.Check(lossy); err == nil {
		t.Fatal("a different number passed")
	}
}

func TestCaseCannotChangeItsFixedInput(t *testing.T) {
	s := toySuite(rejectEmpty)
	s.Cases[0].Produce = func(in map[string]any) (map[string]any, error) {
		out, err := sum(in)
		in["message"] = "changed"
		return out, err
	}
	r := newRunner(t, s)
	fsys := generate(t, r, nil)
	if err := r.Check(fsys); err != nil {
		t.Fatal(err)
	}
	if err := r.Check(fsys); err != nil {
		t.Fatalf("second check: %v", err)
	}
}

func TestNewValidates(t *testing.T) {
	if _, err := New("g", nil); !errors.Is(err, core.ErrSpecVersionRequired) {
		t.Fatalf("no spec version: %v", err)
	}
	if _, err := New("g", nil, core.WithSpecVersion("0.9.0-draft.1")); !errors.Is(err, core.ErrSpecVersionUnsupported) {
		t.Fatalf("draft.1: %v", err)
	}
	if _, err := New("", nil, core.WithSpecVersion(spec)); err == nil {
		t.Fatal("empty generator accepted")
	}
	in := map[string]any{"a": 1}
	bad := map[string]Suite{
		"suite name":      {Name: "../x"},
		"case name":       {Name: "s", Cases: []Case{{Name: "A", Mode: ModeDeterministic, Input: in, Produce: sum}}},
		"no input":        {Name: "s", Cases: []Case{{Name: "a", Mode: ModeDeterministic, Produce: sum}}},
		"input not json":  {Name: "s", Cases: []Case{{Name: "a", Mode: ModeDeterministic, Input: map[string]any{"f": func() {}}, Produce: sum}}},
		"no produce":      {Name: "s", Cases: []Case{{Name: "a", Mode: ModeDeterministic, Input: in}}},
		"verify no check": {Name: "s", Cases: []Case{{Name: "a", Mode: ModeVerify, Input: in, Produce: sum}}},
		"reject reason":   {Name: "s", Cases: []Case{{Name: "a", Mode: ModeReject, Input: in, Reject: rejectEmpty}}},
		"reject produce":  {Name: "s", Cases: []Case{{Name: "a", Mode: ModeReject, Input: in, Reject: rejectEmpty, Reason: obs.ReasonMalformed, Produce: sum}}},
		"unknown mode":    {Name: "s", Cases: []Case{{Name: "a", Mode: "other", Input: in, Produce: sum}}},
		"duplicate case": {Name: "s", Cases: []Case{
			{Name: "a", Mode: ModeDeterministic, Input: in, Produce: sum},
			{Name: "a", Mode: ModeDeterministic, Input: in, Produce: sum},
		}},
	}
	for name, s := range bad {
		if _, err := New("g", []Suite{s}, core.WithSpecVersion(spec)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := New("g", []Suite{{Name: "s"}, {Name: "s"}}, core.WithSpecVersion(spec)); err == nil {
		t.Error("duplicate suite accepted")
	}
	var zero Runner
	if err := zero.Check(fstest.MapFS{}); !errors.Is(err, core.ErrSpecVersionRequired) {
		t.Errorf("zero runner check: %v", err)
	}
}

// publishedHashes pins testdata/v1 to sage-spec a69e901 v1/vectors.
var publishedHashes = map[string]string{
	"crypto.json":  "2aaccd7b46138d22974e9e8c53552d9c2c66a23f3f1bcff248eec095d43fa341",
	"did.json":     "3515c47f53ffa4fe9e6977fff752b37d181aabe559ea9362497dc731d3c8ced7",
	"hpke.json":    "cb2903e10ac4143f1b1e8d5762e61e6767ab79f7351b7284c79932256244e771",
	"jcs.json":     "ba33d3b77d747963f0a83bac67fd61f7828b909fd229908fe61682ba20451681",
	"rfc9421.json": "44b5625ca69c0c77b5fd133004d7d1d379d1f347f1f154ff2bc2d6d7f37f6568",
	"session.json": "1f16e4e37f3ac347d500c448c6379e7549a13cecce4210b905090cc0b69958d4",
}

// TestPublishedFiles reads the published files with the strict reader and
// checks that the encoder reproduces them byte for byte, so suites moved
// into the core regenerate the same files.
func TestPublishedFiles(t *testing.T) {
	fsys := os.DirFS("testdata/v1")
	total := 0
	for name, want := range publishedHashes {
		data, err := os.ReadFile("testdata/v1/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if got := sha256.Sum256(data); hex.EncodeToString(got[:]) != want {
			t.Errorf("%s: hash differs from the pinned sage-spec file", name)
		}
		f, err := readFile(fsys, name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if f.SpecVersion != spec {
			t.Errorf("%s: spec_version %q", name, f.SpecVersion)
		}
		out, err := encodeFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(out, data) {
			t.Errorf("%s: re-encoding changes bytes", name)
		}
		total += len(f.Vectors)
	}
	if total != 26 {
		t.Errorf("published files hold %d vectors, want 26", total)
	}
}

// FuzzCheck feeds arbitrary bytes as a suite file. Check must report a
// failure or pass, and never panic, even with cases that assert types.
func FuzzCheck(f *testing.F) {
	r, err := New("g", []Suite{toySuite(rejectEmpty)}, core.WithSpecVersion(spec))
	if err != nil {
		f.Fatal(err)
	}
	files, err := r.Generate(nil)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(files["toy.json"])
	f.Add([]byte(`{"suite":"toy","spec_version":"` + spec + `","generator":"g","vectors":[{"name":"nonce","mode":"verify","input":{"n":1},"output":{"nonce":1}}]}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		_ = r.Check(fstest.MapFS{"toy.json": &fstest.MapFile{Data: data}})
	})
}

func FuzzReadFile(f *testing.F) {
	f.Add([]byte(`{"suite":"toy","spec_version":"x","generator":"g","vectors":[]}`))
	f.Add([]byte(`{"a":{"a":1,"a":2}}`))
	f.Add([]byte(strings.Repeat("[", 100)))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = readFile(fstest.MapFS{"f.json": &fstest.MapFile{Data: data}}, "f.json")
	})
}
