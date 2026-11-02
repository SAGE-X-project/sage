package registry011

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestProviderHistoryContentDecodeMatchesIndependentWholeRows(t *testing.T) {
	for _, tc := range []struct {
		limits providerHistoryContentDecodeLimits
		units  uint64
	}{
		{providerHistoryContentDecodeLimits{1, 1, 1, 1}, 3411978},
		{providerHistoryContentDecodeLimits{1024, 128, 2, 2}, 10006329},
		{providerHistoryContentDecodeLimits{65536, 1024, 1, 1}, 12890673123},
		{providerHistoryContentDecodeLimits{1048576, 2048, 32, 64}, 206305356621},
		{providerHistoryContentDecodeLimits{67108864, 8388608, 16384, 65536}, 1688908175081613},
		{providerHistoryContentDecodeLimits{67108864, 67108864, 16384, 65536}, 13510857196929165},
	} {
		want := candidateWorkVector{2: tc.limits.EvidenceBytes, 3: 1, 4: tc.limits.Transactions, 6: tc.limits.Logs, 10: tc.units}
		got, ok := providerHistoryContentDecodeUpper(tc.limits)
		if !ok || got.normal != want || got.envelope != want {
			t.Fatal("independent complete content-decode row", tc, got, want, ok)
		}
	}
}

func TestProviderHistoryContentDecodeRejectsInvalidBoundData(t *testing.T) {
	for _, p := range []providerHistoryContentDecodeLimits{
		{0, 1, 1, 1}, {(64 << 20) + 1, 1, 1, 1}, {1024, 0, 1, 1}, {1024, 1025, 1, 1},
		{1024, 1, 0, 1}, {1024, 1, 16385, 1}, {1024, 1, 1, 0}, {1024, 1, 1, 65537},
		{^uint64(0), 1, 1, 1}, {1024, ^uint64(0), 1, 1}, {1024, 1, ^uint64(0), 1}, {1024, 1, 1, ^uint64(0)},
	} {
		if got, ok := providerHistoryContentDecodeUpper(p); ok || got != (providerHistoryControlPair{}) {
			t.Fatal("invalid content-decode DATA published partial work", p, got, ok)
		}
	}
}

// Original byte lengths and independently decoded receipt counts precede the
// target observer. These private fixtures are not authenticated chain evidence.
func historyContentDecodeSelection(t *testing.T, raw ReplayBlock) (providerHistoryControlPair, ReplayLimits, historyBinaryCoreShape) {
	t.Helper()
	shape := historyBinaryCoreOriginalShape(t, raw)
	p := providerHistoryContentDecodeLimits{shape.bytes, max(1, shape.maximumValue), max(1, shape.transactions), max(1, shape.logs)}
	cost, ok := providerHistoryContentDecodeUpper(p)
	if !ok {
		t.Fatal("finite complete content-decode precursor")
	}
	limits := replayTestLimits()
	limits.MaxEvidenceBytes, limits.MaxTransactions, limits.MaxLogs = int(p.EvidenceBytes), int(p.Transactions), int(p.Logs)
	return cost, limits, shape
}

func TestProviderHistoryContentDecodeFundsWholeOriginalBlocks(t *testing.T) {
	for _, raw := range replayTestInput(t).Blocks[:3] {
		cost, limits, shape := historyContentDecodeSelection(t, raw)
		m := providerHistoryContentUnitMeter(cost.normal)
		b := replayBudget{limits: limits, observer: m}
		out, err := replayDecodeContent(context.Background(), raw, &b)
		if err != nil || m.failed || m.clocks != 0 || out.header == nil || len(out.txs) != len(raw.Transactions) || len(out.receipts) != len(raw.Receipts) ||
			b.bytes != int(shape.bytes) || b.txs != int(shape.transactions) || b.logs != int(shape.logs) ||
			m.actual[2] != shape.bytes || m.actual[3] != 1 || m.actual[4] != shape.transactions || m.actual[6] != shape.logs {
			t.Fatal("whole original content-decode left its selected DATA", err, m.actual, b.bytes, b.txs, b.logs)
		}
		historyEventQueueWithin(t, m.actual, cost.normal)
	}
}

