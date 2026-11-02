package registry011

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func providerHistoryNodeJSONUnitMeter() *providerWorkMeter {
	return providerHistoryContentUnitMeter(candidateWorkVector{2: 1 << 30, 3: 1 << 20, 4: 1 << 20, 6: 1 << 20, 10: 1 << 60})
}

func TestProviderHistoryNodeJSONObservedReaderKeepsDynamicPriceAndParityAlias(t *testing.T) {
	block, raw, _, _ := nodeJSONFixtureBytes(t)
	want, err := rlp.EncodeToBytes(block)
	if err != nil {
		t.Fatal("normal binary precursor", err)
	}
	index := -1
	for i, tx := range block.Transactions() {
		if tx.Type() >= types.DynamicFeeTxType {
			index = i
			break
		}
	}
	if index < 0 {
		t.Fatal("recorded dynamic-fee precursor")
	}
	tx := block.Transactions()[index]
	effective := new(big.Int).Add(block.BaseFee(), tx.GasTipCap())
	if effective.Cmp(tx.GasFeeCap()) > 0 {
		effective.Set(tx.GasFeeCap())
	}
	variant := func(price *big.Int, badParity bool) []byte {
		return nodeJSONEditObject(raw, func(fields map[string]json.RawMessage) {
			rows, ok := nodeRawArray(fields["transactions"], 16384)
			if !ok {
				t.Fatal("normal rows")
			}
			rows[index] = nodeJSONEditObject(rows[index], func(row map[string]json.RawMessage) {
				nodeJSONPut(row, "gasPrice", "0x"+price.Text(16))
				delete(row, "yParity")
				if badParity {
					v, _, _ := tx.RawSignatureValues()
					nodeJSONPut(row, "yParity", "0x"+new(big.Int).Xor(v, big.NewInt(1)).Text(16))
				}
			})
			nodeJSONPut(fields, "transactions", rows)
		})
	}
	limits := nodeTestLimits().Replay
	for _, tc := range []struct {
		name  string
		price *big.Int
	}{{"effective", effective}, {"cap", tx.GasFeeCap()}} {
		t.Run(tc.name, func(t *testing.T) {
			input := variant(tc.price, false)
			old, err := nodeJSONBlock(context.Background(), input, replayHeader(block.Header()), &replayBudget{limits: limits})
			if err != nil || !bytes.Equal(old, want) {
				t.Fatal("normal legacy price/explicit-v precursor", err)
			}
			m := providerHistoryNodeJSONUnitMeter()
			got, err := nodeJSONBlock(context.Background(), input, replayHeader(block.Header()), &replayBudget{limits: limits, observer: m})
			if err != nil || !bytes.Equal(got, want) || m.failed || m.actual[4] != 120 || m.actual[10] == 0 {
				t.Fatal("same nonnil full reader price/alias contract", err, m.actual)
			}
		})
	}
	for _, input := range [][]byte{variant(new(big.Int).Add(tx.GasFeeCap(), big.NewInt(1)), false), variant(effective, true)} {
		m := providerHistoryNodeJSONUnitMeter()
		if got, err := nodeJSONBlock(context.Background(), input, replayHeader(block.Header()), &replayBudget{limits: limits, observer: m}); err == nil || got != nil {
			t.Fatal("changed price or conflicting parity returned partial block")
		}
	}
}

