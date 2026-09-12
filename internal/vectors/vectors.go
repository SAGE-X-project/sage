// Package vectors generates and checks the SAGE protocol test vectors that
// are published in the sage-spec repository.
//
// Every vector belongs to a suite (crypto, jcs, rfc9421, hpke, session, did)
// and is either deterministic or verify-only:
//
//   - deterministic: the output is a pure function of the input; "check"
//     recomputes it with this implementation and compares byte for byte.
//   - verify: the Go code draws randomness (P-256 signatures, session nonces,
//     proof timestamps), so the stored output is kept as long as it still
//     verifies; "check" runs the verification path against it.
//
// Files are JSON, one per suite, with all binary values hex encoded.
package vectors

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"github.com/sage-x-project/sage/pkg/version"
)

// Mode says how a vector is checked.
type Mode string

const (
	// ModeDeterministic vectors are recomputed and compared.
	ModeDeterministic Mode = "deterministic"
	// ModeVerify vectors are verified, not recomputed.
	ModeVerify Mode = "verify"
)

// Vector is one test case as stored on disk.
type Vector struct {
	Name        string         `json:"name"`
	Mode        Mode           `json:"mode"`
	Description string         `json:"description,omitempty"`
	Input       map[string]any `json:"input"`
	Output      map[string]any `json:"output"`
}

// File is the on-disk representation of a suite.
type File struct {
	Suite       string   `json:"suite"`
	SpecVersion string   `json:"spec_version"`
	Generator   string   `json:"generator"`
	Description string   `json:"description,omitempty"`
	Vectors     []Vector `json:"vectors"`
}

// SpecVersion is the version of sage-spec these vectors are generated for.
const SpecVersion = "1.0.0-draft.1"

// Case defines one vector: fixed inputs, a producer and (for verify mode)
// a verifier. Produce must be deterministic for ModeDeterministic cases.
type Case struct {
	Name        string
	Mode        Mode
	Description string
	Input       map[string]any
	// Produce computes the output from Input.
	Produce func(in map[string]any) (map[string]any, error)
	// Verify checks a stored output against Input. Required for ModeVerify;
	// optional for deterministic cases (run in addition to the comparison).
	Verify func(in, out map[string]any) error
}

// Suite groups cases under one file name.
type Suite struct {
	Name        string
	Description string
	Cases       []Case
}

// Suites returns every suite in generation order.
func Suites() []Suite {
	return []Suite{
		cryptoSuite(),
		jcsSuite(),
		rfc9421Suite(),
		hpkeSuite(),
		sessionSuite(),
		didSuite(),
	}
}

func generatorID() string { return "sage-vectors " + version.Version }

func fileName(dir, suite string) string { return filepath.Join(dir, suite+".json") }

func readFile(path string) (*File, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- vector directory chosen by the operator
	if err != nil {
		return nil, err
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &f, nil
}

func writeFile(path string, f *File) error {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644) // #nosec G306 -- published vectors are public
}

// Generate writes every suite into dir. Verify-mode vectors already present
// in dir are kept when they still verify, so regenerating does not churn
// randomised outputs.
func Generate(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, s := range Suites() {
		path := fileName(dir, s.Name)
		previous := map[string]Vector{}
		if old, err := readFile(path); err == nil {
			for _, v := range old.Vectors {
				previous[v.Name] = v
			}
		}
		f := &File{Suite: s.Name, SpecVersion: SpecVersion, Generator: generatorID(), Description: s.Description}
		for _, c := range s.Cases {
			v := Vector{Name: c.Name, Mode: c.Mode, Description: c.Description, Input: c.Input}
			if c.Mode == ModeVerify {
				if old, ok := previous[c.Name]; ok && reflect.DeepEqual(normalise(old.Input), normalise(c.Input)) {
					if err := c.Verify(c.Input, old.Output); err == nil {
						v.Output = old.Output
						f.Vectors = append(f.Vectors, v)
						continue
					}
				}
			}
			out, err := c.Produce(c.Input)
			if err != nil {
				return fmt.Errorf("%s/%s: %w", s.Name, c.Name, err)
			}
			v.Output = out
			f.Vectors = append(f.Vectors, v)
		}
		if err := writeFile(path, f); err != nil {
			return err
		}
	}
	return nil
}

// Check reads every suite from dir and verifies each vector. It reports
// every failure, not just the first.
func Check(dir string) error {
	var failures []string
	for _, s := range Suites() {
		f, err := readFile(fileName(dir, s.Name))
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}
		if f.Suite != s.Name {
			failures = append(failures, fmt.Sprintf("%s: suite field is %q", s.Name, f.Suite))
		}
		stored := map[string]Vector{}
		for _, v := range f.Vectors {
			stored[v.Name] = v
		}
		for _, c := range s.Cases {
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
			if !hasCase(s, name) {
				failures = append(failures, fmt.Sprintf("%s/%s: unknown vector (not produced by this implementation)", s.Name, name))
			}
		}
	}
	if len(failures) > 0 {
		sort.Strings(failures)
		return errors.New("vector check failed:\n  " + joinLines(failures))
	}
	return nil
}

func checkCase(c Case, v Vector) error {
	if v.Mode != c.Mode {
		return fmt.Errorf("mode is %q, expected %q", v.Mode, c.Mode)
	}
	if !reflect.DeepEqual(normalise(v.Input), normalise(c.Input)) {
		return errors.New("input differs from this implementation's fixed input")
	}
	switch c.Mode {
	case ModeDeterministic:
		out, err := c.Produce(c.Input)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(normalise(out), normalise(v.Output)) {
			return fmt.Errorf("output mismatch:\n    stored:   %s\n    computed: %s", compact(v.Output), compact(out))
		}
		if c.Verify != nil {
			return c.Verify(c.Input, v.Output)
		}
		return nil
	case ModeVerify:
		if c.Verify == nil {
			return errors.New("verify-mode case without a verifier")
		}
		return c.Verify(c.Input, v.Output)
	default:
		return fmt.Errorf("unknown mode %q", c.Mode)
	}
}

func hasCase(s Suite, name string) bool {
	for _, c := range s.Cases {
		if c.Name == name {
			return true
		}
	}
	return false
}

// normalise round-trips a value through JSON so that maps built in Go
// (with typed values) compare equal to maps decoded from disk.
func normalise(v any) any {
	data, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		return v
	}
	return out
}

func compact(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(data)
}

func joinLines(lines []string) string {
	out := ""
	for i, l := range lines {
		if i > 0 {
			out += "\n  "
		}
		out += l
	}
	return out
}
