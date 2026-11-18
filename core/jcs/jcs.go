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

// Package jcs reads signed JSON strictly and writes it in the JSON
// Canonicalization Scheme of RFC 8785 (spec 0.9.0-draft.1.1, 02-jcs.md).
//
// Parse accepts one well-formed RFC 8259 UTF-8 text and rejects, with a
// *core.Rejection, everything 02-jcs.md §2 forbids: a byte order mark,
// invalid UTF-8, unpaired surrogate escapes, trailing data, duplicate member
// names, names equal under ASCII case folding, numbers where the limits
// forbid them, and input over its size, depth or entry limits. The Value it
// returns is the one parse a verifier both canonicalises and acts on.
//
// Parse does not know which object it reads. The verifier of each signed
// object MUST also require a top-level object, its exact member set (which
// rejects a name that differs from a defined name only in case, E02-03) and
// any object-specific cap such as the 32 card keys and services (E02-06).
// Case folding is ASCII only, as 02-jcs.md §2 defines it: "kid" and "\u212aid"
// (KELVIN SIGN) are different names, so a host must not decode the same bytes
// again with a parser that folds Unicode case.
//
// Memory: parsing allocates about 15 to 20 bytes per byte of text for the
// worst texts measured (many empty arrays, many one-member objects;
// TestMemoryBound), so up to about 20 MiB for a 1 MiB transport envelope.
// Hosts bound the number of texts parsed at once.
package jcs

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/core/obs"
)

// Limits of signed JSON (02-jcs.md §2, errata E02-06).
const (
	// MaxSignedBytes bounds the HPKE envelopes and the agent card.
	MaxSignedBytes = 64 << 10
	// MaxTransportBytes bounds WireMessage and WireResponse (08-transport.md).
	MaxTransportBytes = 1 << 20
	// MaxDepth is the deepest nesting of objects and arrays.
	MaxDepth = 16
	// MaxEntries bounds the members of one object and the elements of one
	// array.
	MaxEntries = 256
	// HardMaxDepth is the largest MaxDepth a Limits may set; it bounds the
	// parser's recursion.
	HardMaxDepth = 64
)

// ErrLimits is returned for a Limits value that is not usable. It is a
// programming error of the caller, not a rejection of input.
var ErrLimits = errors.New("jcs: invalid limits")

// ErrInvalidValue is returned by Canonical for a Value that Parse did not
// make, such as the zero Value. It is a programming error of the caller.
var ErrInvalidValue = errors.New("jcs: invalid value")

// Limits bounds one JSON text. The zero value is not usable; start from
// SignedLimits or TransportLimits.
type Limits struct {
	MaxBytes   int  // whole text, checked before parsing
	MaxDepth   int  // 1 to HardMaxDepth; a top-level object or array is depth 1
	MaxEntries int  // members of one object, elements of one array
	Numbers    bool // accept JSON numbers; signed SAGE objects have none (E02-05)
}

// SignedLimits returns the limits of the HPKE envelopes and the agent card.
func SignedLimits() Limits {
	return Limits{MaxBytes: MaxSignedBytes, MaxDepth: MaxDepth, MaxEntries: MaxEntries}
}

// TransportLimits returns the limits of WireMessage and WireResponse.
func TransportLimits() Limits {
	return Limits{MaxBytes: MaxTransportBytes, MaxDepth: MaxDepth, MaxEntries: MaxEntries}
}

func (l Limits) valid() bool {
	return l.MaxBytes > 0 && l.MaxDepth > 0 && l.MaxDepth <= HardMaxDepth && l.MaxEntries > 0
}

// Kind is the JSON type of a Value.
type Kind uint8

// Value kinds. The zero Kind marks a Value that was not made by Parse.
const (
	KindInvalid Kind = iota
	KindNull
	KindBool
	KindNumber
	KindString
	KindArray
	KindObject
)

// Value is one parsed JSON value. It is immutable: no method changes it, and
// methods that return parts return copies or values that share no writable
// state with the caller.
type Value struct {
	kind    Kind
	boolean bool
	number  float64
	str     string
	c       *container // nil for scalars and empty containers
}

// container holds the entries of an array or object in slices of exact
// length. For an object, names[i] is the name of items[i].
type container struct {
	items []Value
	names []string
}

func (v Value) items() []Value {
	if v.c == nil {
		return nil
	}
	return v.c.items
}

func (v Value) names() []string {
	if v.c == nil {
		return nil
	}
	return v.c.names
}

