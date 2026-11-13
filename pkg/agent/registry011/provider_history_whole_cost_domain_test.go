package registry011

import (
	"math/big"
	"testing"
)

// These are conditional arithmetic inputs, not a sealed production manifest,
// observed chain workload, selected host or a sufficient resource profile.
// The example checks a real composition boundary: finite local quotations do
// not imply a representable SUM of complete admitted stage quotations.
func TestProviderHistoryWholeContentCostRejectsFiniteLocalOverflow(t *testing.T) {
	input := providerHistoryCollectionInput{
		blocks: 38420, native: 38420, effects: 4096, versions: 4096,
		presentStops: 10, windowBlocks: 4096, contentBlocks: 10,
	}
	counts, ok := providerHistoryCollectionMaximumCounts(input)
	if !ok || counts.ancestryWindows != 10 || counts.contentWindows != 10 ||
		counts.contentGroups != 3846 || counts.groups != 9226 || counts.requests != 162391 {
		t.Fatal("conditional whole-range grammar counts changed", counts, ok)
	}
	pair, ok := providerHistoryContentDecodePoolUpper(providerHistoryContentDecodePoolLimits{
		Windows: counts.contentWindows, Blocks: input.blocks, PendingBlocks: input.contentBlocks,
		EvidenceBytes: 64 << 20, ValueBytes: 8 << 20, Transactions: 1, Logs: 1,
	})
	const decoderTerm uint64 = 50665495807918080
	if !ok || pair.normal[10] < decoderTerm || pair.envelope != pair.normal {
		t.Fatal("finite decoder quotation omitted its complete original pool")
	}
	// Match the current source's whole-journey-first marginal. This is a
	// conditional cost model only; no derivation, owner or live meter is made.
	model := providerHistoryBoundaryRoleModel{groups: counts.groups}
	model.tiers[0] = providerHistoryBoundaryNormalTier{count: 1, work: pair.normal}
	model.tiers[1].count = counts.groups - 1
	if !model.valid() {
		t.Fatal("finite marginal inventory")
	}
	for _, prior := range []uint64{0, 1, counts.groups - 2, counts.groups - 1} {
		advance := prior+1 < counts.groups
		local, localOK := providerHistoryContentRoleComponentsUpper(counts.groups, prior, advance, pair, providerHistoryControlPair{})
		expected, expectedOK := model.roleUpper(prior, advance)
		if !localOK || !expectedOK || local != expected {
			t.Fatal("a local source cut unexpectedly refused before the whole sum", prior)
		}
	}
	// Independently calculate the positive lower term with unbounded integers.
	// An overflow of the complete sum must refuse, never wrap or return a prefix.
	minimum := new(big.Int).SetUint64(decoderTerm)
	minimum.Mul(minimum, new(big.Int).SetUint64(14*counts.groups-11))
	maximum := new(big.Int).SetUint64(^uint64(0))
	if minimum.String() != "6543600780080043786240" || minimum.Cmp(maximum) <= 0 {
		t.Fatal("independent whole-sum obstruction arithmetic")
	}
	if out, accepted := model.futureUpper(); accepted || out != (candidateWorkVector{}) {
		t.Fatal("finite local quotations escaped as an overflowed complete future")
	}
	// A finite control exercises the same whole-sum path. This is component
	// DATA, not evidence that even a one-stage operating profile is qualified.
	one := providerHistoryBoundaryRoleModel{groups: 1}
	one.tiers[0] = providerHistoryBoundaryNormalTier{count: 1, work: pair.normal}
	out, accepted := one.futureUpper()
	if !accepted {
		t.Fatal("representable control quotation refused")
	}
	for coordinate, units := range pair.normal {
		if out[coordinate] != 3*units {
			t.Fatal("one-stage complete closure quotation changed", coordinate)
		}
	}
}

func TestProviderHistoryWholeStorageQuoteMustFitOriginalPayloadDomain(t *testing.T) {
	for _, objectBytes := range []uint64{512 << 10, 32 << 20} {
		cost, ok := providerHistoryCollectionSuccessfulWriteUpper(9226, 162391, objectBytes, 256<<10, 4096)
		if !ok || cost.storage.payload <= 16<<30 {
			t.Fatal("conditional successful writer quotation", objectBytes, ok)
		}
		maximum := providerHistoryGroupTotals{
			work: candidateWorkVector{0: 1, 1: 1, 10: 1}, physical: cost.storage.physical,
			retainedRaw: cost.storage.raw, payload: cost.storage.payload, metadata: cost.storage.metadata,
			objects: cost.storage.objects, files: cost.storage.files, clocks: 1,
		}
		// Numeric constructor-shape control only. Nothing is installed in a
		// source, live store, clock, reservation or cost-derivation witness.
		if maximum.valid() {
			t.Fatal("finite writer arithmetic bypassed the original whole payload limit")
		}
		maximum.payload = 16 << 30
		if !maximum.valid() {
			t.Fatal("the payload guard was masked by another invalid numeric dimension")
		}
	}
}
