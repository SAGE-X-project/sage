package registry011

const historyPreparationInputFormat = "sage-registry-history-preparation-input-v1"

// Build only an unminted, detached-source proposal. No operating limit is
// defaulted, reduced or selected here, and no derivation or store is created.
// The original owned compiler must still freeze and compile this same config.
func buildHistoryPreparationConfig(in historyPreparationInput) (providerCustodyConfig, error) {
	var empty providerCustodyConfig
	if in.Format != historyPreparationInputFormat || !in.Capture.valid() {
		return empty, ErrConfiguration
	}
	policy, err := ParseProviderPolicyCandidate([]byte(in.Policy), in.MaximumPolicyBytes)
	if err != nil {
		return empty, err
	}
	return buildHistoryPreparationConfigParts(policy, in.Namespace, in.SourceID, in.Generation, in.Store, in.Capture, in.Collection, in.Bounds)
}

// Both DATA readers use these same semantic constructors and guards. Native
// preparation supplies temporary Go sections only for the existing numeric
// compiler. Those allocations still need separate resource funding; they are
// not the native plan's retained representation.
func buildHistoryPreparationConfigParts(
	policy *ProviderPolicyCandidate, namespace, sourceID, generation string,
	store historyPreparationStore, capture ProviderCaptureLimits,
	collection historyPreparationCollection, bounds historyPreparationBounds,
) (providerCustodyConfig, error) {
	var empty providerCustodyConfig
	if policy == nil || !capture.valid() {
		return empty, ErrConfiguration
	}
	candidate, err := policy.BindCandidate(namespace, sourceID, generation)
	if err != nil {
		return empty, err
	}
	identity, err := candidate.Identity()
	if err != nil {
		return empty, err
	}
	b := historyPreparationStoreBounds(bounds)
	if b.maximumEntries <= 0 {
		return empty, ErrConfiguration
	}
	// Preserve the caller's complete proposed read bounds. The existing source
	// constructor supplies an upper bound that the proposed work must cover.
	// This numeric check does not reserve resources or establish provenance.
	legacy := providerHistoryLegacyControlLimits{
		metadata: b.maximumMetadataPerCommit, objectBytes: b.maximumObjectBytes,
		firstObjects: collection.FirstRead.Objects, firstRaw: collection.FirstRead.Raw,
		entries: uint64(b.maximumEntries), attempts: b.scan.maximumAttempts, chunks: b.scan.maximumChunks,
	}
	firstRead, ok := providerHistoryFirstPReadUpper(legacy, capture)
	if !ok || firstRead.objects != collection.FirstRead.Objects || firstRead.raw != collection.FirstRead.Raw {
		return empty, ErrConfiguration
	}
	proposedWork := collection.FirstRead.Work.vector()
	if !candidateFits(candidateBudgetTotals{}, firstRead.work, proposedWork) {
		return empty, ErrConfiguration
	}
	firstRead.work = proposedWork
	var codes [13]nodeCodeIdentity
	for i, code := range collection.Codes {
		codes[i] = nodeCodeIdentity(code)
	}
	manifest, err := newProviderHistoryCollectionManifestCandidate(providerHistoryCollectionInput{
		journal: collection.Journal, codes: codes, capture: capture,
		encoding: collection.Encoding, gas: collection.Gas,
		maximumSelected: collection.MaximumSelected, blocks: collection.Blocks,
		native: collection.Native, effects: collection.Effects, versions: collection.Versions,
		presentStops: collection.PresentStops, windowBlocks: collection.WindowBlocks,
		contentBlocks: collection.ContentBlocks, firstRead: firstRead, traceTimeout: collection.TraceTimeout,
	})
	if err != nil || manifest == nil || bounds.History.Stages != manifest.counts.groups {
		return empty, ErrConfiguration
	}
	b.history.collection, b.history.stages = manifest, manifest.counts.groups
	config := providerCustodyConfig{
		custodyDirectory: store.CustodyDirectory, evidenceDirectory: store.EvidenceDirectory,
		anchor: store.Anchor, expectedTip: store.ExpectedTip, expectedSetupTip: store.ExpectedSetupTip,
		profile: store.Profile, identity: identity, policy: policy, capture: capture,
		bounds: b, attempt: 0, expectedScanHead: store.ExpectedScanHead,
	}
	if !b.validStructure() {
		return empty, ErrConfiguration
	}
	if _, ok := providerHistorySourceLimitsFromConfig(config); !ok {
		return empty, ErrConfiguration
	}
	return config, nil
}

