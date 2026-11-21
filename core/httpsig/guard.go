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
	"time"

	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/core/clock"
	"github.com/sage-x-project/sage/core/did"
	"github.com/sage-x-project/sage/core/obs"
	"github.com/sage-x-project/sage/core/replay"
)

// ErrGuard is returned by NewGuard for a missing clock or replay store or
// for limits outside the spec ranges, and by the methods of a nil Guard.
var ErrGuard = errors.New("httpsig: invalid timing and replay configuration")

// Guard applies the Timing and Replay rules of 03 §5 (steps 5 and 8) with
// one clock, one replay store and one Limits value. Because both rules
// read the same MaxAge and MaxClockSkew, the store keeps each pair at
// least as long as the timing check accepts its signature (E03-11,
// E03-12). Neither check can be disabled: MaxAge is in (0, 300 s] and
// MaxClockSkew in [0, 300 s], or NewGuard fails.
type Guard struct {
	clock clock.Clock
	lim   core.Limits
	store replay.Store
}

// NewGuard returns a Guard. It refuses a nil clock or store and limits
// that core.Limits.Validate rejects, among them MaxAge = 0 and
// MaxAge > 300 s. The store should read the same clock. A store serves one
// configuration (replay.ErrWindows): give every Guard that shares a store
// the same limits, and a new store when they change. A new store rejects
// signatures created before its restore time plus the new MaxClockSkew
// (E03-12); when the skew is lowered, the old skew is longer, so the new
// store must not take traffic until the old MaxClockSkew has passed since
// its creation.
func NewGuard(c clock.Clock, lim core.Limits, store replay.Store) (*Guard, error) {
	if c == nil || store == nil {
		return nil, ErrGuard
	}
	if err := lim.Validate(); err != nil {
		return nil, errors.Join(ErrGuard, err)
	}
	return &Guard{clock: c, lim: lim, store: store}, nil
}

// CheckTime is step 5 of 03 §5 (Timing) for a request or a response
// signature. It rejects a signature without created, and rejects when
// now - created > MaxAge (stale), when created - now > MaxClockSkew
// (future), when expires <= created, or when now >= expires (expired). A
// clock that fails rejects.
func (g *Guard) CheckTime(p Parsed) error {
	if !g.ok() {
		return ErrGuard
	}
	now, err := clock.Read(g.clock)
	if err != nil {
		return reject(obs.ReasonClock, "the clock failed")
	}
	return checkTime(p, now, g.lim)
}

func (g *Guard) ok() bool { return g != nil && g.clock != nil && g.store != nil }

func checkTime(p Parsed, now time.Time, lim core.Limits) error {
	created, ok := p.Created()
	if !ok {
		return reject(obs.ReasonParameters, "signature lacks created")
	}
	// Parse admits only non-negative integers of at most 15 digits, and
	// Sub saturates, so none of these comparisons overflows.
	signed := time.Unix(created, 0)
	if now.Sub(signed) > lim.MaxAge {
		return reject(obs.ReasonStale, "created is older than MaxAge")
	}
	if signed.Sub(now) > lim.MaxClockSkew {
		return reject(obs.ReasonFuture, "created is further in the future than MaxClockSkew")
	}
	if expires, ok := p.Expires(); ok {
		if expires <= created {
			return reject(obs.ReasonExpired, "expires is not after created")
		}
		if !now.Before(time.Unix(expires, 0)) {
			return reject(obs.ReasonExpired, "expires has passed")
		}
	}
	return nil
}

// Reserve is step 8 of 03 §5 (Replay): it checks and reserves (signer
// DID, nonce) in the replay store, where the signer DID is the canonical
// DID of keyid, so keyid with and without a fragment share one namespace
// (E03-12). It takes a Signed, so it runs only for a signature that
// verified; it must still be the last step: call it only after every other
// step, Content-Digest included, has passed, so that a message that fails
// an earlier step keeps its nonce. A request signature always carries a
// nonce. A response signature is not reserved: its replay protection is
// the binding to the signed request by "signature";req (00-overview.md
// §7.4 DP-4), so Reserve returns nil for it.
//
// Reserve checks the timing again at the time of the reservation, so it
// never reserves a pair the timing check would reject now. A nonce seen
// before, or a signature dated before the store was restored plus
// MaxClockSkew, is rejected with ReasonReplay; a full store with
// ReasonReplayFull. Any other error, such as a cancelled context or
// replay.ErrWindows for a store that serves other limits, is returned as it
// is and must also be treated as a failure.
func (g *Guard) Reserve(ctx context.Context, s Signed) error {
	if !g.ok() {
		return ErrGuard
	}
	if s.response {
		return nil
	}
	p := s.p
	keyid, okKey := p.KeyID()
	nonce, okNonce := p.Nonce()
	created, okCreated := p.Created()
	if !okKey || !okNonce || !okCreated {
		return reject(obs.ReasonParameters, "signature lacks keyid, nonce or created")
	}
	if !validNonce(nonce) {
		return reject(obs.ReasonParameters, "nonce is not 16 to 64 characters of ALPHA, DIGIT, - and _")
	}
	ref, err := did.ParseKeyRef(keyid)
	if err != nil {
		return reject(obs.ReasonIdentifier, "keyid is not a did:sage key reference")
	}
	now, err := clock.Read(g.clock)
	if err != nil {
		return reject(obs.ReasonClock, "the clock failed")
	}
	if err := checkTime(p, now, g.lim); err != nil {
		return err
	}
	signer := ref.DID().String()
	err = g.store.Reserve(ctx, replay.Reservation{
		Scope: replay.ScopeHTTPRequest, Owner: signer, ID: signer, Nonce: nonce,
		Created: time.Unix(created, 0), MaxAge: g.lim.MaxAge, Skew: g.lim.MaxClockSkew,
	})
	switch {
	case err == nil:
		return nil
	case errors.Is(err, replay.ErrSeen):
		return reject(obs.ReasonReplay, "nonce already used by this signer")
	case errors.Is(err, replay.ErrRestored):
		return reject(obs.ReasonReplay, "signature predates the replay store")
	case errors.Is(err, replay.ErrFull):
		return reject(obs.ReasonReplayFull, "replay store full")
	case errors.Is(err, replay.ErrInvalid):
		// The store reads its clock after ours: at the MaxAge boundary the
		// signature may have become stale in between.
		if now, cerr := clock.Read(g.clock); cerr == nil {
			if terr := checkTime(p, now, g.lim); terr != nil {
				return terr
			}
		}
	}
	return err
}
