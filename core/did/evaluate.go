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

package did

import (
	"context"
	"errors"
	"strconv"

	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/core/clock"
	"github.com/sage-x-project/sage/core/crypto"
	"github.com/sage-x-project/sage/core/obs"
)

// Record member limits of E06-14 (06-did-sage.md §3). They are not
// configurable.
const (
	MaxNameBytes         = 256
	MaxDescriptionBytes  = 4 << 10
	MaxEndpointBytes     = 2 << 10
	MaxKeys              = 32
	MaxCapabilitiesBytes = 16 << 10
)

// ErrPolicy is returned by Evaluate for a Policy without a clock or with a
// Settings value not made by core.NewSettings.
var ErrPolicy = errors.New("did: invalid evaluation policy")

// Policy holds what Evaluate needs besides the record: the clock that dates
// the observation, and the settings whose MaxResolutionAge bounds it and
// whose tracer receives diagnostics.
type Policy struct {
	Clock    clock.Clock
	Settings core.Settings
}

// TrustedKey is a signing key that Evaluate selected under 06-did-sage.md
// §3.1: listed in the record and carrying a §4 proof of possession that the
// core verified itself. Only Evaluate makes one; the zero value holds no
// key.
type TrustedKey struct {
	pub crypto.PublicKey
	typ KeyType
}

// PublicKey returns the key, or nil for the zero value.
func (k TrustedKey) PublicKey() crypto.PublicKey { return k.pub }

// Type returns the registry type of the entry the key came from.
func (k TrustedKey) Type() KeyType { return k.typ }

// IsZero reports whether k holds no key.
func (k TrustedKey) IsZero() bool { return k.pub == nil }

// Agent is an agent whose registry record Evaluate accepted: active, read
// recently enough, within the E06-14 limits, and with a signing key chosen by
// §3.1. Only Evaluate makes one; the zero value holds no agent.
type Agent struct {
	did     DID
	signing TrustedKey
}

// DID returns the record's DID as stored.
func (a Agent) DID() DID { return a.did }

// SigningKey returns the signing key of §3.1.
func (a Agent) SigningKey() TrustedKey { return a.signing }

// IsZero reports whether a holds no agent.
func (a Agent) IsZero() bool { return a.signing.IsZero() }

// Evaluate decides what the record resolved for the DID want may be used
// for (06-did-sage.md §3, §3.1, §4). It is the only constructor of Agent and
// TrustedKey: the record is the registry's claim, and nothing in it is
// trusted until Evaluate accepts it. In order it rejects:
//
//   - a record whose DID is not want in canonical form (06 §2), or a zero
//     want (resolution), so a resolver that returns another agent's record
//     cannot lend that agent's key to want;
//   - an inactive agent (inactive), which has no usable keys, however old
//     the observation, since a negative result may be cached (E06-12);
//   - a record over the E06-14 limits (resolution);
//   - an observation older than MaxResolutionAge (the zero time included)
//     or dated after the clock's now (resolution_stale); a failing clock or
//     one that returns the zero time gives clock;
//   - a record without an eligible signing key (key_not_eligible).
//
// The signing key is the first eligible type-0 entry in registry order, else
// the first eligible type-1 entry. An entry is eligible when its signature
// is a valid §4 proof for the record's DID as stored and its key_data
// (crypto.VerifyPoP); an entry whose proof does not verify is skipped. Each
// entry Evaluate reaches in that order is decoded first: when its key_data
// does not decode, resolution fails and no other key is used (§3.1 "does not
// fall back to another key", fixture select-malformed-fails). The registry's
// verified flag is not consulted (E06-08, E06-09); a listed key whose flag
// is false is reported to the tracer as a diagnostic, because with a fixed
// registry it signals tampering or an old contract. No key is synthesised
// from other members of the record.
//
// The KEM key (§3.1, E04-02) is not selected here.
func Evaluate(ctx context.Context, want DID, rec Record, o Observation, p Policy) (Agent, error) {
	if p.Clock == nil || p.Settings.Validate() != nil {
		return Agent{}, ErrPolicy
	}
	if !rec.DID.Equal(want) { // the zero DID equals nothing
		return Agent{}, core.Reject(obs.ReasonResolution, obs.Known("record is not the record of the resolved DID"))
	}
	if !rec.IsActive {
		return Agent{}, core.Reject(obs.ReasonInactive, obs.Known("agent is not active"))
	}
	if err := checkLimits(rec); err != nil {
		return Agent{}, err
	}
	if err := checkFresh(o, p); err != nil {
		return Agent{}, err
	}
	diagnoseFlags(ctx, rec, p.Settings)
	key, err := selectSigningKey(rec)
	if err != nil {
		return Agent{}, err
	}
	return Agent{did: rec.DID, signing: key}, nil
}

