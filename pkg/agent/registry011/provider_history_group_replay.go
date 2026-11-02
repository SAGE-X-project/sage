package registry011

import "context"

// A preparation binds newly reconstructed DATA to the original C instance.
// It is not a transport permit: no caller can dial, send, publish S or select
// a Source with this object. The complete dispatch/settlement owner must still
// bind and consume the actual connection attempt under the original envelope.
type providerHistoryGroupDispatch struct {
	owner                  *providerHistoryGroupDispatch
	group                  *providerHistoryGroupOwner
	acknowledgment         *providerHistoryGroupAcknowledgment
	order                  *providerHistoryGroupOrder
	meter                  *providerWorkMeter
	coordinator, custodian *providerHistoryGroupLedger
	replayReads            [2]*providerHistoryReplayReads
	memberReplaySeal       *providerHistoryMemberReplaySeal
	begin, end             ClockSample
	ticketClock            CandidateClockDiagnostic
	attempt                *providerHistoryDispatchAttempt
	retained               int
	originals              [providerHistoryGroupMembers]*providerHistoryMemberOriginal
}

func (*providerHistoryGroupDispatch) UnmarshalJSON([]byte) error { return errProviderIntegration }

// Scalar provenance only. This is also called with meter.mu held, so it must
// not perform observed codecs, filesystem reads or native clock sampling.
func (o *providerHistoryGroupOwner) consumptionShapeLocked() bool {
	return o.committedConsumptionShapeLocked() && o.order.state == providerHistoryOrderConfirmed && o.order.next == 0 && o.order.sent == 0
}

// The retained C records remain immutable while the SAME original order
// advances its attempted/completed prefix. This is not a reset to pre-dispatch.
func (o *providerHistoryGroupOwner) committedConsumptionShapeLocked() bool {
	if o == nil || o.owner != o || o.parent == nil || o.start == nil || o.admission == nil || o.last == nil ||
		o.admission == o.last || o.admission.owner != o || o.last.owner != o || o.last.pin != o.pin ||
		o.coordinator == nil || o.custodian == nil || o.coordinator == o.custodian ||
		o.coordinator.owner != o.coordinator || o.custodian.owner != o.custodian ||
		o.coordinator.phase != "consumed" || o.custodian.phase != "consumed" {
		return false
	}
	// The original store lock keeps this pure selection stable for this
	// scalar call. Revalidate it on every call; retain no cross-call cache.
	base := o.stageBaseLocked()
	if base == nil {
		return false
	}
	stage := base.frame.stage + 1
	if stage > o.parent.bounds.stages {
		stage = 0 // Preserve nextStageLocked's original invalid-stage value.
	}
	if o.coordinator.stage+1 != stage || o.custodian.stage+1 != stage ||
		o.coordinator.sequence != 3*o.coordinator.stage+2 || o.custodian.sequence != 3*o.custodian.stage+2 ||
		o.coordinator.head != o.custodian.head || o.coordinator.ticket == nil || o.custodian.ticket == nil ||
		o.coordinator.ticket == o.custodian.ticket || o.coordinator.consumption == nil || o.custodian.consumption == nil ||
		o.coordinator.consumption == o.custodian.consumption {
		return false
	}
	t, c := o.admission.claim, o.last.claim
	if t == nil || t.owner != t || t.group != o || t.meter != o.meter || !t.used || !t.renamed ||
		t.frame == nil || t.frame.before != o.stageBefore.pin || t.frame.after != o.admission.pin || t.frame.phase != "admit" || t.frame.stage != stage ||
		t.inventory == nil || t.count != 2 || t.inventory.count != 2 || t.request != t.inventory.entries[0] || t.ticket != t.inventory.entries[1] ||
		t.request.kind != "requests" || t.ticket.kind != "ticket" || t.coordinator == nil || t.coordinator.owner != t.coordinator ||
		t.coordinator.phase != "admitted" || t.coordinator.ticket == nil || t.members < 1 || t.members > providerHistoryGroupMembers ||
		c == nil || c.owner != c || c.group != o || c.meter != o.meter || !c.used || !c.renamed || c.admission != o.admission ||
		c.frame == nil || c.frame.before != o.admission.pin || c.frame.after != o.pin || c.frame.phase != "consume" || c.frame.stage != stage ||
		c.inventory == nil || c.count != 1 || c.inventory.count != 1 || c.inventory.previous != t.inventory.digest ||
		c.inventory.digest != o.inventory || c.consumption != c.inventory.entries[0] || c.consumption.kind != "consumption" ||
		c.coordinator != o.coordinator || c.order == nil || c.order != o.order || c.order.owner != c.order || !c.order.valid() ||
		c.order.state != providerHistoryOrderConfirmed && c.order.state != providerHistoryOrderInFlight {
		return false
	}
	work, ok := t.upper.work()
	all, maskOK := providerHistoryMemberPrefix(t.members)
	return ok && maskOK && o.window && o.windowUpper == work && o.clockUpper == t.upper.clocks &&
		c.order.program == o.parent.bounds.program && c.order.table == t.coordinator.ticket.table.digest && c.order.count == t.members &&
		c.order.consumed == all && o.coordinator.consumption.consumed == all && o.custodian.consumption.consumed == all &&
		o.coordinator.ticket.table.count == t.members && o.custodian.ticket.table.count == t.members &&
		o.coordinator.ticket.digest == t.ticket.digest && o.custodian.ticket.digest == t.ticket.digest &&
		o.coordinator.consumption.digest == c.consumption.digest && o.custodian.consumption.digest == c.consumption.digest
}

