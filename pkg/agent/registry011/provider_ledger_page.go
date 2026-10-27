package registry011

import "strconv"

const providerLedgerSealLimit uint64 = 64 << 10

// Fixed closed fields, three 64-byte digests and bounded decimal ordinals.
// This upper is checked before allocating the page's maps or codec buffers.
func providerLedgerPageUpper(records uint64) (uint64, bool) {
	if records == 0 || records > 4096 {
		return 0, false
	}
	entries, ok := providerUpperProduct(records, 512)
	if !ok {
		return 0, false
	}
	upper, ok := providerUpperSum(entries, 1024)
	return upper, ok && upper <= 4<<20
}

type providerLedgerRecord struct {
	ordinal                   uint64
	kind, entry, node, budget string
}

func (m *CandidateAdmissionLedger) noteProviderRecord(kind, entry string, change *CandidateAdmissionLedgerChange) error {
	c := m.continuation
	if c == nil {
		return nil
	}
	node := change.nodes[len(change.nodes)-1]
	digest := providerReplayDigest("sage-provider-ledger-node-v1", providerReplayPhaseBytes(node, m.meter), m.meter)
	if digest == "" {
		return ErrAdmissionLedger
	}
	c.records = append(c.records, providerLedgerRecord{change.sequence, kind, entry, digest, change.digest})
	return nil
}

func (m *CandidateAdmissionLedger) providerPage() (providerIntegrationObject, error) {
	if m == nil {
		return providerIntegrationObject{}, ErrAdmissionLedger
	}
	c := m.continuation
	if !m.valid() || c == nil || len(c.records) == 0 || uint64(len(c.records)) > c.bounds.pageRecords {
		return providerIntegrationObject{}, ErrAdmissionLedger
	}
	pageUpper, ok := providerLedgerPageUpper(uint64(len(c.records)))
	if !ok || !c.metadataRoom(m, uint64(len(c.records)), 0) {
		return providerIntegrationObject{}, ErrAdmissionLedger
	}
	// Include previously built page/seal values held by boundary validation as
	// well as the new output and parsing scratch. Actual allocator overhead is
	// still a separately qualified profile assumption.
	scratch, ok := providerUpperProduct(pageUpper, 3)
	if !ok {
		return providerIntegrationObject{}, ErrAdmissionLedger
	}
	scratch, ok = providerUpperSum(scratch, 2*providerLedgerSealLimit)
	if !ok {
		return providerIntegrationObject{}, ErrAdmissionLedger
	}
	retained, ok := providerUpperSum(m.current.payload, m.current.privatePayload, m.current.metadata, m.current.privateMetadata)
	if !ok {
		return providerIntegrationObject{}, ErrAdmissionLedger
	}
	operations, ok := providerUpperSum(m.current.sequence, 2)
	if !ok {
		return providerIntegrationObject{}, ErrAdmissionLedger
	}
	if _, ok = c.bounds.residentUpper(uint64(len(m.tickets)), operations, uint64(len(m.segments)), 0, retained, scratch); !ok {
		return providerIntegrationObject{}, ErrAdmissionLedger
	}
	records := make([]map[string]string, 0, len(c.records))
	for i, record := range c.records {
		if record.ordinal != c.sealedRecords+uint64(i) || !lowerHash(record.entry) || !lowerHash(record.node) || !lowerHash(record.budget) || record.kind != "ticket" && record.kind != "consumption" && record.kind != "settlement" {
			return providerIntegrationObject{}, ErrAdmissionLedger
		}
		records = append(records, map[string]string{"ordinal": strconv.FormatUint(record.ordinal, 10), "kind": record.kind, "entry_digest": record.entry, "ledger_node_digest": record.node, "budget_root": record.budget})
	}
	var previousPage, previousBudget any
	if c.page != 0 {
		previousPage, previousBudget = c.lastPage, c.lastBudget
	}
	raw := providerReplayCanonical(map[string]any{"format": "sage-provider-ledger-page-v1", "admission_digest": m.intent.digest, "index": strconv.FormatUint(c.page, 10), "previous_page": previousPage, "first_record": strconv.FormatUint(c.sealedRecords, 10), "last_record": strconv.FormatUint(c.records[len(c.records)-1].ordinal, 10), "previous_budget": previousBudget, "last_budget": m.current.digest, "records": records}, pageUpper, m.meter)
	if uint64(len(raw)) > pageUpper {
		return providerIntegrationObject{}, ErrAdmissionLedger
	}
	return newProviderIntegrationObjectObserved("ledger-page", raw, 4<<20, m.meter)
}

