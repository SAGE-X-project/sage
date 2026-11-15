package registry011

import "context"

// Scalar-only role for the exact failed attempt following the retained
// successful prefix. It never advances next or pretends the failed member
// completed. Called while the original store or observer mutex is held.
func (d *providerHistoryGroupDispatch) failedMemberShapeLocked() bool {
	if d == nil || d.owner != d || d.group == nil || d.group.dispatch != d || !d.group.committedConsumptionShapeLocked() ||
		d.acknowledgment != d.group.last || d.ticketClock != d.group.admission.claim.frame.clock || d.meter != d.group.meter ||
		d.order == nil || d.order != d.group.order || !d.order.valid() || d.order.state != providerHistoryOrderInFlight ||
		d.retained != d.order.next || d.retained < 0 || d.retained >= d.order.count ||
		d.coordinator == nil || d.custodian == nil || d.coordinator == d.custodian || d.coordinator.owner != d.coordinator || d.custodian.owner != d.custodian ||
		d.coordinator.ticket == nil || d.custodian.ticket == nil || d.coordinator.ticket == d.custodian.ticket || d.coordinator.ticket.table.count != d.order.count || d.custodian.ticket.table.count != d.order.count ||
		d.coordinator.consumption == nil || d.custodian.consumption == nil || d.coordinator.consumption == d.custodian.consumption || d.coordinator.head != d.group.coordinator.head || d.custodian.head != d.group.custodian.head {
		return false
	}
	a := d.attempt
	if !d.group.parent.recovery.memberAttemptShapeLocked(a) || d.group.parent.recovery.memberCapture != nil {
		return false
	}
	if a == nil || a.owner != a || a.dispatch != d || a.ack != d.acknowledgment || a.ordinal != d.retained || !a.used || a.files != 0 || a.original != nil || a.connection != nil ||
		a.failure == nil || !a.failure.knownFailure || a.failure.uncertain || a.member != d.coordinator.ticket.table.members[d.retained] {
		return false
	}
	for i := range d.retained {
		r := d.originals[i]
		if r == nil || r.owner != r || r.dispatch != d || r.attempt == nil || r.attempt.owner != r.attempt || r.attempt.dispatch != d || r.attempt.ordinal != i ||
			!r.attempt.used || r.attempt.files != 2 || r.attempt.original != r || r.attempt.connection != nil || !lowerHash(r.result) {
			return false
		}
	}
	return d.group.parent.recovery.memberSelectionLocked(d.group)
}

func (o *providerHistoryGroupOwner) failedSettlementClaimShapeLocked(c *providerHistoryWriteClaim) bool {
	if c == nil || c.dispatch == nil || c.dispatch != o.dispatch || !c.dispatch.failedMemberShapeLocked() || c.consumed != o.last || c.order != o.order ||
		c.inventory == nil || c.inventory.previous != o.inventory || c.failure.kind != "failure" || c.failure != c.inventory.entries[0] ||
		c.coordinator == nil || c.coordinator.owner != c.coordinator || c.coordinator.phase != "failed" || !c.coordinator.failed || !c.coordinator.terminal ||
		c.coordinator.ticket != nil || c.coordinator.consumption != nil {
		return false
	}
	count := 2
	if c.dispatch.retained > 0 {
		count = 3
	}
	if c.count != count || c.inventory.count != count || c.settlement != c.inventory.entries[count-1] || c.settlement.kind != "settlement" {
		return false
	}
	if c.dispatch.retained == 0 {
		return c.results == (providerHistoryStoredReference{})
	}
	return c.results == c.inventory.entries[1] && c.results.kind == "results"
}

