package registry011

import (
	"math"
	"time"
)

// Pure deadline DATA. The caller must separately select the original native
// backend, original T and current sample, and consume its one-use attempt.
// This does not sample a clock, install a timer, renew a window or grant I/O.
// The tail is a positively derived S envelope, not an optional caller delay.
func providerHistoryTransportDeadline(origin time.Time, parent, ticket, current ClockSample, lifetimeMS uint64, member, tail time.Duration, caller time.Time) (time.Time, bool) {
	if origin.IsZero() || lifetimeMS == 0 || member <= 0 || member > 5*time.Minute || tail <= 0 || tail >= 5*time.Minute ||
		!candidateLedgerClockAdvance(providerClockDiagnostic(parent), providerClockDiagnostic(ticket)) ||
		!candidateLedgerClockAdvance(providerClockDiagnostic(ticket), providerClockDiagnostic(current)) ||
		parent.MonotonicMS < 0 || ticket.MonotonicMS < 0 || current.MonotonicMS < 0 {
		return time.Time{}, false
	}
	groupEnd, a := providerUpperSum(uint64(ticket.MonotonicMS), 300000)
	parentEnd, b := providerUpperSum(uint64(parent.MonotonicMS), lifetimeMS)
	// Truncate toward an EARLIER deadline. A submillisecond member never
	// becomes a fresh full millisecond and cannot finance a positive tail.
	memberEnd, c := providerUpperSum(uint64(current.MonotonicMS), uint64(member/time.Millisecond))
	endMS := min(groupEnd, parentEnd, memberEnd)
	if !a || !b || !c || endMS > uint64(math.MaxInt64/int64(time.Millisecond)) ||
		uint64(current.MonotonicMS) >= endMS {
		return time.Time{}, false
	}
	end := origin.Add(time.Duration(endMS) * time.Millisecond)
	if !caller.IsZero() && caller.Before(end) {
		end = caller
	}
	deadline := end.Add(-tail)
	currentLower := origin.Add(time.Duration(current.MonotonicMS) * time.Millisecond)
	// The retained native sample has millisecond precision. Reserve that
	// entire truncation interval rather than claiming a submillisecond slot.
	if deadline.Sub(currentLower) <= time.Millisecond {
		return time.Time{}, false
	}
	return deadline, true
}

// No native read occurs here. The parent must still own its unchanged backend
// and the current sample must be the last actually consumed original claim.
// Caller serializes the group with store.mu; this method only locks clock.mu.
// It is preparation DATA inside an unfinished transport/S bundle, not a permit.
func (d *providerHistoryGroupDispatch) deadlineCandidateLocked(ordinal int, tail time.Duration, caller time.Time) (time.Time, bool) {
	if d == nil || d.owner != d || d.group == nil || d.group.dispatch != d || d.acknowledgment == nil ||
		d.group.phase != "dispatch-prepared" || !d.group.consumptionShapeLocked() || d.group.last != d.acknowledgment ||
		d.group.order != d.order || d.group.meter != d.meter || d.meter == nil || d.meter.owner != d.meter || d.meter.failed || d.meter.closed ||
		d.group.store == nil || d.group.store.activeMeter.Load() != nil || d.group.parent == nil || !d.group.parent.matchesLocked() ||
		d.coordinator == nil || d.custodian == nil || d.coordinator == d.custodian || d.coordinator.ticket == nil ||
		d.custodian.ticket == nil || d.coordinator.ticket == d.custodian.ticket || d.coordinator.ticket.table.count <= 0 ||
		d.coordinator.ticket.table.count > providerHistoryGroupMembers ||
		ordinal < 0 || ordinal >= d.coordinator.ticket.table.count {
		return time.Time{}, false
	}
	p, c := d.group.parent, d.group.parent.clock
	c.mu.Lock()
	defer c.mu.Unlock()
	claim := p.clockClaim
	if c.owner != c || c.failed || c.historyRevoked || c.history != p || !p.clockSelectionMatches(c) ||
		claim == nil || claim.owner != claim || claim.parent != p || claim.meter != d.meter || !claim.used ||
		claim.expectedRead == math.MaxUint64 || c.read != claim.expectedRead+1 || c.last != d.end ||
		d.group.admission.claim.frame.clock != d.ticketClock {
		return time.Time{}, false
	}
	return providerHistoryTransportDeadline(c.origin, p.original, ClockSample(d.ticketClock), c.last,
		p.bounds.durationMS, d.coordinator.ticket.table.members[ordinal].capture.Timeout, tail, caller)
}
