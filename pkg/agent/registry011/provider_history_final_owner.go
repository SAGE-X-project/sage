package registry011

// A successful final reader is selected before H0. Its observer, read models
// and two original clock claims are disjoint from execution and failed
// inspection. Sharing the original-byte decoder never revives a failed role.
type providerHistoryFinalOwner struct {
	owner                *providerHistoryFinalOwner
	parent               *providerHistoryProgramOwner
	reader               *providerHistoryRecoveryOwner
	recorded             *providerHistoryRecoveryOwner
	bounds               providerHistoryFinalBounds
	upper                providerScanUpper
	begin                ClockSample
	executionWork        candidateWorkVector
	executionClocks      uint64
	used, closed, failed bool
}

func (*providerHistoryFinalOwner) UnmarshalJSON([]byte) error { return errProviderIntegration }

// Caller holds the original clock mutex. No sample, root, replacement
// execution observer or production derivation is constructed here.
func newProviderHistoryFinalBeforeOpen(p *providerHistoryProgramOwner) *providerHistoryFinalOwner {
	if p == nil || p.owner != p || p.store == nil || p.store.owner != p.store || p.final != nil || p.recovery == nil ||
		p.store.custody != nil || p.store.evidence != nil || p.original != (ClockSample{}) ||
		!p.bounds.final.valid(p.config.bounds, p.bounds.catalogEntries, p.bounds.durationMS) {
		return nil
	}
	upper, ok := p.bounds.final.upper()
	if !ok || upper.clocks != 2 {
		return nil
	}
	f := &providerHistoryFinalOwner{parent: p, recorded: p.recovery, bounds: p.bounds.final, upper: upper}
	f.owner = f
	r := &providerHistoryRecoveryOwner{parent: p, store: p.store, config: p.config, bounds: f.bounds.read, final: f}
	r.owner = r
	r.meter = newProviderWorkMeter(nil, providerOperationReservation{work: upper.work, execution: upper.work})
	r.meter.historyFinal, r.meter.maximumClocks = f, 2
	f.reader = r
	return f
}

// Bind only handles acquired by the original opener, before its first root
// read. Neither the final caller nor the shared decoder can supply a path.
func (f *providerHistoryFinalOwner) bindOriginalRootsBeforeReadLocked() bool {
	if f == nil || f.owner != f || f.used || f.closed || f.failed || f.reader == nil || f.recorded == nil ||
		f.parent == nil || f.parent.final != f || f.parent.recovery != f.recorded ||
		f.reader.custody != nil || f.reader.evidence != nil || f.recorded.custodyRoot == nil || f.recorded.evidenceRoot == nil {
		return false
	}
	r, original := f.reader, f.recorded
	r.custody, r.evidence = original.custody, original.evidence
	r.custodyRoot, r.evidenceRoot = original.custodyRoot, original.evidenceRoot
	r.custodyLock, r.evidenceLock = original.custodyLock, original.evidenceLock
	return true
}

func (f *providerHistoryFinalOwner) shapeLocked(m *providerWorkMeter) bool {
	if f == nil || f.owner != f || f.parent == nil || f.parent.final != f || f.parent.recovery != f.recorded ||
		f.recorded == nil || f.recorded.final != nil || f.reader == nil || f.reader == f.recorded ||
		f.bounds != f.parent.bounds.final || !f.bounds.valid(f.parent.config.bounds, f.parent.bounds.catalogEntries, f.parent.bounds.durationMS) ||
		!f.parent.matchesLocked() {
		return false
	}
	upper, ok := f.bounds.upper()
	r := f.reader
	return ok && f.upper == upper && r.owner == r && r.final == f && r.parent == f.parent && r.store == f.parent.store &&
		r.config == f.parent.config && r.bounds == f.bounds.read && r.meter == m && m != nil && m.owner == m && m.historyFinal == f &&
		m.historyRecovery == nil && m.historyGroup == nil && m.historyInputGroup == nil && m.replayReads == nil &&
		m.integration == nil && m.scan == nil && m.scanBody == nil && m.firstAdmission == nil && m.firstOperation == nil &&
		m.guardian == nil && m.continuing == nil && m.parent == nil && !m.authority &&
		m.upper == f.upper.work && m.reservation == f.upper.work && m.remaining == (candidateWorkVector{}) &&
		m.maximumClocks == 2 && m.clocks <= 2 && m.historyClockClaims <= m.clocks && m.ticket == "" && !m.claimed && m.end == (ClockSample{})
}

