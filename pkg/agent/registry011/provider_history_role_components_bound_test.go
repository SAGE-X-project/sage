package registry011

import "testing"

func historyRoleComponentInputs(t *testing.T) (*providerHistoryCollectionManifest, providerMethodBounds, providerTLSBounds, uint64) {
	t.Helper()
	b, _ := providerFrontierBoundsFixture()
	return historyCollectionManifest(t, historyCollectionInput(t)), *b.methods, *b.tls, b.maximumObjectBytes
}

func TestProviderHistoryRoleComponentsPreserveOriginalCutsAndPermissions(t *testing.T) {
	manifest, methods, certificates, maximum := historyRoleComponentInputs(t)
	// Independent ordinary child-domain precursor: canonical V is bounded
	// by the whole body, while native H retains its original 1024-byte spare.
	// No payload, Source owner or live allowance is constructed here.
	in := manifest.input
	if in.capture.BodyBytes != 128<<10 || in.journal.Replay.MaxEvidenceBytes != 64<<20 {
		t.Fatal("ordinary canonical/native fixture precursor changed")
	}
	childLimits := providerHistoryContentJourneyLimits{
		Windows: manifest.counts.contentWindows, Blocks: in.blocks, PendingBlocks: in.contentBlocks,
		EvidenceBytes: 64 << 20, BodyBytes: 128 << 10,
		ValueBytes: 128 << 10, TransactionBytes: (128 << 10) + 1024,
		Transactions: uint64(in.journal.Replay.MaxTransactions), Frames: uint64(in.journal.Replay.MaxFrames), Logs: uint64(in.journal.Replay.MaxLogs),
		Effects: in.effects, NativeBlocks: in.native,
	}
	mechanical, childOK := providerHistoryContentJourneyUpper(childLimits, in.encoding, in.gas)
	if !childOK {
		t.Fatal("ordinary unchanged mechanical child domain must be admitted")
	}
	tooWide := childLimits
	tooWide.ValueBytes++
	if _, ok := providerHistoryContentJourneyUpper(tooWide, in.encoding, in.gas); ok {
		t.Fatal("native spare must not relax the canonical whole-body ceiling")
	}
	original, oldMethods, oldTLS := *manifest, methods, certificates
	for _, row := range []struct {
		prior     uint64
		advancing bool
	}{{0, false}, {0, true}, {3, true}, {manifest.counts.groups - 1, false}} {
		cost, ok := providerHistoryCollectionRoleComponentsUpper(manifest, methods, certificates, maximum, row.prior, 32, row.advancing)
		if !ok || cost.mechanical != mechanical {
			t.Fatal("finite selected assembly must retain the independently admitted canonical/native child", row, ok)
		}
		advanceClocks := uint64(0)
		if row.advancing {
			advanceClocks = 2
		}
		clocks := [6]uint64{2, 6, 1, 2, 2, advanceClocks}
		for index, part := range [6]providerHistoryStageWorkCost{
			cost.joined.initial, cost.joined.before, cost.joined.preparation, cost.joined.member, cost.joined.closure, cost.joined.advance,
		} {
			if part.clocks != clocks[index] || part.work[0] != 0 || part.work[1] != 0 || part.work[8] != 0 || part.work[9] != 0 {
				t.Fatal("assigned costs invented RPC, wire, segment, disk or clock permission", row, index, part)
			}
		}
		if cost.content.initial != (candidateWorkVector{}) || cost.content.preparation != (candidateWorkVector{}) || cost.content.member != (candidateWorkVector{}) ||
			cost.rpc.initial != (candidateWorkVector{}) || cost.rpc.preparation != (candidateWorkVector{}) || cost.rpc.member != (candidateWorkVector{}) ||
			cost.boundary.initial != (candidateWorkVector{}) || cost.boundary.preparation != (candidateWorkVector{}) || cost.boundary.member != (candidateWorkVector{}) {
			t.Fatal("member copies replayed content or boundary commands", row)
		}
		if cost.callbacks.selectedAttempts != manifest.input.versions || cost.mechanical.normal[7] != 0 ||
			cost.content.successClosure[7] != 3*128*manifest.input.versions || cost.joined.closure.work[7] != cost.content.successClosure[7] {
			t.Fatal("selected state proof count escaped its original closure component", row, cost.content)
		}
		if row.prior == 0 && (cost.content.before != (candidateWorkVector{}) || cost.content.failedClosure != (candidateWorkVector{})) {
			t.Fatal("first failed capture acquired a prior semantic journey")
		}
		if !row.advancing && (cost.joined.advance != (providerHistoryStageWorkCost{}) || cost.content.advance != (candidateWorkVector{})) {
			t.Fatal("final stage acquired an advance")
		}
		for c, prepared := range cost.joined.preparation.work {
			if prepared > cost.joined.member.work[c] {
				t.Fatal("full member dropped its already-entered preparation", row, c)
			}
		}
	}
	if *manifest != original || methods != oldMethods || certificates != oldTLS {
		t.Fatal("pure component quote changed sealed inputs")
	}
}

