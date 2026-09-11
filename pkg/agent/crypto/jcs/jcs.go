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

// Package jcs implements the JSON Canonicalization Scheme (RFC 8785).
//
// Every signed JSON structure in SAGE (A2A card proofs, the HPKE handshake
// response) is canonicalized with this package before hashing or signing, so
// that any implementation, in any language, produces and verifies exactly the
// same bytes from the same JSON value. The rules are those of RFC 8785:
// object members sorted by the UTF-16 code units of their names, no
// insignificant whitespace, strings escaped as ECMAScript JSON.stringify does
// (only `"`, `\` and control characters below U+0020), and numbers formatted
// as ECMAScript Number::toString.
package jcs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// Marshal encodes v with encoding/json and returns its canonical form.
func Marshal(v interface{}) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return Canonicalize(raw)
}

// Canonicalize returns the RFC 8785 canonical form of a JSON document.
func Canonicalize(raw []byte) ([]byte, error) {
	if !utf8.Valid(raw) {
		return nil, errors.New("jcs: input is not valid UTF-8")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v interface{}
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("jcs: invalid JSON: %w", err)
	}
	if dec.More() {
		return nil, errors.New("jcs: trailing data after JSON document")
	}
	var buf bytes.Buffer
	if err := writeValue(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeValue(buf *bytes.Buffer, v interface{}) error {
	switch x := v.(type) {
	case nil:
		buf.WriteString("null")
	case bool:
		if x {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case json.Number:
		f, err := strconv.ParseFloat(string(x), 64)
		if err != nil {
			return fmt.Errorf("jcs: number %q is not an IEEE 754 double: %w", x, err)
		}
		return writeNumber(buf, f)
	case float64:
		return writeNumber(buf, x)
	case string:
		return writeString(buf, x)
	case []interface{}:
		buf.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeValue(buf, e); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	case map[string]interface{}:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return lessUTF16(keys[i], keys[j]) })
		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeString(buf, k); err != nil {
				return err
			}
			buf.WriteByte(':')
			if err := writeValue(buf, x[k]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	default:
		return fmt.Errorf("jcs: unsupported value of type %T", v)
	}
	return nil
}

// lessUTF16 orders strings by their UTF-16 code units (RFC 8785 Section 3.2.3).
func lessUTF16(a, b string) bool {
	ua, ub := utf16.Encode([]rune(a)), utf16.Encode([]rune(b))
	for i := 0; i < len(ua) && i < len(ub); i++ {
		if ua[i] != ub[i] {
			return ua[i] < ub[i]
		}
	}
	return len(ua) < len(ub)
}

// writeString escapes s as ECMAScript JSON.stringify does (RFC 8785 Section 3.2.2.2).
func writeString(buf *bytes.Buffer, s string) error {
	if !utf8.ValidString(s) {
		return errors.New("jcs: string is not valid UTF-8")
	}
	buf.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\b':
			buf.WriteString(`\b`)
		case '\f':
			buf.WriteString(`\f`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(buf, `\u%04x`, r)
			} else {
				buf.WriteRune(r)
			}
		}
	}
	buf.WriteByte('"')
	return nil
}

// writeNumber formats f as ECMAScript Number::toString (RFC 8785 Section 3.2.2.3).
func writeNumber(buf *bytes.Buffer, f float64) error {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return errors.New("jcs: NaN and Infinity cannot be serialized")
	}
	if f == 0 {
		buf.WriteByte('0') // covers -0
		return nil
	}
	if f < 0 {
		buf.WriteByte('-')
		f = -f
	}
	// Shortest round-trip digits: d.ddddde±xx
	e := strconv.FormatFloat(f, 'e', -1, 64)
	mant, expStr, _ := strings.Cut(e, "e")
	exp, err := strconv.Atoi(expStr)
	if err != nil {
		return fmt.Errorf("jcs: cannot format number: %w", err)
	}
	digits := strings.Replace(mant, ".", "", 1)
	k := len(digits)
	n := exp + 1 // value = 0.digits × 10^n

	switch {
	case k <= n && n <= 21:
		buf.WriteString(digits)
		buf.WriteString(strings.Repeat("0", n-k))
	case 0 < n && n <= 21:
		buf.WriteString(digits[:n])
		buf.WriteByte('.')
		buf.WriteString(digits[n:])
	case -6 < n && n <= 0:
		buf.WriteString("0.")
		buf.WriteString(strings.Repeat("0", -n))
		buf.WriteString(digits)
	default:
		buf.WriteByte(digits[0])
		if k > 1 {
			buf.WriteByte('.')
			buf.WriteString(digits[1:])
		}
		buf.WriteByte('e')
		if n-1 >= 0 {
			buf.WriteByte('+')
		}
		buf.WriteString(strconv.Itoa(n - 1))
	}
	return nil
}
