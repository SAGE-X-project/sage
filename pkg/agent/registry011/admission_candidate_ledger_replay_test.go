package registry011

import (
	"bytes"
	"testing"
)

func ledgerTestReplayFixture(t *testing.T) (*ProviderPolicyCandidate, *CandidateAdmissionIntent, []CandidateLedgerReplayEntry, *CandidateAdmissionLedgerChange) {
	t.Helper()
	p, i, first := ledgerTestFirst(t, nil)
	m := ledgerTestModel(t, p, i, first, ledgerTestBounds())
	records := []CandidateLedgerReplayEntry{{Ticket: first, ExpectedBudget: ledgerTestCurrent(t, m).BudgetBytes()}}
	c0 := ledgerTestConsumption(t, first, nil, 10)
	change, err := m.AppendCandidateConsumption(c0)
	if err != nil {
		t.Fatal(err)
	}
	records = append(records, CandidateLedgerReplayEntry{Consumption: c0, ExpectedBudget: change.BudgetBytes()})
	s0 := ledgerTestSettlement(t, first, "complete", 20)
	change, err = m.AppendCandidateSettlement(s0)
	if err != nil {
		t.Fatal(err)
	}
	records = append(records, CandidateLedgerReplayEntry{Settlement: s0, ExpectedBudget: change.BudgetBytes()})
	second := ledgerTestNext(t, p, first, 1)
	change, err = m.AppendCandidateTicket(second)
	if err != nil {
		t.Fatal(err)
	}
	records = append(records, CandidateLedgerReplayEntry{Ticket: second, ExpectedBudget: change.BudgetBytes()})
	c1 := ledgerTestConsumption(t, second, c0, 30)
	change, err = m.AppendCandidateConsumption(c1)
	if err != nil {
		t.Fatal(err)
	}
	records = append(records, CandidateLedgerReplayEntry{Consumption: c1, ExpectedBudget: change.BudgetBytes()})
	s1 := ledgerTestSettlement(t, second, "uncertain", 2000)
	change, err = m.AppendCandidateSettlement(s1)
	if err != nil {
		t.Fatal(err)
	}
	records = append(records, CandidateLedgerReplayEntry{Settlement: s1, ExpectedBudget: change.BudgetBytes()})
	return p, i, records, change
}

func TestAdmissionCandidateLedgerOnePassReplay(t *testing.T) {
	p, i, records, original := ledgerTestReplayFixture(t)
	m, err := ReplayCandidateAdmissionLedger(p, i, records, ledgerTestBounds())
	if err != nil {
		t.Fatal(err)
	}
	replayed := ledgerTestCurrent(t, m)
	a, _ := original.Diagnostic()
	b, _ := replayed.Diagnostic()
	if !bytes.Equal(original.BudgetBytes(), replayed.BudgetBytes()) || a != b {
		t.Fatal("replay changed counters, chronology, roots or terminal state", a, b)
	}
	records[0].ExpectedBudget[0] = '!'
	records[0].Ticket = nil
	if !bytes.Equal(original.BudgetBytes(), replayed.BudgetBytes()) {
		t.Fatal("replay retained caller slices")
	}
	last := records[len(records)-1].Settlement
	ledgerTestRefusesWithoutMutation(t, m, func() error { _, err := m.AppendCandidateSettlement(last); return err })
}

