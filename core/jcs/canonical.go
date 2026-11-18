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

package jcs

import (
	"cmp"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Canonical returns the RFC 8785 bytes of v: members sorted by their UTF-16
// code units, strings escaped as ECMAScript JSON.stringify does, numbers in
// ECMAScript Number::toString form, no whitespace. It returns
// ErrInvalidValue only for a Value that Parse did not make.
func Canonical(v Value) ([]byte, error) {
	b, ok := appendValue(nil, v)
	if !ok {
		return nil, ErrInvalidValue
	}
	return b, nil
}

func appendValue(b []byte, v Value) ([]byte, bool) {
	switch v.kind {
	case KindNull:
		return append(b, "null"...), true
	case KindBool:
		return strconv.AppendBool(b, v.boolean), true
	case KindNumber:
		if math.IsNaN(v.number) || math.IsInf(v.number, 0) {
			return nil, false
		}
		return append(b, formatNumber(v.number)...), true
	case KindString:
		return appendString(b, v.str), true
	case KindArray:
		b = append(b, '[')
		for i, item := range v.items() {
			if i > 0 {
				b = append(b, ',')
			}
			var ok bool
			if b, ok = appendValue(b, item); !ok {
				return nil, false
			}
		}
		return append(b, ']'), true
	case KindObject:
		return appendObject(b, v.names(), v.items())
	default:
		return nil, false
	}
}

func appendObject(b []byte, names []string, items []Value) ([]byte, bool) {
	order := make([]int, len(names))
	for i := range order {
		order[i] = i
	}
	slices.SortFunc(order, func(x, y int) int { return compareUTF16(names[x], names[y]) })
	b = append(b, '{')
	for i, k := range order {
		if i > 0 {
			b = append(b, ',')
		}
		b = appendString(b, names[k])
		b = append(b, ':')
		var ok bool
		if b, ok = appendValue(b, items[k]); !ok {
			return nil, false
		}
	}
	return append(b, '}'), true
}

// compareUTF16 orders two valid UTF-8 strings by their UTF-16 code units
// (RFC 8785 §3.2.3) without converting them. Code point order and UTF-16
// order differ only where a supplementary character (first unit a high
// surrogate, 0xD800-0xDBFF) meets a character in U+E000-U+FFFF.
func compareUTF16(a, b string) int {
	for a != "" && b != "" {
		ra, na := utf8.DecodeRuneInString(a)
		rb, nb := utf8.DecodeRuneInString(b)
		if ra != rb {
			if ua, ub := firstUnit(ra), firstUnit(rb); ua != ub {
				return cmp.Compare(ua, ub)
			}
			// Same high surrogate: both are supplementary, and their low
			// surrogates are in code point order.
			return cmp.Compare(ra, rb)
		}
		a, b = a[na:], b[nb:]
	}
	return cmp.Compare(len(a), len(b))
}

func firstUnit(r rune) rune {
	if r < 0x10000 {
		return r
	}
	return 0xd800 + (r-0x10000)>>10
}

const hexDigits = "0123456789abcdef"

// appendString escapes s as ECMAScript JSON.stringify does. s is valid UTF-8
// without surrogates, because Parse rejects anything else.
func appendString(b []byte, s string) []byte {
	b = append(b, '"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"':
			b = append(b, '\\', '"')
		case '\\':
			b = append(b, '\\', '\\')
		case '\b':
			b = append(b, '\\', 'b')
		case '\f':
			b = append(b, '\\', 'f')
		case '\n':
			b = append(b, '\\', 'n')
		case '\r':
			b = append(b, '\\', 'r')
		case '\t':
			b = append(b, '\\', 't')
		default:
			if c < 0x20 {
				b = append(b, '\\', 'u', '0', '0', hexDigits[c>>4], hexDigits[c&0xf])
			} else {
				b = append(b, c)
			}
		}
	}
	return append(b, '"')
}

// formatNumber returns the ECMAScript Number::toString form of a finite f:
// the shortest digits that round-trip, positional notation for
// 1e-6 <= |f| < 1e21 and exponent notation without a leading exponent zero
// otherwise; negative zero is "0".
func formatNumber(f float64) string {
	if f == 0 {
		return "0"
	}
	sign := ""
	if f < 0 {
		sign, f = "-", -f
	}
	format := byte('e')
	if f >= 1e-6 && f < 1e21 {
		format = 'f'
	}
	s := strconv.FormatFloat(f, format, -1, 64)
	if e := strings.IndexByte(s, 'e'); e > 0 && s[e+2] == '0' {
		s = s[:e+2] + s[e+3:]
	}
	return sign + s
}
