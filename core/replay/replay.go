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

// Package replay keeps the nonces a receiver has accepted (spec
// 0.9.0-draft.1.1, 00-overview.md §7 DP-4, errata E03-12, E04-14, E08-01).
//
// A receiver reserves a nonce as the last check, after the signature and
// every other check have passed. The store itself applies the rules the
// spec gives each scope, so a receiver cannot get them wrong: it computes
// how long an entry is kept, rejects input dated before the store was
// restored, never evicts a live entry and refuses new input when full. The
// windows those rules use are the receiver's own MaxAge and MaxClockSkew,
// which the receiver passes with each reservation (03 §5 Replay, E08-01),
// so the store cannot keep an entry for less time than the receiver
// accepts its signature. A store serves one receiver configuration: every
// reservation of a scope must carry the same windows, so a receiver with a
// longer window never finds an entry that a shorter one let expire.
// Capacity is charged to the authenticated identity that sent the input,
// so one sender cannot fill the store for others (L2, R15).
package replay

import (
	"container/heap"
	"context"
	"errors"
	"sync"
	"time"

	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/core/clock"
)

// Scope selects the protocol rules and separates namespaces.
type Scope uint8

// Scopes.
const (
	ScopeUnknown      Scope = iota
	ScopeHTTPRequest        // RFC 9421: (signer DID, nonce), E03-12
	ScopeHPKEInit           // HPKE init: (ctxID, nonce), E04-14
	ScopeWireEnvelope       // WireMessage: (did, nonce), E08-01
	scopeEnd
)

// Windows the spec fixes per scope.
const (
	// TransportWindow is the fixed acceptance window of WireMessage
	// created (E08-01); it does not follow Limits.MaxAge.
	TransportWindow = 300 * time.Second
	// HPKEInitWindow is the ±120 s window of the init ts (04 §6) and the
	// restore offset of E04-14.
	HPKEInitWindow = 120 * time.Second
	// HPKEInitRetention is how long an init nonce is kept (E04-14).
	HPKEInitRetention = 600 * time.Second
	// MaxFieldBytes bounds Owner, ID and Nonce.
	MaxFieldBytes = 512
)

// ErrSeen is returned for a nonce that is recorded and not expired.
var ErrSeen = errors.New("replay: already seen")

// ErrFull is returned when there is no room; the store never evicts a live
// entry to make room.
var ErrFull = errors.New("replay: store full")

// ErrRestored is returned for input dated before the store's restore time
// plus the scope's window (E03-12, E04-14).
var ErrRestored = errors.New("replay: input predates the replay store")

// ErrWindows is returned for a reservation whose MaxAge or Skew differs
// from those of the earlier reservations of its scope: a store serves one
// receiver configuration, and a receiver whose limits change needs a new
// store (which applies the restore rule).
var ErrWindows = errors.New("replay: windows differ from those the store serves")

// ErrInvalid is returned for an invalid argument or for input whose time is
// outside its acceptance window, which the receiver must have rejected
// before reserving: both are caller errors.
var ErrInvalid = errors.New("replay: invalid argument")

// Reservation names one accepted nonce.
type Reservation struct {
	Scope Scope
	// Owner is the authenticated identity charged for the entry: the
	// canonical DID of the signer (for HPKE, of initDid).
	Owner string
	// ID is the namespace of the nonce: the canonical signer DID for RFC
	// 9421 and envelopes, the ctxID for HPKE.
	ID    string
	Nonce string
	// Created is the signed time: RFC 9421 created, WireMessage created,
	// HPKE ts.
	Created time.Time
	// MaxAge is the receiver's RFC 9421 MaxAge, in (0, 300 s], for
	// ScopeHTTPRequest; it is zero for the other scopes, whose windows the
	// spec fixes.
	MaxAge time.Duration
	// Skew is the receiver's MaxClockSkew, in [0, 300 s], for
	// ScopeHTTPRequest and ScopeWireEnvelope; it is zero for ScopeHPKEInit.
	Skew time.Duration
}

// Store reserves nonces. Implementations are safe for concurrent use.
type Store interface {
	// Reserve records r. It returns ErrSeen, ErrFull or ErrRestored for
	// input that must be rejected, ErrWindows when r's windows differ from
	// those of the earlier reservations of its scope, and consumes nothing
	// when it fails.
	Reserve(ctx context.Context, r Reservation) error
	// RestoredAt returns the time since which the store holds every
	// reservation it accepted.
	RestoredAt() time.Time
}

// Capacity bounds a MemStore. Each value must be positive, and
// PerOwner <= PerScope <= Total. An entry costs about 200 bytes plus its
// strings; Go maps keep their peak size, so Total bounds memory for the
// life of the process.
type Capacity struct {
	PerOwner int // live entries of one owner in one scope
	PerScope int // live entries of one scope
	Total    int // live entries of the store
}

type windows struct {
	set          bool
	maxAge, skew time.Duration
}

type entryKey struct {
	scope     Scope
	id, nonce string
}

type ownerKey struct {
	scope Scope
	owner string
}

type item struct {
	until time.Time
	key   entryKey
	owner string
}

// MemStore is a Store in memory. It forgets everything when the process
// ends, so its RestoredAt is its creation time.
type MemStore struct {
	clock    clock.Clock
	cap      Capacity
	restored time.Time

	mu       sync.Mutex
	windows  [scopeEnd]windows // fixed by the first reservation of a scope
	entries  map[entryKey]time.Time
	perOwner map[ownerKey]int
	perScope [scopeEnd]int
	expiry   expiryHeap
}

