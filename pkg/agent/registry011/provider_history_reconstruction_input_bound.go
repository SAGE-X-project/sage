package registry011

// Input attempts and their selected byte ceilings, not a unique-file count,
// durable storage charge or permission to read. Each original pass obtains
// fresh bytes. Missing/partial inputs consume the same attempt; no retry or
// replacement observer is included. Native CPU/backing/live RAM are separate.
type providerHistoryReconstructionFileCost struct {
	metadataPerPass, packsPerPass, spoolPairsPerPass uint64
	firstObjects, firstRaw                           uint64
	files, readBytes, maximumBytes, maximumFile      uint64
	normal, incoming, envelope                       candidateWorkVector
}

// failedInspection selects the closed failure graph's additional pending and
// attempted originals. It is numeric source DATA, never a caller-selected
// live inspection role. The enclosing compiler must bind this value and every
// limit to its original manifest/config before selecting either R or F.
func providerHistoryReconstructionFilesUpper(manifest *providerHistoryCollectionManifest, limits providerHistoryRoleSourceLimits, failedInspection bool) (providerHistoryReconstructionFileCost, bool) {
	var empty providerHistoryReconstructionFileCost
	if !manifest.validSnapshot() || limits.setupRecoveries > 64 || limits.catalogEntries == 0 || limits.catalogEntries > 1<<20 ||
		limits.legacy.firstObjects != manifest.input.firstRead.objects || limits.legacy.firstRaw != manifest.input.firstRead.raw {
		return empty, false
	}
	if _, ok := providerHistoryLegacyControlUpper(limits.legacy); !ok {
		return empty, false
	}
	l := limits.legacy
	g, requests := manifest.counts.groups, manifest.counts.requests
	var a providerHistoryReadArithmetic
	var out providerHistoryReconstructionFileCost
	out.firstObjects = min(l.firstObjects, a.product(4, l.entries))
	// The genuine completed initializer loaded these SAME four inventories
	// and all referenced objects under its original firstRead bounds. Keep
	// that source premise, not the later store map or returned object count.
	out.firstRaw = min(l.firstRaw, a.product(out.firstObjects, l.objectBytes))
	// Per pass: scan4+5A+4C; legacy anchors2, setups3(1+S),
	// highwaters<=5+S and four main rows of four controls; two group HEADs;
	// then18G group controls/inventories and3G small T/C/S objects.
	out.metadataPerPass = a.sum(32, a.product(4, limits.setupRecoveries), a.product(5, l.attempts), a.product(4, l.chunks), a.product(21, g))
	out.packsPerPass = a.product(2, g) // T requests and S results.
	// Settled spools are read once by the group model. The current stage's
	// completed spools are separately read AGAIN by completedMemberModels.
	out.spoolPairsPerPass = a.sum(requests, min(requests, uint64(providerHistoryGroupMembers)))
	if failedInspection {
		// At most ONE terminal failed S adds a failure object. A pending
		// T/C/S tail has at most two P-sized objects, one small object and
		// seven controls. Its one unfinished member adds one spool pair.
		// Full maxima are conservative even when alternatives cannot coexist.
		out.metadataPerPass = a.sum(out.metadataPerPass, 8)
		out.packsPerPass = a.sum(out.packsPerPass, 3)
		out.spoolPairsPerPass = a.sum(out.spoolPairsPerPass, 1)
	}
	// inputPlanLocked refuses before native input if a selected response
	// maximum exceeds the existing32MiB reader domain. Bound ENTERED reads
	// by that domain; this does not promise successful reconstruction for an
	// otherwise incompatible transport profile or narrow its wire allowance.
	wire := min(uint64(manifest.input.capture.WireBytes), uint64(providerHistoryOriginalPackBytes))
	tls := min(l.objectBytes, uint64(64<<10))
	perPassFiles := a.sum(out.metadataPerPass, 4, out.firstObjects, out.packsPerPass, a.product(2, out.spoolPairsPerPass), 2)
	perPassBytes := a.sum(a.product(out.metadataPerPass, l.metadata), 4*(64<<10), out.firstRaw,
		a.product(out.packsPerPass, l.objectBytes), a.product(out.spoolPairsPerPass, a.sum(wire, tls)), 2)
	// Each pass has two intentional empty lock inputs (maximum1). Only
	// finishCatalog reads the final HEAD ONCE after both independent passes.
	out.files, out.readBytes = a.sum(a.product(2, perPassFiles), 1), a.sum(a.product(2, perPassBytes), l.metadata)
	out.maximumBytes = a.sum(out.readBytes, out.files) // maximum+1 at every input.
	out.maximumFile = max(l.metadata, l.objectBytes, wire, uint64(64<<10))
	input, inputOK := providerHistoryRecoveryInputUpper(out.maximumFile)
	if !inputOK {
		return empty, false
	}
	// Sum the actual input primitive's3*(maximum+1)+15, then independently
	// fund its bounded filename construction and original-catalog lookup.
	// The longest name role joins255 bytes and compares two255-byte names.
	match := a.sum(a.product(limits.catalogEntries, 511), 1)
	out.normal[10] = a.sum(a.product(3, out.readBytes), a.product(18, out.files), a.product(767, out.files), a.product(match, out.files))
	// ONE maximum still-outstanding selected-input preflight is sufficient
	// across both passes. Neither per-call guard SUM nor another input quota
	// is manufactured after refusal. Inner native cleanup stays prospective.
	out.incoming[10] = a.sum(input[10], match)
	out.envelope[10] = a.sum(out.normal[10], out.incoming[10])
	if a.failed {
		return empty, false
	}
	return out, true
}

