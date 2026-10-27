package registry011

// This is private arithmetic state only. It does not qualify or activate a
// store and cannot turn a seal digest/candidate JSON into dispatch permission.
// The connected custodian must independently reconstruct every page/index and
// obtain its exact live frontier ACK before accepting the next page.
type providerLedgerContinuation struct {
	owner                              *CandidateAdmissionLedger
	bounds                             providerLedgerBounds
	page, sealedRecords                uint64
	sealedPayload, sealedMetadata      uint64
	extraMetadata, sealedExtra         uint64
	auxiliaryMetadata, sealedAuxiliary uint64
	auxiliaryPayload, sealedExternal   uint64
	lastSeal                           string
	lastPage, lastBudget               string
	records                            []providerLedgerRecord
	final                              bool
	pending                            *providerLedgerBoundary
}

type providerLedgerBoundary struct {
	owner                            *CandidateAdmissionLedger
	change                           *CandidateAdmissionLedgerChange
	page, records, payload, metadata uint64
	final                            bool
	used                             bool
	pageObject                       providerIntegrationObject
	auxiliaryAtBuild, sealMetadata   uint64
	sealDigest                       string
}

func (*providerLedgerContinuation) UnmarshalJSON([]byte) error { return ErrAdmissionLedger }
func (*providerLedgerBoundary) UnmarshalJSON([]byte) error     { return ErrAdmissionLedger }

func newProviderContinuingLedger(policy *ProviderPolicyCandidate, intent *CandidateAdmissionIntent, first *CandidateAdmissionTicket, bounds providerLedgerBounds) (*CandidateAdmissionLedger, error) {
	return newProviderContinuingLedgerObserved(policy, intent, first, bounds, nil)
}

func newProviderContinuingLedgerObserved(policy *ProviderPolicyCandidate, intent *CandidateAdmissionIntent, first *CandidateAdmissionTicket, bounds providerLedgerBounds, meter *providerWorkMeter) (*CandidateAdmissionLedger, error) {
	if !bounds.valid() {
		return nil, ErrConfiguration
	}
	// Only the first record is made by the existing constructor. Neither this
	// model nor later pages are ever returned through a public candidate factory.
	initial := CandidateLedgerBounds{int(bounds.pageRecords), min(bounds.pagePayloadBytes, 16<<20), min(bounds.pageMetadataBytes, 64<<20)}
	m, err := newCandidateAdmissionLedgerObserved(policy, intent, first, initial, meter)
	if err != nil {
		return nil, err
	}
	m.continuation = &providerLedgerContinuation{owner: m, bounds: bounds}
	if err := m.noteProviderRecord("ticket", first.digest, m.current); err != nil {
		return nil, err
	}
	if !m.valid() || !m.continuation.allowsInitial(m) {
		return nil, ErrAdmissionLedger
	}
	return m, nil
}

func (m *CandidateAdmissionLedger) workingLimits() (uint64, uint64, uint64, bool) {
	if !candidateLedgerBoundsValid(m.bounds) {
		return 0, 0, 0, false
	}
	if m.continuation == nil {
		return uint64(m.bounds.MaximumRecords), m.bounds.MaximumPayloadBytes, m.bounds.MaximumMetadataBytes, true
	}
	c := m.continuation
	if !c.valid(m) || c.final || c.pending != nil {
		return 0, 0, 0, false
	}
	return c.bounds.maximumRecords, c.bounds.maximumPayloadBytes - c.auxiliaryPayload, c.bounds.maximumMetadataBytes - c.extraMetadata - c.auxiliaryMetadata, true
}

