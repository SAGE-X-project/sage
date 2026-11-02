package registry011

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestProviderHistoryRootGasNormalizationPoolMatchesIndependentRows(t *testing.T) {
	w := uint64(strconv.IntSize / 8)
	s := uint64(reflect.TypeFor[types.Transaction]().Size()) + uint64(reflect.TypeFor[types.BlobTx]().Size()) + 2
	for _, row := range []struct{ c, q, v, h, count, decoded, normal, envelope, word, sidecar uint64 }{
		{4, 1, 1, 1, 1, 193, 104523, 170573, 12, 1},
		{1024, 2048, 1024, 2048, 2, 8514, 14925437, 20233854, 24, 2},
		{8192, 4096, 2048, 67108864, 40, 71976, 647180645, 4503634634355046, 480, 40},
		{8388608, 8388608, 8388608, 67108864, 16384, 69746688, 809244682027013, 5312878669201414, 196608, 16384},
		{8388608, 67108864, 67108864, 67108864, 16384, 69746688, 9922012320301061, 14425646307475462, 196608, 16384},
	} {
		got, ok := providerHistoryRootGasNormalizationPoolUpper(row.c, row.q, row.v, row.h, row.count)
		want := providerHistoryRootGasNormalizationCost{
			normal:   candidateWorkVector{2: row.decoded, 10: row.normal + row.word*w + row.sidecar*s},
			envelope: candidateWorkVector{2: row.decoded, 10: row.envelope + row.word*w + row.sidecar*s},
		}
		if !ok || got != want {
			t.Fatal("independent matched whole-normalizer rows", row, got, want, ok)
		}
	}
}

func TestProviderHistoryRootGasNormalizationPoolRejectsUnboundNumbers(t *testing.T) {
	for _, row := range []struct{ c, q, v, h, count uint64 }{
		{0, 1024, 512, 1024, 1}, {3, 1024, 512, 1024, 1}, {(8 << 20) + 1, 1024, 512, 1024, 1},
		{1024, 0, 512, 1024, 1}, {1024, (64 << 20) + 1, 512, 1024, 1},
		{1024, 1024, 0, 1024, 1}, {1024, 1024, 1025, 2048, 1}, {1024, 2048, 1025, 1024, 1},
		{1024, 1024, 512, 0, 1}, {1024, 1024, 512, (64 << 20) + 1, 1},
		{1024, 1024, 512, 1024, 0}, {1024, 1024, 512, 1024, 16385},
		{^uint64(0), 1024, 512, 1024, 1}, {1024, ^uint64(0), 512, 1024, 1},
		{1024, 1024, ^uint64(0), 1024, 1}, {1024, 1024, 512, ^uint64(0), 1}, {1024, 1024, 512, 1024, ^uint64(0)},
	} {
		if got, ok := providerHistoryRootGasNormalizationPoolUpper(row.c, row.q, row.v, row.h, row.count); ok || got != (providerHistoryRootGasNormalizationCost{}) {
			t.Fatal("invalid bounds published a partial pair", row, got, ok)
		}
	}
}

// The library encodes these finite existing DATA values. They are not a
// consensus-valid block, authenticated provider, accepted settlement or Source.
func TestProviderHistoryRootGasCanonicalPoolIncludesEveryOriginalIndex(t *testing.T) {
	txs := providerHistoryNodeJSONUnitTransactions()
	header := &types.Header{Number: big.NewInt(1), Difficulty: new(big.Int)}
	body := providerHistoryNativeBlockBody{Header: header, Txs: txs}
	raw, err := rlp.EncodeToBytes(&body)
	if err != nil {
		t.Fatal("finite block encoding precursor", err)
	}
	var decoded types.Block
	var fields, rows []rlp.RawValue
	if rlp.DecodeBytes(raw, &decoded) != nil || rlp.DecodeBytes(raw, &fields) != nil || len(fields) < 3 || rlp.DecodeBytes(fields[1], &rows) != nil || len(rows) != len(txs) {
		t.Fatal("complete canonical transaction-list precursor")
	}
	canonical, err := rlp.EncodeToBytes(&decoded)
	if err != nil || !bytes.Equal(raw, canonical) || len(decoded.Transactions()) != len(txs) {
		t.Fatal("canonical block round trip", err)
	}
	var sum uint64
	for i, tx := range decoded.Transactions() {
		standalone, err := tx.MarshalBinary()
		if err != nil || len(standalone) == 0 || len(tx.Data()) > len(standalone) {
			t.Fatal("complete standalone transaction", i, err)
		}
		if tx.Type() == types.LegacyTxType {
			if !bytes.Equal(standalone, rows[i]) {
				t.Fatal("legacy transaction is not its original embedded list", i)
			}
		} else {
			var typed []byte
			if rlp.DecodeBytes(rows[i], &typed) != nil || !bytes.Equal(typed, standalone) || len(rows[i]) <= len(standalone) {
				t.Fatal("typed transaction lost its original outer RLP string", i)
			}
		}
		sum += uint64(len(standalone))
	}
	if sum == 0 || sum >= uint64(len(raw)) {
		t.Fatal("standalone transactions exceeded their shared original block bytes", sum, len(raw))
	}
}

