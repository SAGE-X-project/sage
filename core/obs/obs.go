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

// Package obs defines the events the core reports to its host: the stage and
// reason of a rejection, bounded and escaped detail, and the spec version of
// the code that produced the event.
//
// Events never carry keys, seeds, session keys, plaintext, bodies or cookies
// (spec 0.9.0-draft.1.1, 00-overview.md §7.6). Strings taken from input are
// held in a Detail, which bounds and escapes them.
package obs

import (
	"context"
	"log/slog"
)

// SpecCommon is the Event.Spec value of code that every spec version shares,
// such as canonical JSON and hashing.
const SpecCommon = "common"

// Stage is a step of the inbound pipeline. Every rejection names the stage
// that failed.
type Stage uint8

// Inbound pipeline stages, in processing order.
const (
	StageUnknown   Stage = iota
	StageRead            // S0 bounded read
	StageParse           // S1 strict parse
	StageCheck           // S2 cheap checks
	StageCanonical       // S3 rebuild the signing input
	StageTiming          // S4 freshness against the clock
	StageResolve         // S5 resolution and key evaluation
	StageVerify          // S6 signature verification
	StageBindBody        // S7 body digest
	StageReplay          // S8 replay reservation
	StageAuthorize       // S9 authorisation
	StageDeliver         // S10 delivery to the host
)

// String returns the stage label S0..S10, or "unknown".
func (s Stage) String() string {
	switch s {
	case StageRead:
		return "S0"
	case StageParse:
		return "S1"
	case StageCheck:
		return "S2"
	case StageCanonical:
		return "S3"
	case StageTiming:
		return "S4"
	case StageResolve:
		return "S5"
	case StageVerify:
		return "S6"
	case StageBindBody:
		return "S7"
	case StageReplay:
		return "S8"
	case StageAuthorize:
		return "S9"
	case StageDeliver:
		return "S10"
	default:
		return "unknown"
	}
}

// Reason is the closed set of rejection causes. Each reason belongs to
// exactly one stage (StageOf).
type Reason uint8

// Rejection reasons.
const (
	ReasonUnknown Reason = iota

	ReasonTooLarge // S0: an input exceeds its size limit

	ReasonMalformed     // S1: syntax, encoding or JSON strictness
	ReasonNonCanonical  // S1: a value not in its one accepted form
	ReasonDuplicate     // S1: duplicate member, label or field line
	ReasonUnknownMember // S1: a member or parameter outside the fixed set

	ReasonIdentifier     // S2: DID, key id, nonce or ctxID syntax or limit
	ReasonParameters     // S2: signature parameters
	ReasonCoverage       // S2: covered components
	ReasonAlgorithm      // S2: algorithm not allowed or not matching the key
	ReasonCookie         // S2: denial-of-service cookie
	ReasonHeaderMismatch // S2: a header differs from the signed body
	ReasonRecipient      // S2: the message is not addressed to this agent

	ReasonCanonicalize // S3: the signing input cannot be rebuilt

	ReasonStale   // S4: created or ts too old
	ReasonFuture  // S4: created or ts too far in the future
	ReasonExpired // S4: expires has passed
	ReasonClock   // S4: the clock failed

	ReasonResolution      // S5: resolution failed or returned malformed data
	ReasonResolutionStale // S5: the resolution result is too old
	ReasonInactive        // S5: the agent is not active
	ReasonKeyNotEligible  // S5: no listed key with a valid proof of possession

	ReasonSignature // S6: the signature does not verify

	ReasonDigest // S7: Content-Digest missing or not matching

	ReasonReplay     // S8: the nonce was already used
	ReasonReplayFull // S8: the replay store is full

	ReasonPeerMismatch // S9: expected peer or authority differs
	ReasonUnauthorized // S9: the host's authorizer denied

	ReasonInspectorDenied // S10: the host's inspector denied delivery

	reasonEnd // sentinel, not a reason
)

// Reasons returns every defined reason, for tests and documentation.
func Reasons() []Reason {
	out := make([]Reason, 0, int(reasonEnd)-1)
	for r := ReasonUnknown + 1; r < reasonEnd; r++ {
		out = append(out, r)
	}
	return out
}

// StageOf returns the one stage a reason belongs to, or StageUnknown for a
// value outside the defined set.
func StageOf(r Reason) Stage {
	switch r {
	case ReasonTooLarge:
		return StageRead
	case ReasonMalformed, ReasonNonCanonical, ReasonDuplicate, ReasonUnknownMember:
		return StageParse
	case ReasonIdentifier, ReasonParameters, ReasonCoverage, ReasonAlgorithm,
		ReasonCookie, ReasonHeaderMismatch, ReasonRecipient:
		return StageCheck
	case ReasonCanonicalize:
		return StageCanonical
	case ReasonStale, ReasonFuture, ReasonExpired, ReasonClock:
		return StageTiming
	case ReasonResolution, ReasonResolutionStale, ReasonInactive, ReasonKeyNotEligible:
		return StageResolve
	case ReasonSignature:
		return StageVerify
	case ReasonDigest:
		return StageBindBody
	case ReasonReplay, ReasonReplayFull:
		return StageReplay
	case ReasonPeerMismatch, ReasonUnauthorized:
		return StageAuthorize
	case ReasonInspectorDenied:
		return StageDeliver
	default:
		return StageUnknown
	}
}

