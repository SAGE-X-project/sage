package registry011

import (
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"sync"
)

var ErrAdmissionLedger = errors.New("registry011: candidate admission ledger refused")

const candidateLedgerMetadataLimit = 8192

// CandidateLedgerBounds limit this ephemeral model's complete lifetime, not an
// operating workload, retaining authority or physical filesystem allocation.
type CandidateLedgerBounds struct {
	MaximumRecords       int
	MaximumPayloadBytes  uint64
	MaximumMetadataBytes uint64
}

type candidateLedgerTotals struct {
	bootstrap                  candidateBudgetTotals
	liveUsed, liveReserved     uint64
	retainedUsed, retainedHeld uint64
}

type candidateLedgerTicket struct {
	ticket      *CandidateAdmissionTicket
	consumption *CandidateConsumption
	terminal    string
}

// CandidateAdmissionLedger models one bounded, complete candidate-data prefix.
// Its initial ticket may be local work or any parsed allowlisted read. It does
// not qualify the actual bootstrap's first exact finalized request or params.
// Recreating it loses state. It MUST NOT reset or stand in for an actual
// retaining authority, authenticated restart, custody ACK or work admission.
// No Source, capture, replay executor or operating store accepts this type.
type CandidateAdmissionLedger struct {
	mu              sync.Mutex
	owner           *CandidateAdmissionLedger
	policy          string
	intent          *CandidateAdmissionIntent
	bounds          CandidateLedgerBounds
	limits          [3]candidateWorkVector
	live, retained  uint64
	tickets         map[string]candidateLedgerTicket
	segments        map[uint64]candidateBudgetTotals
	lastTicket      *CandidateAdmissionTicket
	lastConsumption *CandidateConsumption
	current         *CandidateAdmissionLedgerChange
	initialNodes    []string
	initialHeads    [3]string
	initialBytes    uint64
}

// CandidateAdmissionLedgerChange owns candidate metadata only. Its bytes are
// not a durable write, independently retained root, permission or trusted clock.
type CandidateAdmissionLedgerChange struct {
	raw, digest       string
	admission         string
	sequence          uint64
	heads             [3]string
	counts            [3]uint64
	totals            candidateLedgerTotals
	payload, metadata uint64
	nodes             []string
	clock             CandidateClockDiagnostic
}

type CandidateAdmissionLedgerDiagnostic struct {
	BudgetRoot, Admission                                                      string
	Sequence, Tickets, Consumptions, Settlements                               uint64
	Charged, Outstanding                                                       CandidateWork
	LiveUsedUpper, LiveReservedUpper, RetainedUsedUpper, RetainedReservedUpper uint64
	PayloadBytes, MetadataBytes                                                uint64
	LedgerHeads                                                                [3]string
	CandidateClock                                                             CandidateClockDiagnostic
}

func (*CandidateAdmissionLedger) UnmarshalJSON([]byte) error       { return ErrAdmissionLedger }
func (*CandidateAdmissionLedgerChange) UnmarshalJSON([]byte) error { return ErrAdmissionLedger }

func candidateLedgerBoundsValid(b CandidateLedgerBounds) bool {
	return b.MaximumRecords > 0 && b.MaximumRecords <= 4096 && b.MaximumPayloadBytes > 0 && b.MaximumPayloadBytes <= 16<<20 && b.MaximumMetadataBytes > 0 && b.MaximumMetadataBytes <= 64<<20
}