func historyPreparationStoreBounds(in historyPreparationBounds) providerIntegrationBounds {
	c, s, h := in.Continuation, in.Scan, in.History
	continuation := &providerLedgerBounds{
		maximumRecords: c.MaximumRecords, pageRecords: c.PageRecords, maximumPages: c.MaximumPages,
		maximumPayloadBytes: c.MaximumPayloadBytes, maximumMetadataBytes: c.MaximumMetadataBytes,
		pagePayloadBytes: c.PagePayloadBytes, pageMetadataBytes: c.PageMetadataBytes,
		maximumResidentBytes: c.MaximumResidentBytes, codecMemoryMultiplier: c.CodecMemoryMultiplier,
		ticketMemoryOverhead: c.TicketMemoryOverhead, operationMemoryOverhead: c.OperationMemoryOverhead,
		segmentMemoryOverhead: c.SegmentMemoryOverhead, objectMemoryOverhead: c.ObjectMemoryOverhead,
		maximumCommits: c.MaximumCommits, maximumObjects: c.MaximumObjects,
	}
	methods := &providerMethodBounds{
		scalar: historyPreparationMethodBound(in.Methods.Scalar), header: historyPreparationMethodBound(in.Methods.Header),
		code: historyPreparationMethodBound(in.Methods.Code), state: historyPreparationMethodBound(in.Methods.State),
		block: historyPreparationMethodBound(in.Methods.Block), receipts: historyPreparationMethodBound(in.Methods.Receipts),
		gas: historyPreparationMethodBound(in.Methods.Gas), execution: historyPreparationMethodBound(in.Methods.Execution),
	}
	scan := &providerScanBounds{
		maximumAttempts: s.MaximumAttempts, maximumRecords: s.MaximumRecords, maximumObjects: s.MaximumObjects,
		maximumRawBytes: s.MaximumRawBytes, maximumPages: s.MaximumPages, maximumChunks: s.MaximumChunks,
		chunkRecords: s.ChunkRecords, chunkObjects: s.ChunkObjects, chunkRawBytes: s.ChunkRawBytes,
		maximumResidentBytes: s.MaximumResidentBytes, filenameEntryOverhead: s.FilenameEntryOverhead,
		maximumFilenameBytes: s.MaximumFilenameBytes, maximumClockReads: s.MaximumClockReads,
		chunkDurationMS: s.ChunkDurationMS, totalDurationMS: s.TotalDurationMS,
		maximumCatalogEntries: s.MaximumCatalogEntries, maximumCatalogNameBytes: s.MaximumCatalogNameBytes,
		maximumPreIRetainedBytes: s.MaximumPreIRetainedBytes, maximumPreIClockReads: s.MaximumPreIClockReads,
		chunkWork: s.ChunkWork, maximumPreIWork: s.MaximumPreIWork,
	}
	admission := &providerFirstAdmissionBounds{
		work: in.FirstAdmission.Work, retained: in.FirstAdmission.Retained, live: in.FirstAdmission.Live,
		clocks: in.FirstAdmission.Clocks, durationMS: in.FirstAdmission.DurationMS,
	}
	history := &providerHistoryProgramBounds{
		program: h.Program, costs: h.Costs,
		future: providerHistoryGroupTotals{
			work: h.Future.Work, physical: h.Future.Physical, retainedRaw: h.Future.RetainedRaw,
			payload: h.Future.Payload, metadata: h.Future.Metadata, objects: h.Future.Objects,
			files: h.Future.Files, clocks: h.Future.Clocks,
		},
		startup:  providerScanUpper{work: h.Startup.Work, retained: h.Startup.Retained, clocks: h.Startup.Clocks},
		recovery: historyPreparationRecoveryBound(h.Recovery),
		final:    providerHistoryFinalBounds{read: historyPreparationRecoveryBound(h.Final.Read), elapsedMS: h.Final.ElapsedMS},
		stages:   h.Stages, records: h.Records, commits: h.Commits, catalogEntries: h.CatalogEntries,
		catalogNameBytes: h.CatalogNameBytes, resident: h.Resident, durationMS: h.DurationMS,
		// collection is installed only by its original constructor above;
		// derivation stays nil. Neither field can be restored from JSON.
	}
	return providerIntegrationBounds{
		ledger: in.Ledger, continuation: continuation, methods: methods,
		tls:  &providerTLSBounds{chains: in.TLS.Chains, certificates: in.TLS.Certificates, certificateBytes: in.TLS.CertificateBytes},
		scan: scan, firstAdmission: admission, history: history,
		maximumCommits: in.MaximumCommits, maximumObjects: in.MaximumObjects, maximumEntries: in.MaximumEntries,
		maximumObjectBytes: in.MaximumObjectBytes, maximumStoredBytes: in.MaximumStoredBytes,
		maximumMetadataPerCommit: in.MaximumMetadataPerCommit, fileOverheadBytes: in.FileOverheadBytes,
		maximumSetupRecoveries: in.MaximumSetupRecoveries, setupWork: in.SetupWork,
		setupLive: in.SetupLive, setupRetained: in.SetupRetained,
		setupMaximumClockReads: in.SetupMaximumClockReads, setupMaximumDurationMS: in.SetupMaximumDurationMS,
	}
}

func historyPreparationMethodBound(in historyPreparationMethod) providerMethodLimit {
	return providerMethodLimit{body: in.Body, wire: in.Wire}
}

func historyPreparationRecoveryBound(in historyPreparationRecovery) providerHistoryRecoveryBounds {
	return providerHistoryRecoveryBounds{work: in.Work, maximumFiles: in.MaximumFiles, maximumBytes: in.MaximumBytes}
}
