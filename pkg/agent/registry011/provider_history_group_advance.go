package registry011

import "context"

// The new multistage path admits complete uppers, including initialization
// outside all T windows. Transition work is already inside each old T upper.
// This does not reinterpret the older single-stage candidate as a sufficient
// whole-program derivation; that separate production factory is still absent.
// The caller holds the real meter mutex; only checked scalar arithmetic runs.
func (o *providerHistoryGroupOwner) admissionBudgetFitsLocked(upper providerHistoryGroupUpper, work candidateWorkVector) bool {
	charged, ok := o.coordinator.charged.add(upper, o.parent.bounds.future)
	if !ok {
		return false
	}
	if o.parent.bounds.stages == 1 {
		return true
	}
	initial, ok := providerHistoryGroupInitializationForProfileUpper(o.store.config.bounds)
	if !ok {
		return false
	}
	for i, value := range initial {
		need, sumOK := providerUpperSum(value, charged.work[i])
		if !sumOK || need > o.parent.bounds.future.work[i] || !candidateCapacity(0, o.meter.actual[i], work[i], o.meter.upper[i]) {
			return false
		}
	}
	clocks, ok := providerUpperSum(2, charged.clocks)
	return ok && clocks <= o.parent.bounds.future.clocks && candidateCapacity(0, o.meter.clocks, upper.clocks, o.meter.maximumClocks)
}

// Scalar-only idle T selection, also used by the active observer predicate.
// Across a transition the ledger stays settled at 3*k; only admit advances it.
func (o *providerHistoryGroupOwner) admissionBaseShapeLocked() bool {
	c := o.stageBaseLocked()
	if c == nil || o.stageRetentionStart != c.retention || o.nextStageLocked() == 0 || o.parent.recovery.confirmed != c || o.pin != c.pin || o.last != c.ack ||
		o.admission != nil || o.order != nil || o.dispatch != nil || !o.parent.recovery.memberSelectionLocked(o) ||
		o.coordinator == nil || o.custodian == nil || o.coordinator == o.custodian ||
		o.coordinator.owner != o.coordinator || o.custodian.owner != o.custodian ||
		o.coordinator.failed || o.custodian.failed || o.coordinator.terminal || o.custodian.terminal ||
		o.coordinator.head != c.model || o.custodian.head != c.model || o.coordinator.charged != o.custodian.charged ||
		o.coordinator.ticket != nil || o.custodian.ticket != nil || o.coordinator.consumption != nil || o.custodian.consumption != nil ||
		o.coordinator.stage != c.frame.stage || o.custodian.stage != c.frame.stage ||
		o.coordinator.sequence != 3*c.frame.stage || o.custodian.sequence != 3*c.frame.stage {
		return false
	}
	if c == o.parent.recovery.initial {
		return o.coordinator.phase == "prepared" && o.custodian.phase == "prepared" && o.inventory == o.start.objects
	}
	return o.coordinator.phase == "settled" && o.custodian.phase == "settled" && o.inventory == c.inventory.digest
}

// Do not call completedMemberShape here: S deliberately cleared the live
// ledger's T/C pointers. The original archive and closed order are the seals.
func (o *providerHistoryGroupOwner) successfulSettlementShapeLocked() bool {
	if o == nil || o.owner != o || o.parent == nil || o.parent.recovery == nil || o.stageBaseLocked() == nil ||
		o.coordinator == nil || o.custodian == nil || o.coordinator == o.custodian {
		return false
	}
	r, c := o.parent.recovery, o.parent.recovery.confirmed
	if !o.healthyCheckpointShapeLocked(c) || c == o.stageBefore || c.failedS || c.frame.phase != "settle" || c.ack != o.last ||
		c.pin != o.pin || c.inventory.digest != o.inventory || c.admission.checkpoint.previous != o.stageBefore ||
		c.frame.stage != o.nextStageLocked() || c.memberArchive == nil || c.memberArchive.order != o.order ||
		o.order == nil || o.order.state != providerHistoryOrderClosed || !o.order.valid() ||
		o.dispatch == nil || o.dispatch != c.memberArchive.dispatch || o.dispatch.attempt != nil ||
		r.pending != nil || r.pendingAck != nil || r.memberAttempt != nil || r.memberCapture != nil || r.memberOriginal != nil ||
		!r.memberSelectionLocked(o) || c.memberArchive.count != o.order.count ||
		o.coordinator.owner != o.coordinator || o.custodian.owner != o.custodian ||
		o.coordinator.phase != "settled" || o.custodian.phase != "settled" ||
		o.coordinator.failed || o.custodian.failed || o.coordinator.terminal || o.custodian.terminal ||
		o.coordinator.stage != c.frame.stage || o.custodian.stage != c.frame.stage ||
		o.coordinator.sequence != 3*c.frame.stage || o.custodian.sequence != 3*c.frame.stage ||
		o.coordinator.head != c.model || o.custodian.head != c.model || o.coordinator.charged != o.custodian.charged ||
		o.coordinator.ticket != nil || o.custodian.ticket != nil || o.coordinator.consumption != nil || o.custodian.consumption != nil {
		return false
	}
	return true
}