// Kind returns the JSON type, or KindInvalid for the zero Value.
func (v Value) Kind() Kind { return v.kind }

// Bool returns a boolean value.
func (v Value) Bool() (bool, bool) { return v.boolean, v.kind == KindBool }

// Number returns a number value.
func (v Value) Number() (float64, bool) { return v.number, v.kind == KindNumber }

// Str returns a string value.
func (v Value) Str() (string, bool) { return v.str, v.kind == KindString }

// Len returns the number of members of an object or elements of an array,
// and 0 for any other kind.
func (v Value) Len() int {
	if v.kind != KindObject && v.kind != KindArray {
		return 0
	}
	return len(v.items())
}

// Index returns element i of an array.
func (v Value) Index(i int) (Value, bool) {
	if v.kind != KindArray || i < 0 || i >= len(v.items()) {
		return Value{}, false
	}
	return v.c.items[i], true
}

// Get returns the member of an object with exactly this name.
func (v Value) Get(name string) (Value, bool) {
	if v.kind != KindObject {
		return Value{}, false
	}
	for i, n := range v.names() {
		if n == name {
			return v.c.items[i], true
		}
	}
	return Value{}, false
}

// Names returns the member names of an object in received order.
func (v Value) Names() []string {
	if v.kind != KindObject {
		return nil
	}
	return append([]string{}, v.names()...)
}

// Without returns the object without its top-level member name, and that
// member's value. It is how a verifier removes the signature member before
// canonicalising (02-jcs.md §2): only the one top-level member is removed.
// A missing member gives ok == false; the verifier rejects the object, it
// does not treat it as unsigned.
func (v Value) Without(name string) (rest, removed Value, ok bool) {
	if v.kind != KindObject {
		return Value{}, Value{}, false
	}
	names := v.names()
	for i, n := range names {
		if n != name {
			continue
		}
		rest = Value{kind: KindObject}
		if len(names) > 1 {
			rest.c = &container{
				items: concat(v.c.items[:i], v.c.items[i+1:]),
				names: concat(names[:i], names[i+1:]),
			}
		}
		return rest, v.c.items[i], true
	}
	return Value{}, Value{}, false
}

func concat[T any](a, b []T) []T {
	out := make([]T, 0, len(a)+len(b))
	return append(append(out, a...), b...)
}

// Parse reads one JSON text within lim. Input that breaks a rule of
// 02-jcs.md §2 is rejected with a *core.Rejection; a Limits that is not
// usable returns ErrLimits.
func Parse(data []byte, lim Limits) (Value, error) {
	if !lim.valid() {
		return Value{}, ErrLimits
	}
	if len(data) > lim.MaxBytes {
		return Value{}, reject(obs.ReasonTooLarge, "json text exceeds its size limit")
	}
	if !utf8.Valid(data) {
		return Value{}, reject(obs.ReasonMalformed, "json text is not valid UTF-8")
	}
	p := parser{data: data, lim: lim}
	p.space()
	v, err := p.value(0)
	if err != nil {
		return Value{}, err
	}
	p.space()
	if p.pos != len(p.data) {
		return Value{}, reject(obs.ReasonMalformed, "data after the json value")
	}
	return v, nil
}

// Canonicalize parses data within lim and returns its RFC 8785 form.
func Canonicalize(data []byte, lim Limits) ([]byte, error) {
	v, err := Parse(data, lim)
	if err != nil {
		return nil, err
	}
	return Canonical(v)
}

func reject(reason obs.Reason, detail string) error {
	return core.Reject(reason, obs.Known(detail))
}

func malformed(detail string) error { return reject(obs.ReasonMalformed, detail) }

type parser struct {
	data []byte
	pos  int
	lim  Limits
	// Entries of the containers being read are pushed here and copied into
	// slices of exact length when their container closes, so a Value holds
	// no spare capacity.
	items []Value
	names []string
}

// take moves the entries pushed since base into a new container. The two
// stacks have their own bases: an array pushes items but no names.
func (p *parser) take(base stackBase, object bool) *container {
	c := &container{items: append([]Value(nil), p.items[base.items:]...)}
	clear(p.items[base.items:])
	p.items = p.items[:base.items]
	if object {
		c.names = append([]string(nil), p.names[base.names:]...)
		clear(p.names[base.names:])
		p.names = p.names[:base.names]
	}
	return c
}

type stackBase struct{ items, names int }

func (p *parser) base() stackBase { return stackBase{items: len(p.items), names: len(p.names)} }

