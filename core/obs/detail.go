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

package obs

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// MaxDetailBytes bounds the escaped text of a Detail (00-overview.md §7.6).
const MaxDetailBytes = 256

const truncationMark = "..."

// Detail is a string that is safe to log: it is escaped so that control
// characters and non-ASCII text cannot forge or hide log lines, and it is at
// most MaxDetailBytes long. A Detail made from input that has not been
// verified is marked unverified.
type Detail struct {
	text       string
	unverified bool
}

// Unverified makes a Detail from a value taken from input that has not been
// verified yet, such as the keyid DID before the signature verifies.
func Unverified(s string) Detail {
	return Detail{text: escapeBounded(s), unverified: true}
}

// Known makes a Detail from a value the core produced or verified itself.
// It is escaped and bounded in the same way.
func Known(s string) Detail {
	return Detail{text: escapeBounded(s)}
}

// String returns the escaped, bounded text.
func (d Detail) String() string { return d.text }

// Unverified reports whether the value came from unverified input.
func (d Detail) Unverified() bool { return d.unverified }

// IsZero reports whether the Detail holds no text.
func (d Detail) IsZero() bool { return d.text == "" }

// escapeBounded escapes every byte outside printable ASCII, and the
// backslash. A result longer than MaxDetailBytes is cut after the last whole
// escape piece that leaves room for "...", and ends with "...".
func escapeBounded(s string) string {
	var b strings.Builder
	cut := -1
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		piece := escapeRune(s[i], r, size)
		if cut < 0 && b.Len()+len(piece) > MaxDetailBytes-len(truncationMark) {
			cut = b.Len()
		}
		if b.Len()+len(piece) > MaxDetailBytes {
			return b.String()[:cut] + truncationMark
		}
		b.WriteString(piece)
		i += size
	}
	return b.String()
}

func escapeRune(first byte, r rune, size int) string {
	switch {
	case r == utf8.RuneError && size <= 1:
		return `\x` + hex2(first)
	case r == '\\':
		return `\\`
	case r >= 0x20 && r < 0x7f:
		return string(r)
	case r == '\n':
		return `\n`
	case r == '\r':
		return `\r`
	case r == '\t':
		return `\t`
	case r < 0x80:
		return `\x` + hex2(byte(r))
	case r <= 0xffff:
		return `\u` + pad(strconv.FormatInt(int64(r), 16), 4)
	default:
		return `\U` + pad(strconv.FormatInt(int64(r), 16), 8)
	}
}

func hex2(b byte) string { return pad(strconv.FormatInt(int64(b), 16), 2) }

func pad(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return strings.Repeat("0", n-len(s)) + s
}