// Joining the two reconstructed models and closing the final original HEAD.
// The actual model construction, namespace matching/observations and the HEAD
// input primitive have different source rows. This is only their terminal
// comparison/copy subgraph, not permission to use their results or a native
// CPU bound. Tail hashes are either empty or freshly generated66-byte digests.
func providerHistoryReconstructionJoinUpper(manifest *providerHistoryCollectionManifest, metadata uint64, failedInspection bool) (candidateWorkVector, bool) {
	if !manifest.validSnapshot() || metadata == 0 || metadata > 1<<20 || manifest.counts.requests == 0 {
		return candidateWorkVector{}, false
	}
	members := min(manifest.counts.requests, uint64(providerHistoryGroupMembers))
	// Each tail comparison observes one scalar equality, attempt-result text,
	// then ALL17 entries, including unused slots. Successful final provenance
	// has no pending checkpoint/member attempt, so ALL18 texts are empty.
	// Failed inspection keeps the full66-byte allowance for each text.
	tail := uint64(2 * (providerHistoryRecoveryTailInputs + 1))
	if failedInspection {
		tail = 134 * uint64(providerHistoryRecoveryTailInputs+1)
	}
	// Two ledger hash equalities266; HEAD text equality2D+1; FOUR tail
	// comparisons; two models each compare current M result hashes TWICE;
	// final original HEAD copy/equality3D+2; one result-copy event2.
	// With D<=1MiB and M<=32 this sum is below2^23, without native allocation,
	// codec, read, clock, storage or another outstanding-input allowance.
	units := 5*metadata + 4*tail + 4*members*133 + 271
	return candidateWorkVector{10: units}, true
}

// One selection's raw file/namespace/join subgraph. Model/codec/semantic/final
// DATA preparation, compiler-own work, outer cleanup and native qualification
// are deliberately not represented as completed by this I/O composition.
type providerHistoryReconstructionIOCost struct {
	limits                     providerHistoryRoleSourceLimits
	files                      providerHistoryReconstructionFileCost
	namespace                  providerHistoryReconstructionNamespaceCost
	joining                    candidateWorkVector
	normal, incoming, envelope candidateWorkVector
}

