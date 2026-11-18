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

// Package vectors runs the normative test vectors of the SAGE specification
// (sage-spec v1/vectors) against suites that core packages define.
//
// A vector file is one JSON object per suite. Each vector has a mode:
// deterministic vectors are recomputed and compared, verify vectors are
// checked by the verification path, and reject vectors must be rejected
// (spec 0.9.0-draft.1.1, vectors/README.md). A reject vector passes only when
// the case returns a *core.Rejection whose reason, and therefore stage, is
// the one the case names.
//
// The runner reads and writes no files: Check reads from an fs.FS the host
// supplies, and Generate returns the file contents for the host to write.
package vectors

import (
	"errors"
	"fmt"
	"io/fs"
	"reflect"
	"sort"
	"strings"

	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/core/obs"
)

// Mode says how a vector is checked.
type Mode string

// Vector modes.
const (
	// ModeDeterministic vectors are recomputed from the input and compared.
	ModeDeterministic Mode = "deterministic"
	// ModeVerify vectors are checked by the verification path; their output
	// was drawn with randomness and differs between generators.
	ModeVerify Mode = "verify"
	// ModeReject vectors hold an input that must be rejected.
	ModeReject Mode = "reject"
)

// Vector is one test case as stored on disk.
type Vector struct {
	Name        string         `json:"name"`
	Mode        Mode           `json:"mode"`
	Description string         `json:"description,omitempty"`
	Input       map[string]any `json:"input"`
	Output      map[string]any `json:"output"`
}

// File is the on-disk form of a suite.
type File struct {
	Suite       string   `json:"suite"`
	SpecVersion string   `json:"spec_version"`
	Generator   string   `json:"generator"`
	Description string   `json:"description,omitempty"`
	Vectors     []Vector `json:"vectors"`
}

// Case defines one vector: a fixed input and the functions that check it.
//
//   - ModeDeterministic: Produce is required and must be deterministic;
//     Verify, when set, also runs on the stored output.
//   - ModeVerify: Produce and Verify are required.
//   - ModeReject: Reject and Reason are required. Reject must return a
//     *core.Rejection with Reason.
type Case struct {
	Name        string
	Mode        Mode
	Description string
	Input       map[string]any
	Produce     func(in map[string]any) (map[string]any, error)
	Verify      func(in, out map[string]any) error
	Reject      func(in map[string]any) error
	Reason      obs.Reason
}

// Suite groups cases under one file name, <Name>.json.
type Suite struct {
	Name        string
	Description string
	Cases       []Case
}

// Runner checks and generates vector files for a fixed set of suites and one
// spec version.
type Runner struct {
	settings  core.Settings
	generator string
	suites    []Suite
}

// New validates the suites and returns a runner. It requires
// core.WithSpecVersion; files whose spec_version differs are rejected.
// generator is written into generated files, for example "sage-vectors 1.6.0".
func New(generator string, suites []Suite, opts ...core.Option) (*Runner, error) {
	s, err := core.NewSettings(opts...)
	if err != nil {
		return nil, err
	}
	if generator == "" {
		return nil, errors.New("vectors: generator is required")
	}
	if err := validateSuites(suites); err != nil {
		return nil, err
	}
	return &Runner{settings: s, generator: generator, suites: copySuites(suites)}, nil
}

// Suites returns the suites the runner checks, in order.
func (r *Runner) Suites() []Suite { return copySuites(r.suites) }

// Count returns the number of cases in all suites.
func (r *Runner) Count() int {
	n := 0
	for _, s := range r.suites {
		n += len(s.Cases)
	}
	return n
}

// FileName returns the file name of a suite.
func FileName(suite string) string { return suite + ".json" }

