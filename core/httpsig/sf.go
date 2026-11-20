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

package httpsig

import (
	"encoding/base64"
	"strconv"
	"strings"

	"github.com/sage-x-project/sage/core/obs"
)

// This file is a strict RFC 8941 (Structured Field Values) reader and
// writer for the two dictionaries RFC 9421 uses. It reads by byte index and
// never builds strings rune by rune, so its cost is linear in the input
// (security review H2). RFC 8941 lets a repeated dictionary or parameter key
// overwrite the earlier one; this reader keeps every occurrence and marks
// repeated parameters, so that Parse can reject a repeat in the signature it
// verifies and ignore one in any other (E03-06). The 8 KiB field limit
// bounds the work.

type kind uint8

const (
	kindInteger kind = iota + 1
	kindDecimal
	kindString
	kindToken
	kindBytes
	kindBoolean
)

// bare is one RFC 8941 bare item.
type bare struct {
	kind    kind
	integer int64  // integer; decimal is stored as text
	text    string // string, token, decimal text, decoded byte sequence
	boolean bool
}

type param struct {
	key   string
	value bare
}

type item struct {
	value  bare
	params []param
	repeat bool // a parameter key occurs twice
}

// member is one dictionary member: an item or an inner list.
type member struct {
	key       string
	inner     bool
	item      item   // when !inner
	list      []item // when inner
	params    []param
	repeat    bool   // a parameter key occurs twice, on the list or an item
	raw       string // the received bytes of the member, key included
	canonical string // its RFC 8941 serialisation
}

type scanner struct {
	s   string
	pos int
}

func malformed(detail string) error { return reject(obs.ReasonMalformed, detail) }
func (sc *scanner) done() bool      { return sc.pos >= len(sc.s) }
func (sc *scanner) peek() byte      { return sc.s[sc.pos] }
func (sc *scanner) skipSP()         { sc.skipWhile(func(c byte) bool { return c == ' ' }) }
func (sc *scanner) skipOWS()        { sc.skipWhile(func(c byte) bool { return c == ' ' || c == '\t' }) }
func (sc *scanner) skipWhile(f func(byte) bool) {
	for !sc.done() && f(sc.peek()) {
		sc.pos++
	}
}

// parseDictionary reads a whole field value as a dictionary. It returns
// every member in order, repeated keys included.
func parseDictionary(s string) ([]member, error) {
	sc := &scanner{s: s}
	sc.skipSP()
	var out []member
	for !sc.done() {
		start := sc.pos
		m, err := sc.member()
		if err != nil {
			return nil, err
		}
		m.repeat = m.repeat || m.item.repeat
		for _, it := range m.list {
			m.repeat = m.repeat || it.repeat
		}
		m.raw = s[start:sc.pos]
		m.canonical = m.serialise()
		out = append(out, m)
		sc.skipOWS()
		if sc.done() {
			break
		}
		if sc.peek() != ',' {
			return nil, malformed("expected comma between dictionary members")
		}
		sc.pos++
		sc.skipOWS()
		if sc.done() {
			return nil, malformed("trailing comma")
		}
	}
	return out, nil
}

func (sc *scanner) member() (member, error) {
	key, err := sc.key()
	if err != nil {
		return member{}, err
	}
	m := member{key: key}
	if sc.done() || sc.peek() != '=' {
		// A key without a value is the boolean true with parameters.
		m.item.value = bare{kind: kindBoolean, boolean: true}
		m.item.params, m.item.repeat, err = sc.parameters()
		return m, err
	}
	sc.pos++
	if !sc.done() && sc.peek() == '(' {
		m.inner = true
		m.list, err = sc.innerList()
		if err != nil {
			return member{}, err
		}
		m.params, m.repeat, err = sc.parameters()
		return m, err
	}
	m.item, err = sc.item()
	return m, err
}

func (sc *scanner) innerList() ([]item, error) {
	sc.pos++ // (
	var out []item
	for {
		sc.skipSP()
		if sc.done() {
			return nil, malformed("unterminated inner list")
		}
		if sc.peek() == ')' {
			sc.pos++
			return out, nil
		}
		it, err := sc.item()
		if err != nil {
			return nil, err
		}
		out = append(out, it)
		if sc.done() {
			return nil, malformed("unterminated inner list")
		}
		if c := sc.peek(); c != ' ' && c != ')' {
			return nil, malformed("expected space or ) in inner list")
		}
	}
}

