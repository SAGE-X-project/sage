package registry011

import "context"

// Private completion DATA, produced after the exact physical S CAS and two
// original-byte reconstructions. It is not a Source or an operating profile.
// Production entry remains unavailable until whole-program derivation closes.
type providerHistorySettledResults struct {
	owner   *providerHistorySettledResults
	group   *providerHistoryGroupOwner
	ack     *providerHistoryGroupAcknowledgment
	count   int
	results [providerHistoryGroupMembers]providerHistoryMemberResult
	raw     string
}

func (*providerHistorySettledResults) UnmarshalJSON([]byte) error { return errProviderIntegration }

// Scalar provenance only; safe inside the original observer mutex. A complete
// retained prefix is still under immutable C until physical S and readback ACK.
func (d *providerHistoryGroupDispatch) completedMemberShapeLocked() bool {
	if d == nil || d.owner != d || d.group == nil || d.group.dispatch != d ||
		!d.group.committedConsumptionShapeLocked() || d.acknowledgment != d.group.last || d.group.admission.claim.frame.clock != d.ticketClock ||
		d.order == nil || d.order != d.group.order || !d.order.valid() || d.order.state != providerHistoryOrderConfirmed ||
		d.retained != d.order.count || d.order.next != d.order.count || d.attempt != nil || d.meter != d.group.meter ||
		d.coordinator == nil || d.custodian == nil || d.coordinator == d.custodian ||
		d.coordinator.owner != d.coordinator || d.custodian.owner != d.custodian ||
		d.coordinator.ticket == nil || d.custodian.ticket == nil || d.coordinator.ticket == d.custodian.ticket ||
		d.coordinator.consumption == nil || d.custodian.consumption == nil || d.coordinator.consumption == d.custodian.consumption ||
		d.coordinator.head != d.group.coordinator.head || d.custodian.head != d.group.custodian.head ||
		d.coordinator.ticket.table.count != d.order.count || d.custodian.ticket.table.count != d.order.count {
		return false
	}
	all, ok := providerHistoryMemberPrefix(d.order.count)
	if !ok || d.order.consumed != all || d.order.sent != all {
		return false
	}
	for i := range d.retained {
		r := d.originals[i]
		if r == nil || r.owner != r || r.dispatch != d || r.attempt == nil || r.attempt.owner != r.attempt ||
			r.attempt.dispatch != d || r.attempt.ack != d.acknowledgment || r.attempt.ordinal != i ||
			!r.attempt.used || r.attempt.files != 2 || r.attempt.original != r || r.attempt.connection != nil ||
			r.attempt.member != d.coordinator.ticket.table.members[i] || !lowerHash(r.result) {
			return false
		}
	}
	return d.group.parent.recovery.memberSelectionLocked(d.group)
}

