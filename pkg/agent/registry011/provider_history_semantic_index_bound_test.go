package registry011

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestProviderHistorySemanticIndexBoundsMatchHandwrittenEventRows(t *testing.T) {
	slot := uint64(reflect.TypeFor[providerHistorySemanticIndexEntry]().Size())
	header := uint64(reflect.TypeFor[[]providerHistorySemanticIndexEntry]().Size())
	for _, tc := range []struct {
		claims          bool
		entries, rounds uint64
		units64         uint64
	}{
		{true, 0, 0, 67109036}, {false, 0, 0, 67109036},
		{true, 1, 0, 67439360}, {false, 1, 0, 67516135},
		{true, 3, 2, 68102024}, {false, 3, 2, 68331661},
		{true, 5, 3, 68765383}, {false, 5, 3, 69147624},
		{true, 17, 5, 72749309}, {false, 17, 5, 74045798},
	} {
		// The literal oracle uses32-byte entries/24-byte slice descriptors.
		// Adjust only the explicit copy bytes for the actual compiled target:
		// 2n buffer+n population+n per merge round, and two headers/round.
		copies, headers := (3+tc.rounds)*tc.entries, 2*tc.rounds
		wantUnits := tc.units64 - copies*32 - headers*24 + copies*slot + headers*header
		got, ok := providerHistorySemanticIndexUpper(tc.entries, tc.claims)
		want := candidateWorkVector{10: wantUnits}
		if !ok || got != want {
			t.Fatal("semantic index upper differs from handwritten source-event rows", tc, got, want, ok)
		}
	}
}

func TestProviderHistorySemanticIndexBoundsMatchActualFixedMetadataSchemas(t *testing.T) {
	// Fixed scalar DATA exercises real canonical ASCII escaping and numeric
	// widths. It is not a source record, actual index or claimed credential.
	for _, tc := range []struct {
		name, kind string
		value      map[string]any
		bytes      int
		upper      int
	}{
		{"initial", "semantic-index", map[string]any{"index": "created", "count": 1 << 20}, 112, 112},
		{"claim-entry", "semantic-index-entry", map[string]any{"index": "claims", "previous": "0x" + strings.Repeat("a", 64), "ordinal": (1 << 20) - 1,
			"key": strings.Repeat("\x01", 42) + ":" + strings.Repeat("0", 64), "block": "18446744073709551615", "flag": false}, 569, 570},
		{"created-entry", "semantic-index-entry", map[string]any{"index": "created", "previous": "0x" + strings.Repeat("a", 64), "ordinal": (1 << 20) - 1,
			"key": strings.Repeat("\x01", 64), "block": "18446744073709551615", "flag": false}, 637, 637},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.value["format"], tc.value["kind"] = providerHistoryCollectionStateDomain, tc.kind
			raw := providerReplayCanonical(tc.value, candidateLedgerMetadataLimit, nil)
			if len(raw) != tc.bytes || len(raw) > tc.upper || tc.upper > candidateLedgerMetadataLimit {
				t.Fatal("fixed schema no longer fits its independently counted bytes", len(raw), tc.bytes, tc.upper)
			}
		})
	}
}

// Ordinary unqualified semantic DATA only. These entries cannot supply a
// source, live fold, retained checkpoint, inspection result or program witness.
func historySemanticIndexCostData(count int) replayStream {
	s := replayStream{claims: map[replayClaimID]replayClaimState{}, created: map[string]bool{}}
	for i := count - 1; i >= 0; i-- {
		id := replayClaimID{actor: replayTestWallet, digest: [32]byte{byte(i), byte(i >> 8)}}
		s.claims[id] = replayClaimState{block: uint64(sepoliaDeploymentBlock) + uint64(i), consumed: i%2 == 0}
		key := strings.Repeat("a", 59) + fmt.Sprintf("%05d", i)
		s.created[key] = i%2 == 0
	}
	return s
}

func TestProviderHistorySemanticIndexBoundsFundOriginalEventsAndPreflights(t *testing.T) {
	for _, count := range []int{0, 1, 2, 3, 7, 8, 9, 17, 257} {
		s := historySemanticIndexCostData(count)
		for _, claims := range []bool{true, false} {
			upper, ok := providerHistorySemanticIndexUpper(uint64(count), claims)
			if !ok {
				t.Fatal("prospective logical index row")
			}
			baseline := providerHistorySemanticUnitMeter()
			want := providerHistorySemanticIndexRootObserved(s, claims, max(1, count), baseline)
			if want == "" || baseline.failed {
				t.Fatal("normal unqualified index-root precursor", count, claims)
			}
			bounded := providerHistoryContentUnitMeter(upper)
			root := providerHistorySemanticIndexRootObserved(s, claims, max(1, count), bounded)
			// Map iteration may change merge comparison counts between runs;
			// the sorted root stays exact and each actual stays within its upper.
			if root == "" || root != want || bounded.failed {
				t.Fatal("quoted row missed an original event or preflight", count, claims, bounded.actual, upper)
			}
			for coordinate, actual := range bounded.actual {
				if actual > upper[coordinate] {
					t.Fatal("original index events exceed scoped upper", count, claims, coordinate)
				}
			}
		}
	}
}

func TestProviderHistorySemanticIndexBoundsRetainEncodingPreflightRoom(t *testing.T) {
	// A successful root's measured logical events alone can be less than its
	// next encode(M) preflight. Replaying with only that actual vector refuses;
	// the prospective quote funds the real preflight without renewing a hold.
	// This does not assert that every allowance below the upper must refuse.
	s := historySemanticIndexCostData(0)
	baseline := providerHistorySemanticUnitMeter()
	if providerHistorySemanticIndexRootObserved(s, true, 1, baseline) == "" || baseline.failed {
		t.Fatal("empty index-root precursor")
	}
	upper, ok := providerHistorySemanticIndexUpper(0, true)
	if !ok || baseline.actual[10] >= upper[10] {
		t.Fatal("actual/preflight difference precursor", baseline.actual, upper)
	}
	actualOnly := providerHistoryContentUnitMeter(baseline.actual)
	if providerHistorySemanticIndexRootObserved(s, true, 1, actualOnly) != "" || !actualOnly.failed {
		t.Fatal("actual-only vector incorrectly funded prospective encode preflight")
	}
	bounded := providerHistoryContentUnitMeter(upper)
	if providerHistorySemanticIndexRootObserved(s, true, 1, bounded) == "" || bounded.failed {
		t.Fatal("quoted encoding row failed its original preflight")
	}
}

func TestProviderHistorySemanticIndexBoundsCoverOptionalCountsAndRefuseInvalidDomain(t *testing.T) {
	for _, claims := range []bool{true, false} {
		var previous uint64
		for _, count := range []uint64{0, 1, 2, 3, 4, 5, 7, 8, 9, 15, 16, 17, 31, 32, 33, 1023, 1024, 1025, 1 << 20} {
			got, ok := providerHistorySemanticIndexUpper(count, claims)
			if !ok || got[10] < previous || got[10] == 0 || got != (candidateWorkVector{10: got[10]}) {
				t.Fatal("smaller index count exceeds a later prospective maximum", count, claims, got, ok)
			}
			previous = got[10]
		}
		for _, count := range []uint64{1<<20 + 1, ^uint64(0)} {
			if got, ok := providerHistorySemanticIndexUpper(count, claims); ok || got != (candidateWorkVector{}) {
				t.Fatal("out-of-domain index count returned a partial upper", count, claims, got, ok)
			}
		}
	}
}
