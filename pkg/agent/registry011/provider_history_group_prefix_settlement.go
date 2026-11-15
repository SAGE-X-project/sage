package registry011

import "context"

// This parses an earlier SUCCESSFUL S using the fresh T/C ledger and actual
// archived spools. Old dispatch models, live permits and recovery observers
// never enter the reconstruction. Failure/pending S cannot be a healthy base.
func (o *providerHistoryGroupOwner) applyHealthySettlementLocked(ctx context.Context, p *providerHistoryHealthyPrefix, c *providerHistoryRecoveryCheckpoint, f *providerHistoryCustodyFrame, objects [8]providerHistoryStoredObject) error {
	if ctx == nil || ctx.Err() != nil {
		return errProviderIntegration
	}
	s, m, ledger := o.store, o.meter, p.ledger
	if c.memberArchive == nil || c.failedS || c.inventory.count != 2 || objects[0].kind != "results" || objects[1].kind != "settlement" ||
		ledger.ticket == nil || ledger.consumption == nil || c.previous == nil || c.previous.frame.phase != "consume" ||
		c.completed != ledger.ticket.table.count || c.completed <= 0 || c.completed > providerHistoryGroupMembers ||
		!providerHistoryGroupClockFits(c.admission.clock, f.clock) || !c.memberArchive.shapeLocked(o.parent.recovery, c) ||
		c.memberArchive.order.state != providerHistoryOrderClosed {
		return errProviderIntegration
	}
	policy, ok := providerReplayJSON(providerReplayBytes(o.parent.policy.raw, m), m).(map[string]any)
	provider, providerOK := policy["provider"].(map[string]any)
	roots := contentString(provider["tls_roots_artifact"])
	if !ok || !providerOK || !lowerHash(roots) {
		return errProviderIntegration
	}
	pack, results, err := parseProviderHistoryResultPackObserved(providerReplayBytes(objects[0].raw, m), objects[0].digest,
		ledger.ticket, ledger.consumption, c.completed, roots, o.parent.tls, s.config.bounds.maximumObjectBytes, m)
	if err != nil || !m.event(providerCopyEvent, uint64(c.completed)) {
		return errProviderIntegration
	}
	digests := make([]string, c.completed)
	for ordinal := range c.completed {
		if ctx.Err() != nil {
			return errProviderIntegration
		}
		sealed := o.parent.recovery.settledMemberLocked(c, ordinal)
		if sealed == nil || sealed.capture.intent.consumption != c.previous ||
			!providerHistoryGroupRequestEqualObserved(&sealed.capture.intent.member.request, &ledger.ticket.table.members[ordinal].request, m) ||
			sealed.capture.intent.member.class != ledger.ticket.table.members[ordinal].class ||
			sealed.capture.intent.member.capture != ledger.ticket.table.members[ordinal].capture {
			return errProviderIntegration
		}
		capture := sealed.capture
		if capture.responseBytes == 0 || capture.responseBytes > uint64(sealed.capture.intent.member.capture.WireBytes) || capture.tlsBytes == 0 || capture.tlsBytes > 64<<10 {
			return errProviderIntegration
		}
		for i, input := range [2]struct {
			kind, digest string
			bytes        uint64
		}{{"response", capture.response, capture.responseBytes}, {"tls", capture.tls, capture.tlsBytes}} {
			name := o.archivedMemberNameLocked(c, ordinal, input.kind)
			if name == "" || input.bytes == 0 || input.bytes > uint64(^uint64(0)>>1) {
				return errProviderIntegration
			}
			raw, readErr := s.read(s.evidence, name, int64(input.bytes))
			kind, text, found := pack.original(2*ordinal + i)
			if readErr != nil || ctx.Err() != nil || !found || kind != input.kind || uint64(len(raw)) != input.bytes ||
				!providerReplayTextEqual(providerReplayDigest("", raw, m), input.digest, m) ||
				!providerReplayEqual(raw, providerReplayBytes(text, m), m) {
				return errProviderIntegration
			}
		}
		if !providerReplayTextEqual(results[ordinal].digest, sealed.result, m) {
			return errProviderIntegration
		}
		raw, a := providerUpperSum(capture.responseBytes, capture.tlsBytes)
		overhead, b := providerUpperProduct(s.config.bounds.fileOverheadBytes, 2)
		physical, d := providerUpperSum(raw, overhead)
		charge := providerHistoryStorageCharge{physical: physical, raw: raw, payload: capture.responseBytes, metadata: capture.tlsBytes, objects: 2, files: 2}
		retained, e := p.retention.add(charge, o.parent.bounds.future)
		charged, g := p.retention.add(charge, ledger.charged)
		if !a || !b || !d || !e || !g || retained != charged {
			return errProviderIntegration
		}
		p.retention, digests[ordinal] = retained, results[ordinal].digest
	}
	settlement, err := parseProviderHistoryGroupSettlementObserved(providerReplayBytes(objects[1].raw, m), ledger.ticket,
		ledger.consumption, ledger.head, p.inventory, "complete", "", digests, m)
	if err != nil || !providerReplayTextEqual(settlement.digest, f.operation, m) || ledger.settleOriginalsObserved(ctx, settlement, results[:c.completed], s.config.bounds.methods, m) != nil || !ledger.validObserved(m) || ctx.Err() != nil {
		return errProviderIntegration
	}
	return nil
}

// The allowed catalog is independently derived from the immutable first
// bridge plus every original acknowledged checkpoint and archived spool.
// Cached claim catalogs and enumerated disk names cannot extend this set.
func (o *providerHistoryGroupOwner) prefixHistoryCatalogLocked(cutoff *providerHistoryRecoveryCheckpoint) ([2]map[string]bool, bool) {
	if o == nil || !o.activeLocked(o.meter) || cutoff == nil ||
		cutoff != o.stageBefore && (o.phase != "advancing" || cutoff != o.parent.recovery.confirmed) {
		return [2]map[string]bool{}, false
	}
	names, ok := o.initialHistoryCatalogLocked()
	if !ok {
		return [2]map[string]bool{}, false
	}
	var count uint64
	for c := cutoff; c != o.parent.recovery.initial; c = c.previous {
		if count >= o.parent.bounds.records || !o.healthyCheckpointShapeLocked(c) || c.failedS || !o.meter.event(providerCompareEvent, 1) {
			return [2]map[string]bool{}, false
		}
		add := func(root int, name string) bool {
			if name == "" || len(name) > 255 || names[root][name] || uint64(len(names[root])) >= o.parent.bounds.catalogEntries {
				return false
			}
			names[root][name] = true
			return true
		}
		for i := range c.inventory.count {
			if !add(1, providerHistoryStoredName(c.inventory.entries[i], o.meter)) {
				return [2]map[string]bool{}, false
			}
		}
		if !add(1, providerHistoryInventoryName(c.inventory.digest, o.meter)) || !add(1, o.store.name("history-pending", c.pin.sequence)) ||
			!add(0, o.store.name("intent", c.pin.sequence)) || !add(0, o.store.name("commit", c.pin.sequence)) ||
			!add(0, o.store.name("frontier", c.pin.sequence)) || !add(0, o.store.name("highwater", c.pin.ordinal)) {
			return [2]map[string]bool{}, false
		}
		if c.frame.phase == "settle" {
			for ordinal := range c.completed {
				for _, kind := range [2]string{"response", "tls"} {
					if !add(1, o.archivedMemberNameLocked(c, ordinal, kind)) {
						return [2]map[string]bool{}, false
					}
				}
			}
		}
		count++
	}
	return names, true
}
