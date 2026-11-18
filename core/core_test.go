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
	"context"
	"errors"
	"testing"

	"github.com/sage-x-project/sage/core/obs"
)

func TestZeroSettingsFailsClosed(t *testing.T) {
	var s Settings
	if !errors.Is(s.Validate(), ErrSpecVersionRequired) {
		t.Fatal("zero Settings must not validate")
	}
	ok, err := NewSettings(WithSpecVersion(string(V0_9_0_Draft1_1)), Option{})
	if err != nil || ok.Validate() != nil {
		t.Fatalf("valid settings: %v %v", err, ok.Validate())
	}
}

func TestNewSettingsRequiresSpecVersion(t *testing.T) {
	if _, err := NewSettings(); !errors.Is(err, ErrSpecVersionRequired) {
		t.Fatalf("no option: got %v, want ErrSpecVersionRequired", err)
	}
	if _, err := NewSettings(WithTracer(nil)); !errors.Is(err, ErrSpecVersionRequired) {
		t.Fatalf("tracer only: got %v, want ErrSpecVersionRequired", err)
	}
}

func TestWithSpecVersionIsExact(t *testing.T) {
	s, err := NewSettings(WithSpecVersion("0.9.0-draft.1.1"))
	if err != nil {
		t.Fatalf("supported version rejected: %v", err)
	}
	if s.Spec() != V0_9_0_Draft1_1 {
		t.Fatalf("Spec() = %q", s.Spec())
	}
	for _, v := range []string{
		"", "0.9.0", "0.9.0-draft.1", "0.9.0-draft.2", "0.10.0",
		" 0.9.0-draft.1.1", "0.9.0-draft.1.1 ", "0.9.0-DRAFT.1.1", "v0.9.0-draft.1.1",
		"0.9.0-draft.1.10",
	} {
		if _, err := NewSettings(WithSpecVersion(v)); !errors.Is(err, ErrSpecVersionUnsupported) {
			t.Errorf("WithSpecVersion(%q): got %v, want ErrSpecVersionUnsupported", v, err)
		}
	}
}

func TestRejectionNeverDisclosesDetail(t *testing.T) {
	secret := "keyid=did:sage:ethereum:0xabc reason=replay"
	for _, r := range obs.Reasons() {
		rej := Reject(r, obs.Unverified(secret))
		if rej.Error() != AuthenticationFailedText {
			t.Errorf("%v: Error() = %q", r, rej.Error())
		}
		if !errors.Is(rej, ErrAuthentication) {
			t.Errorf("%v: does not unwrap to ErrAuthentication", r)
		}
		var got *Rejection
		if !errors.As(error(rej), &got) || got.Reason() != r {
			t.Errorf("%v: errors.As failed", r)
		}
		if rej.Stage() != obs.StageOf(r) || rej.Stage() == obs.StageUnknown {
			t.Errorf("%v: stage %v", r, rej.Stage())
		}
		if rej.Detail().String() == "" || !rej.Detail().Unverified() {
			t.Errorf("%v: detail lost", r)
		}
	}
}

type recorder struct{ events []obs.Event }

func (r *recorder) Trace(_ context.Context, e obs.Event) { r.events = append(r.events, e) }

type panicker struct{}

func (panicker) Trace(context.Context, obs.Event) { panic("tracer failure") }

func TestSettingsTrace(t *testing.T) {
	rec := &recorder{}
	s, err := NewSettings(WithSpecVersion(string(V0_9_0_Draft1_1)), WithTracer(rec))
	if err != nil {
		t.Fatal(err)
	}
	rej := Reject(obs.ReasonReplay, obs.Known("nonce reused"))
	s.Trace(context.Background(), rej.Event(s.Spec()))
	s.Trace(context.Background(), obs.Event{Outcome: obs.OutcomeAccepted})
	if len(rec.events) != 2 || rec.events[1].Spec != "0.9.0-draft.1.1" {
		t.Fatalf("got %d events", len(rec.events))
	}
	e := rec.events[0]
	if e.Spec != "0.9.0-draft.1.1" || e.Stage != obs.StageReplay || e.Reason != obs.ReasonReplay {
		t.Fatalf("event = %+v", e)
	}

	p, err := NewSettings(WithSpecVersion(string(V0_9_0_Draft1_1)), WithTracer(panicker{}))
	if err != nil {
		t.Fatal(err)
	}
	p.Trace(context.Background(), e) // must not panic
}
