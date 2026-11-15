package registry011

import (
	"bytes"
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func providerHistoryContentDecodeUnitMeter() *providerWorkMeter {
	return providerHistoryContentUnitMeter(candidateWorkVector{2: 1 << 30, 3: 4096, 4: 16384, 6: 65536, 10: 1 << 60})
}

func TestProviderReplayContentObservationPreservesCanonicalValues(t *testing.T) {
	in := replayTestInput(t)
	for i, raw := range in.Blocks {
		m := providerHistoryContentDecodeUnitMeter()
		observed := replayBudget{limits: replayTestLimits(), observer: m}
		got, err := replayDecodeContent(context.Background(), raw, &observed)
		legacy := replayBudget{limits: replayTestLimits()}
		want, legacyErr := replayDecodeContent(context.Background(), raw, &legacy)
		if err != nil || legacyErr != nil || got.header.Hash() != want.header.Hash() || len(got.txs) != len(want.txs) || len(got.receipts) != len(want.receipts) {
			t.Fatal("normal content differs from the original decoder", i, err, legacyErr)
		}
		decodedBytes := len(raw.HeaderRLP)
		logs := 0
		for j, tx := range got.txs {
			b, txErr := tx.MarshalBinary()
			r, receiptErr := got.receipts[j].MarshalBinary()
			if txErr != nil || receiptErr != nil || !bytes.Equal(b, raw.Transactions[j]) || !bytes.Equal(r, raw.Receipts[j]) {
				t.Fatal("canonical transaction or receipt changed", i, j)
			}
			decodedBytes += len(raw.Transactions[j]) + len(raw.Receipts[j])
			logs += len(got.receipts[j].Logs)
		}
		if m.failed || m.actual[2] != uint64(decodedBytes) || m.actual[3] != 1 || m.actual[4] != uint64(len(raw.Transactions)) || m.actual[6] != uint64(logs) || m.actual[10] <= uint64(decodedBytes) || observed.bytes != legacy.bytes {
			t.Fatal("actual primitive/object counters or legacy evidence accounting", i, m.actual)
		}
		for _, index := range []int{0, 1, 5, 7, 8, 9} {
			if m.actual[index] != 0 {
				t.Fatal("content decoder invented network, trace, proof or disk work", i, m.actual)
			}
		}
	}
}

func TestProviderReplayContentObservationUsesRecordedConsensusRoots(t *testing.T) {
	block, _, _, receipts := nodeJSONFixtureBytes(t)
	header, err := rlp.EncodeToBytes(block.Header())
	if err != nil {
		t.Fatal(err)
	}
	raw := ReplayBlock{HeaderRLP: header, Receipts: receipts, Transactions: make([][]byte, len(block.Transactions()))}
	for i, tx := range block.Transactions() {
		raw.Transactions[i], err = tx.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
	}
	m := providerHistoryContentDecodeUnitMeter()
	budget := replayBudget{limits: replayTestLimits(), observer: m}
	got, err := replayDecodeContent(context.Background(), raw, &budget)
	if err != nil || got.header.Hash() != block.Hash() || len(got.txs) != 120 || m.actual[4] != 120 {
		t.Fatal("recorded deployment roots", err, m.actual)
	}
	// Content equality still cannot substitute for the required post-fork
	// native gas witness. No RPC, normalizer or gas-policy fallback is added.
	strict := replayBudget{limits: replayTestLimits(), observer: providerHistoryContentDecodeUnitMeter()}
	if _, err := replayDecodeBlock(context.Background(), raw, &strict); err == nil {
		t.Fatal("content observation bypassed the pinned native gas boundary")
	}
}

func TestProviderReplayContentObservationOriginalAllowancesNeverReset(t *testing.T) {
	raw := replayTestInput(t).Blocks[0]
	probe := providerHistoryContentDecodeUnitMeter()
	budget := replayBudget{limits: replayTestLimits(), observer: probe}
	if _, err := replayDecodeContent(context.Background(), raw, &budget); err != nil {
		t.Fatal("normal precursor", err)
	}
	for _, index := range []int{2, 3, 4, 6, 10} {
		if probe.actual[index] == 0 {
			t.Fatal("missing counted processing coordinate", index)
		}
		upper := probe.actual
		upper[index]--
		m := providerHistoryContentUnitMeter(upper)
		b := replayBudget{limits: replayTestLimits(), observer: m}
		if _, err := replayDecodeContent(context.Background(), raw, &b); err == nil || !m.failed {
			t.Fatal("insufficient original allowance produced content", index, err)
		}
		before := m.actual
		if _, err := replayDecodeContent(context.Background(), raw, &b); err == nil || m.actual != before {
			t.Fatal("failed processing could reuse or refund its meter", index)
		}
	}
	exact := providerHistoryContentUnitMeter(probe.actual)
	b := replayBudget{limits: replayTestLimits(), observer: exact}
	if _, err := replayDecodeContent(context.Background(), raw, &b); err != nil || exact.actual != probe.actual || exact.failed {
		t.Fatal("exact measured precursor allowance", err, exact.actual, probe.actual)
	}
}

func TestProviderReplayContentObservationCountsOnlyEnteredTransactions(t *testing.T) {
	raw := replayTestInput(t).Blocks[0]
	raw.Transactions = [][]byte{{1}, raw.Transactions[0]}
	raw.Receipts = [][]byte{raw.Receipts[0], raw.Receipts[0]}
	m := providerHistoryContentDecodeUnitMeter()
	b := replayBudget{limits: replayTestLimits(), observer: m}
	if _, err := replayDecodeContent(context.Background(), raw, &b); err == nil || m.actual[4] != 1 || m.actual[6] != 0 || m.actual[2] != uint64(len(raw.HeaderRLP)+1) {
		t.Fatal("unentered second transaction or receipt was reported as actual", err, m.actual)
	}
	// Evidence admission reserves the complete input independently; this is
	// deliberately distinct from the actual entered-processing counters.
	if b.txs != 2 {
		t.Fatal("legacy complete-input admission changed")
	}
}

func TestProviderReplayObservationRefusesPrimitiveOverrun(t *testing.T) {
	m := providerHistoryContentDecodeUnitMeter()
	b := replayBudget{limits: replayTestLimits(), observer: m}
	if !b.encodePreflight(1) || b.encodedObserved([]byte{1, 2}, 1) || !m.failed || m.actual != (candidateWorkVector{}) {
		t.Fatal("encoding maximum became a reported actual or overrun remained reusable", m.actual)
	}
	var missing *replayBudget
	if missing.decodeObserved(nil) || missing.encodePreflight(1) || missing.encodedObserved(nil, 1) || missing.equalObserved(nil, nil) || missing.blockObserved() || missing.transactionObserved() || missing.logsObserved(0) {
		t.Fatal("missing replay budget supplied an observation")
	}
	legacy := replayBudget{limits: replayTestLimits()}
	if !legacy.decodeObserved([]byte{1}) || !legacy.blockObserved() || !legacy.transactionObserved() || !legacy.logsObserved(0) || legacy.logsObserved(-1) || legacy.encodePreflight(0) || legacy.encodePreflight(legacy.limits.MaxEvidenceBytes+1) {
		t.Fatal("legacy DATA hooks or closed encoding bound")
	}
	// Use a normal empty pre-fork body to exercise the empty commitment through
	// the actual decoder rather than only through a standalone trie helper.
	header := &types.Header{Number: new(big.Int), Difficulty: new(big.Int), TxHash: types.EmptyRootHash, ReceiptHash: types.EmptyRootHash, UncleHash: types.EmptyUncleHash}
	encoded, err := rlp.EncodeToBytes(header)
	if err != nil {
		t.Fatal(err)
	}
	empty := replayBudget{limits: replayTestLimits(), observer: providerHistoryContentDecodeUnitMeter()}
	if _, err := replayDecodeContent(context.Background(), ReplayBlock{HeaderRLP: encoded}, &empty); err != nil {
		t.Fatal("empty content commitments", err)
	}
}
