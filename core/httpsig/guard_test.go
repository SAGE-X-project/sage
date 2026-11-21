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
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/core/clock"
	"github.com/sage-x-project/sage/core/crypto"
	"github.com/sage-x-project/sage/core/obs"
	"github.com/sage-x-project/sage/core/replay"
)

// t0 is the time the test signatures are created at.
var t0 = time.Unix(1788609600, 0)

const testNonce = "n-0123456789abcdef"

// newGuard returns a guard whose store was restored long before t0, so the
// restore rule does not apply, and whose clock is at t0.
func newGuard(t *testing.T, lim core.Limits, capacity replay.Capacity) (*Guard, *clock.Fixed) {
	t.Helper()
	c := clock.NewFixed(t0.Add(-time.Hour))
	store, err := replay.NewMemStore(c, capacity)
	if err != nil {
		t.Fatal(err)
	}
	c.Set(t0)
	g, err := NewGuard(c, lim, store)
	if err != nil {
		t.Fatal(err)
	}
	return g, c
}

// reserve reserves p as if its signature had verified; the tests of
// TestFailedStepKeepsTheNonce and the suite go through VerifySignature.
func reserve(g *Guard, ctx context.Context, p Parsed) error { return g.Reserve(ctx, Signed{p: p}) }

func roomy() replay.Capacity { return replay.Capacity{PerOwner: 16, PerScope: 64, Total: 64} }

// timed parses a request signature with the given trailing parameters.
func timed(t *testing.T, keyid, params string) Parsed {
	t.Helper()
	return parsedOf(t, `sig1=("@method" "@target-uri" "@authority");keyid="`+keyid+`"`+params)
}

func at(d time.Duration) string { return strconv.FormatInt(t0.Add(d).Unix(), 10) }

func TestNewGuard(t *testing.T) {
	c := clock.NewFixed(t0)
	store, err := replay.NewMemStore(c, roomy())
	if err != nil {
		t.Fatal(err)
	}
	lim := core.DefaultLimits()
	if _, err := NewGuard(nil, lim, store); !errors.Is(err, ErrGuard) {
		t.Errorf("nil clock: %v", err)
	}
	if _, err := NewGuard(c, lim, nil); !errors.Is(err, ErrGuard) {
		t.Errorf("nil store: %v", err)
	}
	for name, mut := range map[string]func(*core.Limits){
		"MaxAge 0":        func(l *core.Limits) { l.MaxAge = 0 },
		"MaxAge > 300 s":  func(l *core.Limits) { l.MaxAge = core.SpecMaxAge + time.Second },
		"skew > 300 s":    func(l *core.Limits) { l.MaxClockSkew = core.SpecMaxClockSkew + time.Second },
		"negative skew":   func(l *core.Limits) { l.MaxClockSkew = -time.Second },
		"zero limits":     func(l *core.Limits) { *l = core.Limits{} },
		"MaxAge negative": func(l *core.Limits) { l.MaxAge = -time.Second },
	} {
		l := lim
		mut(&l)
		_, err := NewGuard(c, l, store)
		if !errors.Is(err, ErrGuard) || !errors.Is(err, core.ErrLimits) {
			t.Errorf("%s: %v", name, err)
		}
	}
	var nilGuard *Guard
	if err := nilGuard.CheckTime(Parsed{}); !errors.Is(err, ErrGuard) {
		t.Errorf("nil guard CheckTime: %v", err)
	}
	if err := nilGuard.Reserve(context.Background(), Signed{}); !errors.Is(err, ErrGuard) {
		t.Errorf("nil guard Reserve: %v", err)
	}
	var zero Guard
	if err := zero.CheckTime(Parsed{}); !errors.Is(err, ErrGuard) {
		t.Errorf("zero guard CheckTime: %v", err)
	}
	if err := zero.Reserve(context.Background(), Signed{}); !errors.Is(err, ErrGuard) {
		t.Errorf("zero guard Reserve: %v", err)
	}
}