func (f *providerHistoryFinalOwner) pristineLocked() bool {
	if f == nil || f.reader == nil || !f.shapeLocked(f.reader.meter) || f.used || f.closed || f.failed || f.begin != (ClockSample{}) ||
		f.executionWork != (candidateWorkVector{}) || f.executionClocks != 0 {
		return false
	}
	r, m := f.reader, f.reader.meter
	return !r.used && !r.active && !r.closed && !r.failed && r.group == nil && r.originalGroup == nil && r.originalMeter == nil &&
		r.initial == nil && r.confirmed == nil && r.pending == nil && r.pendingAck == nil && r.initialDescriptor == nil &&
		r.order == nil && r.consumption == nil && r.originalDispatch == nil && r.memberReplaySeal == nil &&
		r.memberAttempt == nil && r.memberCapture == nil && r.memberOriginal == nil && r.members == ([providerHistoryGroupMembers]*providerHistoryRecoveryMemberComplete{}) &&
		r.consumed == 0 && r.sent == 0 && r.completed == 0 && r.inputFiles == 0 && r.inputBytes == 0 &&
		r.inventoryWalks == ([2]*providerHistoryRecoveryInventoryWalk{}) && r.scanWalks == ([2]*providerHistoryRecoveryScanWalk{}) &&
		r.legacyPrefixes == ([2]*providerHistoryRecoveryLegacyPrefix{}) && r.initialModels == ([2]*providerHistoryRecoveryInitialModel{}) &&
		r.groupModels == ([2]*providerHistoryRecoveryGroupModel{}) && r.completedMemberModels == ([2]*providerHistoryRecoveryCompletedMembers{}) &&
		r.tailModels == ([2]*providerHistoryRecoveryTailPayloads{}) && r.tailSeals == ([2]*providerHistoryRecoveryTailData{}) &&
		r.catalogBefore == nil && r.catalogAfter == nil && r.catalogResult == nil &&
		!m.failed && !m.closed && m.actual == (candidateWorkVector{}) && m.clocks == 0 && m.historyClockClaims == 0
}

func (f *providerHistoryFinalOwner) executionShapeLocked() bool {
	if f == nil || f.parent == nil || f.reader == nil || f.reader.originalGroup == nil || f.reader.originalMeter == nil {
		return false
	}
	g, m := f.reader.originalGroup, f.reader.originalMeter
	return g.owner == g && g.parent == f.parent && g.store == f.parent.store && g.start == f.parent.start && m.owner == m &&
		g.meter == m && m.historyGroup == g && m.historyInputGroup == g && m.historyRecovery == nil && m.historyFinal == nil &&
		m.integration == nil && m.scan == nil && m.scanBody == nil && m.firstAdmission == nil && m.firstOperation == nil &&
		m.guardian == nil && m.continuing == nil && m.parent == nil && m.authority &&
		m.upper == f.parent.bounds.future.work && m.maximumClocks == f.parent.bounds.future.clocks &&
		m.actual == f.executionWork && m.clocks == f.executionClocks
}

