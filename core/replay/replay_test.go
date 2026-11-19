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

package replay

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/core/clock"
)

var start = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

const did = "did:sage:ethereum:0xabc"

func newStore(t testing.TB, c clock.Clock, capacity Capacity) *MemStore {
	t.Helper()
	s, err := NewMemStore(c, capacity, core.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// setup creates a store at start and moves the clock an hour on, past the
// restore window, as a store that has been running for a while.
func setup(t testing.TB, capacity Capacity, lim core.Limits) (*clock.Fixed, *MemStore) {
	t.Helper()
	c := clock.NewFixed(start)
	s, err := NewMemStore(c, capacity, lim)
	if err != nil {
		t.Fatal(err)
	}
	c.Set(start.Add(time.Hour))
	return c, s
}

func now(c *clock.Fixed) time.Time {
	t, _ := c.Now()
	return t
}

// httpRes is an RFC 9421 reservation created at the clock's time, so it is
// fresh.
func httpRes(c *clock.Fixed, owner, nonce string) Reservation {
	return Reservation{Scope: ScopeHTTPRequest, Owner: owner, ID: owner, Nonce: nonce, Created: now(c)}
}

func TestOneOfManyConcurrentReservationsWins(t *testing.T) {
	c, s := setup(t, Capacity{PerOwner: 8, PerScope: 64, Total: 128}, core.DefaultLimits())
	r := httpRes(c, did, "nonce-0123456789abcdef")
	var won atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			switch err := s.Reserve(context.Background(), r); {
			case err == nil:
				won.Add(1)
			case !errors.Is(err, ErrSeen):
				t.Errorf("unexpected %v", err)
			}
		}()
	}
	wg.Wait()
	if won.Load() != 1 {
		t.Fatalf("%d reservations succeeded, want exactly one", won.Load())
	}
}

func TestCapsNeverEvict(t *testing.T) {
	c, s := setup(t, Capacity{PerOwner: 2, PerScope: 3, Total: 4}, core.DefaultLimits())
	ctx := context.Background()
	must := func(r Reservation) {
		t.Helper()
		if err := s.Reserve(ctx, r); err != nil {
			t.Fatalf("%+v: %v", r, err)
		}
	}
	full := func(r Reservation, why string) {
		t.Helper()
		if err := s.Reserve(ctx, r); !errors.Is(err, ErrFull) {
			t.Fatalf("%s: %v", why, err)
		}
	}
	must(httpRes(c, "a", "1"))
	must(httpRes(c, "a", "2"))
	full(httpRes(c, "a", "3"), "owner cap")
	must(httpRes(c, "b", "1"))
	full(httpRes(c, "c", "1"), "scope cap")
	hpke := Reservation{Scope: ScopeHPKEInit, Owner: "a", ID: "ctx-1", Nonce: "n", Created: now(c)}
	must(hpke)
	full(Reservation{Scope: ScopeWireEnvelope, Owner: "d", ID: "d", Nonce: "1", Created: now(c)}, "total cap")
	for _, r := range []Reservation{httpRes(c, "a", "1"), httpRes(c, "a", "2"), httpRes(c, "b", "1"), hpke} {
		if err := s.Reserve(ctx, r); !errors.Is(err, ErrSeen) {
			t.Fatalf("%+v lost: %v", r, err)
		}
	}
	if s.Len() != 4 {
		t.Fatalf("len %d", s.Len())
	}
}

// TestHPKECapacityIsChargedToTheSigner: a signer that picks a new ctxID for
// every init still uses its own share only (L2, R15).
func TestHPKECapacityIsChargedToTheSigner(t *testing.T) {
	c, s := setup(t, Capacity{PerOwner: 2, PerScope: 100, Total: 100}, core.DefaultLimits())
	ctx := context.Background()
	init := func(owner, ctxID string) error {
		return s.Reserve(ctx, Reservation{Scope: ScopeHPKEInit, Owner: owner, ID: ctxID, Nonce: "00000000-0000-4000-8000-000000000000", Created: now(c)})
	}
	for i := 0; i < 2; i++ {
		if err := init("attacker", "ctx-"+string(rune('a'+i))); err != nil {
			t.Fatal(err)
		}
	}
	for i := 2; i < 50; i++ {
		if err := init("attacker", "ctx-"+string(rune('a'+i))); !errors.Is(err, ErrFull) {
			t.Fatalf("init %d: %v", i, err)
		}
	}
	if err := init("victim", "ctx-v"); err != nil {
		t.Fatalf("victim blocked: %v", err)
	}
}

