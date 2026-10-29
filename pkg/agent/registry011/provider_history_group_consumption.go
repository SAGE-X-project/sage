package registry011

// This is arithmetic over diagnostic DATA, not a native clock qualification.
// The concrete caller must independently replay origin from the retained T.
func providerHistoryGroupClockFits(origin, current CandidateClockDiagnostic) bool {
	return candidateLedgerClockAdvance(origin, current) && uint64(current.MonotonicMS-origin.MonotonicMS) <= 300000
}

// Scalar provenance only, safe inside the observer mutex. All originals and
// model digests are independently replayed before this shape can become C.
func (o *providerHistoryGroupOwner) admissionShapeLocked() bool {
	if o == nil || o.owner != o || o.start == nil || o.admission == nil || o.admission != o.last || o.admission.owner != o ||
		o.admission.pin != o.pin || o.coordinator == nil || o.custodian == nil || o.coordinator == o.custodian ||
		o.coordinator.owner != o.coordinator || o.custodian.owner != o.custodian ||
		o.coordinator.phase != "admitted" || o.custodian.phase != "admitted" || o.coordinator.head != o.custodian.head ||
		o.coordinator.ticket == nil || o.custodian.ticket == nil || o.coordinator.ticket == o.custodian.ticket {
		return false
	}
	c := o.admission.claim
	return c != nil && c.owner == c && c.group == o && c.meter == o.meter && c.used && c.renamed &&
		c.frame != nil && c.frame.before == o.start.main && c.frame.after == o.pin && c.frame.phase == "admit" && c.frame.stage == 1 &&
		c.inventory != nil && c.inventory.count == 2 && c.inventory.digest == o.inventory && c.count == 2 &&
		c.request == c.inventory.entries[0] && c.ticket == c.inventory.entries[1] && c.request.kind == "requests" && c.ticket.kind == "ticket" &&
		c.coordinator == o.coordinator && c.members > 0 && c.members <= providerHistoryGroupMembers
}

// Reconstruct the original initial model and the entire acknowledged T from
// retained bytes. Selecting an old immutable pin never changes today's HEAD,
// old scan caches, original clock or live role. The result is DATA only.
func (o *providerHistoryGroupOwner) replayCommittedAdmissionLocked(head string) (*providerHistoryGroupLedger, error) {
	if !o.admissionShapeLocked() || !o.activeLocked(o.meter) {
		return nil, errProviderIntegration
	}
	raw, err := o.start.main.bytesObserved(o.store.config, o.meter)
	if err != nil {
		return nil, errProviderIntegration
	}
	initial, scope, err := o.initialModelAtLocked(o.start.main, providerReplayString(raw, o.meter), head)
	if err != nil || scope != o.scope {
		return nil, errProviderIntegration
	}
	model, err := o.readAdmissionDataLocked(o.admission.claim, o.start.main, providerReplayString(raw, o.meter), initial, head)
	if err != nil || !o.coordinator.validObserved(o.meter) || !o.custodian.validObserved(o.meter) ||
		!providerReplayTextEqual(model.head, o.coordinator.head, o.meter) || !providerReplayTextEqual(model.head, o.custodian.head, o.meter) {
		return nil, errProviderIntegration
	}
	return model, nil
}

