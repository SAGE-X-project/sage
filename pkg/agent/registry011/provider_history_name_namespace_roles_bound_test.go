package registry011

import "testing"

func TestProviderHistoryNameNamespaceRolesMatchOriginalCutLiterals(t *testing.T) {
	manifest, limits := historyOriginalFileRoleInputs(t)
	for _, row := range []struct {
		prior, members uint64
		advancing      bool
		names, catalog [6]uint64
	}{
		{0, 0, true, [6]uint64{178, 6536, 15548, 31186, 26106, 46526}, [6]uint64{0, 4200843, 70640, 143005, 16881264, 48276}},
		{3, 96, true, [6]uint64{178, 632234, 15548, 31186, 234672, 185570}, [6]uint64{0, 4479003, 626960, 1255645, 17715744, 187356}},
		{39, 241, false, [6]uint64{178, 1905266, 15548, 31186, 659016, 0}, [6]uint64{0, 5045659, 1760272, 3522269, 19415712, 0}},
	} {
		got, ok := providerHistoryCollectionNameNamespaceRoleUpper(manifest, limits, 0, 4096, row.prior, row.advancing)
		if !ok || got.priorMembers != row.members || got.initialPerCatalog != 958 ||
			got.directMemberNames != ([3]uint64{128, 260, 192}) || got.catalogMemberNames != ([3]uint64{496, 1007, 704}) ||
			got.mapCopies != ([6]uint64{0, 2, 0, 0, 8, 0}) {
			t.Fatal("same-source local counts or original request maximum changed", row.prior, ok)
		}
		names := [6]providerHistoryStageWorkCost{got.names.initial, got.names.before, got.names.preparation, got.names.member, got.names.closure, got.names.advance}
		catalog := [6]providerHistoryStageWorkCost{got.namespace.initial, got.namespace.before, got.namespace.preparation, got.namespace.member, got.namespace.closure, got.namespace.advance}
		joined := [6]providerHistoryStageWorkCost{got.joined.initial, got.joined.before, got.joined.preparation, got.joined.member, got.joined.closure, got.joined.advance}
		for role := range names {
			if names[role].work != (candidateWorkVector{10: row.names[role]}) || catalog[role].work != (candidateWorkVector{10: row.catalog[role]}) ||
				joined[role].work != (candidateWorkVector{10: row.names[role] + row.catalog[role]}) || joined[role].clocks != 0 {
				t.Fatal("direct file names and allowed-map names must stay disjoint logical-only rows", row.prior, role)
			}
		}
		if got.joined.member.work[10] < got.joined.preparation.work[10] {
			t.Fatal("full member must include its entered preparation")
		}
	}
}

func TestProviderHistoryNameNamespaceRolesBindSelectedSetupAndCopies(t *testing.T) {
	manifest, limits := historyOriginalFileRoleInputs(t)
	base, ok := providerHistoryCollectionNameNamespaceRoleUpper(manifest, limits, 0, 4096, 3, true)
	if !ok {
		t.Fatal("ordinary original source precursor")
	}
	// Select different offline limits before quoting; no owned meter or
	// original source is modified. The future factory must bind this DATA.
	limits.attempts = 64
	got, ok := providerHistoryCollectionNameNamespaceRoleUpper(manifest, limits, 64, 4097, 3, true)
	if !ok || got.names != base.names || got.initialPerCatalog != 20116 {
		t.Fatal("setup/attempt/map limits changed direct filenames or failed their valid domain")
	}
	before := [6]providerHistoryStageWorkCost{base.namespace.initial, base.namespace.before, base.namespace.preparation, base.namespace.member, base.namespace.closure, base.namespace.advance}
	after := [6]providerHistoryStageWorkCost{got.namespace.initial, got.namespace.before, got.namespace.preparation, got.namespace.member, got.namespace.closure, got.namespace.advance}
	// Per-catalog delta19158, plus two/eight one-entry map-copy deltas.
	want := [6]uint64{0, 77656, 153264, 306528, 233992, 38316}
	for role := range before {
		if after[role].work[10]-before[role].work[10] != want[role] {
			t.Fatal("map copies multiplied by elements or selected setup/attempt rows lost", role)
		}
	}
}

