package registry011

import "reflect"

// A value snapshot binds ALL non-byte inputs consumed by the original full
// replay, not merely the immediate previous checkpoint. Pointed-to semantic
// children are copied separately; equality never treats a pointer as its data.
type providerHistoryMemberAncestorValue struct {
	checkpoint providerHistoryRecoveryCheckpoint
	initial    providerHistoryRecoveryInitialDescriptor
	admission  providerHistoryRecoveryAdmissionDescriptor
	archive    providerHistoryRecoveryMemberArchive
	order      providerHistoryGroupOrder
}

type providerHistoryMemberAncestorFact struct {
	complete providerHistoryRecoveryMemberComplete
	capture  providerHistoryRecoveryMemberCapture
	intent   providerHistoryRecoveryMemberAttempt
	attempt  providerHistoryDispatchAttempt
	original providerHistoryMemberOriginal
}

type providerHistoryMemberAncestor struct {
	value      providerHistoryMemberAncestorValue
	comparison uint64
	members    []providerHistoryMemberAncestorFact
	weights    []uint64
}

func providerHistoryMemberAncestorValueLocked(c *providerHistoryRecoveryCheckpoint) (providerHistoryMemberAncestorValue, bool) {
	if c == nil || c.owner != c {
		return providerHistoryMemberAncestorValue{}, false
	}
	v := providerHistoryMemberAncestorValue{checkpoint: *c}
	if c.descriptor != nil {
		v.initial = *c.descriptor
	}
	if c.admission != nil {
		v.admission = *c.admission
	}
	if c.memberArchive != nil {
		if c.memberArchive.order == nil {
			return providerHistoryMemberAncestorValue{}, false
		}
		v.archive, v.order = *c.memberArchive, *c.memberArchive.order
	}
	return v, true
}

func providerHistoryMemberAncestorFactLocked(n *providerHistoryRecoveryMemberComplete) (providerHistoryMemberAncestorFact, bool) {
	if n == nil || n.owner != n || n.capture == nil || n.capture.intent == nil || n.capture.intent.attempt == nil || n.capture.original == nil {
		return providerHistoryMemberAncestorFact{}, false
	}
	return providerHistoryMemberAncestorFact{*n, *n.capture, *n.capture.intent, *n.capture.intent.attempt, *n.capture.original}, true
}

// Only the still healthy constructor uses this. Exact arrays avoid implicit
// append growth. Copy/compare events are logical; complete backing/native RAM
// and lifetime qualification is still a separate whole-program obligation.
func (q *providerHistoryMemberReplaySeal) captureAncestorsObservedLocked(m *providerWorkMeter) bool {
	if q == nil || m == nil || q.recovery == nil || q.dispatch == nil || q.recovery.initial == nil || len(q.ancestors) != 0 {
		return false
	}
	maximum, ok := providerUpperSum(q.fixed.bounds.records, 1)
	if !ok || maximum > uint64(^uint(0)>>1) {
		return false
	}
	q.initial = q.recovery.initial
	var count uint64
	for c := q.recovery.consumption; ; c = c.previous {
		if c == nil || c.owner != c || count >= maximum || !m.event(providerCompareEvent, 0) {
			return false
		}
		count++
		if c == q.initial {
			break
		}
	}
	slots, ok := providerUpperProduct(count, uint64(reflect.TypeFor[providerHistoryMemberAncestor]().Size()))
	if !ok || !m.event(providerCopyEvent, slots) {
		return false
	}
	q.ancestors = make([]providerHistoryMemberAncestor, int(count))
	c := q.recovery.consumption
	for i := range q.ancestors {
		row := &q.ancestors[i]
		row.value, ok = providerHistoryMemberAncestorValueLocked(c)
		if !ok {
			return false
		}
		row.comparison, ok = providerHistoryMemberComparisonWeightObserved(row.value, m)
		if !ok {
			return false
		}
		if c.memberArchive != nil {
			a := c.memberArchive
			if !a.shapeLocked(q.recovery, c) || a.count < 1 || a.count > providerHistoryGroupMembers {
				return false
			}
			bytes, added := providerUpperProduct(uint64(a.count), uint64(reflect.TypeFor[providerHistoryMemberAncestorFact]().Size()+8))
			if !added || !m.event(providerCopyEvent, bytes) {
				return false
			}
			row.members, row.weights = make([]providerHistoryMemberAncestorFact, a.count), make([]uint64, a.count)
			for j := range row.members {
				row.members[j], ok = providerHistoryMemberAncestorFactLocked(a.members[j])
				if !ok {
					return false
				}
				row.weights[j], ok = providerHistoryMemberComparisonWeightObserved(row.members[j], m)
				if !ok {
					return false
				}
			}
		}
		c = c.previous
	}
	return q.ancestorsMatchObservedLocked(m)
}

// Explicit observed boundary, NEVER called from scalar activeMemberShape or
// while meter.mu is held. The original complete chain and every original
// archived member fact must still equal its value at the full replay.
func (q *providerHistoryMemberReplaySeal) ancestorsMatchObservedLocked(m *providerWorkMeter) bool {
	if q == nil || m == nil || q.recovery == nil || q.initial == nil || q.recovery.initial != q.initial ||
		q.recovery.initialDescriptor != q.initial.descriptor || len(q.ancestors) == 0 {
		return false
	}
	maximum, ok := providerUpperSum(q.fixed.bounds.records, 1)
	if !ok || uint64(len(q.ancestors)) > maximum {
		return false
	}
	c := q.recovery.consumption
	for i := range q.ancestors {
		row := &q.ancestors[i]
		if c == nil || c != row.value.checkpoint.owner {
			return false
		}
		actual, ok := providerHistoryMemberAncestorValueLocked(c)
		if !ok || !providerHistoryMemberEqualObserved(actual, row.value, row.comparison, m) {
			return false
		}
		if len(row.members) > providerHistoryGroupMembers || len(row.members) != len(row.weights) || c.memberArchive == nil && len(row.members) != 0 ||
			c.memberArchive != nil && len(row.members) != c.memberArchive.count {
			return false
		}
		for j, expected := range row.members {
			n := c.memberArchive.members[j]
			actual, ok := providerHistoryMemberAncestorFactLocked(n)
			if !ok || n != expected.complete.owner || !providerHistoryMemberEqualObserved(actual, expected, row.weights[j], m) {
				return false
			}
		}
		if c == q.initial {
			return i == len(q.ancestors)-1
		}
		c = c.previous
	}
	return false
}
