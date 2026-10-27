package registry011

// Only the initial live F can bind this original future guardian to the first
// execution owner. This precedes I and does not make G ready for later work.
func (g *providerGuardianOwner) bindFirstOwnerLocked(o *providerFirstOperationOwner) bool {
	if !g.selectedLocked() || !g.initialized || g.admitted || g.ready || g.active || g.failed || g.first != nil ||
		g.pin.ordinal != 0 || g.ordinal != 0 || g.origin == "" || g.genesis == "" || g.genesisMetadata == 0 ||
		o == nil || o.store != g.store || o.first == nil || o.first.state != providerFirstAdmissionStarted ||
		o.first.operation != o || g.store.firstOperation != o || o.state != providerFirstOperationPrepared ||
		g.main != o.first.mainPin || g.store.frontierFiles.pin != g.main || o.config != g.config || o.origin != g.original {
		return false
	}
	g.first = o
	return true
}

func (g *providerGuardianOwner) checkFirstOriginalsLocked(f *providerFirstAdmissionOwner, view *providerCustodyStore) bool {
	if !g.selectedLocked() || !g.initialized || g.admitted || g.active || g.ordinal != 0 || g.pin.ordinal != 0 ||
		f == nil || f.store != g.store || f.state != providerFirstAdmissionStarted || f.meter == nil ||
		g.store.activeMeter.Load() != f.meter || g.main != f.mainPin || view == nil || view == g.store {
		return false
	}
	data, err := providerGuardianInitialOriginals(view, f.meter, f.mainPin.commit, f.mainPin.frontier, f.mainPin.setup, providerClockDiagnostic(f.origin))
	return err == nil && data.origin == g.origin && data.genesis == g.genesis && data.metadata == g.genesisMetadata &&
		g.readExactLocked("GUARDIAN-HEAD", providerReplayBytes(g.pinRaw, f.meter)) &&
		g.readExactLocked(g.store.name("guardian-highwater", 0), providerReplayBytes(g.pinRaw, f.meter))
}

// Called while the already measured F still owns the transaction, after the
// exact E1 binding/seed was independently reconstructed and durably read back.
func (g *providerGuardianOwner) bindFirstIntegrationLocked(o *providerFirstOperationOwner) bool {
	if !g.selectedLocked() || !g.initialized || g.admitted || g.ready || g.active || g.first != o ||
		o == nil || o.store != g.store || o.first == nil || o.first.state != providerFirstAdmissionStarted ||
		o.integration == nil || !o.integration.valid() || o.indexes == nil || o.integration.ledger != o.indexes.ledger ||
		g.store.activeMeter.Load() != o.first.meter || g.store.sequence != 2 || g.store.frontierReplay == nil ||
		g.store.frontierReplay.indexes == o.indexes || g.store.frontierReplay.body == nil ||
		g.store.frontierReplay.body.ledger == nil || g.store.frontierReplay.body.ledger == o.integration.ledger ||
		g.store.frontierReplay.body.intent == nil || g.store.frontierReplay.body.intent.digest != o.integration.intent.digest {
		return false
	}
	g.integration, g.indexes, g.admitted = o.integration, o.indexes, true
	return true
}

// The caller just completed its original measured modelEqual under R and
// holds the same store.mu until R closes and first owner becomes DONE. This
// scalar-only handoff neither repeats that work for free nor copies maps.
func (g *providerGuardianOwner) handoffFirstLocked(o *providerFirstOperationOwner) bool {
	if !g.selectedLocked() || !g.initialized || !g.admitted || g.ready || g.active || g.first != o ||
		o == nil || o.state != providerFirstOperationSettlement || o.integration != g.integration || o.indexes != g.indexes ||
		o.remaining == nil || o.remaining.closed || o.remaining.failed || g.store.activeMeter.Load() != o.remaining ||
		!o.consumption || !o.settlement || o.execution == nil || !o.execution.closed || !o.execution.claimed || o.execution.failed ||
		g.store.sequence != 4 || g.store.last != nil || g.store.frontierReplay == nil || g.store.frontierReplay.body == nil ||
		g.store.frontierReplay.body.firstExchange == "" || g.integration.firstExchange != "" ||
		g.integration.root.digest != g.store.enrollment || g.integration.ledger.current.digest != g.store.frontierReplay.body.ledger.current.digest ||
		g.pin.ordinal != 0 || g.ordinal != 0 || g.claim != nil || g.meter != nil {
		return false
	}
	r := g.store.frontierReplay.body.guardian
	if r == nil || r.failed || !r.consumed || !r.terminal || r.head != g.pin || r.metadata != g.genesisMetadata || r.ticket != g.integration.ledger.lastTicket.digest {
		return false
	}
	prefix, ok := providerContinuingPrefixData(g)
	if !ok {
		return false
	}
	g.prefix = prefix
	g.main, g.ready = g.store.frontierFiles.pin, true
	return true
}