// Every invocation reconstructs the immutable initial and all earlier T/C/S,
// then the current T/C under today's selected HEAD. Historical frame selection is explicit DATA; live pin/phase/models,
// the last ACK and every consumed/sent bit are never rewound for the replay.
func (o *providerHistoryGroupOwner) replayCommittedConsumptionLocked(ctx context.Context) (*providerHistoryGroupLedger, error) {
	if ctx == nil || ctx.Err() != nil {
		return nil, errProviderIntegration
	}
	if o == nil || o.store == nil || o.store.frontierFiles == nil {
		return nil, errProviderIntegration
	}
	return o.replayCommittedConsumptionAtLocked(ctx, o.store.frontierFiles.pinRaw)
}

// Physical S rename changes HEAD before logical publication. Select that
// exact current HEAD as DATA; never rewind the live C pin, phase or models.
func (o *providerHistoryGroupOwner) replayCommittedConsumptionAtLocked(ctx context.Context, head string) (*providerHistoryGroupLedger, error) {
	if ctx == nil || ctx.Err() != nil {
		return nil, errProviderIntegration
	}
	if o == nil || !o.committedConsumptionShapeLocked() || !o.activeLocked(o.meter) {
		return nil, errProviderIntegration
	}
	m, t, c := o.meter, o.admission.claim, o.last.claim
	before, err := o.replayHealthyPrefixLocked(ctx, o.stageBefore, head)
	if err != nil {
		return nil, errProviderIntegration
	}
	admitted, err := o.readAdmissionDataLocked(t, before.pin, o.stageBefore.head, before.inventory, before.ledger, head)
	if err != nil {
		return nil, errProviderIntegration
	}
	model, err := o.readConsumptionDataLocked(c, o.admission.pin, t.inventory.digest, admitted, head)
	if err != nil || ctx.Err() != nil || !model.validObserved(m) || !o.coordinator.validObserved(m) || !o.custodian.validObserved(m) ||
		!providerReplayTextEqual(model.head, o.coordinator.head, m) || !providerReplayTextEqual(model.head, o.custodian.head, m) ||
		!o.committedConsumptionShapeLocked() {
		return nil, errProviderIntegration
	}
	return model, nil
}

func (o *providerHistoryGroupOwner) committedConsumptionCatalogLocked() bool {
	if o == nil || !o.committedConsumptionShapeLocked() || !o.activeLocked(o.meter) {
		return false
	}
	names, ok := o.prefixHistoryCatalogLocked(o.stageBefore)
	return ok && o.extendHistoryCatalogLocked(names, o.admission.claim) && o.extendHistoryCatalogLocked(names, o.last.claim) &&
		o.exactHistoryCatalogLocked(names, "")
}

