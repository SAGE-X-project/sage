package registry011

import "context"

func (r *providerHistoryRecoveryOwner) finishCatalogCandidate(ctx context.Context) (*providerHistoryRecoveryCatalogResult, error) {
	if r == nil || r.store == nil {
		return nil, errProviderIntegration
	}
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if !r.activeLocked(r.meter) || r.catalogBefore == nil || !r.catalogBefore.selectedLocked(r) || r.catalogAfter != nil || r.catalogResult != nil ||
		r.groupModels[0] == nil || !r.groupModels[0].selectedLocked(r) || r.groupModels[1] == nil || !r.groupModels[1].selectedLocked(r) ||
		r.completedMemberModels[0] == nil || !r.completedMemberModels[0].selectedLocked(r) ||
		r.completedMemberModels[1] == nil || !r.completedMemberModels[1].selectedLocked(r) ||
		r.tailModels[0] == nil || !r.tailModels[0].selectedLocked(r) || r.tailModels[1] == nil || !r.tailModels[1].selectedLocked(r) {
		return nil, errProviderIntegration
	}
	accepted := false
	defer func() {
		if !accepted {
			r.stopLocked(true)
		}
	}()
	a, b := r.groupModels[0], r.groupModels[1]
	if !r.inputReadyLocked(ctx) || a == b || a.ledger == b.ledger || a.initial == b.initial ||
		a.pin != b.pin || a.physical != b.physical || a.retention != b.retention || a.tailState != b.tailState || a.tailKind != b.tailKind || a.tailIndex != b.tailIndex ||
		!providerReplayTextEqual(a.ledger.head, b.ledger.head, r.meter) || !providerReplayTextEqual(a.confirmedModel, b.confirmedModel, r.meter) ||
		!providerReplayTextEqual(a.head, b.head, r.meter) {
		return nil, errProviderIntegration
	}
	for pass := range 2 {
		if !r.tailDataEqualLocked(&r.tailModels[pass].data, &r.tailSeals[pass]) ||
			!r.tailDataEqualLocked(&r.tailModels[pass].data, &r.tailModels[0].data) || pass == 1 && r.tailModels[0] == r.tailModels[1] {
			return nil, errProviderIntegration
		}
		members := r.completedMemberModels[pass]
		if members.count != r.completedMemberModels[0].count || pass == 1 && (members == r.completedMemberModels[0] || members.group == r.completedMemberModels[0].group) {
			return nil, errProviderIntegration
		}
		for i := range members.count {
			if !lowerHash(members.results[i]) || !providerReplayTextEqual(members.results[i], r.members[i].result, r.meter) ||
				!providerReplayTextEqual(members.results[i], r.completedMemberModels[0].results[i], r.meter) {
				return nil, errProviderIntegration
			}
		}
		want, err := r.expectedCatalogLocked(ctx, pass)
		if err != nil || !want.matches(r.catalogBefore) {
			return nil, errProviderIntegration
		}
	}
	after, err := r.readCatalogLocked(ctx)
	if err != nil || !r.catalogsEqualLocked(r.catalogBefore, after) {
		return nil, errProviderIntegration
	}
	// Neither the second snapshot nor the result is published before the
	// final original HEAD/root observations have also succeeded.
	head, err := r.readSelectedInputLocked(ctx, providerHistoryRecoveryInputSelection{checkpoint: r.initial, kind: providerHistoryRecoveryHead})
	if err != nil || head == nil || head.state != providerHistoryRecoveryWholeInput ||
		!providerReplayEqual(head.raw, providerReplayPhaseBytes(a.head, r.meter), r.meter) ||
		!r.meter.event(providerCopyEvent, 1) || !r.finalCatalogRootsLocked(ctx, after) {
		return nil, errProviderIntegration
	}
	out := &providerHistoryRecoveryCatalogResult{recovery: r, before: r.catalogBefore, after: after, files: r.inputFiles, bytes: r.inputBytes,
		work: r.meter.actual, entries: uint64(len(after.entries)), names: after.names}
	out.owner = out
	r.catalogAfter, r.catalogResult, accepted = after, out, true
	return out, nil
}

func (r *providerHistoryRecoveryOwner) catalogsEqualLocked(a, b *providerHistoryRecoveryCatalog) bool {
	if a == nil || b == nil || a == b || a.recovery != r || b.recovery != r || a.owner != a || b.owner != b || !a.complete || !b.complete ||
		len(a.entries) != len(b.entries) || a.names != b.names {
		return false
	}
	for root := range 2 {
		if !r.meter.event(providerCompareEvent, 0) || !providerHistoryRecoverySameMetadata(a.roots[root], b.roots[root]) {
			return false
		}
	}
	for _, before := range a.entries {
		found := false
		for _, after := range b.entries {
			if !r.meter.event(providerCompareEvent, uint64(len(before.name)+len(after.name))) {
				return false
			}
			if before.evidence != after.evidence || before.name != after.name {
				continue
			}
			if !r.meter.event(providerCompareEvent, 0) || !providerHistoryRecoverySameMetadata(before.info, after.info) {
				return false
			}
			found = true
			break
		}
		if !found {
			return false
		}
	}
	return true
}

func (r *providerHistoryRecoveryOwner) finalCatalogRootsLocked(ctx context.Context, c *providerHistoryRecoveryCatalog) bool {
	for root, disk := range [2]*retentionDisk{r.custody, r.evidence} {
		if !r.inputReadyLocked(ctx) || !r.inputIdentityLocked(disk) || !r.meter.event(providerReadEvent, 0) {
			return false
		}
		info, err := disk.root.Stat(".")
		if err != nil || !r.meter.event(providerCompareEvent, 0) || !providerHistoryRecoverySameMetadata(c.roots[root], info) {
			return false
		}
	}
	return r.inputReadyLocked(ctx)
}
