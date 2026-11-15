package registry011

import "testing"

// Small chronological DATA examples. This deliberately walks per-phase file
// classes instead of substituting the closed weighted-prefix formulas.
func historyCollectionReadByteSequence(members []uint64, in providerHistoryCollectionReadByteInput) uint64 {
	d := max(in.metadata, uint64(8192))
	total := 2 * (4*d + in.firstRaw)
	healthy := 4*d + in.firstRaw + d
	for group, count := range members {
		total += 12 * healthy
		for member := uint64(0); member < count; member++ {
			for range 2 {
				total += healthy + in.object + 18*d
				total += 7*d + d // original C anchor and final member HEAD
			}
			for range 2 {
				total += member * (in.wire + 65536)
				total += (member + 1) * (in.wire + 65536)
			}
		}
		// T physical writer and pre/post original T readers.
		total += 3*d + 2*(in.object+9*d)
		// C physical writer, initial T, and pre/post outer highwater + T/C.
		total += 3*d + in.object + 9*d + 2*(d+in.object+18*d)
		// Dispatch's two T/C suffixes and its independent actual C anchor.
		total += 2*(in.object+18*d) + 7*d
		// S writer, initial T/C, two S outer highwaters, result/settlement,
		// inventory, controls and T/C. Three complete member spool prefixes.
		total += 3*d + in.object + 18*d
		for range 2 {
			total += d + in.object + d + d + 4*d + in.object + 18*d
		}
		for range 3 {
			total += count * (in.wire + 65536)
		}
		// Prior T/C/S each has highwater + four controls + inventory, plus
		// the five inner objects and two archived spools per member.
		for _, packs := range []int{1, 0, 1} {
			healthy += 6*d + d
			if packs != 0 {
				healthy += in.object
			}
		}
		healthy += count * (in.wire + 65536)
		if group+1 < len(members) {
			total += 2*healthy + d // two next-prefix passes plus final advance HEAD
		}
	}
	// This expands scan reads by class; the independently source-walked
	// catalog caller sequence is carried from the previous numeric component.
	perCatalog := 2 * d
	for range in.attempts {
		perCatalog += 5 * d
	}
	perCatalog += 4 * d
	for range in.chunks {
		perCatalog += 4 * d
	}
	for range 4 {
		perCatalog += 65536
	}
	calls := historyCollectionCatalogReadSequence(members, 1, 1).calls
	for read := uint64(0); read < calls; read++ {
		total += perCatalog
	}
	return total
}

func TestProviderHistoryCollectionReadBytesMatchHandwrittenAndSaturatedRows(t *testing.T) {
	for _, tc := range []struct {
		name          string
		members       []uint64
		bytes, counts uint64
	}{
		{"one-member", []uint64{1}, 46243840, 2077},
		{"two-full-groups", []uint64{32, 32}, 1831641088, 51405},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests uint64
			for _, count := range tc.members {
				requests += count
			}
			in := providerHistoryCollectionReadByteInput{uint64(len(tc.members)), requests, 64, 1, 1, 1 << 20, 8192, 1 << 20, 65536}
			got, ok := providerHistoryCollectionSuccessfulReadByteUpper(in)
			if !ok || got.bytes != tc.bytes || got.counts.contentReads != tc.counts || got.work[10] != tc.bytes+tc.counts ||
				historyCollectionReadByteSequence(tc.members, in) != tc.bytes {
				t.Fatal("weighted source bytes differ from handwritten or exact saturated chronology", got, ok)
			}
			for i, n := range got.work {
				if i != 10 && n != 0 {
					t.Fatal("scoped read component claimed a different work coordinate", got.work)
				}
			}
		})
	}
}

func TestProviderHistoryCollectionReadBytesCoverSmallerOptionalRealizations(t *testing.T) {
	maximum := providerHistoryCollectionReadByteInput{4, 128, 64, 4, 8, 1 << 20, 65536, 1 << 20, 65536}
	upper, ok := providerHistoryCollectionSuccessfulReadByteUpper(maximum)
	if !ok {
		t.Fatal("finite original weighted maximum")
	}
	for _, members := range [][]uint64{{1}, {32}, {1, 32}, {18, 2, 1, 7}, {32, 32, 32, 32}} {
		var requests uint64
		for _, count := range members {
			requests += count
		}
		in := providerHistoryCollectionReadByteInput{uint64(len(members)), requests, 32, 1, 1, 512 << 10, 8192, 512 << 10, 32768}
		if actual := historyCollectionReadByteSequence(members, in); actual > upper.bytes {
			t.Fatal("smaller realized source bytes exceed prospective weighted maximum", members, actual, upper)
		}
	}
	// A single missing final member exercises a nonzero weighted tail. This
	// order maximizes earlier-prefix weight (192) but its later member weight
	// is only189. The independent upper uses192 for both separate rows, so
	// the two-role member replay has a handwritten6*V=20840448 byte gap.
	partial := maximum
	partial.requests = 127
	partialUpper, ok := providerHistoryCollectionSuccessfulReadByteUpper(partial)
	actualPartial := historyCollectionReadByteSequence([]uint64{32, 32, 32, 31}, partial)
	if !ok || partialUpper.bytes != 14403567616 || actualPartial != 14382727168 ||
		actualPartial > partialUpper.bytes || partialUpper.bytes-actualPartial != 20840448 {
		t.Fatal("partial weighted tail violates its independent conservative gap", actualPartial, partialUpper, ok)
	}
	// The 8KiB fixed history metadata cap must still be funded when a numeric
	// store metadata input is smaller; this grants no actual store validity.
	low := providerHistoryCollectionReadByteInput{1, 1, 64, 1, 1, 1 << 20, 1, 1 << 20, 65536}
	got, ok := providerHistoryCollectionSuccessfulReadByteUpper(low)
	if !ok || got.bytes != 46243840 {
		t.Fatal("small metadata input silently shrank fixed history read bounds", got, ok)
	}
}

func TestProviderHistoryCollectionReadBytesRefuseIncompleteOrOverflowedTotals(t *testing.T) {
	valid := providerHistoryCollectionReadByteInput{1, 1, 64, 1, 1, 1 << 20, 8192, 1 << 20, 65536}
	for _, field := range []string{"groups", "catalog-overflow", "zero-first", "large-first", "zero-metadata", "large-metadata", "zero-object", "large-object", "zero-wire", "large-wire", "byte-overflow", "whole-domain-overflow"} {
		t.Run(field, func(t *testing.T) {
			in := valid
			switch field {
			case "groups":
				in.groups = 0
			case "catalog-overflow":
				in.chunks = ^uint64(0)
			case "zero-first":
				in.firstRaw = 0
			case "large-first":
				in.firstRaw = 64<<20 + 1
			case "zero-metadata":
				in.metadata = 0
			case "large-metadata":
				in.metadata = ^uint64(0)
			case "zero-object":
				in.object = 0
			case "large-object":
				in.object = 32<<20 + 1
			case "zero-wire":
				in.wire = 0
			case "large-wire":
				in.wire = 16<<20 + 1
			case "byte-overflow":
				in.metadata = ^uint64(0) >> 1
			case "whole-domain-overflow":
				in.groups = (1 << 20) / 3
				in.requests = 32 * in.groups
				in.object, in.wire = 32<<20, 16<<20
			}
			if got, ok := providerHistoryCollectionSuccessfulReadByteUpper(in); ok || got != (providerHistoryCollectionReadByteCost{}) {
				t.Fatal("invalid or unrepresentable whole bytes returned a partial row", got, ok)
			}
		})
	}
}