func TestProviderHistoryRootGasPoolNativeActualsMatchConsensusEncodings(t *testing.T) {
	txs := providerHistoryNodeJSONUnitTransactions()
	var pool, maximum uint64
	for _, tx := range txs {
		raw, err := tx.MarshalBinary()
		if err != nil {
			t.Fatal("complete original transaction precursor", err)
		}
		pool += uint64(len(raw))
		maximum = max(maximum, uint64(len(raw)))
	}
	cost, ok := providerHistoryRootGasNormalizationPoolUpper(4096, pool, maximum, 64<<20, uint64(len(txs)))
	if !ok {
		t.Fatal("finite canonical byte-pool precursor")
	}
	m := providerHistoryContentUnitMeter(cost.envelope)
	var units uint64
	for i, tx := range txs {
		raw, err := tx.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		hash, hashOK := nodeTransactionHashObserved(tx, m)
		signing, signingErr := nodeSigningHashObserved(tx, m)
		var signer types.Signer = types.LatestSignerForChainID(big.NewInt(11155111))
		if tx.Type() == types.LegacyTxType && !tx.Protected() {
			signer = types.HomesteadSigner{}
		}
		x := uint64(len(raw))
		y := uint64(len(historyNodeBindingPreimage(t, tx)))
		units += x*x + x + 2 + y*y + y + 2
		if tx.Type() != types.LegacyTxType {
			units += y + 3
		}
		if !hashOK || signingErr != nil || hash != tx.Hash() || signing != signer.Hash(tx) || m.failed || m.actual != (candidateWorkVector{10: units}) || m.clocks != 0 {
			t.Fatal("unconsumed native preflight was charged or encodings changed", i, signingErr, m.actual, units)
		}
	}
	historyEventQueueWithin(t, m.actual, cost.normal)
	if cost.envelope[10]-cost.normal[10] != 4503633987174401 {
		t.Fatal("native room was multiplied by the number of rows", cost)
	}
}

func historyRootGasPoolQuote(t *testing.T, raw []byte, block replayDecodedBlock, maximum int) providerHistoryRootGasNormalizationCost {
	t.Helper()
	var pool, largest uint64
	for _, tx := range block.txs {
		canonical, err := tx.MarshalBinary()
		if err != nil || len(canonical) == 0 {
			t.Fatal("finite private canonical transaction precursor", err)
		}
		pool += uint64(len(canonical))
		largest = max(largest, uint64(len(canonical)))
	}
	// Repeated private fixture rows each consume their own bytes in Q. Empty
	// arrays keep a positive original cap; never replace it with actual count0.
	cost, ok := providerHistoryRootGasNormalizationPoolUpper(max(4, uint64(len(raw))), max(1, pool), max(1, largest), 64<<20, uint64(maximum))
	if !ok {
		t.Fatal("finite private matched normalization quote")
	}
	return cost
}

func TestProviderHistoryRootGasPoolFundsOriginalSuccessfulNormalizers(t *testing.T) {
	for _, mode := range []string{"one", "two", "execution", "escaped-hash"} {
		dimension, wantProfile := "regularGasUsed", SepoliaAmsterdamRootRegularGasProfile
		if mode == "execution" {
			dimension, wantProfile = "executionGasUsed", SepoliaAmsterdamRootExecutionGasProfile
		}
		fields, tx := historyRootGasNormalizationFields(t, dimension)
		rows := []map[string]json.RawMessage{fields}
		if mode == "two" {
			rows = append(rows, fields)
		}
		raw, block := historyRootGasNormalizationTrace(t, rows, tx, mode == "escaped-hash")
		before := bytes.Clone(raw)
		cost := historyRootGasPoolQuote(t, raw, block, len(rows))
		m := providerHistoryContentUnitMeter(cost.envelope)
		limits := replayTestLimits()
		limits.MaxTransactions = len(rows)
		b := replayBudget{limits: limits, observer: m}
		legacy := replayBudget{limits: limits}
		want, expected, oldErr := nodeRootGas(context.Background(), raw, block, &legacy)
		profile, got, err := nodeRootGas(context.Background(), raw, block, &b)
		if err != nil || oldErr != nil || profile != wantProfile || profile != want || !bytes.Equal(got, expected) || len(got) > len(raw) || !bytes.Equal(raw, before) || m.failed || m.clocks != 0 || b.bytes != len(raw) {
			t.Fatal("same whole-normalizer pair changed original output or ownership", mode, err, oldErr, m.actual)
		}
		historyEventQueueWithin(t, m.actual, cost.normal)
	}
}