func (sc *scanner) item() (item, error) {
	v, err := sc.bareItem()
	if err != nil {
		return item{}, err
	}
	ps, repeat, err := sc.parameters()
	return item{value: v, params: ps, repeat: repeat}, err
}

// parameters reads parameters in order; repeat reports a key that occurs
// more than once.
func (sc *scanner) parameters() (out []param, repeat bool, err error) {
	var seen map[string]struct{} // a map, so many parameters stay linear
	for !sc.done() && sc.peek() == ';' {
		sc.pos++
		sc.skipSP()
		key, err := sc.key()
		if err != nil {
			return nil, false, err
		}
		if seen == nil {
			seen = map[string]struct{}{}
		}
		if _, dup := seen[key]; dup {
			repeat = true
		}
		seen[key] = struct{}{}
		v := bare{kind: kindBoolean, boolean: true}
		if !sc.done() && sc.peek() == '=' {
			sc.pos++
			if v, err = sc.bareItem(); err != nil {
				return nil, false, err
			}
		}
		out = append(out, param{key: key, value: v})
	}
	return out, repeat, nil
}

func (sc *scanner) key() (string, error) {
	start := sc.pos
	if sc.done() || (!isLower(sc.peek()) && sc.peek() != '*') {
		return "", malformed("invalid key")
	}
	sc.skipWhile(func(c byte) bool {
		return isLower(c) || isDigit(c) || c == '_' || c == '-' || c == '.' || c == '*'
	})
	return sc.s[start:sc.pos], nil
}

func (sc *scanner) bareItem() (bare, error) {
	if sc.done() {
		return bare{}, malformed("missing item")
	}
	switch c := sc.peek(); {
	case c == '-' || isDigit(c):
		return sc.number()
	case c == '"':
		return sc.str()
	case c == ':':
		return sc.bytes()
	case c == '?':
		return sc.boolean()
	case isAlpha(c) || c == '*':
		start := sc.pos
		sc.skipWhile(func(c byte) bool { return isTchar(c) || c == ':' || c == '/' })
		return bare{kind: kindToken, text: sc.s[start:sc.pos]}, nil
	default:
		return bare{}, malformed("invalid item")
	}
}

func (sc *scanner) number() (bare, error) {
	start := sc.pos
	neg := sc.peek() == '-'
	if neg {
		sc.pos++
	}
	digits := sc.pos
	sc.skipWhile(isDigit)
	intDigits := sc.pos - digits
	if intDigits == 0 {
		return bare{}, malformed("invalid number")
	}
	if !sc.done() && sc.peek() == '.' {
		sc.pos++
		frac := sc.pos
		sc.skipWhile(isDigit)
		if intDigits > 12 || sc.pos-frac == 0 || sc.pos-frac > 3 {
			return bare{}, malformed("invalid decimal")
		}
		return bare{kind: kindDecimal, text: sc.s[start:sc.pos]}, nil
	}
	if intDigits > 15 {
		return bare{}, malformed("integer too long")
	}
	n, err := strconv.ParseInt(sc.s[start:sc.pos], 10, 64)
	if err != nil {
		return bare{}, malformed("invalid integer")
	}
	return bare{kind: kindInteger, integer: n}, nil
}

func (sc *scanner) str() (bare, error) {
	sc.pos++ // "
	start := sc.pos
	var b *strings.Builder // only when the string has escapes
	for !sc.done() {
		c := sc.peek()
		switch {
		case c == '"':
			text := sc.s[start:sc.pos]
			if b != nil {
				b.WriteString(sc.s[start:sc.pos])
				text = b.String()
			}
			sc.pos++
			return bare{kind: kindString, text: text}, nil
		case c == '\\':
			if sc.pos+1 >= len(sc.s) || (sc.s[sc.pos+1] != '"' && sc.s[sc.pos+1] != '\\') {
				return bare{}, malformed("invalid string escape")
			}
			if b == nil {
				b = &strings.Builder{}
			}
			b.WriteString(sc.s[start:sc.pos])
			b.WriteByte(sc.s[sc.pos+1])
			sc.pos += 2
			start = sc.pos
		case c < 0x20 || c > 0x7e:
			return bare{}, malformed("invalid string character")
		default:
			sc.pos++
		}
	}
	return bare{}, malformed("unterminated string")
}

