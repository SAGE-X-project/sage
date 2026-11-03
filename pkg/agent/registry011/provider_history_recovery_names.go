package registry011

import "context"

type providerHistoryRecoveryExpectedName struct {
	evidence, optional bool
	name               string
}

type providerHistoryRecoveryExpectedCatalog struct {
	recovery *providerHistoryRecoveryOwner
	names    []providerHistoryRecoveryExpectedName
	bytes    uint64
}

func (w *providerHistoryRecoveryExpectedCatalog) add(evidence bool, name string, optional bool) bool {
	r := w.recovery
	if name == "" || len(name) > 255 || len(w.names) == cap(w.names) ||
		!candidateCapacity(0, w.bytes, uint64(len(name)), r.parent.bounds.catalogNameBytes) {
		return false
	}
	for _, prior := range w.names {
		if !r.meter.event(providerCompareEvent, uint64(len(prior.name)+len(name))) || prior.evidence == evidence && prior.name == name {
			return false
		}
	}
	copy := providerHistoryRecoveryCloneObserved(name, r.meter)
	if copy == "" || !r.meter.event(providerCopyEvent, 1) {
		return false
	}
	w.names = append(w.names, providerHistoryRecoveryExpectedName{evidence: evidence, optional: optional, name: copy})
	w.bytes += uint64(len(name))
	return true
}

func (r *providerHistoryRecoveryOwner) catalogOrdinalNameLocked(prefix string, ordinal uint64) string {
	if !r.meter.event(providerCopyEvent, uint64(len(prefix)+21)) {
		return ""
	}
	return providerCustodyName(prefix, ordinal)
}

// Only actual reconstructed frames/receipt originals and sealed original
// checkpoints supply names. Enumerated names and producer catalog caches do
// not enter this derivation, including for optional unfinished write slots.
func (r *providerHistoryRecoveryOwner) expectedCatalogLocked(ctx context.Context, pass int) (*providerHistoryRecoveryExpectedCatalog, error) {
	if pass < 0 || pass >= 2 || r.initialModels[pass] == nil || !r.initialModels[pass].selectedLocked(r) ||
		r.groupModels[pass] == nil || !r.groupModels[pass].selectedLocked(r) || !r.inputReadyLocked(ctx) ||
		!r.catalogSlotsLocked() || !r.meter.event(providerCopyEvent, r.parent.bounds.catalogEntries) {
		return nil, errProviderIntegration
	}
	w := &providerHistoryRecoveryExpectedCatalog{recovery: r, names: make([]providerHistoryRecoveryExpectedName, 0, int(r.parent.bounds.catalogEntries))}
	if !w.addInitialNames(r.legacyPrefixes[pass]) {
		return nil, errProviderIntegration
	}
	for c, count := r.confirmed, uint64(0); c != r.initial; c, count = c.previous, count+1 {
		if c == nil || count >= r.parent.bounds.records || !w.addCheckpoint(c, false) {
			return nil, errProviderIntegration
		}
		// The current prefix is added below. Earlier settled stages derive
		// their own names from their original archive, never today's C.
		if c.frame.phase == "settle" && c.previous != r.consumption {
			if !r.checkpointMemberArchiveShapeLocked(c) {
				return nil, errProviderIntegration
			}
			for ordinal := range c.completed {
				if !w.addMemberAt(c, ordinal, false) {
					return nil, errProviderIntegration
				}
			}
		}
	}
	if r.pending != nil && !w.addCheckpoint(r.pending, true) {
		return nil, errProviderIntegration
	}
	for ordinal := range r.completed {
		if !w.addMember(ordinal, false) {
			return nil, errProviderIntegration
		}
	}
	if r.memberAttempt != nil && !w.addMember(r.completed, true) {
		return nil, errProviderIntegration
	}
	if !r.inputReadyLocked(ctx) {
		return nil, errProviderIntegration
	}
	return w, nil
}