// NewMemStore returns an empty store.
func NewMemStore(c clock.Clock, capacity Capacity) (*MemStore, error) {
	if c == nil || capacity.PerOwner <= 0 || capacity.PerOwner > capacity.PerScope || capacity.PerScope > capacity.Total {
		return nil, ErrInvalid
	}
	// A zero time would make RestoredAt zero and disable the restore rule
	// of E03-12, so clock.Read refuses it.
	now, err := clock.Read(c)
	if err != nil {
		return nil, err
	}
	return &MemStore{
		clock: c, cap: capacity, restored: now,
		entries: map[entryKey]time.Time{}, perOwner: map[ownerKey]int{},
	}, nil
}

// RestoredAt returns the creation time of the store.
func (s *MemStore) RestoredAt() time.Time { return s.restored }

// Len returns the number of live entries.
func (s *MemStore) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.entries)
}

// Reserve implements Store.
func (s *MemStore) Reserve(ctx context.Context, r Reservation) error {
	if s == nil || s.clock == nil || ctx == nil || !valid(r) {
		return ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	now, err := clock.Read(s.clock)
	if err != nil {
		return err
	}
	key := entryKey{scope: r.Scope, id: r.ID, nonce: r.Nonce}
	owner := ownerKey{scope: r.Scope, owner: r.Owner}

	s.mu.Lock()
	defer s.mu.Unlock()
	w := &s.windows[r.Scope]
	if w.set && (w.maxAge != r.MaxAge || w.skew != r.Skew) {
		return ErrWindows
	}
	until, restoreWindow := rules(r, now)
	if r.Created.Before(s.restored.Add(restoreWindow)) {
		return ErrRestored
	}
	// The receiver checked freshness before reserving, so until lies in
	// [now, now + 900 s]; anything else is a receiver bug.
	if until.Before(now) || until.After(now.Add(core.SpecMaxAge+2*core.SpecMaxClockSkew)) {
		return ErrInvalid
	}
	s.expire(now)
	if _, seen := s.entries[key]; seen {
		return ErrSeen
	}
	if s.perOwner[owner] >= s.cap.PerOwner || s.perScope[r.Scope] >= s.cap.PerScope || len(s.entries) >= s.cap.Total {
		return ErrFull
	}
	*w = windows{set: true, maxAge: r.MaxAge, skew: r.Skew} // only an accepted reservation fixes them
	s.entries[key] = until
	s.perOwner[owner]++
	s.perScope[r.Scope]++
	heap.Push(&s.expiry, item{until: until, key: key, owner: r.Owner})
	return nil
}

// rules returns how long r is kept and the restore window of its scope,
// from the receiver's windows in r.
func rules(r Reservation, now time.Time) (until time.Time, restoreWindow time.Duration) {
	switch r.Scope {
	case ScopeHTTPRequest: // E03-12
		return r.Created.Add(r.MaxAge + r.Skew), r.Skew
	case ScopeWireEnvelope: // E08-01
		return r.Created.Add(TransportWindow + r.Skew), r.Skew
	default: // ScopeHPKEInit, E04-14
		return now.Add(HPKEInitRetention), HPKEInitWindow
	}
}

// expire removes the entries whose until has passed. An entry is live up
// to and including its until (E03-12: "at least until"). Each entry is
// removed once, so the cost is O(log n) per entry over its lifetime.
func (s *MemStore) expire(now time.Time) {
	for s.expiry.Len() > 0 && now.After(s.expiry[0].until) {
		it := heap.Pop(&s.expiry).(item)
		delete(s.entries, it.key)
		owner := ownerKey{scope: it.key.scope, owner: it.owner}
		if n := s.perOwner[owner] - 1; n > 0 {
			s.perOwner[owner] = n
		} else {
			delete(s.perOwner, owner)
		}
		s.perScope[it.key.scope]--
	}
}

func valid(r Reservation) bool {
	return r.Scope > ScopeUnknown && r.Scope < scopeEnd && !r.Created.IsZero() &&
		r.Owner != "" && r.ID != "" && r.Nonce != "" &&
		len(r.Owner) <= MaxFieldBytes && len(r.ID) <= MaxFieldBytes && len(r.Nonce) <= MaxFieldBytes &&
		validWindows(r)
}

// validWindows checks the receiver's windows against the ranges of E03-11:
// a window the scope does not use must be zero, so a caller cannot believe
// it applies.
func validWindows(r Reservation) bool {
	skewOK := r.Skew >= 0 && r.Skew <= core.SpecMaxClockSkew
	switch r.Scope {
	case ScopeHTTPRequest:
		return r.MaxAge > 0 && r.MaxAge <= core.SpecMaxAge && skewOK
	case ScopeWireEnvelope:
		return r.MaxAge == 0 && skewOK
	default:
		return r.MaxAge == 0 && r.Skew == 0
	}
}

type expiryHeap []item

func (h expiryHeap) Len() int           { return len(h) }
func (h expiryHeap) Less(i, j int) bool { return h[i].until.Before(h[j].until) }
func (h expiryHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *expiryHeap) Push(x any)        { *h = append(*h, x.(item)) }
func (h *expiryHeap) Pop() any {
	old := *h
	it := old[len(old)-1]
	old[len(old)-1] = item{}
	*h = old[:len(old)-1]
	return it
}