func TestProviderHistoryRoleComponentsZeroSelectedCapacityKeepsSharedWork(t *testing.T) {
	in := historyCollectionInput(t)
	in.versions, in.presentStops, in.effects, in.native = 0, 0, 0, 0
	manifest := historyCollectionManifest(t, in)
	b, _ := providerFrontierBoundsFixture()
	cost, ok := providerHistoryCollectionRoleComponentsUpper(manifest, *b.methods, *b.tls, b.maximumObjectBytes, 0, 1, true)
	if !ok || cost.callbacks.selectedAttempts != 0 || cost.callbacks.state != (candidateWorkVector{}) || cost.callbacks.queue != (candidateWorkVector{}) ||
		cost.callbacks.outer[2] == 0 || cost.callbacks.command[10] == 0 || cost.joined.before.work[10] == 0 ||
		cost.joined.closure.work[7] != 0 || cost.joined.closure.work[10] == 0 {
		t.Fatal("zero selected versions erased other-agent/global or original role work", ok)
	}
}

func TestProviderHistoryRoleComponentsRefuseInvalidOrUnpriceableContracts(t *testing.T) {
	manifest, methods, certificates, maximum := historyRoleComponentInputs(t)
	bad := *manifest
	bad.owner = &bad
	bad.input.blocks++ // Private pre-H0 DATA, never a live owner or replacement.
	badMethods := methods
	badMethods.state.body = manifest.input.capture.BodyBytes + 1
	for _, row := range []struct {
		manifest                *providerHistoryCollectionManifest
		methods                 providerMethodBounds
		certificates            providerTLSBounds
		maximum, prior, members uint64
		advancing               bool
	}{
		{nil, methods, certificates, maximum, 0, 1, false},
		{&bad, methods, certificates, maximum, 0, 1, false},
		{manifest, badMethods, certificates, maximum, 0, 1, false},
		{manifest, methods, providerTLSBounds{}, maximum, 0, 1, false},
		{manifest, methods, certificates, maximum, 0, 0, false},
		{manifest, methods, certificates, maximum, 0, 33, false},
		{manifest, methods, certificates, maximum, manifest.counts.groups, 1, false},
		{manifest, methods, certificates, maximum, ^uint64(0), 1, true},
		{manifest, methods, certificates, maximum, manifest.counts.groups - 1, 1, true},
		{manifest, methods, certificates, 0, 0, 1, false},
		{manifest, methods, certificates, ^uint64(0), 0, 1, false},
	} {
		got, ok := providerHistoryCollectionRoleComponentsUpper(row.manifest, row.methods, row.certificates, row.maximum, row.prior, row.members, row.advancing)
		if ok || got != (providerHistoryRoleComponentCost{}) {
			t.Fatal("invalid component contract leaked an earlier partial quote", row.prior, row.members, ok)
		}
	}
	// A sealed, syntactically valid tiny capture has no four-byte block/trace
	// representation. It must reach and retain the existing mechanical-child
	// refusal; the parent cannot substitute another profile or return controls.
	in := historyCollectionInput(t)
	in.capture.BodyBytes = 1
	tiny := historyCollectionManifest(t, in)
	limit := providerMethodLimit{1, 16 << 10}
	tinyMethods := providerMethodBounds{limit, limit, limit, limit, limit, limit, limit, limit}
	if !tinyMethods.fits(tiny.input.capture) {
		t.Fatal("valid selected tiny method/capture precursor")
	}
	if got, ok := providerHistoryCollectionRoleComponentsUpper(tiny, tinyMethods, certificates, maximum, 0, 1, false); ok || got != (providerHistoryRoleComponentCost{}) {
		t.Fatal("unpriceable mechanical child was replaced by earlier positive components")
	}
}

