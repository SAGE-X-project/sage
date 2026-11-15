package registry011

import "testing"

// Expand the actual physical allocation topology without using the closed
// 24G/26G/5G formulas. Sizes are worst-case numeric DATA, not owned originals.
func historyCollectionWriteSequence(members []uint64, objectBytes, wireBytes, overhead uint64) providerHistoryCollectionWriteCost {
	var out providerHistoryCollectionWriteCost
	write := func(bytes uint64, payload, object bool) {
		out.writes++
		out.storage.files++
		out.storage.raw += bytes
		out.storage.physical += bytes + overhead
		if payload {
			out.storage.payload += bytes
		} else {
			out.storage.metadata += bytes
		}
		if object {
			out.storage.objects++
		}
		out.syncs += 2
		for range 2 {
			out.identityReads += 4
		}
		out.identityReads += 2 // open and the close observation
		out.work[9] += bytes
		out.work[10] += bytes + 1 + 2 + 10
	}
	for _, count := range members {
		// T has request pack/ticket, C consumption, S result pack/settlement.
		for _, packs := range []int{1, 0, 1} {
			if packs != 0 {
				write(objectBytes, true, true)
			}
			write(8192, false, true)
			// Inventory, intent/pending, commit, frontier, highwater, HEAD.
			for range 7 {
				write(8192, false, false)
			}
			out.renames++
			out.syncs++
			out.work[10] += 2
		}
		for range count {
			write(wireBytes, true, true)
			write(65536, false, true)
		}
	}
	return out
}

func TestProviderHistoryCollectionWriteBoundsMatchIndependentAllocationRows(t *testing.T) {
	for _, tc := range []struct {
		name                                 string
		members                              []uint64
		objectBytes, wireBytes, overhead     uint64
		storage                              providerHistoryStorageCharge
		writes, syncs, renames, reads, units uint64
	}{
		{"one-member", []uint64{1}, 32, 128, 4096, providerHistoryStorageCharge{377024, 262336, 192, 262144, 7, 28}, 28, 59, 3, 280, 262706},
		{"two-full-groups", []uint64{32, 32}, 1, 1, 1, providerHistoryStorageCharge{4587768, 4587588, 68, 4587520, 138, 180}, 180, 366, 6, 1800, 4589940},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests uint64
			for _, count := range tc.members {
				requests += count
			}
			got, ok := providerHistoryCollectionSuccessfulWriteUpper(uint64(len(tc.members)), requests, tc.objectBytes, tc.wireBytes, tc.overhead)
			var work candidateWorkVector
			work[9], work[10] = tc.storage.raw, tc.units
			want := providerHistoryCollectionWriteCost{tc.storage, tc.writes, tc.syncs, tc.renames, tc.reads, work}
			if !ok || got != want || historyCollectionWriteSequence(tc.members, tc.objectBytes, tc.wireBytes, tc.overhead) != want {
				t.Fatal("successful allocation/writer rows differ from literal source topology", got, want, ok)
			}
		})
	}
}

func TestProviderHistoryCollectionWriteBoundsCoverFewerRealizedGroupsAndMembers(t *testing.T) {
	upper, ok := providerHistoryCollectionSuccessfulWriteUpper(4, 128, 1024, 4096, 4096)
	if !ok {
		t.Fatal("original finite maxima")
	}
	for _, members := range [][]uint64{{1}, {32}, {1, 32}, {18, 2, 1, 7}, {32, 32, 32, 32}} {
		actual := historyCollectionWriteSequence(members, 1024, 4096, 4096)
		for _, pair := range [][2]uint64{{actual.storage.physical, upper.storage.physical}, {actual.storage.raw, upper.storage.raw},
			{actual.storage.payload, upper.storage.payload}, {actual.storage.metadata, upper.storage.metadata}, {actual.storage.objects, upper.storage.objects},
			{actual.storage.files, upper.storage.files}, {actual.syncs, upper.syncs}, {actual.renames, upper.renames}, {actual.identityReads, upper.identityReads}} {
			if pair[0] > pair[1] {
				t.Fatal("smaller successful realization exceeds original no-refund bound", members, actual, upper)
			}
		}
		for i, value := range actual.work {
			if value > upper.work[i] {
				t.Fatal("scoped writer work exceeds original maximum", i, value, upper.work)
			}
		}
	}
}

func TestProviderHistoryCollectionWriteBoundsRejectInvalidAndOverflowedRows(t *testing.T) {
	for _, tc := range []struct {
		name                                               string
		groups, requests, objectBytes, wireBytes, overhead uint64
	}{
		{"zero-group", 0, 1, 1, 1, 1}, {"few-requests", 2, 1, 1, 1, 1}, {"many-requests", 1, 33, 1, 1, 1},
		{"group-product-overflow", ^uint64(0), 1, 1, 1, 1}, {"group-domain", (1<<20)/3 + 1, (1<<20)/3 + 1, 1, 1, 1},
		{"zero-object", 1, 1, 0, 1, 1}, {"object-domain", 1, 1, 32<<20 + 1, 1, 1},
		{"zero-wire", 1, 1, 1, 0, 1}, {"wire-domain", 1, 1, 1, 16<<20 + 1, 1},
		{"zero-overhead", 1, 1, 1, 1, 0}, {"allocation-overflow", 1, 1, 1, 1, ^uint64(0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, ok := providerHistoryCollectionSuccessfulWriteUpper(tc.groups, tc.requests, tc.objectBytes, tc.wireBytes, tc.overhead); ok || got != (providerHistoryCollectionWriteCost{}) {
				t.Fatal("invalid allocation returned a partially funded writer row", got, ok)
			}
		})
	}
	groups := uint64((1 << 20) / 3)
	if got, ok := providerHistoryCollectionSuccessfulWriteUpper(groups, 32*groups, 32<<20, 16<<20, 4096); !ok || got.storage.physical == 0 || got.work[9] != got.storage.raw {
		t.Fatal("maximum finite original domain needs no group-sized allocation", got, ok)
	}
}
