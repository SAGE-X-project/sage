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

// Package did parses and compares did:sage identifiers (spec
// 0.9.0-draft.1.1, 06-did-sage.md §1 and §2) and defines the resolution
// interface the host implements (§3).
//
// A DID keeps two forms. The canonical form (String) names the chain
// after ParseChain and is what every comparison uses (E06-04). The form
// as sent (AsSent) is what byte strings that embed a DID use: HPKE info
// and th, and the proof-of-possession challenge. No function here derives
// an owner address from the identifier (E06-01).
package did

import (
	"errors"
	"strings"

	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/core/obs"
)

// Size limits of E06-03.
const (
	MaxDIDBytes    = 256
	MaxKeyRefBytes = 289
)

const prefix = "did:sage:"

// Chain is the chain a DID names. Networks within a chain are
// configuration, not part of the DID (§2).
type Chain uint8

// Chains of §2.
const (
	ChainUnknown Chain = iota
	ChainEthereum
	ChainSolana
)

// String returns the canonical chain name, or "" for an unknown chain.
func (c Chain) String() string {
	switch c {
	case ChainEthereum:
		return "ethereum"
	case ChainSolana:
		return "solana"
	default:
		return ""
	}
}

// ErrUnknownChain is returned by ParseChain for a name that is not a chain.
var ErrUnknownChain = errors.New("did: unknown chain")

// ParseChain trims whitespace and lower-cases name, then accepts ethereum,
// eth, solana and sol (§2). It is for names in configuration; inside a DID
// the chain may not contain whitespace (E06-02), which Parse checks first.
func ParseChain(name string) (Chain, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "ethereum", "eth":
		return ChainEthereum, nil
	case "solana", "sol":
		return ChainSolana, nil
	default:
		return ChainUnknown, ErrUnknownChain
	}
}

// DID is a parsed did:sage identifier. The zero value is not a DID and
// equals nothing. DID values cannot be compared with == or used as map
// keys, because that would compare the form as sent; compare with Equal
// and key maps by String, the canonical form (E06-04).
type DID struct {
	_      [0]func() // not comparable
	asSent string
	chain  Chain
	id     string
}

// Parse accepts did:sage:<chain>:<identifier> per §1 and E06-02: the
// method prefix exactly, a chain ParseChain accepts written in ASCII
// letters, and an identifier of non-empty segments of ALPHA, DIGIT, "-",
// ".", "_" and "~" separated by ":". A DID URL, any other character and a
// DID over 256 bytes are rejected with reason identifier.
func Parse(s string) (DID, error) {
	if len(s) > MaxDIDBytes {
		return DID{}, reject("did exceeds 256 bytes")
	}
	rest, ok := strings.CutPrefix(s, prefix)
	if !ok {
		return DID{}, reject("not a did:sage DID")
	}
	chainPart, id, ok := strings.Cut(rest, ":")
	if !ok || !letters(chainPart) {
		return DID{}, reject("did chain missing or malformed")
	}
	chain, err := ParseChain(chainPart)
	if err != nil {
		return DID{}, reject("did chain unknown")
	}
	if !validIdentifier(id) {
		return DID{}, reject("did identifier malformed")
	}
	return DID{asSent: s, chain: chain, id: id}, nil
}

// New builds the canonical DID of identifier on chain.
func New(chain Chain, identifier string) (DID, error) {
	if chain.String() == "" {
		return DID{}, ErrUnknownChain
	}
	return Parse(prefix + chain.String() + ":" + identifier)
}

// String returns the canonical form: did:sage:<chain>:<identifier>. The
// 256-byte limit applies to the DID as sent (E06-03); when it uses an alias
// the canonical form can be up to five bytes longer, and is then used only
// for comparison, never emitted (New refuses it).
func (d DID) String() string {
	if d.IsZero() {
		return ""
	}
	return prefix + d.chain.String() + ":" + d.id
}

// AsSent returns the DID exactly as it was parsed.
func (d DID) AsSent() string { return d.asSent }

// Chain returns the chain.
func (d DID) Chain() Chain { return d.chain }

// Identifier returns everything after the chain, unchanged.
func (d DID) Identifier() string { return d.id }

// IsZero reports whether d is the zero value.
func (d DID) IsZero() bool { return d.chain == ChainUnknown }

// Equal compares canonical forms (E06-04). The zero DID equals nothing.
func (d DID) Equal(o DID) bool {
	return !d.IsZero() && !o.IsZero() && d.chain == o.chain && d.id == o.id
}

// KeyRef is a keyid: a DID, optionally followed by #key-<n> (E03-14). The
// fragment is informative (E03-15): verification resolves DID() and uses
// the key chosen by §3.1.
type KeyRef struct {
	_        [0]func() // not comparable, like DID
	did      DID
	fragment string
	asSent   string
}

// ParseKeyRef accepts a DID, or a DID followed by "#key-" and a decimal
// integer without leading zeros, of at most 289 bytes in total (E03-14,
// E06-03).
func ParseKeyRef(s string) (KeyRef, error) {
	if len(s) > MaxKeyRefBytes {
		return KeyRef{}, reject("keyid exceeds 289 bytes")
	}
	didPart, fragment, hasFragment := strings.Cut(s, "#")
	d, err := Parse(didPart)
	if err != nil {
		return KeyRef{}, err
	}
	if hasFragment && !validFragment(fragment) {
		return KeyRef{}, reject("keyid fragment is not key-<n>")
	}
	return KeyRef{did: d, fragment: fragment, asSent: s}, nil
}

// DID returns the DID the keyid names.
func (r KeyRef) DID() DID { return r.did }

// Fragment returns "key-<n>", or "" when the keyid has none.
func (r KeyRef) Fragment() string { return r.fragment }

// AsSent returns the keyid exactly as it was parsed.
func (r KeyRef) AsSent() string { return r.asSent }

func validFragment(f string) bool {
	n, ok := strings.CutPrefix(f, "key-")
	if !ok || n == "" || (len(n) > 1 && n[0] == '0') {
		return false
	}
	for i := 0; i < len(n); i++ {
		if n[i] < '0' || n[i] > '9' {
			return false
		}
	}
	return true
}

func letters(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i] | 0x20
		if c < 'a' || c > 'z' {
			return false
		}
	}
	return true
}

func validIdentifier(id string) bool {
	if id == "" {
		return false
	}
	segment := 0
	for i := 0; i < len(id); i++ {
		c := id[i]
		switch {
		case c == ':':
			if segment == 0 {
				return false
			}
			segment = 0
			continue
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9',
			c == '-', c == '.', c == '_', c == '~':
		default:
			return false
		}
		segment++
	}
	return segment > 0
}

func reject(detail string) error {
	return core.Reject(obs.ReasonIdentifier, obs.Known(detail))
}