func TestRetentionPerScope(t *testing.T) {
	ctx := context.Background()
	lim := core.DefaultLimits()
	lim.MaxAge = 60 * time.Second // lowered: applies to RFC 9421 only
	for _, c := range []struct {
		name  string
		scope Scope
		keep  time.Duration // from Created, which is the reservation time here
	}{
		{"rfc9421 created + MaxAge + skew (E03-12)", ScopeHTTPRequest, 60*time.Second + lim.MaxClockSkew},
		{"envelope created + 300 s + skew (E08-01)", ScopeWireEnvelope, TransportWindow + lim.MaxClockSkew},
		{"hpke 600 s (E04-14)", ScopeHPKEInit, HPKEInitRetention},
	} {
		t.Run(c.name, func(t *testing.T) {
			clk, s := setup(t, Capacity{PerOwner: 1, PerScope: 1, Total: 1}, lim)
			r := Reservation{Scope: c.scope, Owner: "a", ID: "a", Nonce: "1", Created: now(clk)}
			if err := s.Reserve(ctx, r); err != nil {
				t.Fatal(err)
			}
			// Live up to and including until.
			clk.Advance(c.keep)
			if err := s.Reserve(ctx, Reservation{Scope: c.scope, Owner: "a", ID: "a", Nonce: "1", Created: now(clk)}); !errors.Is(err, ErrSeen) {
				t.Fatalf("at until: %v", err)
			}
			clk.Advance(time.Nanosecond)
			if err := s.Reserve(ctx, Reservation{Scope: c.scope, Owner: "a", ID: "a", Nonce: "1", Created: now(clk)}); err != nil {
				t.Fatalf("after until the capacity and nonce are free: %v", err)
			}
		})
	}
}

func TestRestoredAt(t *testing.T) {
	ctx := context.Background()
	restore := start.Add(time.Hour)
	c := clock.NewFixed(restore)
	s := newStore(t, c, Capacity{PerOwner: 4, PerScope: 4, Total: 12})
	if !s.RestoredAt().Equal(restore) {
		t.Fatalf("restored at %v", s.RestoredAt())
	}
	skew := core.DefaultLimits().MaxClockSkew
	for _, c := range []struct {
		scope  Scope
		window time.Duration
	}{{ScopeHTTPRequest, skew}, {ScopeWireEnvelope, skew}, {ScopeHPKEInit, HPKEInitWindow}} {
		early := Reservation{Scope: c.scope, Owner: "a", ID: "a", Nonce: "early", Created: restore.Add(c.window - time.Nanosecond)}
		if err := s.Reserve(ctx, early); !errors.Is(err, ErrRestored) {
			t.Fatalf("scope %d: input dated before restore + window: %v", c.scope, err)
		}
		ok := Reservation{Scope: c.scope, Owner: "a", ID: "a", Nonce: "ok", Created: restore.Add(c.window)}
		if err := s.Reserve(ctx, ok); err != nil {
			t.Fatalf("scope %d: input dated restore + window: %v", c.scope, err)
		}
	}
}

// TestNewStoreRejectsUntilTheWindowPasses: a memory store cannot tell a
// first start from a restart, so it applies E03-12 from its creation:
// input signed in the first skew (120 s for HPKE) is rejected.
func TestNewStoreRejectsUntilTheWindowPasses(t *testing.T) {
	c := clock.NewFixed(start)
	s := newStore(t, c, Capacity{PerOwner: 2, PerScope: 2, Total: 2})
	if err := s.Reserve(context.Background(), httpRes(c, "a", "1")); !errors.Is(err, ErrRestored) {
		t.Fatalf("at creation: %v", err)
	}
	c.Advance(core.DefaultLimits().MaxClockSkew)
	if err := s.Reserve(context.Background(), httpRes(c, "a", "1")); err != nil {
		t.Fatalf("after the skew: %v", err)
	}
}

func TestUntilOutsideTheWindowIsACallerError(t *testing.T) {
	c, s := setup(t, Capacity{PerOwner: 1, PerScope: 1, Total: 1}, core.DefaultLimits())
	ctx := context.Background()
	stale := httpRes(c, "a", "1")
	stale.Created = now(c).Add(-20 * time.Minute) // after the restore window, but freshness was not checked
	future := httpRes(c, "a", "2")
	future.Created = now(c).Add(time.Hour)
	for _, r := range []Reservation{stale, future} {
		if err := s.Reserve(ctx, r); !errors.Is(err, ErrInvalid) {
			t.Errorf("%v: %v", r.Created, err)
		}
	}
}