func TestCheckTime(t *testing.T) {
	lim := core.DefaultLimits() // MaxAge 300 s, skew 60 s
	g, c := newGuard(t, lim, roomy())
	created := ";created=" + at(0) + `;nonce="` + testNonce + `"`
	for _, tc := range []struct {
		name   string
		params string
		now    time.Duration
		want   obs.Reason // 0: accepted
	}{
		{"now = created", created, 0, 0},
		{"now = created + MaxAge", created, lim.MaxAge, 0},
		{"too-old: created + 301 s", created, 301 * time.Second, obs.ReasonStale},
		{"just older than MaxAge", created, lim.MaxAge + time.Nanosecond, obs.ReasonStale},
		{"created = now + skew", created, -lim.MaxClockSkew, 0},
		{"created beyond the skew", created, -lim.MaxClockSkew - time.Second, obs.ReasonFuture},
		{"too-far-future: created = now + 301 s", created, -301 * time.Second, obs.ReasonFuture},
		{"expires after now", ";created=" + at(0) + ";expires=" + at(10*time.Second), 9 * time.Second, 0},
		{"expired: expires = now - 1", ";created=" + at(0) + ";expires=" + at(10*time.Second), 11 * time.Second, obs.ReasonExpired},
		{"expires = now", ";created=" + at(0) + ";expires=" + at(10*time.Second), 10 * time.Second, obs.ReasonExpired},
		{"expires-before-created: expires = created", ";created=" + at(0) + ";expires=" + at(0), 0, obs.ReasonExpired},
		{"expires before created", ";created=" + at(0) + ";expires=" + at(-time.Second), 0, obs.ReasonExpired},
		{"no created", `;nonce="` + testNonce + `"`, 0, obs.ReasonParameters},
		{"created at the 15-digit maximum", ";created=999999999999999", 0, obs.ReasonFuture},
		{"created zero", ";created=0", 0, obs.ReasonStale},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c.Set(t0.Add(tc.now))
			err := g.CheckTime(timed(t, vecDID, tc.params))
			if tc.want == 0 {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			wantReason(t, err, tc.want)
		})
	}
}

func TestClockFailureRejects(t *testing.T) {
	store, err := replay.NewMemStore(clock.NewFixed(t0), roomy())
	if err != nil {
		t.Fatal(err)
	}
	g, err := NewGuard(clock.Failing(errors.New("down")), core.DefaultLimits(), store)
	if err != nil {
		t.Fatal(err)
	}
	p := timed(t, vecDID, ";created="+at(0)+`;nonce="`+testNonce+`"`)
	wantReason(t, g.CheckTime(p), obs.ReasonClock)
	wantReason(t, reserve(g, context.Background(), p), obs.ReasonClock)
}

func TestReserve(t *testing.T) {
	g, _ := newGuard(t, core.DefaultLimits(), roomy())
	ctx := context.Background()
	params := func(nonce string) string { return ";created=" + at(0) + `;nonce="` + nonce + `"` }
	if err := reserve(g, ctx, timed(t, vecDID+"#key-1", params(testNonce))); err != nil {
		t.Fatal(err)
	}
	// replay-other-fragment: the same signer DID without the fragment, and
	// with another fragment, is the same namespace (E03-12).
	wantReason(t, reserve(g, ctx, timed(t, vecDID, params(testNonce))), obs.ReasonReplay)
	wantReason(t, reserve(g, ctx, timed(t, vecDID+"#key-2", params(testNonce))), obs.ReasonReplay)
	// Another signer may use the same nonce.
	other := "did:sage:ethereum:0x2222222222222222222222222222222222222222"
	if err := reserve(g, ctx, timed(t, other, params(testNonce))); err != nil {
		t.Fatalf("other signer: %v", err)
	}
	for name, nonce := range map[string]string{
		"15 characters": strings.Repeat("a", 15),
		"with +":        "n-0123456789abc+ef",
		"65 characters": strings.Repeat("a", 65),
		"with a space":  "n-0123456789 abcdef",
	} {
		t.Run(name, func(t *testing.T) {
			wantReason(t, reserve(g, ctx, timed(t, vecDID, params(nonce))), obs.ReasonParameters)
		})
	}
	for _, nonce := range []string{strings.Repeat("a", 16), strings.Repeat("Z", 64), "8f0c5b1c-0d1a-4c8e-9a51-6d2e4b7f1a10"} {
		if err := reserve(g, ctx, timed(t, vecDID, params(nonce))); err != nil {
			t.Errorf("%s: %v", nonce, err)
		}
	}
	wantReason(t, reserve(g, ctx, timed(t, vecDID, ";created="+at(0))), obs.ReasonParameters)
	wantReason(t, reserve(g, ctx, timed(t, "k", params("n-aaaaaaaaaaaaaaaa"))), obs.ReasonIdentifier)
	// A stale signature is not reserved.
	wantReason(t, reserve(g, ctx, timed(t, vecDID, ";created="+at(-time.Hour)+`;nonce="n-bbbbbbbbbbbbbbbb"`)), obs.ReasonStale)
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	err := reserve(g, cancelled, timed(t, vecDID, params("n-cccccccccccccccc")))
	var rej *core.Rejection
	if !errors.Is(err, context.Canceled) || errors.As(err, &rej) {
		t.Fatalf("cancelled: %v", err)
	}
}

