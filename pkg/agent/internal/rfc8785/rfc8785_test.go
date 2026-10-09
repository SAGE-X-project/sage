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
package rfc8785

import "testing"

// The implementation is lenient by design; 0.10.0 callers check duplicates,
// limits and syntax bounds before calling it.
func TestCanonicalizeIsLenient(t *testing.T) {
	for in, want := range map[string]string{
		`{"b":1,"a":[2,{"d":true,"c":null}]}`: `{"a":[2,{"c":null,"d":true}],"b":1}`,
		`{"a":1,"a":2}`:                       `{"a":2}`,
		`-0`:                                  `0`,
		` [1.50 , "x"] `:                      `[1.5,"x"]`,
	} {
		got, err := Canonicalize([]byte(in))
		if err != nil || string(got) != want {
			t.Fatalf("%s: got %s, %v; want %s", in, got, err, want)
		}
	}
	for _, in := range []string{``, `{"a":}`, "\xff", `{} {}`} {
		if _, err := Canonicalize([]byte(in)); err == nil {
			t.Fatalf("accepted %q", in)
		}
	}
}

func TestMarshalCanonicalizesEncodedValue(t *testing.T) {
	got, err := Marshal(map[string]any{"z": 1, "a": "é"})
	if err != nil || string(got) != `{"a":"é","z":1}` {
		t.Fatalf("got %s, %v", got, err)
	}
}