// NewCandidateAdmissionLedger checks a generic initial candidate ticket and
// its accounting chain only. Actual bootstrap selection must independently
// bind the first finalized method, parameters and owned request bytes; success
// here establishes none of those conditions and never authorizes dispatch.
func NewCandidateAdmissionLedger(policy *ProviderPolicyCandidate, intent *CandidateAdmissionIntent, first *CandidateAdmissionTicket, bounds CandidateLedgerBounds) (*CandidateAdmissionLedger, error) {
	if policy == nil || policy.raw == "" || intent == nil || intent.raw == "" || policy.policy != intent.identity.Policy || !candidateLedgerBoundsValid(bounds) {
		return nil, ErrConfiguration
	}
	if first == nil || first.raw == "" || first.intent.digest != intent.digest || first.policy != policy.raw || first.sequence != 0 || first.segment != 0 || first.previous != "" || first.work[8] != 1 {
		return nil, ErrAdmissionLedger
	}
	m := &CandidateAdmissionLedger{policy: policy.raw, intent: intent, bounds: bounds, tickets: make(map[string]candidateLedgerTicket), segments: make(map[uint64]candidateBudgetTotals)}
	m.owner = m
	p := contentJSON([]byte(policy.raw)).(map[string]any)
	limits := p["limits"].(map[string]any)
	for i, level := range []string{"operation", "segment", "bootstrap"} {
		m.limits[i], _ = candidateAdmissionWorkLimit(limits[level])
	}
	m.live, _ = providerCandidateUint(limits["live_disk_bytes"], true)
	m.retained, _ = providerCandidateUint(limits["retained_bytes"], true)
	for i, kind := range []string{"tickets", "consumptions", "settlements"} {
		raw := candidateLedgerNode(intent.digest, kind, 0, "", "")
		if len(raw) > candidateLedgerMetadataLimit || !candidateCapacity(0, m.initialBytes, uint64(len(raw)), bounds.MaximumMetadataBytes) {
			return nil, ErrAdmissionLedger
		}
		m.initialNodes = append(m.initialNodes, string(raw))
		m.initialBytes += uint64(len(raw))
		m.initialHeads[i] = providerCandidateHash("sage-provider-ledger-node-v1", raw)
	}
	if _, err := m.appendTicket(first); err != nil {
		return nil, err
	}
	m.initialNodes = nil
	return m, nil
}

func (m *CandidateAdmissionLedger) valid() bool {
	return m.owner == m && m.policy != "" && m.intent != nil && m.current != nil && m.current.raw != "" && m.tickets != nil && m.segments != nil && candidateLedgerBoundsValid(m.bounds)
}

