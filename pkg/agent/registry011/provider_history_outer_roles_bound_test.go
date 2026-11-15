package registry011

import "testing"

func TestProviderHistoryOuterRolesMatchIndependentOriginalCuts(t *testing.T) {
	manifest, limits := historyOriginalFileRoleInputs(t)
	for _, row := range []struct {
		prior            uint64
		advance          bool
		decoded, logical [6]uint64
	}{
		{0, true, [6]uint64{0, 114688, 49152, 49152, 65536, 98304}, [6]uint64{0, 15122776626, 2684863568, 2684863568, 9545723108, 7258972028}},
		{3, true, [6]uint64{0, 1441792, 49152, 49152, 507904, 393216}, [6]uint64{0, 113118899004, 2684863568, 2684863568, 42211097234, 29035888112}},
		{39, false, [6]uint64{0, 17367040, 49152, 49152, 5816320, 0}, [6]uint64{0, 1289072367540, 2684863568, 2684863568, 434195586746, 0}},
	} {
		got, ok := providerHistoryCollectionOuterRoleUpper(manifest, limits, row.prior, row.advance)
		if !ok || got.joined.initial != (providerHistoryStageWorkCost{}) || got.joined.member != got.joined.preparation {
			t.Fatal("outer DATA duplicated initial legacy or reparsed inner T/C after capture", row.prior, ok)
		}
		roles := [6]providerHistoryStageWorkCost{got.joined.initial, got.joined.before, got.joined.preparation, got.joined.member, got.joined.closure, got.joined.advance}
		for role, value := range roles {
			if value.work != (candidateWorkVector{2: row.decoded[role], 10: row.logical[role]}) || value.clocks != 0 {
				t.Fatal("independent original outer codec role literal changed", row.prior, role)
			}
		}
	}
}

func TestProviderHistoryOuterRolesKeepRebuildAndReaderCallsSeparate(t *testing.T) {
	manifest, limits := historyOriginalFileRoleInputs(t)
	got, ok := providerHistoryCollectionOuterRoleUpper(manifest, limits, 3, true)
	if !ok || got.calls != ([6]providerHistoryOuterControlCalls{
		{}, {2, 93, 2, 92, 62, 94, 61, 92, 6, 83}, {0, 2, 0, 0, 0, 0, 0, 0, 0, 4},
		{0, 2, 0, 0, 0, 0, 0, 0, 0, 4}, {1, 35, 1, 35, 24, 36, 23, 35, 4, 27}, {0, 24, 0, 24, 16, 24, 16, 24, 0, 24},
	}) {
		t.Fatal("actual current/prefix codec and caller-copy counts changed")
	}
	if got.objects.preparation != (providerHistoryStageWorkCost{}) || got.inventories.member != (providerHistoryStageWorkCost{}) ||
		got.readerInputs.member != (providerHistoryStageWorkCost{}) || got.frames.member.work[2] != 16384 || got.highwaters.member.work[2] != 32768 {
		t.Fatal("same-C anchor gained an inner object/inventory reconstruction")
	}
	// Another offline scalar selection exercises the child's real metadata
	// minimum. It selects no store, source, original role or execution quota.
	limits.metadata, limits.objectBytes = 4096, 4096
	small, ok := providerHistoryCollectionOuterRoleUpper(manifest, limits, 0, true)
	if !ok {
		t.Fatal("independent smaller metadata/object scalar domain")
	}
	roles := [6]providerHistoryStageWorkCost{small.joined.initial, small.joined.before, small.joined.preparation, small.joined.member, small.joined.closure, small.joined.advance}
	want := [6]uint64{0, 9113129522, 1778820176, 1778820176, 5706288356, 4531380092}
	for role, value := range roles {
		if value.work[10] != want[role] {
			t.Fatal("smaller object/control minimum diverges from independent byte-cost literal", role)
		}
	}
}

func TestProviderHistoryOuterRolesAdvanceThroughOwnCompletedStage(t *testing.T) {
	manifest, limits := historyOriginalFileRoleInputs(t)
	got, ok := providerHistoryCollectionOuterRoleUpper(manifest, limits, 38, true)
	if !ok || got.calls[5] != (providerHistoryOuterControlCalls{0, 234, 0, 234, 156, 234, 156, 234, 0, 234}) {
		t.Fatal("own A must parse both prefixes through the current completed T/C/S")
	}
	stopped, ok := providerHistoryCollectionOuterRoleUpper(manifest, limits, 38, false)
	if !ok || stopped.calls[5] != (providerHistoryOuterControlCalls{}) || stopped.joined.advance != (providerHistoryStageWorkCost{}) ||
		stopped.joined.before != got.joined.before || stopped.joined.member != got.joined.member {
		t.Fatal("absent own advance changed current source rows")
	}
}

func TestProviderHistoryOuterRolesRefuseWrongSourceLimitsAndLocalWork(t *testing.T) {
	manifest, limits := historyOriginalFileRoleInputs(t)
	badRaw, badObjects, badChunks, badMetadata, badObject := limits, limits, limits, limits, limits
	badRaw.firstRaw++
	badObjects.firstObjects++
	badChunks.chunks = ^uint64(0)
	badMetadata.metadata = 0
	badObject.objectBytes = providerHistoryOriginalPackBytes + 1
	for _, row := range []struct {
		manifest *providerHistoryCollectionManifest
		limits   providerHistoryLegacyControlLimits
		prior    uint64
		advance  bool
	}{
		{nil, limits, 0, false}, {manifest, badRaw, 0, false}, {manifest, badObjects, 0, false},
		{manifest, badChunks, 0, false}, {manifest, badMetadata, 0, false}, {manifest, badObject, 0, false},
		{manifest, limits, manifest.counts.groups, false}, {manifest, limits, ^uint64(0), false},
		{manifest, limits, manifest.counts.groups - 1, true},
	} {
		if got, ok := providerHistoryCollectionOuterRoleUpper(row.manifest, row.limits, row.prior, row.advance); ok || got != (providerHistoryOuterRoleCost{}) {
			t.Fatal("wrong source selection leaked partial outer codec DATA", row.prior, ok)
		}
	}
	// Select this deficient DATA before the genuine manifest constructor.
	// The test does not edit an already owned source/observer/window/seal.
	in := historyCollectionInput(t)
	in.firstRead.work = candidateWorkVector{2: 1, 10: 1}
	underpriced := historyCollectionManifest(t, in)
	if got, ok := providerHistoryCollectionOuterRoleUpper(underpriced, limits, 0, false); ok || got != (providerHistoryOuterRoleCost{}) {
		t.Fatal("outer quote used global slack instead of original firstReading sublimit")
	}
}
