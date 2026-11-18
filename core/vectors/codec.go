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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"unicode/utf8"
)

// MaxFileBytes bounds one vector file.
const MaxFileBytes = 8 << 20

// maxDepth bounds the nesting of a vector file.
const maxDepth = 64

// readFile reads one vector file strictly: bounded size, valid UTF-8, no
// duplicate object members, exactly the member names of the format (compared
// byte for byte, so "Output" or "ſuite" is not taken for "output" or
// "suite"), no data after the object, and numbers kept as their literal text.
func readFile(fsys fs.FS, name string) (*File, error) {
	fh, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = fh.Close() }()
	data, err := io.ReadAll(io.LimitReader(fh, MaxFileBytes+1))
	if err != nil {
		return nil, err
	}
	f, err := decodeFile(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return f, nil
}

func decodeFile(data []byte) (*File, error) {
	if len(data) > MaxFileBytes {
		return nil, fmt.Errorf("larger than %d bytes", MaxFileBytes)
	}
	if !utf8.Valid(data) {
		return nil, errors.New("not valid UTF-8")
	}
	if err := checkStructure(data); err != nil {
		return nil, err
	}
	top, err := members(data, "suite", "spec_version", "generator", "description", "vectors")
	if err != nil {
		return nil, err
	}
	var f File
	if err := decodeMember(top, "suite", &f.Suite, true); err != nil {
		return nil, err
	}
	if err := decodeMember(top, "spec_version", &f.SpecVersion, true); err != nil {
		return nil, err
	}
	if err := decodeMember(top, "generator", &f.Generator, true); err != nil {
		return nil, err
	}
	if err := decodeMember(top, "description", &f.Description, false); err != nil {
		return nil, err
	}
	var raw []json.RawMessage
	if err := decodeMember(top, "vectors", &raw, true); err != nil {
		return nil, err
	}
	f.Vectors = make([]Vector, 0, len(raw))
	for i, r := range raw {
		v, err := decodeVector(r)
		if err != nil {
			return nil, fmt.Errorf("vectors[%d]: %w", i, err)
		}
		f.Vectors = append(f.Vectors, v)
	}
	return &f, nil
}

func decodeVector(data []byte) (Vector, error) {
	m, err := members(data, "name", "mode", "description", "input", "output")
	if err != nil {
		return Vector{}, err
	}
	var v Vector
	if err := decodeMember(m, "name", &v.Name, true); err != nil {
		return Vector{}, err
	}
	var mode string
	if err := decodeMember(m, "mode", &mode, true); err != nil {
		return Vector{}, err
	}
	v.Mode = Mode(mode)
	if err := decodeMember(m, "description", &v.Description, false); err != nil {
		return Vector{}, err
	}
	if err := decodeMember(m, "input", &v.Input, true); err != nil {
		return Vector{}, err
	}
	if err := decodeMember(m, "output", &v.Output, true); err != nil {
		return Vector{}, err
	}
	return v, nil
}

// members decodes a JSON object into its raw members and rejects any name
// outside allowed. Names are compared exactly.
func members(data []byte, allowed ...string) (map[string]json.RawMessage, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m == nil {
		return nil, errors.New("not an object")
	}
	for name := range m {
		known := false
		for _, a := range allowed {
			if name == a {
				known = true
				break
			}
		}
		if !known {
			return nil, fmt.Errorf("unknown member %q", name)
		}
	}
	return m, nil
}

// decodeMember decodes one member into dst. Numbers inside maps stay
// json.Number. A required member that is absent or null is an error.
func decodeMember(m map[string]json.RawMessage, name string, dst any, required bool) error {
	raw, ok := m[name]
	if !ok || string(raw) == "null" {
		if required {
			return fmt.Errorf("member %q is required", name)
		}
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("member %q: %w", name, err)
	}
	return nil
}

// checkStructure walks the JSON tokens of data and rejects duplicate object
// members, nesting deeper than maxDepth and anything after the first value.
// encoding/json keeps the last of duplicate members silently, which would
// let one file carry two different outputs for one vector.
func checkStructure(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	if err := walkValue(dec, 0); err != nil {
		return err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return errors.New("data after the top-level value")
	}
	return nil
}

func walkValue(dec *json.Decoder, depth int) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil
	}
	if depth >= maxDepth {
		return fmt.Errorf("nesting deeper than %d", maxDepth)
	}
	switch delim {
	case '{':
		return walkObject(dec, depth+1)
	case '[':
		for dec.More() {
			if err := walkValue(dec, depth+1); err != nil {
				return err
			}
		}
		_, err := dec.Token() // ']'
		return err
	default:
		return fmt.Errorf("unexpected %q", delim)
	}
}

func walkObject(dec *json.Decoder, depth int) error {
	seen := map[string]bool{}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		key, ok := tok.(string)
		if !ok {
			return errors.New("object member name is not a string")
		}
		if seen[key] {
			return fmt.Errorf("duplicate member %q", key)
		}
		seen[key] = true
		if err := walkValue(dec, depth); err != nil {
			return err
		}
	}
	_, err := dec.Token() // '}'
	return err
}

// encodeFile writes a file the way the reference generator always has: two
// space indentation, sorted map keys and a final newline.
func encodeFile(f *File) ([]byte, error) {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// normalise round-trips a value through JSON so that maps built in Go, with
// typed values, compare equal to maps decoded from a file. Numbers become
// json.Number and compare by their literal text, so 1, 1.0 and 1e0 differ
// and no integer loses precision.
func normalise(v any) (any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var out any
	if err := dec.Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

// clone returns a deep copy of m, so a case cannot change the fixed input
// or the stored output it is given. Numbers in the copy are json.Number.
func clone(m map[string]any) (map[string]any, error) {
	if m == nil {
		return nil, nil
	}
	n, err := normalise(m)
	if err != nil {
		return nil, err
	}
	out, ok := n.(map[string]any)
	if !ok {
		return nil, errors.New("not an object")
	}
	return out, nil
}

func compact(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "<unencodable>"
	}
	return string(data)
}