func (c *providerLedgerContinuation) valid(m *CandidateAdmissionLedger) bool {
	if c == nil || c.owner != m || m == nil || m.current == nil || !c.bounds.valid() || c.page > c.bounds.maximumPages || !c.final && c.page >= c.bounds.maximumPages || c.sealedPayload > m.current.payload || c.sealedMetadata > m.current.metadata || c.sealedExtra > c.extraMetadata || c.sealedAuxiliary > c.auxiliaryMetadata || c.sealedExternal > c.auxiliaryPayload {
		return false
	}
	private, ok := providerUpperSum(c.extraMetadata, c.auxiliaryMetadata)
	if !ok || m.current.privateMetadata != private || m.current.privatePayload != c.auxiliaryPayload || !candidateCapacity(0, m.current.metadata, private, c.bounds.maximumMetadataBytes) || !c.payloadRoom(m, 0) {
		return false
	}
	records, ok := providerUpperSum(m.current.counts[0], m.current.counts[1], m.current.counts[2])
	if !ok || records > c.bounds.maximumRecords || records < c.sealedRecords || records-c.sealedRecords > c.bounds.pageRecords {
		return false
	}
	if c.final {
		for _, value := range m.current.totals.bootstrap.outstanding {
			if value != 0 {
				return false
			}
		}
		return c.page > 0 && lowerHash(c.lastSeal) && lowerHash(c.lastPage) && lowerHash(c.lastBudget) && records == c.sealedRecords && len(c.records) == 0 && c.sealedPayload == m.current.payload && c.sealedMetadata == m.current.metadata && c.sealedExtra == c.extraMetadata && c.sealedAuxiliary == c.auxiliaryMetadata && c.sealedExternal == c.auxiliaryPayload
	}
	if c.page == 0 {
		return c.lastSeal == "" && c.lastPage == "" && c.lastBudget == "" && c.sealedRecords == 0 && c.sealedPayload == 0 && c.sealedMetadata == 0 && c.sealedExtra == 0 && c.extraMetadata == 0 && c.sealedAuxiliary == 0 && c.sealedExternal == 0 && uint64(len(c.records)) == records
	}
	sealed, exact := providerUpperProduct(c.page, c.bounds.pageRecords)
	return exact && c.sealedRecords == sealed && lowerHash(c.lastSeal) && lowerHash(c.lastPage) && lowerHash(c.lastBudget) && uint64(len(c.records)) == records-c.sealedRecords
}

func (c *providerLedgerContinuation) allowsInitial(m *CandidateAdmissionLedger) bool {
	if !c.valid(m) || m.current.payload > c.bounds.pagePayloadBytes || m.current.payload > c.bounds.maximumPayloadBytes || !c.metadataRoom(m, uint64(len(c.records)), 0) {
		return false
	}
	scratch, valid := providerUpperSum(m.current.metadata, providerLedgerSealLimit)
	if !valid {
		return false
	}
	retained, valid := providerUpperSum(m.current.payload, m.current.privatePayload, m.current.metadata, m.current.privateMetadata)
	if !valid {
		return false
	}
	_, ok := c.bounds.residentUpper(uint64(len(m.tickets)), 2, uint64(len(m.segments)), 0, retained, scratch)
	return ok
}

func (c *providerLedgerContinuation) allows(m *CandidateAdmissionLedger, records, payload, metadata uint64, newTicket bool) bool {
	if !c.valid(m) || c.final || records < c.sealedRecords || records-c.sealedRecords >= c.bounds.pageRecords || !c.payloadRoom(m, payload) || !c.metadataRoom(m, records-c.sealedRecords+1, metadata) {
		return false
	}
	tickets, segments := uint64(len(m.tickets)), uint64(len(m.segments))
	if newTicket {
		tickets++
		segments++ // Conservatively reserve a new map entry even if reused.
	}
	retained, ok := providerUpperSum(m.current.payload, c.auxiliaryPayload, payload, m.current.metadata, m.current.privateMetadata)
	if !ok {
		return false
	}
	scratch, ok := providerUpperSum(metadata, 3*candidateLedgerMetadataLimit, providerLedgerSealLimit)
	if !ok {
		return false
	}
	_, ok = c.bounds.residentUpper(tickets, records+2, segments, 0, retained, scratch)
	return ok
}

// Reserve the entire active page and its future seal before growing its first
// record. These are separate from T/C/S nodes and budget bytes. Exact page and
// seal bytes are debited on sealing; the reserved upper is never a quota reset.
func (c *providerLedgerContinuation) metadataRoom(m *CandidateAdmissionLedger, records, added uint64) bool {
	page, ok := providerLedgerPageUpper(records)
	if !ok {
		return false
	}
	future, ok := providerUpperSum(page, providerLedgerSealLimit)
	if !ok {
		return false
	}
	global, ok := providerUpperSum(m.current.metadata, c.extraMetadata, c.auxiliaryMetadata, added, future)
	if !ok || global > c.bounds.maximumMetadataBytes {
		return false
	}
	active, ok := providerUpperSum(m.current.metadata-c.sealedMetadata, c.extraMetadata-c.sealedExtra, c.auxiliaryMetadata-c.sealedAuxiliary, added, future)
	return ok && active <= c.bounds.pageMetadataBytes
}

// One combined quota includes index chains and, when connected, the custody
// metadata. A separate map or store cannot spend the same global quota again.
// This arithmetic charge is not storage confirmation or an external ACK.
func (m *CandidateAdmissionLedger) addProviderMetadata(before *CandidateAdmissionLedgerChange, added uint64) error {
	return m.chargeProviderMetadata(before, added, nil, "")
}

