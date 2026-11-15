package registry011

import "testing"

func historyOriginalFileRoleInputs(t *testing.T) (*providerHistoryCollectionManifest, providerHistoryLegacyControlLimits) {
	t.Helper()
	manifest, _, _, maximum := historyRoleComponentInputs(t)
	limits := providerHistoryLegacyControlLimits{metadata: 8192, objectBytes: maximum,
		firstObjects: manifest.input.firstRead.objects, firstRaw: manifest.input.firstRead.raw, entries: 32, attempts: 1, chunks: 1}
	return manifest, limits
}

func TestProviderHistoryOriginalFileRolesMatchIndependentCutLiterals(t *testing.T) {
	manifest, limits := historyOriginalFileRoleInputs(t)
	if limits.objectBytes != 256<<10 || manifest.input.capture.WireBytes != 256<<10 || limits.firstRaw != 1<<20 || limits.firstObjects != 64 {
		t.Fatal("ordinary original file scalar precursor changed")
	}
	got, ok := providerHistoryCollectionOriginalFileRoleUpper(manifest, limits, 4096, 4096*255, 4096, 0, true)
	if !ok {
		t.Fatal("ordinary selected firstRead and file quote")
	}
	want := [6]providerHistoryOriginalFileRoleCounts{
		{models: 2, identities: 2, readBytes: 65536, readCalls: 8},
		{models: 9, catalogs: 4, directoryPairs: 10, identities: 11, readBytes: 5029888, readCalls: 242, writes: 17, renames: 2},
		{catalogs: 8, directoryPairs: 8, identities: 6, readBytes: 23445504, readCalls: 334},
		{catalogs: 16, directoryPairs: 16, identities: 9, readBytes: 44417024, readCalls: 462, writes: 2},
		{models: 3, catalogs: 12, directoryPairs: 11, identities: 6, readBytes: 36241408, readCalls: 404, writes: 9, renames: 1},
		{models: 2, catalogs: 2, directoryPairs: 2, identities: 2, readBytes: 22454272, readCalls: 185},
	}
	if got.counts != want {
		t.Fatal("firstReading exclusion, full member preparation or own-advance cut differs from independent literals", got.counts)
	}
	for _, role := range [6]providerHistoryStageWorkCost{got.joined.initial, got.joined.before, got.joined.preparation, got.joined.member, got.joined.closure, got.joined.advance} {
		if role.clocks != 0 || role.work[0] != 0 || role.work[1] != 0 || role.work[8] != 0 {
			t.Fatal("file DATA created request, wire, segment or clock permission")
		}
	}
}

func TestProviderHistoryOriginalFileRolesKeepExistingFirstReadBridgeSeparate(t *testing.T) {
	manifest, limits := historyOriginalFileRoleInputs(t)
	first, firstOK := providerHistoryFirstPReadUpper(limits, manifest.input.capture)
	legacy, legacyOK := providerHistoryLegacyControlUpper(limits)
	if !firstOK || !legacyOK {
		t.Fatal("independent firstReading precursor")
	}
	got, ok := providerHistoryCollectionOriginalFileRoleUpper(manifest, limits, 4096, 4096*255, 4096, 3, true)
	if !ok || got.first != first {
		t.Fatal("same original firstReading inputs must retain their quote")
	}
	for c, units := range first.work {
		if got.firstInitialization[c] != 2*units || got.legacy.initial.work[c] != 2*legacy.modelControl[c] ||
			got.legacy.before.work[c] != 9*(units+legacy.modelControl[c]) || got.legacy.closure.work[c] != 3*(units+legacy.modelControl[c]) ||
			got.legacy.advance.work[c] != 2*(units+legacy.modelControl[c]) {
			t.Fatal("existing initializer bridge duplicated or independent first readers dropped", c)
		}
	}
	if got.legacy.preparation != (providerHistoryStageWorkCost{}) || got.legacy.member != (providerHistoryStageWorkCost{}) {
		t.Fatal("raw same-C member refresh repeated firstReading parsers")
	}
	if got.reads.member.work[10] <= got.reads.preparation.work[10] || got.reads.preparation.work[10] == 0 {
		t.Fatal("full M must include its already-entered P and new-original readback")
	}
}