// Retain a known failure while the SAME original meter, clock, C, window and
// allowance are healthy. The caller holds store.mu and has closed the actual
// attempt connection. Poisoned/cancelled/expired paths never borrow a new
// observer to manufacture failed S; their ordinary quarantine remains final.
// A successful return acknowledges failure custody, never a protected result.
func (o *providerHistoryGroupOwner) retainFailedCaptureLocked(ctx context.Context, d *providerHistoryGroupDispatch, a *providerHistoryDispatchAttempt) error {
	if ctx == nil || ctx.Err() != nil || o == nil || o.owner != o || o.store == nil || o.parent == nil || o.parent.group != o || o.failed ||
		o.phase != "capturing-member" || d == nil || d != o.dispatch || a == nil || a != d.attempt || !d.failedMemberShapeLocked() ||
		o.store.frontierFiles == nil || o.store.frontierFiles.pending != nil || o.store.frontierFiles.historyPending != nil ||
		o.meter == nil || o.meter.owner != o.meter || o.meter.failed || o.meter.closed || o.store.activeMeter.Load() != o.meter ||
		!o.parent.matchesLocked() || !o.parent.dispatchBounds.valid() {
		return errProviderIntegration
	}
	s, m := o.store, o.meter
	if m.historyGroup != o || m.historyRecovery != nil || m.integration != nil || m.scan != nil || m.scanBody != nil || m.firstAdmission != nil || m.firstOperation != nil || m.guardian != nil || m.continuing != nil || m.parent != nil ||
		!m.authority || m.upper != o.parent.bounds.future.work || m.maximumClocks != o.parent.bounds.future.clocks {
		return errProviderIntegration
	}
	o.phase = "preparing-failed-settle"
	rows := o.parent.dispatchBounds
	if !m.preflightWork(rows.closure) || m.clocks > m.maximumClocks || rows.closureClocks > m.maximumClocks-m.clocks ||
		m.clocks < o.clockStart || m.clocks-o.clockStart > o.clockUpper || rows.closureClocks > o.clockUpper-(m.clocks-o.clockStart) ||
		!o.identityLocked() || !d.memberCatalogLocked() {
		return errProviderIntegration
	}
	begin, err := s.samplePrimitive()
	if err != nil || !o.clockFits(begin) || !providerHistoryGroupClockFits(d.ticketClock, providerClockDiagnostic(begin)) ||
		!candidateLedgerClockAdvance(providerClockDiagnostic(a.begin), providerClockDiagnostic(begin)) {
		return errProviderIntegration
	}
	base, err := o.replayCommittedConsumptionLocked(ctx)
	if err != nil {
		return errProviderIntegration
	}
	roots := d.rootsObservedLocked()
	failure, err := newProviderHistoryFailureOriginalObserved(base.ticket, base.consumption, a.ordinal, a.failure, roots, o.parent.tls, s.config.bounds.maximumObjectBytes, m)
	if err != nil {
		return errProviderIntegration
	}
	failureObject, err := newProviderHistoryStoredObjectObserved("failure", failure.raw, s.config.bounds.maximumObjectBytes, m)
	if err != nil || !providerReplayTextEqual(failureObject.digest, failure.digest, m) {
		return errProviderIntegration
	}
	fresh := []providerHistoryStoredObject{failureObject}
	digests := make([]string, d.retained)
	if d.retained > 0 {
		prefix, err := d.readOriginalPrefixCandidateLocked(d.retained, roots)
		if err != nil {
			return errProviderIntegration
		}
		object, err := newProviderHistoryStoredObjectObserved("results", prefix.pack.raw, s.config.bounds.maximumObjectBytes, m)
		if err != nil {
			return errProviderIntegration
		}
		fresh = append(fresh, object)
		for i := range digests {
			digests[i] = prefix.results[i].digest
		}
	}
	settlement, err := newProviderHistoryGroupSettlementObserved(base.ticket, base.consumption, base.head, o.inventory, "failed", failure.digest, digests, m)
	if err != nil {
		return errProviderIntegration
	}
	settlementObject, err := newProviderHistoryStoredObjectObserved("settlement", settlement.raw, s.config.bounds.maximumObjectBytes, m)
	if err != nil {
		return errProviderIntegration
	}
	fresh = append(fresh, settlementObject)
	inventory, err := newProviderHistoryInventoryObserved(o.inventory, fresh, min(s.config.bounds.maximumMetadataPerCommit, candidateLedgerMetadataLimit), m)
	if err != nil {
		return errProviderIntegration
	}
	coordinator := providerHistoryLedgerCopyObserved(base, m)
	if coordinator == nil || coordinator.settleOriginalsObserved(ctx, settlement, nil, nil, m) != nil || !coordinator.terminalDataValidObserved(m) {
		return errProviderIntegration
	}
	frame, err := newProviderHistoryCustodyFrameObserved(s.config, o.scope, o.pin, base.ticket.table.stage, "settle", o.inventory, inventory.digest, base.head, coordinator.head, settlement.digest, providerClockDiagnostic(begin), m)
	if err != nil {
		return errProviderIntegration
	}
	raw, err := frame.after.bytesObserved(s.config, m)
	charge, chargeOK := providerHistoryStorageChargeFor(inventory, fresh, frame, uint64(len(raw)), s.config.bounds.fileOverheadBytes)
	retained, retainedOK := o.retention.add(charge, o.parent.bounds.future)
	groupRetained, groupOK := o.retention.add(charge, coordinator.charged)
	if err != nil || !chargeOK || !retainedOK || !groupOK || retained != groupRetained || !retained.withinStage(o.stageRetentionStart, base.ticket.upper) ||
		uint64(len(s.objects))+retained.objects > uint64(s.config.bounds.maximumObjects) || frame.after.sequence > uint64(s.config.bounds.maximumCommits) {
		return errProviderIntegration
	}
	c := &providerHistoryWriteClaim{group: o, meter: m, frame: frame, inventory: inventory, count: len(fresh), failure: inventory.entries[0], settlement: inventory.entries[len(fresh)-1],
		dispatch: d, consumed: o.last, order: o.order, coordinator: coordinator, charge: charge, highwater: providerReplayString(raw, m)}
	c.owner = c
	copy(c.fresh[:], fresh)
	if d.retained > 0 {
		c.results = inventory.entries[1]
	}
	if c.highwater == "" || !o.admissionCatalogFitsLocked(c) || !o.prepareAdmissionCatalogLocked(c) || !o.exactAdmissionCatalogLocked(c, false) || !o.parent.recovery.prepareWriteCheckpointLocked(c) {
		return errProviderIntegration
	}
	checkpoint, ack := c.checkpoint, c.checkpoint.ack
	o.retention = retained
	s.frontierFiles.historyPending, o.phase = c, "writing-failed-settle"
	if o.writeAdmissionLocked(ctx, c) != nil {
		return errProviderIntegration
	}
	custodian, err := o.readFailedSettlementLocked(ctx, c, true)
	if err != nil || custodian == coordinator || !providerReplayTextEqual(custodian.head, coordinator.head, m) {
		return errProviderIntegration
	}
	if !o.claimActiveLocked(c) || !o.identityLocked() || !o.exactAdmissionCatalogLocked(c, true) || !o.parent.recovery.pendingCheckpointDataMatchesLocked(c) ||
		!o.parent.recovery.memberAttemptDataMatchesLocked(a.recoveryAttempt, a) {
		return errProviderIntegration
	}
	finalWork := m.actual
	clock := o.parent.clock
	clock.mu.Lock()
	finalRead := clock.read
	clock.mu.Unlock()
	end, err := s.samplePrimitive()
	if err != nil || !o.clockFits(end) || !candidateLedgerClockAdvance(providerClockDiagnostic(begin), providerClockDiagnostic(end)) ||
		!providerHistoryGroupClockFits(d.ticketClock, providerClockDiagnostic(end)) {
		return errProviderIntegration
	}
	for i, value := range finalWork {
		if i != 10 && m.actual[i] != value {
			return errProviderIntegration
		}
	}
	units, unitsOK := providerUpperSum(finalWork[10], 1)
	clock.mu.Lock()
	read := clock.read
	clock.mu.Unlock()
	if !unitsOK || m.actual[10] != units || finalRead == ^uint64(0) || read != finalRead+1 || ctx.Err() != nil || !d.failedMemberShapeLocked() {
		return errProviderIntegration
	}
	// Terminal failure never calls order.close or fabricates completed bits.
	// The final ACK is already prepared; only scalar/pointer publication and
	// revocation of this same lifetime follow the final original end sample.
	s.frontierFiles.pin, s.frontierFiles.pinRaw = frame.after, c.highwater
	o.pin, o.coordinator, o.custodian, o.inventory = frame.after, coordinator, custodian, inventory.digest
	s.sequence, s.tip = frame.after.sequence, frame.after.commit
	s.retainedRaw += charge.raw
	c.used, s.frontierFiles.historyPending, o.last = true, nil, ack
	o.parent.recovery.confirmed, o.parent.recovery.pending, o.parent.recovery.pendingAck = checkpoint, nil, nil
	o.failLocked()
	return nil
}

