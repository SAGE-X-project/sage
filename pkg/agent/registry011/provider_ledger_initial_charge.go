package registry011

// This unexported value checks arithmetic DATA only. The future first-I
// bridge must derive it independently from the selected V4 original closure.
// Constructing it, or a ledger accepting it, grants no ACK or work permission.
type providerLedgerInitialCharge struct {
	work     candidateWorkVector
	retained uint64 // V4 live-used is normatively this same conservative upper.
}

func (*providerLedgerInitialCharge) UnmarshalJSON([]byte) error { return ErrAdmissionLedger }

func (s providerLedgerInitialCharge) fits(m *CandidateAdmissionLedger) bool {
	if s.retained == 0 || s.retained > m.live || s.retained > m.retained || s.work[9] > s.retained || s.work[0] != 0 || s.work[1] != 0 {
		return false
	}
	for coordinate, n := range s.work {
		if n > m.limits[2][coordinate] || coordinate != 2 && coordinate != 9 && coordinate != 10 && n != 0 {
			return false
		}
	}
	return s.work[2] > 0 && s.work[9] > 0 && s.work[10] > 0
}
