package registry011

import (
	"bytes"
	"encoding/json"
	"testing"
)

func historyRecordUpper(count int) providerHistoryGroupUpper {
	// Symbolic finite replay-test units, not source-derived operating costs.
	// No test here admits a whole program or invokes a protected effect.
	u := providerHistoryGroupUpper{physical: 8192, retainedRaw: 1024, payload: 256, metadata: 128, objects: 4, files: 18, clocks: 17}
	u.execution = candidateWorkVector{uint64(count), 128, 4, 1, 1, 1, 1, 1, 1, 32, 128}
	u.custody[2], u.custody[9], u.custody[10] = 8, 256, 1024
	return u
}

func historyRecordTicket(t *testing.T, count int, stage uint64, previous string) *providerHistoryGroupTicket {
	t.Helper()
	requests := make([]*providerIntegrationRequest, count)
	for i := range requests {
		requests[i] = historyGroupSimple(t)
	}
	bounds, capture := historyGroupBounds()
	program := admissionTestDigest("a")
	table, err := newProviderHistoryGroupTable(program, "content", stage, requests, bounds, capture)
	if err != nil {
		t.Fatal(err)
	}
	ticket, err := newProviderHistoryGroupTicket(program, admissionTestDigest("b"), admissionTestDigest("c"), previous, admissionTestDigest("d"), table, historyRecordUpper(count))
	if err != nil {
		t.Fatal(err)
	}
	return ticket
}

func TestProviderHistoryGroupRecordsPreserveOrderedOriginalBindings(t *testing.T) {
	ticket := historyRecordTicket(t, providerHistoryGroupMembers, 1, admissionTestDigest("e"))
	parsed, err := parseProviderHistoryGroupTicket([]byte(ticket.raw), ticket.program, ticket.profile, ticket.costs, ticket.previous, ticket.inventory, &ticket.table, ticket.upper)
	if err != nil || *parsed != *ticket || !providerHistoryGroupTicketValid(ticket) {
		t.Fatal("exact original ticket failed reconstruction")
	}
	consumption, err := newProviderHistoryGroupConsumption(parsed, admissionTestDigest("f"), admissionTestDigest("1"))
	if err != nil || consumption.consumed != ^uint32(0) {
		t.Fatal("C did not describe all thirty-two disjoint members")
	}
	got, err := parseProviderHistoryGroupConsumption([]byte(consumption.raw), parsed, consumption.previous, consumption.inventory)
	if err != nil || *got != *consumption || !providerHistoryGroupConsumptionValid(got, parsed) {
		t.Fatal("exact original C failed reconstruction")
	}
	results := make([]string, providerHistoryGroupMembers)
	for i := range results {
		results[i] = admissionTestDigest("2") // Equal responses remain distinct ordinals.
	}
	settlement, err := newProviderHistoryGroupSettlement(parsed, got, admissionTestDigest("3"), admissionTestDigest("4"), "complete", "", results)
	if err != nil || settlement.count != providerHistoryGroupMembers {
		t.Fatal("complete ordered result list was collapsed")
	}
	results[0] = admissionTestDigest("5")
	if settlement.results[0] != admissionTestDigest("2") {
		t.Fatal("caller slice changed the retained-result commitment")
	}
	rebuilt, err := parseProviderHistoryGroupSettlement([]byte(settlement.raw), parsed, got, settlement.previous, settlement.inventory, "complete", "", settlement.results[:settlement.count])
	if err != nil || *rebuilt != *settlement {
		t.Fatal("exact original S failed reconstruction")
	}
}

func TestProviderHistoryGroupRecordsRejectIncompleteAndReplacedOriginals(t *testing.T) {
	ticket := historyRecordTicket(t, 2, 1, admissionTestDigest("e"))
	c, err := newProviderHistoryGroupConsumption(ticket, admissionTestDigest("f"), admissionTestDigest("1"))
	if err != nil {
		t.Fatal(err)
	}
	results := []string{admissionTestDigest("2"), admissionTestDigest("3")}
	s, err := newProviderHistoryGroupSettlement(ticket, c, admissionTestDigest("4"), admissionTestDigest("5"), "complete", "", results)
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range [][]byte{nil, make([]byte, candidateLedgerMetadataLimit+1), []byte(ticket.raw + "\n"), bytes.Replace([]byte(ticket.raw), []byte(ticket.profile), []byte(admissionTestDigest("6")), 1)} {
		if _, err := parseProviderHistoryGroupTicket(raw, ticket.program, ticket.profile, ticket.costs, ticket.previous, ticket.inventory, &ticket.table, ticket.upper); err == nil {
			t.Fatal("non-original T was accepted")
		}
	}
	for _, raw := range [][]byte{nil, make([]byte, candidateLedgerMetadataLimit+1), []byte(c.raw + "\n"), bytes.Replace([]byte(c.raw), []byte(`"all_consumed":"3"`), []byte(`"all_consumed":"1"`), 1)} {
		if _, err := parseProviderHistoryGroupConsumption(raw, ticket, c.previous, c.inventory); err == nil {
			t.Fatal("non-original or partial C was accepted")
		}
	}
	for _, raw := range [][]byte{nil, make([]byte, candidateLedgerMetadataLimit+1), []byte(s.raw + "\n"), bytes.Replace([]byte(s.raw), []byte(s.results[0]), []byte(admissionTestDigest("6")), 1)} {
		if _, err := parseProviderHistoryGroupSettlement(raw, ticket, c, s.previous, s.inventory, s.outcome, s.failure, s.results[:s.count]); err == nil {
			t.Fatal("non-original S was accepted")
		}
	}
	for _, row := range []struct {
		outcome, failure string
		results          []string
	}{{"complete", "", results[:1]}, {"complete", admissionTestDigest("6"), results}, {"failed", "", results[:1]}, {"unknown", "", results}, {"complete", "", append(results, admissionTestDigest("6"))}, {"complete", "", []string{"", results[1]}}} {
		if _, err := newProviderHistoryGroupSettlement(ticket, c, s.previous, s.inventory, row.outcome, row.failure, row.results); err == nil {
			t.Fatal("incomplete result or unbound failure became S", row.outcome)
		}
	}
	for _, count := range []int{0, 1, 2} {
		if failed, err := newProviderHistoryGroupSettlement(ticket, c, s.previous, s.inventory, "failed", admissionTestDigest("6"), results[:count]); err != nil || failed.outcome != "failed" || failed.count != count {
			t.Fatal("bounded terminal failure prefix refused", count)
		}
	}
}

