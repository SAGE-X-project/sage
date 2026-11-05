package registry011

import (
	"strings"
	"testing"
)

func TestProviderHistoryReconstructionContentKeepsHistoricalProofAttemptsForBothFreshModels(t *testing.T) {
	base := historySelectedSourceConfig(t).bounds.history.collection
	for _, versions := range []uint64{0, 1, 3, 128} {
		in := base.input
		in.versions = versions
		manifest, err := newProviderHistoryCollectionManifestCandidate(in)
		if err != nil {
			t.Fatal("independently selected history input", err)
		}
		before := *manifest
		got, ok := providerHistoryReconstructionContentUpper(manifest)
		attempts := min(versions, manifest.counts.contentWindows*uint64(in.journal.Replay.MaxLogs))
		// One original selected mutation admits at most128 new keys. Both
		// fresh models must fund those proofs, including the KEM endorsement.
		proofs := 2 * attempts * 128
		if !ok || *manifest != before || got.selectedAttempts != attempts || got.mechanical[7] != 0 ||
			got.callback[7] != proofs || got.normal[7] != proofs || got.envelope[7] != proofs || got.incoming[7] != 0 {
			t.Fatal("read reconstruction omitted historical proofs or changed the original selection", versions, got, ok)
		}
		for _, coordinate := range []int{0, 1, 8, 9} {
			if got.envelope[coordinate] != 0 {
				t.Fatal("historical proof verification acquired request, wire, segment or write permission", coordinate)
			}
		}
		if versions != 1 {
			continue
		}
		// Component runtime only: fresh ordinary parsed key DATA for each
		// pass, one inert accounting meter, no R/F owner or live authority.
		meter := providerHistoryContentUnitMeter(got.envelope)
		for pass := 0; pass < 2; pass++ {
			keys := providerHistoryProofCostParsedKeys(t, replayTestCreate(t))
			if len(keys) != 2 || !replayVerifyNewKeysObserved(replayTestRegistry, replayTestAgent, keys, keys, 0, meter) {
				t.Fatal("the composed proof row did not fund original signing and KEM proofs", pass)
			}
		}
		if meter.failed || meter.actual[7] != 4 || meter.authority || meter.clocks != 0 {
			t.Fatal("fresh historical proof components lost their separate counted attempts", meter.actual)
		}
	}
}

func TestProviderHistoryReconstructionSourceComposesTheSameSelectedReadGraph(t *testing.T) {
	config := historySelectedSourceConfig(t)
	before, history, manifest := config, *config.bounds.history, *config.bounds.history.collection
	for _, failed := range []bool{false, true} {
		got, ok := providerHistoryCompileReconstructionSource(config, failed)
		if !ok || config != before || *config.bounds.history != history || *config.bounds.history.collection != manifest {
			t.Fatal("read graph changed or lost its original source selection", failed)
		}
		for coordinate := range got.normal {
			if got.normal[coordinate] != got.io.normal[coordinate]+got.content.normal[coordinate]+got.ledger.normal[coordinate]+got.cursor.normal[coordinate]+got.requests.normal[coordinate]+got.boundary.normal[coordinate]+got.outer.normal[coordinate] ||
				got.incoming[coordinate] != max(got.io.incoming[coordinate], got.content.incoming[coordinate], got.ledger.incoming[coordinate], got.cursor.incoming[coordinate], got.requests.incoming[coordinate], got.boundary.incoming[coordinate], got.outer.incoming[coordinate]) ||
				got.envelope[coordinate] != got.normal[coordinate]+got.incoming[coordinate] {
				t.Fatal("source graph lost an actual child or summed mutually outstanding guards", coordinate)
			}
		}
	}
}

