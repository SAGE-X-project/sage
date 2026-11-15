package registry011

import "testing"

// Chronological caller classes, independently expanded rather than using the
// closed 16R+21G-2 or47G+9R expressions. No filesystem/clock/owner is created.
func historyCollectionMetadataSequence(members []uint64) (pairs, identities uint64) {
	identities += 2 // initialization's own pre/post identities
	for range 2 {
		identities += 2 // each actual initial first-P closure
	}
	for group, count := range members {
		pairs += 1 + 1 + 2 // T capacity enumeration, before and two generic exacts
		identities += 1 + 2 + 1
		identities++  // final identity after the two collection table derivations
		for range 4 { // T planning and independent original readers
			identities += 2
		}
		pairs += 1 + 1 + 2 // C
		identities += 1 + 2 + 1
		for range 3 {
			identities += 2
		}
		pairs += 2 // dispatch's original catalog pre/post
		identities += 2
		for range 2 {
			identities += 2
		}
		for ordinal := uint64(0); ordinal < count; ordinal++ {
			pairs++ // member entry
			identities++
			for range 2 {
				pairs += 2 // original equivalence role's pre/post catalogs
				identities += 2
			}
			pairs++ // after both role models
			identities++
			if ordinal != 0 {
				pairs += 2 // earlier member prefixes
			}
			pairs += 2 // between response/TLS writes and after both
			identities++
			for range 2 {
				pairs += 2 // wrapper plus selected post-write prefix
			}
			pairs++ // final retained-original metadata
			identities++
			pairs++ // outer member completion
			identities++
		}
		pairs++ // S's initial member catalog
		pairs++ // coordinator's spool prefix
		pairs++ // allocation capacity enumeration
		pairs++ // exact before physical S; preparation itself adds no sweep
		for range 2 {
			pairs += 2 // S reader's spool prefix and final exact catalog
		}
		pairs += 2 // generic exact checks around rename
		pairs++    // outer S completion
		identities += 1 + 2 + 2 + 1
		for range 3 {
			identities += 2 // S's full original H closures
		}
		if group+1 < len(members) {
			pairs += 2
			identities += 2
			for range 2 {
				identities += 2 // next-stage full H closures
			}
		}
	}
	return pairs, identities
}

func TestProviderHistoryCollectionMetadataBoundsMatchHandwrittenSourceCalls(t *testing.T) {
	for _, tc := range []struct {
		name                                     string
		members                                  []uint64
		entries, names, pairs, identities, reads uint64
		observed, padding, units                 uint64
	}{
		{"one-member", []uint64{1}, 10, 1000, 35, 56, 420, 71498, 8925, 80423},
		{"two-full-groups", []uint64{32, 32}, 1024, 65536, 1064, 670, 1091664, 141655680, 271320, 141927000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests uint64
			for _, count := range tc.members {
				requests += count
			}
			got, ok := providerHistoryCollectionSuccessfulMetadataUpper(uint64(len(tc.members)), requests, tc.entries, tc.names)
			pairs, identities := historyCollectionMetadataSequence(tc.members)
			var work candidateWorkVector
			work[10] = tc.units
			want := providerHistoryCollectionMetadataCost{tc.pairs, tc.identities, tc.reads, tc.observed, tc.padding, work}
			if !ok || got != want || pairs != tc.pairs || identities != tc.identities {
				t.Fatal("metadata quote differs from literal or actual caller topology", got, want, pairs, identities, ok)
			}
		})
	}
}

// Small numeric word of a normal directory scan, not an I/O adapter, a live
// budget, an authority or a callback into the project. ReadDir preflights256
// before it can know whether the result is an entry or EOF.
func historyDirectoryMetadataPreflightsFit(roots [2][]uint64, budget uint64) bool {
	var spent uint64
	for _, names := range roots {
		if spent > budget || 4 > budget-spent {
			return false
		}
		spent += 4 // caller's observed original identity
		for _, bytes := range names {
			if spent > budget || 256 > budget-spent {
				return false
			}
			spent += bytes + 1 // entry read
			spent += bytes + 1 // name comparison
		}
		if spent > budget || 256 > budget-spent {
			return false
		}
		spent++ // actual EOF observation
	}
	return spent <= budget
}

func TestProviderHistoryCollectionMetadataFundsLastDirectoryPreflight(t *testing.T) {
	// Ordinary name lengths, including both roots' ANCHOR/AUTHORITY.LOCK
	// lengths. Their actual observation total is1122, but the final EOF's
	// preflight needs255 additional units, not just its actual one unit.
	for _, roots := range [][2][]uint64{
		{{6, 14, 255, 255}, {6, 14}},
		{{6, 14}, {6, 14, 255, 255}},
		{{6, 14, 255}, {6, 14, 255}},
	} {
		if historyDirectoryMetadataPreflightsFit(roots, 1122) || !historyDirectoryMetadataPreflightsFit(roots, 1377) {
			t.Fatal("actual-only metadata row cannot fund the original final EOF preflight", roots)
		}
	}
}

func TestProviderHistoryCollectionMetadataBoundsCoverFewerOptionalCalls(t *testing.T) {
	upper, ok := providerHistoryCollectionSuccessfulMetadataUpper(4, 128, 128, 32768)
	if !ok {
		t.Fatal("finite prospective metadata rows")
	}
	for _, members := range [][]uint64{{1}, {32}, {1, 32}, {18, 2, 1, 7}, {32, 32, 32, 32}} {
		pairs, identities := historyCollectionMetadataSequence(members)
		if pairs > upper.directoryPairs || identities > upper.identityCalls {
			t.Fatal("smaller original member sequence exceeds metadata caller bound", members, pairs, identities, upper)
		}
	}
}

func TestProviderHistoryCollectionMetadataBoundsRefuseInvalidOrOverflowedRows(t *testing.T) {
	for _, tc := range []struct {
		name                             string
		groups, requests, entries, names uint64
	}{
		{"zero-groups", 0, 1, 1, 1}, {"few-requests", 2, 1, 1, 1}, {"many-requests", 1, 33, 1, 1},
		{"group-overflow", ^uint64(0), 1, 1, 1}, {"group-domain", (1<<20)/3 + 1, (1<<20)/3 + 1, 1, 1},
		{"zero-entries", 1, 1, 0, 1}, {"entry-domain", 1, 1, providerHistoryCatalogEntriesMaximum + 1, 1}, {"zero-names", 1, 1, 1, 0},
		{"one-sweep-overflow", 1, 1, 1, ^uint64(0)}, {"repeated-sweep-overflow", 1, 1, 1, ^uint64(0)/2 - 1024},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, ok := providerHistoryCollectionSuccessfulMetadataUpper(tc.groups, tc.requests, tc.entries, tc.names); ok || got != (providerHistoryCollectionMetadataCost{}) {
				t.Fatal("invalid metadata maximum returned a partial cost row", got, ok)
			}
		})
	}
	groups := uint64((1 << 20) / 3)
	if got, ok := providerHistoryCollectionSuccessfulMetadataUpper(groups, 32*groups, 1<<20, 255*(1<<20)); !ok || got.work[10] == 0 {
		t.Fatal("largest finite original metadata domain needs no input-sized allocation", got, ok)
	}
}
