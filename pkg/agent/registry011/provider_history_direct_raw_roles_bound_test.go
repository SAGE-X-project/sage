package registry011

import "testing"

func TestProviderHistoryDirectRawRolesMatchIndependentOriginalCuts(t *testing.T) {
	manifest, limits := historyOriginalFileRoleInputs(t)
	for _, row := range []struct {
		prior   uint64
		advance bool
		decoded [6]uint64
		logical [6]uint64
	}{
		{0, true, [6]uint64{0, 0, 65536, 131072, 196608, 131072}, [6]uint64{24646, 2210814, 4335854258, 8673839037, 12967650881, 8674926480}},
		{3, true, [6]uint64{0, 1769472, 65536, 131072, 786432, 524288}, [6]uint64{24646, 117112216149, 4335854258, 8673839037, 52004319326, 34699372110}},
		{39, false, [6]uint64{0, 23003136, 65536, 131072, 7864320, 0}, [6]uint64{24646, 1510549554294, 4335854258, 8673839037, 516483432041, 0}},
	} {
		got, ok := providerHistoryCollectionDirectRawRoleUpper(manifest, limits, row.prior, row.advance)
		if !ok {
			t.Fatal("ordinary constructor-backed original scalar precursor", row.prior)
		}
		roles := [6]providerHistoryStageWorkCost{got.joined.initial, got.joined.before, got.joined.preparation, got.joined.member, got.joined.closure, got.joined.advance}
		for role, value := range roles {
			if value.work != (candidateWorkVector{2: row.decoded[role], 10: row.logical[role]}) || value.clocks != 0 {
				t.Fatal("raw rows changed original role or acquired request/I-O/clock authority", row.prior, role)
			}
		}
		if got.joined.member.work[2] < got.joined.preparation.work[2] || got.joined.member.work[10] < got.joined.preparation.work[10] {
			t.Fatal("full member must include entered preparation")
		}
	}
}

func TestProviderHistoryDirectRawRolesKeepResultAndWrapperCallsSeparate(t *testing.T) {
	manifest, limits := historyOriginalFileRoleInputs(t)
	got, ok := providerHistoryCollectionDirectRawRoleUpper(manifest, limits, 3, true)
	if !ok || got.policyCalls != ([6]uint64{0, 27, 1, 2, 12, 8}) || got.wrapperHashes != ([6]uint64{0, 16, 2, 4, 7, 1}) ||
		got.spoolVisits != ([3]uint64{62, 126, 96}) || got.resultBindings != ([3]uint64{62, 124, 96}) || got.writers != ([2]uint64{524315, 393230}) {
		t.Fatal("actual local policy/wrapper/spool or writer distribution changed")
	}
	result, wrapper := got.rows[providerHistoryRawCurrentResult], got.rows[providerHistoryRawWrapper]
	if result.preparation.work[10] != 8246 || result.member.work[10] != 16492 || result.closure.work[10] != 12768 ||
		wrapper.before.work[10] != 2128 || wrapper.preparation.work[10] != 266 || wrapper.member.work[10] != 532 || wrapper.closure.work[10] != 931 {
		t.Fatal("new result compared twice or original fixed wrapper calls lost")
	}
	if got.rows[providerHistoryRawCheckpoint].initial.work[10] != 24646 || got.rows[providerHistoryRawCheckpoint].before.work[10] != 245857 ||
		got.rows[providerHistoryRawArchive].member.work[10] != 67 || got.rows[providerHistoryRawArchive].closure.work[10] != 4097 ||
		got.rows[providerHistoryRawWriter].before.work[10] != 524315 || got.rows[providerHistoryRawWriter].closure.work[10] != 393230 {
		t.Fatal("initial bridge or actual T-C/S strings and archive copies changed")
	}
}

func TestProviderHistoryDirectRawRolesKeepOwnAdvanceOptional(t *testing.T) {
	manifest, limits := historyOriginalFileRoleInputs(t)
	got, ok := providerHistoryCollectionDirectRawRoleUpper(manifest, limits, 38, true)
	if !ok || got.priorMembers != 241 || got.advanceMembers != 242 || got.wrapperHashes[5] != 1 || got.rows[providerHistoryRawArchive].advance.work[10] != 4097 {
		t.Fatal("current C request cap or its own completed S archive changed")
	}
	stopped, ok := providerHistoryCollectionDirectRawRoleUpper(manifest, limits, 38, false)
	if !ok || stopped.advanceMembers != 0 || stopped.wrapperHashes[5] != 0 || stopped.policyCalls[5] != 0 ||
		stopped.joined.advance != (providerHistoryStageWorkCost{}) || stopped.joined.before != got.joined.before || stopped.joined.member != got.joined.member {
		t.Fatal("absent own advance changed another original role")
	}
}

func TestProviderHistoryDirectRawRolesRefuseWithoutBorrowingOrPartialRows(t *testing.T) {
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
		{nil, limits, 0, false},
		{manifest, badRaw, 0, false},
		{manifest, badObjects, 0, false},
		{manifest, badChunks, 0, false},
		{manifest, badMetadata, 0, false},
		{manifest, badObject, 0, false},
		{manifest, limits, manifest.counts.groups, false},
		{manifest, limits, ^uint64(0), false},
		{manifest, limits, manifest.counts.groups - 1, true},
	} {
		if got, ok := providerHistoryCollectionDirectRawRoleUpper(row.manifest, row.limits, row.prior, row.advance); ok || got != (providerHistoryDirectRawRoleCost{}) {
			t.Fatal("invalid original scalar or inherited child failure leaked partial raw DATA", row.prior, ok)
		}
	}
	// The deficient local DATA is chosen BEFORE its genuine constructor,
	// never patched into an already owned meter, source, window or live seal.
	in := historyCollectionInput(t)
	in.firstRead.work = candidateWorkVector{2: 1, 10: 1}
	underpriced := historyCollectionManifest(t, in)
	if got, ok := providerHistoryCollectionDirectRawRoleUpper(underpriced, limits, 0, false); ok || got != (providerHistoryDirectRawRoleCost{}) {
		t.Fatal("global allowance replaced a deficient original firstReading sublimit")
	}
}
