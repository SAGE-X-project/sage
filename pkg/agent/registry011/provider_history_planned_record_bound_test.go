package registry011

import (
	"strings"
	"testing"
)

func TestProviderHistoryPlannedRecordControlMatchesIndependentRows(t *testing.T) {
	for _, tc := range []struct{ members, ticket, consumption, context, settlement uint64 }{
		{1, 4250986, 4508344, 8693788, 75851852},
		{2, 6067494, 6324852, 12326804, 79484868},
		{3, 8236802, 8494160, 16665420, 83823484},
		{32, 224614734, 224872092, 449421284, 516579348},
	} {
		got, ok := providerHistoryPlannedRecordControlUpper(tc.members)
		original, originalOK := providerHistoryRecordControlUpper(tc.members)
		want := providerHistoryRecordControlCost{
			ticket:      candidateWorkVector{2: 1000 * tc.members, 10: tc.ticket},
			consumption: candidateWorkVector{2: 1000 * tc.members, 10: tc.consumption},
			context:     candidateWorkVector{2: 2000 * tc.members, 10: tc.context},
			settlement:  candidateWorkVector{2: 2000 * tc.members, 10: tc.settlement},
		}
		if !ok || !originalOK || got.normal != want || got.envelope != original {
			t.Fatal("planned record parts differ from independent complete control rows", tc.members, got, want)
		}
	}
	for members := uint64(1); members <= providerHistoryGroupMembers; members++ {
		parts, ok := providerHistoryPlannedRecordControlUpper(members)
		if !ok {
			t.Fatal("valid finite planned record cardinality", members)
		}
		for _, pair := range [][2]candidateWorkVector{
			{parts.normal.ticket, parts.envelope.ticket}, {parts.normal.consumption, parts.envelope.consumption},
			{parts.normal.context, parts.envelope.context}, {parts.normal.settlement, parts.envelope.settlement},
		} {
			for coordinate, normal := range pair[0] {
				if normal > pair[1][coordinate] {
					t.Fatal("planned normal escaped unchanged generic envelope", members, coordinate)
				}
			}
		}
	}
}

func TestProviderHistoryPlannedRecordControlFundsOriginalTypedOperations(t *testing.T) {
	for _, count := range []int{1, 32} {
		ticket := historyRecordTicket(t, count, 1, admissionTestDigest("e"))
		consumption, err := newProviderHistoryGroupConsumption(ticket, admissionTestDigest("f"), admissionTestDigest("1"))
		parts, ok := providerHistoryPlannedRecordControlUpper(uint64(count))
		if err != nil || !ok {
			t.Fatal("finite typed record precursor", count, err)
		}
		beforeT, beforeC := *ticket, *consumption
		for _, outcome := range []string{"complete", "failed"} {
			length, failure := count, ""
			if outcome == "failed" {
				length, failure = count-1, admissionTestDigest("5")
			}
			digests := make([]string, length)
			for i := range digests {
				digests[i] = admissionTestDigest("2")
			}
			settlement, err := newProviderHistoryGroupSettlement(ticket, consumption, admissionTestDigest("3"), admissionTestDigest("4"), outcome, failure, digests)
			if err != nil {
				t.Fatal("original complete/failed S DATA precursor", count, outcome, err)
			}
			beforeS := *settlement
			// Separate inert subjects with their full original envelopes chosen
			// before observation. No live role/original-reader authority or I/O.
			for _, row := range []struct {
				name             string
				normal, envelope candidateWorkVector
				run              func(*providerWorkMeter) bool
			}{
				{"T-constructor", parts.normal.ticket, parts.envelope.ticket, func(m *providerWorkMeter) bool {
					v, e := newProviderHistoryGroupTicketObserved(ticket.program, ticket.profile, ticket.costs, ticket.previous, ticket.inventory, &ticket.table, ticket.upper, m)
					return e == nil && v != nil && *v == beforeT
				}},
				{"T-parser-with-caller-copy", parts.normal.ticket, parts.envelope.ticket, func(m *providerWorkMeter) bool {
					v, e := parseProviderHistoryGroupTicketObserved(providerReplayBytes(ticket.raw, m), ticket.program, ticket.profile, ticket.costs, ticket.previous, ticket.inventory, &ticket.table, ticket.upper, m)
					return e == nil && v != nil && *v == beforeT
				}},
				{"C-constructor", parts.normal.consumption, parts.envelope.consumption, func(m *providerWorkMeter) bool {
					v, e := newProviderHistoryGroupConsumptionObserved(ticket, consumption.previous, consumption.inventory, m)
					return e == nil && v != nil && *v == beforeC
				}},
				{"C-parser-with-caller-copy", parts.normal.consumption, parts.envelope.consumption, func(m *providerWorkMeter) bool {
					v, e := parseProviderHistoryGroupConsumptionObserved(providerReplayBytes(consumption.raw, m), ticket, consumption.previous, consumption.inventory, m)
					return e == nil && v != nil && *v == beforeC
				}},
				{"S-constructor-with-fixed-array-equality", parts.normal.settlement, parts.envelope.settlement, func(m *providerWorkMeter) bool {
					v, e := newProviderHistoryGroupSettlementObserved(ticket, consumption, settlement.previous, settlement.inventory, outcome, failure, digests, m)
					return e == nil && v != nil && providerHistoryGroupSettlementEqualObserved(v, settlement, m)
				}},
				{"S-parser-with-caller-copy", parts.normal.settlement, parts.envelope.settlement, func(m *providerWorkMeter) bool {
					v, e := parseProviderHistoryGroupSettlementObserved(providerReplayBytes(settlement.raw, m), ticket, consumption, settlement.previous, settlement.inventory, outcome, failure, digests, m)
					return e == nil && v != nil && *v == beforeS
				}},
				{"context-first-ordinal", parts.normal.context, parts.envelope.context, func(m *providerWorkMeter) bool {
					return providerHistoryResultContextObserved(ticket, consumption, 0, m)
				}},
				{"context-last-ordinal", parts.normal.context, parts.envelope.context, func(m *providerWorkMeter) bool {
					return providerHistoryResultContextObserved(ticket, consumption, count-1, m)
				}},
			} {
				meter := providerHistoryContentUnitMeter(row.envelope)
				if !row.run(meter) || meter.failed || *ticket != beforeT || *consumption != beforeC || *settlement != beforeS {
					t.Fatal("original typed record refused or changed DATA", count, outcome, row.name)
				}
				for coordinate, units := range meter.actual {
					if units > row.normal[coordinate] {
						t.Fatal("original record escaped planned entered-control row", count, outcome, row.name, coordinate, units)
					}
				}
			}
		}
	}
}

