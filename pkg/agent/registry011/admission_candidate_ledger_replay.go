package registry011

import "bytes"

// CandidateLedgerReplayEntry is explicit candidate data only. Exactly one typed
// record is required. ExpectedBudget compares immutable data, not a trusted pin,
// independent custodian acknowledgement or permission to resume actual work.
type CandidateLedgerReplayEntry struct {
	Ticket         *CandidateAdmissionTicket
	Consumption    *CandidateConsumption
	Settlement     *CandidateAdmissionSettlement
	ExpectedBudget []byte
}

// ReplayCandidateAdmissionLedger reconstructs a model in one bounded pass.
// It authenticates neither bytes nor absence and performs no writes or work.
// Every failure returns no model; no partial reconstructed prefix escapes.
func ReplayCandidateAdmissionLedger(policy *ProviderPolicyCandidate, intent *CandidateAdmissionIntent, records []CandidateLedgerReplayEntry, bounds CandidateLedgerBounds) (*CandidateAdmissionLedger, error) {
	if !candidateLedgerBoundsValid(bounds) {
		return nil, ErrConfiguration
	}
	if len(records) == 0 || len(records) > bounds.MaximumRecords || records[0].Ticket == nil {
		return nil, ErrAdmissionLedger
	}
	var payload, expected uint64
	for _, record := range records {
		raw, count := candidateLedgerReplayPayload(record)
		if count != 1 || len(raw) == 0 || len(raw) > 64<<10 || len(record.ExpectedBudget) == 0 || len(record.ExpectedBudget) > candidateLedgerMetadataLimit || !candidateCapacity(0, payload, uint64(len(raw)), bounds.MaximumPayloadBytes) || !candidateCapacity(0, expected, uint64(len(record.ExpectedBudget)), bounds.MaximumMetadataBytes) {
			return nil, ErrAdmissionLedger
		}
		payload += uint64(len(raw))
		expected += uint64(len(record.ExpectedBudget))
	}
	// Expected roots are a subset of generated metadata. Keeping this separate
	// from payload preserves creation -> exact replay under identical bounds.
	m, err := NewCandidateAdmissionLedger(policy, intent, records[0].Ticket, bounds)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal([]byte(m.current.raw), records[0].ExpectedBudget) {
		return nil, ErrAdmissionLedger
	}
	for _, record := range records[1:] {
		var change *CandidateAdmissionLedgerChange
		switch {
		case record.Ticket != nil:
			change, err = m.AppendCandidateTicket(record.Ticket)
		case record.Consumption != nil:
			change, err = m.AppendCandidateConsumption(record.Consumption)
		case record.Settlement != nil:
			change, err = m.AppendCandidateSettlement(record.Settlement)
		}
		if err != nil {
			return nil, err
		}
		if !bytes.Equal([]byte(change.raw), record.ExpectedBudget) {
			return nil, ErrAdmissionLedger
		}
	}
	return m, nil
}

func candidateLedgerReplayPayload(record CandidateLedgerReplayEntry) (string, int) {
	var raw string
	count := 0
	if record.Ticket != nil {
		raw = record.Ticket.raw
		count++
	}
	if record.Consumption != nil {
		raw = record.Consumption.raw
		count++
	}
	if record.Settlement != nil {
		raw = record.Settlement.raw
		count++
	}
	return raw, count
}