func TestProviderHistoryReconstructionCursorKeepsOriginalRecoveryRootSequence(t *testing.T) {
	manifest := historySelectedSourceConfig(t).bounds.history.collection
	before := *manifest
	got, ok := providerHistoryReconstructionCursorUpper(manifest)
	// Two independent original group models: initial ledger copy/root,
	// then each T valid/advance, C valid/advance and S valid/copy/boundary/
	// advance/post-valid. selectedLocked does not validate the ledger again.
	var roots, copies uint64
	for pass := 0; pass < 2; pass++ {
		roots++ // Initial copy's commitment.
		copies++
		roots++ // Initial ledger's root.
		for group := uint64(0); group < manifest.counts.groups; group++ {
			for _, phase := range []string{"T", "C", "S"} {
				roots++ // Original ledger validation.
				if phase == "S" {
					roots++ // Boundary copy's commitment.
					copies++
					roots++ // Boundary's explicit final commitment.
				}
				roots++ // Original ledger advance.
				if phase == "S" {
					roots++ // Original settlement's post-validation.
				}
			}
		}
	}
	if !ok || manifest.counts.groups != 40 || roots != 724 || copies != 82 ||
		got.calls != (providerHistoryCollectionStateRootCalls{724, 82}) || *manifest != before {
		t.Fatal("recovery cursor borrowed repeated producer prefixes or lost an original root/copy", got, ok, roots, copies)
	}
	one, valid := providerHistoryAppliedCursorUpper(got.limits, uint64(manifest.input.journal.Replay.History.MaxBytes), providerHistoryAppliedStateKinds{})
	if !valid {
		t.Fatal("same original cursor component")
	}
	initial := providerHistoryCursorInitializationUpper().cursor
	for coordinate := range got.envelope {
		want := 2*initial[coordinate] + 642*one.normalCommitment[coordinate] + 82*one.normalCopy[coordinate]
		extra := max(one.envelopeCommitment[coordinate]-one.normalCommitment[coordinate], one.envelopeCopy[coordinate]-one.normalCopy[coordinate])
		if got.initialization[coordinate] != 2*initial[coordinate] || got.normal[coordinate] != want ||
			got.incoming[coordinate] != extra || got.envelope[coordinate] != want+extra {
			t.Fatal("cursor copy counted its root twice or lost original incoming work", coordinate)
		}
		if coordinate != 2 && coordinate != 10 && got.envelope[coordinate] != 0 {
			t.Fatal("cursor DATA acquired a network, proof, storage or semantic execution permission", coordinate)
		}
	}
	if zero, valid := providerHistoryReconstructionCursorUpper(nil); valid || zero != (providerHistoryReconstructionCursorCost{}) {
		t.Fatal("missing original returned a partial cursor quote")
	}
	changed := before
	changed.input.blocks++
	if zero, valid := providerHistoryReconstructionCursorUpper(&changed); valid || zero != (providerHistoryReconstructionCursorCost{}) {
		t.Fatal("changed original retained the sealed cursor quote")
	}
}

func TestProviderHistoryReconstructionCursorFundsBothFreshInitialLedgerCopies(t *testing.T) {
	manifest := historySelectedSourceConfig(t).bounds.history.collection
	quote, ok := providerHistoryReconstructionCursorUpper(manifest)
	if !ok {
		t.Fatal("original two-model cursor quote")
	}
	// Original constructors under one inert UNIT meter. This tests only the
	// initial group-model subgraph, not full reconstruction or live authority.
	meter := providerHistoryContentUnitMeter(quote.envelope)
	var previous *providerHistoryCollectionCursor
	for pass := 0; pass < 2; pass++ {
		cursor, err := newProviderHistoryCollectionCursorObserved(admissionTestDigest("a"), manifest, manifest.input.journal.Deployment, meter)
		if err != nil || cursor == nil || !cursor.initial() {
			t.Fatal("fresh original cursor construction", pass, err)
		}
		ledger, err := newProviderHistoryCollectionLedgerObserved(admissionTestDigest("a"), admissionTestDigest("b"), admissionTestDigest("c"),
			admissionTestDigest("e"), manifest.counts.groups, historyLedgerMaximum(), cursor, meter)
		if err != nil || ledger == nil || ledger.collection == nil || ledger.collection == cursor || ledger.collection == previous ||
			ledger.collection.semantic == cursor.semantic || ledger.collection.semantic.observer != meter || ledger.collection.semantic.sealedObserver != meter {
			t.Fatal("original ledger copy lost independent DATA or replaced its observer", pass, err)
		}
		previous = ledger.collection
	}
	if meter.failed || meter.authority || meter.clocks != 0 || meter.upper != quote.envelope {
		t.Fatal("two fresh constructors changed the original allowance or gained authority")
	}
	for coordinate, actual := range meter.actual {
		if actual > quote.normal[coordinate] {
			t.Fatal("initial subgraph exceeds its reserved cursor normal work", coordinate, actual, quote.normal[coordinate])
		}
	}
}

