package registry011

import "strconv"

// Closed index values are retained data, not authenticated maps or membership
// proofs. Only full typed replay and the selected live custodian can own a map.
type providerTicketIndexValue struct {
	ticket, segment, input, consumption, settlement string
	sequence                                        uint64
}

func providerTicketIndexFromState(state candidateLedgerTicket) (providerTicketIndexValue, error) {
	return providerTicketIndexFromStateObserved(state, nil)
}

// The future connected owner observes the segment's canonical encode and
// domain hash. Legacy callers keep their original nil-observer contract.
func providerTicketIndexFromStateObserved(state candidateLedgerTicket, meter *providerWorkMeter) (providerTicketIndexValue, error) {
	if state.ticket == nil || state.ticket.raw == "" || state.ticket.intent == nil {
		return providerTicketIndexValue{}, ErrAdmissionLedger
	}
	t := state.ticket
	segment := candidateAdmissionSegmentObserved(t.intent.digest, t.segment, meter)
	if !lowerHash(segment) {
		return providerTicketIndexValue{}, ErrAdmissionLedger
	}
	value := providerTicketIndexValue{ticket: t.digest, sequence: t.sequence, segment: segment, input: t.input, settlement: state.terminal}
	if state.consumption != nil {
		if state.consumption.ticket == nil || state.consumption.ticket.digest != t.digest {
			return providerTicketIndexValue{}, ErrAdmissionLedger
		}
		value.consumption = state.consumption.digest
	}
	return value, nil
}

func (v providerTicketIndexValue) object() map[string]any {
	var consumed, settled any
	if v.consumption != "" {
		consumed = v.consumption
	}
	if v.settlement != "" {
		settled = v.settlement
	}
	return map[string]any{"ticket_digest": v.ticket, "sequence": strconv.FormatUint(v.sequence, 10), "segment": v.segment, "input_digest": v.input, "consumption_digest": consumed, "settlement_digest": settled}
}

func parseProviderTicketIndex(value any) (providerTicketIndexValue, bool) {
	m := contentObject(value, "ticket_digest", "sequence", "segment", "input_digest", "consumption_digest", "settlement_digest")
	sequence, valid := providerCandidateUint(m["sequence"], false)
	if m == nil || !valid || !providerCandidateDigest(m["ticket_digest"]) || !providerCandidateDigest(m["segment"]) || !providerCandidateDigest(m["input_digest"]) || !providerIndexNullableDigest(m["consumption_digest"]) || !providerIndexNullableDigest(m["settlement_digest"]) {
		return providerTicketIndexValue{}, false
	}
	return providerTicketIndexValue{contentString(m["ticket_digest"]), contentString(m["segment"]), contentString(m["input_digest"]), contentString(m["consumption_digest"]), contentString(m["settlement_digest"]), sequence}, true
}

func providerIndexNullableDigest(value any) bool {
	return value == nil || providerCandidateDigest(value)
}

func providerTicketIndexTransition(before *providerTicketIndexValue, after providerTicketIndexValue, kind, entry string) bool {
	if !lowerHash(entry) || !lowerHash(after.ticket) || !lowerHash(after.segment) || !lowerHash(after.input) || after.consumption != "" && !lowerHash(after.consumption) || after.settlement != "" && !lowerHash(after.settlement) {
		return false
	}
	if before == nil {
		return kind == "ticket" && entry == after.ticket && after.consumption == "" && after.settlement == ""
	}
	if before.ticket != after.ticket || before.sequence != after.sequence || before.segment != after.segment || before.input != after.input || before.settlement != "" {
		return false
	}
	switch kind {
	case "consumption":
		return before.consumption == "" && after.consumption == entry && after.settlement == ""
	case "settlement":
		return before.consumption == after.consumption && after.settlement == entry
	}
	return false
}

type providerOperationIndexValue struct {
	operation, kind string
	sequence        uint64
}

func (v providerOperationIndexValue) object() map[string]any {
	return map[string]any{"operation": v.operation, "kind": v.kind, "sequence": strconv.FormatUint(v.sequence, 10)}
}

func parseProviderOperationIndex(value any) (providerOperationIndexValue, bool) {
	m := contentObject(value, "operation", "kind", "sequence")
	sequence, valid := providerCandidateUint(m["sequence"], true)
	kind := contentString(m["kind"])
	if m == nil || !valid || !providerCandidateDigest(m["operation"]) || !providerIndexedCustodyKind(kind) || kind == "enroll" && sequence != 1 {
		return providerOperationIndexValue{}, false
	}
	return providerOperationIndexValue{contentString(m["operation"]), kind, sequence}, true
}

func providerIndexedCustodyKind(kind string) bool {
	switch kind {
	case "enroll", "admit", "consume", "settle", "seal", "verify":
		return true
	}
	return false
}

type providerSegmentIndexValue struct {
	index  uint64
	totals candidateBudgetTotals
}

func (v providerSegmentIndexValue) object() map[string]any {
	return map[string]any{"index": strconv.FormatUint(v.index, 10), "charged": providerIndexWorkObject(v.totals.charged), "outstanding": providerIndexWorkObject(v.totals.outstanding)}
}

func providerIndexWorkObject(work candidateWorkVector) map[string]any {
	value := make(map[string]any, 11)
	for name, amount := range candidateLedgerWorkObject(work) {
		value[name] = amount
	}
	return value
}

func parseProviderSegmentIndex(value any) (providerSegmentIndexValue, bool) {
	m := contentObject(value, "index", "charged", "outstanding")
	index, a := providerCandidateUint(m["index"], false)
	charged, b := candidateAdmissionWork(m["charged"])
	outstanding, c := candidateAdmissionWork(m["outstanding"])
	if m == nil || !a || !b || !c {
		return providerSegmentIndexValue{}, false
	}
	return providerSegmentIndexValue{index, candidateBudgetTotals{charged, outstanding}}, true
}

func providerSegmentIndexTransition(before candidateBudgetTotals, reservation candidateWorkVector, settle bool) (candidateBudgetTotals, bool) {
	after := before
	for i, value := range reservation {
		var ok bool
		if settle {
			if before.outstanding[i] < value {
				return candidateBudgetTotals{}, false
			}
			after.outstanding[i] -= value
			after.charged[i], ok = providerUpperSum(before.charged[i], value)
		} else {
			after.outstanding[i], ok = providerUpperSum(before.outstanding[i], value)
		}
		if !ok {
			return candidateBudgetTotals{}, false
		}
	}
	return after, true
}
