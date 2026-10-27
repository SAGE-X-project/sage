package registry011

import (
	"context"
	"math"
	"testing"
)

func TestProviderIntegrationHistoryCapacityPreservesLifetimeCap(t *testing.T) {
	base := FinalizedHeader{admissionTestDigest("a"), admissionTestDigest("b"), sepoliaDeploymentBlock, 1}
	selected := base
	selected.Number += 26791 // The retained old actual interval, as inert data.
	bounds := CandidateLedgerBounds{4096, 16 << 20, 64 << 20}
	if providerHistoryCapacity(bounds, [3]uint64{}, base, selected) {
		t.Fatal("long deployed history fit a reset or expanded ledger")
	}
	selected.Number = base.Number + 2
	if !providerHistoryCapacity(bounds, [3]uint64{1, 1, 1}, base, selected) {
		t.Fatal("small bounded interval")
	}
	for _, counts := range [][3]uint64{{4096, 0, 0}, {math.MaxUint64, 1, 0}, {4090, 0, 0}} {
		if providerHistoryCapacity(bounds, counts, base, selected) {
			t.Fatal("used claims or future local work disappeared")
		}
	}
	selected.Number = base.Number - 1
	if providerHistoryCapacity(bounds, [3]uint64{}, base, selected) || providerHistoryCapacity(CandidateLedgerBounds{}, [3]uint64{}, base, base) {
		t.Fatal("unsupported history bounds")
	}
	selected.Number = math.MaxUint64
	if providerHistoryCapacity(bounds, [3]uint64{}, base, selected) {
		t.Fatal("record lower-bound overflow")
	}
}

func TestProviderIntegrationCollectionScopeAndModeRefuseBeforeReads(t *testing.T) {
	for _, reads := range []*PinnedAgentReads{nil, {}, {rpc: &ReadRPC{}}, {rpc: &ReadRPC{acquisition: &rpcAcquisition{}}}} {
		if providerCollectionScope(reads) {
			t.Fatal("non-owned client became a provider scope")
		}
		if _, err := collectSourceBatchMode(context.Background(), reads, JournalConfig{}, JournalCheckpoint{}, SourceReadLimits{}, nil, nodeIdentity{}, sourceProviderStateGas); err == nil {
			t.Fatal("provider collection started without an owned scope")
		}
	}
	if _, err := collectSourceBatchMode(nil, nil, JournalConfig{}, JournalCheckpoint{}, SourceReadLimits{}, nil, nodeIdentity{}, sourceAcquisitionMode(255)); err == nil {
		t.Fatal("unknown acquisition mode")
	}
}
