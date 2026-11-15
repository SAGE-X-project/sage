package registry011

import "reflect"

// These are the immutable ORIGINAL inputs consumed by the current T/C full
// readers. Checkpoint.claim alone binds a pointer, not these pointed values.
// Catalog maps and pending-write payloads are not reader inputs. Live order
// progress, retention, meter actuals and clock counters must not be frozen.
type providerHistoryMemberClaimInput struct {
	ack                *providerHistoryGroupAcknowledgment
	ackValue           providerHistoryGroupAcknowledgment
	claim, owner       *providerHistoryWriteClaim
	group              *providerHistoryGroupOwner
	meter              *providerWorkMeter
	frame              *providerHistoryCustodyFrame
	frameValue         providerHistoryCustodyFrame
	inventory          *providerHistoryInventory
	inventoryValue     providerHistoryInventory
	count, members     int
	highwater, purpose string
	request, ticket    providerHistoryStoredReference
	consumption        providerHistoryStoredReference
	upper              providerHistoryGroupUpper
	coordinator        *providerHistoryGroupLedger
	coordinatorOwner   *providerHistoryGroupLedger
	coordinatorHead    string
	coordinatorTicket  *providerHistoryGroupTicket
	tableDigest        string
	admission          *providerHistoryGroupAcknowledgment
	order              *providerHistoryGroupOrder
	checkpoint         *providerHistoryRecoveryCheckpoint
	used, renamed      bool
}

// Only explicit observed boundaries call this; it is never a scalar active
// predicate. The fixed typed copy is charged before accessing pointed values.
func providerHistoryMemberClaimInputObservedLocked(a *providerHistoryGroupAcknowledgment, m *providerWorkMeter) (providerHistoryMemberClaimInput, bool) {
	if a == nil || a.claim == nil || m == nil {
		return providerHistoryMemberClaimInput{}, false
	}
	c := a.claim
	if c.owner != c || c.group == nil || a.owner != c.group || c.meter != m || c.frame == nil || c.inventory == nil ||
		c.coordinator == nil || c.coordinator.owner != c.coordinator || c.coordinator.ticket == nil ||
		!m.event(providerCopyEvent, uint64(reflect.TypeFor[providerHistoryMemberClaimInput]().Size())) {
		return providerHistoryMemberClaimInput{}, false
	}
	return providerHistoryMemberClaimInput{ack: a, ackValue: *a, claim: c, owner: c.owner, group: c.group, meter: c.meter,
		frame: c.frame, frameValue: *c.frame, inventory: c.inventory, inventoryValue: *c.inventory,
		count: c.count, members: c.members, highwater: c.highwater, purpose: c.purpose,
		request: c.request, ticket: c.ticket, consumption: c.consumption, upper: c.upper,
		coordinator: c.coordinator, coordinatorOwner: c.coordinator.owner, coordinatorHead: c.coordinator.head,
		coordinatorTicket: c.coordinator.ticket, tableDigest: c.coordinator.ticket.table.digest,
		admission: c.admission, order: c.order, checkpoint: c.checkpoint, used: c.used, renamed: c.renamed}, true
}

func (q *providerHistoryMemberReplaySeal) captureClaimsObservedLocked(m *providerWorkMeter) bool {
	if q == nil || q.dispatch == nil || q.dispatch.group == nil || m == nil {
		return false
	}
	o := q.dispatch.group
	for i, ack := range [2]*providerHistoryGroupAcknowledgment{o.admission, o.last} {
		value, ok := providerHistoryMemberClaimInputObservedLocked(ack, m)
		if !ok {
			return false
		}
		weight, ok := providerHistoryMemberComparisonWeightObserved(value, m)
		if !ok {
			return false
		}
		q.claims[i], q.claimComparisons[i] = value, weight
	}
	return q.claimsMatchObservedLocked(m)
}

func (q *providerHistoryMemberReplaySeal) claimsMatchObservedLocked(m *providerWorkMeter) bool {
	if q == nil || q.dispatch == nil || q.dispatch.group == nil || m == nil {
		return false
	}
	o := q.dispatch.group
	for i, ack := range [2]*providerHistoryGroupAcknowledgment{o.admission, o.last} {
		actual, ok := providerHistoryMemberClaimInputObservedLocked(ack, m)
		if !ok || !providerHistoryMemberEqualObserved(actual, q.claims[i], q.claimComparisons[i], m) {
			return false
		}
	}
	return true
}
