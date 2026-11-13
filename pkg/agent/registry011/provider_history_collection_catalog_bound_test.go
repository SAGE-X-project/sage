package registry011

import (
	"strconv"
	"testing"
)

// Independent literal read topology, expanded chronologically rather than
// using the closed 10+5A+4C or16R+16G-2 formula as its own oracle.
func historyCollectionCatalogReadSequence(members []uint64, attempts, chunks uint64) providerHistoryCollectionCatalogBound {
	var out providerHistoryCollectionCatalogBound
	for range 2 { // initial scan highwater in each original root
		out.perCatalog++
	}
	for range attempts {
		for range 5 { // intent, claim, commit and two highwaters
			out.perCatalog++
		}
	}
	for range 2 { // current highwater and SCAN-HEAD in each original root
		out.perCatalog += 2
	}
	for range chunks {
		for range 2 { // chunk and command
			for range 2 { // custody and evidence originals
				out.perCatalog++
			}
		}
	}
	for range 4 { // initial legacy inventories, not their object payloads
		out.perCatalog++
	}
	for group, count := range members {
		out.calls++    // T prepareAdmissionCatalog
		out.calls++    // C prepareAdmissionCatalog
		out.calls += 2 // dispatch before/after original transcript reconstruction
		for ordinal := uint64(0); ordinal < count; ordinal++ {
			out.calls++ // member preparation entry
			for range 2 {
				out.calls += 2 // each original replay role checks catalog before/after
			}
			out.calls++ // after both role models
			if ordinal != 0 {
				out.calls += 2 // two earlier-member prefix passes
			}
			out.calls += 2 // between response/TLS writes and after both writes
			for range 2 {
				out.calls += 2 // post-write prefix wrapper plus selected prefix itself
			}
			out.calls++ // final retained-original catalog
			out.calls++ // enclosing member completion before final clock
		}
		out.calls++    // initial S member catalog
		out.calls++    // completed spool prefix
		out.calls += 2 // prepareAdmissionCatalog: initial prefix plus selected dispatch
		out.calls++    // independent exact catalog before physical S
		for range 2 {  // pre/post rename S reader
			out.calls++ // selected spool prefix's catalog
			out.calls++ // S reader's final exact catalog
		}
		out.calls += 2 // generic write: exact catalog before/after rename
		out.calls++    // enclosing S's final exact catalog
		if group+1 < len(members) {
			out.calls += 2 // each original stage-advance endpoint
		}
	}
	for read := uint64(0); read < out.calls; read++ {
		out.contentReads += out.perCatalog
	}
	return out
}

func TestProviderHistoryCollectionCatalogBoundsMatchHandwrittenSourceRows(t *testing.T) {
	for _, tc := range []struct {
		name             string
		members          []uint64
		attempts, chunks uint64
		want             providerHistoryCollectionCatalogBound
	}{
		{"one-member", []uint64{1}, 1, 1, providerHistoryCollectionCatalogBound{19, 30, 570}},
		{"one-full-group", []uint64{32}, 1, 1, providerHistoryCollectionCatalogBound{19, 526, 9994}},
		{"one-block-unit", []uint64{18, 2, 1, 7}, 4, 8, providerHistoryCollectionCatalogBound{62, 510, 31620}},
		{"three-full-groups", []uint64{32, 32, 32}, 64, 128, providerHistoryCollectionCatalogBound{842, 1582, 1332044}},
		{"partial-split", []uint64{1, 32}, 2, 3, providerHistoryCollectionCatalogBound{32, 558, 17856}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests uint64
			for _, count := range tc.members {
				requests += count
			}
			got, ok := providerHistoryCollectionCatalogReadBound(uint64(len(tc.members)), requests, tc.attempts, tc.chunks)
			if !ok || got != tc.want || historyCollectionCatalogReadSequence(tc.members, tc.attempts, tc.chunks) != tc.want {
				t.Fatal("closed catalog bound differs from literal source sequence", got, ok)
			}
		})
	}
}

func TestProviderHistoryCollectionCatalogBoundsCoverSmallerOptionalRealizations(t *testing.T) {
	for maxGroups := uint64(1); maxGroups <= 4; maxGroups++ {
		for maxRequests := maxGroups; maxRequests <= maxGroups+5; maxRequests++ {
			t.Run(strconv.FormatUint(maxGroups, 10)+"-"+strconv.FormatUint(maxRequests, 10), func(t *testing.T) {
				bound, ok := providerHistoryCollectionCatalogReadBound(maxGroups, maxRequests, 4, 8)
				if !ok {
					t.Fatal("prospective finite bound")
				}
				for groups := uint64(1); groups <= maxGroups; groups++ {
					for requests := groups; requests <= maxRequests; requests++ {
						members := make([]uint64, int(groups))
						members[0] = requests - groups + 1
						for i := 1; i < len(members); i++ {
							members[i] = 1
						}
						actual := historyCollectionCatalogReadSequence(members, 1, 1)
						if actual.calls > bound.calls || actual.contentReads > bound.contentReads || actual.perCatalog > bound.perCatalog {
							t.Fatal("smaller original realization exceeds its prospective catalog bound", actual, bound)
						}
					}
				}
			})
		}
	}
}