func (o *providerHistoryGroupOwner) readFailedSettlementLocked(ctx context.Context, c *providerHistoryWriteClaim, renamed bool) (*providerHistoryGroupLedger, error) {
	if ctx == nil || ctx.Err() != nil {
		return nil, errProviderIntegration
	}
	if !o.claimActiveLocked(c) || c.frame.phase != "settle" || c.failure.kind != "failure" || c.renamed != renamed {
		return nil, errProviderIntegration
	}
	s, m := o.store, o.meter
	head := s.frontierFiles.pinRaw
	if renamed {
		head = c.highwater
	}
	base, err := o.replayCommittedConsumptionAtLocked(ctx, head)
	if err != nil || !s.frontierFiles.readExact(s.name("highwater", c.frame.after.ordinal), providerReplayBytes(c.highwater, m)) {
		return nil, errProviderIntegration
	}
	var objects [3]providerHistoryStoredObject
	for i := range c.count {
		ref := c.inventory.entries[i]
		raw, err := s.read(s.evidence, providerHistoryStoredName(ref, m), int64(ref.bytes))
		if err != nil || ctx.Err() != nil || uint64(len(raw)) != ref.bytes {
			return nil, errProviderIntegration
		}
		objects[i], err = newProviderHistoryStoredObjectObserved(ref.kind, providerReplayString(raw, m), s.config.bounds.maximumObjectBytes, m)
		if err != nil || !providerReplayTextEqual(objects[i].digest, ref.digest, m) {
			return nil, errProviderIntegration
		}
	}
	inventoryRaw, err := s.read(s.evidence, providerHistoryInventoryName(c.inventory.digest, m), int64(s.config.bounds.maximumMetadataPerCommit))
	if err != nil {
		return nil, errProviderIntegration
	}
	if _, err := parseProviderHistoryInventoryObserved(inventoryRaw, o.inventory, c.inventory.digest, objects[:c.count], min(s.config.bounds.maximumMetadataPerCommit, candidateLedgerMetadataLimit), m); err != nil {
		return nil, errProviderIntegration
	}
	d := c.dispatch
	roots := d.rootsObservedLocked()
	failure, err := parseProviderHistoryFailureOriginalObserved(providerReplayBytes(objects[0].raw, m), c.failure.digest, base.ticket, base.consumption, d.retained, roots, o.parent.tls, s.config.bounds.maximumObjectBytes, m)
	if err != nil || failure.stage != d.attempt.failure.failureStage || failure.written != d.attempt.failure.requestWritten || failure.received != d.attempt.failure.responseRead {
		return nil, errProviderIntegration
	}
	digests := make([]string, d.retained)
	if d.retained > 0 {
		prefix, err := d.readOriginalPrefixCandidateLocked(d.retained, roots)
		if err != nil || !providerReplayTextEqual(objects[1].raw, prefix.pack.raw, m) {
			return nil, errProviderIntegration
		}
		for i := range digests {
			digests[i] = prefix.results[i].digest
		}
	}
	settlement, err := parseProviderHistoryGroupSettlementObserved(providerReplayBytes(objects[c.count-1].raw, m), base.ticket, base.consumption, base.head, o.inventory, "failed", failure.digest, digests, m)
	if err != nil || !providerReplayTextEqual(settlement.digest, c.settlement.digest, m) {
		return nil, errProviderIntegration
	}
	model := providerHistoryLedgerCopyObserved(base, m)
	if model == nil || model.settleOriginalsObserved(ctx, settlement, nil, nil, m) != nil || !model.terminalDataValidObserved(m) || !providerReplayTextEqual(model.head, c.coordinator.head, m) {
		return nil, errProviderIntegration
	}
	intent, a := s.read(s.custody, s.name("intent", c.frame.after.sequence), providerHistoryCustodyFrameBytes)
	pending, b := s.read(s.evidence, s.name("history-pending", c.frame.after.sequence), providerHistoryCustodyFrameBytes)
	commit, e := s.read(s.custody, s.name("commit", c.frame.after.sequence), providerHistoryCustodyFrameBytes)
	frontier, f := s.read(s.custody, s.name("frontier", c.frame.after.sequence), providerHistoryCustodyFrameBytes)
	frame, err := parseProviderHistoryCustodyFrameObserved(s.config, o.scope, o.pin, c.frame.after, intent, commit, frontier, m)
	if a != nil || b != nil || e != nil || f != nil || err != nil || !providerReplayEqual(intent, pending, m) || frame.previousInventory != o.inventory || frame.inventory != c.inventory.digest ||
		frame.previousModel != base.head || frame.model != model.head || frame.operation != settlement.digest || frame.phase != "settle" || frame.stage != base.ticket.table.stage || frame.clock != c.frame.clock ||
		!o.identityLocked() || !o.exactAdmissionCatalogLocked(c, true) || ctx.Err() != nil {
		return nil, errProviderIntegration
	}
	return model, nil
}
