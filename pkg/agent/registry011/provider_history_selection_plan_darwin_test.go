//go:build darwin && registry_native_limits

package registry011

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/registry011/internal/compilermemory"
)

func TestProviderHistoryNativeOwnedSelectionConsumesCompiledNativeManifest(t *testing.T) {
	raw, identity, manifestDigest := nativeHistoryTestRaw(t)
	session, state, _ := nativeHistoryTestSession(t, len(raw))
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Error(err)
		}
	})
	compiler := session.compiler
	decoded, err := decodeHistoryPreparationInput(compiler.ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := buildHistoryPreparationConfig(decoded)
	if err != nil || legacy.identity != identity || legacy.bounds.history.collection.digest != manifestDigest {
		t.Fatalf("original semantic fixture precondition: %v", err)
	}
	if err := session.CompileOwned(raw); err != nil {
		t.Fatal(err)
	}
	selection, plan := state.selection, session.plan
	if selection == nil || plan == nil || plan.retained != selection || selection.plan != plan ||
		session.compiler != compiler || selection.compiler != compiler || !compiler.used.Load() ||
		plan.compilerDeadline != compiler.deadline || plan.clock == nil || plan.clock.origin != compiler.origin ||
		plan.config != (providerCustodyConfig{}) {
		t.Fatal("actual native consumer replaced or duplicated its original retained plan")
	}
	policy, err := selection.transientPolicy()
	if err != nil || !bytes.Equal(policy.CanonicalBytes(), legacy.policy.CanonicalBytes()) {
		t.Fatalf("retained native policy differs from the original candidate: %v", err)
	}
	bound, err := policy.BindCandidate(decoded.Namespace, decoded.SourceID, decoded.Generation)
	if err != nil {
		t.Fatal(err)
	}
	if actual, err := bound.Identity(); err != nil || actual != identity {
		t.Fatalf("native subtree hashes changed original identities: %v", err)
	}
	digest, counts, err := selection.retainedManifest()
	if err != nil || digest != manifestDigest || counts != legacy.bounds.history.collection.counts {
		t.Fatalf("retained native manifest differs from compiled DATA: %v", err)
	}
	length, err := selection.manifestOwner.SpanLength(selection.manifestSpan)
	if err != nil {
		t.Fatal(err)
	}
	span, err := selection.manifestOwner.Slice(selection.manifestSpan, nativeHistoryManifestHeaderBytes, length-nativeHistoryManifestHeaderBytes)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := nativeHistorySpanBytes(selection.manifestOwner, span, providerHistorySourceRawTextLimit)
	if err != nil || string(actual) != legacy.bounds.history.collection.raw {
		t.Fatalf("actual original manifest bytes were not retained: %v", err)
	}
	clear(raw)
	store, err := nativeHistorySelectionPart[historyPreparationStore](selection, "store")
	if err != nil || store != decoded.Store {
		t.Fatalf("retained source depended on released caller input: %v", err)
	}
	copyPlan, copySelection := *plan, *selection
	if err := selection.checkRetainedPlan(&copyPlan); err != ErrConfiguration {
		t.Fatalf("copied plan consumed the original selection: %v", err)
	}
	if err := copySelection.checkRetainedPlan(plan); err != ErrConfiguration {
		t.Fatalf("copied selection consumed the original plan: %v", err)
	}
	if _, err := compiler.compileSelection(selection); err != ErrConfiguration || session.plan != plan {
		t.Fatalf("native compilation attempt was reused: %v", err)
	}
	output, err := session.FinishDiagnostic()
	if err != nil {
		t.Fatal(err)
	}
	var summary HistoryPreparationSummary
	if err := json.Unmarshal(output, &summary); err != nil || summary.SourceReady || summary.Identity != identity || summary.ManifestSHA256 != manifestDigest {
		t.Fatalf("native output changed the original DATA contract: %v", err)
	}
	if plan.retained != nil || !selection.closed || selection.plan != nil || selection.manifestOwner != nil ||
		!state.root.closed || !state.policy.closed || !state.manifest.closed || !state.result.closed {
		t.Fatal("terminal consumption retained its original selection resources")
	}
	if err := selection.checkRetainedPlan(plan); err != ErrConfiguration {
		t.Fatalf("closed selection remained usable: %v", err)
	}
}