func TestProviderHistoryReconstructionRequestsKeepFreshTablesOutsideRecordValidation(t *testing.T) {
	config := historySelectedSourceConfig(t)
	manifest := config.bounds.history.collection
	before := *manifest
	got, ok := providerHistoryReconstructionRequestUpper(manifest, config.bounds.maximumObjectBytes)
	if !ok || manifest.counts.groups != 40 || got.packParses != 80 || got.nextTables != 160 || *manifest != before {
		t.Fatal("each fresh T pack or independently generated admit/S table was lost", got, ok)
	}
	pack, packOK := providerHistoryRequestControlUpper(32, config.bounds.maximumObjectBytes)
	next, nextOK := providerHistoryPlannedNextTableControlUpper(32,
		min(manifest.input.effects, manifest.input.contentBlocks*uint64(manifest.input.journal.Replay.MaxTransactions)), config.bounds.maximumObjectBytes)
	if !packOK || !nextOK {
		t.Fatal("original raw pack and generated-table children")
	}
	for coordinate := range got.normal {
		// T parses its pack once per pass; next-table selection occurs in T
		// AND S. Their record constructors/validators have a separate owner.
		wantPack := 80 * pack.packParsed[coordinate]
		wantPlan := 160 * (next.generated.normal[coordinate] + next.equality.normal[coordinate])
		extra := max(next.generated.envelope[coordinate]-next.generated.normal[coordinate], next.equality.envelope[coordinate]-next.equality.normal[coordinate])
		if got.packs[coordinate] != wantPack || got.planning[coordinate] != wantPlan || got.normal[coordinate] != wantPack+wantPlan ||
			got.incoming[coordinate] != extra || got.envelope[coordinate] != wantPack+wantPlan+extra {
			t.Fatal("raw request inputs were narrowed or generated-table guards were summed", coordinate)
		}
	}
	for _, maximum := range []uint64{0, 1 << 40} {
		if zero, valid := providerHistoryReconstructionRequestUpper(manifest, maximum); valid || zero != (providerHistoryReconstructionRequestCost{}) {
			t.Fatal("invalid original object limit returned a partial request quote", maximum)
		}
	}
	if zero, valid := providerHistoryReconstructionRequestUpper(nil, config.bounds.maximumObjectBytes); valid || zero != (providerHistoryReconstructionRequestCost{}) {
		t.Fatal("missing original manifest retained request work")
	}
}

func TestProviderHistoryReconstructionBoundaryReplaysWholeJourneyOnlyTwice(t *testing.T) {
	config := historySelectedSourceConfig(t)
	manifest := config.bounds.history.collection
	before := *manifest
	got, ok := providerHistoryReconstructionBoundaryUpper(manifest, *config.bounds.methods)
	rpc, rpcOK := providerHistoryCollectionBoundaryRoleModelUpper(manifest, *config.bounds.methods)
	semantic, semanticOK := providerHistoryCollectionSemanticBoundaryRoleModelUpper(manifest)
	if !ok || !rpcOK || !semanticOK || *manifest != before {
		t.Fatal("same selected boundary journey", ok, rpcOK, semanticOK)
	}
	for coordinate := range got.normal {
		// Independently enumerate all prospective phase/item capacities. The
		// reconstruction has two fresh journeys, not live prefix triangular
		// repeats or a new32-item allowance for every historical item.
		var rpcWork, semanticWork uint64
		for _, tier := range rpc.tiers {
			rpcWork += tier.count * tier.work[coordinate]
		}
		for _, tier := range semantic.tiers {
			semanticWork += tier.count * tier.work[coordinate]
		}
		extra := max(rpc.delta[coordinate], semantic.delta[coordinate])
		if got.rpc[coordinate] != 2*rpcWork || got.semantic[coordinate] != 2*semanticWork ||
			got.normal[coordinate] != 2*(rpcWork+semanticWork) || got.incoming[coordinate] != extra ||
			got.envelope[coordinate] != 2*(rpcWork+semanticWork)+extra {
			t.Fatal("boundary phases lost one fresh model or borrowed producer replays", coordinate)
		}
	}
	if zero, valid := providerHistoryReconstructionBoundaryUpper(manifest, providerMethodBounds{}); valid || zero != (providerHistoryReconstructionBoundaryCost{}) {
		t.Fatal("missing original method bounds returned partial boundary work")
	}
	if zero, valid := providerHistoryReconstructionBoundaryUpper(nil, *config.bounds.methods); valid || zero != (providerHistoryReconstructionBoundaryCost{}) {
		t.Fatal("missing original returned partial boundary work")
	}
}

