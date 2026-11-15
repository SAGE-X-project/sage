package registry011

import (
	"strings"
	"testing"
)

// These are finite DATA observations only. This synthetic observer is not a
// selected store, whole-program reservation, clock role or dispatch owner.
func historyGroupCodecMeter() *providerWorkMeter {
	var upper candidateWorkVector
	upper[2], upper[10] = 1<<30, 1<<60
	return newProviderWorkMeter(&providerIntegration{}, providerOperationReservation{work: upper, execution: upper})
}

func TestProviderHistoryGroupObservedRecordsPreserveOriginalTranscript(t *testing.T) {
	bounds, capture := historyGroupBounds()
	program := admissionTestDigest("a")
	requests := []*providerIntegrationRequest{
		historyGroupRequest(t, "eth_getBlockByNumber", []any{"finalized", false}),
		historyGroupRequest(t, "debug_traceTransaction", []any{program, map[string]any{"tracer": "stateGasTracer", "reexec": 0}}),
	}
	m := historyGroupCodecMeter()
	plain, err := newProviderHistoryGroupTable(program, "content", 1, requests, bounds, capture)
	if err != nil {
		t.Fatal(err)
	}
	table, err := parseProviderHistoryGroupTableObserved([]byte(plain.raw), program, "content", 1, requests, bounds, capture, m)
	if err != nil || *table != *plain {
		t.Fatal("observed request reconstruction changed the immutable table", err)
	}
	a, b := historyLedgerFixture(t, historyLedgerMaximum()), historyLedgerFixture(t, historyLedgerMaximum())
	ticket, err := newProviderHistoryGroupTicket(a.program, a.profile, a.costs, a.head, admissionTestDigest("1"), plain, historyRecordUpper(2))
	if err != nil {
		t.Fatal(err)
	}
	originalT, err := parseProviderHistoryGroupTicketObserved([]byte(ticket.raw), b.program, b.profile, b.costs, b.head, ticket.inventory, table, ticket.upper, m)
	if err != nil || *originalT != *ticket || !providerHistoryGroupTicketValidObserved(originalT, m) || a.admit(ticket) != nil || b.admitObserved(originalT, m) != nil {
		t.Fatal("observed T changed the typed transcript", err)
	}
	if a.head != b.head || a.ticket == b.ticket || b.ticket == originalT || !b.validObserved(m) {
		t.Fatal("independent models shared mutable ticket state or disagreed")
	}
	c, err := newProviderHistoryGroupConsumption(ticket, a.head, admissionTestDigest("2"))
	if err != nil {
		t.Fatal(err)
	}
	originalC, err := parseProviderHistoryGroupConsumptionObserved([]byte(c.raw), originalT, b.head, c.inventory, m)
	if err != nil || *originalC != *c || !providerHistoryGroupConsumptionValidObserved(originalC, originalT, m) || a.consume(c) != nil || b.consumeObserved(originalC, m) != nil {
		t.Fatal("observed all-member consumption disagreed", err)
	}
	if a.head != b.head || a.consumption == b.consumption || b.consumption == originalC || b.consumption.consumed != 3 || !b.validObserved(m) {
		t.Fatal("C was shared, partial or changed the replay root")
	}
	results := []string{admissionTestDigest("3"), admissionTestDigest("4")}
	s, err := newProviderHistoryGroupSettlement(ticket, c, a.head, admissionTestDigest("5"), "complete", "", results)
	if err != nil {
		t.Fatal(err)
	}
	originalS, err := parseProviderHistoryGroupSettlementObserved([]byte(s.raw), originalT, originalC, b.head, s.inventory, s.outcome, s.failure, results, m)
	if err != nil || *originalS != *s || a.settle(s) != nil || b.settleObserved(originalS, m) != nil || a.head != b.head || a.charged != b.charged || !b.validObserved(m) {
		t.Fatal("observed complete settlement differed from the original replay", err)
	}
	if m.failed || m.closed || m.actual[2] == 0 || m.actual[10] == 0 || m.actual[0] != 0 || m.actual[1] != 0 || m.actual[9] != 0 || m.clocks != 0 {
		t.Fatal("codec work was absent or manufactured an external operation", m.actual)
	}
}