func TestProviderHistoryContentDecodeKeepsWholeWorkOnLastRootMismatch(t *testing.T) {
	for _, original := range replayTestInput(t).Blocks[:3] {
		cost, limits, _ := historyContentDecodeSelection(t, original)
		m := providerHistoryContentUnitMeter(cost.normal)
		b := replayBudget{limits: limits, observer: m}
		if _, err := replayDecodeContent(context.Background(), original, &b); err != nil || m.failed {
			t.Fatal("complete original reference content", err, m.actual)
		}
		var header types.Header
		if rlp.DecodeBytes(original.HeaderRLP, &header) != nil {
			t.Fatal("canonical private header precursor")
		}
		header.ReceiptHash[0] ^= 1
		raw := original
		var err error
		raw.HeaderRLP, err = rlp.EncodeToBytes(&header)
		if err != nil || len(raw.HeaderRLP) != len(original.HeaderRLP) {
			t.Fatal("same-sized final root mismatch precursor", err)
		}
		failed := providerHistoryContentUnitMeter(cost.normal)
		budget := replayBudget{limits: limits, observer: failed}
		out, err := replayDecodeContent(context.Background(), raw, &budget)
		if err == nil || failed.failed || failed.clocks != 0 || out.header == nil || len(out.txs) != len(raw.Transactions) ||
			failed.actual != m.actual || budget.bytes != b.bytes || budget.txs != b.txs || budget.logs != b.logs {
			t.Fatal("last receipt-root mismatch lost entered work", err, failed.actual, m.actual)
		}
		historyEventQueueWithin(t, failed.actual, cost.normal)
	}
}

func TestProviderHistoryContentDecodeRetainsMalformedReceiptPrefix(t *testing.T) {
	raw := replayTestInput(t).Blocks[0]
	cost, limits, _ := historyContentDecodeSelection(t, raw)
	// The valid header and transaction are decoded before this finite malformed
	// one-byte receipt. No trie or receipt log count has been entered yet.
	raw.Receipts = [][]byte{{0xc0}}
	h, tx := uint64(len(raw.HeaderRLP)), uint64(len(raw.Transactions[0]))
	want := candidateWorkVector{2: h + tx + 1, 3: 1, 4: 1, 10: 2*h*h + 2*h + 3 + 2*tx*tx + 2*tx + 3 + 2}
	m := providerHistoryContentUnitMeter(cost.normal)
	b := replayBudget{limits: limits, observer: m}
	out, err := replayDecodeContent(context.Background(), raw, &b)
	if err == nil || m.failed || m.clocks != 0 || out.header == nil || len(out.txs) != 1 || out.txs[0] != nil || out.receipts[0] != nil ||
		m.actual != want || b.bytes != int(h+tx+1) || b.txs != 1 || b.logs != 0 {
		t.Fatal("malformed receipt lost original decode prefix", err, m.actual, want)
	}
	historyEventQueueWithin(t, m.actual, cost.normal)
}

func TestProviderHistoryContentDecodeRetainsCanonicalTypeMismatchPrefix(t *testing.T) {
	raw := replayTestInput(t).Blocks[0]
	var receipt types.Receipt
	if receipt.UnmarshalBinary(raw.Receipts[0]) != nil {
		t.Fatal("canonical private receipt precursor")
	}
	if receipt.Type == types.LegacyTxType {
		receipt.Type = types.AccessListTxType
	} else {
		receipt.Type = types.LegacyTxType
	}
	var err error
	raw.Receipts = make([][]byte, 1)
	raw.Receipts[0], err = receipt.MarshalBinary()
	if err != nil {
		t.Fatal("different canonical receipt type precursor", err)
	}
	cost, limits, shape := historyContentDecodeSelection(t, raw)
	want := shape.work
	want[6], want[10] = 0, want[10]-130 // Neither trie nor root comparison is entered.
	m := providerHistoryContentUnitMeter(cost.normal)
	b := replayBudget{limits: limits, observer: m}
	out, err := replayDecodeContent(context.Background(), raw, &b)
	if err == nil || m.failed || m.clocks != 0 || out.header == nil || out.txs[0] != nil || out.receipts[0] != nil ||
		m.actual != want || b.bytes != int(shape.bytes) || b.logs != 0 {
		t.Fatal("canonical type mismatch lost earlier canonical work", err, m.actual, want)
	}
	historyEventQueueWithin(t, m.actual, cost.normal)
}

func TestProviderHistoryContentDecodeCannotRenewOriginalEvidenceOrCounts(t *testing.T) {
	raw := replayTestInput(t).Blocks[0]
	cost, limits, shape := historyContentDecodeSelection(t, raw)
	m := providerHistoryContentUnitMeter(cost.normal)
	b := replayBudget{limits: limits, observer: m}
	if _, err := replayDecodeContent(context.Background(), raw, &b); err != nil || m.failed {
		t.Fatal("first complete private content precursor", err)
	}
	before := m.actual
	for range 2 {
		out, err := replayDecodeContent(context.Background(), raw, &b)
		if err == nil || out.header != nil || out.txs != nil || out.receipts != nil || m.failed || m.actual != before ||
			b.bytes != int(shape.bytes) || b.txs != int(shape.transactions) || b.logs != int(shape.logs) || b.limits != limits {
			t.Fatal("exhausted original evidence/count budget renewed", err, m.actual, before)
		}
	}
}
