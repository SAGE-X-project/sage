package registry011

// A fresh DATA template derived from one replay's own pre-seal boundary.
// It contains no G credential and cannot acknowledge or authorize a write.
func providerGuardianSealBoundaryData(ledger *CandidateAdmissionLedger, indexes *providerLedgerIndexes, main providerFrontierHighwater) (providerGuardianSealBinding, bool) {
	if ledger == nil || !ledger.valid() || ledger.intent == nil || ledger.current == nil || ledger.continuation == nil ||
		indexes == nil || !indexes.valid() || indexes.ledger != ledger || ledger.lastTicket == nil {
		return providerGuardianSealBinding{}, false
	}
	c := ledger.continuation
	if !c.valid(ledger) || c.pending != nil || c.final || uint64(len(c.records)) != c.bounds.pageRecords ||
		c.page+1 >= c.bounds.maximumPages {
		return providerGuardianSealBinding{}, false
	}
	for _, n := range ledger.current.totals.bootstrap.outstanding {
		if n != 0 {
			return providerGuardianSealBinding{}, false
		}
	}
	return providerGuardianSealBinding{main: main.commit, frontier: main.frontier, setup: main.setup, admission: ledger.intent.digest,
		budget: ledger.current.digest, pageIndex: c.page, records: uint64(len(c.records)), indexes: indexes.heads, clock: ledger.intent.clock}, true
}

// Before this replay prepares a boundary, add only the four immutable G
// originals (five allocated files) to its own common ledger, exactly once.
// Binding/index/main raw bytes remain with applyRetained's pending-boundary
// charge, and page+seal raw bytes remain with applyProviderBoundary.
func (r *providerCustodyReplay) guardianSealBefore(s *providerCustodyStore) (providerGuardianSealBinding, error) {
	var empty providerGuardianSealBinding
	if r == nil || r.guardian == nil || r.guardian.failed || !r.guardian.terminal || !r.guardian.consumed ||
		r.intent == nil || r.ledger == nil || r.ledger.intent != r.intent || r.ledger.lastTicket == nil || r.guardian.ticket != r.ledger.lastTicket.digest || s == nil ||
		s.config.bounds.guardian == nil || r.frontier == nil || r.frontier.body != r || r.frontier.current.commit != s.tip ||
		r.frontier.current.sequence != s.sequence || r.meter == nil || r.meter.closed || r.meter.failed || s.activeMeter.Load() != r.meter {
		return empty, errProviderIntegration
	}
	if len(s.newReferences) == 0 || len(s.newReferences) > s.config.bounds.maximumEntries {
		return empty, errProviderIntegration
	}
	var digest string
	for _, ref := range s.newReferences {
		if !r.meter.event(providerCompareEvent, uint64(len(ref.kind))) {
			return empty, errProviderIntegration
		}
		if ref.kind == "guardian-seal-binding" {
			if digest != "" || !lowerHash(ref.digest) {
				return empty, errProviderIntegration
			}
			digest = ref.digest
		}
	}
	raw, readErr := s.object("guardian-seal-binding", digest)
	x, codecErr := newProviderScanCodec(s.config, r.meter)
	if readErr != nil || codecErr != nil {
		return empty, errProviderIntegration
	}
	binding, err := x.parseGuardianSealBinding(raw)
	ordinal, ok := providerUpperSum(s.sequence, uint64(max(0, r.frontier.setupOrdinal)))
	if err != nil || !ok || binding.ordinal != r.guardian.head.ordinal+1 {
		return empty, errProviderIntegration
	}
	main, _, err := s.originalHighwaterObserved(ordinal, r.meter)
	value, valid := providerGuardianSealBoundaryData(r.ledger, r.frontier.indexes, main)
	if err != nil || !valid || main.commit != s.tip || main.sequence != s.sequence {
		return empty, errProviderIntegration
	}
	target, err := x.guardianSealTarget(value)
	if err != nil || target != binding.target {
		return empty, errProviderIntegration
	}
	data, err := providerGuardianClaimOriginals(s, r.meter, r.guardian.head, main, r.intent.digest, target, "ledger-page-seal")
	if err != nil {
		return empty, errProviderIntegration
	}
	value.page, value.seal = binding.page, binding.seal
	expected, err := providerGuardianSealBindingForBoundary(x, data, value)
	metadata, sumOK := providerUpperSum(r.guardian.metadata, data.metadata)
	if err != nil || expected.digest != digest || !providerReplayPhaseRawTextEqual(raw, expected.raw, r.meter) ||
		!sumOK || metadata > s.config.bounds.continuation.maximumMetadataBytes || r.ledger.addProviderRetention(r.ledger.current, 0, data.metadata) != nil {
		return empty, errProviderIntegration
	}
	// A subsequent mismatch leaves the surrounding retained replay failed.
	// No T, C, S, ticket tombstone or outstanding counter changes here.
	r.guardian.head, r.guardian.metadata = data.head, metadata
	return binding, nil
}
