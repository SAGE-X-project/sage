//go:build darwin && registry_native_limits

package registry011

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/registry011/internal/compilermemory"
)

func nativeClockSelectionFixture(t *testing.T) (*HistoryPreparationSession, *nativeHistoryPreparation, *nativeHistoryOwnedSelection, *ProviderPolicyCandidate) {
	t.Helper()
	raw, _, _ := nativeHistoryTestRaw(t)
	session, state, _ := nativeHistoryTestSession(t, len(raw))
	session.used.Store(true)
	selection, err := state.decodeOwnedSelection(session, raw)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := selection.transientPolicy()
	if err != nil {
		t.Fatal(err)
	}
	return session, state, selection, policy
}

func TestProviderHistoryNativeSelectionClockReadsOriginalOwnedPolicy(t *testing.T) {
	raw, _, _ := nativeHistoryTestRaw(t)
	session, state, _ := nativeHistoryTestSession(t, len(raw))
	compiler := session.compiler
	if err := session.CompileOwned(raw); err != nil {
		t.Fatal(err)
	}
	selection := state.selection
	policy, err := selection.transientPolicy()
	if err != nil {
		t.Fatal(err)
	}
	expected, err := (providerOwnedClockGoPolicyReader{}).readOwnedClockPolicy(policy)
	if err != nil {
		t.Fatal(err)
	}
	if !selection.numbers.closed {
		t.Fatal("clock policy kept its temporary numeric arena")
	}
	fields, err := selection.readOwnedClockPolicy(policy)
	if err != nil || fields != expected {
		t.Fatal("clock policy did not consume the original owned bytes", fields, err)
	}
	plan := session.plan
	if plan == nil || plan.clock == nil || plan.retained != selection || compiler != session.compiler ||
		plan.clock.origin != compiler.origin || plan.compilerDeadline != compiler.deadline || plan.clock.id != expected.id ||
		plan.clock.policy != policy.policy || !reflect.DeepEqual(plan.config, providerCustodyConfig{}) {
		t.Fatal("native policy did not reach the same compiler and clock")
	}
	clock := plan.clock
	if _, err := compiler.compileSelection(selection); !errors.Is(err, ErrConfiguration) {
		t.Fatal("clock reader restarted the compiler attempt", err)
	}
	if _, err := session.FinishDiagnostic(); err != nil {
		t.Fatal(err)
	}
	if !selection.closed || state.selection != nil || session.plan != nil || clock.origin != compiler.origin {
		t.Fatal("diagnostic retained the native plan or replaced the original clock")
	}
	if fields, err := selection.readOwnedClockPolicy(policy); err == nil || fields != (providerOwnedClockPolicyFields{}) {
		t.Fatal("closed selection produced clock fields", err)
	}
}

