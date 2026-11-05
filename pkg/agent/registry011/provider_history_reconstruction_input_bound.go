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

// Each fresh group model constructs its own cursor, copies it into a new
// ledger, then replays one retained T/C/S sequence. The constructor's copy
// commits once and the initial ledger root commits again. Each complete group
// adds two T roots, two C roots, three S ledger roots, one boundary copy and
// its explicit closing commitment. Recovery does not run the extra final
// validObserved call made by a live prefix reader. A failed/pending last group
// consumes a subset; it cannot acquire a further group after the selected G.
type providerHistoryReconstructionCursorCost struct {
	limits         providerHistoryCollectionCursorLimits
	calls          providerHistoryCollectionStateRootCalls
	initialization candidateWorkVector

	normal, incoming, envelope candidateWorkVector
}

func providerHistoryReconstructionCursorUpper(manifest *providerHistoryCollectionManifest) (providerHistoryReconstructionCursorCost, bool) {
	var empty providerHistoryReconstructionCursorCost
	if !manifest.validSnapshot() || manifest.counts.groups == 0 {
		return empty, false
	}
	in := manifest.input
	limits := providerHistoryCollectionCursorLimits{
		Spine: in.blocks, Effects: min(in.effects, in.contentBlocks*uint64(in.journal.Replay.MaxTransactions)),
		PendingBlocks: in.contentBlocks, Transactions: uint64(in.journal.Replay.MaxTransactions),
		EvidenceBytes: uint64(in.journal.Replay.MaxEvidenceBytes), BodyBytes: uint64(in.capture.BodyBytes),
		Semantic: providerHistorySemanticCommitmentLimits{
			ConfirmedClaims: uint64(in.journal.MaxClaims), ConfirmedCreated: uint64(in.journal.MaxIdentities),
			WorkingClaims: uint64(in.journal.MaxClaims), WorkingCreated: uint64(in.journal.MaxIdentities),
			Queue: min(in.versions, in.journal.Replay.History.MaxItems), Window: true, Pending: true,
		},
	}
	var a providerHistoryReadArithmetic
	calls := providerHistoryCollectionStateRootCalls{
		roots:  a.product(2, a.sum(2, a.product(9, manifest.counts.groups))),
		copies: a.product(2, a.sum(1, manifest.counts.groups)),
	}
	// Full selected cardinalities cover every intermediate state, with neither
	// stream assumed empty. Copies already include their initial commitment.
	parts, partsOK := calls.pointwisePartsUpper(limits, uint64(in.journal.Replay.History.MaxBytes), providerHistoryAppliedStateKinds{})
	if !partsOK {
		return empty, false
	}
	initialization := providerHistoryCursorInitializationUpper().cursor
	out := providerHistoryReconstructionCursorCost{limits: limits, calls: calls}
	for coordinate := range out.normal {
		// The original constructor row keeps its full encode preflight. It is
		// reserved twice even when either constructor refuses before the copy.
		out.initialization[coordinate] = a.product(2, initialization[coordinate])
		out.normal[coordinate] = a.sum(out.initialization[coordinate], parts.normal[coordinate])
		out.incoming[coordinate] = parts.headroom[coordinate]
		out.envelope[coordinate] = a.sum(out.normal[coordinate], out.incoming[coordinate])
	}
	if a.failed || !providerHistoryReadWorkDomain(out.envelope) {
		return empty, false
	}
	return out, true
}

// Original T packs are parsed once per group in EACH fresh model. Each T
// admission and each complete S boundary independently regenerates/compares
// its next table. These are not producer planning or repeated prefix reads.
// Raw request-pack inputs keep their COMPLETE old row; only the generated
// closed-template subgraph uses the existing conditional normal/envelope.
type providerHistoryReconstructionRequestCost struct {
	packParses, nextTables uint64
	packs, planning        candidateWorkVector

	normal, incoming, envelope candidateWorkVector
}

func providerHistoryReconstructionRequestUpper(manifest *providerHistoryCollectionManifest, maximum uint64) (providerHistoryReconstructionRequestCost, bool) {
	var empty providerHistoryReconstructionRequestCost
	if !manifest.validSnapshot() || manifest.counts.groups == 0 {
		return empty, false
	}
	requests, requestsOK := providerHistoryRequestControlUpper(providerHistoryGroupMembers, maximum)
	effects := min(manifest.input.effects, manifest.input.contentBlocks*uint64(manifest.input.journal.Replay.MaxTransactions))
	planning, planningOK := providerHistoryPlannedNextTableControlUpper(providerHistoryGroupMembers, effects, maximum)
	if !requestsOK || !planningOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	out := providerHistoryReconstructionRequestCost{
		packParses: a.product(2, manifest.counts.groups),
		nextTables: a.product(4, manifest.counts.groups),
	}
	for coordinate := range out.normal {
		out.packs[coordinate] = a.product(out.packParses, requests.packParsed[coordinate])
		out.planning[coordinate] = a.product(out.nextTables, a.sum(planning.generated.normal[coordinate], planning.equality.normal[coordinate]))
		out.normal[coordinate] = a.sum(out.packs[coordinate], out.planning[coordinate])
		out.incoming[coordinate] = max(a.difference(planning.generated.envelope[coordinate], planning.generated.normal[coordinate]),
			a.difference(planning.equality.envelope[coordinate], planning.equality.normal[coordinate]))
		out.envelope[coordinate] = a.sum(out.normal[coordinate], out.incoming[coordinate])
	}
	if a.failed || !providerHistoryReadWorkDomain(out.envelope) {
		return empty, false
	}
	return out, true
}