func checkLimits(rec Record) error {
	switch {
	case len(rec.Name) > MaxNameBytes,
		len(rec.Description) > MaxDescriptionBytes,
		len(rec.Endpoint) > MaxEndpointBytes,
		len(rec.Keys) > MaxKeys,
		len(rec.Capabilities) > MaxCapabilitiesBytes:
		return core.Reject(obs.ReasonResolution, obs.Known("record member over its limit"))
	}
	return nil
}

func checkFresh(o Observation, p Policy) error {
	now, err := clock.Read(p.Clock)
	if err != nil {
		return core.Reject(obs.ReasonClock, obs.Known("the clock failed"))
	}
	switch {
	case o.At.After(now):
		return core.Reject(obs.ReasonResolutionStale, obs.Known("observation is dated in the future"))
	case now.Sub(o.At) > p.Settings.Limits().MaxResolutionAge:
		return core.Reject(obs.ReasonResolutionStale, obs.Known("observation is older than MaxResolutionAge"))
	}
	return nil
}

// diagnoseFlags reports, in one event per record, how many listed keys have
// the verified flag false, so a record of up to 32 keys cannot turn one
// evaluation into many log lines.
func diagnoseFlags(ctx context.Context, rec Record, s core.Settings) {
	n := 0
	for _, k := range rec.Keys {
		if !k.Verified {
			n++
		}
	}
	if n == 0 {
		return
	}
	s.Trace(ctx, obs.Event{
		Outcome: obs.OutcomeDiagnostic,
		Stage:   obs.StageResolve,
		Subject: obs.Unverified(rec.DID.String()),
		Detail:  obs.Known(strconv.Itoa(n) + " listed key(s) with verified=false: tampering or an old registry"),
	})
}

func selectSigningKey(rec Record) (TrustedKey, error) {
	for _, t := range []struct {
		typ KeyType
		alg crypto.Alg
	}{{KeyTypeECDSA, crypto.AlgES256K}, {KeyTypeEd25519, crypto.AlgEd25519}} {
		for _, k := range rec.Keys {
			if k.Type != t.typ {
				continue
			}
			pub, err := crypto.ParsePublicKey(t.alg, k.KeyData)
			if err != nil {
				return TrustedKey{}, core.Reject(obs.ReasonResolution, obs.Known("selected key does not decode"))
			}
			if crypto.VerifyPoP(pub, k.KeyData, rec.DID.AsSent(), k.Signature) != nil {
				continue
			}
			return TrustedKey{pub: pub, typ: t.typ}, nil
		}
	}
	return TrustedKey{}, core.Reject(obs.ReasonKeyNotEligible, obs.Known("no listed signing key with a valid proof of possession"))
}

// ResolveAgent resolves d with r and evaluates the record for d (06 §3,
// §3.1): it is step 4 of 03 §5 and the resolution step of every other
// verification path. Any resolver error, a nil resolver among them, is a
// rejection with reason resolution, so every caller maps it the same way and
// fails closed (E06-12); a rejection from Evaluate is returned unchanged.
func ResolveAgent(ctx context.Context, r KeyResolver, d DID, p Policy) (Agent, error) {
	if p.Clock == nil || p.Settings.Validate() != nil {
		return Agent{}, ErrPolicy
	}
	if r == nil {
		return Agent{}, core.Reject(obs.ReasonResolution, obs.Known("no resolver"))
	}
	rec, o, err := r.Resolve(ctx, d)
	if err != nil {
		return Agent{}, core.Reject(obs.ReasonResolution, obs.Known("the resolver failed"))
	}
	return Evaluate(ctx, d, rec, o, p)
}
