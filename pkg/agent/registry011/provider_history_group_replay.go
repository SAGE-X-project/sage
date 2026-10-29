package registry011

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
	begin, end             ClockSample
}

func (*providerHistoryGroupDispatch) UnmarshalJSON([]byte) error { return errProviderIntegration }

// Scalar provenance only. This is also called with meter.mu held, so it must
// not perform observed codecs, filesystem reads or native clock sampling.
func (o *providerHistoryGroupOwner) consumptionShapeLocked() bool {
	if o == nil || o.owner != o || o.parent == nil || o.start == nil || o.admission == nil || o.last == nil ||
		o.admission == o.last || o.admission.owner != o || o.last.owner != o || o.last.pin != o.pin ||
		o.coordinator == nil || o.custodian == nil || o.coordinator == o.custodian ||
		o.coordinator.owner != o.coordinator || o.custodian.owner != o.custodian ||
		o.coordinator.phase != "consumed" || o.custodian.phase != "consumed" ||
		o.coordinator.stage != 0 || o.custodian.stage != 0 || o.coordinator.sequence != 2 || o.custodian.sequence != 2 ||
		o.coordinator.head != o.custodian.head || o.coordinator.ticket == nil || o.custodian.ticket == nil ||
		o.coordinator.ticket == o.custodian.ticket || o.coordinator.consumption == nil || o.custodian.consumption == nil ||
		o.coordinator.consumption == o.custodian.consumption {
		return false
	}
	t, c := o.admission.claim, o.last.claim
	if t == nil || t.owner != t || t.group != o || t.meter != o.meter || !t.used || !t.renamed ||
		t.frame == nil || t.frame.before != o.start.main || t.frame.after != o.admission.pin || t.frame.phase != "admit" || t.frame.stage != 1 ||
		t.inventory == nil || t.count != 2 || t.inventory.count != 2 || t.request != t.inventory.entries[0] || t.ticket != t.inventory.entries[1] ||
		t.request.kind != "requests" || t.ticket.kind != "ticket" || t.coordinator == nil || t.coordinator.owner != t.coordinator ||
		t.coordinator.phase != "admitted" || t.coordinator.ticket == nil || t.members < 1 || t.members > providerHistoryGroupMembers ||
		c == nil || c.owner != c || c.group != o || c.meter != o.meter || !c.used || !c.renamed || c.admission != o.admission ||
		c.frame == nil || c.frame.before != o.admission.pin || c.frame.after != o.pin || c.frame.phase != "consume" || c.frame.stage != 1 ||
		c.inventory == nil || c.count != 1 || c.inventory.count != 1 || c.inventory.previous != t.inventory.digest ||
		c.inventory.digest != o.inventory || c.consumption != c.inventory.entries[0] || c.consumption.kind != "consumption" ||
		c.coordinator != o.coordinator || c.order == nil || c.order != o.order || c.order.owner != c.order || !c.order.valid() ||
		c.order.state != providerHistoryOrderConfirmed || c.order.next != 0 || c.order.sent != 0 {
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

// Every invocation reconstructs immutable initial -> T -> C under today's C
// HEAD. Historical frame selection is explicit DATA; live pin/phase/models,
// the last ACK and every consumed/sent bit are never rewound for the replay.
func (o *providerHistoryGroupOwner) replayCommittedConsumptionLocked() (*providerHistoryGroupLedger, error) {
	if o == nil || !o.consumptionShapeLocked() || !o.activeLocked(o.meter) {
		return nil, errProviderIntegration
	}
	s, m, t, c := o.store, o.meter, o.admission.claim, o.last.claim
	raw, err := o.start.main.bytesObserved(s.config, m)
	if err != nil {
		return nil, errProviderIntegration
	}
	head := s.frontierFiles.pinRaw
	initial, scope, err := o.initialModelAtLocked(o.start.main, providerReplayString(raw, m), head)
	if err != nil || scope != o.scope {
		return nil, errProviderIntegration
	}
	admitted, err := o.readAdmissionDataLocked(t, o.start.main, providerReplayString(raw, m), initial, head)
	if err != nil {
		return nil, errProviderIntegration
	}
	model, err := o.readConsumptionDataLocked(c, o.admission.pin, t.inventory.digest, admitted, head)
	if err != nil || !model.validObserved(m) || !o.coordinator.validObserved(m) || !o.custodian.validObserved(m) ||
		!providerReplayTextEqual(model.head, o.coordinator.head, m) || !providerReplayTextEqual(model.head, o.custodian.head, m) ||
		!o.consumptionShapeLocked() {
		return nil, errProviderIntegration
	}
	return model, nil
}

func (o *providerHistoryGroupOwner) committedConsumptionCatalogLocked() bool {
	if o == nil || !o.consumptionShapeLocked() || !o.activeLocked(o.meter) {
		return false
	}
	names, ok := o.initialHistoryCatalogLocked()
	return ok && o.extendHistoryCatalogLocked(names, o.admission.claim) && o.extendHistoryCatalogLocked(names, o.last.claim) &&
		o.exactHistoryCatalogLocked(names, "")
}

// This is preparation within the still-unfinished dispatch/S implementation
// bundle. It grants no network effect or completed group. The same original
// hold, meter, group window and T clock origin fund both independent passes.
func (o *providerHistoryGroupOwner) prepareDispatchCandidate(ack *providerHistoryGroupAcknowledgment) (*providerHistoryGroupDispatch, error) {
	if o == nil || o.store == nil {
		return nil, errProviderIntegration
	}
	s := o.store
	s.mu.Lock()
	defer s.mu.Unlock()
	if o.phase != "consumed" || o.failed || o.parent == nil || o.parent.group != o || o.dispatch != nil ||
		ack == nil || ack != o.last || !o.consumptionShapeLocked() || s.activeMeter.Load() != nil ||
		s.frontierFiles == nil || s.frontierFiles.pending != nil || s.frontierFiles.historyPending != nil ||
		o.meter == nil || o.meter.owner != o.meter {
		return nil, errProviderIntegration
	}
	m := o.meter
	if m.historyGroup != o || m.integration != nil || m.scan != nil || m.scanBody != nil || m.firstAdmission != nil ||
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
	a, err := o.replayCommittedConsumptionLocked()
	if err != nil {
		return nil, errProviderIntegration
	}
	b, err := o.replayCommittedConsumptionLocked()
	if err != nil || a == b || a.ticket == b.ticket || a.consumption == b.consumption ||
		!providerReplayTextEqual(a.head, b.head, m) || !o.committedConsumptionCatalogLocked() || !o.identityLocked() {
		return nil, errProviderIntegration
	}
	end, err := s.samplePrimitive()
	if err != nil || !o.clockFits(end) || !candidateLedgerClockAdvance(providerClockDiagnostic(begin), providerClockDiagnostic(end)) ||
		!providerHistoryGroupClockFits(o.admission.claim.frame.clock, providerClockDiagnostic(end)) ||
		!o.consumptionShapeLocked() || m.failed || m.closed {
		return nil, errProviderIntegration
	}
	d := &providerHistoryGroupDispatch{group: o, acknowledgment: ack, order: o.order, meter: m, coordinator: a, custodian: b, begin: begin, end: end}
	d.owner = d
	o.dispatch, o.phase = d, "dispatch-prepared"
	s.activeMeter.Store(nil)
	succeeded = true
	return d, nil
}
