package registry011

import "testing"

func TestProviderContinuingConsumptionPhysicalClaimIsExactAndOneUse(t *testing.T) {
	t.Parallel()
	s, o := providerContinuingPreparedFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	defer o.failLocked()
	clock, err := o.integration.currentClock()
	if err != nil {
		t.Fatal(err)
	}
	consumption, err := providerContinuingConsumptionData(o.ticket, o.integration.ledger.lastConsumption, providerClockDiagnostic(clock), o.execution)
	if err != nil {
		t.Fatal(err)
	}
	change, err := o.integration.ledger.AppendCandidateConsumption(consumption)
	if err != nil {
		t.Fatal(err)
	}
	o.state = providerContinuingConsumption
	p, err := providerContinuingChangeData(s, o.integration.root, o.integration.ledger, o.indexes, s.frontierReplay.current, "consume", consumption.digest, change, []providerIntegrationObjectInput{{"consumption", providerReplayPhaseBytes(consumption.raw, o.execution)}}, providerClockDiagnostic(clock), o.execution)
	if err != nil {
		t.Fatal(err)
	}
	o.change = p
	cost, err := providerFrontierRetentionCost(s.config.identity, p.inventory, p.fresh, p.intent, p.commit, p.frontier, s.config.bounds)
	head, headErr := p.after.bytes(s.config)
	headCost, ok := providerHighwaterFileCost(uint64(len(head)), s.config.bounds)
	physical, sumOK := providerUpperSum(cost.stored, headCost.stored)
	if err != nil || headErr != nil || !ok || !sumOK || physical == 0 || !o.writingLocked(s) {
		t.Fatal("exact finite C proposal did not close", err, headErr)
	}
	stored, sequence, raw := s.stored, s.sequence, s.retainedRaw
	upper := o.plan.consumption
	o.plan.consumption = physical - 1
	if o.consumeStorageLocked(physical) || o.consumption || o.physical != 0 || s.stored != stored || s.sequence != sequence || s.retainedRaw != raw {
		t.Fatal("one-short physical role wrote, refunded or consumed C")
	}
	o.plan.consumption = upper
	if o.consumeStorageLocked(0) || !o.consumeStorageLocked(physical) || !o.consumption || o.physical != physical || o.consumeStorageLocked(physical) ||
		s.stored != stored || s.sequence != sequence || s.retainedRaw != raw || o.permit != nil || o.compositeLocked() {
		t.Fatal("physical unit DATA became a retained C or a dispatch receipt")
	}
}

func TestProviderContinuingSegmentSelectionKeepsTheOriginalLedger(t *testing.T) {
	t.Parallel()
	s, o := providerContinuingPreparedFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	ledger := o.integration.ledger
	before, original, ticket := ledger.current, ledger.intent, ledger.lastTicket
	quote := o.plan.reservation
	if segment, ok := providerContinuingSegment(ledger, quote); !ok || segment < ticket.segment || segment > ticket.segment+1 {
		t.Fatal("finite next segment could not be selected")
	}
	if _, ok := providerContinuingSegment(nil, quote); ok {
		t.Fatal("missing ledger selected a new segment")
	}
	if ledger.current != before || ledger.intent != original || ledger.lastTicket != ticket {
		t.Fatal("segment quote reset a ledger or original admission")
	}
}