// This private builder formats initial description DATA into bounded names.
// The caller selects the original models before entry and rechecks readiness
// after all names are prepared. This helper supplies no reading or ACK role.
func (w *providerHistoryRecoveryExpectedCatalog) addInitialNames(p *providerHistoryRecoveryLegacyPrefix) bool {
	r, n := w.recovery, w.recovery.initialDescriptor
	for _, evidence := range []bool{false, true} {
		if !w.add(evidence, "AUTHORITY.LOCK", false) || !w.add(evidence, "ANCHOR", false) || !w.add(evidence, "SCAN-HEAD", false) {
			return false
		}
		for ordinal := uint64(0); ordinal <= n.scan.ordinal; ordinal++ {
			if !w.add(evidence, r.catalogOrdinalNameLocked("scan-highwater", ordinal), false) {
				return false
			}
		}
	}
	if !w.add(false, "HEAD", false) {
		return false
	}
	for ordinal := uint64(0); ordinal <= n.main.ordinal; ordinal++ {
		if !w.add(false, r.catalogOrdinalNameLocked("highwater", ordinal), false) {
			return false
		}
	}
	for ordinal := uint64(0); ordinal < n.setups; ordinal++ {
		if !w.add(false, r.catalogOrdinalNameLocked("setup-intent", ordinal), false) || !w.add(false, r.catalogOrdinalNameLocked("setup-commit", ordinal), false) ||
			!w.add(true, r.catalogOrdinalNameLocked("setup-claim", ordinal), false) {
			return false
		}
	}
	for ordinal := uint64(1); ordinal <= n.scan.ordinal; ordinal++ {
		if !w.add(false, r.catalogOrdinalNameLocked("scan-intent", ordinal), false) || !w.add(false, r.catalogOrdinalNameLocked("scan-commit", ordinal), false) ||
			!w.add(true, r.catalogOrdinalNameLocked("scan-claim", ordinal), false) {
			return false
		}
	}
	for i, frame := range p.source.frames {
		for _, prefix := range []string{"intent", "commit", "frontier"} {
			if !w.add(false, r.catalogOrdinalNameLocked(prefix, uint64(i+1)), false) {
				return false
			}
		}
		if !w.add(true, r.catalogOrdinalNameLocked("pending", uint64(i+1)), false) ||
			!w.add(true, providerReplayJoin([]string{"inventory-", frame.inventory.digest[2:]}, r.meter), false) {
			return false
		}
		for _, object := range frame.objects {
			if !w.add(true, providerReplayJoin([]string{"object-", object.kind, "-", object.digest[2:]}, r.meter), false) {
				return false
			}
		}
	}
	for _, record := range p.scan.records {
		for _, item := range []struct{ kind, domain, raw string }{{"chunk", providerScanChunkV2, record.chunk}, {"command", providerScanCommandV2, record.command}} {
			digest := providerReplayDigest(item.domain, providerReplayPhaseBytes(item.raw, r.meter), r.meter)
			name := providerScanReceiptName(item.kind, digest, r.meter)
			if !w.add(false, name, false) || !w.add(true, name, false) {
				return false
			}
		}
	}
	return true
}

func (w *providerHistoryRecoveryExpectedCatalog) addCheckpoint(c *providerHistoryRecoveryCheckpoint, optional bool) bool {
	r := w.recovery
	if !r.selectedCheckpointLocked(c) || c == r.initial || c.inventory.count <= 0 || c.inventory.count > len(c.inventory.entries) {
		return false
	}
	for _, kind := range []providerHistoryRecoveryFileKind{providerHistoryRecoveryHighwater, providerHistoryRecoveryIntent, providerHistoryRecoveryCommit,
		providerHistoryRecoveryFrontier, providerHistoryRecoveryPending, providerHistoryRecoveryInventory} {
		evidence := kind == providerHistoryRecoveryPending || kind == providerHistoryRecoveryInventory
		if !w.add(evidence, r.checkpointInputNameLocked(c, kind, 0), optional) {
			return false
		}
	}
	for index := range c.inventory.count {
		if !w.add(true, r.checkpointInputNameLocked(c, providerHistoryRecoveryObject, index), optional) {
			return false
		}
	}
	return !optional || w.add(false, r.catalogOrdinalNameLocked("HEAD.pending", c.pin.ordinal), true)
}

func (w *providerHistoryRecoveryExpectedCatalog) addMember(ordinal int, optional bool) bool {
	return w.addMemberAt(w.recovery.consumption, ordinal, optional)
}

func (w *providerHistoryRecoveryExpectedCatalog) addMemberAt(c *providerHistoryRecoveryCheckpoint, ordinal int, optional bool) bool {
	r := w.recovery
	if c == nil {
		return false
	}
	for _, kind := range []providerHistoryRecoveryFileKind{providerHistoryRecoveryMemberResponse, providerHistoryRecoveryMemberTLS} {
		if !w.add(true, r.namespaceInputNameLocked(c, kind, ordinal), optional) {
			return false
		}
	}
	return true
}

func (w *providerHistoryRecoveryExpectedCatalog) matches(c *providerHistoryRecoveryCatalog) bool {
	r := w.recovery
	if !c.selectedLocked(r) || len(c.entries) > len(w.names) {
		return false
	}
	var matched int
	for _, want := range w.names {
		found := false
		for _, entry := range c.entries {
			if !r.meter.event(providerCompareEvent, uint64(len(want.name)+len(entry.name))) {
				return false
			}
			if want.evidence == entry.evidence && want.name == entry.name {
				found = true
				matched++
				break
			}
		}
		if !found && !want.optional {
			return false
		}
	}
	return matched == len(c.entries)
}