// Check reads <suite>.json for every suite from fsys and checks every
// vector. It reports every failure, not only the first. A stored vector that
// no case defines is a failure, so a file cannot carry vectors this
// implementation does not run.
func (r *Runner) Check(fsys fs.FS) error {
	if err := r.settings.Validate(); err != nil {
		return err
	}
	var failures []string
	for _, s := range r.suites {
		failures = append(failures, r.checkSuite(fsys, s)...)
	}
	return joinFailures(failures)
}

func (r *Runner) checkSuite(fsys fs.FS, s Suite) []string {
	f, err := readFile(fsys, FileName(s.Name))
	if err != nil {
		return []string{fmt.Sprintf("%s: %v", s.Name, err)}
	}
	var failures []string
	if f.Suite != s.Name {
		failures = append(failures, fmt.Sprintf("%s: suite field is %q", s.Name, f.Suite))
	}
	if f.SpecVersion != string(r.settings.Spec()) {
		failures = append(failures, fmt.Sprintf("%s: spec_version is %q, this runner implements %q",
			s.Name, f.SpecVersion, r.settings.Spec()))
	}
	stored := map[string]Vector{}
	for _, v := range f.Vectors {
		if _, dup := stored[v.Name]; dup {
			failures = append(failures, fmt.Sprintf("%s/%s: duplicate vector name", s.Name, v.Name))
			continue
		}
		stored[v.Name] = v
	}
	defined := map[string]bool{}
	for _, c := range s.Cases {
		defined[c.Name] = true
		v, ok := stored[c.Name]
		if !ok {
			failures = append(failures, fmt.Sprintf("%s/%s: missing", s.Name, c.Name))
			continue
		}
		if err := checkCase(c, v); err != nil {
			failures = append(failures, fmt.Sprintf("%s/%s: %v", s.Name, c.Name, err))
		}
	}
	for name := range stored {
		if !defined[name] {
			failures = append(failures, fmt.Sprintf("%s/%s: unknown vector (no case defines it)", s.Name, name))
		}
	}
	return failures
}

// checkCase checks one stored vector. A panic in the case, for example on a
// hostile output, is reported as a failure of that vector.
func checkCase(c Case, v Vector) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("case panicked: %v", p)
		}
	}()
	if v.Mode != c.Mode {
		return fmt.Errorf("mode is %q, expected %q", v.Mode, c.Mode)
	}
	if !equalJSON(v.Input, c.Input) {
		return errors.New("input differs from the case's fixed input")
	}
	switch c.Mode {
	case ModeDeterministic:
		out, err := produce(c)
		if err != nil {
			return err
		}
		if !equalJSON(out, v.Output) {
			return fmt.Errorf("output mismatch:\n    stored:   %s\n    computed: %s", compact(v.Output), compact(out))
		}
		if c.Verify != nil {
			return verify(c, v.Output)
		}
		return nil
	case ModeVerify:
		return verify(c, v.Output)
	case ModeReject:
		if err := checkRejectOutput(v.Output, c.Reason); err != nil {
			return err
		}
		return runReject(c)
	default:
		return fmt.Errorf("unknown mode %q", c.Mode)
	}
}

func produce(c Case) (map[string]any, error) {
	in, err := clone(c.Input)
	if err != nil {
		return nil, err
	}
	return c.Produce(in)
}

func verify(c Case, out map[string]any) error {
	in, err := clone(c.Input)
	if err != nil {
		return err
	}
	o, err := clone(out)
	if err != nil {
		return err
	}
	return c.Verify(in, o)
}

// checkRejectOutput requires the output to be exactly
// {"reject": true, "code": <reason name>}.
func checkRejectOutput(out map[string]any, reason obs.Reason) error {
	if len(out) != 2 {
		return fmt.Errorf("reject output must hold exactly reject and code, got %s", compact(out))
	}
	if b, ok := out["reject"].(bool); !ok || !b {
		return errors.New(`reject output must hold "reject": true`)
	}
	code, ok := out["code"].(string)
	if !ok {
		return errors.New(`reject output must hold a string "code"`)
	}
	if code != reason.String() {
		return fmt.Errorf("code is %q, the case rejects with %q", code, reason)
	}
	return nil
}

