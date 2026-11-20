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

// Package httpsig reads and checks RFC 9421 HTTP message signatures as SAGE
// 0.9.0-draft.1.1 profiles them (03-rfc9421.md).
//
// Parse is step 1 of §5: it reads Signature-Input and Signature into a
// Parsed value, which is syntactically valid but not verified. It enforces
// the size limits (E03-21), one field line per signature-relevant field
// (E03-07), one label (E03-06), the five defined signature parameters and no
// tag (E03-03), and the canonical serialisation of the member (E03-05).
// Covered components are parsed but checked by the verifier (§2 to §4).
package httpsig

import (
	"net/http"
	"strings"

	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/core/obs"
)

// Field names (canonical MIME form, as net/http keys them).
const (
	FieldSignatureInput = "Signature-Input"
	FieldSignature      = "Signature"
	FieldContentDigest  = "Content-Digest"
	FieldSAGEDID        = "X-Sage-Did"
)

// Parsed is a signature read from a message. Nothing in it is verified.
type Parsed struct {
	label      string
	components []Component
	params     string // the @signature-params value: inner list and parameters
	keyid      opt[string]
	alg        opt[string]
	nonce      opt[string]
	created    opt[int64]
	expires    opt[int64]
	signature  []byte
}

type opt[T any] struct {
	v  T
	ok bool
}

// Component is one covered component identifier: its name and parameters.
type Component struct {
	name   string
	params []param
}

// Name returns the component name, for example "@method" or "content-type".
func (c Component) Name() string { return c.name }

// Has reports whether the component carries parameter key, such as "req".
func (c Component) Has(key string) bool {
	for _, p := range c.params {
		if p.key == key {
			return true
		}
	}
	return false
}

// Identifier returns the serialised component identifier as it appears in
// the signature base, for example `"@method";req`.
func (c Component) Identifier() string {
	var b strings.Builder
	writeItem(&b, item{value: bare{kind: kindString, text: c.name}, params: c.params})
	return b.String()
}

// Label returns the selected signature label.
func (p Parsed) Label() string { return p.label }

// Components returns a copy of the covered components in order.
func (p Parsed) Components() []Component { return append([]Component(nil), p.components...) }

// SignatureParams returns the @signature-params value: the received member
// without "<label>=", which equals its canonical serialisation.
func (p Parsed) SignatureParams() string { return p.params }

// KeyID returns the keyid parameter.
func (p Parsed) KeyID() (string, bool) { return p.keyid.v, p.keyid.ok }

// Alg returns the alg parameter.
func (p Parsed) Alg() (string, bool) { return p.alg.v, p.alg.ok }

// Nonce returns the nonce parameter.
func (p Parsed) Nonce() (string, bool) { return p.nonce.v, p.nonce.ok }

// Created returns the created parameter.
func (p Parsed) Created() (int64, bool) { return p.created.v, p.created.ok }

// Expires returns the expires parameter.
func (p Parsed) Expires() (int64, bool) { return p.expires.v, p.expires.ok }

// Signature returns a copy of the signature bytes of the selected label.
func (p Parsed) Signature() []byte { return append([]byte(nil), p.signature...) }

// SectionSize returns the size of the header section as transferred:
// "name: value\r\n" for every field line.
func SectionSize(h http.Header) int {
	n := 0
	for name, values := range h {
		for _, v := range values {
			n += len(name) + len(": ") + len(v) + len("\r\n")
		}
	}
	return n
}

// Parse reads the signature labelled label from h, or the only signature
// when label is "". With a label, other members of both dictionaries are
// ignored but must still be valid structured fields (E03-06). lim supplies
// the size limits; a Limits outside the spec ranges returns core.ErrLimits.
func Parse(h http.Header, label string, lim core.Limits) (Parsed, error) {
	if err := lim.Validate(); err != nil {
		return Parsed{}, err
	}
	if SectionSize(h) > lim.MaxHeaderBytes {
		return Parsed{}, reject(obs.ReasonTooLarge, "header section exceeds its limit")
	}
	inputValue, err := oneLine(h, FieldSignatureInput, true)
	if err != nil {
		return Parsed{}, err
	}
	sigValue, err := oneLine(h, FieldSignature, true)
	if err != nil {
		return Parsed{}, err
	}
	for _, name := range []string{FieldContentDigest, FieldSAGEDID} {
		if _, err := oneLine(h, name, false); err != nil {
			return Parsed{}, err
		}
	}
	if len(inputValue) > lim.MaxSignatureFieldBytes || len(sigValue) > lim.MaxSignatureFieldBytes {
		return Parsed{}, reject(obs.ReasonTooLarge, "signature field exceeds 8 KiB")
	}
	inputs, err := parseDictionary(inputValue)
	if err != nil {
		return Parsed{}, err
	}
	sigs, err := parseDictionary(sigValue)
	if err != nil {
		return Parsed{}, err
	}
	in, sig, err := selectLabel(inputs, sigs, label)
	if err != nil {
		return Parsed{}, err
	}
	return build(in, sig, lim)
}

