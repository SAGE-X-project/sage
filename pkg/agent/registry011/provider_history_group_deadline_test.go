package registry011

import (
	"context"
	"math"
	"testing"
	"time"
)

func historyDeadlineSamples() (time.Time, ClockSample, ClockSample, ClockSample) {
	origin := time.Unix(100, 0)
	p := ClockSample{ClockID: "owned", Epoch: admissionTestDigest("a"), WallMS: 100000, MonotonicMS: 0, UncertaintyMS: 10}
	ticket, current := p, p
	ticket.WallMS, ticket.MonotonicMS = 100100, 100
	current.WallMS, current.MonotonicMS = 100250, 250
	return origin, p, ticket, current
}

func TestProviderHistoryTransportDeadlineKeepsAllOriginalCeilings(t *testing.T) {
	for _, row := range []struct {
		name       string
		lifetime   uint64
		member     time.Duration
		callerMS   int64
		expectedMS int64
	}{
		{"member", 1000000, 30 * time.Second, 0, 29250},
		{"group", 1000000, 5 * time.Minute, 0, 299100},
		{"parent", 10000, 30 * time.Second, 0, 9000},
		{"caller", 1000000, 30 * time.Second, 7000, 6000},
		{"fractional-member", 1000000, 30*time.Second + 900*time.Microsecond, 0, 29250},
	} {
		t.Run(row.name, func(t *testing.T) {
			origin, parent, ticket, current := historyDeadlineSamples()
			var caller time.Time
			if row.callerMS > 0 {
				caller = origin.Add(time.Duration(row.callerMS) * time.Millisecond)
			}
			deadline, ok := providerHistoryTransportDeadline(origin, parent, ticket, current, row.lifetime, row.member, time.Second, caller)
			if !ok || !deadline.Equal(origin.Add(time.Duration(row.expectedMS)*time.Millisecond)) {
				t.Fatal("one of the original ceilings was renewed or its S tail was omitted", deadline, ok)
			}
		})
	}
	// A later member keeps the SAME T and parent ceilings. This is pure DATA:
	// changing its current argument neither samples nor qualifies a backend.
	origin, parent, ticket, current := historyDeadlineSamples()
	first, ok := providerHistoryTransportDeadline(origin, parent, ticket, current, 1000000, 5*time.Minute, time.Second, time.Time{})
	current.MonotonicMS, current.WallMS = 50000, 150000
	later, laterOK := providerHistoryTransportDeadline(origin, parent, ticket, current, 1000000, 5*time.Minute, time.Second, time.Time{})
	if !ok || !laterOK || !first.Equal(later) {
		t.Fatal("member progression renewed the group ceiling")
	}
}

func TestProviderHistoryTransportDeadlineRefusesUnfundedOrOverflowedTime(t *testing.T) {
	for _, mode := range []string{"origin", "lifetime-zero", "lifetime-overflow", "member-zero", "member-long", "tail-zero", "tail-long", "submillisecond-member", "submillisecond-space", "epoch", "backward", "negative", "expired-parent", "expired-group", "offset-duration-overflow", "caller-expired", "caller-tail"} {
		t.Run(mode, func(t *testing.T) {
			origin, parent, ticket, current := historyDeadlineSamples()
			lifetime, member, tail, caller := uint64(1000000), 30*time.Second, time.Second, time.Time{}
			switch mode {
			case "origin":
				origin = time.Time{}
			case "lifetime-zero":
				lifetime = 0
			case "lifetime-overflow":
				parent.MonotonicMS, parent.WallMS = 100, 100100
				lifetime = math.MaxUint64
			case "member-zero":
				member = 0
			case "member-long":
				member = 5*time.Minute + time.Nanosecond
			case "tail-zero":
				tail = 0
			case "tail-long":
				tail = 5 * time.Minute
			case "submillisecond-member":
				member, tail = time.Microsecond, time.Nanosecond
			case "submillisecond-space":
				member, tail = 2*time.Millisecond, time.Millisecond
			case "epoch":
				current.Epoch = admissionTestDigest("b")
			case "backward":
				current.MonotonicMS = 99
			case "negative":
				parent.MonotonicMS = -1
			case "expired-parent":
				lifetime = 250
			case "expired-group":
				current.MonotonicMS, current.WallMS = 300100, 400100
			case "offset-duration-overflow":
				parent.MonotonicMS, parent.WallMS = math.MaxInt64/2, math.MaxInt64/2
				ticket, current = parent, parent
			case "caller-expired":
				caller = origin.Add(250 * time.Millisecond)
			case "caller-tail":
				caller = origin.Add(1250 * time.Millisecond)
			}
			if deadline, ok := providerHistoryTransportDeadline(origin, parent, ticket, current, lifetime, member, tail, caller); ok || !deadline.IsZero() {
				t.Fatal("invalid finite time DATA produced a dispatch window", deadline)
			}
		})
	}
}

func TestProviderHistoryDeadlinePreparationKeepsTheActualBackendAndLastRead(t *testing.T) {
	o, ack := historyDispatchReplayFixture(t, 1)
	d, err := o.prepareDispatchCandidate(context.Background(), ack)
	if err != nil {
		t.Fatal("genuine retained C preparation", err)
	}
	s, m, c := o.store, o.meter, o.parent.clock
	actual, reads, held, last := m.actual, c.read, s.stored, o.last
	s.mu.Lock()
	deadline, ok := d.deadlineCandidateLocked(0, time.Millisecond, time.Time{})
	s.mu.Unlock()
	if !ok || deadline.IsZero() {
		t.Fatal("otherwise-valid original backend refused pure deadline preparation")
	}
	for _, mode := range []string{"nil", "copied", "ordinal-negative", "ordinal-end", "changed-origin", "changed-claim", "unused-claim", "changed-last", "changed-ticket-clock", "copied-ack", "failed-meter", "wrong-phase"} {
		t.Run(mode, func(t *testing.T) {
			input, ordinal := d, 0
			origin, claim, used, clockLast := c.origin, o.parent.clockClaim, o.parent.clockClaim.used, c.last
			ticketClock, inputAck, failed, phase := o.admission.claim.frame.clock, d.acknowledgment, m.failed, o.phase
			switch mode {
			case "nil":
				input = nil
			case "copied":
				copy := *d
				input = &copy
			case "ordinal-negative":
				ordinal = -1
			case "ordinal-end":
				ordinal = d.coordinator.ticket.table.count
			case "changed-origin":
				c.origin = c.origin.Add(time.Nanosecond)
			case "changed-claim":
				copy := *claim
				o.parent.clockClaim = &copy
			case "unused-claim":
				claim.used = false
			case "changed-last":
				c.last.MonotonicMS++
			case "changed-ticket-clock":
				o.admission.claim.frame.clock.MonotonicMS++
			case "copied-ack":
				copy := *inputAck
				d.acknowledgment = &copy
			case "failed-meter":
				m.failed = true
			case "wrong-phase":
				o.phase = "consumed"
			}
			s.mu.Lock()
			got, valid := input.deadlineCandidateLocked(ordinal, time.Millisecond, time.Time{})
			s.mu.Unlock()
			c.origin, c.last, o.parent.clockClaim, claim.used = origin, clockLast, claim, used
			o.admission.claim.frame.clock, d.acknowledgment, m.failed, o.phase = ticketClock, inputAck, failed, phase
			if valid || !got.IsZero() || m.actual != actual || c.read != reads || s.stored != held || o.last != last || s.activeMeter.Load() != nil {
				t.Fatal("detached deadline input observed or renewed the original budget")
			}
		})
	}
}