func TestProviderHistoryReconstructionOuterKeepsBothFullCheckpointPasses(t *testing.T) {
	config := historySelectedSourceConfig(t)
	manifest, metadata, maximum := config.bounds.history.collection, config.bounds.maximumMetadataPerCommit, config.bounds.maximumObjectBytes
	before := *manifest
	full, fullOK := providerHistoryOuterControlUpper(metadata, maximum, 8)
	final, finalOK := providerHistoryReconstructionOuterUpper(manifest, metadata, maximum, false)
	failed, failedOK := providerHistoryReconstructionOuterUpper(manifest, metadata, maximum, true)
	if !fullOK || !finalOK || !failedOK || manifest.counts.groups != 40 || *manifest != before ||
		final.checkpoints != 240 || failed.checkpoints != 240 || final.objects != 400 || failed.objects != 402 {
		t.Fatal("each T/C/S checkpoint or the separate failed-S original was lost", final, failed)
	}
	for _, cost := range []providerHistoryReconstructionOuterCost{final, failed} {
		for coordinate := range cost.normal {
			primitive := 240*(full.frameParsed[coordinate]+full.inventoryParsed[coordinate]+full.highwaterParsed[coordinate]) + cost.objects*full.storedObject[coordinate]
			if cost.primitives[coordinate] != primitive || cost.normal[coordinate] != primitive+cost.comparisons[coordinate] ||
				cost.envelope[coordinate] != cost.normal[coordinate] || cost.incoming[coordinate] != 0 {
				t.Fatal("complete raw outer envelopes were narrowed or reserved twice", coordinate)
			}
			if coordinate != 2 && coordinate != 10 && cost.envelope[coordinate] != 0 {
				t.Fatal("checkpoint DATA acquired another execution permission", coordinate)
			}
		}
	}
	// Relative to final provenance, failed inspection permits the pending
	// HEAD alternative in BOTH fresh passes and ONE extra object per pass.
	// Its complete parser rows keep the same original eight-object envelope.
	for coordinate := range final.envelope {
		extra := 2 * full.storedObject[coordinate]
		if coordinate == 10 {
			extra += 2*(maximum+134) + 2*(3*metadata+2)
		}
		if failed.envelope[coordinate] != final.envelope[coordinate]+extra {
			t.Fatal("failed inspection omitted an original or reset the selected HEAD allowance", coordinate)
		}
	}
}

func TestProviderHistoryReconstructionOuterRefusesInvalidOriginalLimits(t *testing.T) {
	manifest := historySelectedSourceConfig(t).bounds.history.collection
	for _, limits := range [][2]uint64{{0, 1}, {1<<20 + 1, 1}, {1, 0}, {1, 32<<20 + 1}} {
		if zero, ok := providerHistoryReconstructionOuterUpper(manifest, limits[0], limits[1], true); ok || zero != (providerHistoryReconstructionOuterCost{}) {
			t.Fatal("invalid original checkpoint limits returned partial work", limits)
		}
	}
	if zero, ok := providerHistoryReconstructionOuterUpper(nil, 16384, 524288, true); ok || zero != (providerHistoryReconstructionOuterCost{}) {
		t.Fatal("missing original returned partial checkpoint work")
	}
	changed := *manifest
	changed.input.blocks++
	if zero, ok := providerHistoryReconstructionOuterUpper(&changed, 16384, 524288, true); ok || zero != (providerHistoryReconstructionOuterCost{}) {
		t.Fatal("changed original retained its sealed checkpoint quote")
	}
}