// Both fresh models advance through one monotone collection journey. Reuse
// the assigned RPC and semantic boundary leaves at their ORIGINAL phase cuts,
// not the execution program's many prefix readers or triangular role sums.
// Mechanical content/callback interpretation, cursor roots/copies, nextTable
// work and result custody have separate rows and are not counted here.
type providerHistoryReconstructionBoundaryCost struct {
	rpc, semantic candidateWorkVector

	normal, incoming, envelope candidateWorkVector
}

func providerHistoryReconstructionBoundaryUpper(manifest *providerHistoryCollectionManifest, methods providerMethodBounds) (providerHistoryReconstructionBoundaryCost, bool) {
	var empty providerHistoryReconstructionBoundaryCost
	rpc, rpcOK := providerHistoryCollectionBoundaryRoleModelUpper(manifest, methods)
	semantic, semanticOK := providerHistoryCollectionSemanticBoundaryRoleModelUpper(manifest)
	if !rpcOK || !semanticOK || rpc.groups != semantic.groups {
		return empty, false
	}
	rpcNormal, _, rpcOK := rpc.rankedUpper(rpc.groups, false)
	semanticNormal, _, semanticOK := semantic.rankedUpper(semantic.groups, false)
	if !rpcOK || !semanticOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryReconstructionBoundaryCost
	for coordinate := range out.normal {
		out.rpc[coordinate], out.semantic[coordinate] = a.product(2, rpcNormal[coordinate]), a.product(2, semanticNormal[coordinate])
		out.normal[coordinate] = a.sum(out.rpc[coordinate], out.semantic[coordinate])
		// Each original leaf already funds its siblings through first refusal.
		// The enclosing boundary/recovery stops, so the other fresh pass does
		// not execute a second outstanding failed leaf after that refusal.
		out.incoming[coordinate] = max(rpc.delta[coordinate], semantic.delta[coordinate])
		out.envelope[coordinate] = a.sum(out.normal[coordinate], out.incoming[coordinate])
	}
	if a.failed || !providerHistoryReadWorkDomain(out.envelope) {
		return empty, false
	}
	return out, true
}

// Group checkpoint control rebuilt from EACH model's own files. One group
// has T/C/S frames, inventories and highwaters, plus five direct stored
// objects. A terminal failed S can add ONE failure object. Retain the complete
// original eight-object inventory row for every parser, including partial or
// refusing originals; no reported inventory length narrows that envelope.
// Initial legacy/frontier interpretation and member/result custody remain
// separate. These numeric rows authorize no file read or checkpoint ACK.
type providerHistoryReconstructionOuterCost struct {
	checkpoints, objects    uint64
	primitives, comparisons candidateWorkVector

	normal, incoming, envelope candidateWorkVector
}