func TestProviderHistoryNodeJSONObservationPreservesRecordedConsensusBytes(t *testing.T) {
	block, raw, receipts, originals := nodeJSONFixtureBytes(t)
	want, err := rlp.EncodeToBytes(block)
	if err != nil || len(block.Transactions()) != 120 {
		t.Fatal("recorded complete block precursor", err)
	}
	limits := nodeTestLimits().Replay
	legacy := &replayBudget{limits: limits}
	old, err := nodeJSONBlock(context.Background(), raw, replayHeader(block.Header()), legacy)
	if err != nil || !bytes.Equal(old, want) {
		t.Fatal("legacy normal JSON precursor", err)
	}
	m := providerHistoryNodeJSONUnitMeter()
	b := &replayBudget{limits: limits, observer: m}
	got, err := nodeJSONBlock(context.Background(), raw, replayHeader(block.Header()), b)
	if err != nil || !bytes.Equal(got, want) || b.bytes != legacy.bytes || m.failed || m.actual[3] != 1 || m.actual[4] != 120 || m.actual[2] < uint64(len(raw)) {
		t.Fatal("typed block JSON changed consensus bytes or original observation", err, m.actual, b.bytes, legacy.bytes)
	}
	legacy = &replayBudget{limits: limits}
	oldReceipts, err := nodeJSONReceipts(context.Background(), receipts, block, legacy)
	if err != nil || !reflect.DeepEqual(oldReceipts, originals) {
		t.Fatal("normal receipt JSON precursor", err)
	}
	b = &replayBudget{limits: limits, observer: m}
	gotReceipts, err := nodeJSONReceipts(context.Background(), receipts, block, b)
	if err != nil || !reflect.DeepEqual(gotReceipts, originals) || b.bytes != legacy.bytes || m.failed {
		t.Fatal("complete receipt bytes or original meter continuity", err, m.actual)
	}
	var logCount uint64
	for _, raw := range originals {
		var r types.Receipt
		if r.UnmarshalBinary(raw) != nil {
			t.Fatal("normal receipt")
		}
		logCount += uint64(len(r.Logs))
	}
	if m.actual[6] != logCount {
		t.Fatal("only entered typed receipt logs", m.actual, logCount)
	}
	for i, n := range m.actual {
		if i != 2 && i != 3 && i != 4 && i != 6 && i != 10 && n != 0 {
			t.Fatal("JSON DATA invented I/O or authority", m.actual)
		}
	}
}

func TestProviderHistoryNodeJSONObservationRefusesShortOriginalBudgetsWithoutPartialData(t *testing.T) {
	block, raw, receipts, _ := nodeJSONFixtureBytes(t)
	limits := nodeTestLimits().Replay
	for _, tc := range []struct {
		name     string
		receipts bool
	}{{"block", false}, {"receipts", true}} {
		t.Run(tc.name, func(t *testing.T) {
			probe := providerHistoryNodeJSONUnitMeter()
			if tc.receipts {
				if _, err := nodeJSONReceipts(context.Background(), receipts, block, &replayBudget{limits: limits, observer: probe}); err != nil {
					t.Fatal("receipt positive precursor", err)
				}
			} else {
				if _, err := nodeJSONBlock(context.Background(), raw, replayHeader(block.Header()), &replayBudget{limits: limits, observer: probe}); err != nil {
					t.Fatal("block positive precursor", err)
				}
			}
			for _, early := range []bool{false, true} {
				upper := candidateWorkVector{2: probe.actual[2] - 1, 3: 1 << 20, 4: 1 << 20, 6: 1 << 20, 10: 1 << 60}
				if early {
					upper = candidateWorkVector{}
				}
				m := providerHistoryContentUnitMeter(upper)
				b := &replayBudget{limits: limits, observer: m}
				if tc.receipts {
					got, err := nodeJSONReceipts(context.Background(), receipts, block, b)
					if err == nil || got != nil || !m.failed {
						t.Fatal("partial receipt output", err, m.actual)
					}
				} else {
					got, err := nodeJSONBlock(context.Background(), raw, replayHeader(block.Header()), b)
					if err == nil || got != nil || !m.failed {
						t.Fatal("partial block output", err, m.actual)
					}
				}
				if !early && m.actual[10] == 0 {
					t.Fatal("late refusal lost entered prefix")
				}
				before := m.actual
				if tc.receipts {
					_, _ = nodeJSONReceipts(context.Background(), receipts, block, b)
				} else {
					_, _ = nodeJSONBlock(context.Background(), raw, replayHeader(block.Header()), b)
				}
				if m.actual != before {
					t.Fatal("failed original observer resumed", before, m.actual)
				}
			}
		})
	}
}

