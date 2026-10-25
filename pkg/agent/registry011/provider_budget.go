package registry011

import (
	"errors"
	"sync"
)

var ErrCandidateBudget = errors.New("registry011: candidate budget unavailable")
var ErrCandidateBudgetLimit = errors.New("registry011: candidate budget limit")
var ErrCandidateBudgetTicket = errors.New("registry011: invalid candidate budget ticket")

// CandidateWork is a proposed full work vector. It is not an owned measurement
// or permission to perform work. Zero means inapplicable, not an omitted bound.
// The future qualified estimator must include metadata and repeated work.
type CandidateWork struct {
	Requests, WireBytes, DecodedBytes, Blocks, Transactions, Frames uint64
	Logs, Proofs, Segments, DiskBytesWritten, WorkUnits             uint64
}

type candidateWorkVector [11]uint64

func (w CandidateWork) vector() candidateWorkVector {
	return candidateWorkVector{w.Requests, w.WireBytes, w.DecodedBytes, w.Blocks, w.Transactions, w.Frames, w.Logs, w.Proofs, w.Segments, w.DiskBytesWritten, w.WorkUnits}
}

func (v candidateWorkVector) diagnostic() CandidateWork {
	return CandidateWork{v[0], v[1], v[2], v[3], v[4], v[5], v[6], v[7], v[8], v[9], v[10]}
}

type candidateBudgetTotals struct{ charged, outstanding candidateWorkVector }

// CandidateBudgetModel is an ephemeral, conservative accounting model, not a
// retaining authority, durable ledger, clock check or qualified admission API.
// Recreating it loses state; it MUST NOT be used to reset operational counters.
// No Source, capture or replay API accepts the model or its tickets. There is
// deliberately no success/refund, GC, restore or operating activation method.
// All capacity values are conservative upper bounds, not measured live bytes.
type CandidateBudgetModel struct {
	mu                        sync.Mutex
	owner                     *CandidateBudgetModel
	policy                    string
	limits                    [3]candidateWorkVector
	liveLimit, retainLimit    uint64
	maxTickets                int
	bootstrap                 candidateBudgetTotals
	segments                  map[string]candidateBudgetTotals
	tickets                   []candidateBudgetRecord
	liveUsed, livePending     uint64
	retainUsed, retainPending uint64
}

type candidateBudgetRecord struct {
	segment        string
	work           candidateWorkVector
	live, retained uint64
	outcome        string
}

// CandidateBudgetTicket is private owner-bound model state, not a work permit.
// Its zero value and JSON restoration refuse; another model cannot settle it.
type CandidateBudgetTicket struct {
	owner *CandidateBudgetModel
	index int
}

// CandidateBudgetDiagnostic contains copied, forgeable model results only.
// Used includes conservatively charged outcomes; Pending is still reserved.
// These numbers do not establish exact storage use or completed work.
type CandidateBudgetDiagnostic struct {
	Policy                                                                     string
	Charged, Outstanding                                                       CandidateWork
	LiveUsedUpper, LiveReservedUpper, RetainedUsedUpper, RetainedReservedUpper uint64
	Tickets, Segments                                                          int
}

func (*CandidateBudgetModel) UnmarshalJSON([]byte) error  { return ErrCandidateBudget }
func (*CandidateBudgetTicket) UnmarshalJSON([]byte) error { return ErrCandidateBudgetTicket }

// NewCandidateBudgetModel copies only a previously parsed candidate's limits.
// maxTickets explicitly bounds the model's memory (1..4096), not an operating
// workload. Parsed digests and model arithmetic qualify no selected artifacts.
// Each ticket represents one proposed complete operation; splitting a real
// operation into tickets must be prevented by the future owned integration.
func NewCandidateBudgetModel(policy *ProviderPolicyCandidate, maxTickets int) (*CandidateBudgetModel, error) {
	if policy == nil || policy.raw == "" || maxTickets <= 0 || maxTickets > 4096 {
		return nil, ErrConfiguration
	}
	root := contentJSON([]byte(policy.raw)).(map[string]any)
	limits := root["limits"].(map[string]any)
	m := &CandidateBudgetModel{policy: policy.policy, maxTickets: maxTickets, segments: make(map[string]candidateBudgetTotals)}
	m.owner = m
	names := [...]string{"requests", "wire_bytes", "decoded_bytes", "blocks", "transactions", "frames", "logs", "proofs", "segments", "disk_bytes_written", "work_units"}
	for level, name := range []string{"operation", "segment", "bootstrap"} {
		fields := limits[name].(map[string]any)
		for i, key := range names {
			m.limits[level][i], _ = providerCandidateUint(fields[key], true)
		}
	}
	m.liveLimit, _ = providerCandidateUint(limits["live_disk_bytes"], true)
	m.retainLimit, _ = providerCandidateUint(limits["retained_bytes"], true)
	return m, nil
}