func TestProviderHistoryContentRoleComponentsMatchIndependentCutLiterals(t *testing.T) {
	mechanical := providerHistoryControlPair{
		candidateWorkVector{2: 5, 3: 7, 4: 11, 5: 13, 6: 17, 10: 19},
		candidateWorkVector{2: 28, 3: 7, 4: 11, 5: 13, 6: 17, 10: 48},
	}
	callbacks := providerHistoryControlPair{candidateWorkVector{2: 31, 7: 37, 10: 41}, candidateWorkVector{2: 31, 7: 37, 10: 84}}
	closure := candidateWorkVector{2: 131, 3: 21, 4: 33, 5: 39, 6: 51, 7: 111, 10: 223}
	for _, row := range []struct {
		prior                   uint64
		advancing               bool
		before, failed, advance candidateWorkVector
	}{
		{0, false, candidateWorkVector{}, candidateWorkVector{}, candidateWorkVector{}},
		{1, true, candidateWorkVector{2: 347, 3: 63, 4: 99, 5: 117, 6: 153, 7: 333, 10: 583}, closure,
			candidateWorkVector{2: 95, 3: 14, 4: 22, 5: 26, 6: 34, 7: 74, 10: 163}},
	} {
		got, ok := providerHistoryContentRoleComponentsUpper(4, row.prior, row.advancing, mechanical, callbacks)
		want := providerHistoryBoundaryRoleCost{before: row.before, successClosure: closure, failedClosure: row.failed, closure: closure, advance: row.advance}
		if !ok || got != want {
			t.Fatal("original nine/three/two cuts or one serial incoming MAX changed", row.prior, got, ok)
		}
	}
}

func TestProviderHistoryContentRoleComponentsRefuseChildAndParentOverflow(t *testing.T) {
	for _, row := range []struct {
		groups, prior uint64
		advancing     bool
	}{
		{0, 0, false}, {(1<<20)/3 + 1, 0, false}, {4, 4, false}, {4, 3, true}, {4, ^uint64(0), true},
	} {
		if got, ok := providerHistoryContentRoleComponentsUpper(row.groups, row.prior, row.advancing, providerHistoryControlPair{}, providerHistoryControlPair{}); ok || got != (providerHistoryBoundaryRoleCost{}) {
			t.Fatal("invalid source role domain leaked a partial row")
		}
	}
	for _, children := range [][2]providerHistoryControlPair{
		{{candidateWorkVector{2: 1}, candidateWorkVector{}}, {}},
		{{}, {candidateWorkVector{10: 1}, candidateWorkVector{}}},
		{{candidateWorkVector{10: ^uint64(0)}, candidateWorkVector{10: ^uint64(0)}}, {candidateWorkVector{10: 1}, candidateWorkVector{10: 1}}},
		{{candidateWorkVector{10: ^uint64(0) / 4}, candidateWorkVector{10: ^uint64(0) / 4}}, {}},
	} {
		if got, ok := providerHistoryContentRoleComponentsUpper(4, 1, true, children[0], children[1]); ok || got != (providerHistoryBoundaryRoleCost{}) {
			t.Fatal("negative child room, joined normal overflow or finite-child nine-copy overflow leaked state")
		}
	}
}

func TestProviderHistoryRoleComponentJoinRefusesLateWorkAndClockOverflow(t *testing.T) {
	for coordinate := range (candidateWorkVector{}) {
		first := providerHistoryRoleControlParts{initial: providerHistoryStageWorkCost{work: candidateWorkVector{2: 7}}}
		last := providerHistoryRoleControlParts{}
		first.advance.work[coordinate], last.advance.work[coordinate] = ^uint64(0), 1
		if got, ok := providerHistoryJoinRoleComponentParts(first, last); ok || got != (providerHistoryRoleControlParts{}) {
			t.Fatal("late coordinate overflow leaked initial or earlier-role work", coordinate)
		}
	}
	first := providerHistoryRoleControlParts{initial: providerHistoryStageWorkCost{work: candidateWorkVector{2: 7}}, advance: providerHistoryStageWorkCost{clocks: ^uint64(0)}}
	last := providerHistoryRoleControlParts{advance: providerHistoryStageWorkCost{clocks: 1}}
	if got, ok := providerHistoryJoinRoleComponentParts(first, last); ok || got != (providerHistoryRoleControlParts{}) {
		t.Fatal("late clock overflow leaked work from an earlier role")
	}
	for _, parts := range [][]providerHistoryRoleControlParts{nil, make([]providerHistoryRoleControlParts, 9)} {
		if got, ok := providerHistoryJoinRoleComponentParts(parts...); ok || got != (providerHistoryRoleControlParts{}) {
			t.Fatal("unbounded or absent child list accepted")
		}
	}
}