func TestProviderHistoryNodeJSONObservedNestedListsKeepExistingClosedBounds(t *testing.T) {
	for _, tc := range []struct {
		name, raw string
		want      bool
	}{
		{"absent", `{}`, true}, {"null", `{"accessList":null}`, true},
		{"access", `{"accessList":[{"address":"0x0000000000000000000000000000000000000001","storageKeys":[]}]}`, true},
		{"access-missing", `{"accessList":[{"address":"0x1"}]}`, false},
		{"access-keys", `{"accessList":[{"address":"0x1","storageKeys":{}}]}`, false},
		{"blob", `{"blobVersionedHashes":["0x0000000000000000000000000000000000000000000000000000000000000001"]}`, true},
		{"blob-value", `{"blobVersionedHashes":["0x1"]}`, false},
		{"authorization", `{"authorizationList":[{"chainId":"0x1","address":"0x1","nonce":"0x0","yParity":"0x0","r":"0x0","s":"0x0"}]}`, true},
		{"authorization-missing", `{"authorizationList":[{"chainId":"0x1"}]}`, false},
		{"not-array", `{"accessList":{}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var fields map[string]json.RawMessage
			if json.Unmarshal([]byte(tc.raw), &fields) != nil {
				t.Fatal("literal DATA fixture")
			}
			m := providerHistoryNodeJSONUnitMeter()
			if old, got := nodeJSONTransactionLists(fields), nodeJSONTransactionListsObserved(fields, m); old != tc.want || got != old {
				t.Fatal("closed nested-list contract", old, got, tc.want)
			}
		})
	}
	m := providerHistoryContentUnitMeter(candidateWorkVector{})
	if nodeJSONTransactionListsObserved(map[string]json.RawMessage{"accessList": json.RawMessage("null")}, m) || !m.failed {
		t.Fatal("failed optional/null comparison accepted")
	}
}

func TestProviderHistoryNodeJSONObservationKeepsDuplicateContextAndLocationGuards(t *testing.T) {
	limits := nodeTestLimits().Replay
	for _, raw := range []string{`{"a":1,"a":2}`, `{"a":"\ud800"}`, `{`, `{} {}`} {
		if nodeJSONPreflight(context.Background(), []byte(raw), &replayBudget{limits: limits, observer: providerHistoryNodeJSONUnitMeter()}) {
			t.Fatal("invalid original JSON", raw)
		}
	}
	if nodeJSONPreflight(nil, []byte(`{}`), &replayBudget{limits: limits, observer: providerHistoryNodeJSONUnitMeter()}) || nodeJSONPreflight(context.Background(), nil, nil) {
		t.Fatal("missing original context/budget")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if nodeJSONPreflight(ctx, []byte(`{}`), &replayBudget{limits: limits, observer: providerHistoryNodeJSONUnitMeter()}) {
		t.Fatal("cancelled original context")
	}
	block, raw, _, _ := nodeJSONFixtureBytes(t)
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		t.Fatal("normal block DATA")
	}
	rows, ok := nodeRawArray(fields["transactions"], 16384)
	if !ok || len(rows) != 120 {
		t.Fatal("normal complete rows")
	}
	var transaction map[string]json.RawMessage
	if json.Unmarshal(rows[0], &transaction) != nil {
		t.Fatal("normal row")
	}
	if !nodeJSONLocationObserved(transaction, block.Hash(), block.NumberU64(), block.Transactions()[0].Hash(), 0, providerHistoryNodeJSONUnitMeter()) {
		t.Fatal("positive location precursor")
	}
	if nodeJSONLocationObserved(transaction, block.Hash(), block.NumberU64(), block.Transactions()[0].Hash(), 1, providerHistoryNodeJSONUnitMeter()) {
		t.Fatal("wrong ordinal")
	}
	if got, err := nodeJSONTransactionObserved(rows[0], nil, 0, providerHistoryNodeJSONUnitMeter()); err == nil || got != nil {
		t.Fatal("missing typed header")
	}
	if got, err := nodeJSONTransactionObserved(rows[0], block.Header(), 0, providerHistoryContentUnitMeter(candidateWorkVector{})); err == nil || got != nil {
		t.Fatal("unfunded typed transaction")
	}
}