// This is preparation within the still-unfinished dispatch/S implementation
// bundle. It grants no network effect or completed group. The same original
// hold, meter, group window and T clock origin fund both independent passes.
func (o *providerHistoryGroupOwner) prepareDispatchCandidate(ctx context.Context, ack *providerHistoryGroupAcknowledgment) (*providerHistoryGroupDispatch, error) {
	if ctx == nil || ctx.Err() != nil {
		return nil, errProviderIntegration
	}
	if o == nil || o.store == nil {
		return nil, errProviderIntegration
	}
	s := o.store
	s.mu.Lock()
	defer s.mu.Unlock()
	if ctx.Err() != nil || o.phase != "consumed" || o.failed || o.parent == nil || o.parent.group != o || o.dispatch != nil ||
		o.parent.recovery == nil || o.parent.recovery.originalDispatch != nil || o.parent.recovery.order != o.order ||
		o.parent.recovery.consumption == nil || o.parent.recovery.consumption != o.parent.recovery.confirmed || o.parent.recovery.consumption.ack != ack ||
		ack == nil || ack != o.last || !o.consumptionShapeLocked() || s.activeMeter.Load() != nil ||
		s.frontierFiles == nil || s.frontierFiles.pending != nil || s.frontierFiles.historyPending != nil ||
		o.meter == nil || o.meter.owner != o.meter {
		return nil, errProviderIntegration
	}
	m := o.meter
	if m.historyGroup != o || m.historyRecovery != nil || m.integration != nil || m.scan != nil || m.scanBody != nil || m.firstAdmission != nil ||
		m.firstOperation != nil || m.guardian != nil || m.continuing != nil || m.parent != nil || !m.authority ||
		m.upper != o.parent.bounds.future.work || m.maximumClocks != o.parent.bounds.future.clocks {
		return nil, errProviderIntegration
	}
	m.mu.Lock()
	usable := !m.closed && !m.failed
	m.mu.Unlock()
	if !usable {
		return nil, errProviderIntegration
	}
	o.phase = "preparing-dispatch"
	s.activeMeter.Store(m)
	succeeded := false
	defer func() {
		if !succeeded {
			o.failLocked()
		}
	}()
	if !o.fits(m) || !o.identityLocked() || !o.committedConsumptionCatalogLocked() {
		return nil, errProviderIntegration
	}
	begin, err := s.samplePrimitive()
	if err != nil || !o.clockFits(begin) || !providerHistoryGroupClockFits(o.admission.claim.frame.clock, providerClockDiagnostic(begin)) {
		return nil, errProviderIntegration
	}
	a, leftReads, err := o.replayRecordedConsumptionLocked(ctx, 0)
	if err != nil {
		return nil, errProviderIntegration
	}
	b, rightReads, err := o.replayRecordedConsumptionLocked(ctx, 1)
	if err != nil || a == b || a.ticket == b.ticket || a.consumption == b.consumption ||
		!providerReplayTextEqual(a.head, b.head, m) || !o.committedConsumptionCatalogLocked() || !o.identityLocked() || ctx.Err() != nil {
		return nil, errProviderIntegration
	}
	d := &providerHistoryGroupDispatch{group: o, acknowledgment: ack, order: o.order, meter: m, coordinator: a, custodian: b, begin: begin,
		replayReads: [2]*providerHistoryReplayReads{leftReads, rightReads},
		ticketClock: o.admission.claim.frame.clock}
	d.owner = d
	seal, err := newProviderHistoryMemberReplaySealLocked(d)
	if err != nil {
		return nil, errProviderIntegration
	}
	end, err := s.samplePrimitive()
	if err != nil || !o.clockFits(end) || !candidateLedgerClockAdvance(providerClockDiagnostic(begin), providerClockDiagnostic(end)) ||
		!providerHistoryGroupClockFits(o.admission.claim.frame.clock, providerClockDiagnostic(end)) ||
		!o.consumptionShapeLocked() || m.failed || m.closed || ctx.Err() != nil {
		return nil, errProviderIntegration
	}
	d.end = end
	o.dispatch, o.phase = d, "dispatch-prepared"
	o.parent.recovery.originalDispatch = d
	d.memberReplaySeal, o.parent.recovery.memberReplaySeal = seal, seal
	s.activeMeter.Store(nil)
	succeeded = true
	return d, nil
}
