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

func TestProviderHistoryGasCheckPoolMatchesPortableWholeRows(t *testing.T) {
	w := uint64(strconv.IntSize / 8)
	s := uint64(reflect.TypeFor[types.Transaction]().Size()) + uint64(reflect.TypeFor[types.BlobTx]().Size()) + 2
	j := uint64(reflect.TypeFor[big.Int]().Size())
	for _, row := range []struct{ c, q, v, h, count, decoded, normal, envelope, word, structure uint64 }{
		{4, 1, 1, 1, 1, 121, 4295125134, 4295191184, 7, 1},
		{1024, 2048, 1024, 2048, 2, 6338, 4306774591, 4312083008, 12, 2},
		{8192, 4096, 2048, 67108864, 40, 53032, 7424474921, 4503641411649322, 240, 40},
		{8388608, 8388608, 8388608, 67108864, 16384, 51920896, 599241874919825, 5102875862094226, 98304, 16384},
		{8388608, 67108864, 67108864, 67108864, 16384, 51920896, 9712009513193873, 14215643500368274, 98304, 16384},
	} {
		got, ok := providerHistoryGasCheckPoolUpper(row.c, row.q, row.v, row.h, row.count)
		want := providerHistoryGasCheckCost{
			normal:   candidateWorkVector{2: row.decoded, 10: row.normal + row.word*w + row.structure*(s+j)},
			envelope: candidateWorkVector{2: row.decoded, 10: row.envelope + row.word*w + row.structure*(s+j)},
		}
		if !ok || got != want {
			t.Fatal("independent complete strict-check rows", row, got, want, ok)
		}
	}
}

func TestProviderHistoryGasCheckPoolRejectsInvalidBoundData(t *testing.T) {
	for _, row := range []struct{ c, q, v, h, count uint64 }{
		{0, 1024, 512, 1024, 1}, {3, 1024, 512, 1024, 1}, {(8 << 20) + 1, 1024, 512, 1024, 1},
		{1024, 0, 512, 1024, 1}, {1024, (64 << 20) + 1, 512, 1024, 1},
		{1024, 1024, 0, 1024, 1}, {1024, 1024, 1025, 2048, 1}, {1024, 2048, 1025, 1024, 1},
		{1024, 1024, 512, 0, 1}, {1024, 1024, 512, (64 << 20) + 1, 1},
		{1024, 1024, 512, 1024, 0}, {1024, 1024, 512, 1024, 16385},
		{^uint64(0), 1024, 512, 1024, 1}, {1024, ^uint64(0), 512, 1024, 1},
		{1024, 1024, ^uint64(0), 1024, 1}, {1024, 1024, 512, ^uint64(0), 1}, {1024, 1024, 512, 1024, ^uint64(0)},
	} {
		if got, ok := providerHistoryGasCheckPoolUpper(row.c, row.q, row.v, row.h, row.count); ok || got != (providerHistoryGasCheckCost{}) {
			t.Fatal("invalid data published a partial strict-check pair", row, got, ok)
		}
	}
}

// Finite existing public transaction/quantity DATA only. Header and receipt
// fields are private checker inputs, not a consensus-valid block, authenticated
// RPC response, accepted settlement, deployed Source or dispatch authority.
func historyGasCheckPoolFixture(t *testing.T, profile string, count int) (ReplayBlock, replayDecodedBlock) {
	t.Helper()
	_, tx := historyRootGasNormalizationFields(t, "regularGasUsed")
	header := &types.Header{Number: big.NewInt(2), Difficulty: new(big.Int), Time: sepoliaAmsterdamTime,
		GasLimit: 30000000, GasUsed: uint64(count) * 21000}
	headerRaw, err := rlp.EncodeToBytes(header)
	if err != nil || len(headerRaw) == 0 || len(headerRaw) > 65536 {
		t.Fatal("finite original canonical header precursor", err)
	}
	field := "executionGasUsed"
	if profile == SepoliaAmsterdamNethermindGasProfile || profile == SepoliaAmsterdamRootRegularGasProfile {
		field = "regularGasUsed"
	}
	block := replayDecodedBlock{header: header}
	entries := make([]map[string]any, count)
	for i := range count {
		block.txs = append(block.txs, tx)
		block.receipts = append(block.receipts, &types.Receipt{Status: 1, CumulativeGasUsed: uint64(i+1) * 21000})
		entries[i] = map[string]any{"txHash": tx.Hash().Hex(), "result": map[string]json.RawMessage{
			"gasUsed": historyRootContextString(t, "0x5208"), field: historyRootContextString(t, "0x5208"),
			"stateGasUsed": historyRootContextString(t, "0x0"), "gasRefund": historyRootContextString(t, "0x0"),
		}}
	}
	trace, err := json.Marshal(entries)
	if err != nil {
		t.Fatal("finite complete gas-array precursor", err)
	}
	return ReplayBlock{Gas: &ReplayGasEvidence{profile, header.Hash().Hex(), trace}}, block
}