func (p *parser) space() {
	for p.pos < len(p.data) {
		switch p.data[p.pos] {
		case ' ', '\t', '\n', '\r':
			p.pos++
		default:
			return
		}
	}
}

// value reads one value. depth counts the containers that enclose it.
func (p *parser) value(depth int) (Value, error) {
	if p.pos >= len(p.data) {
		return Value{}, malformed("json value expected")
	}
	switch c := p.data[p.pos]; {
	case c == '{':
		return p.object(depth + 1)
	case c == '[':
		return p.array(depth + 1)
	case c == '"':
		s, err := p.string()
		return Value{kind: KindString, str: s}, err
	case c == 't':
		return Value{kind: KindBool, boolean: true}, p.literal("true")
	case c == 'f':
		return Value{kind: KindBool}, p.literal("false")
	case c == 'n':
		return Value{kind: KindNull}, p.literal("null")
	case c == '-' || (c >= '0' && c <= '9'):
		return p.number()
	default:
		return Value{}, malformed("json value expected")
	}
}

func (p *parser) literal(word string) error {
	if len(p.data)-p.pos < len(word) || string(p.data[p.pos:p.pos+len(word)]) != word {
		return malformed("invalid literal")
	}
	p.pos += len(word)
	return nil
}

func (p *parser) enter(depth int) error {
	if depth > p.lim.MaxDepth {
		return reject(obs.ReasonTooLarge, "json nested too deeply")
	}
	p.pos++ // opening bracket
	p.space()
	return nil
}

func (p *parser) separator(closing byte) (done bool, err error) {
	p.space()
	if p.pos >= len(p.data) {
		return false, malformed("unterminated json container")
	}
	switch p.data[p.pos] {
	case ',':
		p.pos++
		p.space()
		return false, nil
	case closing:
		p.pos++
		return true, nil
	default:
		return false, malformed("expected separator")
	}
}

func (p *parser) array(depth int) (Value, error) {
	if err := p.enter(depth); err != nil {
		return Value{}, err
	}
	if p.pos < len(p.data) && p.data[p.pos] == ']' {
		p.pos++
		return Value{kind: KindArray}, nil
	}
	base := p.base()
	for count := 1; ; count++ {
		item, err := p.value(depth)
		if err != nil {
			return Value{}, err
		}
		// Counted after the entry is read, so a syntax error is reported
		// as malformed whatever the count.
		if count > p.lim.MaxEntries {
			return Value{}, reject(obs.ReasonTooLarge, "json array has too many elements")
		}
		p.items = append(p.items, item)
		done, err := p.separator(']')
		if err != nil {
			return Value{}, err
		}
		if done {
			return Value{kind: KindArray, c: p.take(base, false)}, nil
		}
	}
}

func (p *parser) object(depth int) (Value, error) {
	if err := p.enter(depth); err != nil {
		return Value{}, err
	}
	if p.pos < len(p.data) && p.data[p.pos] == '}' {
		p.pos++
		return Value{kind: KindObject}, nil
	}
	base := p.base()
	var folded map[string]struct{} // made when a second member arrives
	for count := 1; ; count++ {
		if p.pos >= len(p.data) || p.data[p.pos] != '"' {
			return Value{}, malformed("member name expected")
		}
		name, err := p.string()
		if err != nil {
			return Value{}, err
		}
		p.space()
		if p.pos >= len(p.data) || p.data[p.pos] != ':' {
			return Value{}, malformed("colon expected")
		}
		p.pos++
		p.space()
		item, err := p.value(depth)
		if err != nil {
			return Value{}, err
		}
		if count > p.lim.MaxEntries {
			return Value{}, reject(obs.ReasonTooLarge, "json object has too many members")
		}
		// Exact duplicates are equal under folding too (E02-02, E02-03).
		if count == 2 {
			folded = make(map[string]struct{}, 8)
			folded[foldASCII(p.names[base.names])] = struct{}{}
		}
		if folded != nil {
			key := foldASCII(name)
			if _, dup := folded[key]; dup {
				return Value{}, reject(obs.ReasonDuplicate, "duplicate or case-variant member name")
			}
			folded[key] = struct{}{}
		}
		p.items = append(p.items, item)
		p.names = append(p.names, name)
		done, err := p.separator('}')
		if err != nil {
			return Value{}, err
		}
		if done {
			return Value{kind: KindObject, c: p.take(base, true)}, nil
		}
	}
}