func TestAdmissionCandidateLedgerGenerationReplayQuotaInvariant(t *testing.T) {
	p, i, first := ledgerTestFirst(t, nil)
	c := ledgerTestConsumption(t, first, nil, 10)
	wide := ledgerTestModel(t, p, i, first, ledgerTestBounds())
	firstRoot := ledgerTestCurrent(t, wide).BudgetBytes()
	second, err := wide.AppendCandidateConsumption(c)
	if err != nil {
		t.Fatal(err)
	}
	d, _ := second.Diagnostic()
	bounds := ledgerTestBounds()
	bounds.MaximumPayloadBytes = uint64(len(first.raw) + len(c.raw))
	bounds.MaximumMetadataBytes = d.MetadataBytes
	if uint64(len(firstRoot)+len(second.raw)) <= bounds.MaximumPayloadBytes {
		t.Fatal("fixture does not expose the former combined-payload replay cap")
	}
	exact := ledgerTestModel(t, p, i, first, bounds)
	end, err := exact.AppendCandidateConsumption(c)
	if err != nil {
		t.Fatal("creation refused exact quotas", err)
	}
	records := []CandidateLedgerReplayEntry{{Ticket: first, ExpectedBudget: firstRoot}, {Consumption: c, ExpectedBudget: end.BudgetBytes()}}
	replayed, err := ReplayCandidateAdmissionLedger(p, i, records, bounds)
	if err != nil || replayed == nil {
		t.Fatal("successful creation was not replayable with identical quotas", err)
	}
	got, _ := ledgerTestCurrent(t, replayed).Diagnostic()
	if got.PayloadBytes != bounds.MaximumPayloadBytes || got.MetadataBytes != bounds.MaximumMetadataBytes || got != d {
		t.Fatal("expected roots were double charged", got, d)
	}
	for _, mode := range []string{"payload-one-byte", "metadata-one-byte"} {
		short := bounds
		if mode == "payload-one-byte" {
			short.MaximumPayloadBytes--
		} else {
			short.MaximumMetadataBytes--
		}
		model := ledgerTestModel(t, p, i, first, short)
		ledgerTestRefusesWithoutMutation(t, model, func() error { _, err := model.AppendCandidateConsumption(c); return err })
		if got, err := ReplayCandidateAdmissionLedger(p, i, records, short); got != nil || err == nil {
			t.Fatal("one-byte-over replay accepted", mode)
		}
	}
}

func TestAdmissionCandidateLedgerReplayClosedInputs(t *testing.T) {
	for _, mode := range []string{"empty", "records-bound", "invalid-bound", "first-not-ticket", "two-types", "empty-type", "zero-ticket", "initial-root", "later-root", "empty-root", "oversize-root", "payload-preflight", "expected-preflight", "foreign-policy", "foreign-ticket", "terminal-reconsume", "duplicate-settlement", "wrong-consumption-sequence"} {
		t.Run(mode, func(t *testing.T) {
			p, i, records, _ := ledgerTestReplayFixture(t)
			bounds := ledgerTestBounds()
			switch mode {
			case "empty":
				records = nil
			case "records-bound":
				bounds.MaximumRecords = 1
			case "invalid-bound":
				bounds.MaximumMetadataBytes = 0
			case "first-not-ticket":
				records = records[1:]
			case "two-types":
				records[1].Ticket = records[0].Ticket
			case "empty-type":
				records[1].Consumption = nil
			case "zero-ticket":
				records[0].Ticket = &CandidateAdmissionTicket{}
			case "initial-root":
				records[0].ExpectedBudget = []byte(`{}`)
			case "later-root":
				records[1].ExpectedBudget = []byte(`{}`)
			case "empty-root":
				records[0].ExpectedBudget = nil
			case "oversize-root":
				records[0].ExpectedBudget = bytes.Repeat([]byte(" "), candidateLedgerMetadataLimit+1)
			case "payload-preflight":
				bounds.MaximumPayloadBytes = 1
			case "expected-preflight":
				bounds.MaximumMetadataBytes = 1
			case "foreign-policy":
				p = nil
			case "foreign-ticket":
				object := providerPolicyTestObject()
				object["clock"].(map[string]any)["id"] = "other-clock"
				_, _, other := ledgerTestFirst(t, object)
				records[3].Ticket = other
			case "terminal-reconsume":
				records = append(records[:3], CandidateLedgerReplayEntry{Consumption: ledgerTestConsumption(t, records[0].Ticket, records[1].Consumption, 30), ExpectedBudget: records[2].ExpectedBudget})
			case "duplicate-settlement":
				records = append(records[:3], records[2])
			case "wrong-consumption-sequence":
				records[1].Consumption = ledgerTestConsumption(t, records[0].Ticket, records[1].Consumption, 20)
			}
			if got, err := ReplayCandidateAdmissionLedger(p, i, records, bounds); got != nil || err == nil {
				t.Fatal("invalid replay returned a model", mode)
			}
		})
	}
}
