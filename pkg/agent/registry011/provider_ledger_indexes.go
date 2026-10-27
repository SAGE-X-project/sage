package registry011

import "strconv"

// A separately replayed append-only mutation chain owns each map. Neither a
// supplied JSON snapshot nor these arithmetic objects is a custody credential.
// The custodian derives its own instance from the full selected prefix.
type providerLedgerIndexes struct {
	owner       *providerLedgerIndexes
	identity    ProviderCandidateIdentity
	fence       uint64
	bounds      providerLedgerBounds
	heads       [3]string
	counts      [3]uint64
	tickets     map[string]providerTicketIndexValue
	operations  map[string]providerOperationIndexValue
	segments    map[string]providerSegmentIndexValue
	admission   string
	schemaBytes uint64
	ledger      *CandidateAdmissionLedger
	meter       *providerWorkMeter
}

type providerIndexMutation struct {
	chain           int
	key             string
	expected, value any
	object          providerIntegrationObject
}

type providerIndexDelta struct {
	owner           *providerLedgerIndexes
	previous        [3]string
	counts          [3]uint64
	admission       string
	mutations       []providerIndexMutation
	metadata        uint64
	used            bool
	kind, operation string
	sequence        uint64
	entry           providerIntegrationObject
	root            *CandidateEnrollmentRoot
	ledger          *CandidateAdmissionLedger
	change          *CandidateAdmissionLedgerChange
}

func (*providerLedgerIndexes) UnmarshalJSON([]byte) error { return ErrAdmissionLedger }
func (*providerIndexDelta) UnmarshalJSON([]byte) error    { return ErrAdmissionLedger }

func newProviderLedgerIndexes(id ProviderCandidateIdentity, fence uint64, bounds providerLedgerBounds) (*providerLedgerIndexes, []providerIntegrationObject, error) {
	return newProviderLedgerIndexesObserved(id, fence, bounds, nil)
}

func newProviderLedgerIndexesObserved(id ProviderCandidateIdentity, fence uint64, bounds providerLedgerBounds, meter *providerWorkMeter) (*providerLedgerIndexes, []providerIntegrationObject, error) {
	if !lowerHash(id.Namespace) || !lowerHash(id.Instance) || !lowerHash(id.SourceID) || fence == 0 || !bounds.valid() || bounds.maximumMetadataBytes < 3*4096 {
		return nil, nil, ErrAdmissionLedger
	}
	// Preflight even the three bounded initial encodings before map allocation.
	if _, ok := bounds.residentUpper(0, 0, 0, 0, 3*4096, 3*4096); !ok {
		return nil, nil, ErrAdmissionLedger
	}
	s := &providerLedgerIndexes{identity: id, fence: fence, bounds: bounds, tickets: make(map[string]providerTicketIndexValue), operations: make(map[string]providerOperationIndexValue), segments: make(map[string]providerSegmentIndexValue)}
	s.owner = s
	anchors := make([]providerIntegrationObject, 0, 3)
	for index, kind := range []string{"tickets", "operations", "segments"} {
		if meter != nil && !meter.preflight(providerEncodeEvent, 4096) {
			return nil, nil, errProviderIntegration
		}
		raw := providerReplayCanonical(map[string]any{"format": "sage-provider-index-empty-v1", "namespace": id.Namespace, "instance_key": id.Instance, "source_id": id.SourceID, "generation": strconv.FormatUint(id.Generation, 10), "fence": strconv.FormatUint(fence, 10), "kind": kind}, 4096, meter)
		anchor, err := newProviderIntegrationObjectObserved("index-empty", raw, 4096, meter)
		if err != nil {
			return nil, nil, err
		}
		s.heads[index] = anchor.digest
		s.schemaBytes += uint64(len(raw))
		anchors = append(anchors, anchor)
	}
	return s, anchors, nil
}

func (s *providerLedgerIndexes) bindLedger(m *CandidateAdmissionLedger) error {
	return s.bindRetainedLedger(m, providerFrontierRetention{})
}

func (s *providerLedgerIndexes) bindRetainedLedger(m *CandidateAdmissionLedger, prior providerFrontierRetention) error {
	if !s.valid() || s.ledger != nil || s.admission != "" || len(s.operations) != 1 || len(s.tickets) != 0 || len(s.segments) != 0 || m == nil || !m.valid() || m.continuation == nil || m.continuation.bounds != s.bounds || m.intent.identity != s.identity || m.intent.fence != s.fence || m.continuation.auxiliaryMetadata != 0 || s.meter != m.meter {
		return ErrAdmissionLedger
	}
	metadata, ok := providerUpperSum(s.schemaBytes, prior.metadata)
	if !ok {
		return ErrAdmissionLedger
	}
	if err := m.addProviderRetention(m.current, prior.payload, metadata); err != nil {
		return err
	}
	s.ledger = m
	return nil
}