func foldASCII(s string) string {
	if !strings.ContainsFunc(s, func(r rune) bool { return r >= 'A' && r <= 'Z' }) {
		return s
	}
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}

// string reads a string at p.pos, which holds the opening quote. The text is
// already known to be valid UTF-8.
func (p *parser) string() (string, error) {
	p.pos++
	var out []byte
	start := p.pos
	for p.pos < len(p.data) {
		c := p.data[p.pos]
		switch {
		case c == '"':
			out = append(out, p.data[start:p.pos]...)
			p.pos++
			return string(out), nil
		case c < 0x20:
			return "", malformed("control character in json string")
		case c == '\\':
			out = append(out, p.data[start:p.pos]...)
			var err error
			if out, err = p.escape(out); err != nil {
				return "", err
			}
			start = p.pos
		default:
			p.pos++
		}
	}
	return "", malformed("unterminated json string")
}

// escape appends the character of the escape at p.pos to out.
func (p *parser) escape(out []byte) ([]byte, error) {
	if p.pos+1 >= len(p.data) {
		return nil, malformed("unterminated escape")
	}
	c := p.data[p.pos+1]
	p.pos += 2
	switch c {
	case '"', '\\', '/':
		return append(out, c), nil
	case 'b':
		return append(out, '\b'), nil
	case 'f':
		return append(out, '\f'), nil
	case 'n':
		return append(out, '\n'), nil
	case 'r':
		return append(out, '\r'), nil
	case 't':
		return append(out, '\t'), nil
	case 'u':
	default:
		return nil, malformed("invalid escape")
	}
	r, err := p.hex4()
	if err != nil {
		return nil, err
	}
	switch {
	case utf16.IsSurrogate(r) && r < 0xdc00:
		if p.pos+1 >= len(p.data) || p.data[p.pos] != '\\' || p.data[p.pos+1] != 'u' {
			return nil, malformed("unpaired surrogate escape")
		}
		p.pos += 2
		low, err := p.hex4()
		if err != nil {
			return nil, err
		}
		if low < 0xdc00 || low > 0xdfff {
			return nil, malformed("unpaired surrogate escape")
		}
		r = utf16.DecodeRune(r, low)
	case utf16.IsSurrogate(r):
		return nil, malformed("unpaired surrogate escape")
	}
	return utf8.AppendRune(out, r), nil
}

func (p *parser) hex4() (rune, error) {
	if len(p.data)-p.pos < 4 {
		return 0, malformed("short unicode escape")
	}
	var r rune
	for _, c := range p.data[p.pos : p.pos+4] {
		var d byte
		switch {
		case c >= '0' && c <= '9':
			d = c - '0'
		case c >= 'a' && c <= 'f':
			d = c - 'a' + 10
		case c >= 'A' && c <= 'F':
			d = c - 'A' + 10
		default:
			return 0, malformed("invalid unicode escape")
		}
		r = r<<4 | rune(d)
	}
	p.pos += 4
	return r, nil
}

// number reads a number per the RFC 8259 grammar; strconv.ParseFloat alone
// would also accept forms such as "0x1p3", "Inf" and "1_0".
func (p *parser) number() (Value, error) {
	start := p.pos
	if p.data[p.pos] == '-' {
		p.pos++
	}
	switch {
	case p.pos < len(p.data) && p.data[p.pos] == '0':
		p.pos++
	case p.digits() == 0:
		return Value{}, malformed("invalid number")
	}
	if p.pos < len(p.data) && p.data[p.pos] == '.' {
		p.pos++
		if p.digits() == 0 {
			return Value{}, malformed("invalid number")
		}
	}
	if p.pos < len(p.data) && (p.data[p.pos] == 'e' || p.data[p.pos] == 'E') {
		p.pos++
		if p.pos < len(p.data) && (p.data[p.pos] == '+' || p.data[p.pos] == '-') {
			p.pos++
		}
		if p.digits() == 0 {
			return Value{}, malformed("invalid number")
		}
	}
	if !p.lim.Numbers {
		return Value{}, malformed("number in signed json")
	}
	f, err := strconv.ParseFloat(string(p.data[start:p.pos]), 64)
	if err != nil || math.IsInf(f, 0) {
		return Value{}, malformed("number out of range")
	}
	return Value{kind: KindNumber, number: f}, nil
}

func (p *parser) digits() int {
	n := 0
	for p.pos < len(p.data) && p.data[p.pos] >= '0' && p.data[p.pos] <= '9' {
		p.pos++
		n++
	}
	return n
}
