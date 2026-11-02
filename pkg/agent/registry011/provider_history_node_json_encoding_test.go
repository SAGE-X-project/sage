package registry011

import (
	"bytes"
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

func TestProviderHistoryNodeJSONBinaryEncodingUsesOriginalPreflight(t *testing.T) {
	txs := providerHistoryNodeJSONUnitTransactions()
	receipt := &types.Receipt{Status: 1, CumulativeGasUsed: 21000, Logs: []*types.Log{}}
	withdrawal := &types.Withdrawal{Index: 1, Validator: 2, Address: common.Address{1}, Amount: 3}
	block := types.NewBlockWithHeader(&types.Header{Number: big.NewInt(1), Difficulty: big.NewInt(0)}).WithBody(types.Body{Transactions: types.Transactions{txs[0]}, Withdrawals: types.Withdrawals{withdrawal}})
	for _, value := range []any{txs[0], receipt, withdrawal, block} {
		var want []byte
		var err error
		switch v := value.(type) {
		case *types.Transaction:
			want, err = v.MarshalBinary()
		case *types.Receipt:
			want, err = v.MarshalBinary()
		default:
			want, err = rlp.EncodeToBytes(v)
		}
		if err != nil || len(want) < 2 {
			t.Fatal("normal binary precursor", err)
		}
		for _, meter := range []*providerWorkMeter{nil, providerReplayUnitMeter()} {
			got, err := nodeJSONRLPObserved(value, len(want), meter)
			if err != nil || !bytes.Equal(got, want) {
				t.Fatal("same canonical encoding", err)
			}
		}
		m := providerHistoryContentUnitMeter(candidateWorkVector{})
		if got, err := nodeJSONRLPObserved(value, len(want), m); err == nil || got != nil || !m.failed {
			t.Fatal("unfunded encoder returned data")
		}
		m = providerReplayUnitMeter()
		if got, err := nodeJSONRLPObserved(value, len(want)-1, m); err == nil || got != nil || !m.failed {
			t.Fatal("derived output ceiling violated")
		}
	}
	for _, value := range []any{true, (*types.Transaction)(nil), (*types.Receipt)(nil), (*types.Block)(nil), (*types.Withdrawal)(nil)} {
		if got, err := nodeJSONRLPObserved(value, 100, providerReplayUnitMeter()); err == nil || got != nil {
			t.Fatal("unsupported typed encoding")
		}
	}
	for _, maximum := range []int{0, -1, (64 << 20) + 1} {
		if got, err := nodeJSONRLPObserved(receipt, maximum, providerReplayUnitMeter()); err == nil || got != nil {
			t.Fatal("binary source ceiling")
		}
	}
	negative := types.NewBlockWithHeader(&types.Header{Number: big.NewInt(1), Difficulty: big.NewInt(-1)})
	m := providerReplayUnitMeter()
	if got, err := nodeJSONRLPObserved(negative, 65536, m); err == nil || got != nil || !m.failed {
		t.Fatal("encoding failure must not publish partial bytes")
	}
}

func TestProviderHistoryNodeJSONRootsAndSelectedHeaderUseObservedBytes(t *testing.T) {
	txs := providerHistoryNodeJSONUnitTransactions()
	withdrawal := &types.Withdrawal{Index: 1, Validator: 2, Address: common.Address{1}, Amount: 3}
	withdrawals := types.Withdrawals{withdrawal}
	want := types.DeriveSha(withdrawals, trie.NewStackTrie(nil))
	h := &types.Header{Number: big.NewInt(1), Difficulty: big.NewInt(0), Time: 2, WithdrawalsHash: &want, TxHash: types.DeriveSha(types.Transactions{txs[0]}, trie.NewStackTrie(nil))}
	tx, err := txs[0].MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	w, err := rlp.EncodeToBytes(withdrawal)
	if err != nil {
		t.Fatal(err)
	}
	if !nodeJSONRootsObserved(context.Background(), [][]byte{tx}, [][]byte{w}, h, providerReplayUnitMeter()) {
		t.Fatal("pinned transaction/withdrawal roots")
	}
	if !nodeJSONSelectedObserved(h, replayHeader(h), providerReplayUnitMeter()) || !nodeJSONSelectedObserved(h, replayHeader(h), nil) {
		t.Fatal("positive selected header")
	}
	selected := replayHeader(h)
	selected.Number++
	if nodeJSONSelectedObserved(h, selected, providerReplayUnitMeter()) || nodeJSONSelectedObserved(nil, selected, providerReplayUnitMeter()) || nodeJSONSelectedObserved(&types.Header{}, selected, providerReplayUnitMeter()) {
		t.Fatal("invalid/mismatched selected header")
	}
	huge := *h
	huge.Number = new(big.Int).Lsh(big.NewInt(1), 65)
	if nodeJSONSelectedObserved(&huge, selected, providerReplayUnitMeter()) || nodeJSONSelectedObserved(h, replayHeader(h), providerHistoryContentUnitMeter(candidateWorkVector{})) {
		t.Fatal("number/encoder original guard")
	}
	if nodeJSONRootsObserved(context.Background(), [][]byte{tx}, [][]byte{w}, nil, providerReplayUnitMeter()) || nodeJSONRootsObserved(context.Background(), nil, nil, &types.Header{}, providerReplayUnitMeter()) {
		t.Fatal("missing typed root context")
	}
	bad := *h
	bad.TxHash = common.Hash{}
	if nodeJSONRootsObserved(context.Background(), [][]byte{tx}, [][]byte{w}, &bad, providerReplayUnitMeter()) {
		t.Fatal("transaction root mismatch")
	}
	bad = *h
	zero := common.Hash{}
	bad.WithdrawalsHash = &zero
	if nodeJSONRootsObserved(context.Background(), [][]byte{tx}, [][]byte{w}, &bad, providerReplayUnitMeter()) {
		t.Fatal("withdrawal root mismatch")
	}
	if nodeJSONRootsObserved(context.Background(), [][]byte{tx}, [][]byte{w}, h, providerHistoryContentUnitMeter(candidateWorkVector{})) {
		t.Fatal("root work unavailable")
	}
}

func TestProviderHistoryNodeJSONSliceAndScanGuardsUseConcreteCopySizes(t *testing.T) {
	m := providerReplayUnitMeter()
	if !nodeJSONScanObserved([]byte("normal"), m) || !nodeJSONScanObserved(nil, nil) || !nodeJSONSliceObserved(2, 3, m) || !nodeJSONSliceObserved(0, 1, nil) {
		t.Fatal("concrete positive primitives")
	}
	for _, c := range [][2]int{{-1, 1}, {65537, 1}, {1, 0}, {1, 4}} {
		if nodeJSONSliceObserved(c[0], c[1], providerReplayUnitMeter()) {
			t.Fatal("closed storage shape")
		}
	}
	if nodeJSONSliceObserved(1, 1, providerHistoryContentUnitMeter(candidateWorkVector{})) || nodeJSONScanObserved(nil, providerHistoryContentUnitMeter(candidateWorkVector{})) {
		t.Fatal("unfunded concrete primitive")
	}
}
