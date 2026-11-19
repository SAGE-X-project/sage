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

package core

import (
	"errors"
	"time"
)

// Spec maxima of 00-overview.md §7 DP-1 and DP-4 (spec 0.9.0-draft.1.1).
const (
	SpecMaxBodyBytes           = 1 << 20           // HTTP body, WebSocket message (E08-10)
	SpecMaxDigestBodyBytes     = 16 << 20          // body buffered for Content-Digest (E03-13)
	SpecMaxSignatureFieldBytes = 8 << 10           // Signature-Input, Signature (E03-21)
	SpecMaxCoveredComponents   = 32                // RFC 9421 covered components (E03-21)
	SpecMaxHeaderBytes         = 64 << 10          // HTTP header section (E03-21)
	SpecMaxHPKEInitBytes       = 4 << 10           // HPKE init payload (E04-06)
	SpecMaxRecordBytes         = 16<<20 + 36       // session record (E05-08)
	SpecMaxAADBytes            = 4 << 10           // caller AAD (E05-08)
	SpecMaxAge                 = 300 * time.Second // RFC 9421 MaxAge, default (E03-11); RFC 9421 only
	SpecMaxClockSkew           = 300 * time.Second // MaxClockSkew (E03-11)
	SpecMaxResolutionAge       = 300 * time.Second // positive resolution result (E06-12)
)

// MinCoveredComponents is the largest set a response signature must cover
// (E03-10: @status, the three ;req components, "signature";req,
// "content-digest";req and content-digest), so a lower limit would make
// conforming responses unverifiable.
const MinCoveredComponents = 7

// Recommended values of E03-11 and E06-12.
const (
	RecommendedClockSkew     = 60 * time.Second
	RecommendedResolutionAge = 60 * time.Second
)

// ErrLimits is returned for a Limits value outside the spec ranges.
var ErrLimits = errors.New("sage: limits outside the spec ranges")

// Limits holds the size and time limits a receiver enforces (DP-1, DP-4).
// It is a plain value; components take it through WithLimits. Every limit
// may be lowered below its spec maximum but never raised above it. A
// lowered size limit makes the host reject some messages a conforming peer
// may send; that is the host's choice. Syntax limits (DID 256 bytes, nonce
// 16-64 characters, ctxID 128 characters) are not here: they define valid
// input and cannot be changed. MaxAge is the RFC 9421 MaxAge only; the
// envelope window (300 s, E08-01) and the HPKE ts window (±120 s) are fixed
// by the spec and live in their packages.
type Limits struct {
	MaxBodyBytes           int64
	MaxDigestBodyBytes     int64
	MaxSignatureFieldBytes int
	MaxCoveredComponents   int
	MaxHeaderBytes         int
	MaxHPKEInitBytes       int
	MaxRecordBytes         int64
	MaxAADBytes            int
	MaxAge                 time.Duration // (0, 300 s]
	MaxClockSkew           time.Duration // [0, 300 s]
	MaxResolutionAge       time.Duration // (0, 300 s]
}

// DefaultLimits returns the spec maxima, with the recommended clock skew
// and resolution age (E03-11, E06-12).
func DefaultLimits() Limits {
	return Limits{
		MaxBodyBytes:           SpecMaxBodyBytes,
		MaxDigestBodyBytes:     SpecMaxDigestBodyBytes,
		MaxSignatureFieldBytes: SpecMaxSignatureFieldBytes,
		MaxCoveredComponents:   SpecMaxCoveredComponents,
		MaxHeaderBytes:         SpecMaxHeaderBytes,
		MaxHPKEInitBytes:       SpecMaxHPKEInitBytes,
		MaxRecordBytes:         SpecMaxRecordBytes,
		MaxAADBytes:            SpecMaxAADBytes,
		MaxAge:                 SpecMaxAge,
		MaxClockSkew:           RecommendedClockSkew,
		MaxResolutionAge:       RecommendedResolutionAge,
	}
}

// Validate reports ErrLimits when a value is outside its spec range. A
// freshness check cannot be disabled: MaxAge and MaxResolutionAge must be
// positive (E03-11, DP-4).
func (l Limits) Validate() error {
	ok := within(l.MaxBodyBytes, SpecMaxBodyBytes) &&
		within(l.MaxDigestBodyBytes, SpecMaxDigestBodyBytes) &&
		within(l.MaxSignatureFieldBytes, SpecMaxSignatureFieldBytes) &&
		within(l.MaxCoveredComponents, SpecMaxCoveredComponents) && l.MaxCoveredComponents >= MinCoveredComponents &&
		within(l.MaxHeaderBytes, SpecMaxHeaderBytes) &&
		within(l.MaxHPKEInitBytes, SpecMaxHPKEInitBytes) &&
		within(l.MaxRecordBytes, SpecMaxRecordBytes) &&
		within(l.MaxAADBytes, SpecMaxAADBytes) &&
		within(l.MaxAge, SpecMaxAge) &&
		l.MaxClockSkew >= 0 && l.MaxClockSkew <= SpecMaxClockSkew &&
		within(l.MaxResolutionAge, SpecMaxResolutionAge)
	if !ok {
		return ErrLimits
	}
	return nil
}

func within[T int | int64 | time.Duration](v, max T) bool { return v > 0 && v <= max }