func TestProviderHistoryReconstructionContentRejectsAnUnboundOriginal(t *testing.T) {
	if got, ok := providerHistoryReconstructionContentUpper(nil); ok || got != (providerHistoryReconstructionContentCost{}) {
		t.Fatal("missing manifest returned partial content work")
	}
	config := historySelectedSourceConfig(t)
	changed := *config.bounds.history.collection
	changed.input.versions++
	if got, ok := providerHistoryReconstructionContentUpper(&changed); ok || got != (providerHistoryReconstructionContentCost{}) {
		t.Fatal("changed original retained its sealed source quote")
	}
	if got, ok := providerHistoryCompileReconstructionSource(providerCustodyConfig{}, false); ok || got != (providerHistoryReconstructionSourceCost{}) {
		t.Fatal("missing source selection returned a partial reconstruction graph")
	}
}

func TestProviderHistoryReconstructionLedgerKeepsBothOriginalTransitionSequences(t *testing.T) {
	manifest := historySelectedSourceConfig(t).bounds.history.collection
	before := *manifest
	got, ok := providerHistoryReconstructionLedgerUpper(manifest)
	if !ok || manifest.counts.groups != 40 || got.roots != 562 || got.recordCalls != 160 || *manifest != before {
		t.Fatal("initial plus T/C/S/post-S roots or both direct record passes were lost", got, ok)
	}
	one, oneOK := providerHistoryPlannedLedgerControlUpper(32)
	record, recordOK := providerHistoryRecordControlUpper(32)
	if !oneOK || !recordOK {
		t.Fatal("original independent ledger/record children")
	}
	for coordinate := range got.normal {
		wantLedger := 562 * one.normal.ledger[coordinate]
		wantRecords := 160 * (record.ticket[coordinate] + record.consumption[coordinate] + record.settlement[coordinate])
		if got.ledger[coordinate] != wantLedger || got.records[coordinate] != wantRecords ||
			got.normal[coordinate] != wantLedger+wantRecords || got.incoming[coordinate] != one.envelope.ledger[coordinate]-one.normal.ledger[coordinate] ||
			got.envelope[coordinate] != got.normal[coordinate]+got.incoming[coordinate] {
			t.Fatal("fresh read ledger control borrowed producer roles or dropped a raw record operation", coordinate)
		}
		if coordinate != 2 && coordinate != 10 && got.envelope[coordinate] != 0 {
			t.Fatal("ledger control acquired another resource permission", coordinate)
		}
	}
	if zero, valid := providerHistoryReconstructionLedgerUpper(nil); valid || zero != (providerHistoryReconstructionLedgerCost{}) {
		t.Fatal("missing original manifest retained partial ledger work")
	}
}

// Run after the existing normal owning invocation returned detached DATA.
// Only a copied numeric selection loses its synthetic UNIT derivation here;
// no live config/owner/observer is changed or opened by this comparison.
func historyAssertFinalInputSourceCost(t *testing.T, config providerCustodyConfig, data providerHistoryFinalData) {
	t.Helper()
	before, original := config, *config.bounds.history
	copyConfig, history := config, original
	history.derivation = nil
	copyConfig.bounds.history = &history
	quote, ok := providerHistoryCompileReconstructionIO(copyConfig, false)
	if !ok || data.files > quote.files.files || data.bytes > quote.files.readBytes || config != before || *config.bounds.history != original ||
		history.derivation != nil || copyConfig.bounds.valid() {
		t.Fatal("original final input counters exceed the source quote, or numeric comparison changed authority", ok)
	}
}

