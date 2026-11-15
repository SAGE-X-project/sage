package registry011

import (
	"context"
	"os"
)

// The failed-inspection observer is a disjoint pre-H0 reservation, never a
// replacement for execution; it has no network, write, clock or ACK role.
// Successful final verification reuses this decoder state only on a separate
// pre-H0 owner/meter, with explicit sealed-provenance and clock-role checks.
// A reservation alone yields no reconstruction or protected result.
type providerHistoryRecoveryOwner struct {
	custody, evidence            *retentionDisk
	custodyRoot, evidenceRoot    *os.Root
	custodyLock, evidenceLock    *os.File
	owner                        *providerHistoryRecoveryOwner
	parent                       *providerHistoryProgramOwner
	final                        *providerHistoryFinalOwner
	store                        *providerCustodyStore
	config                       providerCustodyConfig
	bounds                       providerHistoryRecoveryBounds
	meter                        *providerWorkMeter
	group                        *providerHistoryGroupOwner
	originalGroup                *providerHistoryGroupOwner
	originalMeter                *providerWorkMeter
	initial, confirmed, pending  *providerHistoryRecoveryCheckpoint
	initialDescriptor            *providerHistoryRecoveryInitialDescriptor
	pendingAck                   *providerHistoryGroupAcknowledgment
	order                        *providerHistoryGroupOrder
	consumption                  *providerHistoryRecoveryCheckpoint
	originalDispatch             *providerHistoryGroupDispatch
	memberReplaySeal             *providerHistoryMemberReplaySeal
	memberAttempt                *providerHistoryRecoveryMemberAttempt
	memberCapture                *providerHistoryRecoveryMemberCapture
	memberOriginal               *providerHistoryMemberOriginal
	members                      [providerHistoryGroupMembers]*providerHistoryRecoveryMemberComplete
	consumed, sent               uint32
	completed                    int
	inputFiles, inputBytes       uint64
	inventoryWalks               [2]*providerHistoryRecoveryInventoryWalk
	scanWalks                    [2]*providerHistoryRecoveryScanWalk
	legacyPrefixes               [2]*providerHistoryRecoveryLegacyPrefix
	initialModels                [2]*providerHistoryRecoveryInitialModel
	groupModels                  [2]*providerHistoryRecoveryGroupModel
	completedMemberModels        [2]*providerHistoryRecoveryCompletedMembers
	tailModels                   [2]*providerHistoryRecoveryTailPayloads
	tailSeals                    [2]*providerHistoryRecoveryTailData
	catalogBefore                *providerHistoryRecoveryCatalog
	catalogAfter                 *providerHistoryRecoveryCatalog
	catalogResult                *providerHistoryRecoveryCatalogResult
	used, active, closed, failed bool
}

func (*providerHistoryRecoveryOwner) UnmarshalJSON([]byte) error { return errProviderIntegration }

// Caller holds the original clock mutex; this performs no native operation.
func newProviderHistoryRecoveryBeforeOpen(p *providerHistoryProgramOwner) *providerHistoryRecoveryOwner {
	if p == nil || p.owner != p || p.store == nil || p.store.owner != p.store || p.recovery != nil ||
		p.store.custody != nil || p.store.evidence != nil || p.original != (ClockSample{}) || p.store.setupOrigin != (ClockSample{}) ||
		!p.bounds.recovery.valid(p.config.bounds, p.bounds.catalogEntries) {
		return nil
	}
	r := &providerHistoryRecoveryOwner{parent: p, store: p.store, config: p.config, bounds: p.bounds.recovery}
	r.owner = r
	r.meter = newProviderWorkMeter(nil, providerOperationReservation{work: r.bounds.work, execution: r.bounds.work})
	r.meter.historyRecovery = r
	return r
}

// The original opener calls this once after acquiring both original roots.
// It accepts no replacement path or handles supplied by an inspection caller.
func (r *providerHistoryRecoveryOwner) bindOriginalRootsBeforeReadLocked() bool {
	if r == nil || !r.unspentLocked() || r.custody != nil || r.evidence != nil || r.store.custody == nil || r.store.evidence == nil ||
		r.store.custody.root == nil || r.store.evidence.root == nil || r.store.custody.lock == nil || r.store.evidence.lock == nil ||
		r.store.custody.directory != r.config.custodyDirectory || r.store.evidence.directory != r.config.evidenceDirectory {
		return false
	}
	r.custody, r.evidence = r.store.custody, r.store.evidence
	r.custodyRoot, r.evidenceRoot = r.custody.root, r.evidence.root
	r.custodyLock, r.evidenceLock = r.custody.lock, r.evidence.lock
	return true
}

