package registry011

import "sync"

type providerWorkEvent uint8

const (
	providerReadEvent providerWorkEvent = iota
	providerWriteEvent
	providerCodecEvent
	providerHashEvent
	providerCopyEvent
	providerClockEvent
	providerSyncEvent
	providerTLSEvent
	providerEncodeEvent
	providerRenameEvent
	providerCompareEvent
)

// A private meter counts actual invocations/bytes under the pinned logical
// event metric. It never accepts a reported actual vector or returns authority.
// Unit weights are not measured CPU instructions or filesystem allocation.
type providerWorkMeter struct {
	mu                            sync.Mutex
	owner                         *providerWorkMeter
	integration                   *providerIntegration
	scan                          *providerScanControl
	clocks, maximumClocks         uint64
	ticket                        string
	upper, remaining, reservation candidateWorkVector
	actual                        candidateWorkVector
	closed, failed, claimed       bool
	end                           ClockSample
	parent                        *providerWorkMeter
	authority                     bool
}

type providerMeasuredWork struct{ meter *providerWorkMeter }

func (*providerWorkMeter) UnmarshalJSON([]byte) error    { return errProviderIntegration }
func (*providerMeasuredWork) UnmarshalJSON([]byte) error { return errProviderIntegration }

func newProviderWorkMeter(owner *providerIntegration, reservation providerOperationReservation) *providerWorkMeter {
	m := &providerWorkMeter{integration: owner, upper: reservation.execution, remaining: reservation.remaining, reservation: reservation.work}
	m.owner = m
	return m
}

func (m *providerWorkMeter) add(work candidateWorkVector) bool {
	if m == nil {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.owner != m || !m.hasOwner() || m.failed || m.closed {
		return false
	}
	var next candidateWorkVector
	for i, n := range work {
		if !candidateCapacity(0, m.actual[i], n, m.upper[i]) {
			m.failed = true
			return false
		}
		next[i] = m.actual[i] + n
	}
	m.actual = next
	return true
}

// Event arithmetic uses the actual bounded input size, not the ticket upper.
// A codec unit is a pinned weighted invocation n*n+1; other byte primitives
// use n+1, and actual clock/sync calls use1. These logical units still need
// independent worst-case CPU/profile qualification before operating selection.
func (m *providerWorkMeter) event(kind providerWorkEvent, bytes uint64) bool {
	if kind == providerClockEvent && m != nil && m.scan != nil {
		m.mu.Lock()
		if m.maximumClocks == 0 || m.clocks >= m.maximumClocks || m.closed || m.failed {
			m.failed = true
			m.mu.Unlock()
			return false
		}
		m.clocks++
		m.mu.Unlock()
	}
	work, ok := providerEventVector(kind, bytes)
	if !ok {
		return m.refuse()
	}
	return m.add(work)
}

func (m *providerWorkMeter) preflight(kind providerWorkEvent, bytes uint64) bool {
	if m == nil {
		return false
	}
	work, ok := providerEventVector(kind, bytes)
	if !ok {
		return m.refuse()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.owner != m || !m.hasOwner() || m.closed || m.failed {
		return false
	}
	for i, n := range work {
		if !candidateCapacity(0, m.actual[i], n, m.upper[i]) {
			m.failed = true
			return false
		}
	}
	return true
}

func (m *providerWorkMeter) hasOwner() bool {
	return m.integration != nil && m.scan == nil || m.integration == nil && m.scan != nil && m.scan.owner == m.scan
}

func providerEventVector(kind providerWorkEvent, bytes uint64) (candidateWorkVector, bool) {
	var work candidateWorkVector
	units, ok := providerUpperSum(bytes, 1)
	if !ok {
		return work, false
	}
	switch kind {
	case providerReadEvent, providerHashEvent, providerCopyEvent, providerTLSEvent, providerCompareEvent:
	case providerWriteEvent:
		work[9] = bytes
	case providerCodecEvent, providerEncodeEvent:
		units, ok = providerUpperProduct(bytes, bytes)
		if !ok {
			return work, false
		}
		units, ok = providerUpperSum(units, 1)
		if !ok {
			return work, false
		}
		if kind == providerCodecEvent {
			work[2] = bytes
		}
	case providerClockEvent, providerSyncEvent, providerRenameEvent:
		if bytes != 0 {
			return work, false
		}
		units = 1
	default:
		return work, false
	}
	work[10] = units
	return work, true
}

func (m *providerWorkMeter) refuse() bool {
	if m != nil {
		m.mu.Lock()
		m.failed = true
		m.mu.Unlock()
	}
	return false
}

func (m *providerWorkMeter) finish(owner *providerIntegration, ticket *CandidateAdmissionTicket, clock ClockSample) (*providerMeasuredWork, error) {
	if m == nil || ticket == nil {
		return nil, errProviderIntegration
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.owner != m || m.scan != nil || m.integration != owner || m.authority || m.failed || m.closed || m.ticket != ticket.digest || ticket.work != m.reservation || m.actual[8] != ticket.work[8] || m.actual[9] == 0 || m.actual[10] == 0 || !candidateLedgerClockAdvance(ticket.intent.clock, providerClockDiagnostic(clock)) || candidateLedgerExpired(providerClockDiagnostic(clock), ticket.deadline) {
		m.failed = true
		return nil, errProviderIntegration
	}
	for i := range m.actual {
		if !candidateCapacity(0, m.actual[i], m.remaining[i], m.reservation[i]) {
			m.failed = true
			return nil, errProviderIntegration
		}
	}
	m.closed, m.end = true, clock
	return &providerMeasuredWork{m}, nil
}

func (m *providerMeasuredWork) claim(owner *providerIntegration, ticket *CandidateAdmissionTicket, clock ClockSample) (candidateWorkVector, bool) {
	if m == nil || m.meter == nil || ticket == nil {
		return candidateWorkVector{}, false
	}
	v := m.meter
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.owner != v || v.scan != nil || v.integration != owner || v.failed || !v.closed || v.claimed || v.ticket != ticket.digest || v.end != clock {
		return candidateWorkVector{}, false
	}
	v.claimed = true
	return v.actual, true
}