func TestProviderHistoryRootGasPoolRetainsEmptyRawArraySlots(t *testing.T) {
	w := uint64(strconv.IntSize / 8)
	for _, tc := range []struct {
		raw              string
		decoded, logical uint64
	}{
		{"[]", 4, 17}, {" [ ] ", 10, 59 + 3*w},
	} {
		raw := []byte(tc.raw)
		cost := historyRootGasPoolQuote(t, raw, replayDecodedBlock{}, 1)
		m := providerHistoryContentUnitMeter(cost.envelope)
		limits := replayTestLimits()
		limits.MaxTransactions = 1
		b := replayBudget{limits: limits, observer: m}
		profile, got, err := nodeRootGas(context.Background(), raw, replayDecodedBlock{}, &b)
		if err != nil || profile != SepoliaAmsterdamRootRegularGasProfile || !bytes.Equal(got, []byte("[]")) || m.failed || m.actual != (candidateWorkVector{2: tc.decoded, 10: tc.logical}) || m.clocks != 0 {
			t.Fatal("empty original array lost allocated raw slots", tc.raw, err, m.actual)
		}
		historyEventQueueWithin(t, m.actual, cost.normal)
	}
}

func TestProviderHistoryRootGasPoolKeepsEarlierWorkOnLaterRowRefusal(t *testing.T) {
	var previous candidateWorkVector
	var originalLength int
	for _, badIndex := range []int{0, 1} {
		first, tx := historyRootGasNormalizationFields(t, "regularGasUsed")
		second, _ := historyRootGasNormalizationFields(t, "regularGasUsed")
		rows := []map[string]json.RawMessage{first, second}
		rows[badIndex]["gasRefund"] = historyRootContextString(t, "0x"+strings.Repeat("f", 64))
		raw, block := historyRootGasNormalizationTrace(t, rows, tx, false)
		cost := historyRootGasPoolQuote(t, raw, block, 2)
		m := providerHistoryContentUnitMeter(cost.envelope)
		limits := replayTestLimits()
		limits.MaxTransactions = 2
		b := replayBudget{limits: limits, observer: m}
		profile, got, err := nodeRootGas(context.Background(), raw, block, &b)
		if err == nil || profile != "" || got != nil || m.failed || m.clocks != 0 {
			t.Fatal("ordinary terminal row changed the original observer", badIndex, err, m.actual)
		}
		if badIndex == 0 {
			previous, originalLength = m.actual, len(raw)
		} else if len(raw) != originalLength || m.actual[2] <= previous[2] || m.actual[10] <= previous[10] {
			t.Fatal("later row erased earlier native or raw observations", m.actual, previous)
		}
		historyEventQueueWithin(t, m.actual, cost.normal)
	}
}

func TestProviderHistoryRootGasPoolCannotReplaceOrRenewOriginalNativePreflight(t *testing.T) {
	fields, tx := historyRootGasNormalizationFields(t, "regularGasUsed")
	raw, block := historyRootGasNormalizationTrace(t, []map[string]json.RawMessage{fields}, tx, false)
	cost := historyRootGasPoolQuote(t, raw, block, 1)
	if cost.normal[10] >= 4503599627370497 {
		t.Fatal("finite fixture must distinguish actual work from original wide preflight")
	}
	m := providerHistoryContentUnitMeter(cost.normal)
	limits := replayTestLimits()
	limits.MaxTransactions = 1
	b := replayBudget{limits: limits, observer: m}
	profile, got, err := nodeRootGas(context.Background(), raw, block, &b)
	if err == nil || profile != "" || got != nil || !m.failed || m.actual[2] == 0 || m.actual[10] == 0 || m.clocks != 0 {
		t.Fatal("actual upper bypassed the original native room", err, m.actual)
	}
	before := m.actual
	if _, _, err := nodeRootGas(context.Background(), raw, block, &b); err == nil || m.actual != before || !m.failed || m.clocks != 0 {
		t.Fatal("failed native preflight observer was renewed", err, m.actual, before)
	}
	historyEventQueueWithin(t, before, cost.normal)
}