func (s *providerLedgerIndexes) valid() bool {
	if s == nil || s.meter != nil && (s.meter.failed || s.meter.closed) || s.owner != s || !s.bounds.valid() || s.tickets == nil || s.operations == nil || s.segments == nil || s.fence == 0 || s.admission != "" && !lowerHash(s.admission) {
		return false
	}
	for _, head := range s.heads {
		if !lowerHash(head) {
			return false
		}
	}
	return uint64(len(s.tickets)) <= s.bounds.maximumRecords && uint64(len(s.operations)) <= s.bounds.maximumCommits && uint64(len(s.segments)) <= s.bounds.maximumRecords && s.counts[0] <= s.bounds.maximumRecords && s.counts[2] <= s.bounds.maximumRecords && s.counts[1] == uint64(len(s.operations)) && s.schemaBytes <= s.bounds.maximumMetadataBytes
}

func (s *providerLedgerIndexes) mutation(chain int, key string, expected, value any, entry, budget, admission string) (providerIndexMutation, error) {
	if chain < 0 || chain > 2 || !lowerHash(key) || !lowerHash(entry) || admission == "" && budget != "" || admission != "" && (!lowerHash(admission) || !lowerHash(budget)) || s.counts[chain] == ^uint64(0) {
		return providerIndexMutation{}, ErrAdmissionLedger
	}
	var admissionValue, budgetValue any
	if admission != "" {
		admissionValue, budgetValue = admission, budget
	}
	raw := providerReplayCanonical(map[string]any{"format": "sage-provider-index-mutation-v1", "admission_digest": admissionValue, "kind": []string{"tickets", "operations", "segments"}[chain], "index": strconv.FormatUint(s.counts[chain], 10), "previous": s.heads[chain], "key": key, "expected": expected, "value": value, "entry_digest": entry, "budget_root": budgetValue}, candidateLedgerMetadataLimit, s.meter)
	object, err := newProviderIntegrationObjectObserved("index-mutation", raw, candidateLedgerMetadataLimit, s.meter)
	return providerIndexMutation{chain, key, expected, value, object}, err
}

