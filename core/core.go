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

// Package core is the root of the SAGE protocol core: the spec versions it
// implements, the settings every constructor takes, and the one error class a
// peer may observe.
//
// The core depends only on the standard library and a small fixed set of
// cryptography modules. It has no package state that changes, no init
// functions, no environment or file access and no console output; every
// effect goes through an interface the host injects.
package core

import (
	"context"
	"errors"

	"github.com/sage-x-project/sage/core/obs"
)

// SpecVersion names one version of the SAGE specification.
type SpecVersion string

// Spec versions implemented by this module.
const (
	// V0_9_0_Draft1_1 is 0.9.0-draft.1 finalized with its errata
	// (sage-spec v1/).
	V0_9_0_Draft1_1 SpecVersion = "0.9.0-draft.1.1"
)

// SupportedSpecVersions returns the versions this module implements.
func SupportedSpecVersions() []SpecVersion {
	return []SpecVersion{V0_9_0_Draft1_1}
}

// ErrSpecVersionRequired is returned when a constructor is called without
// WithSpecVersion.
var ErrSpecVersionRequired = errors.New("sage: spec version is required")

// ErrSpecVersionUnsupported is returned when WithSpecVersion names a version
// this module does not implement. The comparison is exact: no trimming, no
// case folding and no prefix match.
var ErrSpecVersionUnsupported = errors.New("sage: spec version is not supported")

// AuthenticationFailedText is the only failure text a peer may observe.
const AuthenticationFailedText = "sage: authentication failed"

// ErrAuthentication is the only failure class a peer may observe. Every
// rejection of inbound input unwraps to it. Compare with errors.Is.
var ErrAuthentication = errors.New(AuthenticationFailedText)

// Settings holds the choices fixed when a component is constructed. Its
// fields cannot be changed after NewSettings returns. The zero value is not
// valid: every constructor in this module takes ...Option and calls
// NewSettings itself, and a method on a zero Settings fails closed.
type Settings struct {
	spec   SpecVersion
	tracer obs.Tracer
}

// Option sets one field of Settings. Options are applied only by
// NewSettings.
type Option struct {
	apply func(*Settings) error
}

// WithSpecVersion selects the spec version a component implements. It is
// required, and it cannot be changed after construction.
func WithSpecVersion(v string) Option {
	return Option{apply: func(s *Settings) error {
		for _, supported := range SupportedSpecVersions() {
			if v == string(supported) {
				s.spec = supported
				return nil
			}
		}
		return ErrSpecVersionUnsupported
	}}
}

// WithTracer sets the tracer that receives events. Without it events are
// discarded.
func WithTracer(t obs.Tracer) Option {
	return Option{apply: func(s *Settings) error {
		s.tracer = t
		return nil
	}}
}

// NewSettings applies opts in order. It fails when no spec version was set or
// when an option fails.
func NewSettings(opts ...Option) (Settings, error) {
	var s Settings
	for _, opt := range opts {
		if opt.apply == nil {
			continue
		}
		if err := opt.apply(&s); err != nil {
			return Settings{}, err
		}
	}
	if s.spec == "" {
		return Settings{}, ErrSpecVersionRequired
	}
	return s, nil
}

// Spec returns the selected spec version, or "" for a zero Settings.
func (s Settings) Spec() SpecVersion { return s.spec }

// Validate returns ErrSpecVersionRequired for a Settings not made by
// NewSettings. Components call it before any processing.
func (s Settings) Validate() error {
	if s.spec == "" {
		return ErrSpecVersionRequired
	}
	return nil
}

// Trace sends e to the configured tracer. An event without a spec version is
// labelled with the selected one. A panic in the tracer is contained.
func (s Settings) Trace(ctx context.Context, e obs.Event) {
	if e.Spec == "" {
		e.Spec = string(s.spec)
	}
	obs.Emit(ctx, s.tracer, e)
}

// Rejection reports why inbound input was rejected. Its Error text is always
// the generic ErrAuthentication text, so a host that forwards err.Error() to a
// peer discloses nothing (00-overview.md §7.5). The host reads the stage,
// reason and detail with errors.As or through its Tracer. A Rejection is
// immutable; its stage is derived from its reason.
type Rejection struct {
	reason obs.Reason
	detail obs.Detail
}

// Reject builds a Rejection.
func Reject(reason obs.Reason, detail obs.Detail) *Rejection {
	return &Rejection{reason: reason, detail: detail}
}

// Reason returns why the input was rejected.
func (r *Rejection) Reason() obs.Reason { return r.reason }

// Stage returns the pipeline stage of the reason.
func (r *Rejection) Stage() obs.Stage { return obs.StageOf(r.reason) }

// Detail returns the bounded, escaped detail.
func (r *Rejection) Detail() obs.Detail { return r.detail }

// Error returns the generic text and never the detail.
func (r *Rejection) Error() string { return AuthenticationFailedText }

// Unwrap returns ErrAuthentication.
func (r *Rejection) Unwrap() error { return ErrAuthentication }

// Event returns the rejection as an event of the code that implements spec.
func (r *Rejection) Event(spec SpecVersion) obs.Event {
	return obs.Event{
		Spec:    string(spec),
		Outcome: obs.OutcomeRejected,
		Stage:   r.Stage(),
		Reason:  r.reason,
		Detail:  r.detail,
	}
}