func historyGasCheckPoolQuote(t *testing.T, raw ReplayBlock, block replayDecodedBlock, maximum int) providerHistoryGasCheckCost {
	t.Helper()
	var pool, largest uint64
	for _, tx := range block.txs {
		encoded, err := tx.MarshalBinary()
		if err != nil || len(encoded) == 0 {
			t.Fatal("complete canonical transaction precursor", err)
		}
		pool += uint64(len(encoded))
		largest = max(largest, uint64(len(encoded)))
	}
	// Keep the real inert64MiB original owner selection. Repeated DATA rows
	// each consume bytes in Q; empty rows keep their original positive cap.
	cost, ok := providerHistoryGasCheckPoolUpper(max(4, uint64(len(raw.Gas.Trace))), max(1, pool), max(1, largest), 64<<20, uint64(maximum))
	if !ok {
		t.Fatal("finite complete strict-check pair")
	}
	return cost
}

func TestProviderHistoryGasCheckPoolPreservesAllSelectedProfiles(t *testing.T) {
	for _, profile := range []string{SepoliaAmsterdamGasProfile, SepoliaAmsterdamNethermindGasProfile, SepoliaAmsterdamRootRegularGasProfile, SepoliaAmsterdamRootExecutionGasProfile} {
		for _, count := range []int{1, 2} {
			raw, block := historyGasCheckPoolFixture(t, profile, count)
			original := bytes.Clone(raw.Gas.Trace)
			cost := historyGasCheckPoolQuote(t, raw, block, count)
			m := providerHistoryContentUnitMeter(cost.envelope)
			limits := replayTestLimits()
			limits.MaxTransactions = count
			b := replayBudget{limits: limits, observer: m}
			legacy := replayBudget{limits: limits}
			if err := replayCheckGas(context.Background(), raw, block, &legacy); err != nil {
				t.Fatal("ordinary legacy profile precursor", profile, count, err)
			}
			if err := replayCheckGas(context.Background(), raw, block, &b); err != nil || m.failed || m.clocks != 0 || b.bytes != legacy.bytes || !bytes.Equal(original, raw.Gas.Trace) {
				t.Fatal("original strict gas profile changed", profile, count, err, m.actual)
			}
			historyEventQueueWithin(t, m.actual, cost.normal)
		}
	}
}

func TestProviderHistoryGasCheckPoolRetainsEmptyOriginalRawSlots(t *testing.T) {
	for _, trace := range []string{"[]", " [ ] "} {
		raw, block := historyGasCheckPoolFixture(t, SepoliaAmsterdamRootRegularGasProfile, 0)
		raw.Gas.Trace = []byte(trace)
		cost := historyGasCheckPoolQuote(t, raw, block, 1)
		m := providerHistoryContentUnitMeter(cost.envelope)
		limits := replayTestLimits()
		limits.MaxTransactions = 1
		b := replayBudget{limits: limits, observer: m}
		if err := replayCheckGas(context.Background(), raw, block, &b); err != nil || m.failed || m.actual[2] != 2*uint64(len(trace)) || m.clocks != 0 {
			t.Fatal("empty strict check lost original raw-array accounting", trace, err, m.actual)
		}
		historyEventQueueWithin(t, m.actual, cost.normal)
	}
}

