package registry011

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
)

func TestProviderHistoryRootGasNormalizationEnvelopeMatchesPortableRows(t *testing.T) {
	w := uint64(strconv.IntSize / 8)
	s := uint64(reflect.TypeFor[types.Transaction]().Size()) + uint64(reflect.TypeFor[types.BlobTx]().Size()) + 2
	for _, row := range []struct{ body, transaction, count, decoded, constant, word, sidecar uint64 }{
		{4, 1, 1, 193, 104008, 12, 1},
		{4, 1, 16384, 193, 104008, 12, 1},
		{1024, 2048, 2, 8514, 28555899, 24, 2},
		{8192, 9216, 40, 71976, 7592599869, 480, 40},
		{8388608, 8389632, 16384, 69746688, 2307109966404567045, 196608, 16384},
	} {
		got, ok := providerHistoryRootGasNormalizationRefusalUpper(row.body, row.transaction, row.count)
		want := candidateWorkVector{2: row.decoded, 10: row.constant + row.word*w + row.sidecar*s}
		if !ok || got != want {
			t.Fatal("portable complete normalization row", row, got, want, ok)
		}
	}
}

func TestProviderHistoryRootGasNormalizationEnvelopeRejectsInvalidOrOverflowingData(t *testing.T) {
	for _, row := range []struct{ body, transaction, count uint64 }{
		{0, 1024, 1}, {3, 1024, 1}, {(8 << 20) + 1, 1024, 1},
		{1024, 0, 1}, {1024, (64 << 20) + 1, 1}, {1024, 1024, 0}, {1024, 1024, 16385},
		{^uint64(0), 1024, 1}, {1024, ^uint64(0), 1}, {1024, 1024, ^uint64(0)},
		{8 << 20, 64 << 20, 16384},
	} {
		if got, ok := providerHistoryRootGasNormalizationRefusalUpper(row.body, row.transaction, row.count); ok || got != (candidateWorkVector{}) {
			t.Fatal("domain or late overflow published partial work", row, got, ok)
		}
	}
}

func historyRootGasNormalizationFields(t *testing.T, dimension string) (map[string]json.RawMessage, *types.Transaction) {
	t.Helper()
	fields, tx := historyRootContextFixture(t, false)
	fields["gasUsed"] = historyRootContextString(t, "0x5208")
	fields[dimension] = historyRootContextString(t, "0x5208")
	fields["stateGasUsed"] = historyRootContextString(t, "0x0")
	fields["gasRefund"] = historyRootContextString(t, "0x0")
	return fields, tx
}

// Finite private normalizer DATA only. Reusing the public signature-decoder
// fixture in two rows does not claim a canonical block, valid settlement,
// provider response, deployment, Source or dispatch permission.
func historyRootGasNormalizationTrace(t *testing.T, rows []map[string]json.RawMessage, tx *types.Transaction, escaped bool) ([]byte, replayDecodedBlock) {
	t.Helper()
	entries := make([]map[string]any, len(rows))
	var block replayDecodedBlock
	for i, fields := range rows {
		hash := historyRootContextString(t, tx.Hash().Hex())
		if escaped {
			hash = json.RawMessage(`"\u0030x` + tx.Hash().Hex()[2:] + `"`)
		}
		entries[i] = map[string]any{"txHash": hash, "result": fields}
		block.txs = append(block.txs, tx)
		block.receipts = append(block.receipts, &types.Receipt{Status: 1})
	}
	raw, err := json.Marshal(entries)
	if err != nil {
		t.Fatal(err)
	}
	return raw, block
}

func historyRootGasNormalizationQuote(t *testing.T, raw []byte, maximum int) candidateWorkVector {
	t.Helper()
	// Preserve the real wide inert selector. No collection owner is invented
	// to make its original transaction/signing preflights smaller.
	upper, ok := providerHistoryRootGasNormalizationRefusalUpper(max(4, uint64(len(raw))), 64<<20, uint64(maximum))
	if !ok {
		t.Fatal("finite private normalization envelope")
	}
	return upper
}

