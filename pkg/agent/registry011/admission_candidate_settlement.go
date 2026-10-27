package registry011

// CandidateAdmissionSettlement is proposed settlement data, not an owned work
// measurement or qualified retained Exchange. The ledger model charges the full
// original reservation even when these candidate data claim a complete result.
type CandidateAdmissionSettlement struct {
	raw, digest string
	ticket      *CandidateAdmissionTicket
	outcome     string
	actual      candidateWorkVector
	retained    string
	clock       CandidateClockDiagnostic
}

type CandidateSettlementDiagnostic struct {
	Digest, Ticket, Outcome, Retained string
	Actual                            CandidateWork
	CandidateClock                    CandidateClockDiagnostic
}

func (*CandidateAdmissionSettlement) UnmarshalJSON([]byte) error { return ErrAdmissionCandidate }

func ParseCandidateAdmissionSettlement(ticket *CandidateAdmissionTicket, raw []byte, maximumBytes int) (*CandidateAdmissionSettlement, error) {
	return parseCandidateAdmissionSettlementObserved(ticket, raw, maximumBytes, nil)
}

func parseCandidateAdmissionSettlementObserved(ticket *CandidateAdmissionTicket, raw []byte, maximumBytes int, meter *providerWorkMeter) (*CandidateAdmissionSettlement, error) {
	if ticket == nil || ticket.raw == "" || maximumBytes <= 0 || maximumBytes > 64<<10 {
		return nil, ErrConfiguration
	}
	if len(raw) == 0 || len(raw) > maximumBytes {
		return nil, ErrAdmissionCandidate
	}
	m := contentObject(providerReplayJSON(raw, meter), "format", "ticket_digest", "outcome", "actual_work", "retained_root", "clock_end")
	if m == nil || m["format"] != "sage-provider-budget-settlement-v2" || m["ticket_digest"] != ticket.digest {
		return nil, ErrAdmissionCandidate
	}
	p, policyOK := providerReplayJSON(providerReplayPhaseBytes(ticket.policy, meter), meter).(map[string]any)
	if !policyOK {
		return nil, ErrAdmissionCandidate
	}
	clock, ok := candidateAdmissionClock(m["clock_end"], p["clock"].(map[string]any))
	if !ok || !candidateLedgerClockAdvance(ticket.intent.clock, clock) {
		return nil, ErrAdmissionCandidate
	}
	outcome := contentString(m["outcome"])
	var actual candidateWorkVector
	switch outcome {
	case "complete":
		actual, ok = candidateAdmissionWork(m["actual_work"])
		if !ok || !candidateFits(candidateBudgetTotals{}, actual, ticket.work) || !providerCandidateDigest(m["retained_root"]) || actual[8] != ticket.work[8] || actual[9] == 0 || actual[10] == 0 || uint64(clock.WallMS)+uint64(clock.UncertaintyMS) > ticket.deadline {
			return nil, ErrAdmissionCandidate
		}
		if ticket.kind == "rpc-capture" && (actual[0] != 1 || actual[1] == 0) || ticket.kind == "local-work" && actual[0] != 0 {
			return nil, ErrAdmissionCandidate
		}
	case "failed", "cancelled", "uncertain":
		// A terminal recovery may occur after expiry. It never permits work,
		// extends the original deadline, refunds or drops the retained peaks.
		if m["actual_work"] != nil || m["retained_root"] != nil {
			return nil, ErrAdmissionCandidate
		}
	default:
		return nil, ErrAdmissionCandidate
	}
	digest := providerReplayDigest("sage-provider-budget-settlement-v2", raw, meter)
	retained := providerReplayString(raw, meter)
	if digest == "" || retained == "" {
		return nil, ErrAdmissionCandidate
	}
	return &CandidateAdmissionSettlement{retained, digest, ticket, outcome, actual, contentString(m["retained_root"]), clock}, nil
}

// This checks candidate sample relations only, never a fresh trusted clock.
func candidateLedgerClockAdvance(before, after CandidateClockDiagnostic) bool {
	if before.WallMS < 0 || before.MonotonicMS < 0 || before.UncertaintyMS < 0 || after.UncertaintyMS < 0 || after.ClockID != before.ClockID || after.Epoch != before.Epoch || after.WallMS < before.WallMS || after.MonotonicMS < before.MonotonicMS || after.WallMS < after.UncertaintyMS {
		return false
	}
	wall := uint64(after.WallMS - before.WallMS)
	monotonic := uint64(after.MonotonicMS - before.MonotonicMS)
	uncertainty := uint64(before.UncertaintyMS) + uint64(after.UncertaintyMS)
	if wall >= monotonic {
		return wall-monotonic <= uncertainty
	}
	return monotonic-wall <= uncertainty
}

func (s *CandidateAdmissionSettlement) CanonicalBytes() []byte {
	if s == nil {
		return nil
	}
	return []byte(s.raw)
}

func (s *CandidateAdmissionSettlement) Diagnostic() (CandidateSettlementDiagnostic, error) {
	if s == nil || s.raw == "" {
		return CandidateSettlementDiagnostic{}, ErrAdmissionCandidate
	}
	return CandidateSettlementDiagnostic{s.digest, s.ticket.digest, s.outcome, s.retained, s.actual.diagnostic(), s.clock}, nil
}