func (sc *scanner) bytes() (bare, error) {
	sc.pos++ // :
	start := sc.pos
	sc.skipWhile(func(c byte) bool { return isAlpha(c) || isDigit(c) || c == '+' || c == '/' || c == '=' })
	if sc.done() || sc.peek() != ':' {
		return bare{}, malformed("unterminated byte sequence")
	}
	enc := sc.s[start:sc.pos]
	sc.pos++
	// RFC 8941 §4.2.7: parsers SHOULD NOT fail when padding is missing. The
	// canonical check then rejects the member if it is the one verified.
	if r := len(enc) % 4; r != 0 && !strings.Contains(enc, "=") {
		enc += strings.Repeat("=", 4-r)
	}
	dec, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return bare{}, malformed("invalid base64")
	}
	return bare{kind: kindBytes, text: string(dec)}, nil
}

func (sc *scanner) boolean() (bare, error) {
	if sc.pos+1 >= len(sc.s) || (sc.s[sc.pos+1] != '0' && sc.s[sc.pos+1] != '1') {
		return bare{}, malformed("invalid boolean")
	}
	v := sc.s[sc.pos+1] == '1'
	sc.pos += 2
	return bare{kind: kindBoolean, boolean: v}, nil
}

// serialise returns the RFC 8941 serialisation of the member.
func (m member) serialise() string {
	var b strings.Builder
	b.WriteString(m.key)
	switch {
	case m.inner:
		b.WriteString("=(")
		for i, it := range m.list {
			if i > 0 {
				b.WriteByte(' ')
			}
			writeItem(&b, it)
		}
		b.WriteByte(')')
		writeParams(&b, m.params)
	case m.item.value.kind == kindBoolean && m.item.value.boolean:
		writeParams(&b, m.item.params)
	default:
		b.WriteByte('=')
		writeItem(&b, m.item)
	}
	return b.String()
}

func writeItem(b *strings.Builder, it item) {
	writeBare(b, it.value)
	writeParams(b, it.params)
}

func writeParams(b *strings.Builder, ps []param) {
	for _, p := range ps {
		b.WriteByte(';')
		b.WriteString(p.key)
		if p.value.kind == kindBoolean && p.value.boolean {
			continue
		}
		b.WriteByte('=')
		writeBare(b, p.value)
	}
}

func writeBare(b *strings.Builder, v bare) {
	switch v.kind {
	case kindInteger:
		b.WriteString(strconv.FormatInt(v.integer, 10))
	case kindDecimal:
		b.WriteString(canonicalDecimal(v.text))
	case kindString:
		b.WriteByte('"')
		for i := 0; i < len(v.text); i++ {
			if c := v.text[i]; c == '"' || c == '\\' {
				b.WriteByte('\\')
			}
			b.WriteByte(v.text[i])
		}
		b.WriteByte('"')
	case kindToken:
		b.WriteString(v.text)
	case kindBytes:
		b.WriteByte(':')
		b.WriteString(base64.StdEncoding.EncodeToString([]byte(v.text)))
		b.WriteByte(':')
	case kindBoolean:
		if v.boolean {
			b.WriteString("?1")
		} else {
			b.WriteString("?0")
		}
	}
}

// canonicalDecimal drops leading zeros of the integer part and trailing
// zeros of the fraction, keeping one digit on each side (RFC 8941 §4.1.5).
func canonicalDecimal(s string) string {
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	whole, frac, _ := strings.Cut(s, ".")
	whole = strings.TrimLeft(whole, "0")
	frac = strings.TrimRight(frac, "0")
	if whole == "" {
		whole = "0"
	}
	if frac == "" {
		frac = "0"
	}
	if neg && (whole != "0" || frac != "0") {
		return "-" + whole + "." + frac
	}
	return whole + "." + frac
}

func isLower(c byte) bool { return c >= 'a' && c <= 'z' }
func isAlpha(c byte) bool { return isLower(c) || (c >= 'A' && c <= 'Z') }
func isDigit(c byte) bool { return c >= '0' && c <= '9' }
func isTchar(c byte) bool {
	return isAlpha(c) || isDigit(c) || strings.IndexByte("!#$%&'*+-.^_`|~", c) >= 0
}