func TestProviderHistoryRootGasNormalizationEnvelopeRetainsEmptyArrayAllocations(t *testing.T) {
	w := uint64(strconv.IntSize / 8)
	for _, tc := range []struct {
		raw              string
		decoded, logical uint64
	}{
		{"[]", 4, 17},
		{" [ ] ", 10, 59 + 3*w},
	} {
		raw := []byte(tc.raw)
		upper := historyRootGasNormalizationQuote(t, raw, 1)
		m := providerHistoryContentUnitMeter(upper)
		limits := replayTestLimits()
		limits.MaxTransactions = 1
		b := replayBudget{limits: limits, observer: m}
		profile, got, err := nodeRootGas(context.Background(), raw, replayDecodedBlock{}, &b)
		if err != nil || profile != SepoliaAmsterdamRootRegularGasProfile || !bytes.Equal(got, []byte("[]")) || m.failed ||
			m.actual != (candidateWorkVector{2: tc.decoded, 10: tc.logical}) || m.clocks != 0 || b.bytes != len(raw) {
			t.Fatal("empty normalization lost raw slot allocation or final encode", tc.raw, err, m.actual)
		}
		historyEventQueueWithin(t, m.actual, upper)
	}
}

func TestProviderHistoryRootGasNormalizationEnvelopePreservesSelectedDimensions(t *testing.T) {
	for _, mode := range []string{"regular", "execution", "escaped-hash"} {
		dimension, wantProfile := "regularGasUsed", SepoliaAmsterdamRootRegularGasProfile
		if mode == "execution" {
			dimension, wantProfile = "executionGasUsed", SepoliaAmsterdamRootExecutionGasProfile
		}
		fields, tx := historyRootGasNormalizationFields(t, dimension)
		raw, block := historyRootGasNormalizationTrace(t, []map[string]json.RawMessage{fields}, tx, mode == "escaped-hash")
		original := bytes.Clone(raw)
		limits := replayTestLimits()
		limits.MaxTransactions = 1
		legacy := replayBudget{limits: limits}
		want, wantRaw, oldErr := nodeRootGas(context.Background(), raw, block, &legacy)
		upper := historyRootGasNormalizationQuote(t, raw, 1)
		m := providerHistoryContentUnitMeter(upper)
		b := replayBudget{limits: limits, observer: m}
		profile, got, err := nodeRootGas(context.Background(), raw, block, &b)
		if err != nil || oldErr != nil || profile != wantProfile || want != profile || !bytes.Equal(got, wantRaw) ||
			len(got) > len(raw) || !bytes.Equal(raw, original) || b.bytes != legacy.bytes || m.failed || m.actual[2] < 97 || m.clocks != 0 {
			t.Fatal("original profile or validated subset changed", mode, err, oldErr, m.actual)
		}
		var decoded []struct {
			TxHash string                     `json:"txHash"`
			Result map[string]json.RawMessage `json:"result"`
		}
		if json.Unmarshal(got, &decoded) != nil || len(decoded) != 1 || decoded[0].TxHash != tx.Hash().Hex() || len(decoded[0].Result) != 4 {
			t.Fatal("result is not the original four-dimension subset", mode)
		}
		for _, name := range []string{"gasUsed", dimension, "stateGasUsed", "gasRefund"} {
			if !bytes.Equal(decoded[0].Result[name], fields[name]) {
				t.Fatal("dimension bytes were replaced", mode, name)
			}
		}
		historyEventQueueWithin(t, m.actual, upper)
	}
}

