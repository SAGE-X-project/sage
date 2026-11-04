package registry011

import (
	"context"
	"crypto/tls"
)

// Private, verified collection DATA, not a Source/page/message authorization.
// The copied cursor is independent of both execution and readback models.
type providerHistoryFinalResult struct {
	owner      *providerHistoryFinalResult
	final      *providerHistoryFinalOwner
	checkpoint *providerHistoryRecoveryCheckpoint
	catalog    *providerHistoryRecoveryCatalogResult
	cursor     *providerHistoryCollectionCursor
	begin, end ClockSample
	work       candidateWorkVector
}

func (*providerHistoryFinalResult) UnmarshalJSON([]byte) error { return errProviderIntegration }

func (f *providerHistoryFinalOwner) beginCandidate(ctx context.Context, settled *providerHistorySettledResults) error {
	if f == nil || f.parent == nil || f.parent.store == nil || ctx == nil || ctx.Err() != nil {
		return errProviderIntegration
	}
	s := f.parent.store
	s.mu.Lock()
	defer s.mu.Unlock()
	r, o, g := f.reader, f.recorded, f.parent.group
	if r == nil || o == nil || !f.pristineLocked() ||
		g == nil || g.coordinator == nil || !o.unspentLocked() || !o.checkpointSelectionLocked(g) || !o.memberSelectionLocked(g) {
		return errProviderIntegration
	}
	done, valid := g.collectionSettlementStateLocked(settled, g.coordinator.stage)
	if !done || !valid || o.originalMeter == nil || o.originalMeter.closed || o.originalMeter.failed ||
		o.originalMeter.historyFinal != nil || r.custody != o.custody || r.evidence != o.evidence ||
		r.custodyRoot != o.custodyRoot || r.evidenceRoot != o.evidenceRoot || r.custodyLock != o.custodyLock || r.evidenceLock != o.evidenceLock {
		return errProviderIntegration
	}
	// These are the fixed original seals, not producer models or file paths.
	// Fresh decoder state/arrays stay on r and cannot enter the failure lane.
	r.originalGroup, r.originalMeter, r.group = g, o.originalMeter, g
	r.initial, r.confirmed, r.initialDescriptor = o.initial, o.confirmed, o.initialDescriptor
	r.order, r.consumption, r.originalDispatch = o.order, o.consumption, o.originalDispatch
	r.memberReplaySeal, r.members, r.consumed, r.sent, r.completed = o.memberReplaySeal, o.members, o.consumed, o.sent, o.completed
	f.executionWork, f.executionClocks = g.meter.actual, g.meter.clocks
	f.used, r.used, r.active = true, true, true
	g.meter.mu.Lock()
	g.meter.closed = true // Preserve its actual, clock count, hold and ACKs.
	g.meter.mu.Unlock()
	g.phase = "finalizing"
	s.activeMeter.Store(r.meter)
	accepted := false
	defer func() {
		if !accepted {
			f.stopLocked(true)
		}
	}()
	if !f.activeLocked(r.meter) || !r.meter.event(providerCopyEvent, 1) {
		return errProviderIntegration
	}
	begin, err := s.samplePrimitive()
	if err != nil || !g.clockFits(begin) || ctx.Err() != nil {
		return errProviderIntegration
	}
	f.begin, accepted = begin, true
	return nil
}

func (f *providerHistoryFinalOwner) verifyCandidate(ctx context.Context, settled *providerHistorySettledResults) (*providerHistoryFinalResult, error) {
	if f.beginCandidate(ctx, settled) != nil {
		return nil, errProviderIntegration
	}
	complete := false
	defer func() {
		if !complete {
			f.parent.store.mu.Lock()
			f.stopLocked(true)
			f.parent.store.mu.Unlock()
		}
	}()
	catalog, err := f.reader.reconstructOriginalsCandidate(ctx)
	if err != nil || catalog == nil {
		return nil, errProviderIntegration
	}
	out, err := f.finishCandidate(ctx, catalog)
	if err != nil || out == nil {
		return nil, errProviderIntegration
	}
	complete = true
	return out, nil
}