// This candidate closes the SAME original C/order/hold/window. No response is
// returned from this path until independent post-CAS S reconstruction and the
// final original native-clock/cancellation bracket both succeed.
func (o *providerHistoryGroupOwner) retainSettlementCandidate(ctx context.Context, d *providerHistoryGroupDispatch) (*providerHistorySettledResults, error) {
	if o == nil || o.store == nil {
		return nil, errProviderIntegration
	}
	s := o.store
	s.mu.Lock()
	defer s.mu.Unlock()
	if ctx == nil || ctx.Err() != nil || o.owner != o || o.parent == nil || o.parent.group != o || o.failed ||
		o.phase != "dispatch-prepared" || d == nil || d != o.dispatch || !d.completedMemberShapeLocked() ||
		s.activeMeter.Load() != nil || s.frontierFiles == nil || s.frontierFiles.pending != nil || s.frontierFiles.historyPending != nil ||
		!o.parent.matchesLocked() || !o.parent.dispatchBounds.valid() {
		return nil, errProviderIntegration
	}
	m := o.meter
	if m == nil || m.owner != m || m.failed || m.closed || m.historyGroup != o || m.historyRecovery != nil ||
		m.integration != nil || m.scan != nil || m.scanBody != nil || m.firstAdmission != nil || m.firstOperation != nil ||
		m.guardian != nil || m.continuing != nil || m.parent != nil || !m.authority ||
		m.upper != o.parent.bounds.future.work || m.maximumClocks != o.parent.bounds.future.clocks {
		return nil, errProviderIntegration
	}
	o.phase = "preparing-settle"
	s.activeMeter.Store(m)
	succeeded := false
	defer func() {
		if !succeeded {
			o.failLocked()
		}
	}()
	rows := o.parent.dispatchBounds
	if !o.fits(m) || !m.preflightWork(rows.closure) || m.clocks > m.maximumClocks || rows.closureClocks > m.maximumClocks-m.clocks ||
		m.clocks < o.clockStart || m.clocks-o.clockStart > o.clockUpper || rows.closureClocks > o.clockUpper-(m.clocks-o.clockStart) ||
		!o.identityLocked() || !d.memberCatalogLocked() {
		return nil, errProviderIntegration
	}
	begin, err := s.samplePrimitive()
	if err != nil || !o.clockFits(begin) || !providerHistoryGroupClockFits(d.ticketClock, providerClockDiagnostic(begin)) {
		return nil, errProviderIntegration
	}
	base, err := o.replayCommittedConsumptionLocked(ctx)
	if err != nil {
		return nil, errProviderIntegration
	}
	roots := d.rootsObservedLocked()
	prefix, err := d.readOriginalPrefixCandidateLocked(d.retained, roots)
	if err != nil {
		return nil, errProviderIntegration
	}
	digests := make([]string, d.retained)
	for i := range digests {
		digests[i] = prefix.results[i].digest
	}
	settlement, err := newProviderHistoryGroupSettlementObserved(base.ticket, base.consumption, base.head, o.inventory, "complete", "", digests, m)
	if err != nil {
		return nil, errProviderIntegration
	}
	resultObject, err := newProviderHistoryStoredObjectObserved("results", prefix.pack.raw, s.config.bounds.maximumObjectBytes, m)
	settlementObject, otherErr := newProviderHistoryStoredObjectObserved("settlement", settlement.raw, s.config.bounds.maximumObjectBytes, m)
	if err != nil || otherErr != nil {
		return nil, errProviderIntegration
	}
	fresh := []providerHistoryStoredObject{resultObject, settlementObject}
	inventory, err := newProviderHistoryInventoryObserved(o.inventory, fresh, min(s.config.bounds.maximumMetadataPerCommit, candidateLedgerMetadataLimit), m)
	if err != nil {
		return nil, errProviderIntegration
	}
	coordinator := providerHistoryLedgerCopyObserved(base, m)
	if coordinator == nil || coordinator.settleOriginalsObserved(ctx, settlement, prefix.results[:d.retained], s.config.bounds.methods, m) != nil || !coordinator.validObserved(m) {
		return nil, errProviderIntegration
	}
	frame, err := newProviderHistoryCustodyFrameObserved(s.config, o.scope, o.pin, base.ticket.table.stage, "settle", o.inventory, inventory.digest, base.head, coordinator.head, settlement.digest, providerClockDiagnostic(begin), m)
	if err != nil {
		return nil, errProviderIntegration
	}
	raw, err := frame.after.bytesObserved(s.config, m)
	charge, chargeOK := providerHistoryStorageChargeFor(inventory, fresh, frame, uint64(len(raw)), s.config.bounds.fileOverheadBytes)
	retained, retainedOK := o.retention.add(charge, o.parent.bounds.future)
	groupRetained, groupOK := o.retention.add(charge, coordinator.charged)
	if err != nil || !chargeOK || !retainedOK || !groupOK || retained != groupRetained || !retained.withinStage(o.stageRetentionStart, base.ticket.upper) ||
		uint64(len(s.objects))+retained.objects > uint64(s.config.bounds.maximumObjects) || frame.after.sequence > uint64(s.config.bounds.maximumCommits) {
		return nil, errProviderIntegration
	}
	claim := &providerHistoryWriteClaim{group: o, meter: m, frame: frame, inventory: inventory, count: 2,
		dispatch: d, consumed: o.last, order: o.order, coordinator: coordinator, charge: charge,
		results: inventory.entries[0], settlement: inventory.entries[1], highwater: providerReplayString(raw, m)}
	claim.owner = claim
	copy(claim.fresh[:], fresh)
	if claim.highwater == "" || !o.admissionCatalogFitsLocked(claim) || !o.prepareAdmissionCatalogLocked(claim) || !o.exactAdmissionCatalogLocked(claim, false) || !o.parent.recovery.prepareWriteCheckpointLocked(claim) {
		return nil, errProviderIntegration
	}
	checkpoint, ack := claim.checkpoint, claim.checkpoint.ack
	// The physical allocation is irreversible, including failed writes/rename.
	o.retention = retained
	s.frontierFiles.historyPending, o.phase = claim, "writing-settle"
	if o.writeAdmissionLocked(ctx, claim) != nil {
		return nil, errProviderIntegration
	}
	custodian, checked, err := o.readSettlementLocked(ctx, claim, true)
	if err != nil || custodian == coordinator || !providerReplayTextEqual(custodian.head, coordinator.head, m) {
		return nil, errProviderIntegration
	}
	// All output copies are observed BEFORE the final bracket and publication.
	output := &providerHistorySettledResults{group: o, count: d.retained}
	output.owner = output
	output.raw = providerReplayString(providerReplayBytes(checked.pack.raw, m), m)
	for i := range output.count {
		r := checked.results[i]
		output.results[i] = providerHistoryMemberResult{
			providerReplayString(providerReplayBytes(r.digest, m), m),
			providerReplayString(providerReplayBytes(r.response, m), m),
			providerReplayString(providerReplayBytes(r.result, m), m),
		}
		if output.results[i].digest == "" || output.results[i].response == "" || output.results[i].result == "" {
			return nil, errProviderIntegration
		}
	}
	// Prepare even the small ACK allocation before the final bracket. It is
	// private and unavailable until the single serialized commit below.
	output.ack = ack
	if output.raw == "" || !o.claimActiveLocked(claim) || !o.identityLocked() || !o.exactAdmissionCatalogLocked(claim, true) || !o.parent.recovery.pendingCheckpointDataMatchesLocked(claim) {
		return nil, errProviderIntegration
	}
	finalWork := m.actual
	c := o.parent.clock
	c.mu.Lock()
	finalRead := c.read
	c.mu.Unlock()
	end, err := s.samplePrimitive()
	if err != nil || !o.clockFits(end) || !candidateLedgerClockAdvance(providerClockDiagnostic(begin), providerClockDiagnostic(end)) ||
		!providerHistoryGroupClockFits(d.ticketClock, providerClockDiagnostic(end)) {
		return nil, errProviderIntegration
	}
	for i, value := range finalWork {
		if i != 10 && m.actual[i] != value {
			return nil, errProviderIntegration
		}
	}
	units, unitsOK := providerUpperSum(finalWork[10], 1)
	c.mu.Lock()
	read := c.read
	c.mu.Unlock()
	if !unitsOK || m.actual[10] != units || finalRead == ^uint64(0) || read != finalRead+1 ||
		ctx.Err() != nil || !d.completedMemberShapeLocked() || !o.order.close() {
		return nil, errProviderIntegration
	}
	// No observations, copies or fallible operations follow this single commit.
	s.frontierFiles.pin, s.frontierFiles.pinRaw = frame.after, claim.highwater
	o.pin, o.coordinator, o.custodian, o.inventory = frame.after, coordinator, custodian, inventory.digest
	s.sequence, s.tip = frame.after.sequence, frame.after.commit
	s.retainedRaw += charge.raw
	claim.used, s.frontierFiles.historyPending, o.phase = true, nil, "settled"
	o.last = ack
	o.parent.recovery.confirmed, o.parent.recovery.pending, o.parent.recovery.pendingAck = checkpoint, nil, nil
	o.releaseMemberReplayLocked()
	s.activeMeter.Store(nil)
	succeeded = true
	return output, nil
}