func TestProviderHistoryNativeOwnedSelectionRetainedPlanRejectsLostOwners(t *testing.T) {
	for _, name := range []string{"root", "policy", "manifest", "result", "canceled"} {
		t.Run(name, func(t *testing.T) {
			raw, _, _ := nativeHistoryTestRaw(t)
			session, state, cancel := nativeHistoryTestSession(t, len(raw))
			t.Cleanup(func() {
				if err := session.Close(); err != nil {
					t.Error(err)
				}
			})
			if err := session.CompileOwned(raw); err != nil {
				t.Fatal(err)
			}
			var region *compilermemory.Region
			want := compilermemory.ErrClosed
			switch name {
			case "root":
				region = state.root.region
			case "policy":
				region = state.policy.region
			case "manifest":
				region = state.manifest.region
			case "result":
				region = state.result.region
			case "canceled":
				cancel()
				want = context.Canceled
			}
			if region != nil {
				if err := region.Close(); err != nil {
					t.Fatal(err)
				}
			}
			if output, err := session.FinishDiagnostic(); output != nil || !errors.Is(err, want) {
				t.Fatalf("lost original owner produced output=%q, error=%v; expected %v", output, err, want)
			}
			if session.plan != nil || session.compiler != nil || !state.manifest.closed || !state.result.closed {
				t.Fatal("lost-owner refusal did not discard the original plan")
			}
		})
	}
}