func TestProviderHistoryCollectionCatalogBoundsRefuseOverflowAndIncompleteInputs(t *testing.T) {
	for _, tc := range []struct {
		name                               string
		groups, requests, attempts, chunks uint64
	}{
		{"zero-groups", 0, 1, 1, 1}, {"zero-requests", 1, 0, 1, 1}, {"few-requests", 2, 1, 1, 1},
		{"many-requests", 1, 33, 1, 1}, {"zero-attempts", 1, 1, 0, 1}, {"many-attempts", 1, 1, 65, 1},
		{"zero-chunks", 1, 1, 1, 0}, {"group-overflow", ^uint64(0), 1, 1, 1},
		{"group-domain", (1<<20)/3 + 1, (1<<20)/3 + 1, 1, 1},
		{"single-catalog-overflow", 1, 1, 1, ^uint64(0)},
		{"whole-program-overflow", 1, 1, 1, (^uint64(0) - 15) / 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, ok := providerHistoryCollectionCatalogReadBound(tc.groups, tc.requests, tc.attempts, tc.chunks); ok || got != (providerHistoryCollectionCatalogBound{}) {
				t.Fatal("invalid catalog arithmetic produced a partial quote", got, ok)
			}
		})
	}
	groups := uint64((1 << 20) / 3)
	if got, ok := providerHistoryCollectionCatalogReadBound(groups, 32*groups, 64, 128); !ok || got.contentReads == 0 {
		t.Fatal("largest original finite group domain requires no input-sized allocation", got, ok)
	}
}

func TestProviderHistoryCollectionSuccessfulReadUpperComposesDisjointRows(t *testing.T) {
	for _, tc := range []struct {
		name                                    string
		members                                 []uint64
		objects, attempts, chunks, want, clocks uint64
	}{
		{"one-member", []uint64{1}, 1, 1, 1, 1069, 12},
		{"one-block-unit", []uint64{18, 2, 1, 7}, 64, 4, 8, 54841, 96},
		{"two-full-groups", []uint64{32, 32}, 1, 1, 1, 41577, 148},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests uint64
			for _, members := range tc.members {
				requests += members
			}
			groups := uint64(len(tc.members))
			upper, ok := providerHistoryCollectionSuccessfulReadUpper(groups, requests, tc.objects, tc.attempts, tc.chunks)
			if !ok || upper.contentReads != tc.want || upper.ownerClockSamples != tc.clocks ||
				upper.contentReads != upper.prefix.initialModels+upper.prefix.healthyPrefixes+upper.prefix.memberReplay+
					upper.prefix.memberPreparation+upper.prefix.memberRetention+upper.prefix.advance+upper.catalog.contentReads+upper.remainingContents {
				t.Fatal("complete successful content rows overlap, omit a component or double-count planning", upper, ok)
			}
			// Independently walk phase suffixes. Unlike the upper, actual member
			// distribution is known here; upper rows need not be jointly tight.
			actual := historyCollectionEnumeratedReadCounts(tc.members, tc.objects)
			catalog := historyCollectionCatalogReadSequence(tc.members, tc.attempts, tc.chunks)
			var suffix uint64
			clocks := uint64(2) // initialization
			for group, members := range tc.members {
				suffix += 3 // generic T pre-write HEAD/highwater and pre-rename HEAD
				for range 2 {
					suffix += 10 // pre/post T originals
				}
				suffix += 3 + 10 // generic C, initial T replay
				for range 2 {
					suffix += 1 + 10 + 9 // C outer highwater, T and C originals
				}
				for range 2 {
					suffix += 19 // dispatch T/C suffixes
				}
				suffix += 7      // new dispatch's original C anchor
				suffix += 3 + 19 // generic S, initial T/C suffixes
				for range 2 {
					suffix += 1 + 2 + 1 + 4 + 19 // outer highwater, results/settlement, inventory, controls and T/C
				}
				for range 3 {
					suffix += 2 * members // coordinator and pre/post S spool reads
				}
				clocks += 8
				for range members {
					clocks += 2
				}
				if group+1 < len(tc.members) {
					suffix++ // final advance main HEAD, beyond its two healthy replays
					clocks += 2
				}
			}
			actualReads := actual.initialModels + actual.healthyPrefixes + actual.memberReplay + actual.memberPreparation + actual.memberRetention + actual.advance + catalog.contentReads + suffix
			// Saturated groups attain every prefix row: no slack can mask a
			// missing final-advance HEAD read in the composed bound.
			if tc.name == "two-full-groups" && actualReads != upper.contentReads {
				t.Fatal("fully saturated chronological reads must equal the complete successful upper", actualReads, upper)
			}
			if actualReads > upper.contentReads || suffix != upper.remainingContents || clocks != upper.ownerClockSamples {
				t.Fatal("chronological source walk exceeds a supposedly disjoint successful upper", actualReads, upper)
			}
		})
	}
}

func TestProviderHistoryCollectionSuccessfulReadUpperCannotReturnPartialCosts(t *testing.T) {
	// The catalog itself fits, but adding the other successful-path rows does
	// not. This exercises composition refusal rather than a failed component.
	nearLimitChunks := (^uint64(0)/30 - 15) / 4
	if _, ok := providerHistoryCollectionCatalogReadBound(1, 1, 1, nearLimitChunks); !ok {
		t.Fatal("near-limit catalog must fit before whole-read composition")
	}
	for _, tc := range []struct {
		name                                        string
		groups, requests, objects, attempts, chunks uint64
	}{
		{"bad-prefix", 1, 1, 0, 1, 1}, {"bad-catalog", 1, 1, 1, 0, 1},
		{"catalog-overflow", 1, 1, 1, 1, ^uint64(0)},
		{"composed-total-overflow", 1, 1, 1, 1, nearLimitChunks},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, ok := providerHistoryCollectionSuccessfulReadUpper(tc.groups, tc.requests, tc.objects, tc.attempts, tc.chunks); ok || got != (providerHistoryCollectionSuccessfulReadCost{}) {
				t.Fatal("a failed component returned a partially funded whole row", got, ok)
			}
		})
	}
}