func TestProviderHistoryPlannedRecordControlKeepsWideGuardsAndGenericData(t *testing.T) {
	parts, ok := providerHistoryPlannedRecordControlUpper(1)
	ticket := historyRecordTicket(t, 1, 1, admissionTestDigest("e"))
	if !ok {
		t.Fatal("one-record precursor")
	}
	before := *ticket
	short := providerHistoryContentUnitMeter(parts.normal.ticket)
	if v, err := newProviderHistoryGroupTicketObserved(ticket.program, ticket.profile, ticket.costs, ticket.previous, ticket.inventory, &ticket.table, ticket.upper, short); err == nil || v != nil || !short.failed || short.actual[10] == 0 || *ticket != before {
		t.Fatal("normal DATA incorrectly funded original wide incoming guards")
	}
	prefix := short.actual
	if short.add(candidateWorkVector{}) || short.actual != prefix {
		t.Fatal("refused original meter renewed its allowance")
	}
	full := providerHistoryContentUnitMeter(parts.envelope.ticket)
	if v, err := newProviderHistoryGroupTicketObserved(ticket.program, ticket.profile, ticket.costs, ticket.previous, ticket.inventory, &ticket.table, ticket.upper, full); err != nil || v == nil || *v != before || full.failed {
		t.Fatal("unchanged full record envelope refused original DATA", err)
	}
	// A bounded generic template is deliberately outside the planned grammar.
	// Original generic behavior stays available under its original envelope.
	request := historyGroupRequest(t, "eth_getCode", []any{strings.Repeat("a", 1024), "0x1"})
	bounds, capture := historyGroupBounds()
	table, err := newProviderHistoryGroupTable(ticket.program, "content", 1, []*providerIntegrationRequest{request}, bounds, capture)
	if err != nil {
		t.Fatal("generic finite table precursor", err)
	}
	wide, err := newProviderHistoryGroupTicket(ticket.program, ticket.profile, ticket.costs, ticket.previous, ticket.inventory, table, ticket.upper)
	if err != nil {
		t.Fatal("generic finite T precursor", err)
	}
	consumption, err := newProviderHistoryGroupConsumption(wide, admissionTestDigest("f"), admissionTestDigest("1"))
	if err != nil {
		t.Fatal("generic finite C precursor", err)
	}
	beforeWide, beforeC := *wide, *consumption
	meter := providerHistoryContentUnitMeter(parts.envelope.context)
	if !providerHistoryResultContextObserved(wide, consumption, 0, meter) || meter.failed || *wide != beforeWide || *consumption != beforeC ||
		meter.actual[2] <= parts.normal.context[2] {
		t.Fatal("generic context was narrowed or misrepresented as planned normal DATA", meter.actual)
	}
}

func TestProviderHistoryPlannedRecordControlRejectsInvalidCardinalities(t *testing.T) {
	for _, members := range []uint64{0, 33, ^uint64(0)} {
		if parts, ok := providerHistoryPlannedRecordControlUpper(members); ok || parts != (providerHistoryPlannedRecordControlCost{}) {
			t.Fatal("invalid planned cardinality returned partial control parts", members, parts, ok)
		}
	}
}