func (f *providerHistoryFinalOwner) finishCandidate(ctx context.Context, catalog *providerHistoryRecoveryCatalogResult) (*providerHistoryFinalResult, error) {
	if f == nil || f.parent == nil || f.parent.store == nil || f.reader == nil {
		return nil, errProviderIntegration
	}
	s := f.parent.store
	s.mu.Lock()
	defer s.mu.Unlock()
	r, m := f.reader, f.reader.meter
	if !f.activeLocked(m) || ctx == nil || ctx.Err() != nil || f.begin == (ClockSample{}) ||
		catalog == nil || !catalog.selectedLocked(r) || r.pending != nil || r.confirmed.failedS ||
		r.groupModels[0] == nil || r.groupModels[1] == nil {
		return nil, errProviderIntegration
	}
	left, right := r.groupModels[0], r.groupModels[1]
	if !left.selectedLocked(r) || !right.selectedLocked(r) || left == right || left.ledger == right.ledger ||
		left.physical != r.confirmed || right.physical != r.confirmed || left.pin != r.confirmed.pin || right.pin != r.confirmed.pin {
		return nil, errProviderIntegration
	}
	a, b := left.ledger.collection, right.ledger.collection
	if a == nil || b == nil || a == b || !a.valid() || !b.valid() || a.semantic == b.semantic ||
		a.phase != "done" || b.phase != "done" || !a.semantic.atEnd() || !b.semantic.atEnd() ||
		a.completed != r.confirmed.frame.stage || b.completed != r.confirmed.frame.stage ||
		a.selection != b.selection || a.selection.manifest != f.parent.collectionDigest || a.selection.program != f.parent.bounds.program {
		return nil, errProviderIntegration
	}
	root, other := a.commitmentObserved(m), b.commitmentObserved(m)
	if root == "" || other == "" || !providerReplayTextEqual(root, other, m) {
		return nil, errProviderIntegration
	}
	copy := a.copyObserved(m)
	if copy == nil || copy == a || copy == b || copy.semantic == a.semantic || copy.semantic == b.semantic || !m.event(providerCopyEvent, 1) {
		return nil, errProviderIntegration
	}
	out := &providerHistoryFinalResult{final: f, checkpoint: r.confirmed, catalog: catalog, cursor: copy, begin: f.begin}
	out.owner = out
	// The catalog witness was current above. Subsequent copies/observations
	// intentionally stale its work marker; it cannot become a final claim.
	if !r.finalCatalogRootsLocked(ctx, r.catalogAfter) || !f.activeLocked(m) || m.clocks != 1 || m.historyClockClaims != 1 {
		return nil, errProviderIntegration
	}
	before := m.actual
	c := f.parent.clock
	c.mu.Lock()
	ordinal := c.read
	c.mu.Unlock()
	end, err := s.samplePrimitive()
	if err != nil || !r.group.clockFits(end) || !candidateLedgerClockAdvance(providerClockDiagnostic(f.begin), providerClockDiagnostic(end)) ||
		uint64(end.MonotonicMS-f.begin.MonotonicMS) > f.bounds.elapsedMS || ctx.Err() != nil || !f.activeLocked(m) || m.clocks != 2 || m.historyClockClaims != 2 {
		return nil, errProviderIntegration
	}
	for i, value := range before {
		if i != 10 && m.actual[i] != value {
			return nil, errProviderIntegration
		}
	}
	units, ok := providerUpperSum(before[10], 1)
	c.mu.Lock()
	read := c.read
	c.mu.Unlock()
	if !ok || m.actual[10] != units || ordinal == ^uint64(0) || read != ordinal+1 {
		return nil, errProviderIntegration
	}
	out.end, out.work = end, m.actual
	r.group.phase = "verified"
	f.stopLocked(false)
	return out, nil
}

// A caller must select the final observer before any collection work. The
// ordinary production route accepts no caller-supplied connections; only
// existing package-owned normal TLS tests use the private second entry.
func (p *providerHistoryProgramOwner) runVerifiedCollectionCandidate(ctx context.Context) (*providerHistoryFinalResult, error) {
	return p.runVerifiedCollectionWithConnectionsCandidate(ctx, nil)
}

func (p *providerHistoryProgramOwner) runVerifiedCollectionWithConnectionsCandidate(ctx context.Context, owned []*tls.Conn) (*providerHistoryFinalResult, error) {
	if p == nil || p.store == nil || ctx == nil || ctx.Err() != nil {
		return nil, errProviderIntegration
	}
	p.store.mu.Lock()
	f := p.final
	ready := f != nil && f.pristineLocked() &&
		p.group == nil && p.startReadyLocked()
	p.store.mu.Unlock()
	if !ready {
		return nil, errProviderIntegration
	}
	settled, err := p.runCollectionWithConnectionsCandidate(ctx, owned)
	if err != nil {
		return nil, errProviderIntegration
	}
	out, err := f.verifyCandidate(ctx, settled)
	if err != nil {
		p.store.mu.Lock()
		p.group.failLocked()
		p.store.mu.Unlock()
		return nil, errProviderIntegration
	}
	return out, nil
}
