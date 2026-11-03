package registry011

import (
	"context"
	"testing"
)

func TestProviderHistoryRecoveryInputNamesKeepTheirClosedPurposeAndObservedBudget(t *testing.T) {
	// This writer is unselected UNIT comparison DATA. It has no store, root,
	// original clock, active recovery lane or ability to read an arbitrary path.
	c := &providerHistoryRecoveryCheckpoint{}
	c.pin.ordinal = 7
	c.frame.after.sequence = 9
	for _, tc := range []struct {
		name string
		kind providerHistoryRecoveryFileKind
		want string
	}{
		{"head", providerHistoryRecoveryHead, "HEAD"},
		{"highwater", providerHistoryRecoveryHighwater, "highwater-00000000000000000007"},
		{"intent", providerHistoryRecoveryIntent, "intent-00000000000000000009"},
		{"commit", providerHistoryRecoveryCommit, "commit-00000000000000000009"},
		{"frontier", providerHistoryRecoveryFrontier, "frontier-00000000000000000009"},
		{"pending", providerHistoryRecoveryPending, "history-pending-00000000000000000009"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := historyRecoveryNameUnitWriter().recovery
			before := r.meter.actual
			if got := r.checkpointInputNameLocked(c, tc.kind, 0); got != tc.want ||
				r.meter.actual[10] <= before[10] || r.meter.failed || r.owner != nil || r.store != nil || r.active || r.used {
				t.Fatal("closed input purpose changed its ordinal or escaped observation", tc.name, got)
			}
		})
	}
	for _, kind := range []providerHistoryRecoveryFileKind{providerHistoryRecoveryHead, providerHistoryRecoveryIntent} {
		r := historyRecoveryNameUnitWriter().recovery
		r.meter.upper[10] = 0 // Selected before this separate UNIT invocation.
		before := r.meter.actual
		if got := r.checkpointInputNameLocked(c, kind, 0); got != "" || !r.meter.failed || r.meter.actual != before ||
			r.owner != nil || r.store != nil || r.active || r.used {
			t.Fatal("unfunded input purpose exposed a partial filename or gained a role", kind, got)
		}
		if r.checkpointInputNameLocked(c, kind, 0) != "" || r.meter.actual != before {
			t.Fatal("refused private filename renewed its observer", kind)
		}
	}
	r := historyRecoveryNameUnitWriter().recovery
	before := r.meter.actual
	if r.checkpointInputNameLocked(c, 0, 0) != "" || r.meter.actual != before ||
		r.inputIdentityLocked(&retentionDisk{}) || r.meter.actual != before || r.meter.failed ||
		r.owner != nil || r.store != nil || r.active || r.used {
		t.Fatal("unselected purpose or unrelated root gained an input role or spent observation")
	}
}

func TestProviderHistoryRecoveryInputWithoutAnOriginalStoreReturnsNoObservation(t *testing.T) {
	for _, r := range []*providerHistoryRecoveryOwner{nil, {}} {
		if out, err := r.readCheckpointInputCandidate(context.Background(), nil, providerHistoryRecoveryHead, 0); err == nil || out != nil {
			t.Fatal("missing original store supplied a file observation", err)
		}
	}
}