func TestProviderHistoryReconstructionFilesUpperMatchesIndependentSourceGrammar(t *testing.T) {
	config := historySelectedSourceConfig(t)
	limits, ok := providerHistorySourceLimitsFromConfig(config)
	if !ok || limits.setupRecoveries != 3 || limits.legacy.attempts != 4 || limits.legacy.chunks != 2 ||
		config.bounds.history.collection.counts.groups != 40 || config.bounds.history.collection.counts.requests != 242 {
		t.Fatal("independent source-grammar fixture changed its original selection")
	}
	for _, tc := range []struct {
		failure                 bool
		metadata, packs, spools uint64
		files                   uint64
	}{
		{false, 912, 80, 274, 3221},
		{true, 920, 83, 275, 3247},
	} {
		got, ok := providerHistoryReconstructionFilesUpper(config.bounds.history.collection, limits, tc.failure)
		wire := min(uint64(config.capture.WireBytes), uint64(32<<20))
		// Literal multiplicities independently follow the four old inventory
		// frames, scan/legacy controls, T/C/S objects, duplicate current spools,
		// two empty locks per pass and the sole final outer HEAD.
		wantBytes := 2*(tc.metadata*16384+4*65536+1048576+tc.packs*524288+tc.spools*(wire+65536)+2) + 16384
		if !ok || got.metadataPerPass != tc.metadata || got.packsPerPass != tc.packs || got.spoolPairsPerPass != tc.spools ||
			got.firstObjects != 64 || got.firstRaw != 1048576 || got.files != tc.files || got.readBytes != wantBytes || got.maximumBytes != wantBytes+tc.files ||
			got.maximumFile != max(uint64(524288), wire) || !providerHistoryReadWorkDomain(got.envelope) {
			t.Fatal("file source grammar lost an independent read or changed another work domain", tc.failure, ok, got)
		}
		for coordinate, units := range got.envelope {
			if coordinate != 10 && units != 0 {
				t.Fatal("raw file input acquired decode, semantic, RPC, wire or write permission", coordinate)
			}
		}
		// Three bytes-plus-progress/allocation rows and fifteen metadata
		// units per maximum+1 input; filename and lookup are extra invocations.
		wantNormal := 3*wantBytes + 18*tc.files + 767*tc.files + (4096*511+1)*tc.files
		wantIncoming := 3*(got.maximumFile+1) + 15 + 4096*511 + 1
		if got.normal != (candidateWorkVector{10: wantNormal}) || got.incoming != (candidateWorkVector{10: wantIncoming}) ||
			got.envelope != (candidateWorkVector{10: wantNormal + wantIncoming}) {
			t.Fatal("selected input guard or original filename/catalog work disappeared")
		}
	}
}

func TestProviderHistoryReconstructionIOComposesOneSelectionWithOneOutstandingGuard(t *testing.T) {
	config := historySelectedSourceConfig(t)
	before, history, manifest := config, *config.bounds.history, *config.bounds.history.collection
	final, ok := providerHistoryCompileReconstructionIO(config, false)
	recovery, recoveryOK := providerHistoryCompileReconstructionIO(config, true)
	if !ok || !recoveryOK || final.namespace != recovery.namespace || final.limits != recovery.limits ||
		recovery.files.files-final.files.files != 26 || recovery.incoming != final.incoming {
		t.Fatal("failure-only tails changed another selection or added per-file outstanding guards")
	}
	for _, value := range []providerHistoryReconstructionIOCost{final, recovery} {
		for coordinate := range value.normal {
			if value.normal[coordinate] != value.files.normal[coordinate]+value.namespace.normal[coordinate]+value.joining[coordinate] ||
				value.incoming[coordinate] != max(value.files.incoming[coordinate], value.namespace.incoming[coordinate]) ||
				value.envelope[coordinate] != value.normal[coordinate]+value.incoming[coordinate] {
				t.Fatal("source composition changed an actual row or summed mutually outstanding guards", coordinate)
			}
		}
		if value.envelope[10]-value.joining[10] >= value.files.envelope[10]+value.namespace.envelope[10] {
			t.Fatal("two already complete envelopes were added without separating their outstanding room")
		}
	}
	if config != before || *config.bounds.history != history || *config.bounds.history.collection != manifest || config.bounds.valid() || config.bounds.history.derivation != nil {
		t.Fatal("I/O-only compilation changed original DATA or granted execution")
	}
}