func nativeHistorySelectionAcquisitionFault(t *testing.T, position int) {
	t.Helper()
	for _, partial := range []bool{false, true} {
		name := "nil-owner"
		if partial {
			name = "partial-owner"
		}
		t.Run(name, func(t *testing.T) {
			raw, _, _ := nativeHistoryTestRaw(t)
			session, state, _ := nativeHistoryTestSession(t, len(raw))
			acquire := state.acquire
			marker := errors.New("simulated owned acquisition failure")
			calls := 0
			var acquired []*compilermemory.Region
			state.acquire = func(action, cleanup context.Context, capacity int) (*compilermemory.Region, error) {
				calls++
				if calls == position && !partial {
					return nil, marker
				}
				region, err := acquire(action, cleanup, capacity)
				if region != nil {
					acquired = append(acquired, region)
				}
				if err == nil && calls == position {
					return region, marker
				}
				return region, err
			}
			err := session.CompileOwned(raw)
			if !errors.Is(err, marker) || calls != position || session.plan != nil || session.compiler != nil || !session.closed {
				t.Fatalf("intended acquisition boundary was not reached: calls=%d, error=%v", calls, err)
			}
			for _, region := range acquired {
				if _, err := region.SpanLength(compilermemory.Span{}); err != compilermemory.ErrClosed {
					t.Fatalf("partial original owner remained accessible: %v", err)
				}
			}
			if err := session.CompileOwned(raw); err != ErrConfiguration || calls != position {
				t.Fatalf("failed owned attempt was retried: %v", err)
			}
			if err := session.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestProviderHistoryNativeOwnedSelectionManifestAcquisitionFailuresRemainTerminal(t *testing.T) {
	nativeHistorySelectionAcquisitionFault(t, 4)
}

func TestProviderHistoryNativeOwnedSelectionResultAcquisitionFailuresRemainTerminal(t *testing.T) {
	nativeHistorySelectionAcquisitionFault(t, 5)
}

func TestProviderHistoryNativeOwnedSelectionCleanupFailureDiscardsPlan(t *testing.T) {
	for _, name := range []string{"temporary-numbers", "retained-manifest"} {
		t.Run(name, func(t *testing.T) {
			raw, _, _ := nativeHistoryTestRaw(t)
			origin := time.Now()
			action, cancel := context.WithDeadline(context.Background(), origin.Add(time.Minute))
			cleanup, stopCleanup := context.WithDeadline(context.Background(), origin.Add(2*time.Minute))
			t.Cleanup(cancel)
			t.Cleanup(stopCleanup)
			session, err := NewNativeHistoryPreparationSession(action, cleanup, origin, len(raw), 4096)
			if err != nil {
				t.Fatal(err)
			}
			state := session.native.(*nativeHistoryPreparation)
			var expectedCleanup error
			t.Cleanup(func() {
				if err := session.Close(); err != expectedCleanup {
					t.Errorf("cleanup changed the original result: got %v, want %v", err, expectedCleanup)
				}
				checkNativeHistoryReleased(t, session, state)
			})
			marker := errors.New("simulated cached cleanup refusal")
			calls := 0
			injected := false
			acquire := state.acquire
			state.acquire = func(action, cleanup context.Context, capacity int) (*compilermemory.Region, error) {
				calls++
				region, err := acquire(action, cleanup, capacity)
				if err == nil && calls == 5 && name == "temporary-numbers" {
					// Release the real mapping before injecting the owner's cached
					// boundary error. This does not simulate an OS unmap failure.
					if err := state.selection.numbers.close(); err != nil {
						return region, err
					}
					state.selection.numbers.closeErr = marker
					injected = true
				}
				return region, err
			}
			err = session.CompileOwned(raw)
			if name == "retained-manifest" {
				if err != nil {
					t.Fatalf("original positive compile precondition: %v", err)
				}
				if err := state.manifest.close(); err != nil {
					t.Fatal(err)
				}
				state.manifest.closeErr = marker
				injected = true
				var output []byte
				output, err = session.FinishDiagnostic()
				if output != nil {
					t.Fatal("cleanup refusal released diagnostic output")
				}
			}
			if !injected || calls != 5 || !errors.Is(err, marker) || !errors.Is(err, ErrHistoryPreparationCleanup) ||
				session.plan != nil || session.compiler != nil || !state.root.closed || !state.policy.closed || !state.manifest.closed || !state.result.closed {
				t.Fatalf("cached cleanup failure was lost or masked: calls=%d, error=%v", calls, err)
			}
			cached := session.cleanupErr
			expectedCleanup = cached
			if err := session.Close(); err != cached || !errors.Is(err, marker) || calls != 5 {
				t.Fatalf("cleanup result was retried or replaced: %v", err)
			}
		})
	}
}

func TestProviderHistoryNativeOwnedSelectionHeaderKeepsOriginalRefusalOrder(t *testing.T) {
	for _, name := range []string{"format", "capture"} {
		t.Run(name, func(t *testing.T) {
			raw, _, _ := nativeHistoryTestRaw(t)
			var object map[string]json.RawMessage
			if err := json.Unmarshal(raw, &object); err != nil {
				t.Fatal(err)
			}
			object["policy"] = json.RawMessage(`"{}"`)
			if name == "format" {
				object["format"] = json.RawMessage(`"different"`)
			} else {
				var capture map[string]json.RawMessage
				if err := json.Unmarshal(object["capture"], &capture); err != nil {
					t.Fatal(err)
				}
				capture["Timeout"] = json.RawMessage(`0`)
				data, err := json.Marshal(capture)
				if err != nil {
					t.Fatal(err)
				}
				object["capture"] = data
			}
			changed, err := json.Marshal(object)
			if err != nil {
				t.Fatal(err)
			}
			session, _, _ := nativeHistoryTestSession(t, len(changed))
			decoded, err := decodeHistoryPreparationInput(session.compiler.ctx, changed)
			if err != nil {
				t.Fatalf("typed input precondition: %v", err)
			}
			if _, err := buildHistoryPreparationConfig(decoded); err != ErrConfiguration {
				t.Fatalf("original format/capture refusal was not reached: %v", err)
			}
			if err := session.CompileOwned(changed); err != ErrConfiguration {
				t.Fatalf("native policy validation masked the original header refusal: %v", err)
			}
			if err := session.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
