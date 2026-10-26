package registry011

import "math"

// CandidateConsumption is proposed consumption metadata only. It does not
// durably consume a ticket, return a work permit or dispatch anything. The
// qualified owned executor/custody/unique-use ledger is still unimplemented.
type CandidateConsumption struct {
	raw, digest string
	ticket      *CandidateAdmissionTicket
	sequence    uint64
	previous    string
	clock       CandidateClockDiagnostic
}

type CandidateConsumptionDiagnostic struct {
	Digest, Ticket, Previous string
	Sequence                 uint64
	CandidateClock           CandidateClockDiagnostic
}

func (*CandidateConsumption) UnmarshalJSON([]byte) error { return ErrAdmissionCandidate }

// ParseCandidateConsumption checks a closed record against the exact candidate
// ticket, immutable original intent and bounded clock relations. All samples
// remain JSON-derived candidate data and cannot serve as trusted clock inputs.
func ParseCandidateConsumption(ticket *CandidateAdmissionTicket, raw []byte, maximumBytes int) (*CandidateConsumption, error) {
	if ticket == nil || ticket.raw == "" || maximumBytes <= 0 || maximumBytes > 64<<10 {
		return nil, ErrConfiguration
	}
	if len(raw) == 0 || len(raw) > maximumBytes {
		return nil, ErrAdmissionCandidate
	}
	m := contentObject(contentJSON(raw), "format", "ticket_digest", "namespace", "instance_key", "source_id", "generation", "fence", "sequence", "previous_consumption", "kind", "input_digest", "clock_start")
	id := ticket.intent.identity
	if m == nil || m["format"] != "sage-provider-budget-consumption-v1" || m["ticket_digest"] != ticket.digest || m["namespace"] != id.Namespace || m["instance_key"] != id.Instance || m["source_id"] != id.SourceID || m["kind"] != ticket.kind || m["input_digest"] != ticket.input {
		return nil, ErrAdmissionCandidate
	}
	generation, a := providerCandidateUint(m["generation"], false)
	fence, b := providerCandidateUint(m["fence"], true)
	sequence, c := providerCandidateUint(m["sequence"], false)
	if !a || !b || !c || generation != id.Generation || fence != ticket.intent.fence || sequence == 0 && m["previous_consumption"] != nil || sequence != 0 && !providerCandidateDigest(m["previous_consumption"]) {
		return nil, ErrAdmissionCandidate
	}
	policy := contentJSON([]byte(ticket.policy)).(map[string]any)
	clock, ok := candidateAdmissionClock(m["clock_start"], policy["clock"].(map[string]any))
	if !ok || !candidateConsumptionClock(ticket, clock, policy) {
		return nil, ErrAdmissionCandidate
	}
	return &CandidateConsumption{string(raw), providerCandidateHash("sage-provider-budget-consumption-v1", raw), ticket, sequence, contentString(m["previous_consumption"]), clock}, nil
}

func candidateConsumptionClock(ticket *CandidateAdmissionTicket, end CandidateClockDiagnostic, policy map[string]any) bool {
	start := ticket.intent.clock
	if start.Epoch != end.Epoch || end.WallMS < start.WallMS || end.MonotonicMS < start.MonotonicMS || end.WallMS < end.UncertaintyMS {
		return false
	}
	wall := uint64(end.WallMS - start.WallMS)
	elapsed := uint64(end.MonotonicMS - start.MonotonicMS)
	uncertainty := uint64(start.UncertaintyMS) + uint64(end.UncertaintyMS)
	var difference uint64
	if wall >= elapsed {
		difference = wall - elapsed
	} else {
		difference = elapsed - wall
	}
	if difference > uncertainty || uint64(end.WallMS)+uint64(end.UncertaintyMS) > ticket.deadline {
		return false
	}
	limits := policy["limits"].(map[string]any)
	duration, _ := providerCandidateUint(limits["operation"].(map[string]any)["deadline_ms"], true)
	lower := uint64(end.WallMS - end.UncertaintyMS)
	return duration <= math.MaxInt64-lower && ticket.deadline <= lower+duration
}

// CheckCandidateConsumptionSuccessor checks only adjacent immutable records.
// The actual store must separately reject every reused ticket across the whole
// retained prefix and record consumption durably before work. This comparison
// is never an operation-consumption ACK and cannot satisfy that requirement.
func CheckCandidateConsumptionSuccessor(before, after *CandidateConsumption) error {
	if before == nil || after == nil || before.raw == "" || after.raw == "" || before.sequence == math.MaxUint64 || after.sequence != before.sequence+1 || after.previous != before.digest || before.ticket.intent.digest != after.ticket.intent.digest || before.ticket.digest == after.ticket.digest || after.clock.Epoch != before.clock.Epoch || after.clock.WallMS < before.clock.WallMS || after.clock.MonotonicMS < before.clock.MonotonicMS {
		return ErrAdmissionCandidate
	}
	return nil
}
func (c *CandidateConsumption) CanonicalBytes() []byte {
	if c == nil {
		return nil
	}
	return []byte(c.raw)
}
func (c *CandidateConsumption) Diagnostic() (CandidateConsumptionDiagnostic, error) {
	if c == nil || c.raw == "" {
		return CandidateConsumptionDiagnostic{}, ErrAdmissionCandidate
	}
	return CandidateConsumptionDiagnostic{c.digest, c.ticket.digest, c.previous, c.sequence, c.clock}, nil
}