// Only immutable original checkpoint/member provenance is shared. Every
// inventory, decoder, semantic model and catalog belongs to the fresh reader.
// Recheck this binding on every decoder primitive; do not cache permission.
func (f *providerHistoryFinalOwner) provenanceMatchesLocked() bool {
	if f == nil || f.owner != f || f.parent == nil || f.parent.final != f || f.recorded == nil || f.recorded.final != nil ||
		f.parent.recovery != f.recorded || f.reader == nil || f.reader.owner != f.reader || f.reader.final != f {
		return false
	}
	r, o := f.reader, f.recorded
	return r != o && r.parent == f.parent && r.store == f.parent.store && o.parent == f.parent && o.store == r.store &&
		r.config == o.config && r.custody == o.custody && r.evidence == o.evidence && r.custodyRoot == o.custodyRoot && r.evidenceRoot == o.evidenceRoot &&
		r.custodyLock == o.custodyLock && r.evidenceLock == o.evidenceLock && r.custodyRoot != nil && r.evidenceRoot != nil &&
		r.custody != nil && r.evidence != nil && r.custodyLock != nil && r.evidenceLock != nil &&
		r.store.custody == r.custody && r.store.evidence == r.evidence && r.custody.root == r.custodyRoot && r.evidence.root == r.evidenceRoot &&
		r.custody.lock == r.custodyLock && r.evidence.lock == r.evidenceLock &&
		r.originalGroup == o.originalGroup && r.originalGroup == f.parent.group && r.originalMeter == o.originalMeter &&
		r.originalGroup != nil && r.originalMeter != nil && r.originalMeter.owner == r.originalMeter &&
		r.originalGroup.meter == r.originalMeter && r.group == r.originalGroup &&
		r.initial == o.initial && r.confirmed == o.confirmed && r.pending == nil && o.pending == nil && r.pendingAck == nil && o.pendingAck == nil &&
		r.initialDescriptor == o.initialDescriptor && r.order == o.order && r.consumption == o.consumption && r.originalDispatch == o.originalDispatch &&
		r.memberReplaySeal == o.memberReplaySeal && r.memberAttempt == nil && o.memberAttempt == nil && r.memberCapture == nil && o.memberCapture == nil &&
		r.memberOriginal == nil && o.memberOriginal == nil && r.members == o.members && r.consumed == o.consumed && r.sent == o.sent && r.completed == o.completed
}

// This selector authorizes only original provenance comparisons, not reads,
// work, clock sampling or recovery entry under the recorded execution owner.
func (r *providerHistoryRecoveryOwner) recordedOwnerLocked() *providerHistoryRecoveryOwner {
	if r == nil {
		return nil
	}
	if r.final == nil {
		return r
	}
	if r.final.reader != r || !r.final.provenanceMatchesLocked() {
		return nil
	}
	return r.final.recorded
}

func (f *providerHistoryFinalOwner) activeLocked(m *providerWorkMeter) bool {
	if !f.shapeLocked(m) || !f.used || f.closed || f.failed || !f.provenanceMatchesLocked() || !f.executionShapeLocked() {
		return false
	}
	r, s, g := f.reader, f.parent.store, f.parent.group
	return r.used && r.active && !r.closed && !r.failed && r.group == g && !g.failed && g.phase == "finalizing" &&
		s.activeMeter.Load() == m && !m.closed && !m.failed && r.originalMeter != m && r.originalMeter.closed && !r.originalMeter.failed &&
		r.confirmed != nil && r.confirmed.pin == g.pin && r.confirmed.ack == g.last &&
		s.frontierFiles != nil && s.frontierFiles.pending == nil && s.frontierFiles.historyPending == nil &&
		s.sequence == g.pin.sequence && s.tip == g.pin.commit && s.setupTip == g.pin.setup
}

// The clock unit cannot be borrowed by a read primitive. This scalar check is
// performed under the same meter mutex after hasOwner, including preflights.
func (f *providerHistoryFinalOwner) fitsReadWindowLocked(m *providerWorkMeter, work candidateWorkVector) bool {
	for i, n := range work {
		next, ok := providerUpperSum(m.actual[i], n)
		if !ok {
			return false
		}
		if i == 10 {
			if next < m.clocks {
				return false
			}
			next -= m.clocks
		}
		if next > f.bounds.read.work[i] {
			return false
		}
	}
	return true
}

// The store lock is held and the meter lock is not. Failed final verification
// quarantines the actual original collection; no failed flag is cleared.
func (f *providerHistoryFinalOwner) stopLocked(failed bool) {
	if f == nil || f.owner != f || f.reader == nil || f.reader.meter == nil || f.reader.meter.owner != f.reader.meter {
		return
	}
	r, m := f.reader, f.reader.meter
	m.mu.Lock()
	f.failed = f.failed || failed || m.failed
	f.closed, r.active, r.closed, r.failed, m.closed, m.failed = true, false, true, f.failed, true, f.failed
	m.mu.Unlock()
	if f.parent != nil && f.parent.store != nil && f.parent.store.activeMeter.Load() == m {
		f.parent.store.activeMeter.Store(nil)
	}
	if f.failed && f.used && f.parent != nil && f.parent.group != nil {
		f.parent.group.failLocked()
	}
}