func providerHistoryCompileReconstructionIO(config providerCustodyConfig, failedInspection bool) (providerHistoryReconstructionIOCost, bool) {
	var empty providerHistoryReconstructionIOCost
	limits, ok := providerHistorySourceLimitsFromConfig(config)
	if !ok {
		return empty, false
	}
	files, filesOK := providerHistoryReconstructionFilesUpper(config.bounds.history.collection, limits, failedInspection)
	namespace, namespaceOK := providerHistoryReconstructionNamespaceUpper(limits.catalogEntries, config.bounds.scan.filenameEntryOverhead, limits.legacy.metadata, limits.legacy.chunks)
	joining, joiningOK := providerHistoryReconstructionJoinUpper(config.bounds.history.collection, limits.legacy.metadata, failedInspection)
	if !filesOK || !namespaceOK || !joiningOK {
		return empty, false
	}
	out := providerHistoryReconstructionIOCost{limits: limits, files: files, namespace: namespace, joining: joining}
	var a providerHistoryReadArithmetic
	for coordinate := range out.normal {
		out.normal[coordinate] = a.sum(files.normal[coordinate], namespace.normal[coordinate], joining[coordinate])
		out.incoming[coordinate] = max(files.incoming[coordinate], namespace.incoming[coordinate])
		out.envelope[coordinate] = a.sum(out.normal[coordinate], out.incoming[coordinate])
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

// Two independent reconstruction passes interpret one monotone retained
// content journey each. Mechanical decoding and the semantic callback are
// disjoint children: selected historical key proofs belong to the callback,
// even though reconstruction performs no network or disk writes. This DATA
// quote does not change the current R/F domain or supply a native certificate.
type providerHistoryReconstructionContentCost struct {
	mechanical, callback       candidateWorkVector
	normal, incoming, envelope candidateWorkVector
	selectedAttempts           uint64
}

func providerHistoryReconstructionContentUpper(manifest *providerHistoryCollectionManifest) (providerHistoryReconstructionContentCost, bool) {
	var empty providerHistoryReconstructionContentCost
	if !manifest.validSnapshot() {
		return empty, false
	}
	in := manifest.input
	evidence, body := uint64(in.journal.Replay.MaxEvidenceBytes), uint64(in.capture.BodyBytes)
	mechanical, mechanicalOK := providerHistoryContentJourneyUpper(providerHistoryContentJourneyLimits{
		Windows: manifest.counts.contentWindows, Blocks: in.blocks, PendingBlocks: in.contentBlocks,
		EvidenceBytes: evidence, BodyBytes: body, ValueBytes: min(evidence, body), TransactionBytes: 64 << 20,
		Transactions: uint64(in.journal.Replay.MaxTransactions), Frames: uint64(in.journal.Replay.MaxFrames), Logs: uint64(in.journal.Replay.MaxLogs),
		Effects: in.effects, NativeBlocks: in.native,
	}, in.encoding, in.gas)
	callback, callbackOK := providerHistoryCallbackJourneyUpper(manifest.counts.contentWindows, in.versions, uint64(in.journal.Replay.MaxLogs), evidence)
	if !mechanicalOK || !callbackOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	out := providerHistoryReconstructionContentCost{selectedAttempts: callback.selectedAttempts}
	for coordinate := range out.normal {
		out.mechanical[coordinate] = a.product(2, mechanical.normal[coordinate])
		out.callback[coordinate] = a.product(2, callback.normal[coordinate])
		out.normal[coordinate] = a.sum(out.mechanical[coordinate], out.callback[coordinate])
		// The first refused transition terminates reconstruction. Its one
		// outstanding preflight cannot run again in the other fresh model.
		out.incoming[coordinate] = max(mechanical.incoming[coordinate], callback.incoming[coordinate])
		out.envelope[coordinate] = a.sum(out.normal[coordinate], out.incoming[coordinate])
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

// Each fresh recovery model constructs one ledger, then replays T/C/S once.
// T and C each use a validation root and an advancing root. S adds its final
// valid/terminal root, giving 1+7G roots per pass. Outside those roots each
// original record is parsed and then validated/constructed once more by the
// transition. Completed and first-aborted prefixes fit the same maxima;
// neither producer execution nor a failed-S retry is reconstruction work.
type providerHistoryReconstructionLedgerCost struct {
	ledger, records            candidateWorkVector
	normal, incoming, envelope candidateWorkVector
	roots, recordCalls         uint64
}

func providerHistoryReconstructionLedgerUpper(manifest *providerHistoryCollectionManifest) (providerHistoryReconstructionLedgerCost, bool) {
	var empty providerHistoryReconstructionLedgerCost
	if !manifest.validSnapshot() || manifest.counts.groups == 0 {
		return empty, false
	}
	one, oneOK := providerHistoryPlannedLedgerControlUpper(providerHistoryGroupMembers)
	record, recordOK := providerHistoryRecordControlUpper(providerHistoryGroupMembers)
	if !oneOK || !recordOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	out := providerHistoryReconstructionLedgerCost{
		roots:       a.product(2, a.sum(1, a.product(7, manifest.counts.groups))),
		recordCalls: a.product(4, manifest.counts.groups),
	}
	for coordinate := range out.normal {
		if one.envelope.ledger[coordinate] < one.normal.ledger[coordinate] {
			return empty, false
		}
		out.ledger[coordinate] = a.product(out.roots, one.normal.ledger[coordinate])
		// Direct raw parsers keep their complete original rows. The normal
		// internal row is usable only after the exact next-table comparison;
		// that premise never narrows these separate original record inputs.
		out.records[coordinate] = a.product(out.recordCalls, a.sum(record.ticket[coordinate], record.consumption[coordinate], record.settlement[coordinate]))
		out.normal[coordinate] = a.sum(out.ledger[coordinate], out.records[coordinate])
		out.incoming[coordinate] = one.envelope.ledger[coordinate] - one.normal.ledger[coordinate]
		out.envelope[coordinate] = a.sum(out.normal[coordinate], out.incoming[coordinate])
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

// One selected source graph, still missing cursor/model/boundary/custody
// control and compiler-own/cleanup/native resource rows. A successful result
// must not be installed as a complete R/F bound or production derivation.
type providerHistoryReconstructionSourceCost struct {
	ledger                     providerHistoryReconstructionLedgerCost
	io                         providerHistoryReconstructionIOCost
	content                    providerHistoryReconstructionContentCost
	normal, incoming, envelope candidateWorkVector
}

func providerHistoryCompileReconstructionSource(config providerCustodyConfig, failedInspection bool) (providerHistoryReconstructionSourceCost, bool) {
	var empty providerHistoryReconstructionSourceCost
	io, ioOK := providerHistoryCompileReconstructionIO(config, failedInspection)
	if !ioOK {
		return empty, false
	}
	content, contentOK := providerHistoryReconstructionContentUpper(config.bounds.history.collection)
	ledger, ledgerOK := providerHistoryReconstructionLedgerUpper(config.bounds.history.collection)
	if !contentOK || !ledgerOK {
		return empty, false
	}
	out := providerHistoryReconstructionSourceCost{io: io, content: content, ledger: ledger}
	var a providerHistoryReadArithmetic
	for coordinate := range out.normal {
		out.normal[coordinate] = a.sum(io.normal[coordinate], content.normal[coordinate], ledger.normal[coordinate])
		out.incoming[coordinate] = max(io.incoming[coordinate], content.incoming[coordinate], ledger.incoming[coordinate])
		out.envelope[coordinate] = a.sum(out.normal[coordinate], out.incoming[coordinate])
	}
	if a.failed {
		return empty, false
	}
	return out, true
}
