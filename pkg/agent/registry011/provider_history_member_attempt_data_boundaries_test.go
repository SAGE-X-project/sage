package registry011

import (
	"context"
	"reflect"
	"testing"
)

func TestProviderHistoryPreparedMemberRejectsChangedAttemptBeforeFurtherObservation(t *testing.T) {
	o, d := historyMemberDispatchFixture(t, 1)
	s, m := o.store, o.meter
	s.mu.Lock()
	defer s.mu.Unlock()
	defer o.failLocked()
	prior := m.actual
	a, bounded, cancel, err := o.prepareMemberCandidateLocked(context.Background(), d, 0)
	if cancel != nil {
		defer cancel()
	}
	if err != nil || a == nil || bounded == nil || cancel == nil || a != d.attempt || a.owner != a ||
		!d.activeMemberShapeLocked() || !a.used || a.connection != nil || a.files != 0 || a.original != nil ||
		s.activeMeter.Load() != m || m.failed || m.closed || o.phase != "capturing-member" {
		t.Fatal("actual irreversibly consumed member preparation before any wire required", err)
	}
	if m.actual[0] != prior[0]+1 || m.actual[1] != prior[1] {
		t.Fatal("preparation must charge its original request while sending no wire")
	}
	before, reads, clocks, held := m.actual, s.clock.read, m.clocks, s.stored
	pin, ack, retention, order := o.pin, o.last, o.retention, *o.order
	originalMember := d.coordinator.ticket.table.members[0]
	originals := providerHistoryProvisionedOriginalBytes(t, s.config)
	// Only this test-owned attempt's private member DATA changes. No original
	// table, retained byte, selected profile, clock or allowance is replaced.
	a.member.request.method = "private-invalid-member-DATA"
	if d.activeMemberShapeLocked() || !a.used || a.connection != nil || a.files != 0 || a.original != nil ||
		m.actual != before || s.clock.read != reads || m.clocks != clocks || s.stored != held ||
		o.pin != pin || o.last != ack || o.retention != retention || *o.order != order ||
		d.coordinator.ticket.table.members[0] != originalMember || s.activeMeter.Load() != m || m.failed || m.closed ||
		!reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, s.config)) {
		t.Fatal("changed private attempt remained selected or the scalar refusal performed an effect")
	}
	// The deferred original failure cleanup is terminal. This test never
	// restores the member, retries the request, dials or publishes a result.
}
