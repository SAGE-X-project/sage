package registry011

import (
	"bytes"
	"testing"
)

func TestProviderReplayObservedLedgerPreservesBudgetAndInitialNodes(t *testing.T) {
	plain, policy := providerPrivateLedgerFixture(t, 8, 32)
	meter := providerReplayUnitMeter()
	observed, err := newProviderContinuingLedgerObserved(policy, plain.intent, plain.lastTicket, plain.continuation.bounds, meter)
	if err != nil || observed.meter != meter || !bytes.Equal(observed.current.BudgetBytes(), plain.current.BudgetBytes()) || observed.initialHeads != plain.initialHeads || observed.continuation.records[0] != plain.continuation.records[0] || meter.actual[2] != uint64(len(policy.raw)) || meter.actual[10] == 0 {
		t.Fatal("independent observed constructor changed its DATA", err)
	}
	before := observed.current
	consumption := ledgerTestConsumption(t, plain.lastTicket, nil, 10)
	want, err := plain.AppendCandidateConsumption(consumption)
	if err != nil {
		t.Fatal(err)
	}
	got, err := observed.AppendCandidateConsumption(consumption)
	if err != nil || !bytes.Equal(got.BudgetBytes(), want.BudgetBytes()) || len(observed.continuation.records) != 2 || got == before {
		t.Fatal("observed consumption changed exact budget or record chain", err)
	}
	settlement := ledgerTestSettlement(t, plain.lastTicket, "failed", 20)
	want, err = plain.AppendCandidateSettlement(settlement)
	if err != nil {
		t.Fatal(err)
	}
	got, err = observed.AppendCandidateSettlement(settlement)
	if err != nil || !bytes.Equal(got.BudgetBytes(), want.BudgetBytes()) || len(observed.continuation.records) != 3 {
		t.Fatal("observed settlement changed exact budget or record chain", err)
	}
	zero := providerReplayUnitMeter()
	zero.upper = candidateWorkVector{}
	if _, err := newProviderContinuingLedgerObserved(policy, plain.intent, plain.lastTicket, plain.continuation.bounds, zero); err == nil || !zero.failed || zero.actual != (candidateWorkVector{}) {
		t.Fatal("unfunded constructor decoded policy")
	}
}

func TestProviderReplayExhaustedLedgerDoesNotInstallPartialChange(t *testing.T) {
	plain, policy := providerPrivateLedgerFixture(t, 8, 32)
	meter := providerReplayUnitMeter()
	m, err := newProviderContinuingLedgerObserved(policy, plain.intent, plain.lastTicket, plain.continuation.bounds, meter)
	if err != nil {
		t.Fatal(err)
	}
	before, records := m.current, len(m.continuation.records)
	meter.upper[10] = meter.actual[10]
	consumption := ledgerTestConsumption(t, plain.lastTicket, nil, 10)
	if _, err := m.AppendCandidateConsumption(consumption); err == nil || !meter.failed || m.current != before || len(m.continuation.records) != records || m.tickets[plain.lastTicket.digest].consumption != nil {
		t.Fatal("exhausted observer installed a partial ledger change")
	}
}

func TestProviderReplayObservedIndexesAndSealRebuildOrdinaryData(t *testing.T) {
	s, m, root := providerIndexFixture(t)
	meter := providerReplayUnitMeter()
	// Only these subsequent finite DATA operations are the measured subject.
	// This arithmetic fixture cannot select a store, ACK or live scan owner.
	s.meter, m.meter = meter, meter
	root = providerIndexApply(t, s, m, root, "admit", "ticket", m.lastTicket.CanonicalBytes())
	consumption := ledgerTestConsumption(t, m.lastTicket, nil, 10)
	if _, err := m.AppendCandidateConsumption(consumption); err != nil {
		t.Fatal(err)
	}
	root = providerIndexApply(t, s, m, root, "consume", "consumption", []byte(consumption.raw))
	settlement := ledgerTestSettlement(t, m.lastTicket, "failed", 20)
	if _, err := m.AppendCandidateSettlement(settlement); err != nil {
		t.Fatal(err)
	}
	root = providerIndexApply(t, s, m, root, "settle", "settlement", settlement.CanonicalBytes())
	before := meter.actual
	boundary, err := m.prepareProviderBoundary(true)
	if err != nil {
		t.Fatal(err)
	}
	heads := s.heads
	seal, err := boundary.sealObject(heads)
	if err != nil || meter.actual[2] <= before[2] || meter.actual[10] <= before[10] {
		t.Fatal("page or seal construction omitted actual calls", err)
	}
	next := providerIndexRoot(t, root, m)
	delta, err := s.prepare("seal", admissionTestDigest("b"), uint64(len(s.operations))+1, seal, next, m)
	if err != nil || s.apply(delta) != nil || m.applyProviderBoundary(boundary, []byte(boundary.pageObject.raw), []byte(seal.raw), heads) != nil || !m.continuation.final || meter.failed {
		t.Fatal("observed final page/index transition refused ordinary DATA", err)
	}
}

func TestProviderReplayObservedIndexRefusalKeepsHeads(t *testing.T) {
	s, m, root := providerIndexFixture(t)
	meter := providerReplayUnitMeter()
	s.meter, m.meter = meter, meter
	entry, err := newProviderIntegrationObject("ticket", m.lastTicket.CanonicalBytes(), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	next := providerIndexRoot(t, root, m)
	before := s.heads
	meter.upper = candidateWorkVector{}
	if delta, err := s.prepare("admit", entry.digest, 2, entry, next, m); err == nil || delta != nil || !meter.failed || s.heads != before || len(s.tickets) != 0 {
		t.Fatal("unfunded index prepare installed a partial mutation")
	}
}

func TestProviderReplayObservedValueComparisonCountsBothEncoders(t *testing.T) {
	value := map[string]string{"value": "ordinary"}
	raw := providerIntegrationJSON(value)
	meter := providerReplayUnitMeter()
	if !bytesEqualProviderValueObserved(value, value, meter) || meter.actual[2] != 0 || meter.actual[10] != uint64(2*(len(raw)*len(raw)+1)+2*len(raw)+1) {
		t.Fatal("comparison did not observe both encoders and exact comparison")
	}
	zero := providerReplayUnitMeter()
	zero.upper = candidateWorkVector{}
	if bytesEqualProviderValueObserved(value, value, zero) || !zero.failed {
		t.Fatal("unfunded value encoding compared equal")
	}
}
