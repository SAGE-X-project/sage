package registry011

import (
	"bytes"
	"encoding/json"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
)

func ledgerTestBounds() CandidateLedgerBounds {
	return CandidateLedgerBounds{MaximumRecords: 64, MaximumPayloadBytes: 1 << 20, MaximumMetadataBytes: 1 << 20}
}

func ledgerTestFirst(t *testing.T, object map[string]any) (*ProviderPolicyCandidate, *CandidateAdmissionIntent, *CandidateAdmissionTicket) {
	t.Helper()
	p, i, m := admissionTestTicketPolicyFixture(t, object)
	return p, i, admissionTestTicket(t, p, i, 0, m)
}

func ledgerTestModel(t *testing.T, p *ProviderPolicyCandidate, i *CandidateAdmissionIntent, first *CandidateAdmissionTicket, bounds CandidateLedgerBounds) *CandidateAdmissionLedger {
	t.Helper()
	m, err := NewCandidateAdmissionLedger(p, i, first, bounds)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func ledgerTestCurrent(t *testing.T, m *CandidateAdmissionLedger) *CandidateAdmissionLedgerChange {
	t.Helper()
	c, err := m.Current()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func ledgerTestUnchanged(t *testing.T, m *CandidateAdmissionLedger, before *CandidateAdmissionLedgerChange) {
	t.Helper()
	if got := ledgerTestCurrent(t, m); got != before || !bytes.Equal(got.BudgetBytes(), before.BudgetBytes()) {
		t.Fatal("refusal changed the budget prefix")
	}
}

func ledgerTestNext(t *testing.T, p *ProviderPolicyCandidate, before *CandidateAdmissionTicket, segment uint64) *CandidateAdmissionTicket {
	t.Helper()
	m := contentJSON(before.CanonicalBytes()).(map[string]any)
	m["sequence"], m["previous_ticket"] = strconv.FormatUint(before.sequence+1, 10), before.digest
	segmentRaw := providerPolicyTestBytes(t, map[string]any{"admission_digest": before.intent.digest, "index": strconv.FormatUint(segment, 10)})
	m["segment"] = enrollmentTestHash("sage-provider-budget-segment-v1", segmentRaw)
	w := m["reservation"].(map[string]any)
	w["segments"] = "0"
	if segment != before.segment {
		w["segments"] = "1"
	}
	return admissionTestTicket(t, p, before.intent, segment, m)
}

func ledgerTestConsumption(t *testing.T, ticket *CandidateAdmissionTicket, previous *CandidateConsumption, delta int64) *CandidateConsumption {
	t.Helper()
	id := ticket.intent.identity
	var prior any
	sequence := uint64(0)
	if previous != nil {
		prior, sequence = previous.digest, previous.sequence+1
	}
	m := map[string]any{"format": "sage-provider-budget-consumption-v1", "ticket_digest": ticket.digest, "namespace": id.Namespace, "instance_key": id.Instance, "source_id": id.SourceID, "generation": strconv.FormatUint(id.Generation, 10), "fence": strconv.FormatUint(ticket.intent.fence, 10), "sequence": strconv.FormatUint(sequence, 10), "previous_consumption": prior, "kind": ticket.kind, "input_digest": ticket.input, "clock_start": ledgerTestClock(ticket, delta)}
	c, err := ParseCandidateConsumption(ticket, providerPolicyTestBytes(t, m), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestAdmissionCandidateLedgerInitialClosureAndCopies(t *testing.T) {
	p, i, first := ledgerTestFirst(t, nil)
	m := ledgerTestModel(t, p, i, first, ledgerTestBounds())
	change := ledgerTestCurrent(t, m)
	nodes := change.LedgerNodeBytes()
	if len(nodes) != 4 {
		t.Fatal("missing explicit empty anchors or first node")
	}
	var heads [3]string
	for n, kind := range []string{"tickets", "consumptions", "settlements"} {
		raw := providerPolicyTestBytes(t, map[string]any{"format": "sage-provider-ledger-node-v1", "admission_digest": i.digest, "kind": kind, "index": "0", "previous": nil, "entry": nil})
		if !bytes.Equal(nodes[n], raw) {
			t.Fatal("wrong empty ledger anchor")
		}
		heads[n] = enrollmentTestHash("sage-provider-ledger-node-v1", raw)
	}
	ticketNode := providerPolicyTestBytes(t, map[string]any{"format": "sage-provider-ledger-node-v1", "admission_digest": i.digest, "kind": "tickets", "index": "1", "previous": heads[0], "entry": first.digest})
	if !bytes.Equal(nodes[3], ticketNode) {
		t.Fatal("wrong ticket ledger predecessor")
	}
	heads[0] = enrollmentTestHash("sage-provider-ledger-node-v1", ticketNode)
	zero, one := map[string]any{}, map[string]any{}
	for _, key := range []string{"requests", "wire_bytes", "decoded_bytes", "blocks", "transactions", "frames", "logs", "proofs", "segments", "disk_bytes_written", "work_units"} {
		zero[key], one[key] = "0", "1"
	}
	want := providerPolicyTestBytes(t, map[string]any{"format": "sage-provider-budget-root-v2", "admission_digest": i.digest, "bootstrap_start": nil, "namespace": i.identity.Namespace, "instance_key": i.identity.Instance, "generation": "0", "fence": "1", "sequence": "0", "previous_root": nil, "tickets_root": heads[0], "consumptions_root": heads[1], "settlements_root": heads[2], "charged": zero, "outstanding": one, "live_disk_used": "0", "live_disk_reserved": "100", "retained_bytes": "0", "retained_bytes_reserved": "90", "deadline_wall_ms": "1001000"})
	d, err := change.Diagnostic()
	if err != nil || !bytes.Equal(change.BudgetBytes(), want) || d.BudgetRoot != enrollmentTestHash("sage-provider-budget-root-v2", want) || d.LedgerHeads != heads || d.Tickets != 1 || d.Sequence != 0 || d.Outstanding != first.work.diagnostic() || d.Charged != (CandidateWork{}) {
		t.Fatal(d, err)
	}
	budgetCopy := change.BudgetBytes()
	budgetCopy[0], nodes[0][0] = '!', '!'
	if !bytes.Equal(change.BudgetBytes(), want) || change.LedgerNodeBytes()[0][0] != '{' {
		t.Fatal("mutable returned metadata")
	}
	consumption := ledgerTestConsumption(t, first, nil, 10)
	if _, err := m.AppendCandidateConsumption(consumption); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(change.BudgetBytes(), want) {
		t.Fatal("earlier snapshot changed after append")
	}
}

func TestAdmissionCandidateLedgerConservativeCompletion(t *testing.T) {
	p, i, object := admissionTestTicketFixture(t)
	object["reservation"].(map[string]any)["wire_bytes"] = "10"
	object["reservation"].(map[string]any)["decoded_bytes"] = "20"
	first := admissionTestTicket(t, p, i, 0, object)
	m := ledgerTestModel(t, p, i, first, ledgerTestBounds())
	c := ledgerTestConsumption(t, first, nil, 10)
	if _, err := m.AppendCandidateConsumption(c); err != nil {
		t.Fatal(err)
	}
	s := ledgerTestSettlementObject(first, "complete", 20)
	s["actual_work"].(map[string]any)["wire_bytes"] = "1"
	s["actual_work"].(map[string]any)["decoded_bytes"] = "1"
	settlement, err := ParseCandidateAdmissionSettlement(first, providerPolicyTestBytes(t, s), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	change, err := m.AppendCandidateSettlement(settlement)
	if err != nil {
		t.Fatal(err)
	}
	d, _ := change.Diagnostic()
	if d.Sequence != 2 || d.Consumptions != 1 || d.Settlements != 1 || d.Charged != first.work.diagnostic() || d.Outstanding != (CandidateWork{}) || d.LiveUsedUpper != 100 || d.LiveReservedUpper != 0 || d.RetainedUsedUpper != 90 || d.RetainedReservedUpper != 0 {
		t.Fatal("complete refunded candidate-supplied work", d)
	}
	if _, err := m.AppendCandidateSettlement(settlement); err == nil {
		t.Fatal("terminal settlement repeated")
	}
	ledgerTestUnchanged(t, m, change)
	if _, err := m.AppendCandidateConsumption(ledgerTestConsumption(t, first, c, 30)); err == nil {
		t.Fatal("charged ticket consumed")
	}
	ledgerTestUnchanged(t, m, change)
}

func TestAdmissionCandidateLedgerInitialTicketIsDataOnly(t *testing.T) {
	for _, mode := range []string{"local-work", "other-read-method", "unqualified-parameters"} {
		t.Run(mode, func(t *testing.T) {
			p, i, raw := admissionTestTicketFixture(t)
			switch mode {
			case "local-work":
				raw["kind"], raw["method"], raw["params_digest"] = "local-work", nil, nil
				raw["reservation"].(map[string]any)["requests"] = "0"
			case "other-read-method":
				raw["method"] = "eth_chainId"
			case "unqualified-parameters":
				// This digest has no retained request bytes or finalized selector.
				raw["params_digest"] = admissionTestDigest("c")
			}
			first := admissionTestTicket(t, p, i, 0, raw)
			m := ledgerTestModel(t, p, i, first, ledgerTestBounds())
			d, err := ledgerTestCurrent(t, m).Diagnostic()
			if err != nil || d.Admission != i.digest || d.Sequence != 0 || d.Tickets != 1 || d.Outstanding != first.work.diagnostic() {
				t.Fatal("generic candidate ticket accounting changed", d, err)
			}
			// Candidate shape/accounting is valid; no actual bootstrap request
			// or dispatch permission is constructed by this data-only model.
		})
	}
}

func TestAdmissionCandidateLedgerTerminalBeforeConsumption(t *testing.T) {
	for _, outcome := range []string{"failed", "cancelled", "uncertain"} {
		t.Run(outcome, func(t *testing.T) {
			p, i, first := ledgerTestFirst(t, nil)
			m := ledgerTestModel(t, p, i, first, ledgerTestBounds())
			terminal := ledgerTestSettlement(t, first, outcome, 10)
			change, err := m.AppendCandidateSettlement(terminal)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := m.AppendCandidateConsumption(ledgerTestConsumption(t, first, nil, 20)); err == nil {
				t.Fatal("settled unconsumed ticket became executable")
			}
			ledgerTestUnchanged(t, m, change)
			if _, err := m.AppendCandidateSettlement(terminal); err == nil {
				t.Fatal("terminal replay appended another record")
			}
			ledgerTestUnchanged(t, m, change)
			next := ledgerTestNext(t, p, first, 0)
			if _, err := m.AppendCandidateTicket(next); err != nil {
				t.Fatal(err)
			}
			d, _ := ledgerTestCurrent(t, m).Diagnostic()
			if d.Charged != first.work.diagnostic() || d.Outstanding != next.work.diagnostic() || d.LiveUsedUpper != 100 || d.LiveReservedUpper != 100 {
				t.Fatal("terminal state reset during later admission")
			}
		})
	}
}

func TestAdmissionCandidateLedgerNonAdjacentConsumption(t *testing.T) {
	p, i, first := ledgerTestFirst(t, nil)
	m := ledgerTestModel(t, p, i, first, ledgerTestBounds())
	c0 := ledgerTestConsumption(t, first, nil, 10)
	if _, err := m.AppendCandidateConsumption(c0); err != nil {
		t.Fatal(err)
	}
	second := ledgerTestNext(t, p, first, 0)
	if _, err := m.AppendCandidateTicket(second); err != nil {
		t.Fatal(err)
	}
	c1 := ledgerTestConsumption(t, second, c0, 20)
	if _, err := m.AppendCandidateConsumption(c1); err != nil {
		t.Fatal(err)
	}
	before := ledgerTestCurrent(t, m)
	reused := ledgerTestConsumption(t, first, c1, 30)
	if CheckCandidateConsumptionSuccessor(c1, reused) != nil {
		t.Fatal("fixture must pass adjacent-only comparison")
	}
	if _, err := m.AppendCandidateConsumption(reused); err == nil {
		t.Fatal("non-adjacent ticket reuse")
	}
	ledgerTestUnchanged(t, m, before)
	if _, err := m.AppendCandidateSettlement(ledgerTestSettlement(t, second, "complete", 30)); err != nil {
		t.Fatal(err)
	}
}

func TestAdmissionCandidateLedgerCrossStreamClockAndExpiry(t *testing.T) {
	p, i, first := ledgerTestFirst(t, nil)
	m := ledgerTestModel(t, p, i, first, ledgerTestBounds())
	if _, err := m.AppendCandidateSettlement(ledgerTestSettlement(t, first, "complete", 10)); err == nil {
		t.Fatal("complete before consumption")
	}
	second := ledgerTestNext(t, p, first, 0)
	if _, err := m.AppendCandidateTicket(second); err != nil {
		t.Fatal(err)
	}
	c := ledgerTestConsumption(t, first, nil, 100)
	if _, err := m.AppendCandidateConsumption(c); err != nil {
		t.Fatal(err)
	}
	before := ledgerTestCurrent(t, m)
	if _, err := m.AppendCandidateSettlement(ledgerTestSettlement(t, first, "complete", 90)); err == nil {
		t.Fatal("settlement predates retained consumption")
	}
	ledgerTestUnchanged(t, m, before)
	if _, err := m.AppendCandidateSettlement(ledgerTestSettlement(t, first, "uncertain", 2000)); err != nil {
		t.Fatal("expired recovery could not retain full charge", err)
	}
	before = ledgerTestCurrent(t, m)
	if _, err := m.AppendCandidateTicket(ledgerTestNext(t, p, second, 0)); err == nil {
		t.Fatal("new ticket after retained expiry")
	}
	if _, err := m.AppendCandidateConsumption(ledgerTestConsumption(t, second, c, 200)); err == nil {
		t.Fatal("old on-time sample bypassed retained expiry")
	}
	ledgerTestUnchanged(t, m, before)
	if _, err := m.AppendCandidateSettlement(ledgerTestSettlement(t, second, "cancelled", 2001)); err != nil {
		t.Fatal("late terminal charge refused", err)
	}
	d, _ := ledgerTestCurrent(t, m).Diagnostic()
	if d.Outstanding != (CandidateWork{}) || d.Charged.Requests != 2 || d.LiveUsedUpper != 200 || d.RetainedUsedUpper != 180 {
		t.Fatal(d)
	}
}

func TestAdmissionCandidateLedgerConcurrentOneUse(t *testing.T) {
	p, i, first := ledgerTestFirst(t, nil)
	m := ledgerTestModel(t, p, i, first, ledgerTestBounds())
	c := ledgerTestConsumption(t, first, nil, 10)
	var group sync.WaitGroup
	var accepted atomic.Int32
	for range 16 {
		group.Add(1)
		go func() {
			defer group.Done()
			if _, err := m.AppendCandidateConsumption(c); err == nil {
				accepted.Add(1)
			}
		}()
	}
	group.Wait()
	if accepted.Load() != 1 {
		t.Fatal("consumption was not one-use within this model")
	}
	d, _ := ledgerTestCurrent(t, m).Diagnostic()
	if d.Sequence != 1 || d.Consumptions != 1 || d.Outstanding != first.work.diagnostic() {
		t.Fatal(d)
	}
}

func TestAdmissionCandidateLedgerOpaqueArguments(t *testing.T) {
	p, i, first := ledgerTestFirst(t, nil)
	var empty *CandidateAdmissionLedger
	if _, err := empty.Current(); err == nil {
		t.Fatal("nil current")
	}
	if _, err := empty.AppendCandidateTicket(first); err == nil {
		t.Fatal("nil ledger ticket")
	}
	if _, err := empty.AppendCandidateConsumption(nil); err == nil {
		t.Fatal("nil ledger consumption")
	}
	if _, err := empty.AppendCandidateSettlement(nil); err == nil {
		t.Fatal("nil ledger settlement")
	}
	if _, err := (&CandidateAdmissionLedger{}).Current(); err == nil {
		t.Fatal("zero ledger")
	}
	m := ledgerTestModel(t, p, i, first, ledgerTestBounds())
	for _, bad := range []*CandidateAdmissionTicket{nil, {}} {
		if _, err := m.AppendCandidateTicket(bad); err == nil {
			t.Fatal("zero ticket")
		}
	}
	for _, bad := range []*CandidateConsumption{nil, {}} {
		if _, err := m.AppendCandidateConsumption(bad); err == nil {
			t.Fatal("zero consumption")
		}
	}
	for _, bad := range []*CandidateAdmissionSettlement{nil, {}} {
		if _, err := m.AppendCandidateSettlement(bad); err == nil {
			t.Fatal("zero settlement")
		}
	}
	// Explicit inert copied-owner state; no mutex is copied and no host access.
	other := &CandidateAdmissionLedger{owner: m, policy: m.policy, intent: m.intent, bounds: m.bounds, tickets: m.tickets, segments: m.segments, current: m.current}
	if _, err := other.Current(); err == nil {
		t.Fatal("copied owner state accepted")
	}
	var change *CandidateAdmissionLedgerChange
	if change.BudgetBytes() != nil || change.LedgerNodeBytes() != nil {
		t.Fatal("nil metadata copies")
	}
	if _, err := change.Diagnostic(); err == nil {
		t.Fatal("nil diagnostic")
	}
	if _, err := (&CandidateAdmissionLedgerChange{}).Diagnostic(); err == nil {
		t.Fatal("zero diagnostic")
	}
	if json.Unmarshal([]byte(`{}`), &CandidateAdmissionLedger{}) != ErrAdmissionLedger || json.Unmarshal([]byte(`{}`), &CandidateAdmissionLedgerChange{}) != ErrAdmissionLedger {
		t.Fatal("JSON reconstructed private model state")
	}
}
