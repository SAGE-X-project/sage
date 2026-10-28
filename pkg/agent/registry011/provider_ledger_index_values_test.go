package registry011

import (
	"math"
	"testing"
)

func TestProviderIndexValuesClosedSchemas(t *testing.T) {
	_, m, _ := providerIndexFixture(t)
	ticket, err := providerTicketIndexFromState(m.tickets[m.lastTicket.digest])
	if err != nil {
		t.Fatal(err)
	}
	if value, ok := parseProviderTicketIndex(ticket.object()); !ok || value != ticket {
		t.Fatal("ticket roundtrip")
	}
	operation := providerOperationIndexValue{admissionTestDigest("d"), "seal", 2}
	if value, ok := parseProviderOperationIndex(operation.object()); !ok || value != operation {
		t.Fatal("seal operation roundtrip")
	}
	operation.kind = "verify"
	if value, ok := parseProviderOperationIndex(operation.object()); !ok || value != operation {
		t.Fatal("receipt operation roundtrip")
	}
	segment := providerSegmentIndexValue{0, candidateBudgetTotals{outstanding: m.lastTicket.work}}
	if value, ok := parseProviderSegmentIndex(segment.object()); !ok || value != segment {
		t.Fatal("segment roundtrip")
	}
	for _, kind := range []string{"ticket", "operation", "segment"} {
		var good map[string]any
		switch kind {
		case "ticket":
			good = ticket.object()
		case "operation":
			good = operation.object()
		case "segment":
			good = segment.object()
		}
		for key := range good {
			for _, mode := range []string{"missing", "wrong"} {
				t.Run(kind+"/"+key+"/"+mode, func(t *testing.T) {
					object := contentJSON(providerIntegrationJSON(good)).(map[string]any)
					if mode == "missing" {
						delete(object, key)
					} else {
						object[key] = "invalid"
					}
					valid := false
					switch kind {
					case "ticket":
						_, valid = parseProviderTicketIndex(object)
					case "operation":
						_, valid = parseProviderOperationIndex(object)
					case "segment":
						_, valid = parseProviderSegmentIndex(object)
					}
					if valid {
						t.Fatal("invalid closed value accepted")
					}
				})
			}
		}
	}
	for _, state := range []candidateLedgerTicket{{}, {ticket: &CandidateAdmissionTicket{raw: "{}"}}, {ticket: m.lastTicket, consumption: &CandidateConsumption{}}} {
		if _, err := providerTicketIndexFromState(state); err == nil {
			t.Fatal("partial ticket state")
		}
	}
}

func TestProviderIndexTicketTransitionsCannotClearTombstones(t *testing.T) {
	_, m, _ := providerIndexFixture(t)
	first, _ := providerTicketIndexFromState(m.tickets[m.lastTicket.digest])
	if !providerTicketIndexTransition(nil, first, "ticket", first.ticket) {
		t.Fatal("initial ticket")
	}
	consumed := first
	consumed.consumption = admissionTestDigest("b")
	if !providerTicketIndexTransition(&first, consumed, "consumption", consumed.consumption) {
		t.Fatal("one consumption")
	}
	settled := consumed
	settled.settlement = admissionTestDigest("c")
	if !providerTicketIndexTransition(&consumed, settled, "settlement", settled.settlement) {
		t.Fatal("one terminal settlement")
	}
	failed := first
	failed.settlement = admissionTestDigest("d")
	if !providerTicketIndexTransition(&first, failed, "settlement", failed.settlement) {
		t.Fatal("unconsumed failure terminal")
	}
	for _, name := range []string{"clear", "reconsume", "resettle", "ticket", "sequence", "segment", "input", "digest", "unknown"} {
		t.Run(name, func(t *testing.T) {
			before, after, kind, entry := consumed, settled, "settlement", settled.settlement
			switch name {
			case "clear":
				before, after = settled, first
			case "reconsume":
				after, kind, entry = consumed, "consumption", consumed.consumption
			case "resettle":
				before = settled
			case "ticket":
				after.ticket = admissionTestDigest("e")
			case "sequence":
				after.sequence++
			case "segment":
				after.segment = admissionTestDigest("e")
			case "input":
				after.input = admissionTestDigest("e")
			case "digest":
				entry = "invalid"
			case "unknown":
				kind = "unknown"
			}
			if providerTicketIndexTransition(&before, after, kind, entry) {
				t.Fatal("cleared or replaced an immutable ticket value")
			}
		})
	}
}

func TestProviderIndexSegmentArithmeticChargesTheFullReservation(t *testing.T) {
	work := candidateWorkVector{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	reserved, ok := providerSegmentIndexTransition(candidateBudgetTotals{}, work, false)
	if !ok || reserved.outstanding != work || reserved.charged != (candidateWorkVector{}) {
		t.Fatal("reservation")
	}
	charged, ok := providerSegmentIndexTransition(reserved, work, true)
	if !ok || charged.charged != work || charged.outstanding != (candidateWorkVector{}) {
		t.Fatal("full terminal charge")
	}
	if _, ok := providerSegmentIndexTransition(charged, work, true); ok {
		t.Fatal("same reservation settled twice")
	}
	for index := range work {
		before := candidateBudgetTotals{}
		before.outstanding[index] = math.MaxUint64
		if _, ok := providerSegmentIndexTransition(before, work, false); ok {
			t.Fatal("outstanding overflow", index)
		}
		before = reserved
		before.charged[index] = math.MaxUint64
		if _, ok := providerSegmentIndexTransition(before, work, true); ok {
			t.Fatal("charged overflow", index)
		}
	}
}
