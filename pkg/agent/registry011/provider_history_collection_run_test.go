package registry011

import (
	"context"
	"testing"
)

// Fixed pre-entry UNIT states, with no directories, native clock, connection,
// original first operation or qualified derivation. A numeric manifest or a
// copied parent must not consume the first-operation bridge or start I/O.
func TestProviderHistoryCollectionRunRefusesUnqualifiedEntryWithoutStateChanges(t *testing.T) {
	for _, mode := range []string{"nil-context", "cancelled-context", "nil-parent", "nil-store", "copied-owner", "wrong-store-parent", "missing-manifest", "numeric-manifest", "already-entered"} {
		t.Run(mode, func(t *testing.T) {
			store := &providerCustodyStore{}
			parent := &providerHistoryProgramOwner{store: store}
			parent.owner, store.history = parent, parent
			parent.bounds.stages = 1
			parent.bounds.collection = &providerHistoryCollectionManifest{}
			ctx := context.Background()
			target := parent
			switch mode {
			case "nil-context":
				ctx = nil
			case "cancelled-context":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "nil-parent":
				target = nil
			case "nil-store":
				parent.store = nil
			case "copied-owner":
				parent.owner = &providerHistoryProgramOwner{}
			case "wrong-store-parent":
				store.history = &providerHistoryProgramOwner{}
			case "missing-manifest":
				parent.bounds.collection = nil
			case "already-entered":
				parent.group = &providerHistoryGroupOwner{}
			}
			history, group, original := store.history, parent.group, parent.original
			if out, err := target.runCollectionCandidate(ctx); err == nil || out != nil {
				t.Fatal("unqualified input entered collection")
			}
			if store.history != history || parent.group != group || parent.original != original ||
				store.failed || store.closed || parent.failed || store.sequence != 0 || store.stored != 0 ||
				store.setupOrigin != (ClockSample{}) || store.activeMeter.Load() != nil ||
				store.custody != nil || store.evidence != nil || store.clock != nil {
				t.Fatal("pre-entry refusal changed the original state or selected native resources")
			}
		})
	}
}

func TestProviderHistoryCollectionRunRejectsDetachedCompletionData(t *testing.T) {
	for _, group := range []*providerHistoryGroupOwner{nil, {}, {store: &providerCustodyStore{}}, {store: &providerCustodyStore{}, parent: &providerHistoryProgramOwner{}}} {
		if done, valid := group.collectionSettlementStateLocked(nil, 1); done || valid {
			t.Fatal("missing original S became terminal collection data")
		}
		if done, valid := group.collectionSettlementStateLocked(&providerHistorySettledResults{}, 0); done || valid {
			t.Fatal("detached completion data selected a final role")
		}
	}
}