func TestReserveFullAndRestored(t *testing.T) {
	ctx := context.Background()
	g, _ := newGuard(t, core.DefaultLimits(), replay.Capacity{PerOwner: 1, PerScope: 1, Total: 1})
	p := func(nonce string) Parsed { return timed(t, vecDID, ";created="+at(0)+`;nonce="`+nonce+`"`) }
	if err := reserve(g, ctx, p("n-aaaaaaaaaaaaaaaa")); err != nil {
		t.Fatal(err)
	}
	wantReason(t, reserve(g, ctx, p("n-bbbbbbbbbbbbbbbb")), obs.ReasonReplayFull)

	// A store restored at t0 rejects signatures created before t0 + skew
	// (E03-12), whichever skew the guard uses.
	lim := core.DefaultLimits()
	lim.MaxClockSkew = core.SpecMaxClockSkew
	c := clock.NewFixed(t0)
	store, err := replay.NewMemStore(c, replay.Capacity{PerOwner: 1, PerScope: 1, Total: 1})
	if err != nil {
		t.Fatal(err)
	}
	restored, err := NewGuard(c, lim, store)
	if err != nil {
		t.Fatal(err)
	}
	c.Set(t0.Add(250 * time.Second))
	wantReason(t, reserve(restored, ctx, timed(t, vecDID, ";created="+at(200*time.Second)+`;nonce="`+testNonce+`"`)), obs.ReasonReplay)
	if err := reserve(restored, ctx, timed(t, vecDID, ";created="+at(300*time.Second)+`;nonce="`+testNonce+`"`)); err != nil {
		t.Fatalf("created at restore + skew: %v", err)
	}
}

// TestFutureSignatureReplayedLater: a signature dated +5 min is accepted at
// once under a 300 s skew and must still be refused when it is used again
// at +9 min, when it is still fresh (E03-12: retained until created +
// MaxAge + skew, not for MaxAge from first sight).
func TestFutureSignatureReplayedLater(t *testing.T) {
	lim := core.DefaultLimits()
	lim.MaxClockSkew = core.SpecMaxClockSkew
	g, c := newGuard(t, lim, roomy())
	ctx := context.Background()
	p := timed(t, vecDID, ";created="+at(5*time.Minute)+`;nonce="`+testNonce+`"`)
	if err := reserve(g, ctx, p); err != nil {
		t.Fatal(err)
	}
	c.Set(t0.Add(9 * time.Minute))
	if err := g.CheckTime(p); err != nil {
		t.Fatalf("still fresh at +9 min: %v", err)
	}
	wantReason(t, reserve(g, ctx, p), obs.ReasonReplay)
}

// verifyAt runs stages S1 to S4, S6 and S8 (03 §5 steps 1, 2, 5, 6 and 8)
// on r; steps 3, 4 and 7 come with W-014.
func verifyAt(g *Guard, r Request, key crypto.PublicKey) error {
	p, err := Parse(r.Header, "", core.DefaultLimits())
	if err != nil {
		return err
	}
	base, err := RequestBase(p, r)
	if err != nil {
		return err
	}
	if err := g.CheckTime(p); err != nil {
		return err
	}
	signed, err := verifySignature(base, key)
	if err != nil {
		return err
	}
	return g.Reserve(context.Background(), signed)
}

// TestFailedStepKeepsTheNonce: a request that fails the signature or the
// timing does not consume its nonce, so the genuine request with the same
// nonce is accepted afterwards (03 §5 step 8). Reserve takes a Signed, so
// a request whose signature fails cannot reach it; the step order itself
// is verifyAt's here and becomes the library's with Verify* in W-014, with
// the Content-Digest case digest-mismatch-keeps-nonce (D30).
func TestFailedStepKeepsTheNonce(t *testing.T) {
	s, cs := testSigner(t, 1, vecDID+"#key-1")
	g, c := newGuard(t, core.DefaultLimits(), roomy())
	r := signed(t, s, testRequest(), Params{Created: t0.Unix(), Nonce: testNonce})

	forged := r
	forged.Header = r.Header.Clone()
	forged.Method = "PUT" // covered by @method: the signature fails
	wantReason(t, verifyAt(g, forged, cs.Public()), obs.ReasonSignature)

	c.Set(t0.Add(time.Hour)) // timing fails
	wantReason(t, verifyAt(g, r, cs.Public()), obs.ReasonStale)

	c.Set(t0)
	if err := verifyAt(g, r, cs.Public()); err != nil {
		t.Fatalf("genuine request after the failures: %v", err)
	}
	wantReason(t, verifyAt(g, r, cs.Public()), obs.ReasonReplay)
}