// prepare derives values from a typed, already advanced ledger. It does not
// mutate the index cache. The independent custodian calls the same pure rules
// with its independently reconstructed ledger and compares exact node bytes.
func (s *providerLedgerIndexes) prepare(kind, operation string, sequence uint64, entry providerIntegrationObject, root *CandidateEnrollmentRoot, m *CandidateAdmissionLedger) (*providerIndexDelta, error) {
	if !s.valid() || !providerIndexedCustodyKind(kind) || !lowerHash(operation) || root == nil || root.identity != s.identity || root.fence != s.fence || sequence != uint64(len(s.operations))+1 || sequence > s.bounds.maximumCommits {
		return nil, ErrAdmissionLedger
	}
	if _, used := s.operations[operation]; used {
		return nil, ErrAdmissionLedger
	}
	if (kind == "admit" || kind == "consume" || kind == "settle") && operation != entry.digest {
		return nil, ErrAdmissionLedger
	}
	if entry.raw == "" || len(entry.raw) > 64<<10 || root.raw == "" || len(root.raw) > candidateLedgerMetadataLimit {
		return nil, ErrAdmissionLedger
	}
	// Three constant-shape nodes at most, including the operation node.
	potential, ok := providerUpperSum(s.schemaBytes, 8*candidateLedgerMetadataLimit)
	if !ok || potential > s.bounds.maximumMetadataBytes {
		return nil, ErrAdmissionLedger
	}
	tickets, segments := uint64(len(s.tickets)), uint64(len(s.segments))
	if kind == "admit" {
		tickets++
		segments++
	}
	if _, ok := s.bounds.residentUpper(tickets, sequence, segments, 0, potential, 8*candidateLedgerMetadataLimit); !ok {
		return nil, ErrAdmissionLedger
	}
	actual, valid := providerIntegrationObjectDigestObserved(entry.kind, providerReplayPhaseBytes(entry.raw, s.meter), s.meter)
	if !valid || actual != entry.digest {
		return nil, ErrAdmissionLedger
	}
	parsedRoot, err := parseCandidateEnrollmentRootObserved(&ProviderCandidateContext{identity: s.identity, bound: true}, providerReplayPhaseBytes(root.raw, s.meter), candidateLedgerMetadataLimit, s.meter)
	if err != nil || parsedRoot.digest != root.digest || parsedRoot.identity != root.identity || parsedRoot.fence != root.fence || parsedRoot.sequence != root.sequence || parsedRoot.budget != root.budget {
		return nil, ErrAdmissionLedger
	}
	root = parsedRoot
	d := &providerIndexDelta{owner: s, previous: s.heads, counts: s.counts, admission: s.admission}
	d.kind, d.operation, d.sequence, d.entry, d.root, d.ledger = kind, operation, sequence, entry, root, m
	if m != nil {
		d.change = m.current
	}
	budget, admission := "", ""
	if kind == "enroll" {
		if sequence != 1 || len(s.tickets) != 0 || len(s.segments) != 0 || s.admission != "" || m != nil || root.sequence != 0 || root.budget != "" || entry.kind != "enrollment" || entry.digest != root.digest || entry.raw != root.raw {
			return nil, ErrAdmissionLedger
		}
	} else {
		if m == nil || s.ledger != m || !m.valid() || m.intent.identity != s.identity || m.intent.fence != s.fence || root.budget != m.current.digest || s.admission != "" && s.admission != m.intent.digest || len(s.operations) == 0 {
			return nil, ErrAdmissionLedger
		}
		admission, budget = m.intent.digest, m.current.digest
		d.admission = admission
		if m.continuation == nil || m.continuation.pending != nil && kind != "seal" {
			return nil, ErrAdmissionLedger
		}
		if kind == "admit" || kind == "consume" || kind == "settle" {
			if err := s.prepareLedgerMutation(d, kind, entry, m); err != nil {
				return nil, err
			}
		} else if kind == "seal" {
			boundary := m.continuation.pending
			if boundary == nil || boundary.sealMetadata != 0 {
				return nil, ErrAdmissionLedger
			}
			exact, err := boundary.sealObject(s.heads)
			if err != nil || exact != entry {
				return nil, ErrAdmissionLedger
			}
			value := providerReplayJSON(providerReplayPhaseBytes(entry.raw, s.meter), s.meter)
			seal, ok := value.(map[string]any)
			if !ok || entry.kind != "ledger-seal" || seal["admission_digest"] != admission || seal["budget_root"] != budget || seal["ticket_index_root"] != s.heads[0] || seal["operation_index_root"] != s.heads[1] || seal["segment_totals_root"] != s.heads[2] {
				return nil, ErrAdmissionLedger
			}
		} else if entry.kind != "verification-receipt" {
			return nil, ErrAdmissionLedger
		}
	}
	operationValue := providerOperationIndexValue{operation, kind, sequence}
	mutation, err := s.mutation(1, operation, nil, operationValue.object(), entry.digest, budget, admission)
	if err != nil {
		return nil, err
	}
	d.mutations = append(d.mutations, mutation)
	for _, node := range d.mutations {
		var ok bool
		d.metadata, ok = providerUpperSum(d.metadata, uint64(len(node.object.raw)))
		if !ok {
			return nil, ErrAdmissionLedger
		}
	}
	return d, nil
}