// runReject runs a reject case. It passes only when the error tree holds
// exactly one *core.Rejection and its reason is the expected one.
func runReject(c Case) error {
	in, err := clone(c.Input)
	if err != nil {
		return err
	}
	err = c.Reject(in)
	if err == nil {
		return errors.New("accepted an input that must be rejected")
	}
	rejs := rejections(err, nil)
	switch len(rejs) {
	case 0:
		return fmt.Errorf("failed without a core.Rejection: %v", err)
	case 1:
	default:
		return fmt.Errorf("error holds %d rejections, want one", len(rejs))
	}
	if rejs[0].Reason() != c.Reason {
		return fmt.Errorf("rejected with %s at %s, expected %s at %s",
			rejs[0].Reason(), rejs[0].Stage(), c.Reason, obs.StageOf(c.Reason))
	}
	return nil
}

// rejections collects every *core.Rejection in the error tree of err.
func rejections(err error, acc []*core.Rejection) []*core.Rejection {
	if err == nil || len(acc) > 16 {
		return acc
	}
	if r, ok := err.(*core.Rejection); ok {
		return append(acc, r)
	}
	switch u := err.(type) {
	case interface{ Unwrap() []error }:
		for _, e := range u.Unwrap() {
			acc = rejections(e, acc)
		}
	case interface{ Unwrap() error }:
		acc = rejections(u.Unwrap(), acc)
	}
	return acc
}

func rejectOutput(reason obs.Reason) map[string]any {
	return map[string]any{"reject": true, "code": reason.String()}
}

// Generate returns the contents of every suite file, keyed by file name.
// Verify-mode outputs present in prev that still verify are kept, so a
// regeneration changes no bytes unless the protocol changed. prev may be
// nil. A reject case that its implementation accepts fails generation.
func (r *Runner) Generate(prev fs.FS) (map[string][]byte, error) {
	if err := r.settings.Validate(); err != nil {
		return nil, err
	}
	files := make(map[string][]byte, len(r.suites))
	for _, s := range r.suites {
		f, err := r.generateSuite(prev, s)
		if err != nil {
			return nil, err
		}
		data, err := encodeFile(f)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", s.Name, err)
		}
		// Never write a file that Check would refuse to read.
		if _, err := decodeFile(data); err != nil {
			return nil, fmt.Errorf("%s: generated file is not readable: %w", s.Name, err)
		}
		files[FileName(s.Name)] = data
	}
	return files, nil
}

func (r *Runner) generateSuite(prev fs.FS, s Suite) (*File, error) {
	previous := map[string]Vector{}
	if prev != nil {
		old, err := readFile(prev, FileName(s.Name))
		switch {
		case err == nil:
			for _, v := range old.Vectors {
				previous[v.Name] = v
			}
		case !errors.Is(err, fs.ErrNotExist):
			// A damaged previous file would silently redraw every
			// verify-mode output and change published bytes.
			return nil, fmt.Errorf("previous file: %w", err)
		}
	}
	f := &File{
		Suite:       s.Name,
		SpecVersion: string(r.settings.Spec()),
		Generator:   r.generator,
		Description: s.Description,
		Vectors:     make([]Vector, 0, len(s.Cases)),
	}
	for _, c := range s.Cases {
		v, err := generateCase(c, previous[c.Name])
		if err != nil {
			return nil, fmt.Errorf("%s/%s: %w", s.Name, c.Name, err)
		}
		f.Vectors = append(f.Vectors, v)
	}
	return f, nil
}