// The interstage row belongs INSIDE the old T upper and the same original
// lifetime. The nonserializable pre-H0 witness must supply it. Synthetic rows
// in isolated tests exercise admission, not a production cost certificate.
func (b providerHistoryDispatchBounds) advanceValid() bool {
	return b.valid() && b.advance[0] == 0 && b.advance[1] == 0 && b.advance[9] == 0 && b.advance[2] > 0 && b.advance[10] > 0 &&
		b.advanceClocks == 2 && b.advanceTailMS > 0 && b.advanceTailMS < 300000
}

func (o *providerHistoryGroupOwner) advanceStageCandidate(ctx context.Context, ack *providerHistoryGroupAcknowledgment) error {
	if o == nil || o.store == nil {
		return errProviderIntegration
	}
	s := o.store
	s.mu.Lock()
	defer s.mu.Unlock()
	if ctx == nil || ctx.Err() != nil || o.phase != "settled" || o.failed || ack == nil || ack != o.last ||
		!o.successfulSettlementShapeLocked() || !o.parent.matchesLocked() || !o.parent.dispatchBounds.advanceValid() ||
		s.activeMeter.Load() != nil || s.frontierFiles == nil || s.frontierFiles.pending != nil || s.frontierFiles.historyPending != nil ||
		o.coordinator.stage >= o.parent.bounds.stages || o.coordinator.sequence > o.parent.bounds.records ||
		3 > o.parent.bounds.records-o.coordinator.sequence || o.pin.sequence > uint64(s.config.bounds.maximumCommits) ||
		3 > uint64(s.config.bounds.maximumCommits)-o.pin.sequence {
		return errProviderIntegration
	}
	m, r, rows := o.meter, o.parent.recovery, o.parent.dispatchBounds
	if m == nil || m.owner != m || m.historyGroup != o || m.historyRecovery != nil || m.integration != nil || m.scan != nil ||
		m.scanBody != nil || m.firstAdmission != nil || m.firstOperation != nil || m.guardian != nil || m.continuing != nil || m.parent != nil ||
		!m.authority || m.upper != o.parent.bounds.future.work || m.maximumClocks != o.parent.bounds.future.clocks {
		return errProviderIntegration
	}
	m.mu.Lock()
	usable := !m.failed && !m.closed
	if usable {
		o.advanceStart, o.advanceClockStart = m.actual, m.clocks
	}
	m.mu.Unlock()
	if !usable {
		return errProviderIntegration
	}
	o.phase = "advancing"
	s.activeMeter.Store(m)
	succeeded := false
	defer func() {
		if !succeeded {
			o.failLocked()
		}
	}()
	// The active advancing role precedes even preflight's role checks.
	if !o.fits(m) || !m.preflightWork(rows.advance) || m.clocks > m.maximumClocks || rows.advanceClocks > m.maximumClocks-m.clocks ||
		!o.window || m.clocks < o.clockStart || m.clocks-o.clockStart > o.clockUpper || rows.advanceClocks > o.clockUpper-(m.clocks-o.clockStart) ||
		!o.identityLocked() {
		return errProviderIntegration
	}
	checkpoint := r.confirmed
	names, ok := o.prefixHistoryCatalogLocked(checkpoint)
	if !ok || !o.exactHistoryCatalogLocked(names, "") || ctx.Err() != nil {
		return errProviderIntegration
	}
	begin, err := s.samplePrimitive()
	if err != nil || !o.clockFits(begin) {
		return errProviderIntegration
	}
	a, err := o.replayHealthyPrefixLocked(ctx, checkpoint, s.frontierFiles.pinRaw)
	if err != nil {
		return errProviderIntegration
	}
	b, err := o.replayHealthyPrefixLocked(ctx, checkpoint, s.frontierFiles.pinRaw)
	if err != nil || a == b || a.ledger == b.ledger || a.pin != checkpoint.pin || b.pin != checkpoint.pin ||
		a.retention != o.retention || b.retention != o.retention || a.inventory != o.inventory || b.inventory != o.inventory ||
		a.ledger.charged != o.coordinator.charged || b.ledger.charged != o.custodian.charged ||
		!providerReplayTextEqual(a.ledger.head, b.ledger.head, m) || !o.coordinator.validObserved(m) || !o.custodian.validObserved(m) ||
		!m.event(providerCopyEvent, providerHistoryRecoveryArchiveSlotBytes) {
		return errProviderIntegration
	}
	// Prepare every comparison/copy and final catalog/HEAD/root check before
	// the original end sample. No fallible operation follows publication.
	names, ok = o.prefixHistoryCatalogLocked(checkpoint)
	if !ok || !o.exactHistoryCatalogLocked(names, "") || !s.frontierFiles.readExact("HEAD", providerReplayBytes(checkpoint.head, m)) ||
		!o.identityLocked() || ctx.Err() != nil || !o.successfulSettlementShapeLocked() {
		return errProviderIntegration
	}
	// Capture the exact completed T/C/S originals BEFORE the final sample.
	// Earlier replay validated the whole chain, but the scalar S selector
	// alone does not bind every earlier claim. No chain is reselected below.
	retiredCheckpoints := [3]*providerHistoryRecoveryCheckpoint{checkpoint.admission.checkpoint, checkpoint.previous, checkpoint}
	retiredDispatch := checkpoint.memberArchive.dispatch
	if retiredCheckpoints[0] == nil || retiredCheckpoints[1] == nil ||
		retiredCheckpoints[0] == retiredCheckpoints[1] || retiredCheckpoints[0] == checkpoint || retiredCheckpoints[1] == checkpoint ||
		retiredCheckpoints[0].previous != o.stageBefore || retiredCheckpoints[1].previous != retiredCheckpoints[0] ||
		retiredCheckpoints[0].frame.phase != "admit" || retiredCheckpoints[1].frame.phase != "consume" || checkpoint.frame.phase != "settle" ||
		retiredDispatch == nil || retiredDispatch.owner != retiredDispatch || retiredDispatch != o.dispatch ||
		retiredDispatch != checkpoint.claim.dispatch || retiredDispatch.group != o || retiredDispatch.meter != m ||
		retiredDispatch.acknowledgment != retiredCheckpoints[1].ack || retiredDispatch.memberReplaySeal != nil ||
		retiredDispatch.replayReads != ([2]*providerHistoryReplayReads{}) || r.memberReplaySeal != nil {
		return errProviderIntegration
	}
	var retiredClaims [3]*providerHistoryWriteClaim
	for i, original := range retiredCheckpoints {
		claim := original.claim
		if original.owner != original || original.recovery != r || original.group != o || original.admission != checkpoint.admission ||
			original.frame.stage != checkpoint.frame.stage || claim == nil || claim.owner != claim || claim.group != o || claim.meter != m ||
			claim.checkpoint != original || !claim.used || !claim.renamed || claim.frame == nil ||
			claim.frame.stage != original.frame.stage || claim.frame.phase != original.frame.phase || claim.frame.after != original.pin ||
			original.ack == nil || original.ack.owner != o || original.ack.claim != claim || original.ack.pin != original.pin {
			return errProviderIntegration
		}
		retiredClaims[i] = claim
	}
	if retiredClaims[0] == retiredClaims[1] || retiredClaims[0] == retiredClaims[2] || retiredClaims[1] == retiredClaims[2] {
		return errProviderIntegration
	}
	finalWork := m.actual
	c := o.parent.clock
	c.mu.Lock()
	finalRead := c.read
	c.mu.Unlock()
	end, err := s.samplePrimitive()
	if err != nil || !o.clockFits(end) || !candidateLedgerClockAdvance(providerClockDiagnostic(begin), providerClockDiagnostic(end)) ||
		uint64(end.MonotonicMS-begin.MonotonicMS) > rows.advanceTailMS || m.failed || m.closed {
		return errProviderIntegration
	}
	for i, value := range finalWork {
		if i != 10 && m.actual[i] != value {
			return errProviderIntegration
		}
	}
	units, ok := providerUpperSum(finalWork[10], 1)
	c.mu.Lock()
	read := c.read
	c.mu.Unlock()
	if !ok || m.actual[10] != units || finalRead == ^uint64(0) || read != finalRead+1 || ctx.Err() != nil ||
		!o.successfulSettlementShapeLocked() || o.meter != m || o.parent.clock != c || !o.fitsWork(m, candidateWorkVector{}) {
		return errProviderIntegration
	}
	// Fixed field retirement and the existing scalar publication share the
	// earlier 4096-byte structural copy charge ONCE. Never mutate a pointed
	// ledger: the old S model can still alias the live owner here. Keep all
	// original files, checkpoints, ACKs and archived consumed/sent bits.
	for _, claim := range retiredClaims {
		claim.fresh = [8]providerHistoryStoredObject{}
		claim.beforeCatalog, claim.afterCatalog = [2]map[string]bool{}, [2]map[string]bool{}
		claim.coordinator = nil
	}
	retiredDispatch.coordinator, retiredDispatch.custodian = nil, nil
	// Only CURRENT selections are reset; every original archived fact stays.
	o.stageBefore, o.coordinator, o.custodian = checkpoint, a.ledger, b.ledger
	o.stageRetentionStart = checkpoint.retention
	o.admission, o.order, o.dispatch = nil, nil, nil
	r.order, r.consumption, r.originalDispatch = nil, nil, nil
	r.memberAttempt, r.memberCapture, r.memberOriginal = nil, nil, nil
	r.members = [providerHistoryGroupMembers]*providerHistoryRecoveryMemberComplete{}
	r.completed, r.consumed, r.sent = 0, 0, 0
	o.window, o.phase = false, "prepared"
	s.activeMeter.Store(nil)
	succeeded = true
	return nil
}