func (s *providerLedgerIndexes) prepareLedgerMutation(d *providerIndexDelta, kind string, entry providerIntegrationObject, m *CandidateAdmissionLedger) error {
	var ticket *CandidateAdmissionTicket
	transition := ""
	switch kind {
	case "admit":
		ticket, transition = m.lastTicket, "ticket"
	case "consume":
		if m.lastConsumption == nil {
			return ErrAdmissionLedger
		}
		ticket, transition = m.lastConsumption.ticket, "consumption"
	case "settle":
		value, ok := providerReplayJSON(providerReplayPhaseBytes(entry.raw, s.meter), s.meter).(map[string]any)
		if !ok {
			return ErrAdmissionLedger
		}
		state, exists := m.tickets[contentString(value["ticket_digest"])]
		if !exists || state.terminal != entry.digest {
			return ErrAdmissionLedger
		}
		ticket, transition = state.ticket, "settlement"
	}
	if ticket == nil || entry.kind != transition || len(m.current.nodes) == 0 {
		return ErrAdmissionLedger
	}
	latest := contentObject(providerReplayJSON(providerReplayPhaseBytes(m.current.nodes[len(m.current.nodes)-1], s.meter), s.meter), "format", "admission_digest", "kind", "index", "previous", "entry")
	if latest == nil || latest["entry"] != entry.digest || latest["admission_digest"] != m.intent.digest {
		return ErrAdmissionLedger
	}
	state, exists := m.tickets[ticket.digest]
	if !exists {
		return ErrAdmissionLedger
	}
	after, err := providerTicketIndexFromState(state)
	if err != nil {
		return err
	}
	var expected any
	var prior *providerTicketIndexValue
	if before, exists := s.tickets[ticket.digest]; exists {
		prior = &before
		expected = before.object()
	}
	if !providerTicketIndexTransition(prior, after, transition, entry.digest) {
		return ErrAdmissionLedger
	}
	mutation, err := s.mutation(0, ticket.digest, expected, after.object(), entry.digest, m.current.digest, m.intent.digest)
	if err != nil {
		return err
	}
	d.mutations = append(d.mutations, mutation)
	if kind == "consume" {
		return nil
	}
	segmentKey := candidateAdmissionSegmentObserved(m.intent.digest, ticket.segment, s.meter)
	before, known := s.segments[segmentKey]
	if !known {
		if kind != "admit" || ticket.segment != uint64(len(s.segments)) {
			return ErrAdmissionLedger
		}
		before.index = ticket.segment
	} else if before.index != ticket.segment {
		return ErrAdmissionLedger
	}
	updated, ok := providerSegmentIndexTransition(before.totals, ticket.work, kind == "settle")
	if !ok || updated != m.segments[ticket.segment] {
		return ErrAdmissionLedger
	}
	expected = nil
	if known {
		expected = before.object()
	}
	segment := providerSegmentIndexValue{ticket.segment, updated}
	mutation, err = s.mutation(2, segmentKey, expected, segment.object(), entry.digest, m.current.digest, m.intent.digest)
	if err != nil {
		return err
	}
	d.mutations = append(d.mutations, mutation)
	return nil
}

// apply is cache arithmetic, never an ACK. The surrounding custodian must
// independently validate the exact typed delta and durable composite frontier.
func (s *providerLedgerIndexes) apply(d *providerIndexDelta) error {
	return s.applyRetained(d, providerFrontierRetention{})
}

// The concrete V2 custodian derives retained from the exact newly written
// inventory/intent/commit/frontier bytes. Combine it with this mutation's debit
// before changing any index. A seal gets one exclusive combined metadata debit.
func (s *providerLedgerIndexes) applyRetained(d *providerIndexDelta, retained providerFrontierRetention) error {
	if !s.valid() || d == nil || d.owner != s || d.used || d.previous != s.heads || d.counts != s.counts || len(d.mutations) == 0 || len(d.mutations) > 3 {
		return ErrAdmissionLedger
	}
	if d.ledger != nil && d.ledger.current != d.change {
		return ErrAdmissionLedger
	}
	// Never accept a candidate node merely because its canonical hash matches.
	// Reconstruct it again from the exact typed original T/C/S and current old
	// maps. This enforces terminal/consumption monotonicity and all eleven work
	// coordinates even when valid-shaped node bytes were replaced in a delta.
	expected, err := s.prepare(d.kind, d.operation, d.sequence, d.entry, d.root, d.ledger)
	if err != nil || expected.admission != d.admission || expected.metadata != d.metadata || len(expected.mutations) != len(d.mutations) {
		return ErrAdmissionLedger
	}
	for i, node := range d.mutations {
		original := expected.mutations[i]
		if node.chain != original.chain || node.key != original.key || node.object != original.object || !bytesEqualProviderValueObserved(node.expected, original.expected, s.meter) || !bytesEqualProviderValueObserved(node.value, original.value, s.meter) {
			return ErrAdmissionLedger
		}
	}
	total, ok := providerUpperSum(s.schemaBytes, d.metadata)
	if !ok || total > s.bounds.maximumMetadataBytes {
		return ErrAdmissionLedger
	}
	// First validate every constant-shape value, expected map entry and node.
	var metadata uint64
	seen := [3]bool{}
	for _, node := range d.mutations {
		if node.chain < 0 || node.chain > 2 || seen[node.chain] {
			return ErrAdmissionLedger
		}
		seen[node.chain] = true
		var ok bool
		metadata, ok = providerUpperSum(metadata, uint64(len(node.object.raw)))
		if !ok {
			return ErrAdmissionLedger
		}
		if err := s.checkMutation(node, d.admission); err != nil {
			return err
		}
	}
	if !seen[1] || metadata != d.metadata {
		return ErrAdmissionLedger
	}
	if d.kind == "enroll" && retained != (providerFrontierRetention{}) {
		return ErrAdmissionLedger // E0 is bound once with its pre-I envelope later.
	}
	if d.kind != "enroll" {
		if s.ledger != d.ledger || s.ledger == nil {
			return ErrAdmissionLedger
		}
		combined, ok := providerUpperSum(d.metadata, retained.metadata)
		if !ok {
			return ErrAdmissionLedger
		}
		var err error
		if d.kind == "seal" {
			err = s.ledger.chargeProviderStorage(d.change, retained.payload, combined, s.ledger.continuation.pending, d.entry.digest)
		} else {
			err = s.ledger.addProviderRetention(d.change, retained.payload, combined)
		}
		if err != nil {
			return err
		}
	}
	for _, node := range d.mutations {
		switch node.chain {
		case 0:
			value, _ := parseProviderTicketIndex(node.value)
			s.tickets[node.key] = value
		case 1:
			value, _ := parseProviderOperationIndex(node.value)
			s.operations[node.key] = value
		case 2:
			value, _ := parseProviderSegmentIndex(node.value)
			s.segments[node.key] = value
		}
		s.heads[node.chain] = node.object.digest
		s.counts[node.chain]++
	}
	s.schemaBytes, s.admission, d.used = total, d.admission, true
	return nil
}