func (m *CandidateAdmissionLedger) addProviderSealMetadata(before *CandidateAdmissionLedgerChange, added uint64, boundary *providerLedgerBoundary, seal string) error {
	return m.chargeProviderMetadata(before, added, boundary, seal)
}

func (m *CandidateAdmissionLedger) chargeProviderMetadata(before *CandidateAdmissionLedgerChange, added uint64, boundary *providerLedgerBoundary, seal string) error {
	return m.chargeProviderStorage(before, 0, added, boundary, seal)
}

// RPC plaintext and other fresh payloads share the complete-lifetime and page
// quota with T/C/S bytes. The concrete custodian derives these amounts from its
// exact new inventory, never a caller's reported actual-work counters.
func (m *CandidateAdmissionLedger) addProviderRetention(before *CandidateAdmissionLedgerChange, payload, metadata uint64) error {
	return m.chargeProviderStorage(before, payload, metadata, nil, "")
}

func (c *providerLedgerContinuation) payloadRoom(m *CandidateAdmissionLedger, added uint64) bool {
	global, ok := providerUpperSum(m.current.payload, c.auxiliaryPayload, added)
	if !ok || global > c.bounds.maximumPayloadBytes {
		return false
	}
	active, ok := providerUpperSum(m.current.payload-c.sealedPayload, c.auxiliaryPayload-c.sealedExternal, added)
	return ok && active <= c.bounds.pagePayloadBytes
}

func (m *CandidateAdmissionLedger) chargeProviderStorage(before *CandidateAdmissionLedgerChange, payload, added uint64, boundary *providerLedgerBoundary, seal string) error {
	if m == nil {
		return ErrAdmissionLedger
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	c := m.continuation
	if !m.valid() || c == nil || c.final || m.current != before || payload == 0 && added == 0 || !c.payloadRoom(m, payload) || !c.metadataRoom(m, uint64(len(c.records)), added) {
		return ErrAdmissionLedger
	}
	if c.pending != nil {
		if payload != 0 || boundary != c.pending || !boundary.valid(m) || boundary.sealMetadata != 0 || added == 0 || !lowerHash(seal) {
			return ErrAdmissionLedger
		}
	} else if boundary != nil || seal != "" {
		return ErrAdmissionLedger
	}
	auxiliary, ok := providerUpperSum(c.auxiliaryMetadata, added)
	if !ok {
		return ErrAdmissionLedger
	}
	private, ok := providerUpperSum(c.extraMetadata, auxiliary)
	if !ok {
		return ErrAdmissionLedger
	}
	updated := *m.current
	updated.privateMetadata = private
	updated.privatePayload = c.auxiliaryPayload + payload // Checked by payloadRoom.
	c.auxiliaryPayload = updated.privatePayload
	c.auxiliaryMetadata, m.current = auxiliary, &updated
	if boundary != nil {
		boundary.sealMetadata, boundary.sealDigest = added, seal
	}
	return nil
}

// This token contains an exact arithmetic boundary, not a retained seal or an
// external acknowledgment. It is consumed only by the future typed custodian
// after its page/seal/index closure and frontier transition are validated.
func (m *CandidateAdmissionLedger) prepareProviderBoundary(final bool) (*providerLedgerBoundary, error) {
	if m == nil {
		return nil, ErrAdmissionLedger
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	c := m.continuation
	if !m.valid() || c == nil || c.final || c.pending != nil || c.page+1 >= c.bounds.maximumPages && !final {
		return nil, ErrAdmissionLedger
	}
	records, ok := providerUpperSum(m.current.counts[0], m.current.counts[1], m.current.counts[2])
	if !ok || records == c.sealedRecords || !final && records-c.sealedRecords != c.bounds.pageRecords {
		return nil, ErrAdmissionLedger
	}
	if final {
		for _, value := range m.current.totals.bootstrap.outstanding {
			if value != 0 {
				return nil, ErrAdmissionLedger
			}
		}
	}
	boundary := &providerLedgerBoundary{owner: m, change: m.current, page: c.page, records: records, payload: m.current.payload, metadata: m.current.metadata, final: final}
	page, err := m.providerPage()
	if err != nil {
		return nil, err
	}
	boundary.pageObject = page
	boundary.auxiliaryAtBuild = c.auxiliaryMetadata
	c.pending = boundary
	return boundary, nil
}
