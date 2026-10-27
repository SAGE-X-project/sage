package registry011

import "testing"

func TestProviderFirstOperationTicketIndexObservesTheSegmentWithoutChangingItsBytes(t *testing.T) {
	_, ledger, _ := providerIndexFixture(t)
	state := ledger.tickets[ledger.lastTicket.digest]
	legacy, err := providerTicketIndexFromState(state)
	if err != nil {
		t.Fatal(err)
	}
	meter := newProviderWorkMeter(&providerIntegration{}, providerOperationReservation{execution: candidateWorkVector{10: 1 << 40}})
	observed, err := providerTicketIndexFromStateObserved(state, meter)
	if err != nil || observed != legacy || meter.actual[10] == 0 || meter.actual[2] != 0 || meter.failed || meter.closed {
		t.Fatal("segment formatting changed or its encode/hash disappeared from actual work", err, meter.actual)
	}
	for coordinate, actual := range meter.actual {
		if coordinate != 10 && actual != 0 {
			t.Fatal("formatting created a request, read, write or decoded result", coordinate)
		}
	}
	// The encoder's declared preflight is distinct from its smaller actual.
	// An unavailable preflight refuses before producing a usable index value.
	short := newProviderWorkMeter(&providerIntegration{}, providerOperationReservation{execution: candidateWorkVector{10: 4096 * 4096}})
	if value, err := providerTicketIndexFromStateObserved(state, short); err == nil || value != (providerTicketIndexValue{}) || !short.failed || short.actual != (candidateWorkVector{}) {
		t.Fatal("one-unit-short declared encoding allowance produced an index", err, short.actual)
	}
}

func TestProviderFirstOperationTicketIndexCannotDropAnUnavailableObserver(t *testing.T) {
	_, ledger, _ := providerIndexFixture(t)
	for _, mode := range []string{"closed", "failed", "detached", "no-owner", "no-ticket", "no-intent", "no-raw", "foreign-consumption"} {
		t.Run(mode, func(t *testing.T) {
			meter := newProviderWorkMeter(&providerIntegration{}, providerOperationReservation{execution: candidateWorkVector{10: 1 << 40}})
			ticket := *ledger.lastTicket
			state := candidateLedgerTicket{ticket: &ticket}
			switch mode {
			case "closed":
				meter.closed = true
			case "failed":
				meter.failed = true
			case "detached":
				meter.owner = nil
			case "no-owner":
				meter.integration = nil
			case "no-ticket":
				state.ticket = nil
			case "no-intent":
				ticket.intent = nil
			case "no-raw":
				ticket.raw = ""
			case "foreign-consumption":
				state.consumption = &CandidateConsumption{}
			}
			if value, err := providerTicketIndexFromStateObserved(state, meter); err == nil || value != (providerTicketIndexValue{}) {
				t.Fatal("finite missing state became a usable index", mode)
			}
		})
	}
}
