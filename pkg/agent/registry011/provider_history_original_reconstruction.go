package registry011

import "context"

// Inspection has its own explicit caller context and the original disjoint
// reservation. It neither resumes the collector nor replaces a cancelled
// execution context. Only a genuinely failed, closed original group may
// enter the existing one-use inspection lane.
func (p *providerHistoryProgramOwner) inspectFailedCollectionCandidate(ctx context.Context) (*providerHistoryRecoveryCatalogResult, error) {
	if p == nil || p.store == nil || ctx == nil || ctx.Err() != nil {
		return nil, errProviderIntegration
	}
	s := p.store
	s.mu.Lock()
	r, group := p.recovery, p.group
	valid := p.owner == p && s.owner == s && s.history == p && r != nil && r.parent == p && r.store == s &&
		group != nil && group.parent == p && group.store == s && r.originalGroup == group
	s.mu.Unlock()
	if !valid || r.beginInspectionCandidate(ctx, group) != nil {
		return nil, errProviderIntegration
	}
	out, err := r.reconstructOriginalsCandidate(ctx)
	if err != nil {
		s.mu.Lock()
		r.stopLocked(true)
		s.mu.Unlock()
		return nil, errProviderIntegration
	}
	return out, nil
}

// Reconstruct both models from their own actual original reads between the
// two catalog observations. This owns no writer, network, execution or clock
// role and does not accept a caller's object resolver or producer caches.
// The selected read-only lane must already be active and separately funded.
// A catalog result is still private DATA, never SourceReady or a page claim.
func (r *providerHistoryRecoveryOwner) reconstructOriginalsCandidate(ctx context.Context) (*providerHistoryRecoveryCatalogResult, error) {
	if r == nil || r.store == nil || ctx == nil || ctx.Err() != nil {
		return nil, errProviderIntegration
	}
	if _, err := r.beginCatalogCandidate(ctx); err != nil {
		return nil, errProviderIntegration
	}
	complete := false
	defer func() {
		if !complete {
			r.store.mu.Lock()
			r.stopLocked(true)
			r.store.mu.Unlock()
		}
	}()
	for pass := range 2 {
		if r.reconstructOriginalPassCandidate(ctx, pass) != nil {
			return nil, errProviderIntegration
		}
	}
	out, err := r.finishCatalogCandidate(ctx)
	if err != nil || out == nil {
		return nil, errProviderIntegration
	}
	complete = true
	return out, nil
}

func (r *providerHistoryRecoveryOwner) reconstructOriginalPassCandidate(ctx context.Context, pass int) error {
	if r == nil || r.store == nil || ctx == nil || ctx.Err() != nil {
		return errProviderIntegration
	}
	r.store.mu.Lock()
	ready := r.activeLocked(r.meter) && r.catalogBefore != nil && r.catalogBefore.selectedLocked(r) && r.catalogAfter == nil && r.catalogResult == nil
	r.store.mu.Unlock()
	if !ready {
		return errProviderIntegration
	}
	walk, err := r.newInventoryWalkCandidate(ctx, pass)
	if err != nil || walk == nil || r.drainOriginalInventoryCandidate(ctx, walk) != nil {
		return errProviderIntegration
	}
	if _, err := walk.finishCandidate(ctx); err != nil {
		return errProviderIntegration
	}
	scan, err := r.newScanWalkCandidate(ctx, pass)
	if err != nil || scan == nil {
		return errProviderIntegration
	}
	for {
		r.store.mu.Lock()
		valid := r.activeLocked(r.meter) && scan.selectedLocked(r)
		done := valid && scan.complete
		r.store.mu.Unlock()
		if !valid {
			return errProviderIntegration
		}
		if done {
			break
		}
		if _, err := scan.nextRecordCandidate(ctx); err != nil {
			return errProviderIntegration
		}
	}
	if _, err := r.replayLegacyPrefixCandidate(ctx, pass); err != nil {
		return errProviderIntegration
	}
	initial, err := r.replayInitialModelCandidate(ctx, pass)
	if err != nil || initial == nil {
		return errProviderIntegration
	}
	r.store.mu.Lock()
	valid := r.activeLocked(r.meter) && initial.selectedLocked(r)
	collection := valid && r.parent.bounds.collection != nil
	r.store.mu.Unlock()
	if !valid {
		return errProviderIntegration
	}
	if collection {
		if _, err := initial.selectFirstFinalizedCandidate(ctx); err != nil {
			return errProviderIntegration
		}
	}
	if _, err := r.replayGroupModelCandidate(ctx, pass); err != nil {
		return errProviderIntegration
	}
	if _, err := r.replayCompletedMemberPayloadsCandidate(ctx, pass); err != nil {
		return errProviderIntegration
	}
	if _, err := r.replayTailPayloadsCandidate(ctx, pass); err != nil {
		return errProviderIntegration
	}
	return nil
}

// Each successful step consumes either one of four original inventories or
// one of their bounded objects. Snapshot only scalar navigation under the
// original store mutex. The closed child alone selects and reads each file.
func (r *providerHistoryRecoveryOwner) drainOriginalInventoryCandidate(ctx context.Context, walk *providerHistoryRecoveryInventoryWalk) error {
	if r == nil || r.store == nil || ctx == nil || ctx.Err() != nil {
		return errProviderIntegration
	}
	for {
		r.store.mu.Lock()
		valid := r.activeLocked(r.meter) && r.catalogBefore != nil && r.catalogBefore.selectedLocked(r) && r.catalogAfter == nil &&
			r.catalogResult == nil && walk.selectedLocked(r) && !walk.complete
		done := valid && walk.remaining == 0 && walk.current == nil
		inventory := valid && walk.current == nil
		r.store.mu.Unlock()
		if !valid {
			return errProviderIntegration
		}
		if done {
			return nil
		}
		if inventory {
			if _, err := walk.nextInventoryCandidate(ctx); err != nil {
				return errProviderIntegration
			}
		} else if _, err := walk.nextObjectCandidate(ctx); err != nil {
			return errProviderIntegration
		}
	}
}