func TestInvalidInputConsumesNothing(t *testing.T) {
	c, s := setup(t, Capacity{PerOwner: 1, PerScope: 1, Total: 1}, core.DefaultLimits())
	ctx := context.Background()
	good := httpRes(c, "a", "1")
	for name, mut := range map[string]func(*Reservation){
		"unknown scope": func(r *Reservation) { r.Scope = Scope(9) },
		"scope zero":    func(r *Reservation) { r.Scope = ScopeUnknown },
		"no owner":      func(r *Reservation) { r.Owner = "" },
		"no id":         func(r *Reservation) { r.ID = "" },
		"no nonce":      func(r *Reservation) { r.Nonce = "" },
		"no created":    func(r *Reservation) { r.Created = time.Time{} },
		"long nonce":    func(r *Reservation) { r.Nonce = strings.Repeat("n", MaxFieldBytes+1) },
	} {
		r := good
		mut(&r)
		if err := s.Reserve(ctx, r); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	var nilCtx context.Context
	if err := s.Reserve(nilCtx, good); !errors.Is(err, ErrInvalid) {
		t.Errorf("nil ctx: %v", err)
	}
	var zero *MemStore
	if err := zero.Reserve(ctx, good); !errors.Is(err, ErrInvalid) {
		t.Errorf("nil store: %v", err)
	}
	if err := (&MemStore{}).Reserve(ctx, good); !errors.Is(err, ErrInvalid) {
		t.Errorf("zero store: %v", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := s.Reserve(cancelled, good); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled: %v", err)
	}
	if err := s.Reserve(ctx, good); err != nil {
		t.Fatalf("no failed call may consume capacity: %v", err)
	}
}

func TestClockFailureFailsClosed(t *testing.T) {
	boom := errors.New("clock down")
	if _, err := NewMemStore(clock.Failing(boom), Capacity{PerOwner: 1, PerScope: 1, Total: 1}, core.DefaultLimits()); !errors.Is(err, boom) {
		t.Fatalf("construction: %v", err)
	}
	c := &switchable{t: start}
	s := newStore(t, c, Capacity{PerOwner: 1, PerScope: 1, Total: 1})
	c.t, c.err = start.Add(time.Hour), boom
	r := Reservation{Scope: ScopeHTTPRequest, Owner: "a", ID: "a", Nonce: "1", Created: c.t}
	if err := s.Reserve(context.Background(), r); !errors.Is(err, boom) {
		t.Fatalf("reserve: %v", err)
	}
}

type switchable struct {
	t   time.Time
	err error
}

func (s *switchable) Now() (time.Time, error) { return s.t, s.err }

func TestConstructionValidated(t *testing.T) {
	c := clock.NewFixed(start)
	for _, capacity := range []Capacity{{}, {PerOwner: 1, PerScope: 1}, {PerOwner: 2, PerScope: 1, Total: 1}, {PerOwner: 1, PerScope: 2, Total: 1}, {PerOwner: -1, PerScope: 1, Total: 1}} {
		if _, err := NewMemStore(c, capacity, core.DefaultLimits()); !errors.Is(err, ErrInvalid) {
			t.Errorf("%+v: %v", capacity, err)
		}
	}
	if _, err := NewMemStore(nil, Capacity{PerOwner: 1, PerScope: 1, Total: 1}, core.DefaultLimits()); !errors.Is(err, ErrInvalid) {
		t.Errorf("nil clock: %v", err)
	}
	if _, err := NewMemStore(c, Capacity{PerOwner: 1, PerScope: 1, Total: 1}, core.Limits{}); !errors.Is(err, core.ErrLimits) {
		t.Errorf("zero limits: %v", err)
	}
}

// TestLongRunningStoreStaysBounded: expired entries leave as time passes.
func TestLongRunningStoreStaysBounded(t *testing.T) {
	lim := core.DefaultLimits()
	c, s := setup(t, Capacity{PerOwner: 400, PerScope: 400, Total: 400}, lim)
	ctx := context.Background()
	for round := 0; round < 100; round++ {
		for i := 0; i < 1; i++ {
			if err := s.Reserve(ctx, httpRes(c, "a", time.Duration(round).String())); err != nil {
				t.Fatalf("round %d: %v", round, err)
			}
		}
		c.Advance(lim.MaxAge + lim.MaxClockSkew + time.Nanosecond)
	}
	if n := s.Len(); n > 1 {
		t.Fatalf("store holds %d entries", n)
	}
}

func TestConcurrentReserveAndClock(t *testing.T) {
	c, s := setup(t, Capacity{PerOwner: 64, PerScope: 1024, Total: 2048}, core.DefaultLimits())
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				owner := string(rune('a' + g))
				_ = s.Reserve(context.Background(), Reservation{Scope: Scope(1 + i%3), Owner: owner, ID: owner, Nonce: time.Duration(i).String(), Created: now(c)})
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			c.Advance(time.Second)
		}
	}()
	wg.Wait()
}

// BenchmarkRefusedWhenFull measures a refusal of a sender at its share
// while other entries keep expiring: it must not scan the store.
func BenchmarkRefusedWhenFull(b *testing.B) {
	c, s := setup(b, Capacity{PerOwner: 1, PerScope: 200000, Total: 200000}, core.DefaultLimits())
	ctx := context.Background()
	for i := 0; i < 100000; i++ {
		id := time.Duration(i).String()
		_ = s.Reserve(ctx, Reservation{Scope: ScopeHTTPRequest, Owner: id, ID: id, Nonce: "n", Created: now(c)})
	}
	_ = s.Reserve(ctx, httpRes(c, "attacker", "0"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Advance(time.Microsecond)
		_ = s.Reserve(ctx, httpRes(c, "attacker", time.Duration(i+1).String()))
	}
}
