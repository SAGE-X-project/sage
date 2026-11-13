package registry011

import (
	"context"
	"reflect"
	"testing"
)

// Missing/unselected UNIT DATA has no store, observer, root or current C.
// Refusing it cannot allocate a replacement seal or return retained bytes.
func TestProviderHistoryMemberReplayRequiresItsOriginalDispatchAndReader(t *testing.T) {
	for _, d := range []*providerHistoryGroupDispatch{nil, {}, {group: &providerHistoryGroupOwner{}}} {
		if q, err := newProviderHistoryMemberReplaySealLocked(d); err == nil || q != nil {
			t.Fatal("missing original preparation supplied a member replay seal")
		}
		if out, err := d.replayMemberEquivalentLocked(context.Background(), 0); err == nil || out != nil {
			t.Fatal("unselected dispatch supplied a semantic member replay")
		}
	}
	for _, q := range []*providerHistoryMemberReplaySeal{nil, {}, {dispatch: &providerHistoryGroupDispatch{group: &providerHistoryGroupOwner{}}}} {
		if root, err := q.currentConsumptionRootLocked(); err == nil || root != "" {
			t.Fatal("missing original consumption supplied a current root")
		}
		if q.readMatchesLocked(nil, 0) || q.readMatchesLocked(&providerHistoryReplayReads{}, 1) {
			t.Fatal("missing original reader was accepted as a complete pass")
		}
	}
}

func TestProviderHistoryUnrelatedOwnerCannotRevokeRegisteredMemberReplay(t *testing.T) {
	var absent *providerHistoryGroupOwner
	absent.releaseMemberReplayLocked()
	(&providerHistoryGroupOwner{}).releaseMemberReplayLocked()
	o, d := historyMemberDispatchFixture(t, 1)
	s, r, q := o.store, o.parent.recovery, d.memberReplaySeal
	originals := providerHistoryProvisionedOriginalBytes(t, s.config)
	s.mu.Lock()
	defer s.mu.Unlock()
	if q == nil || q.revoked || !d.memberReplaySelectedLocked() || r.memberReplaySeal != q || r.originalGroup != o {
		t.Fatal("genuine registered original member replay prerequisite")
	}
	work, reads, pin, ack, held := o.meter.actual, s.clock.read, o.pin, o.last, s.stored
	readers, models, beforeReads := q.reads, q.models, d.replayReads
	// A different unselected owner merely points at the same parent DATA.
	// It gains neither originalGroup identity nor a live observer/dispatch.
	other := &providerHistoryGroupOwner{parent: o.parent}
	other.releaseMemberReplayLocked()
	if q.revoked || q.reads != readers || q.models != models || d.replayReads != beforeReads ||
		d.memberReplaySeal != q || r.memberReplaySeal != q || r.originalDispatch != d || !d.memberReplaySelectedLocked() ||
		o.meter.actual != work || s.clock.read != reads || o.pin != pin || o.last != ack || s.stored != held || s.activeMeter.Load() != nil ||
		other.owner != nil || other.meter != nil || other.dispatch != nil || r.originalGroup != o ||
		!reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, s.config)) {
		t.Fatal("unrelated owner revoked original replay DATA, changed originals or gained an execution role")
	}
}