func TestProviderHistoryGroupObservedRefusalCannotFallBackToPureReplay(t *testing.T) {
	for _, phase := range []string{"table", "ticket", "consumption", "settlement", "admit", "consume", "settle", "validate"} {
		for _, failure := range []string{"exhausted", "closed", "copied"} {
			t.Run(phase+"/"+failure, func(t *testing.T) {
				l := historyLedgerFixture(t, historyLedgerMaximum())
				ticket := historyRecordTicket(t, 1, 1, l.head)
				c, err := newProviderHistoryGroupConsumption(ticket, admissionTestDigest("1"), admissionTestDigest("2"))
				if err != nil {
					t.Fatal(err)
				}
				if phase == "consume" || phase == "settle" {
					if l.admit(ticket) != nil {
						t.Fatal("finite preparation")
					}
					c, err = newProviderHistoryGroupConsumption(ticket, l.head, c.inventory)
					if err != nil {
						t.Fatal(err)
					}
				}
				if phase == "settle" && l.consume(c) != nil {
					t.Fatal("finite C preparation")
				}
				s, err := newProviderHistoryGroupSettlement(ticket, c, l.head, admissionTestDigest("3"), "complete", "", []string{admissionTestDigest("4")})
				if err != nil {
					t.Fatal(err)
				}
				before, charged, sequence := l.head, l.charged, l.sequence
				m := historyGroupCodecMeter()
				switch failure {
				case "exhausted":
					m.upper[10] = 0
				case "closed":
					m.closed = true
				case "copied":
					m = &providerWorkMeter{owner: m, integration: m.integration, upper: m.upper}
				}
				refused := false
				switch phase {
				case "table":
					bounds, capture := historyGroupBounds()
					got, e := newProviderHistoryGroupTableObserved(ticket.program, "content", 1, []*providerIntegrationRequest{historyGroupSimple(t)}, bounds, capture, m)
					refused = e != nil && got == nil
				case "ticket":
					got, e := newProviderHistoryGroupTicketObserved(ticket.program, ticket.profile, ticket.costs, ticket.previous, ticket.inventory, &ticket.table, ticket.upper, m)
					refused = e != nil && got == nil
				case "consumption":
					got, e := newProviderHistoryGroupConsumptionObserved(ticket, c.previous, c.inventory, m)
					refused = e != nil && got == nil
				case "settlement":
					got, e := newProviderHistoryGroupSettlementObserved(ticket, c, s.previous, s.inventory, s.outcome, s.failure, s.results[:s.count], m)
					refused = e != nil && got == nil
				case "admit":
					refused = l.admitObserved(ticket, m) != nil && l.failed
				case "consume":
					refused = l.consumeObserved(c, m) != nil && l.failed
				case "settle":
					refused = l.settleObserved(s, m) != nil && l.failed
				case "validate":
					refused = !l.validObserved(m)
				}
				if !refused || l.head != before || l.charged != charged || l.sequence != sequence || m.actual != (candidateWorkVector{}) {
					t.Fatal("unfunded codec returned data, advanced a prefix or refunded charges", m.actual)
				}
			})
		}
	}
}

