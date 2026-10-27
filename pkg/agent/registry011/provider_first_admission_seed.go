package registry011

// A historical replay view may read the exact same immutable roots/config as
// its active concrete owner. It never substitutes its provisional maps for
// that owner or installs a new meter, current pin or retaining principal.
func providerAdmissionOriginalOwner(s *providerCustodyStore, m *providerWorkMeter) *providerCustodyStore {
	if s == nil || m == nil || m.owner != m || m.closed || m.failed || s.activeMeter.Load() != m {
		return nil
	}
	var owner *providerCustodyStore
	if m.scan != nil && m.integration == nil {
		owner = m.scan.store
	} else if m.integration != nil && m.scan == nil {
		owner = m.integration.store
	}
	if !owner.valid() || owner.activeMeter.Load() != m || owner.custody != s.custody || owner.evidence != s.evidence || owner.clock != s.clock || owner.config != s.config {
		return nil
	}
	return owner
}

// The first typed admit must contain exactly one semantically referenced
// binding. E0 is the already independently replayed prefix boundary, not the
// later live HEAD or producer maps. Original scan controls/chunks supply the
// seed before the first ticket append. No ACK or dispatch follows this DATA.
func (r *providerCustodyReplay) admissionScanSeed(s *providerCustodyStore) (*providerAdmissionControlData, error) {
	if r == nil || s == nil || r.intent == nil || r.anchor == nil || r.ledger != nil || r.frontier == nil || r.frontier.body != r || r.frontier.owner != r.frontier || r.frontier.meter != r.meter || r.frontier.indexes == nil || s.config.bounds.firstAdmission == nil || s.sequence != 1 || len(s.objects) == 0 || len(s.objects) > s.config.bounds.maximumObjects {
		return nil, errProviderIntegration
	}
	owner := providerAdmissionOriginalOwner(s, r.meter)
	if owner == nil {
		return nil, errProviderIntegration
	}
	var digest string
	for key, ref := range s.objects {
		if !r.meter.event(providerCompareEvent, uint64(len(ref.kind))) {
			return nil, errProviderIntegration
		}
		if ref.kind == "scan-admission-binding" {
			want, ok := providerReplayRefKey(ref, r.meter)
			if digest != "" || !ok || key != want {
				return nil, errProviderIntegration
			}
			digest = ref.digest
		}
	}
	raw, readErr := s.object("scan-admission-binding", digest)
	x, codecErr := newProviderScanCodec(s.config, r.meter)
	if readErr != nil || codecErr != nil {
		return nil, errProviderIntegration
	}
	binding, err := x.parseAdmissionBinding(raw)
	f := r.frontier.current
	frontier, frontierErr := f.bytesObserved(r.meter)
	target := providerScanTarget{setup: f.setup, custody: s.tip, frontier: binding.frontier, objects: s.objectsRoot, enrollment: s.enrollment}
	targetRaw, targetErr := x.target(target)
	if err != nil || frontierErr != nil || targetErr != nil || f.sequence != 1 || f.commit != s.tip || f.enrollment != s.enrollment || f.enrollment != r.anchor.digest || f.objects != s.objectsRoot || f.admission != "" || f.budget != "" || f.checkpoint != "" || f.initial != providerClockDiagnostic(s.setupOrigin) || binding.e0 != s.tip || binding.setup != f.setup || binding.frontier != x.hash("sage-provider-custody-frontier-v1", frontier) || binding.target != x.hash("sage-provider-scan-target-v1", targetRaw) || binding.admission != r.intent.digest || binding.clock != r.intent.clock || binding.deadline != r.intent.deadline {
		return nil, errProviderIntegration
	}
	// The historical immutable high-water is selected by this verified E0
	// boundary. Reading HEAD here would incorrectly select a later prefix.
	h, _, highwaterErr := s.originalHighwaterObserved(1, r.meter)
	if highwaterErr != nil || h.ordinal != 1 || h.sequence != 1 || h.commit != binding.e0 || h.frontier != binding.frontier || h.setup != binding.setup || h.fence != r.intent.fence {
		return nil, errProviderIntegration
	}
	controls, err := replayProviderAdmissionControlData(owner, r.meter, binding)
	if err != nil {
		return nil, errProviderIntegration
	}
	receipts, err := replayProviderAdmissionReceipts(owner, r.meter, binding)
	if err != nil || receipts.records != h.ordinal+1 {
		return nil, errProviderIntegration
	}
	metadata, ok := providerUpperSum(controls.metadata, receipts.metadata)
	common, commonOK := providerUpperSum(metadata, r.frontier.preI.metadata, r.frontier.indexes.schemaBytes)
	if !ok || !commonOK || common > s.config.bounds.continuation.maximumMetadataBytes || common > s.config.bounds.continuation.pageMetadataBytes {
		return nil, errProviderIntegration
	}
	return &providerAdmissionControlData{controls.seed, metadata}, nil
}