func generateCase(c Case, old Vector) (v Vector, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("case panicked: %v", p)
		}
	}()
	in, err := clone(c.Input)
	if err != nil {
		return Vector{}, err
	}
	v = Vector{Name: c.Name, Mode: c.Mode, Description: c.Description, Input: in}
	switch c.Mode {
	case ModeReject:
		if err := runReject(c); err != nil {
			return Vector{}, err
		}
		v.Output = rejectOutput(c.Reason)
		return v, nil
	case ModeVerify:
		if old.Name == c.Name && old.Mode == c.Mode && equalJSON(old.Input, c.Input) && verify(c, old.Output) == nil {
			v.Output = old.Output
			return v, nil
		}
	}
	out, err := produce(c)
	if err != nil {
		return Vector{}, err
	}
	if c.Verify != nil {
		if err := verify(c, out); err != nil {
			return Vector{}, fmt.Errorf("produced output does not verify: %w", err)
		}
	}
	v.Output = out
	return v, nil
}

func validateSuites(suites []Suite) error {
	names := map[string]bool{}
	for _, s := range suites {
		if !validName(s.Name) {
			return fmt.Errorf("vectors: suite name %q must match [a-z0-9-]{1,64}", s.Name)
		}
		if names[s.Name] {
			return fmt.Errorf("vectors: duplicate suite %q", s.Name)
		}
		names[s.Name] = true
		if err := validateCases(s); err != nil {
			return err
		}
	}
	return nil
}

func validateCases(s Suite) error {
	names := map[string]bool{}
	for _, c := range s.Cases {
		where := s.Name + "/" + c.Name
		if !validName(c.Name) {
			return fmt.Errorf("vectors: case name %q must match [a-z0-9-]{1,64}", where)
		}
		if names[c.Name] {
			return fmt.Errorf("vectors: duplicate case %s", where)
		}
		names[c.Name] = true
		if c.Input == nil {
			return fmt.Errorf("vectors: %s has no input", where)
		}
		if _, err := normalise(c.Input); err != nil {
			return fmt.Errorf("vectors: %s: input is not JSON: %w", where, err)
		}
		switch c.Mode {
		case ModeDeterministic:
			if c.Produce == nil || c.Reject != nil {
				return fmt.Errorf("vectors: %s: deterministic needs Produce and no Reject", where)
			}
		case ModeVerify:
			if c.Produce == nil || c.Verify == nil || c.Reject != nil {
				return fmt.Errorf("vectors: %s: verify needs Produce and Verify and no Reject", where)
			}
		case ModeReject:
			if c.Reject == nil || c.Produce != nil || c.Verify != nil {
				return fmt.Errorf("vectors: %s: reject needs Reject only", where)
			}
			if obs.StageOf(c.Reason) == obs.StageUnknown {
				return fmt.Errorf("vectors: %s: reject needs a defined Reason", where)
			}
		default:
			return fmt.Errorf("vectors: %s: unknown mode %q", where, c.Mode)
		}
	}
	return nil
}

// validName accepts 1 to 64 bytes of a-z, 0-9 and '-', so a suite name is a
// safe file name and a case name is a stable identifier.
func validName(s string) bool {
	if len(s) == 0 || len(s) > 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
			return false
		}
	}
	return true
}

// copySuites copies suites and deep-copies every case input, so a caller
// cannot change a fixed input after New. Inputs were checked to be JSON by
// validateCases.
func copySuites(in []Suite) []Suite {
	out := make([]Suite, len(in))
	for i, s := range in {
		cases := make([]Case, len(s.Cases))
		for j, c := range s.Cases {
			c.Input, _ = clone(c.Input)
			cases[j] = c
		}
		out[i] = Suite{Name: s.Name, Description: s.Description, Cases: cases}
	}
	return out
}

func equalJSON(a, b map[string]any) bool {
	na, errA := normalise(a)
	nb, errB := normalise(b)
	return errA == nil && errB == nil && reflect.DeepEqual(na, nb)
}

func joinFailures(failures []string) error {
	if len(failures) == 0 {
		return nil
	}
	sort.Strings(failures)
	return errors.New("vector check failed:\n  " + strings.Join(failures, "\n  "))
}