func TestProviderHistoryGasCheckPoolKeepsWorkBeforeLaterOrdinaryRefusal(t *testing.T) {
	var previous candidateWorkVector
	var originalLength int
	for _, badIndex := range []int{0, 1} {
		raw, block := historyGasCheckPoolFixture(t, SepoliaAmsterdamRootExecutionGasProfile, 2)
		var rows []struct {
			TxHash string                     `json:"txHash"`
			Result map[string]json.RawMessage `json:"result"`
		}
		if json.Unmarshal(raw.Gas.Trace, &rows) != nil {
			t.Fatal("complete private rows precursor")
		}
		rows[badIndex].Result["gasRefund"] = historyRootContextString(t, "0x"+strings.Repeat("f", 64))
		trace, err := json.Marshal(rows)
		if err != nil {
			t.Fatal(err)
		}
		raw.Gas.Trace = trace
		cost := historyGasCheckPoolQuote(t, raw, block, 2)
		m := providerHistoryContentUnitMeter(cost.envelope)
		limits := replayTestLimits()
		limits.MaxTransactions = 2
		b := replayBudget{limits: limits, observer: m}
		if err := replayCheckGas(context.Background(), raw, block, &b); err == nil || m.failed || m.clocks != 0 {
			t.Fatal("ordinary later quantity refusal changed original observer", badIndex, err, m.actual)
		}
		if badIndex == 0 {
			previous, originalLength = m.actual, len(trace)
		} else if len(trace) != originalLength || m.actual[2] <= previous[2] || m.actual[10] <= previous[10] {
			t.Fatal("later refusal erased complete earlier gas work", m.actual, previous)
		}
		historyEventQueueWithin(t, m.actual, cost.normal)
	}
}

func TestProviderHistoryGasCheckPoolBothPassesKeepActualsAndOneIncomingRoom(t *testing.T) {
	raw, block := historyGasCheckPoolFixture(t, SepoliaAmsterdamGasProfile, 1)
	cost := historyGasCheckPoolQuote(t, raw, block, 1)
	upper := cost.normal
	for i := range upper {
		upper[i] *= 2
	}
	upper[10] += cost.envelope[10] - cost.normal[10]
	m := providerHistoryContentUnitMeter(upper)
	limits := replayTestLimits()
	limits.MaxTransactions = 1
	b := replayBudget{limits: limits, observer: m}
	var first candidateWorkVector
	for pass := uint64(1); pass <= 2; pass++ {
		if err := replayCheckGas(context.Background(), raw, block, &b); err != nil || m.failed || m.clocks != 0 {
			t.Fatal("repeated original strict check lost its incoming room", pass, err, m.actual)
		}
		if pass == 1 {
			first = m.actual
		}
		for coordinate, value := range first {
			if m.actual[coordinate] != pass*value {
				t.Fatal("second strict pass renewed actuals or charged pending room", pass, coordinate, m.actual, first)
			}
		}
	}
}

func TestProviderHistoryGasCheckPoolCannotReplaceOrRenewNativePreflight(t *testing.T) {
	raw, block := historyGasCheckPoolFixture(t, SepoliaAmsterdamRootRegularGasProfile, 1)
	cost := historyGasCheckPoolQuote(t, raw, block, 1)
	if cost.normal[10] >= 4503599627370497 {
		t.Fatal("finite fixture must separate actual upper from original wide room")
	}
	m := providerHistoryContentUnitMeter(cost.normal)
	limits := replayTestLimits()
	limits.MaxTransactions = 1
	b := replayBudget{limits: limits, observer: m}
	if err := replayCheckGas(context.Background(), raw, block, &b); err == nil || !m.failed || m.actual[2] == 0 || m.actual[10] == 0 || m.clocks != 0 {
		t.Fatal("normal actual upper bypassed original native preflight", err, m.actual)
	}
	before := m.actual
	if err := replayCheckGas(context.Background(), raw, block, &b); err == nil || !m.failed || m.actual != before || m.clocks != 0 {
		t.Fatal("failed original strict observer was renewed", err, m.actual, before)
	}
	historyEventQueueWithin(t, before, cost.normal)
}

func TestProviderHistoryGasCheckPoolPreservesOriginalEarlyRefusals(t *testing.T) {
	for _, mode := range []string{"pre-fork", "unsupported-profile", "header-limit"} {
		raw, block := historyGasCheckPoolFixture(t, SepoliaAmsterdamGasProfile, 1)
		cost := historyGasCheckPoolQuote(t, raw, block, 1)
		switch mode {
		case "pre-fork":
			block.header.Time = sepoliaAmsterdamTime - 1
		case "unsupported-profile":
			raw.Gas.Profile = "unselected-profile"
		case "header-limit":
			block.header.GasLimit = block.header.GasUsed - 1
		}
		m := providerHistoryContentUnitMeter(cost.envelope)
		b := replayBudget{limits: replayTestLimits(), observer: m}
		if err := replayCheckGas(context.Background(), raw, block, &b); err == nil || m.failed || m.actual != (candidateWorkVector{}) || m.clocks != 0 || b.bytes != 0 {
			t.Fatal("early original guard entered a gas profile or observer work", mode, err, m.actual)
		}
	}
}