// String returns a stable snake_case name of the reason.
func (r Reason) String() string {
	switch r {
	case ReasonTooLarge:
		return "too_large"
	case ReasonMalformed:
		return "malformed"
	case ReasonNonCanonical:
		return "non_canonical"
	case ReasonDuplicate:
		return "duplicate"
	case ReasonUnknownMember:
		return "unknown_member"
	case ReasonIdentifier:
		return "identifier"
	case ReasonParameters:
		return "parameters"
	case ReasonCoverage:
		return "coverage"
	case ReasonAlgorithm:
		return "algorithm"
	case ReasonCookie:
		return "cookie"
	case ReasonHeaderMismatch:
		return "header_mismatch"
	case ReasonRecipient:
		return "recipient"
	case ReasonCanonicalize:
		return "canonicalize"
	case ReasonStale:
		return "stale"
	case ReasonFuture:
		return "future"
	case ReasonExpired:
		return "expired"
	case ReasonClock:
		return "clock"
	case ReasonResolution:
		return "resolution"
	case ReasonResolutionStale:
		return "resolution_stale"
	case ReasonInactive:
		return "inactive"
	case ReasonKeyNotEligible:
		return "key_not_eligible"
	case ReasonSignature:
		return "signature"
	case ReasonDigest:
		return "digest"
	case ReasonReplay:
		return "replay"
	case ReasonReplayFull:
		return "replay_full"
	case ReasonPeerMismatch:
		return "peer_mismatch"
	case ReasonUnauthorized:
		return "unauthorized"
	case ReasonInspectorDenied:
		return "inspector_denied"
	default:
		return "unknown"
	}
}

// Outcome says whether an event reports a rejection or an accepted input.
type Outcome uint8

// Event outcomes.
const (
	OutcomeRejected Outcome = iota
	OutcomeAccepted
)

// String returns "rejected" or "accepted".
func (o Outcome) String() string {
	if o == OutcomeAccepted {
		return "accepted"
	}
	return "rejected"
}

// Event is one report from the core to its host. It holds no secret and no
// payload. Spec names the spec version of the code that produced the event
// (for example "0.9.0-draft.1.1"), or SpecCommon.
type Event struct {
	Spec      string
	Outcome   Outcome
	Stage     Stage
	Reason    Reason
	Subject   Detail // DID or key id the event is about
	MessageID Detail
	Detail    Detail
}

// Tracer receives events. Implementations must not block for long and must
// not keep references to the event's strings beyond the call.
type Tracer interface {
	Trace(ctx context.Context, e Event)
}

// Emit sends e to t. A nil tracer discards the event, and a panic inside the
// tracer is contained, so tracing never fails processing.
func Emit(ctx context.Context, t Tracer, e Event) {
	if t == nil {
		return
	}
	defer func() { _ = recover() }()
	t.Trace(ctx, e)
}

// SlogTracer writes events to a log/slog logger at debug level for accepted
// inputs and at info level for rejections. A nil Logger discards events.
type SlogTracer struct {
	Logger *slog.Logger
}

// Trace implements Tracer.
func (t SlogTracer) Trace(ctx context.Context, e Event) {
	if t.Logger == nil {
		return
	}
	level := slog.LevelInfo
	if e.Outcome == OutcomeAccepted {
		level = slog.LevelDebug
	}
	attrs := []slog.Attr{
		slog.String("spec", Known(e.Spec).String()),
		slog.String("outcome", e.Outcome.String()),
		slog.String("stage", e.Stage.String()),
	}
	if e.Outcome == OutcomeRejected {
		attrs = append(attrs, slog.String("reason", e.Reason.String()))
	}
	attrs = appendDetail(attrs, "subject", e.Subject)
	attrs = appendDetail(attrs, "message_id", e.MessageID)
	attrs = appendDetail(attrs, "detail", e.Detail)
	t.Logger.LogAttrs(ctx, level, "sage", attrs...)
}

func appendDetail(attrs []slog.Attr, key string, d Detail) []slog.Attr {
	if d.IsZero() {
		return attrs
	}
	return append(attrs, slog.Group(key,
		slog.String("value", d.String()),
		slog.Bool("unverified", d.Unverified()),
	))
}