func (r *providerHistoryRecoveryOwner) observerShapeLocked(m *providerWorkMeter) bool {
	if r != nil && r.final != nil {
		return r.final.reader == r && r.final.shapeLocked(m)
	}
	return r != nil && r.owner == r && r.parent != nil && r.parent.owner == r.parent && r.parent.recovery == r &&
		r.store != nil && r.store.owner == r.store && r.store.history == r.parent && r.parent.store == r.store &&
		r.store.config == r.config && r.parent.config == r.config && r.config.bounds.history != nil &&
		*r.config.bounds.history == r.parent.bounds && r.parent.bounds.recovery == r.bounds && r.parent.bounds.valid(r.config.bounds) &&
		r.parent.bounds.derivation.dispatch == r.parent.dispatchBounds && r.parent.bounds.derivation.admission == r.parent.collectionAdmission &&
		r.parent.collectionMatchesLocked() && r.store.guardian == nil &&
		r.config.bounds.scan != nil && *r.config.bounds.scan == r.parent.scan &&
		r.config.bounds.continuation != nil && *r.config.bounds.continuation == r.parent.continuation &&
		r.config.bounds.methods != nil && *r.config.bounds.methods == r.parent.methods &&
		r.config.bounds.tls != nil && *r.config.bounds.tls == r.parent.tls &&
		r.config.bounds.firstAdmission != nil && *r.config.bounds.firstAdmission == r.parent.first &&
		r.config.policy != nil && *r.config.policy == r.parent.policy && r.store.firstLimits != nil && *r.store.firstLimits == r.parent.limits &&
		(r.parent.firstOwner == nil || r.store.firstOperation == r.parent.firstOwner && r.parent.firstAdmission != nil && r.parent.firstMeter != nil &&
			r.store.firstAdmissionOwner == r.parent.firstAdmission && r.parent.firstOwner.first == r.parent.firstAdmission && r.parent.firstAdmission.meter == r.parent.firstMeter) &&
		(r.originalGroup == nil && r.originalMeter == nil && r.parent.group == nil ||
			r.originalGroup != nil && r.originalGroup == r.parent.group && r.originalGroup.meter == r.originalMeter) &&
		r.meter == m && m != nil && m.owner == m && m.historyRecovery == r && m.historyFinal == nil && m.historyGroup == nil && m.integration == nil && m.scan == nil &&
		m.scanBody == nil && m.firstAdmission == nil && m.firstOperation == nil && m.guardian == nil && m.continuing == nil && m.parent == nil &&
		!m.authority && m.upper == r.bounds.work && m.reservation == r.bounds.work && m.remaining == (candidateWorkVector{}) &&
		m.maximumClocks == 0 && m.clocks == 0 && m.historyClockClaims == 0 && m.ticket == "" && !m.claimed && m.end == (ClockSample{})
}

// This is scalar-only and may run under the inspection observer mutex.
func (r *providerHistoryRecoveryOwner) unspentLocked() bool {
	return r != nil && r.final == nil && !r.used && !r.active && !r.closed && !r.failed && r.group == nil && r.inputFiles == 0 && r.inputBytes == 0 &&
		r.inventoryWalks == ([2]*providerHistoryRecoveryInventoryWalk{}) &&
		r.scanWalks == ([2]*providerHistoryRecoveryScanWalk{}) &&
		r.legacyPrefixes == ([2]*providerHistoryRecoveryLegacyPrefix{}) &&
		r.initialModels == ([2]*providerHistoryRecoveryInitialModel{}) &&
		r.groupModels == ([2]*providerHistoryRecoveryGroupModel{}) &&
		r.completedMemberModels == ([2]*providerHistoryRecoveryCompletedMembers{}) &&
		r.tailModels == ([2]*providerHistoryRecoveryTailPayloads{}) && r.tailSeals == ([2]*providerHistoryRecoveryTailData{}) &&
		r.catalogBefore == nil && r.catalogAfter == nil && r.catalogResult == nil &&
		r.observerShapeLocked(r.meter) && !r.meter.failed && !r.meter.closed && r.meter.actual == (candidateWorkVector{})
}

