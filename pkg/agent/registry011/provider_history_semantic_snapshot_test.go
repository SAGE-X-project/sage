package registry011

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"testing"
)

func historySemanticRootReference(kind string, value map[string]any) string {
	value["format"], value["kind"] = providerHistoryCollectionStateDomain, kind
	return providerReplayDigest(providerHistoryCollectionStateDomain, providerReplayCanonical(value, candidateLedgerMetadataLimit, nil), nil)
}

func TestProviderHistorySemanticIndexRootMatchesIndependentSortedReference(t *testing.T) {
	for _, count := range []int{0, 1, 2, 3, 7, 8, 15, 16, 17, 257} {
		t.Run(fmt.Sprintf("entries-%d", count), func(t *testing.T) {
			s := replayStream{claims: map[replayClaimID]replayClaimState{}, created: map[string]bool{}}
			claims := map[string]replayClaimState{}
			created := map[string]bool{}
			for i := count - 1; i >= 0; i-- {
				id := replayClaimID{actor: replayTestWallet, digest: [32]byte{byte(i), byte(i >> 8)}}
				value := replayClaimState{block: uint64(i), consumed: i%2 == 0}
				if i == 2 {
					value.block = ^uint64(0)
				}
				s.claims[id] = value
				claims[fmt.Sprintf("%s:%x", id.actor, id.digest)] = value
				key := fmt.Sprintf("agent.%03d", i)
				s.created[key], created[key] = i%2 == 0, i%2 == 0
			}
			for _, isClaims := range []bool{true, false} {
				kind := "created"
				keys := make([]string, 0, count)
				if isClaims {
					kind = "claims"
					for key := range claims {
						keys = append(keys, key)
					}
				} else {
					for key := range created {
						keys = append(keys, key)
					}
				}
				sort.Strings(keys) // Independent standard-library sorting oracle.
				want := historySemanticRootReference("semantic-index", map[string]any{"index": kind, "count": count})
				for i, key := range keys {
					block, flag := uint64(0), created[key]
					if isClaims {
						block, flag = claims[key].block, claims[key].consumed
					}
					want = historySemanticRootReference("semantic-index-entry", map[string]any{"index": kind, "previous": want, "ordinal": i, "key": key, "block": strconv.FormatUint(block, 10), "flag": flag})
				}
				m := providerHistorySemanticUnitMeter()
				if got := providerHistorySemanticIndexRootObserved(s, isClaims, max(1, count), m); got == "" || got != want || m.failed {
					t.Fatal("actual merge order or complete entry fields", kind, count, got, want)
				}
			}
			if count == 0 && providerHistorySemanticIndexRootObserved(s, true, 1, providerHistorySemanticUnitMeter()) == providerHistorySemanticIndexRootObserved(s, false, 1, providerHistorySemanticUnitMeter()) {
				t.Fatal("empty claim and created domains merged")
			}
		})
	}
}

func TestProviderHistorySemanticSnapshotOwnsWindowAndConfirmedState(t *testing.T) {
	f, in, m := historySemanticObservedFixture(t, false)
	historySemanticApply(t, f, in, 0, 1)
	historySemanticCompare(t, f, in, 0, 1)
	historySemanticApply(t, f, in, 2, 4)
	// The handwritten replay fixture has no expiry. This typed ownership-only
	// DATA pointer tests copying; it is not an authenticated command/PoP result.
	const expectedExpiry uint64 = 100
	expires := expectedExpiry
	f.window.run.state.record.Keys[0].Expires = &expires
	before := f.commitmentObserved(providerHistorySemanticUnitMeter())
	snapshot, ok := f.snapshotObserved(m)
	if !ok || before == "" || snapshot == f || snapshot.window == f.window || snapshot.window.run == f.window.run || snapshot.observer != m || snapshot.commitmentObserved(providerHistorySemanticUnitMeter()) != before {
		t.Fatal("owned complete semantic graph snapshot")
	}
	originalExpiry := f.window.run.state.record.Keys[0].Expires
	copiedExpiry := snapshot.window.run.state.record.Keys[0].Expires
	if originalExpiry == nil || copiedExpiry == nil || copiedExpiry == originalExpiry || *copiedExpiry != expectedExpiry {
		t.Fatal("typed snapshot ownership expiry precursor or pointer copy")
	}
	snapshot.window.run.state.record.Keys[0].State = "revoked"
	*copiedExpiry = 0
	snapshot.window.queue[0].command += " "
	snapshot.window.compared++
	for key, value := range snapshot.window.run.claims {
		value.consumed = false
		snapshot.window.run.claims[key] = value
	}
	snapshot.confirmed.created[replayTestAgent] = false
	if *originalExpiry != expectedExpiry || f.commitmentObserved(providerHistorySemanticUnitMeter()) != before {
		t.Fatal("candidate mutation changed an earlier original reader")
	}
	other := providerHistorySemanticUnitMeter()
	if copy, ok := f.snapshotObserved(other); ok || copy != nil || f.observer != m || f.failed {
		t.Fatal("read-only inspection transferred execution to a replacement observer")
	}
	if f.finishWindowCandidate(context.Background()) == nil || !f.failed || f.confirmed.state.version != 1 {
		t.Fatal("snapshot enabled unverified history publication")
	}
}