func TestProviderHistoryOriginalFileRolesSplitWriterAllocationsWithoutRefund(t *testing.T) {
	manifest, limits := historyOriginalFileRoleInputs(t)
	got, ok := providerHistoryCollectionOriginalFileRoleUpper(manifest, limits, 4096, 4096*255, 4096, 0, false)
	if !ok {
		t.Fatal("ordinary writer allocation precursor")
	}
	want := [6]providerHistoryStorageCharge{
		1: {physical: 462848, raw: 393216, payload: 262144, metadata: 131072, objects: 3, files: 17},
		3: {physical: 335872, raw: 327680, payload: 262144, metadata: 65536, objects: 2, files: 2},
		4: {physical: 364544, raw: 327680, payload: 262144, metadata: 65536, objects: 2, files: 9},
	}
	if got.storage != want || got.writes.before.work != (candidateWorkVector{9: 393216, 10: 393441}) ||
		got.writes.member.work != (candidateWorkVector{9: 327680, 10: 327706}) || got.writes.closure.work != (candidateWorkVector{9: 327680, 10: 327799}) {
		t.Fatal("T/C, member spools and S physical/raw/primitive rows changed", got.storage)
	}
	whole, wholeOK := providerHistoryCollectionSuccessfulWriteUpper(1, 32, limits.objectBytes, uint64(manifest.input.capture.WireBytes), 4096)
	if !wholeOK || whole.storage != (providerHistoryStorageCharge{physical: 11575296, raw: 11206656, payload: 8912896, metadata: 2293760, objects: 69, files: 90}) ||
		got.storage[1].physical+32*got.storage[3].physical+got.storage[4].physical != whole.storage.physical {
		t.Fatal("one successful stage split differs from independent full writer literal")
	}
	if got.joined.advance != (providerHistoryStageWorkCost{}) || got.counts[5] != (providerHistoryOriginalFileRoleCounts{}) || got.storage[5] != (providerHistoryStorageCharge{}) {
		t.Fatal("absent own advance acquired file or catalog work")
	}
}

func TestProviderHistoryOriginalFileRolesRefuseMismatchedSublimitsAndLateOverflow(t *testing.T) {
	manifest, limits := historyOriginalFileRoleInputs(t)
	badRaw, badObjects, badChunks := limits, limits, limits
	badRaw.firstRaw++
	badObjects.firstObjects++
	badChunks.chunks = ^uint64(0)
	for _, row := range []struct {
		manifest                        *providerHistoryCollectionManifest
		limits                          providerHistoryLegacyControlLimits
		entries, names, overhead, prior uint64
		advancing                       bool
	}{
		{nil, limits, 4096, 4096 * 255, 4096, 0, false},
		{manifest, badRaw, 4096, 4096 * 255, 4096, 0, false},
		{manifest, badObjects, 4096, 4096 * 255, 4096, 0, false},
		{manifest, badChunks, 4096, 4096 * 255, 4096, 0, false},
		{manifest, limits, 0, 4096 * 255, 4096, 0, false},
		{manifest, limits, providerHistoryCatalogEntriesMaximum + 1, 4096 * 255, 4096, 0, false},
		{manifest, limits, 4096, 0, 4096, 0, false},
		{manifest, limits, 4096, ^uint64(0), 4096, 0, false},
		{manifest, limits, 4096, 4096 * 255, 0, 0, false},
		{manifest, limits, 4096, 4096 * 255, ^uint64(0), 0, false},
		{manifest, limits, 4096, 4096 * 255, 4096, manifest.counts.groups, false},
		{manifest, limits, 4096, 4096 * 255, 4096, ^uint64(0), true},
		{manifest, limits, 4096, 4096 * 255, 4096, manifest.counts.groups - 1, true},
	} {
		if got, ok := providerHistoryCollectionOriginalFileRoleUpper(row.manifest, row.limits, row.entries, row.names, row.overhead, row.prior, row.advancing); ok || got != (providerHistoryOriginalFileRoleCost{}) {
			t.Fatal("detached mismatch or late arithmetic failure leaked an earlier quote", row.prior, ok)
		}
	}
	// Valid offline numeric DATA selected BEFORE the constructor. The local
	// firstReading row is deliberately insufficient; no live meter is patched.
	in := historyCollectionInput(t)
	in.firstRead.work = candidateWorkVector{2: 1, 10: 1}
	underpriced := historyCollectionManifest(t, in)
	if got, ok := providerHistoryCollectionOriginalFileRoleUpper(underpriced, limits, 4096, 4096*255, 4096, 0, false); ok || got != (providerHistoryOriginalFileRoleCost{}) {
		t.Fatal("global role slack replaced the original firstReading sublimit")
	}
}