func TestProviderHistoryNameNamespaceRolesSeparateCurrentAndOwnAdvance(t *testing.T) {
	manifest, limits := historyOriginalFileRoleInputs(t)
	got, ok := providerHistoryCollectionNameNamespaceRoleUpper(manifest, limits, 0, 4096, 38, true)
	if !ok || got.priorMembers != 241 || got.advanceMembers != 242 || got.names.advance.work[10] != 423494 || got.namespace.advance.work[10] != 425700 || got.catalogs != ([6]uint64{0, 4, 8, 16, 12, 2}) {
		t.Fatal("current C must retain one global request while its completed own S may use all requests")
	}
	stopped, ok := providerHistoryCollectionNameNamespaceRoleUpper(manifest, limits, 0, 4096, 38, false)
	if !ok || stopped.advanceMembers != 0 || stopped.advancePrefixNames != 0 || stopped.catalogs[5] != 0 ||
		stopped.joined.advance != (providerHistoryStageWorkCost{}) || stopped.joined.before != got.joined.before || stopped.joined.member != got.joined.member {
		t.Fatal("optional own advance changed a different role or charged an absent next window")
	}
}

func TestProviderHistoryNameNamespaceRolesRefuseWrongOriginalSelection(t *testing.T) {
	manifest, limits := historyOriginalFileRoleInputs(t)
	badRaw, badObjects, badChunks, badAttempts := limits, limits, limits, limits
	badRaw.firstRaw++
	badObjects.firstObjects++
	badChunks.chunks = ^uint64(0)
	badAttempts.attempts = 0
	for _, row := range []struct {
		manifest              *providerHistoryCollectionManifest
		limits                providerHistoryLegacyControlLimits
		setup, entries, prior uint64
		advancing             bool
	}{
		{nil, limits, 0, 4096, 0, false},
		{manifest, badRaw, 0, 4096, 0, false},
		{manifest, badObjects, 0, 4096, 0, false},
		{manifest, badChunks, 0, 4096, 0, false},
		{manifest, badAttempts, 0, 4096, 0, false},
		{manifest, limits, 65, 4096, 0, false},
		{manifest, limits, 0, 0, 0, false},
		{manifest, limits, 0, 1<<20 + 1, 0, false},
		{manifest, limits, 0, 4096, manifest.counts.groups, false},
		{manifest, limits, 0, 4096, ^uint64(0), false},
		{manifest, limits, 0, 4096, manifest.counts.groups - 1, true},
	} {
		if got, ok := providerHistoryCollectionNameNamespaceRoleUpper(row.manifest, row.limits, row.setup, row.entries, row.prior, row.advancing); ok || got != (providerHistoryNameNamespaceRoleCost{}) {
			t.Fatal("wrong original scalar selection leaked partial role DATA", row.prior, ok)
		}
	}
}

func TestProviderHistoryNameNamespaceRolesKeepOriginalFirstReadingGuard(t *testing.T) {
	_, limits := historyOriginalFileRoleInputs(t)
	// Original offline input is chosen before its real manifest constructor.
	// This is not a live allowance, restored seal or executable Source.
	in := historyCollectionInput(t)
	in.firstRead.work = candidateWorkVector{2: 1, 10: 1}
	manifest := historyCollectionManifest(t, in)
	if got, ok := providerHistoryCollectionNameNamespaceRoleUpper(manifest, limits, 0, 4096, 0, false); ok || got != (providerHistoryNameNamespaceRoleCost{}) {
		t.Fatal("name quote borrowed whole-stage slack into deficient original firstReading")
	}
}
