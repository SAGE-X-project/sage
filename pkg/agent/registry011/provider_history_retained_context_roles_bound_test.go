package registry011

import (
	"reflect"
	"testing"
)

func TestProviderHistoryRetainedContextRolesMatchIndependentOriginalCuts(t *testing.T) {
	manifest, limits := historyOriginalFileRoleInputs(t)
	got, ok := providerHistoryCollectionRetainedContextRoleUpper(manifest, limits, 4096, 1024, 3)
	if !ok {
		t.Fatal("ordinary same-source firstRead/capacity precursor")
	}
	if got.capacity != 4288 || got.maximumRows != 1476 || got.rows != 358 || got.outsideRows != 285 || got.priorMembers != 96 || got.ancestors != 12 ||
		got.beforeWalk != ([6]uint64{1, 1, 2, 2, 12, 96}) || got.beforeEqual != got.beforeWalk || got.memberEqual != ([6]uint64{4, 2, 4, 8, 48, 384}) {
		t.Fatal("same-C ordered rows or explicit typed-child traversal counts changed", got)
	}
	constructorCopies := 2 * (uint64(reflect.TypeFor[providerHistoryReplayReads]().Size()) + 1)
	if got.recording.before.work != (candidateWorkVector{10: 68359540 + constructorCopies}) ||
		got.refresh.preparation.work != (candidateWorkVector{10: 70186696}) || got.commitments.before.work != (candidateWorkVector{10: 96380148424}) ||
		got.commitments.preparation != got.commitments.before || got.comparisons.before.work[10] != 532 || got.comparisons.preparation.work[10] != 798 {
		t.Fatal("firstReading recording repeated, complete rolling roots dropped, or fresh hashes omitted")
	}
	if got.joined.member != got.joined.preparation || got.context.preparation.work[10] == 0 || got.refresh.before != (providerHistoryStageWorkCost{}) ||
		got.recording.preparation != (providerHistoryStageWorkCost{}) {
		t.Fatal("full member dropped entered preparation or added a second recorder")
	}
}

func TestProviderHistoryRetainedContextRolesKeepZeroAndLastPrefixesDistinct(t *testing.T) {
	manifest, limits := historyOriginalFileRoleInputs(t)
	for _, row := range []struct {
		prior, members, ancestors, rows, outside, refresh, commitments uint64
	}{
		{0, 0, 3, 97, 24, 3024268, 26309901244},
		{39, 241, 120, 1476, 1403, 215648560, 396527797264},
	} {
		got, ok := providerHistoryCollectionRetainedContextRoleUpper(manifest, limits, 4096, 1024, row.prior)
		if !ok || got.priorMembers != row.members || got.ancestors != row.ancestors || got.rows != row.rows || got.outsideRows != row.outside ||
			got.refresh.member.work[10] != row.refresh || got.commitments.member.work[10] != row.commitments || got.context.before.work[10] == 0 {
			t.Fatal("empty archive removed current T/C or last prefix renewed the global member maximum", row.prior, ok)
		}
		if got.joined.initial != (providerHistoryStageWorkCost{}) || got.joined.closure != (providerHistoryStageWorkCost{}) || got.joined.advance != (providerHistoryStageWorkCost{}) {
			t.Fatal("same-C retained context was assigned to another original execution role")
		}
		for c, units := range got.joined.member.work {
			if c != 10 && units != 0 || got.joined.member.clocks != 0 {
				t.Fatal("logical metadata work acquired decoding, network, disk or clock permission", c)
			}
		}
	}
}

func TestProviderHistoryRetainedContextRolesBindExistingTypeWeights(t *testing.T) {
	manifest, limits := historyOriginalFileRoleInputs(t)
	for _, bytes := range []uint64{2, 1024, 2 << 20} {
		shape, shapeOK := providerHistoryMemberContextUpper(providerHistoryMemberContextInput{groups: 1, requests: 1, directoryBytes: bytes})
		if !shapeOK {
			t.Fatal("existing closed type-shape precursor")
		}
		got, ok := providerHistoryCollectionRetainedContextRoleUpper(manifest, limits, 4096, bytes, 0)
		if !ok || got.weights != shape.weights {
			t.Fatal("same six original typed-context shapes must retain their bound", bytes, ok)
		}
		if got.beforeWalk[5] != 0 || got.memberEqual[5] != 0 || got.context.preparation.work[10] == 0 {
			t.Fatal("zero prior facts erased current claims/context, or created prior fact comparisons")
		}
	}
}

func TestProviderHistoryRetainedContextRolesRefuseWrongLocalSelectionAndCapacity(t *testing.T) {
	manifest, limits := historyOriginalFileRoleInputs(t)
	badRaw, badObjects, badChunks := limits, limits, limits
	badRaw.firstRaw++
	badObjects.firstObjects++
	badChunks.chunks = ^uint64(0)
	for _, row := range []struct {
		manifest                  *providerHistoryCollectionManifest
		limits                    providerHistoryLegacyControlLimits
		catalog, directory, prior uint64
	}{
		{nil, limits, 4096, 1024, 0},
		{manifest, badRaw, 4096, 1024, 0},
		{manifest, badObjects, 4096, 1024, 0},
		{manifest, badChunks, 4096, 1024, 0},
		{manifest, limits, 0, 1024, 0},
		{manifest, limits, providerHistoryCatalogEntriesMaximum + 1, 1024, 0},
		{manifest, limits, 1024, 1024, 0}, // A later original prefix cannot fit capacity1216.
		{manifest, limits, 4096, 1, 0},
		{manifest, limits, 4096, 2<<20 + 1, 0},
		{manifest, limits, 4096, 1024, manifest.counts.groups},
		{manifest, limits, 4096, 1024, ^uint64(0)},
	} {
		if got, ok := providerHistoryCollectionRetainedContextRoleUpper(row.manifest, row.limits, row.catalog, row.directory, row.prior); ok || got != (providerHistoryRetainedContextRoleCost{}) {
			t.Fatal("wrong original selection/capacity leaked earlier firstRead or context DATA", row.prior, ok)
		}
	}
	// Offline DATA is chosen before the genuine manifest constructor. This
	// never changes an already-owned meter or restores a past current-C seal.
	in := historyCollectionInput(t)
	in.firstRead.work = candidateWorkVector{2: 1, 10: 1}
	underpriced := historyCollectionManifest(t, in)
	if got, ok := providerHistoryCollectionRetainedContextRoleUpper(underpriced, limits, 4096, 1024, 0); ok || got != (providerHistoryRetainedContextRoleCost{}) {
		t.Fatal("retained context quote borrowed global slack into original firstReading")
	}
}