func TestProviderHistoryGroupRecordsRefuseChangedTypedInputs(t *testing.T) {
	original := historyRecordTicket(t, 1, 1, admissionTestDigest("e"))
	for _, mutate := range []func(*providerHistoryGroupTicket){
		func(v *providerHistoryGroupTicket) { v.program = "" },
		func(v *providerHistoryGroupTicket) { v.profile = "" },
		func(v *providerHistoryGroupTicket) { v.costs = "" },
		func(v *providerHistoryGroupTicket) { v.previous = "" },
		func(v *providerHistoryGroupTicket) { v.inventory = "" },
		func(v *providerHistoryGroupTicket) { v.raw += " " },
		func(v *providerHistoryGroupTicket) { v.digest = admissionTestDigest("f") },
		func(v *providerHistoryGroupTicket) { v.table.stage = 0 },
		func(v *providerHistoryGroupTicket) { v.table.stage = 1<<53 + 1 },
		func(v *providerHistoryGroupTicket) { v.table.purpose = "unknown" },
		func(v *providerHistoryGroupTicket) { v.table.program = admissionTestDigest("f") },
		func(v *providerHistoryGroupTicket) { v.table.digest = "" },
		func(v *providerHistoryGroupTicket) { v.table.raw += " " },
		func(v *providerHistoryGroupTicket) { v.table.members[0].request.wire += " " },
		func(v *providerHistoryGroupTicket) { v.table.members[0].class = "header" },
		func(v *providerHistoryGroupTicket) { v.upper.execution[0] = 2 },
		func(v *providerHistoryGroupTicket) { v.upper.custody[0] = 1 },
		func(v *providerHistoryGroupTicket) { v.upper.physical = 0 },
	} {
		copy := *original
		mutate(&copy)
		if providerHistoryGroupTicketValid(&copy) {
			t.Fatal("changed typed ticket retained its original binding")
		}
		if _, err := newProviderHistoryGroupConsumption(&copy, original.previous, original.inventory); err == nil {
			t.Fatal("changed T was used to construct C")
		}
	}
	if providerHistoryGroupTicketValid(nil) || providerHistoryGroupConsumptionValid(nil, original) {
		t.Fatal("missing original became a bound record")
	}
	if _, err := newProviderHistoryGroupTicket(original.program, original.profile, original.costs, original.previous, original.inventory, nil, original.upper); err == nil {
		t.Fatal("missing table accepted")
	}
	c, err := newProviderHistoryGroupConsumption(original, original.previous, original.inventory)
	if err != nil {
		t.Fatal(err)
	}
	c.consumed = 0
	if providerHistoryGroupConsumptionValid(c, original) {
		t.Fatal("replaced all-member mask became a valid C")
	}
	if _, err := newProviderHistoryGroupSettlement(original, c, original.previous, original.inventory, "failed", admissionTestDigest("f"), nil); err == nil {
		t.Fatal("partial consumption reached S")
	}
	if json.Unmarshal([]byte(`{}`), original) == nil || json.Unmarshal([]byte(`{}`), c) == nil || json.Unmarshal([]byte(`{}`), &providerHistoryGroupSettlement{}) == nil {
		t.Fatal("JSON reconstructed a typed private record")
	}
	var missing *providerHistoryGroupTicket
	if missing.object() != nil {
		t.Fatal("missing ticket encoded a record")
	}
}

func TestProviderHistoryGroupUpperRejectsOverflowAndMixedResources(t *testing.T) {
	upper := historyRecordUpper(1)
	if !upper.valid(1) {
		t.Fatal("normal finite upper refused")
	}
	for _, mutate := range []func(*providerHistoryGroupUpper){
		func(v *providerHistoryGroupUpper) { v.execution[1] = 0 },
		func(v *providerHistoryGroupUpper) { v.execution[10] = 0 },
		func(v *providerHistoryGroupUpper) { v.custody[1] = 1 },
		func(v *providerHistoryGroupUpper) { v.custody[10] = 0 },
		func(v *providerHistoryGroupUpper) { v.retainedRaw = v.physical + 1 },
		func(v *providerHistoryGroupUpper) { v.payload = v.retainedRaw + 1 },
		func(v *providerHistoryGroupUpper) { v.metadata = 0 },
		func(v *providerHistoryGroupUpper) { v.metadata = v.retainedRaw + 1 },
		func(v *providerHistoryGroupUpper) { v.objects = 0 },
		func(v *providerHistoryGroupUpper) { v.files = v.objects - 1 },
		func(v *providerHistoryGroupUpper) { v.clocks = 0 },
		func(v *providerHistoryGroupUpper) { v.clocks = 1<<20 + 1 },
		func(v *providerHistoryGroupUpper) { v.execution[10], v.custody[10] = ^uint64(0), 1 },
	} {
		copy := upper
		mutate(&copy)
		if copy.valid(1) {
			t.Fatal("incomplete, overflowing or mixed resource upper accepted")
		}
	}
}