func TestCheckNonceSyntax(t *testing.T) {
	r := Request{Method: "GET", Target: testTarget}
	for _, nonce := range []string{strings.Repeat("a", 15), "n-0123456789abc+ef"} {
		p := timed(t, vecDID, ";created=1;nonce=\""+nonce+"\"")
		wantReason(t, CheckRequest(p, r), obs.ReasonParameters)
	}
	req := Request{Method: "GET", Target: testTarget, Header: map[string][]string{"Signature": {"sig1=:AAAA:"}}}
	resp := parsedOf(t, `sig1=("@status" "@method";req "@target-uri";req "@authority";req "signature";req);keyid="k";created=1;nonce="short"`)
	wantReason(t, CheckResponse(resp, Response{Status: 200}, req), obs.ReasonParameters)
}

// TestReserveResponse: a response signature is not reserved, with or
// without a nonce; its replay protection is the binding to the signed
// request (00-overview.md §7.4 DP-4).
func TestReserveResponse(t *testing.T) {
	g, _ := newGuard(t, core.DefaultLimits(), roomy())
	ctx := context.Background()
	withNonce := Signed{p: timed(t, vecDID, ";created="+at(0)+`;nonce="`+testNonce+`"`), response: true}
	without := Signed{p: timed(t, vecDID, ";created="+at(0)), response: true}
	for i := 0; i < 2; i++ {
		for _, s := range []Signed{withNonce, without} {
			if err := g.Reserve(ctx, s); err != nil {
				t.Fatalf("response: %v", err)
			}
		}
	}
	// The response did not take the request's (signer DID, nonce) pair.
	if err := reserve(g, ctx, withNonce.p); err != nil {
		t.Fatalf("request with the nonce of a response: %v", err)
	}
	// A request without a nonce never reaches step 8 (step 2 requires it);
	// if it did, Reserve refuses it.
	wantReason(t, g.Reserve(ctx, Signed{p: without.p}), obs.ReasonParameters)
}

// TestGuardsWithOtherLimitsOnOneStore: two guards with different MaxAge on
// one store would let the longer one accept a pair the shorter one let
// expire; the store refuses the second configuration (D31).
func TestGuardsWithOtherLimitsOnOneStore(t *testing.T) {
	c := clock.NewFixed(t0.Add(-time.Hour))
	store, err := replay.NewMemStore(c, roomy())
	if err != nil {
		t.Fatal(err)
	}
	c.Set(t0)
	short, long := core.DefaultLimits(), core.DefaultLimits()
	short.MaxAge, short.MaxClockSkew, long.MaxClockSkew = 60*time.Second, 0, 0
	a, err := NewGuard(c, short, store)
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewGuard(c, long, store)
	if err != nil {
		t.Fatal(err)
	}
	p := timed(t, vecDID, ";created="+at(0)+`;nonce="`+testNonce+`"`)
	if err := reserve(a, context.Background(), p); err != nil {
		t.Fatal(err)
	}
	c.Set(t0.Add(120 * time.Second))
	err = reserve(b, context.Background(), p)
	var rej *core.Rejection
	if !errors.Is(err, replay.ErrWindows) || errors.As(err, &rej) {
		t.Fatalf("replay through a longer window: %v", err)
	}
}

// TestBoundaryStaleIsARejection: with skew 0 at now = created + MaxAge the
// store, reading its clock later, sees until < now; Reserve reports that
// as stale, not as a raw store error.
func TestBoundaryStaleIsARejection(t *testing.T) {
	lim := core.DefaultLimits()
	lim.MaxClockSkew = 0
	sc := &steppingClock{t: t0.Add(-time.Hour)}
	store, err := replay.NewMemStore(sc, roomy())
	if err != nil {
		t.Fatal(err)
	}
	sc.t = t0.Add(lim.MaxAge - time.Nanosecond) // the guard reads created + MaxAge, the store 1 ns later
	g, err := NewGuard(sc, lim, store)
	if err != nil {
		t.Fatal(err)
	}
	wantReason(t, reserve(g, context.Background(), timed(t, vecDID, ";created="+at(0)+`;nonce="`+testNonce+`"`)), obs.ReasonStale)
}

// steppingClock advances by one nanosecond on every read.
type steppingClock struct{ t time.Time }

func (c *steppingClock) Now() (time.Time, error) {
	c.t = c.t.Add(time.Nanosecond)
	return c.t, nil
}

// TestZeroClockRejects: a clock that returns the zero time rejects with
// reason clock at both steps, not as a future signature.
func TestZeroClockRejects(t *testing.T) {
	g, c := newGuard(t, core.DefaultLimits(), roomy())
	p := timed(t, vecDID+"#key-1", `;created=`+at(0)+`;nonce="n-0123456789abcdef"`)
	c.Set(time.Time{})
	wantReason(t, g.CheckTime(p), obs.ReasonClock)
	wantReason(t, reserve(g, context.Background(), p), obs.ReasonClock)
}