func (s *providerLedgerIndexes) checkMutation(node providerIndexMutation, admission string) error {
	if node.chain < 0 || node.chain > 2 || !lowerHash(node.key) || !lowerHash(node.object.digest) {
		return ErrAdmissionLedger
	}
	value := contentObject(providerReplayJSON(providerReplayPhaseBytes(node.object.raw, s.meter), s.meter), "format", "admission_digest", "kind", "index", "previous", "key", "expected", "value", "entry_digest", "budget_root")
	if value == nil || value["format"] != "sage-provider-index-mutation-v1" || value["kind"] != []string{"tickets", "operations", "segments"}[node.chain] || value["index"] != strconv.FormatUint(s.counts[node.chain], 10) || value["previous"] != s.heads[node.chain] || value["key"] != node.key || !bytesEqualProviderValueObserved(value["expected"], node.expected, s.meter) || !bytesEqualProviderValueObserved(value["value"], node.value, s.meter) || contentString(value["admission_digest"]) != admission || !providerCandidateDigest(value["entry_digest"]) {
		return ErrAdmissionLedger
	}
	actual, ok := providerIntegrationObjectDigestObserved("index-mutation", providerReplayPhaseBytes(node.object.raw, s.meter), s.meter)
	if !ok || actual != node.object.digest {
		return ErrAdmissionLedger
	}
	var expected any
	switch node.chain {
	case 0:
		parsed, ok := parseProviderTicketIndex(node.value)
		if !ok || parsed.ticket != node.key {
			return ErrAdmissionLedger
		}
		if before, exists := s.tickets[node.key]; exists {
			expected = before.object()
		}
	case 1:
		parsed, ok := parseProviderOperationIndex(node.value)
		if !ok || parsed.operation != node.key || parsed.sequence != uint64(len(s.operations))+1 {
			return ErrAdmissionLedger
		}
		if _, exists := s.operations[node.key]; exists {
			return ErrAdmissionLedger
		}
	case 2:
		parsed, ok := parseProviderSegmentIndex(node.value)
		if !ok || node.key != candidateAdmissionSegmentObserved(admission, parsed.index, s.meter) {
			return ErrAdmissionLedger
		}
		if before, exists := s.segments[node.key]; exists {
			expected = before.object()
		}
	}
	if !bytesEqualProviderValueObserved(expected, node.expected, s.meter) {
		return ErrAdmissionLedger
	}
	return nil
}

func bytesEqualProviderValue(a, b any) bool {
	return bytesEqualProviderValueObserved(a, b, nil)
}

func bytesEqualProviderValueObserved(a, b any, meter *providerWorkMeter) bool {
	if meter == nil {
		left, right := providerIntegrationJSON(a), providerIntegrationJSON(b)
		return left != nil && right != nil && string(left) == string(right)
	}
	left := providerReplayCanonical(a, candidateLedgerMetadataLimit, meter)
	right := providerReplayCanonical(b, candidateLedgerMetadataLimit, meter)
	return left != nil && right != nil && providerReplayEqual(left, right, meter)
}