// Independent retained-byte DATA replay. It explicitly selects current HEAD
// before/after physical S rename while all live pins/models still name C.
func (o *providerHistoryGroupOwner) readSettlementLocked(ctx context.Context, c *providerHistoryWriteClaim, renamed bool) (*providerHistoryGroupLedger, *providerHistoryRetainedPrefix, error) {
	if ctx == nil || ctx.Err() != nil {
		return nil, nil, errProviderIntegration
	}
	if !o.claimActiveLocked(c) || c.frame.phase != "settle" || c.renamed != renamed {
		return nil, nil, errProviderIntegration
	}
	s, m := o.store, o.meter
	head := s.frontierFiles.pinRaw
	if renamed {
		head = c.highwater
	}
	base, err := o.replayCommittedConsumptionAtLocked(ctx, head)
	if err != nil {
		return nil, nil, errProviderIntegration
	}
	if !s.frontierFiles.readExact(s.name("highwater", c.frame.after.ordinal), providerReplayBytes(c.highwater, m)) {
		return nil, nil, errProviderIntegration
	}
	var originals [2]providerHistoryStoredObject
	for i := range originals {
		ref := c.inventory.entries[i]
		raw, err := s.read(s.evidence, providerHistoryStoredName(ref, m), int64(ref.bytes))
		if err != nil || uint64(len(raw)) != ref.bytes {
			return nil, nil, errProviderIntegration
		}
		originals[i], err = newProviderHistoryStoredObjectObserved(ref.kind, providerReplayString(raw, m), s.config.bounds.maximumObjectBytes, m)
		if err != nil || !providerReplayTextEqual(originals[i].digest, ref.digest, m) {
			return nil, nil, errProviderIntegration
		}
	}
	inventoryRaw, err := s.read(s.evidence, providerHistoryInventoryName(c.inventory.digest, m), int64(s.config.bounds.maximumMetadataPerCommit))
	if err != nil {
		return nil, nil, errProviderIntegration
	}
	if _, err := parseProviderHistoryInventoryObserved(inventoryRaw, o.inventory, c.inventory.digest, originals[:], min(s.config.bounds.maximumMetadataPerCommit, candidateLedgerMetadataLimit), m); err != nil {
		return nil, nil, errProviderIntegration
	}
	prefix, err := c.dispatch.readOriginalPrefixCandidateLocked(c.dispatch.retained, c.dispatch.rootsObservedLocked())
	if err != nil || !providerReplayTextEqual(originals[0].raw, prefix.pack.raw, m) {
		return nil, nil, errProviderIntegration
	}
	digests := make([]string, c.dispatch.retained)
	for i := range digests {
		digests[i] = prefix.results[i].digest
	}
	settlement, err := parseProviderHistoryGroupSettlementObserved(providerReplayBytes(originals[1].raw, m), base.ticket, base.consumption, base.head, o.inventory, "complete", "", digests, m)
	if err != nil || !providerReplayTextEqual(settlement.digest, c.settlement.digest, m) {
		return nil, nil, errProviderIntegration
	}
	model := providerHistoryLedgerCopyObserved(base, m)
	if model == nil || model.settleOriginalsObserved(ctx, settlement, prefix.results[:c.dispatch.retained], s.config.bounds.methods, m) != nil || !model.validObserved(m) || !providerReplayTextEqual(model.head, c.coordinator.head, m) {
		return nil, nil, errProviderIntegration
	}
	intent, a := s.read(s.custody, s.name("intent", c.frame.after.sequence), providerHistoryCustodyFrameBytes)
	pending, b := s.read(s.evidence, s.name("history-pending", c.frame.after.sequence), providerHistoryCustodyFrameBytes)
	commit, d := s.read(s.custody, s.name("commit", c.frame.after.sequence), providerHistoryCustodyFrameBytes)
	frontier, e := s.read(s.custody, s.name("frontier", c.frame.after.sequence), providerHistoryCustodyFrameBytes)
	frame, err := parseProviderHistoryCustodyFrameObserved(s.config, o.scope, o.pin, c.frame.after, intent, commit, frontier, m)
	if a != nil || b != nil || d != nil || e != nil || err != nil || !providerReplayEqual(intent, pending, m) ||
		frame.previousInventory != o.inventory || frame.inventory != c.inventory.digest || frame.previousModel != base.head ||
		frame.model != model.head || frame.operation != settlement.digest || frame.phase != "settle" || frame.stage != base.ticket.table.stage || frame.clock != c.frame.clock ||
		!o.identityLocked() || !o.exactAdmissionCatalogLocked(c, true) || ctx.Err() != nil {
		return nil, nil, errProviderIntegration
	}
	return model, prefix, nil
}