func TestProviderHistoryGroupObservedSettlementRejectsUnusedResultTail(t *testing.T) {
	l := historyLedgerFixture(t, historyLedgerMaximum())
	ticket := historyRecordTicket(t, 1, 1, l.head)
	if l.admit(ticket) != nil {
		t.Fatal("finite T")
	}
	c, err := newProviderHistoryGroupConsumption(ticket, l.head, admissionTestDigest("1"))
	if err != nil || l.consume(c) != nil {
		t.Fatal("finite C", err)
	}
	s, err := newProviderHistoryGroupSettlement(ticket, c, l.head, admissionTestDigest("2"), "complete", "", []string{admissionTestDigest("3")})
	if err != nil {
		t.Fatal(err)
	}
	s.results[31] = admissionTestDigest("4") // Not part of the encoded prefix.
	m, head, charge := historyGroupCodecMeter(), l.head, l.charged
	if l.settleObserved(s, m) == nil || !l.failed || l.head != head || l.charged != charge || m.actual[10] == 0 {
		t.Fatal("unencoded result tail became a retained replay state")
	}
}

func TestProviderHistoryGroupRootFailurePreservesPreviouslyChargedWork(t *testing.T) {
	l := historyLedgerFixture(t, historyLedgerMaximum())
	ticket := historyRecordTicket(t, 1, 1, l.head)
	if l.admit(ticket) != nil {
		t.Fatal("finite T preparation")
	}
	c, err := newProviderHistoryGroupConsumption(ticket, l.head, admissionTestDigest("1"))
	if err != nil || l.consume(c) != nil {
		t.Fatal("finite all-member debit", err)
	}
	s, err := newProviderHistoryGroupSettlement(ticket, c, l.head, admissionTestDigest("2"), "complete", "", []string{admissionTestDigest("3")})
	if err != nil {
		t.Fatal(err)
	}
	charged, head, mask := l.charged, l.head, l.consumption.consumed
	m := historyGroupCodecMeter()
	m.upper[10] = 0
	// Exercise the private root-publication boundary after a real typed T/C
	// debit, rather than claim a first-event refusal reached this late step.
	// No filesystem transaction, live phase role or retained ACK is created.
	if l.advanceObserved("settled", s.digest, m) == nil || !l.failed || !m.failed || l.head != head || l.charged != charged || l.consumption.consumed != mask || m.actual != (candidateWorkVector{}) {
		t.Fatal("failed root publication restored a debit or manufactured a valid next root")
	}
	if l.admit(ticket) == nil || l.consume(c) == nil || l.settle(s) == nil || l.charged != charged || l.head != head {
		t.Fatal("a quarantined root boundary resumed or refunded the earlier group")
	}
}

func TestProviderHistoryGroupTicketBoundsPrivateTableBeforeEncoding(t *testing.T) {
	ticket := historyRecordTicket(t, 1, 1, admissionTestDigest("e"))
	for _, mode := range []string{"raw-empty", "raw-too-large", "active-method"} {
		t.Run(mode, func(t *testing.T) {
			table := ticket.table
			m := historyGroupCodecMeter()
			switch mode {
			case "raw-empty":
				table.raw = ""
			case "raw-too-large":
				table.raw = strings.Repeat("x", providerHistoryGroupTableBytes+1)
			case "active-method":
				// Inert bounded DATA corruption only. The valid request's tiny
				// params decode has no capacity; it must refuse before a table
				// encoder can see this changed active member string.
				table.members[0].request.method = strings.Repeat("m", 8<<10+1)
				m.upper[2] = 0
			}
			got, err := newProviderHistoryGroupTicketObserved(ticket.program, ticket.profile, ticket.costs, ticket.previous, ticket.inventory, &table, ticket.upper, m)
			if err == nil || got != nil || m.actual[2] != 0 || m.actual[9] != 0 {
				t.Fatal("oversized private table returned a ticket or performed payload work")
			}
			if mode == "active-method" {
				if !m.failed || m.actual[10] == 0 || m.actual[10] > uint64(len(table.members[0].request.params))+1 {
					t.Fatal("table encoding preceded closed request reconstruction", m.actual)
				}
			} else if m.actual != (candidateWorkVector{}) {
				t.Fatal("invalid table length reached a byte primitive", m.actual)
			}
		})
	}
}