func TestProviderHistoryRootGasNormalizationEnvelopeFundsOriginalRefusingPrefixes(t *testing.T) {
	for _, mode := range []string{"witness-empty", "witness-unknown", "hash-shape", "root-empty", "root-spaced-empty", "root-unknown", "destination", "both-dimensions", "missing-dimension", "quantity", "dimension-shape", "extra-row", "trailing", "incomplete"} {
		fields, tx := historyRootGasNormalizationFields(t, "regularGasUsed")
		switch mode {
		case "root-unknown":
			fields[strings.Repeat("z", 4096)] = json.RawMessage("true")
		case "destination":
			fields["to"] = json.RawMessage("true")
		case "both-dimensions":
			fields["executionGasUsed"] = historyRootContextString(t, "0x5208")
		case "missing-dimension":
			delete(fields, "regularGasUsed")
		case "quantity":
			fields["gasRefund"] = historyRootContextString(t, "0x"+strings.Repeat("f", 64))
		case "dimension-shape":
			fields["gasRefund"] = json.RawMessage("true")
		}
		raw, block := historyRootGasNormalizationTrace(t, []map[string]json.RawMessage{fields}, tx, false)
		switch mode {
		case "witness-empty":
			raw = []byte("[{}]")
		case "trailing":
			raw = []byte("[] true")
		case "incomplete":
			raw = []byte("[")
		case "extra-row":
			raw, _ = historyRootGasNormalizationTrace(t, []map[string]json.RawMessage{fields, fields}, tx, false)
		case "witness-unknown", "hash-shape", "root-empty", "root-spaced-empty":
			var rows []map[string]json.RawMessage
			if json.Unmarshal(raw, &rows) != nil {
				t.Fatal("normal outer row precursor")
			}
			switch mode {
			case "witness-unknown":
				rows[0][strings.Repeat("z", 4096)] = json.RawMessage("true")
			case "hash-shape":
				rows[0]["txHash"] = json.RawMessage("true")
			case "root-empty", "root-spaced-empty":
				rows[0]["result"] = json.RawMessage("{}")
			}
			var err error
			raw, err = json.Marshal(rows)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "root-spaced-empty" {
				// Preserve one interior space in the original result span.
				// json.Marshal's RawMessage compaction otherwise erases it.
				raw = bytes.Replace(raw, []byte(`"result":{}`), []byte(`"result":{ }`), 1)
			}
		}
		original := bytes.Clone(raw)
		upper := historyRootGasNormalizationQuote(t, raw, 1)
		m := providerHistoryContentUnitMeter(upper)
		limits := replayTestLimits()
		limits.MaxTransactions = 1
		b := replayBudget{limits: limits, observer: m}
		profile, got, err := nodeRootGas(context.Background(), raw, block, &b)
		if err == nil || profile != "" || got != nil || m.failed || m.actual[2] == 0 || m.actual[10] == 0 ||
			m.clocks != 0 || !bytes.Equal(raw, original) || b.bytes != len(raw) {
			t.Fatal("ordinary refusing prefix exceeded its complete envelope", mode, err, m.actual)
		}
		historyEventQueueWithin(t, m.actual, upper)
	}
}

func TestProviderHistoryRootGasNormalizationEnvelopeKeepsWorkBeforeLaterRowRefusal(t *testing.T) {
	var previous candidateWorkVector
	var originalLength int
	for _, badIndex := range []int{0, 1} {
		first, tx := historyRootGasNormalizationFields(t, "regularGasUsed")
		second, _ := historyRootGasNormalizationFields(t, "regularGasUsed")
		rows := []map[string]json.RawMessage{first, second}
		rows[badIndex]["gasRefund"] = historyRootContextString(t, "0x"+strings.Repeat("f", 64))
		raw, block := historyRootGasNormalizationTrace(t, rows, tx, false)
		upper := historyRootGasNormalizationQuote(t, raw, 2)
		m := providerHistoryContentUnitMeter(upper)
		limits := replayTestLimits()
		limits.MaxTransactions = 2
		b := replayBudget{limits: limits, observer: m}
		profile, got, err := nodeRootGas(context.Background(), raw, block, &b)
		if err == nil || profile != "" || got != nil || m.failed || m.clocks != 0 {
			t.Fatal("later refusal released a partial witness", badIndex, err, m.actual)
		}
		if badIndex == 0 {
			previous, originalLength = m.actual, len(raw)
		} else if len(raw) != originalLength || m.actual[2] <= previous[2] || m.actual[10] <= previous[10] {
			t.Fatal("equal-sized later refusal erased the completed row", m.actual, previous)
		}
		historyEventQueueWithin(t, m.actual, upper)
	}
}

func TestProviderHistoryRootGasNormalizationEnvelopeCannotRenewFailedObserver(t *testing.T) {
	fields, tx := historyRootGasNormalizationFields(t, "regularGasUsed")
	raw, block := historyRootGasNormalizationTrace(t, []map[string]json.RawMessage{fields}, tx, false)
	upper := historyRootGasNormalizationQuote(t, raw, 1)
	// Fund one complete raw codec before constructing the inert meter. The
	// second raw-array codec must fail without erasing the first observation.
	upper[2] = uint64(len(raw))
	m := providerHistoryContentUnitMeter(upper)
	limits := replayTestLimits()
	limits.MaxTransactions = 1
	b := replayBudget{limits: limits, observer: m}
	profile, got, err := nodeRootGas(context.Background(), raw, block, &b)
	n := uint64(len(raw))
	want := candidateWorkVector{2: n, 10: n*n + 1}
	if err == nil || profile != "" || got != nil || !m.failed || m.actual != want {
		t.Fatal("raw-array shortage lost its original codec prefix", err, m.actual, want)
	}
	before := m.actual
	if _, _, err := nodeRootGas(context.Background(), raw, block, &b); err == nil || m.actual != before || !m.failed || m.clocks != 0 {
		t.Fatal("failed normalization observer was renewed", err, m.actual, before)
	}
}