func TestProviderHistorySemanticCommitmentBindsActualStateIndexesAndQueue(t *testing.T) {
	f, in, m := historySemanticObservedFixture(t, false)
	historySemanticApply(t, f, in, 0, 3)
	root := f.commitmentObserved(providerHistorySemanticUnitMeter())
	if root == "" {
		t.Fatal("normal derived state/queue precursor")
	}
	for _, field := range []string{"claim-height", "claim-consumed", "created", "record", "queue-entry", "queue-command", "compared", "budget"} {
		t.Run(field, func(t *testing.T) {
			copy, ok := f.snapshotObserved(m)
			if !ok {
				t.Fatal("owned source snapshot precursor")
			}
			switch field {
			case "claim-height", "claim-consumed":
				for key, value := range copy.window.run.claims {
					if field == "claim-height" {
						value.block++
					} else {
						value.consumed = !value.consumed
					}
					copy.window.run.claims[key] = value
				}
			case "created":
				copy.window.run.created[replayTestAgent] = false
			case "record":
				copy.window.run.state.record.Controller = replayTestOperator
			case "queue-entry":
				copy.window.queue[0].entry += " "
			case "queue-command":
				copy.window.queue[0].command += " "
			case "compared":
				copy.window.compared++
			case "budget":
				copy.window.budget.bytes++
			}
			if copy.commitmentObserved(providerHistorySemanticUnitMeter()) == root || f.commitmentObserved(providerHistorySemanticUnitMeter()) != root {
				t.Fatal("changed owned semantic field was omitted or source aliased", field)
			}
		})
	}
}

func TestProviderHistorySemanticSnapshotRefusesQuotaAndKeepsPendingBytes(t *testing.T) {
	f, _, m := historySemanticObservedFixture(t, false)
	if f.beginWindowCandidate(context.Background()) != nil {
		t.Fatal("normal owned empty window")
	}
	c := replayTestCreate(t)
	if !f.window.prepareSubject(f.window.run, c) {
		t.Fatal("actual pending command precursor")
	}
	before := f.commitmentObserved(providerHistorySemanticUnitMeter())
	snapshot, ok := f.snapshotObserved(m)
	if !ok || before == "" || snapshot.window.pendingCommand != f.window.pendingCommand || snapshot.commitmentObserved(providerHistorySemanticUnitMeter()) != before {
		t.Fatal("pending command snapshot")
	}
	snapshot.window.pendingCommand += " "
	if f.commitmentObserved(providerHistorySemanticUnitMeter()) != before {
		t.Fatal("pending command owned bytes aliased")
	}
	// Same original observer with no remaining work; never install a new one.
	m.upper = m.actual
	if out, ok := f.snapshotObserved(m); ok || out != nil || !m.failed || f.commitmentObserved(providerHistorySemanticUnitMeter()) != before {
		t.Fatal("unfunded snapshot published or rewrote its source graph")
	}
	if !reflect.DeepEqual(f.window.queue, []providerHistoryExpectedItem(nil)) {
		t.Fatal("pending-only clone invented retained history")
	}
}

func TestProviderHistorySemanticSnapshotLateQuotaKeepsIndependentSourceGraph(t *testing.T) {
	prepare := func() (*providerHistorySemanticFold, *providerWorkMeter) {
		f, in, m := historySemanticObservedFixture(t, false)
		historySemanticApply(t, f, in, 0, 4)
		return f, m
	}
	probe, meter := prepare()
	start := meter.actual
	if copy, ok := probe.snapshotObserved(meter); !ok || copy == nil {
		t.Fatal("normal complete snapshot precursor")
	}
	work := meter.actual[10] - start[10]
	f, original := prepare()
	before := f.commitmentObserved(providerHistorySemanticUnitMeter())
	if before == "" || work == 0 {
		t.Fatal("actual source and nonempty copy allowance precursor")
	}
	original.upper[10] = original.actual[10] + work - 1
	if copy, ok := f.snapshotObserved(original); ok || copy != nil || !original.failed || f.commitmentObserved(providerHistorySemanticUnitMeter()) != before || f.observer != original {
		t.Fatal("late private copy failure published or rewrote an original graph")
	}
}

func TestProviderHistorySemanticCommitmentRejectsInvalidOwnedBounds(t *testing.T) {
	for _, field := range []string{"compared", "history-bytes", "pending", "budget", "empty-entry", "empty-command", "absent-record"} {
		t.Run(field, func(t *testing.T) {
			f, in, _ := historySemanticObservedFixture(t, false)
			historySemanticApply(t, f, in, 0, 1)
			switch field {
			case "compared":
				f.window.compared = -1
			case "history-bytes":
				f.window.historyBytes = f.window.budget.limits.History.MaxBytes + 1
			case "pending":
				f.window.pending = true
			case "budget":
				f.window.budget.proofs = f.window.budget.limits.MaxProofs + 1
			case "empty-entry":
				f.window.queue[0].entry = ""
			case "empty-command":
				f.window.queue[0].command = ""
			case "absent-record":
				f.confirmed.state.record.ID = "unexpected zero-version record"
			}
			if f.commitmentObserved(providerHistorySemanticUnitMeter()) != "" {
				t.Fatal("invalid owned state yielded a usable DATA root", field)
			}
		})
	}
	if providerHistorySemanticIndexRootObserved(replayStream{}, true, 0, providerHistorySemanticUnitMeter()) != "" || providerHistorySemanticIndexRootObserved(replayStream{}, false, 1<<20+1, providerHistorySemanticUnitMeter()) != "" {
		t.Fatal("invalid lifetime bounds")
	}
	s := replayStream{created: map[string]bool{"a": true, "b": true}}
	if providerHistorySemanticIndexRootObserved(s, false, 1, providerHistorySemanticUnitMeter()) != "" {
		t.Fatal("actual created lifetime exceeds declared bound")
	}
}