func TestProviderHistoryReconstructionJoinKeepsFourTailClosuresAndBothResultModels(t *testing.T) {
	manifest := historySelectedSourceConfig(t).bounds.history.collection
	// This selected request maximum exceeds32. Both original model comparisons
	// still use only the current group's at-most32 results, never every request
	// in the history or just one model's copy of that current group.
	if manifest.counts.requests != 242 {
		t.Fatal("original comparison cardinality fixture")
	}
	for _, tc := range []struct {
		failure bool
		units   uint64
	}{{false, 17759}, {true, 27263}} {
		got, ok := providerHistoryReconstructionJoinUpper(manifest, 64, tc.failure)
		if !ok || got != (candidateWorkVector{10: tc.units}) {
			t.Fatal("terminal joins omitted unused tail entries, a second model or HEAD copies", tc, got)
		}
		more, moreOK := providerHistoryReconstructionJoinUpper(manifest, 65, tc.failure)
		if !moreOK || more[10]-got[10] != 5 || !providerHistoryReadWorkDomain(got) {
			t.Fatal("one HEAD byte changed independent tail/result work or granted another domain")
		}
		// Execute the ORIGINAL tail comparator and byte/text primitives on
		// inert DATA. This unit object has no owned roots, provenance, active
		// original observer or authority; it is not a reconstructed model.
		r := &providerHistoryRecoveryOwner{meter: providerReplayUnitMeter()}
		hash, head := admissionTestDigest("a"), strings.Repeat("h", 64)
		equal := func(a, b string) {
			t.Helper()
			if !providerReplayTextEqual(a, b, r.meter) {
				t.Fatal("bounded comparison primitive")
			}
		}
		equal(hash, hash) // Current ledger root.
		equal(hash, hash) // Independently retained confirmed root.
		equal(head, head)
		var tail providerHistoryRecoveryTailData
		if tc.failure {
			tail.count, tail.attemptResult = len(tail.entries), hash
			for i := range tail.entries {
				tail.entries[i].digest = hash
			}
		}
		other := tail
		for range 4 {
			if !r.tailDataEqualLocked(&tail, &other) {
				t.Fatal("original terminal tail comparison")
			}
		}
		for range 2 {
			for range 32 {
				equal(hash, hash)
				equal(hash, hash)
			}
		}
		// Use the explicit copy primitive of the original recovery branch.
		// This arithmetic unit supplies no V4 owner to the conditional wrapper.
		if !providerReplayEqual([]byte(head), providerReplayBytes(head, r.meter), r.meter) || !r.meter.event(providerCopyEvent, 1) ||
			r.meter.actual != got || r.meter.clocks != 0 || r.meter.authority {
			t.Fatal("original join primitives differ from their independent literal row", tc.failure, r.meter.actual, got)
		}
	}
	// Literal per-call tail closure difference: FOUR comparisons, each with
	//18 texts changing from empty1 to full-hash133. No new file slot/sample.
	final, _ := providerHistoryReconstructionJoinUpper(manifest, 64, false)
	failed, _ := providerHistoryReconstructionJoinUpper(manifest, 64, true)
	if failed[10]-final[10] != 9504 {
		t.Fatal("failed tail closure was treated as a successful empty tail")
	}
}

func TestProviderHistoryReconstructionJoinRejectsMissingOriginalOrUnboundedHead(t *testing.T) {
	manifest := historySelectedSourceConfig(t).bounds.history.collection
	for _, tc := range []struct {
		manifest *providerHistoryCollectionManifest
		metadata uint64
	}{{nil, 64}, {manifest, 0}, {manifest, 1<<20 + 1}, {manifest, ^uint64(0)}} {
		if got, ok := providerHistoryReconstructionJoinUpper(tc.manifest, tc.metadata, true); ok || got != (candidateWorkVector{}) {
			t.Fatal("invalid join inputs returned a terminal comparison row")
		}
	}
}

