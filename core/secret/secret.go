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

// Package secret holds secret bytes (seeds, shared secrets, session keys)
// so they cannot be printed, logged or encoded, and can be erased when their
// use ends (spec 0.9.0-draft.1.1, 00-overview.md §7.3).
//
// A Bytes never hands out its slice: code borrows it inside Use. Every
// printing and encoding path (fmt verbs, encoding/json, encoding/text,
// log/slog, encoding/gob) yields "[redacted]" or an error. Destroy
// overwrites the bytes with zeros and drops them; later uses fail.
//
// Limits of Go: the garbage collector may move or copy a backing array
// before Destroy runs, and values derived from a secret by other packages
// (crypto/ecdh private keys, AEAD instances) keep internal copies this
// package cannot overwrite. Destroy reduces exposure; it does not guarantee
// that no copy of the bytes remains in memory. Tools that read memory by
// reflection and ignore methods (for example go-spew with DisableMethods,
// which testify uses for its diffs, or reflect.Value) can still print the
// bytes; do not pass a secret to such tools, and do not compare secrets
// with reflect.DeepEqual or testify's Equal. Use Equal.
package secret

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
)

// Redacted is the text every printing or encoding path of a secret yields.
const Redacted = "[redacted]"

// ErrDestroyed is returned when a destroyed or zero Bytes is used.
var ErrDestroyed = errors.New("secret: destroyed or empty")

// ErrEncode is returned by encoders that refuse to encode a secret.
var ErrEncode = errors.New("secret: refusing to encode a secret")

// Bytes is a secret byte string. Copies of a Bytes share one buffer, so
// Destroy on any copy destroys all of them. The zero value is destroyed.
// Bytes is not comparable: == and map keys do not compile; use Equal.
type Bytes struct {
	_ [0]func()
	p *buffer
}

// buffer keeps the bytes two pointer hops away from Bytes. When fmt prints
// a struct holding a Bytes in an unexported field with a verb that does not
// apply to pointers (%s, %q), it expands the first pointer it meets; the
// second hop makes it print an address instead of the bytes.
type buffer struct {
	mu    sync.RWMutex
	b     *[]byte
	alive atomic.Bool
	n     atomic.Int64
}

func newBuffer(b []byte) *buffer {
	buf := &buffer{b: &b}
	buf.alive.Store(true)
	buf.n.Store(int64(len(b)))
	return buf
}

// From copies b into a new secret. The caller still owns b and should clear
// it when it no longer needs it. An empty b gives the zero (destroyed)
// secret, so a missing key cannot pass as an empty one.
func From(b []byte) Bytes {
	if len(b) == 0 {
		return Bytes{}
	}
	owned := make([]byte, len(b))
	copy(owned, b)
	return Bytes{p: newBuffer(owned)}
}

// Random reads n bytes from r into a new secret. r is normally
// crypto/rand.Reader; tests inject a fixed reader.
func Random(r io.Reader, n int) (Bytes, error) {
	if n <= 0 {
		return Bytes{}, errors.New("secret: length must be positive")
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		clear(b)
		return Bytes{}, err
	}
	return Bytes{p: newBuffer(b)}, nil
}

// Use calls f with the secret bytes. f must not keep, return or change the
// slice, and must not block. Inside f, do not call Use, Equal or Destroy on
// the same secret or a copy of it: the read lock is not reentrant, so such a
// call deadlocks when a Destroy is waiting. Len and Destroyed are safe
// inside f. When f borrows two secrets by nesting Use, nest them in the same
// order everywhere. Use returns ErrDestroyed for a destroyed or zero secret,
// and otherwise what f returns. Destroy waits until running uses end.
func (s Bytes) Use(f func(b []byte) error) error {
	if s.p == nil {
		return ErrDestroyed
	}
	s.p.mu.RLock()
	defer s.p.mu.RUnlock()
	if s.p.b == nil {
		return ErrDestroyed
	}
	return f(*s.p.b)
}

// Len returns the length of the secret, or 0 once destroyed. It takes no
// lock.
func (s Bytes) Len() int {
	if s.p == nil {
		return 0
	}
	return int(s.p.n.Load())
}

// Destroyed reports whether the secret was destroyed or is the zero value.
// It takes no lock.
func (s Bytes) Destroyed() bool {
	return s.p == nil || !s.p.alive.Load()
}

// Destroy overwrites the bytes with zeros and drops them. It is safe to call
// more than once and on the zero value. It must not be called inside Use of
// the same secret.
func (s Bytes) Destroy() {
	if s.p == nil {
		return
	}
	s.p.alive.Store(false)
	s.p.n.Store(0)
	s.p.mu.Lock()
	defer s.p.mu.Unlock()
	if s.p.b != nil {
		clear(*s.p.b)
		s.p.b = nil
	}
}

// Equal reports in constant time whether two secrets hold the same bytes.
// Only the lengths may be learned from timing. A destroyed secret equals
// nothing.
func (s Bytes) Equal(o Bytes) bool {
	if s.p == nil || o.p == nil {
		return false
	}
	if s.p == o.p {
		return !s.Destroyed()
	}
	// Copy o under its own lock, then compare under s's lock, so two
	// goroutines comparing in opposite order cannot deadlock. The copy is
	// cleared before returning.
	var other []byte
	if err := o.Use(func(b []byte) error {
		other = make([]byte, len(b))
		copy(other, b)
		return nil
	}); err != nil {
		return false
	}
	defer clear(other)
	equal := false
	_ = s.Use(func(b []byte) error {
		equal = subtle.ConstantTimeCompare(b, other) == 1
		return nil
	})
	return equal
}

// String returns Redacted.
func (s Bytes) String() string { return Redacted }

// GoString returns Redacted, for %#v.
func (s Bytes) GoString() string { return Redacted }

// Format writes Redacted for every verb, including %x, %q and %v.
func (s Bytes) Format(f fmt.State, _ rune) { _, _ = io.WriteString(f, Redacted) }

// MarshalJSON returns the JSON string "[redacted]".
func (s Bytes) MarshalJSON() ([]byte, error) { return []byte(`"` + Redacted + `"`), nil }

// MarshalText returns Redacted.
func (s Bytes) MarshalText() ([]byte, error) { return []byte(Redacted), nil }

// LogValue returns Redacted for log/slog.
func (s Bytes) LogValue() slog.Value { return slog.StringValue(Redacted) }

// GobEncode refuses to encode a secret.
func (s Bytes) GobEncode() ([]byte, error) { return nil, ErrEncode }