func TestProviderHistoryNativeSelectionClockRefusesChangedAndLostOwners(t *testing.T) {
	for _, name := range []string{"nil", "copied", "closed", "policy-nil", "reader-copied", "unvalidated", "sticky-error", "wrong-owner", "wrong-policy", "lost-root", "lost-policy", "invalid-root"} {
		t.Run(name, func(t *testing.T) {
			_, state, selection, policy := nativeClockSelectionFixture(t)
			view := selection
			restore := func() {}
			switch name {
			case "nil":
				view = nil
			case "copied":
				copy := *selection
				view = &copy
			case "closed":
				selection.closed = true
			case "policy-nil":
				original := selection.policy
				selection.policy = nil
				restore = func() { selection.policy = original }
			case "reader-copied":
				original := selection.policy
				selection.policy = &nativeProviderPolicyReader{self: original, source: original.source, numbers: original.numbers, input: original.input, inputBytes: original.inputBytes, root: original.root}
				selection.policy.used.Store(true)
				restore = func() { selection.policy = original }
			case "unvalidated":
				selection.policy.used.Store(false)
			case "sticky-error":
				selection.policy.err = context.Canceled
			case "wrong-owner":
				original := selection.policy.source
				selection.policy.source = state.root.region
				restore = func() { selection.policy.source = original }
			case "wrong-policy":
				copy := *policy
				copy.raw += " "
				policy = &copy
			case "lost-root":
				selection.document = compilermemory.JSONDocument{}
				selection.validated = false
			case "lost-policy":
				if err := state.policy.region.Close(); err != nil {
					t.Fatal(err)
				}
			case "invalid-root":
				selection.policy.root = compilermemory.JSONValue{}
			}
			defer restore()
			if fields, err := view.readOwnedClockPolicy(policy); err == nil || fields != (providerOwnedClockPolicyFields{}) {
				t.Fatal("changed or lost reader produced clock fields", fields, err)
			}
		})
	}
	for _, fixture := range []struct {
		raw string
		n   uint64
		ok  bool
	}{
		{`{}`, 0, false}, {`{"n":0}`, 0, false}, {`{"n":""}`, 0, false},
		{`{"n":"123456789012345678901"}`, 0, false}, {`{"n":"\u0031"}`, 0, false},
		{`{"n":"01"}`, 0, false}, {`{"n":"0"}`, 0, false}, {`{"n":"18446744073709551616"}`, 0, false},
		{`{"n":"1"}`, 1, true}, {`{"n":"18446744073709551615"}`, ^uint64(0), true},
	} {
		t.Run(fixture.raw, func(t *testing.T) {
			buffer := nativeBoundaryDocument(t, []byte(fixture.raw))
			selection := &nativeHistoryOwnedSelection{policy: &nativeProviderPolicyReader{source: buffer.region}}
			n, err := selection.nativeClockPolicyDecimal(buffer.document.Root(), "n")
			if fixture.ok && (err != nil || n != fixture.n) || !fixture.ok && (err == nil || n != 0) {
				t.Fatal("clock decimal contract", n, err)
			}
		})
	}
	for boundary := range 64 {
		t.Run("cancellation-"+strconv.Itoa(boundary), func(t *testing.T) {
			buffer := nativeBoundaryDocument(t, []byte(`{"n":"1"}`))
			selection := &nativeHistoryOwnedSelection{policy: &nativeProviderPolicyReader{source: buffer.region}}
			buffer.action.at.Store(buffer.action.calls.Load() + int64(boundary) + 1)
			n, err := selection.nativeClockPolicyDecimal(buffer.document.Root(), "n")
			if err != nil {
				if !errors.Is(err, context.Canceled) || n != 0 {
					t.Fatal("clock access lost cancellation", n, err)
				}
			} else if n != 1 {
				t.Fatal("uncanceled clock bytes changed", n)
			}
		})
	}
}

func TestProviderHistoryNativeSelectionClockCancellationDiscardsPartialFields(t *testing.T) {
	for boundary := range 128 {
		t.Run(strconv.Itoa(boundary), func(t *testing.T) {
			raw, _, _ := nativeHistoryTestRaw(t)
			origin := time.Now()
			action, cancel := context.WithDeadline(context.Background(), origin.Add(time.Minute))
			t.Cleanup(cancel)
			cleanup, stop := context.WithDeadline(context.Background(), origin.Add(2*time.Minute))
			t.Cleanup(stop)
			control := &nativeBoundaryCancelContext{Context: action, cancel: cancel}
			session, err := NewNativeHistoryPreparationSession(control, cleanup, origin, len(raw), 4096)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := session.Close(); err != nil {
					t.Error(err)
				}
			})
			state := session.native.(*nativeHistoryPreparation)
			session.used.Store(true)
			selection, err := state.decodeOwnedSelection(session, raw)
			if err != nil {
				t.Fatal(err)
			}
			policy, err := selection.transientPolicy()
			if err != nil {
				t.Fatal(err)
			}
			expected, err := (providerOwnedClockGoPolicyReader{}).readOwnedClockPolicy(policy)
			if err != nil {
				t.Fatal(err)
			}
			control.at.Store(control.calls.Load() + int64(boundary) + 1)
			fields, err := selection.readOwnedClockPolicy(policy)
			if action.Err() != nil {
				if !errors.Is(err, context.Canceled) || fields != (providerOwnedClockPolicyFields{}) {
					t.Fatal("canceled native policy published partial clock fields", fields, err)
				}
			} else if err != nil || fields != expected {
				t.Fatal("uncanceled native clock policy changed", fields, err)
			}
		})
	}
}