func providerHistoryReconstructionOuterUpper(manifest *providerHistoryCollectionManifest, metadata, maximum uint64, failedInspection bool) (providerHistoryReconstructionOuterCost, bool) {
	var empty providerHistoryReconstructionOuterCost
	if !manifest.validSnapshot() || manifest.counts.groups == 0 {
		return empty, false
	}
	one, oneOK := providerHistoryOuterControlUpper(metadata, maximum, 8)
	if !oneOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	groups := manifest.counts.groups
	objects := a.product(5, groups)
	if failedInspection {
		objects = a.sum(objects, 1)
	}
	out := providerHistoryReconstructionOuterCost{checkpoints: a.product(6, groups), objects: a.product(2, objects)}
	for coordinate := range out.primitives {
		// These complete old primitive rows already include their original
		// preflights/refusal siblings, so no additional incoming delta is added.
		out.primitives[coordinate] = a.sum(a.product(out.checkpoints, a.sum(one.frameParsed[coordinate], one.inventoryParsed[coordinate], one.highwaterParsed[coordinate])),
			a.product(out.objects, one.storedObject[coordinate]))
	}
	// Each checkpoint compares the raw highwater to its retained head (with
	// one phase copy), intent to pending, and three original model hashes.
	// Direct object text copies and digest comparisons are OUTSIDE the object
	// constructor. T's requests/ticket bindings and the T/C/S operation hashes
	// add five digest comparisons per group in each independent pass.
	checkpoint := a.sum(a.product(5, metadata), 3, 3*133)
	object := a.sum(maximum, 1, 133)
	// Before replay, HEAD is compared with confirmed and, only on mismatch,
	// pending. The last HEAD is read/compared again. Each pass separately
	// compares/clones its confirmed model and builds/walks its checkpoint chain.
	head := a.sum(a.product(7, metadata), 5)
	if failedInspection {
		head = a.sum(a.product(10, metadata), 7)
	}
	// The chain allocation copies count+1 slots; its event adds one more unit.
	chain := a.sum(a.product(15, groups), 2)
	out.comparisons[10] = a.sum(a.product(out.checkpoints, checkpoint), a.product(out.objects, object),
		a.product(2, a.sum(a.product(5*133, groups), head, 200, chain)))
	for coordinate := range out.normal {
		out.normal[coordinate] = a.sum(out.primitives[coordinate], out.comparisons[coordinate])
		out.envelope[coordinate] = out.normal[coordinate]
	}
	if a.failed || !providerHistoryReadWorkDomain(out.envelope) {
		return empty, false
	}
	return out, true
}

// One selected source graph, still missing initial model/member custody
// control and compiler-own/cleanup/native resource rows. A successful result
// must not be installed as a complete R/F bound or production derivation.
type providerHistoryReconstructionSourceCost struct {
	ledger                     providerHistoryReconstructionLedgerCost
	io                         providerHistoryReconstructionIOCost
	content                    providerHistoryReconstructionContentCost
	cursor                     providerHistoryReconstructionCursorCost
	requests                   providerHistoryReconstructionRequestCost
	boundary                   providerHistoryReconstructionBoundaryCost
	outer                      providerHistoryReconstructionOuterCost
	scan                       providerHistoryReconstructionScanCost
	inventory                  providerHistoryReconstructionInventoryCost
	normal, incoming, envelope candidateWorkVector
}

func providerHistoryCompileReconstructionSource(config providerCustodyConfig, failedInspection bool) (providerHistoryReconstructionSourceCost, bool) {
	var empty providerHistoryReconstructionSourceCost
	io, ioOK := providerHistoryCompileReconstructionIO(config, failedInspection)
	if !ioOK || config.policy == nil {
		return empty, false
	}
	scan, scanOK := providerHistoryReconstructionScanUpper(io.limits.legacy, uint64(len(config.policy.raw)))
	inventory, inventoryOK := providerHistoryReconstructionInventoryUpper(io.limits.legacy)
	content, contentOK := providerHistoryReconstructionContentUpper(config.bounds.history.collection)
	ledger, ledgerOK := providerHistoryReconstructionLedgerUpper(config.bounds.history.collection)
	cursor, cursorOK := providerHistoryReconstructionCursorUpper(config.bounds.history.collection)
	requests, requestsOK := providerHistoryReconstructionRequestUpper(config.bounds.history.collection, config.bounds.maximumObjectBytes)
	boundary, boundaryOK := providerHistoryReconstructionBoundaryUpper(config.bounds.history.collection, *config.bounds.methods)
	outer, outerOK := providerHistoryReconstructionOuterUpper(config.bounds.history.collection, io.limits.legacy.metadata, io.limits.legacy.objectBytes, failedInspection)
	if !contentOK || !ledgerOK || !cursorOK || !requestsOK || !boundaryOK || !outerOK || !scanOK || !inventoryOK {
		return empty, false
	}
	out := providerHistoryReconstructionSourceCost{io: io, content: content, ledger: ledger, cursor: cursor, requests: requests, boundary: boundary, outer: outer, scan: scan, inventory: inventory}
	var a providerHistoryReadArithmetic
	for coordinate := range out.normal {
		out.normal[coordinate] = a.sum(io.normal[coordinate], content.normal[coordinate], ledger.normal[coordinate], cursor.normal[coordinate], requests.normal[coordinate], boundary.normal[coordinate], outer.normal[coordinate], scan.normal[coordinate], inventory.normal[coordinate])
		out.incoming[coordinate] = max(io.incoming[coordinate], content.incoming[coordinate], ledger.incoming[coordinate], cursor.incoming[coordinate], requests.incoming[coordinate], boundary.incoming[coordinate], outer.incoming[coordinate], scan.incoming[coordinate], inventory.incoming[coordinate])
		out.envelope[coordinate] = a.sum(out.normal[coordinate], out.incoming[coordinate])
	}
	if a.failed {
		return empty, false
	}
	return out, true
}