func (b *providerLedgerBoundary) sealObject(indexHeads [3]string) (providerIntegrationObject, error) {
	if b == nil || !b.valid(b.owner) {
		return providerIntegrationObject{}, ErrAdmissionLedger
	}
	for _, head := range indexHeads {
		if !lowerHash(head) {
			return providerIntegrationObject{}, ErrAdmissionLedger
		}
	}
	m, c, state := b.owner, b.owner.continuation, b.change
	if !c.valid(m) || b.page != c.page {
		return providerIntegrationObject{}, ErrAdmissionLedger
	}
	id := m.intent.identity
	metadata, payload, err := providerLedgerStateBytes(state)
	if err != nil {
		return providerIntegrationObject{}, err
	}
	var previous any
	if c.page != 0 {
		previous = c.lastSeal
	}
	counts, heads := map[string]string{}, map[string]string{}
	for i, kind := range []string{"tickets", "consumptions", "settlements"} {
		counts[kind], heads[kind] = strconv.FormatUint(state.counts[i], 10), state.heads[i]
	}
	clock := ClockSample{ClockID: m.intent.clock.ClockID, Epoch: m.intent.clock.Epoch, WallMS: m.intent.clock.WallMS, MonotonicMS: m.intent.clock.MonotonicMS, UncertaintyMS: m.intent.clock.UncertaintyMS}
	raw := providerReplayCanonical(map[string]any{"format": "sage-provider-ledger-seal-v1", "admission_digest": m.intent.digest, "namespace": id.Namespace, "instance_key": id.Instance, "source_id": id.SourceID, "generation": strconv.FormatUint(id.Generation, 10), "fence": strconv.FormatUint(m.intent.fence, 10), "page_index": strconv.FormatUint(c.page, 10), "previous_seal": previous, "page_root": b.pageObject.digest, "final": b.final, "budget_root": state.digest, "counts": counts, "heads": heads, "charged": candidateLedgerWorkObject(state.totals.bootstrap.charged), "outstanding": candidateLedgerWorkObject(state.totals.bootstrap.outstanding), "live_used": strconv.FormatUint(state.totals.liveUsed, 10), "live_reserved": strconv.FormatUint(state.totals.liveReserved, 10), "retained_used": strconv.FormatUint(state.totals.retainedUsed, 10), "retained_reserved": strconv.FormatUint(state.totals.retainedHeld, 10), "payload_bytes": strconv.FormatUint(payload, 10), "metadata_bytes": strconv.FormatUint(metadata, 10), "ticket_index_root": indexHeads[0], "operation_index_root": indexHeads[1], "segment_totals_root": indexHeads[2], "initial_clock": providerClockObject(clock), "deadline_wall_ms": strconv.FormatUint(m.intent.deadline, 10)}, providerLedgerSealLimit, m.meter)
	object, err := newProviderIntegrationObjectObserved("ledger-seal", raw, providerLedgerSealLimit, m.meter)
	if err != nil || b.sealDigest != "" && b.sealDigest != object.digest {
		return providerIntegrationObject{}, ErrAdmissionLedger
	}
	return object, nil
}