func (m *CandidateAdmissionLedger) AppendCandidateTicket(ticket *CandidateAdmissionTicket) (*CandidateAdmissionLedgerChange, error) {
	if m == nil {
		return nil, ErrAdmissionLedger
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.valid() || ticket == nil || ticket.raw == "" || ticket.intent.digest != m.intent.digest || ticket.policy != m.policy || CheckCandidateTicketSuccessor(m.lastTicket, ticket) != nil || candidateLedgerExpired(m.current.clock, ticket.deadline) {
		return nil, ErrAdmissionLedger
	}
	return m.appendTicket(ticket)
}

func (m *CandidateAdmissionLedger) appendTicket(ticket *CandidateAdmissionTicket) (*CandidateAdmissionLedgerChange, error) {
	if _, exists := m.tickets[ticket.digest]; exists {
		return nil, ErrAdmissionLedger
	}
	var totals candidateLedgerTotals
	clock := m.intent.clock
	if m.current != nil {
		totals, clock = m.current.totals, m.current.clock
	}
	segment := m.segments[ticket.segment]
	if !candidateFits(candidateBudgetTotals{}, ticket.work, m.limits[0]) || !candidateFits(segment, ticket.work, m.limits[1]) || !candidateFits(totals.bootstrap, ticket.work, m.limits[2]) || !candidateCapacity(totals.liveUsed, totals.liveReserved, ticket.live, m.live) || !candidateCapacity(totals.retainedUsed, totals.retainedHeld, ticket.retained, m.retained) {
		return nil, ErrAdmissionLedger
	}
	for i, n := range ticket.work {
		segment.outstanding[i] += n
		totals.bootstrap.outstanding[i] += n
	}
	totals.liveReserved += ticket.live
	totals.retainedHeld += ticket.retained
	change, err := m.prepare(0, ticket.raw, ticket.digest, totals, clock)
	if err != nil {
		return nil, err
	}
	m.tickets[ticket.digest] = candidateLedgerTicket{ticket: ticket}
	m.segments[ticket.segment] = segment
	m.lastTicket, m.current = ticket, change
	return change, nil
}

func (m *CandidateAdmissionLedger) AppendCandidateConsumption(consumption *CandidateConsumption) (*CandidateAdmissionLedgerChange, error) {
	if m == nil {
		return nil, ErrAdmissionLedger
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.valid() || consumption == nil || consumption.raw == "" {
		return nil, ErrAdmissionLedger
	}
	state, exists := m.tickets[consumption.ticket.digest]
	if !exists || state.terminal != "" || state.consumption != nil || consumption.ticket.intent.digest != m.intent.digest || consumption.sequence != m.current.counts[1] || !candidateLedgerClockAdvance(m.current.clock, consumption.clock) || candidateLedgerExpired(m.current.clock, state.ticket.deadline) {
		return nil, ErrAdmissionLedger
	}
	if m.lastConsumption == nil {
		if consumption.previous != "" {
			return nil, ErrAdmissionLedger
		}
	} else if CheckCandidateConsumptionSuccessor(m.lastConsumption, consumption) != nil {
		return nil, ErrAdmissionLedger
	}
	change, err := m.prepare(1, consumption.raw, consumption.digest, m.current.totals, consumption.clock)
	if err != nil {
		return nil, err
	}
	state.consumption = consumption
	m.tickets[consumption.ticket.digest] = state
	m.lastConsumption, m.current = consumption, change
	return change, nil
}

func (m *CandidateAdmissionLedger) AppendCandidateSettlement(settlement *CandidateAdmissionSettlement) (*CandidateAdmissionLedgerChange, error) {
	if m == nil {
		return nil, ErrAdmissionLedger
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.valid() || settlement == nil || settlement.raw == "" {
		return nil, ErrAdmissionLedger
	}
	state, exists := m.tickets[settlement.ticket.digest]
	if !exists || state.terminal != "" || settlement.ticket.intent.digest != m.intent.digest || !candidateLedgerClockAdvance(m.current.clock, settlement.clock) || settlement.outcome == "complete" && (state.consumption == nil || !candidateLedgerClockAdvance(state.consumption.clock, settlement.clock)) {
		return nil, ErrAdmissionLedger
	}
	ticket := state.ticket
	totals := m.current.totals
	segment := m.segments[ticket.segment]
	if totals.liveReserved < ticket.live || totals.retainedHeld < ticket.retained {
		return nil, ErrAdmissionLedger
	}
	for i, n := range ticket.work {
		if totals.bootstrap.outstanding[i] < n || segment.outstanding[i] < n {
			return nil, ErrAdmissionLedger
		}
	}
	for i, n := range ticket.work {
		totals.bootstrap.outstanding[i] -= n
		totals.bootstrap.charged[i] += n
		segment.outstanding[i] -= n
		segment.charged[i] += n
	}
	totals.liveReserved -= ticket.live
	totals.liveUsed += ticket.live
	totals.retainedHeld -= ticket.retained
	totals.retainedUsed += ticket.retained
	change, err := m.prepare(2, settlement.raw, settlement.digest, totals, settlement.clock)
	if err != nil {
		return nil, err
	}
	state.terminal = settlement.digest
	m.tickets[ticket.digest] = state
	m.segments[ticket.segment] = segment
	m.current = change
	return change, nil
}

func candidateLedgerExpired(clock CandidateClockDiagnostic, deadline uint64) bool {
	return uint64(clock.WallMS)+uint64(clock.UncertaintyMS) > deadline
}

// Prepare serializes constant-shape new metadata only; no prior prefix is
// cloned, recursively replayed or rehashed. Refusal never changes the model.
func (m *CandidateAdmissionLedger) prepare(kind int, payload, entry string, totals candidateLedgerTotals, clock CandidateClockDiagnostic) (*CandidateAdmissionLedgerChange, error) {
	if kind < 0 || kind > 2 {
		return nil, ErrAdmissionLedger
	}
	change := &CandidateAdmissionLedgerChange{admission: m.intent.digest, heads: m.initialHeads, metadata: m.initialBytes}
	var previous any
	if m.current != nil {
		*change = *m.current
		if change.sequence == math.MaxUint64 {
			return nil, ErrAdmissionLedger
		}
		change.sequence++
		previous = m.current.digest
	}
	var records uint64
	for _, count := range change.counts {
		if !candidateCapacity(0, records, count, uint64(m.bounds.MaximumRecords)) {
			return nil, ErrAdmissionLedger
		}
		records += count
	}
	if records >= uint64(m.bounds.MaximumRecords) || !candidateCapacity(0, change.payload, uint64(len(payload)), m.bounds.MaximumPayloadBytes) {
		return nil, ErrAdmissionLedger
	}
	change.counts[kind]++
	names := [...]string{"tickets", "consumptions", "settlements"}
	node := candidateLedgerNode(m.intent.digest, names[kind], change.counts[kind], change.heads[kind], entry)
	change.heads[kind] = providerCandidateHash("sage-provider-ledger-node-v1", node)
	raw := candidateLedgerBudget(m.intent, change.sequence, previous, change.heads, totals)
	if len(node) > candidateLedgerMetadataLimit || len(raw) > candidateLedgerMetadataLimit || !candidateCapacity(0, change.metadata, uint64(len(node))+uint64(len(raw)), m.bounds.MaximumMetadataBytes) {
		return nil, ErrAdmissionLedger
	}
	change.raw, change.digest = string(raw), providerCandidateHash("sage-provider-budget-root-v2", raw)
	change.totals, change.clock = totals, clock
	change.payload += uint64(len(payload))
	change.metadata += uint64(len(node)) + uint64(len(raw))
	change.nodes = []string{string(node)}
	if m.current == nil {
		change.nodes = append(append([]string(nil), m.initialNodes...), string(node))
	}
	return change, nil
}

func candidateLedgerNode(admission, kind string, index uint64, previous, entry string) []byte {
	var previousValue, entryValue any
	if index != 0 {
		previousValue, entryValue = previous, entry
	}
	raw, _ := json.Marshal(map[string]any{"format": "sage-provider-ledger-node-v1", "admission_digest": admission, "kind": kind, "index": strconv.FormatUint(index, 10), "previous": previousValue, "entry": entryValue})
	return raw
}

func candidateLedgerWorkObject(work candidateWorkVector) map[string]string {
	names := [...]string{"requests", "wire_bytes", "decoded_bytes", "blocks", "transactions", "frames", "logs", "proofs", "segments", "disk_bytes_written", "work_units"}
	m := make(map[string]string, len(names))
	for i, name := range names {
		m[name] = strconv.FormatUint(work[i], 10)
	}
	return m
}

func candidateLedgerBudget(intent *CandidateAdmissionIntent, sequence uint64, previous any, heads [3]string, totals candidateLedgerTotals) []byte {
	id := intent.identity
	raw, _ := json.Marshal(map[string]any{"format": "sage-provider-budget-root-v2", "admission_digest": intent.digest, "bootstrap_start": nil, "namespace": id.Namespace, "instance_key": id.Instance, "generation": strconv.FormatUint(id.Generation, 10), "fence": strconv.FormatUint(intent.fence, 10), "sequence": strconv.FormatUint(sequence, 10), "previous_root": previous, "tickets_root": heads[0], "consumptions_root": heads[1], "settlements_root": heads[2], "charged": candidateLedgerWorkObject(totals.bootstrap.charged), "outstanding": candidateLedgerWorkObject(totals.bootstrap.outstanding), "live_disk_used": strconv.FormatUint(totals.liveUsed, 10), "live_disk_reserved": strconv.FormatUint(totals.liveReserved, 10), "retained_bytes": strconv.FormatUint(totals.retainedUsed, 10), "retained_bytes_reserved": strconv.FormatUint(totals.retainedHeld, 10), "deadline_wall_ms": strconv.FormatUint(intent.deadline, 10)})
	return raw
}

func (m *CandidateAdmissionLedger) Current() (*CandidateAdmissionLedgerChange, error) {
	if m == nil {
		return nil, ErrAdmissionLedger
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.valid() {
		return nil, ErrAdmissionLedger
	}
	return m.current, nil
}

func (c *CandidateAdmissionLedgerChange) BudgetBytes() []byte {
	if c == nil {
		return nil
	}
	return []byte(c.raw)
}

func (c *CandidateAdmissionLedgerChange) LedgerNodeBytes() [][]byte {
	if c == nil {
		return nil
	}
	out := make([][]byte, len(c.nodes))
	for i, raw := range c.nodes {
		out[i] = []byte(raw)
	}
	return out
}

func (c *CandidateAdmissionLedgerChange) Diagnostic() (CandidateAdmissionLedgerDiagnostic, error) {
	if c == nil || c.raw == "" {
		return CandidateAdmissionLedgerDiagnostic{}, ErrAdmissionLedger
	}
	return CandidateAdmissionLedgerDiagnostic{BudgetRoot: c.digest, Admission: c.admission, Sequence: c.sequence, Tickets: c.counts[0], Consumptions: c.counts[1], Settlements: c.counts[2], Charged: c.totals.bootstrap.charged.diagnostic(), Outstanding: c.totals.bootstrap.outstanding.diagnostic(), LiveUsedUpper: c.totals.liveUsed, LiveReservedUpper: c.totals.liveReserved, RetainedUsedUpper: c.totals.retainedUsed, RetainedReservedUpper: c.totals.retainedHeld, PayloadBytes: c.payload, MetadataBytes: c.metadata, LedgerHeads: c.heads, CandidateClock: c.clock}, nil
}
