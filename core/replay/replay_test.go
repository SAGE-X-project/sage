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
	s, err := NewMemStore(c, capacity)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// setup creates a store at start and moves the clock an hour on, past the
// restore window, as a store that has been running for a while.
func setup(t testing.TB, capacity Capacity) (*clock.Fixed, *MemStore) {
	t.Helper()
	c := clock.NewFixed(start)
	s, err := NewMemStore(c, capacity)
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
	return win(Reservation{Scope: ScopeHTTPRequest, Owner: owner, ID: owner, Nonce: nonce, Created: now(c)})
}

// win sets the receiver windows a scope uses to the defaults of
// core.DefaultLimits.
func win(r Reservation) Reservation {
	lim := core.DefaultLimits()
	switch r.Scope {
	case ScopeHTTPRequest:
		r.MaxAge, r.Skew = lim.MaxAge, lim.MaxClockSkew
	case ScopeWireEnvelope:
		r.Skew = lim.MaxClockSkew
	}
	return r
}

func TestOneOfManyConcurrentReservationsWins(t *testing.T) {
	c, s := setup(t, Capacity{PerOwner: 8, PerScope: 64, Total: 128})
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
	c, s := setup(t, Capacity{PerOwner: 2, PerScope: 3, Total: 4})
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
	hpke := win(Reservation{Scope: ScopeHPKEInit, Owner: "a", ID: "ctx-1", Nonce: "n", Created: now(c)})
	must(hpke)
	full(win(Reservation{Scope: ScopeWireEnvelope, Owner: "d", ID: "d", Nonce: "1", Created: now(c)}), "total cap")
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
	c, s := setup(t, Capacity{PerOwner: 2, PerScope: 100, Total: 100})
	ctx := context.Background()
	init := func(owner, ctxID string) error {
		return s.Reserve(ctx, win(Reservation{Scope: ScopeHPKEInit, Owner: owner, ID: ctxID, Nonce: "00000000-0000-4000-8000-000000000000", Created: now(c)}))
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
	lim.MaxAge = 60 * time.Second // lowered by the receiver: applies to RFC 9421 only
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
			clk, s := setup(t, Capacity{PerOwner: 1, PerScope: 1, Total: 1})
			lowered := func(r Reservation) Reservation {
				r = win(r)
				if r.Scope == ScopeHTTPRequest {
					r.MaxAge = lim.MaxAge
				}
				return r
			}
			r := lowered(Reservation{Scope: c.scope, Owner: "a", ID: "a", Nonce: "1", Created: now(clk)})
			if err := s.Reserve(ctx, r); err != nil {
				t.Fatal(err)
			}
			// Live up to and including until.
			clk.Advance(c.keep)
			if err := s.Reserve(ctx, lowered(Reservation{Scope: c.scope, Owner: "a", ID: "a", Nonce: "1", Created: now(clk)})); !errors.Is(err, ErrSeen) {
				t.Fatalf("at until: %v", err)
			}
			clk.Advance(time.Nanosecond)
			if err := s.Reserve(ctx, lowered(Reservation{Scope: c.scope, Owner: "a", ID: "a", Nonce: "1", Created: now(clk)})); err != nil {
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
		early := win(Reservation{Scope: c.scope, Owner: "a", ID: "a", Nonce: "early", Created: restore.Add(c.window - time.Nanosecond)})
		if err := s.Reserve(ctx, early); !errors.Is(err, ErrRestored) {
			t.Fatalf("scope %d: input dated before restore + window: %v", c.scope, err)
		}
		ok := win(Reservation{Scope: c.scope, Owner: "a", ID: "a", Nonce: "ok", Created: restore.Add(c.window)})
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
	c, s := setup(t, Capacity{PerOwner: 1, PerScope: 1, Total: 1})
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
	c, s := setup(t, Capacity{PerOwner: 1, PerScope: 1, Total: 1})
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
	if _, err := NewMemStore(clock.Failing(boom), Capacity{PerOwner: 1, PerScope: 1, Total: 1}); !errors.Is(err, boom) {
		t.Fatalf("construction: %v", err)
	}
	c := &switchable{t: start}
	s := newStore(t, c, Capacity{PerOwner: 1, PerScope: 1, Total: 1})
	c.t, c.err = start.Add(time.Hour), boom
	r := win(Reservation{Scope: ScopeHTTPRequest, Owner: "a", ID: "a", Nonce: "1", Created: c.t})
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
		if _, err := NewMemStore(c, capacity); !errors.Is(err, ErrInvalid) {
			t.Errorf("%+v: %v", capacity, err)
		}
	}
	if _, err := NewMemStore(nil, Capacity{PerOwner: 1, PerScope: 1, Total: 1}); !errors.Is(err, ErrInvalid) {
		t.Errorf("nil clock: %v", err)
	}
}

// TestLongRunningStoreStaysBounded: expired entries leave as time passes.
func TestLongRunningStoreStaysBounded(t *testing.T) {
	lim := core.DefaultLimits()
	c, s := setup(t, Capacity{PerOwner: 400, PerScope: 400, Total: 400})
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
	c, s := setup(t, Capacity{PerOwner: 64, PerScope: 1024, Total: 2048})
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				owner := string(rune('a' + g))
				_ = s.Reserve(context.Background(), win(Reservation{Scope: Scope(1 + i%3), Owner: owner, ID: owner, Nonce: time.Duration(i).String(), Created: now(c)}))
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
	c, s := setup(b, Capacity{PerOwner: 1, PerScope: 200000, Total: 200000})
	ctx := context.Background()
	for i := 0; i < 100000; i++ {
		id := time.Duration(i).String()
		_ = s.Reserve(ctx, win(Reservation{Scope: ScopeHTTPRequest, Owner: id, ID: id, Nonce: "n", Created: now(c)}))
	}
	_ = s.Reserve(ctx, httpRes(c, "attacker", "0"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Advance(time.Microsecond)
		_ = s.Reserve(ctx, httpRes(c, "attacker", time.Duration(i+1).String()))
	}
}

// TestWindowsComeFromTheReceiver: retention and the restore window follow
// the MaxAge and MaxClockSkew in the reservation, so a receiver with a
// larger skew is never served by a shorter store window (D31).
func TestWindowsComeFromTheReceiver(t *testing.T) {
	ctx := context.Background()
	restore := start.Add(time.Hour)
	c := clock.NewFixed(restore)
	s := newStore(t, c, Capacity{PerOwner: 8, PerScope: 8, Total: 16})
	wide := func(scope Scope, nonce string, created time.Time) Reservation {
		r := win(Reservation{Scope: scope, Owner: "a", ID: "a", Nonce: nonce, Created: created})
		r.Skew = core.SpecMaxClockSkew
		return r
	}
	for _, scope := range []Scope{ScopeHTTPRequest, ScopeWireEnvelope} {
		// 200 s after the restore is inside a 300 s skew, though outside the
		// default 60 s.
		if err := s.Reserve(ctx, wide(scope, "early", restore.Add(200*time.Second))); !errors.Is(err, ErrRestored) {
			t.Fatalf("scope %d: inside the receiver's skew after restore: %v", scope, err)
		}
	}
	c.Set(restore.Add(time.Hour))
	created := restore.Add(time.Hour)
	keep := map[Scope]time.Duration{
		ScopeHTTPRequest:  core.DefaultLimits().MaxAge + core.SpecMaxClockSkew,
		ScopeWireEnvelope: TransportWindow + core.SpecMaxClockSkew,
	}
	for scope, d := range keep {
		if err := s.Reserve(ctx, wide(scope, "n", created)); err != nil {
			t.Fatal(err)
		}
		c.Set(created.Add(d))
		if err := s.Reserve(ctx, wide(scope, "n", created)); !errors.Is(err, ErrSeen) {
			t.Fatalf("scope %d: forgotten before created + window + receiver skew: %v", scope, err)
		}
		c.Set(created)
	}
}

func TestWindowsValidated(t *testing.T) {
	c, s := setup(t, Capacity{PerOwner: 1, PerScope: 1, Total: 1})
	ctx := context.Background()
	for name, r := range map[string]Reservation{
		"http MaxAge zero":     {Scope: ScopeHTTPRequest, Skew: time.Second},
		"http MaxAge over 300": {Scope: ScopeHTTPRequest, MaxAge: core.SpecMaxAge + 1, Skew: time.Second},
		"http skew over 300":   {Scope: ScopeHTTPRequest, MaxAge: time.Second, Skew: core.SpecMaxClockSkew + 1},
		"http skew negative":   {Scope: ScopeHTTPRequest, MaxAge: time.Second, Skew: -1},
		"envelope with MaxAge": {Scope: ScopeWireEnvelope, MaxAge: time.Second},
		"envelope skew over":   {Scope: ScopeWireEnvelope, Skew: core.SpecMaxClockSkew + 1},
		"hpke with skew":       {Scope: ScopeHPKEInit, Skew: time.Second},
		"hpke with MaxAge":     {Scope: ScopeHPKEInit, MaxAge: time.Second},
	} {
		r.Owner, r.ID, r.Nonce, r.Created = "a", "a", "1", now(c)
		if err := s.Reserve(ctx, r); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if s.Len() != 0 {
		t.Fatalf("an invalid reservation was kept")
	}
}

// TestOneConfigurationPerScope: a store serves one receiver configuration.
// Two receivers with different windows on one store would let the longer
// one accept a pair the shorter one let expire, so the second is refused.
func TestOneConfigurationPerScope(t *testing.T) {
	c, s := setup(t, Capacity{PerOwner: 4, PerScope: 4, Total: 8})
	ctx := context.Background()
	short := httpRes(c, "a", "1")
	short.MaxAge, short.Skew = 60*time.Second, 0
	if err := s.Reserve(ctx, short); err != nil {
		t.Fatal(err)
	}
	long := short
	long.MaxAge, long.Nonce = 300*time.Second, "2"
	if err := s.Reserve(ctx, long); !errors.Is(err, ErrWindows) {
		t.Fatalf("other MaxAge: %v", err)
	}
	long.MaxAge, long.Skew = short.MaxAge, time.Second
	if err := s.Reserve(ctx, long); !errors.Is(err, ErrWindows) {
		t.Fatalf("other skew: %v", err)
	}
	// The envelope scope has its own configuration.
	if err := s.Reserve(ctx, win(Reservation{Scope: ScopeWireEnvelope, Owner: "a", ID: "a", Nonce: "1", Created: now(c)})); err != nil {
		t.Fatalf("envelope: %v", err)
	}
	// The replay the mismatch would allow: the entry expires after 60 s,
	// and a 300 s receiver would accept the same pair at +120 s.
	c.Advance(120 * time.Second)
	again := short
	again.MaxAge = 300 * time.Second
	if err := s.Reserve(ctx, again); !errors.Is(err, ErrWindows) {
		t.Fatalf("replay through a longer window: %v", err)
	}
	if s.Len() != 2 {
		t.Fatalf("len %d", s.Len())
	}
}

// TestFailedReservationFixesNoWindows: only an accepted reservation fixes
// the windows of its scope, so a failed one consumes nothing.
func TestFailedReservationFixesNoWindows(t *testing.T) {
	ctx := context.Background()
	other := func(r Reservation) Reservation { r.MaxAge, r.Skew = 300*time.Second, 300*time.Second; return r }
	for name, fail := range map[string]func(*clock.Fixed, *MemStore) error{
		"full": func(c *clock.Fixed, s *MemStore) error {
			if err := s.Reserve(ctx, win(Reservation{Scope: ScopeWireEnvelope, Owner: "e", ID: "e", Nonce: "1", Created: now(c)})); err != nil {
				t.Fatal(err)
			}
			return s.Reserve(ctx, httpRes(c, "a", "1"))
		},
		"invalid": func(c *clock.Fixed, s *MemStore) error {
			r := httpRes(c, "a", "1")
			r.Created = now(c).Add(-time.Hour)
			return s.Reserve(ctx, r)
		},
		"restored": func(c *clock.Fixed, s *MemStore) error {
			r := httpRes(c, "a", "1")
			r.Created = s.RestoredAt()
			return s.Reserve(ctx, r)
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, s := setup(t, Capacity{PerOwner: 1, PerScope: 1, Total: 1})
			if err := fail(c, s); err == nil {
				t.Fatal("the reservation did not fail")
			}
			c.Advance(time.Hour) // every entry has expired
			if err := s.Reserve(ctx, other(httpRes(c, "b", "2"))); err != nil {
				t.Fatalf("windows fixed by a failed reservation: %v", err)
			}
		})
	}
}

// TestZeroClock: a clock that returns the zero time makes no store, since a
// zero RestoredAt would disable the restore rule (E03-12), and a store whose
// clock later returns it refuses reservations.
func TestZeroClock(t *testing.T) {
	capacity := Capacity{PerOwner: 1, PerScope: 1, Total: 1}
	if _, err := NewMemStore(clock.NewFixed(time.Time{}), capacity); !errors.Is(err, clock.ErrNoTime) {
		t.Fatal(err)
	}
	c := clock.NewFixed(time.Unix(1788609600, 0))
	s, err := NewMemStore(c, capacity)
	if err != nil {
		t.Fatal(err)
	}
	c.Set(time.Time{})
	r := httpRes(clock.NewFixed(time.Unix(1788609600, 0)), "did:sage:ethereum:0x1", "n-0123456789abcdef")
	if err := s.Reserve(context.Background(), r); !errors.Is(err, clock.ErrNoTime) {
		t.Fatal(err)
	}
}