func TestProviderHistoryReconstructionFilesUpperRefusesInvalidOrOverflowingOriginalLimits(t *testing.T) {
	config := historySelectedSourceConfig(t)
	base, ok := providerHistorySourceLimitsFromConfig(config)
	if !ok {
		t.Fatal("positive original limits")
	}
	for _, name := range []string{"manifest", "first-objects", "first-raw", "setup", "catalog-zero", "catalog-large", "metadata", "object", "entries", "attempts", "chunks", "codec-overflow"} {
		t.Run(name, func(t *testing.T) {
			limits, manifest := base, config.bounds.history.collection
			switch name {
			case "manifest":
				manifest = nil
			case "first-objects":
				limits.legacy.firstObjects--
			case "first-raw":
				limits.legacy.firstRaw++
			case "setup":
				limits.setupRecoveries = 65
			case "catalog-zero":
				limits.catalogEntries = 0
			case "catalog-large":
				limits.catalogEntries = 1<<20 + 1
			case "metadata":
				limits.legacy.metadata = 0
			case "object":
				limits.legacy.objectBytes = 0
			case "entries":
				limits.legacy.entries = 0
			case "attempts":
				limits.legacy.attempts = 0
			case "chunks":
				limits.legacy.chunks = 0
			case "codec-overflow":
				limits.legacy.chunks = ^uint64(0)
			}
			if got, ok := providerHistoryReconstructionFilesUpper(manifest, limits, true); ok || got != (providerHistoryReconstructionFileCost{}) {
				t.Fatal("invalid file limits leaked a partial source plan", name, ok)
			}
		})
	}
}

func TestProviderHistoryReconstructionIORefusesChangedSelectionWithoutEarlierNamespace(t *testing.T) {
	base := historySelectedSourceConfig(t)
	for _, name := range []string{"history", "minted", "capture", "filename-overhead"} {
		t.Run(name, func(t *testing.T) {
			config, history, scan := base, *base.bounds.history, *base.bounds.scan
			config.bounds.history, config.bounds.scan = &history, &scan
			switch name {
			case "history":
				config.bounds.history = nil
			case "minted":
				history.derivation = &providerHistoryProgramDerivation{}
			case "capture":
				config.capture.Timeout--
			case "filename-overhead":
				scan.filenameEntryOverhead = 0
			}
			if got, ok := providerHistoryCompileReconstructionIO(config, true); ok || got != (providerHistoryReconstructionIOCost{}) {
				t.Fatal("changed I/O selection exposed a positive earlier component", name, ok)
			}
		})
	}
}

func TestProviderHistoryReconstructionFullIntervalQuoteDoesNotQualifySmallCatalog(t *testing.T) {
	config := historySelectedSourceConfig(t)
	in := config.bounds.history.collection.input
	in.maximumSelected, in.blocks, in.native = in.journal.Deployment.Number+38419, 38420, 38420
	in.windowBlocks, in.contentBlocks, in.effects, in.versions, in.presentStops = 4096, 10, 4096, 4096, 10
	manifest := historyCollectionManifest(t, in)
	config.bounds.history.collection, config.bounds.history.stages = manifest, manifest.counts.groups
	limits, ok := providerHistorySourceLimitsFromConfig(config)
	if !ok || manifest.counts.groups != 9226 || manifest.counts.requests != 162391 {
		t.Fatal("full prospective interval DATA changed its independent grammar")
	}
	for _, tc := range []struct {
		failed bool
		files  uint64
	}{{false, 1074373}, {true, 1074399}} {
		quote, ok := providerHistoryReconstructionFilesUpper(manifest, limits, tc.failed)
		if !ok || quote.files != tc.files || quote.files <= 2*limits.catalogEntries || config.bounds.valid() || config.bounds.history.derivation != nil {
			t.Fatal("finite full-interval arithmetic qualified an insufficient selected catalog", tc.failed, ok, quote.files)
		}
	}
}