func (b *providerLedgerBoundary) valid(m *CandidateAdmissionLedger) bool {
	if b == nil || m == nil || !m.valid() || b.owner != m || b.change == nil || b.used || m.continuation == nil {
		return false
	}
	// The seal's own operation-index metadata may be retained after the page
	// was built. Only this additional accounting may change; no T/C/S may move.
	before, current := *b.change, *m.current
	before.privateMetadata, current.privateMetadata = 0, 0
	before.nodes, current.nodes = nil, nil
	if before.raw != current.raw || before.digest != current.digest || before.admission != current.admission || before.sequence != current.sequence || before.counts != current.counts || before.heads != current.heads || before.totals != current.totals || before.payload != current.payload || before.privatePayload != current.privatePayload || before.metadata != current.metadata || before.clock != current.clock || b.change.privateMetadata > m.current.privateMetadata {
		return false
	}
	c := m.continuation
	auxiliary, valid := providerUpperSum(b.auxiliaryAtBuild, b.sealMetadata)
	if !valid || c.pending != b || auxiliary != c.auxiliaryMetadata || b.sealMetadata == 0 && b.sealDigest != "" || b.sealMetadata != 0 && !lowerHash(b.sealDigest) {
		return false
	}
	records, ok := providerUpperSum(m.current.counts[0], m.current.counts[1], m.current.counts[2])
	if !ok || c.final || b.page != c.page || b.records != records || b.payload != m.current.payload || b.metadata != m.current.metadata || records == c.sealedRecords {
		return false
	}
	if b.final {
		for _, value := range m.current.totals.bootstrap.outstanding {
			if value != 0 {
				return false
			}
		}
	} else if records-c.sealedRecords != c.bounds.pageRecords || c.page+1 >= c.bounds.maximumPages {
		return false
	}
	page, err := m.providerPage()
	return err == nil && page == b.pageObject
}

// This advances private arithmetic only. Its caller must validate exact page,
// index heads and custody frontier before any connected work is authorized.
// Parsing a seal does not acknowledge a write or grant an operating credential.
func (m *CandidateAdmissionLedger) applyProviderBoundary(b *providerLedgerBoundary, pageRaw, sealRaw []byte, indexHeads [3]string) error {
	if m == nil {
		return ErrAdmissionLedger
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !b.valid(m) {
		return ErrAdmissionLedger
	}
	seal, err := b.sealObject(indexHeads)
	if err != nil || !providerReplayEqual(pageRaw, providerReplayPhaseBytes(b.pageObject.raw, m.meter), m.meter) || !providerReplayEqual(sealRaw, providerReplayPhaseBytes(seal.raw, m.meter), m.meter) {
		return ErrAdmissionLedger
	}
	c := m.continuation
	added, ok := providerUpperSum(uint64(len(pageRaw)), uint64(len(sealRaw)))
	if !ok {
		return ErrAdmissionLedger
	}
	extra, ok := providerUpperSum(c.extraMetadata, added)
	private, valid := providerUpperSum(extra, c.auxiliaryMetadata)
	active, activeOK := providerUpperSum(extra-c.sealedExtra, c.auxiliaryMetadata-c.sealedAuxiliary)
	if !ok || !valid || !activeOK || !candidateCapacity(0, m.current.metadata, private, c.bounds.maximumMetadataBytes) || !candidateCapacity(0, m.current.metadata-c.sealedMetadata, active, c.bounds.pageMetadataBytes) {
		return ErrAdmissionLedger
	}
	// Keep previously returned immutable changes intact. Sealing changes only
	// additional storage accounting, never the original budget or work vectors.
	updated := *m.current
	updated.privateMetadata = private
	b.used = true
	c.page++
	c.sealedRecords, c.sealedPayload, c.sealedMetadata = b.records, b.payload, b.metadata
	c.extraMetadata, c.sealedExtra = extra, extra
	c.sealedAuxiliary = c.auxiliaryMetadata
	c.sealedExternal = c.auxiliaryPayload
	c.lastPage, c.lastSeal, c.lastBudget = b.pageObject.digest, seal.digest, b.change.digest
	c.records, c.final = nil, b.final
	c.pending = nil
	m.current = &updated
	return nil
}

// Seal totals include both primary and auxiliary retained bytes.
func providerLedgerStateBytes(state *CandidateAdmissionLedgerChange) (uint64, uint64, error) {
	metadata, ok := providerUpperSum(state.metadata, state.privateMetadata)
	if !ok {
		return 0, 0, ErrAdmissionLedger
	}
	payload, ok := providerUpperSum(state.payload, state.privatePayload)
	if !ok {
		return 0, 0, ErrAdmissionLedger
	}
	return metadata, payload, nil
}