// C consumes the entire already admitted group. It uses the SAME original
// group/lifetime work and clock windows, with no renewal or second hold.
// This private retained acknowledgment does not grant a transport permit.
func (o *providerHistoryGroupOwner) retainConsumptionCandidate(admission *providerHistoryGroupAcknowledgment) (*providerHistoryGroupAcknowledgment, error) {
	if o == nil || o.store == nil {
		return nil, errProviderIntegration
	}
	s := o.store
	s.mu.Lock()
	defer s.mu.Unlock()
	if o.phase != "admitted" || o.failed || o.parent == nil || o.parent.group != o ||
		admission == nil || admission != o.admission || !o.admissionShapeLocked() || o.order != nil ||
		s.activeMeter.Load() != nil || s.frontierFiles == nil || s.frontierFiles.pending != nil || s.frontierFiles.historyPending != nil ||
		!o.window || o.meter == nil || o.meter.owner != o.meter {
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
	o.phase = "preparing-consume"
	s.activeMeter.Store(m)
	succeeded := false
	defer func() {
		if !succeeded {
			o.failLocked()
		}
	}()
	if !o.fits(m) || !o.identityLocked() {
		return nil, errProviderIntegration
	}
	begin, err := s.samplePrimitive()
	if err != nil || !o.clockFits(begin) {
		return nil, errProviderIntegration
	}
	base, err := o.replayCommittedAdmissionLocked(s.frontierFiles.pinRaw)
	if err != nil || !providerHistoryGroupClockFits(admission.claim.frame.clock, providerClockDiagnostic(begin)) {
		return nil, errProviderIntegration
	}
	consumption, err := newProviderHistoryGroupConsumptionObserved(base.ticket, base.head, o.inventory, m)
	if err != nil {
		return nil, errProviderIntegration
	}
	object, err := newProviderHistoryStoredObjectObserved("consumption", consumption.raw, s.config.bounds.maximumObjectBytes, m)
	if err != nil {
		return nil, errProviderIntegration
	}
	fresh := []providerHistoryStoredObject{object}
	inventory, err := newProviderHistoryInventoryObserved(o.inventory, fresh, min(s.config.bounds.maximumMetadataPerCommit, candidateLedgerMetadataLimit), m)
	if err != nil {
		return nil, errProviderIntegration
	}
	coordinator := providerHistoryLedgerCopy(base)
	if coordinator == nil || coordinator.consumeObserved(consumption, m) != nil {
		return nil, errProviderIntegration
	}
	frame, err := newProviderHistoryCustodyFrameObserved(s.config, o.scope, o.pin, 1, "consume", o.inventory, inventory.digest, base.head, coordinator.head, consumption.digest, providerClockDiagnostic(begin), m)
	if err != nil {
		return nil, errProviderIntegration
	}
	raw, err := frame.after.bytesObserved(s.config, m)
	charge, ok := providerHistoryStorageChargeFor(inventory, fresh, frame, uint64(len(raw)), s.config.bounds.fileOverheadBytes)
	retained, retainedOK := o.retention.add(charge, o.parent.bounds.future)
	charged, groupOK := o.retention.add(charge, coordinator.charged)
	if err != nil || !ok || !retainedOK || !groupOK || charged != retained ||
		uint64(len(s.objects))+retained.objects > uint64(s.config.bounds.maximumObjects) || frame.after.sequence > uint64(s.config.bounds.maximumCommits) {
		return nil, errProviderIntegration
	}
	claim := &providerHistoryWriteClaim{group: o, meter: m, frame: frame, inventory: inventory, count: 1,
		admission: admission, consumption: inventory.entries[0], coordinator: coordinator, charge: charge, highwater: providerReplayString(raw, m)}
	claim.owner, claim.fresh[0] = claim, object
	if claim.highwater == "" || !o.admissionCatalogFitsLocked(claim) || !o.prepareAdmissionCatalogLocked(claim) || !o.exactAdmissionCatalogLocked(claim, false) {
		return nil, errProviderIntegration
	}
	order, err := newProviderHistoryGroupOrder(base.program, base.ticket.table.digest, base.ticket.table.count)
	if err != nil || !order.consumeAll() || order.consumed != consumption.consumed {
		return nil, errProviderIntegration
	}
	// Irreversible consumption precedes every physical C write. Even an
	// uncertain rename cannot restore any member or suballocation.
	o.order, claim.order, o.retention = order, order, retained
	s.frontierFiles.historyPending, o.phase = claim, "writing-consume"
	if o.writeAdmissionLocked(claim) != nil {
		return nil, errProviderIntegration
	}
	custodian, err := o.readConsumptionLocked(claim, true)
	if err != nil || custodian == coordinator || !providerReplayTextEqual(custodian.head, coordinator.head, m) ||
		!o.claimActiveLocked(claim) || !o.identityLocked() || !o.consumptionOrderLocked(claim) {
		return nil, errProviderIntegration
	}
	end, err := s.samplePrimitive()
	if err != nil || !o.clockFits(end) || !candidateLedgerClockAdvance(providerClockDiagnostic(begin), providerClockDiagnostic(end)) ||
		!providerHistoryGroupClockFits(admission.claim.frame.clock, providerClockDiagnostic(end)) || m.failed || m.closed ||
		!o.claimActiveLocked(claim) || !o.consumptionOrderLocked(claim) || !order.confirmConsumption() {
		return nil, errProviderIntegration
	}
	// Confirmation cannot fail after publication: every fallible operation
	// precedes the single serialized pin/model/ACK replacement below.
	s.frontierFiles.pin, s.frontierFiles.pinRaw = frame.after, claim.highwater
	o.pin, o.coordinator, o.custodian, o.inventory = frame.after, coordinator, custodian, inventory.digest
	s.sequence, s.tip = frame.after.sequence, frame.after.commit
	s.retainedRaw += charge.raw
	claim.used, s.frontierFiles.historyPending, o.phase = true, nil, "consumed"
	ack := &providerHistoryGroupAcknowledgment{o, claim, frame.after}
	o.last = ack
	s.activeMeter.Store(nil)
	succeeded = true
	return ack, nil
}

func (o *providerHistoryGroupOwner) consumptionOrderLocked(c *providerHistoryWriteClaim) bool {
	return c != nil && c.coordinator != nil && c.coordinator.ticket != nil && c.coordinator.consumption != nil &&
		c.order != nil && c.order == o.order && c.order.owner == c.order && c.order.valid() &&
		c.order.state == providerHistoryOrderConsumed && c.order.program == o.parent.bounds.program &&
		c.order.table == c.coordinator.ticket.table.digest && c.order.count == c.coordinator.ticket.table.count &&
		c.order.consumed == c.coordinator.consumption.consumed && c.order.sent == 0 && c.order.next == 0
}

func (o *providerHistoryGroupOwner) readConsumptionLocked(c *providerHistoryWriteClaim, renamed bool) (*providerHistoryGroupLedger, error) {
	if !o.claimActiveLocked(c) || c.frame.phase != "consume" || c.renamed != renamed || !o.consumptionOrderLocked(c) {
		return nil, errProviderIntegration
	}
	s, m := o.store, o.meter
	head := s.frontierFiles.pinRaw
	if renamed {
		head = c.highwater
	}
	if !s.frontierFiles.readExact(s.name("highwater", c.frame.after.ordinal), providerReplayBytes(c.highwater, m)) {
		return nil, errProviderIntegration
	}
	base, err := o.replayCommittedAdmissionLocked(head)
	if err != nil {
		return nil, errProviderIntegration
	}
	return o.readConsumptionDataLocked(c, o.pin, o.inventory, base, head)
}

// Replaying committed C selects historical DATA explicitly. It never changes
// the live phase, current pin, last acknowledgment or the original T/C models.
func (o *providerHistoryGroupOwner) readConsumptionDataLocked(c *providerHistoryWriteClaim, before providerFrontierHighwater, previousInventory string, base *providerHistoryGroupLedger, head string) (*providerHistoryGroupLedger, error) {
	if o == nil || c == nil || c.owner != c || c.group != o || c.meter != o.meter || c.frame == nil || c.inventory == nil ||
		c.count != 1 || c.inventory.count != 1 || c.consumption != c.inventory.entries[0] || c.consumption.kind != "consumption" ||
		c.coordinator == nil || base == nil || base.ticket == nil || base.phase != "admitted" ||
		c.frame.before != before || !lowerHash(previousInventory) || !o.activeLocked(o.meter) {
		return nil, errProviderIntegration
	}
	s, m := o.store, o.meter
	beforeRaw, err := before.bytesObserved(s.config, m)
	if err != nil || !s.frontierFiles.readExact("HEAD", providerReplayBytes(head, m)) ||
		!s.frontierFiles.readExact(s.name("highwater", before.ordinal), beforeRaw) ||
		!s.frontierFiles.readExact(s.name("highwater", c.frame.after.ordinal), providerReplayBytes(c.highwater, m)) {
		return nil, errProviderIntegration
	}
	ref := c.consumption
	raw, err := s.read(s.evidence, providerHistoryStoredName(ref, m), int64(ref.bytes))
	if err != nil || uint64(len(raw)) != ref.bytes {
		return nil, errProviderIntegration
	}
	object, err := newProviderHistoryStoredObjectObserved("consumption", providerReplayString(raw, m), s.config.bounds.maximumObjectBytes, m)
	if err != nil || !providerReplayTextEqual(object.digest, ref.digest, m) {
		return nil, errProviderIntegration
	}
	inventoryRaw, err := s.read(s.evidence, providerHistoryInventoryName(c.inventory.digest, m), int64(s.config.bounds.maximumMetadataPerCommit))
	if err != nil {
		return nil, errProviderIntegration
	}
	if _, err := parseProviderHistoryInventoryObserved(inventoryRaw, previousInventory, c.inventory.digest, []providerHistoryStoredObject{object}, min(s.config.bounds.maximumMetadataPerCommit, candidateLedgerMetadataLimit), m); err != nil {
		return nil, errProviderIntegration
	}
	consumption, err := parseProviderHistoryGroupConsumptionObserved(raw, base.ticket, base.head, previousInventory, m)
	if err != nil {
		return nil, errProviderIntegration
	}
	model := providerHistoryLedgerCopy(base)
	if model == nil || model.consumeObserved(consumption, m) != nil || !providerReplayTextEqual(model.head, c.coordinator.head, m) {
		return nil, errProviderIntegration
	}
	intent, a := s.read(s.custody, s.name("intent", c.frame.after.sequence), providerHistoryCustodyFrameBytes)
	pending, b := s.read(s.evidence, s.name("history-pending", c.frame.after.sequence), providerHistoryCustodyFrameBytes)
	commit, d := s.read(s.custody, s.name("commit", c.frame.after.sequence), providerHistoryCustodyFrameBytes)
	frontier, e := s.read(s.custody, s.name("frontier", c.frame.after.sequence), providerHistoryCustodyFrameBytes)
	frame, err := parseProviderHistoryCustodyFrameObserved(s.config, o.scope, before, c.frame.after, intent, commit, frontier, m)
	if a != nil || b != nil || d != nil || e != nil || err != nil || !providerReplayEqual(intent, pending, m) ||
		frame.previousInventory != previousInventory || frame.inventory != c.inventory.digest || frame.previousModel != base.head ||
		frame.model != model.head || frame.operation != consumption.digest || frame.phase != "consume" || frame.stage != 1 || frame.clock != c.frame.clock {
		return nil, errProviderIntegration
	}
	return model, nil
}