func (r *providerHistoryRecoveryOwner) terminalOriginalLocked(g *providerHistoryGroupOwner) bool {
	if r == nil || r.parent == nil || r.store == nil || g == nil || g.owner != g || g.parent != r.parent || g.store != r.store ||
		r.parent.group != g || r.originalGroup != g || r.originalMeter != g.meter || !r.parent.failed || !g.failed || !r.store.failed || r.store.closed || r.store.stored != r.parent.held.retained ||
		r.store.activeMeter.Load() != nil || g.meter == nil || g.meter == r.meter || g.meter.owner != g.meter ||
		g.meter.historyGroup != g || g.meter.historyRecovery != nil || !g.meter.failed || !g.meter.closed || !g.meter.authority ||
		g.meter.integration != nil || g.meter.scan != nil || g.meter.scanBody != nil || g.meter.firstAdmission != nil ||
		g.meter.firstOperation != nil || g.meter.guardian != nil || g.meter.continuing != nil || g.meter.parent != nil ||
		g.meter.upper != r.parent.bounds.future.work || g.meter.maximumClocks != r.parent.bounds.future.clocks {
		return false
	}
	return r.custody != nil && r.evidence != nil && r.store.custody == r.custody && r.store.evidence == r.evidence &&
		r.custodyRoot != nil && r.evidenceRoot != nil && r.custodyLock != nil && r.evidenceLock != nil &&
		r.custody.root == r.custodyRoot && r.evidence.root == r.evidenceRoot && r.custody.lock == r.custodyLock && r.evidence.lock == r.evidenceLock &&
		r.custody.directory == r.config.custodyDirectory && r.evidence.directory == r.config.evidenceDirectory &&
		r.store.frontierFiles != nil && r.store.frontierFiles.pending == nil &&
		(g.dispatch == nil || g.dispatch.attempt == nil || g.dispatch.attempt.connection == nil) &&
		r.parent.clock != nil && r.store.clock == r.parent.clock && r.parent.start != nil && r.parent.start.owner == r.parent.start && r.parent.start.parent == r.parent && g.start == r.parent.start
}

func (r *providerHistoryRecoveryOwner) activeLocked(m *providerWorkMeter) bool {
	if r != nil && r.final != nil {
		return r.final.reader == r && r.final.activeLocked(m)
	}
	return r != nil && r.used && r.active && !r.closed && !r.failed && r.observerShapeLocked(m) && r.terminalOriginalLocked(r.group)
}

// Entry consumes the one original lane. It does not reopen a store, clear a
// flag, sample the revoked backend, inspect a file or acknowledge any S.
func (r *providerHistoryRecoveryOwner) beginInspectionCandidate(ctx context.Context, g *providerHistoryGroupOwner) error {
	if r == nil || r.store == nil || ctx == nil || ctx.Err() != nil {
		return errProviderIntegration
	}
	s := r.store
	s.mu.Lock()
	defer s.mu.Unlock()
	if !r.unspentLocked() || !r.terminalOriginalLocked(g) || !r.checkpointSelectionLocked(g) || !r.memberSelectionLocked(g) {
		return errProviderIntegration
	}
	c := r.parent.clock
	c.mu.Lock()
	original := c.owner == c && c.history == r.parent && c.historyRevoked && r.parent.clockSelectionMatches(c) &&
		c.read >= r.parent.initialRead && c.read <= r.parent.clockCeiling
	c.mu.Unlock()
	if !original || ctx.Err() != nil {
		return errProviderIntegration
	}
	r.group, r.used, r.active = g, true, true
	return nil
}

// Caller holds store.mu and must not hold meter.mu. Closing this lane never
// changes the old execution meter, native-clock binding, hold or order prefix.
func (r *providerHistoryRecoveryOwner) stopLocked(failed bool) {
	if r != nil && r.final != nil {
		if r.final.reader == r {
			r.final.stopLocked(failed)
		}
		return
	}
	if r == nil || r.owner != r || !r.used || !r.active || r.meter == nil || r.meter.owner != r.meter {
		return
	}
	r.active, r.closed = false, true
	r.meter.mu.Lock()
	r.failed = r.failed || failed || r.meter.failed
	r.meter.closed, r.meter.failed = true, r.failed
	r.meter.mu.Unlock()
}

// Closing the original store revokes even an unused inspection lane before
// unlocking either root. Reopening a directory cannot reconstruct this owner.
func (r *providerHistoryRecoveryOwner) revokeForCloseLocked() {
	if r == nil || r.owner != r || r.meter == nil || r.meter.owner != r.meter {
		return
	}
	r.active, r.closed, r.failed = false, true, true
	r.meter.mu.Lock()
	r.meter.closed, r.meter.failed = true, true
	r.meter.mu.Unlock()
}
