package registry011

// Reconstructed metadata only. Both independent first-admission passes derive
// this from originals, never by copying the live guardian's private fields.
type providerGuardianInitialData struct {
	origin, genesis, future string
	metadata                uint64
	upper                   providerScanUpper
	head                    providerGuardianHeadData
}

func providerGuardianOriginOriginal(s *providerCustodyStore, meter *providerWorkMeter, setup string, clock CandidateClockDiagnostic) (string, error) {
	if !s.valid() || s.config.bounds.guardian == nil || clock != providerClockDiagnostic(s.setupOrigin) || meter == nil || meter.owner != meter || meter.closed || meter.failed || s.activeMeter.Load() != meter {
		return "", errProviderIntegration
	}
	h0, original, err := s.originalHighwaterObserved(0, meter)
	x, codecErr := newProviderScanCodec(s.config, meter)
	if err != nil || codecErr != nil || h0.ordinal != 0 || h0.sequence != 0 || h0.commit != s.config.anchor || h0.setup != setup || h0.previous != "" || h0.frontier != "" || h0.fence != 1 {
		return "", errProviderIntegration
	}
	raw, err := x.guardianOrigin(x.hash("sage-provider-custody-highwater-v1", original), setup, clock)
	digest := x.hash(providerGuardianOriginDomain, raw)
	if err != nil || !lowerHash(digest) {
		return "", errProviderIntegration
	}
	retained, err := s.object("guardian-origin", digest)
	if err != nil || !x.equal(raw, retained) {
		return "", errProviderIntegration
	}
	// Every fresh E0 object must have a semantic reference. Reading only this
	// one derived digest makes an extra/different origin fail the common exact
	// inventory closure; duplicate equal digests already fail that inventory.
	return digest, nil
}

func providerGuardianExpectedInitial(x *providerScanCodec, e0, frontier, setup string, clock CandidateClockDiagnostic) (*providerGuardianInitialData, []byte, error) {
	if x == nil || x.config.bounds.guardian == nil {
		return nil, nil, errProviderIntegration
	}
	initial := providerFrontierHighwater{identity: x.config.identity, fence: 1, profile: x.config.profile, commit: x.config.anchor, setup: setup}
	h0Raw, err := initial.bytesObserved(x.config, x.meter)
	h0 := x.hash("sage-provider-custody-highwater-v1", h0Raw)
	originRaw, originErr := x.guardianOrigin(h0, setup, clock)
	origin := x.hash(providerGuardianOriginDomain, originRaw)
	deadline, _, clockOK := x.deadline(clock)
	if err != nil || originErr != nil || !clockOK || !lowerHash(origin) {
		return nil, nil, errProviderIntegration
	}
	value := providerGuardianHeadData{origin: origin, e0: e0, frontier: frontier, h0: h0, setup: setup, deadline: deadline, clock: clock}
	head, err := x.guardianHead(value)
	upper := x.config.bounds.guardian.lifetime
	upperRaw, upperErr := x.encode(upper.object())
	genesis := x.hash(providerGuardianHeadDomain, head)
	future := x.hash("sage-provider-guardian-future-upper-v1", upperRaw)
	metadata, ok := providerUpperProduct(2, uint64(len(head)))
	if err != nil || upperErr != nil || !ok || !lowerHash(genesis) || !lowerHash(future) {
		return nil, nil, errProviderIntegration
	}
	return &providerGuardianInitialData{origin: origin, genesis: genesis, future: future, metadata: metadata, upper: providerScanUpper{work: upper.work, retained: upper.retained, clocks: upper.clocks}, head: value}, head, nil
}

// Historical replay reads immutable guardian-highwater0. It must not replace
// that original with a later current HEAD. The live first constructor also
// compares selected current guardian HEAD0 under its separately funded owner.
func providerGuardianInitialOriginals(s *providerCustodyStore, meter *providerWorkMeter, e0, frontier, setup string, clock CandidateClockDiagnostic) (*providerGuardianInitialData, error) {
	origin, err := providerGuardianOriginOriginal(s, meter, setup, clock)
	if err != nil {
		return nil, errProviderIntegration
	}
	x, codecErr := newProviderScanCodec(s.config, meter)
	if codecErr != nil {
		return nil, errProviderIntegration
	}
	data, expected, err := providerGuardianExpectedInitial(x, e0, frontier, setup, clock)
	if err != nil || data.origin != origin {
		return nil, errProviderIntegration
	}
	raw, err := s.read(s.custody, s.name("guardian-highwater", 0), candidateLedgerMetadataLimit)
	if err != nil || !x.equal(raw, expected) || x.hash(providerGuardianHeadDomain, raw) != data.genesis {
		return nil, errProviderIntegration
	}
	return data, nil
}