// ReserveCandidate atomically models all dimensions and both capacity bounds.
// The first ticket for each segment must include Segments=1; later tickets for
// that same segment require 0. Segment histories never disappear on failure.
// No zero/default segment, automatic retry, network, file write or work runs.
func (m *CandidateBudgetModel) ReserveCandidate(segment string, proposed CandidateWork, livePeak, retainedPeak uint64) (*CandidateBudgetTicket, error) {
	if m == nil {
		return nil, ErrCandidateBudget
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.valid() {
		return nil, ErrCandidateBudget
	}
	if !lowerHash(segment) || retainedPeak > livePeak {
		return nil, ErrConfiguration
	}
	previous, exists := m.segments[segment]
	if (exists && proposed.Segments != 0) || (!exists && proposed.Segments != 1) {
		return nil, ErrCandidateBudgetTicket
	}
	work := proposed.vector()
	if len(m.tickets) >= m.maxTickets || !candidateFits(candidateBudgetTotals{}, work, m.limits[0]) ||
		!candidateFits(previous, work, m.limits[1]) || !candidateFits(m.bootstrap, work, m.limits[2]) ||
		!candidateCapacity(m.liveUsed, m.livePending, livePeak, m.liveLimit) ||
		!candidateCapacity(m.retainUsed, m.retainPending, retainedPeak, m.retainLimit) {
		return nil, ErrCandidateBudgetLimit
	}
	for i, n := range work {
		previous.outstanding[i] += n
		m.bootstrap.outstanding[i] += n
	}
	m.livePending += livePeak
	m.retainPending += retainedPeak
	m.segments[segment] = previous
	index := len(m.tickets)
	m.tickets = append(m.tickets, candidateBudgetRecord{segment: segment, work: work, live: livePeak, retained: retainedPeak})
	return &CandidateBudgetTicket{m, index}, nil
}

// ChargeCandidate models failure/cancellation/uncertainty by charging the full
// original vector and retaining the full capacity upper bounds. No caller-
// reported actual work can refund a reservation. Same outcome is idempotent;
// a conflicting outcome refuses. This cannot confirm a real operation's end.
func (m *CandidateBudgetModel) ChargeCandidate(ticket *CandidateBudgetTicket, outcome string) error {
	if m == nil {
		return ErrCandidateBudget
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.valid() {
		return ErrCandidateBudget
	}
	if ticket == nil || ticket.owner != m || ticket.index < 0 || ticket.index >= len(m.tickets) ||
		(outcome != "failed" && outcome != "cancelled" && outcome != "uncertain") {
		return ErrCandidateBudgetTicket
	}
	r := &m.tickets[ticket.index]
	if r.outcome != "" {
		if r.outcome == outcome {
			return nil
		}
		return ErrCandidateBudgetTicket
	}
	segment := m.segments[r.segment]
	for i, n := range r.work {
		segment.outstanding[i] -= n
		segment.charged[i] += n
		m.bootstrap.outstanding[i] -= n
		m.bootstrap.charged[i] += n
	}
	m.segments[r.segment] = segment
	m.livePending -= r.live
	m.liveUsed += r.live
	m.retainPending -= r.retained
	m.retainUsed += r.retained
	r.outcome = outcome
	return nil
}

func (m *CandidateBudgetModel) Diagnostic() (CandidateBudgetDiagnostic, error) {
	if m == nil {
		return CandidateBudgetDiagnostic{}, ErrCandidateBudget
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.valid() {
		return CandidateBudgetDiagnostic{}, ErrCandidateBudget
	}
	return CandidateBudgetDiagnostic{m.policy, m.bootstrap.charged.diagnostic(), m.bootstrap.outstanding.diagnostic(), m.liveUsed, m.livePending, m.retainUsed, m.retainPending, len(m.tickets), len(m.segments)}, nil
}

func (m *CandidateBudgetModel) valid() bool {
	return m.owner == m && m.policy != "" && m.maxTickets > 0 && m.segments != nil
}

// Subtract from a ceiling instead of adding untrusted u64 values. These checks
// cover overflow even if a declared ceiling is MaxUint64.
func candidateCapacity(charged, outstanding, proposed, ceiling uint64) bool {
	return charged <= ceiling && outstanding <= ceiling-charged && proposed <= ceiling-charged-outstanding
}

func candidateFits(totals candidateBudgetTotals, work, ceiling candidateWorkVector) bool {
	for i, n := range work {
		if !candidateCapacity(totals.charged[i], totals.outstanding[i], n, ceiling[i]) {
			return false
		}
	}
	return true
}