// oneLine returns the value of the one field line of name. A required field
// must be present; no field may have more than one line (E03-07). Keys are
// compared case-insensitively, so a header built without net/http's
// canonical keys cannot hide a second line.
func oneLine(h http.Header, name string, required bool) (string, error) {
	var values []string
	for key, vs := range h {
		if strings.EqualFold(key, name) {
			values = append(values, vs...)
		}
	}
	switch {
	case len(values) > 1:
		return "", reject(obs.ReasonDuplicate, "more than one field line")
	case len(values) == 0 && required:
		return "", malformed("missing signature field")
	case len(values) == 0:
		return "", nil
	}
	return values[0], nil
}

// selectLabel picks the signature to verify (E03-06). Without a named
// label each dictionary must hold exactly one member, under the same label;
// a repeated label counts as more than one. With a named label every other
// member is ignored, whatever it holds, but the named label must occur
// exactly once in each dictionary.
func selectLabel(inputs, sigs []member, label string) (member, member, error) {
	if label == "" {
		if len(inputs) > 1 || len(sigs) > 1 {
			return member{}, member{}, reject(obs.ReasonDuplicate, "more than one signature label")
		}
		if len(inputs) == 0 || len(sigs) == 0 {
			return member{}, member{}, malformed("missing signature label")
		}
		if inputs[0].key != sigs[0].key {
			return member{}, member{}, malformed("signature labels differ")
		}
		return inputs[0], sigs[0], nil
	}
	in, nIn := find(inputs, label)
	sig, nSig := find(sigs, label)
	switch {
	case nIn > 1 || nSig > 1:
		return member{}, member{}, reject(obs.ReasonDuplicate, "named signature label repeated")
	case nIn == 0 || nSig == 0:
		return member{}, member{}, malformed("named signature label not present")
	}
	return in, sig, nil
}

// find returns the member with key and how many members have it.
func find(ms []member, key string) (member, int) {
	var found member
	n := 0
	for _, m := range ms {
		if m.key == key {
			found = m
			n++
		}
	}
	return found, n
}

func build(in, sig member, lim core.Limits) (Parsed, error) {
	if !in.inner {
		return Parsed{}, malformed("Signature-Input member is not an inner list")
	}
	if sig.inner || sig.item.value.kind != kindBytes {
		return Parsed{}, malformed("Signature member is not a byte sequence")
	}
	if len(sig.item.params) > 0 {
		return Parsed{}, reject(obs.ReasonUnknownMember, "parameter on the Signature member")
	}
	if in.repeat {
		return Parsed{}, reject(obs.ReasonDuplicate, "repeated parameter")
	}
	p := Parsed{label: in.key, signature: []byte(sig.item.value.text)}
	for _, prm := range in.params {
		if err := p.setParam(prm); err != nil {
			return Parsed{}, err
		}
	}
	for _, it := range in.list {
		if it.value.kind != kindString {
			return Parsed{}, malformed("covered component is not a string")
		}
		p.components = append(p.components, Component{name: it.value.text, params: it.params})
	}
	// E03-05 for Signature-Input; for Signature, 03 §1 (standard base64)
	// and DP-2.1 (base64 values have one accepted form).
	if in.raw != in.canonical || sig.raw != sig.canonical {
		return Parsed{}, reject(obs.ReasonNonCanonical, "signature field is not in canonical form")
	}
	if len(p.components) > lim.MaxCoveredComponents {
		return Parsed{}, reject(obs.ReasonTooLarge, "too many covered components")
	}
	p.params = strings.TrimPrefix(in.canonical, in.key+"=")
	return p, nil
}

// setParam records one signature parameter of 03 §1.
func (p *Parsed) setParam(prm param) error {
	text := func(dst *opt[string]) error {
		if prm.value.kind != kindString {
			return malformed("signature parameter is not a string")
		}
		*dst = opt[string]{v: prm.value.text, ok: true}
		return nil
	}
	integer := func(dst *opt[int64]) error {
		if prm.value.kind != kindInteger || prm.value.integer < 0 {
			return malformed("signature parameter is not a non-negative integer")
		}
		*dst = opt[int64]{v: prm.value.integer, ok: true}
		return nil
	}
	switch prm.key {
	case "keyid":
		return text(&p.keyid)
	case "alg":
		return text(&p.alg)
	case "nonce":
		return text(&p.nonce)
	case "created":
		return integer(&p.created)
	case "expires":
		return integer(&p.expires)
	case "tag":
		return reject(obs.ReasonParameters, "tag is not allowed")
	default:
		return reject(obs.ReasonUnknownMember, "unknown signature parameter")
	}
}

func reject(reason obs.Reason, detail string) error {
	return core.Reject(reason, obs.Known(detail))
}
