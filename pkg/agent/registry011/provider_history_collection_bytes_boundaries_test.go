package registry011

import (
	"bytes"
	"context"
	"reflect"
	"testing"
)

// These tiny byte containers exercise copies only. Their deliberately opaque
// bytes are not decoded consensus evidence, a native gas witness or authority.
func TestProviderHistoryCollectionRetainedBytesRefusePartialCopies(t *testing.T) {
	header := FinalizedHeader{Hash: admissionTestDigest("a"), ParentHash: admissionTestDigest("b"), Number: 1, Timestamp: 2}
	raw := ReplayBlock{HeaderRLP: []byte("header"), Transactions: [][]byte{[]byte("transaction")}, Receipts: [][]byte{[]byte("receipt")},
		Gas: &ReplayGasEvidence{Profile: SepoliaAmsterdamGasProfile, BlockHash: header.Hash, Trace: []byte("[]")}}
	before := replayTestClone(t, raw)
	want := providerHistoryCollectionBlock{header: header, transactionCount: 1, headerRLP: "header", transactions: []string{"transaction"}, receipts: []string{"receipt"},
		gasProfile: SepoliaAmsterdamGasProfile, gasTrace: "[]"}
	historyUnitConversionRefusalPrefixes(t, func(m *providerWorkMeter) (bool, bool) {
		out, ok := providerHistoryCollectionRetainBlockObserved(raw, header, m)
		if !reflect.DeepEqual(raw, before) || ok && !reflect.DeepEqual(out, want) {
			t.Fatal("retention changed its input or normal bytes")
		}
		return ok, ok || reflect.DeepEqual(out, providerHistoryCollectionBlock{})
	})
	historyUnitConversionRefusalPrefixes(t, func(m *providerWorkMeter) (bool, bool) {
		out, ok := want.replayObserved(m)
		if ok && !reflect.DeepEqual(out, raw) {
			t.Fatal("reconstruction changed retained bytes")
		}
		return ok, ok || reflect.DeepEqual(out, ReplayBlock{})
	})
	out, ok := want.replayObserved(providerReplayUnitMeter())
	if !ok {
		t.Fatal("normal byte-copy precursor")
	}
	out.HeaderRLP[0], out.Transactions[0][0], out.Receipts[0][0], out.Gas.Trace[0] = 'X', 'X', 'X', 'X'
	if want.headerRLP != "header" || want.transactions[0] != "transaction" || want.receipts[0] != "receipt" || want.gasTrace != "[]" || !reflect.DeepEqual(raw, before) {
		t.Fatal("returned mutable byte storage aliases retained or original bytes")
	}
	retained, ok := providerHistoryCollectionRetainBlockObserved(raw, header, providerReplayUnitMeter())
	if !ok {
		t.Fatal("normal retention precursor")
	}
	raw.HeaderRLP[0], raw.Transactions[0][0], raw.Receipts[0][0], raw.Gas.Trace[0] = 'Y', 'Y', 'Y', 'Y'
	if !reflect.DeepEqual(retained, want) {
		t.Fatal("retained strings borrow mutable input byte storage")
	}
}

func TestProviderHistoryCollectionDecoderRefusalsKeepOriginalInput(t *testing.T) {
	c, input, methods := historyCollectionContentFixture(t, 4)
	c = historyCollectionContentStep(t, c, input, methods)
	table := historyCollectionBoundaryPlan(t, c, methods)
	results := historyCollectionContentResults(t, c, input, table)
	block, receipts := results[0].result, results[1].result
	header := journalTestHeader(t, input.Blocks[0])
	before := replayTestClone(t, input)
	out, decoded, err := providerHistoryCollectionDecodeBlockObserved(context.Background(), block, receipts, header, c.selection.input, providerHistorySemanticUnitMeter())
	if err != nil || !bytes.Equal(out.HeaderRLP, input.Blocks[0].HeaderRLP) || decoded.header == nil {
		t.Fatal("ordinary canonical block decoding precursor", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	for _, tc := range []struct {
		name, block, receipts, encoding string
		ctx                             context.Context
	}{
		{"nil-context", block, receipts, c.selection.input.encoding, nil},
		{"cancelled-context", block, receipts, c.selection.input.encoding, cancelled},
		{"empty-block", "", receipts, c.selection.input.encoding, context.Background()},
		{"empty-receipts", block, "", c.selection.input.encoding, context.Background()},
		{"unknown-encoding", block, receipts, "unknown", context.Background()},
		{"receipt-string", block, `"ordinary DATA"`, c.selection.input.encoding, context.Background()},
		{"missing-receipt", block, `[]`, c.selection.input.encoding, context.Background()},
		{"receipt-hex", block, `["0xnot-hex"]`, c.selection.input.encoding, context.Background()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			selected := c.selection.input
			selected.encoding = tc.encoding // Decoder DATA, never a live selection.
			out, decoded, err := providerHistoryCollectionDecodeBlockObserved(tc.ctx, tc.block, tc.receipts, header, selected, providerHistorySemanticUnitMeter())
			if err == nil || !reflect.DeepEqual(out, ReplayBlock{}) || !reflect.DeepEqual(decoded, replayDecodedBlock{}) || !reflect.DeepEqual(input, before) {
				t.Fatal("decoder refusal changed its original input or exposed a partial value", err)
			}
		})
	}
	for _, fuel := range []uint64{0, uint64(len(block) + 1)} {
		m := providerHistorySemanticUnitMeter()
		m.upper[10] = fuel
		out, decoded, err := providerHistoryCollectionDecodeBlockObserved(context.Background(), block, receipts, header, c.selection.input, m)
		if err == nil || !m.failed || !reflect.DeepEqual(out, ReplayBlock{}) || !reflect.DeepEqual(decoded, replayDecodedBlock{}) || m.actual[10] > fuel {
			t.Fatal("unfunded original input copy reached decoding", err, m.actual)
		}
		prefix := m.actual
		if _, _, err := providerHistoryCollectionDecodeBlockObserved(context.Background(), block, receipts, header, c.selection.input, m); err == nil || m.actual != prefix {
			t.Fatal("refused decoder renewed its observer", err)
		}
	}
}

func TestProviderHistoryCollectionByteShapesRefuseWithoutPartialValues(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  ReplayBlock
	}{
		{"missing-header", ReplayBlock{}},
		{"oversized-header", ReplayBlock{HeaderRLP: bytes.Repeat([]byte{1}, 65537)}},
		{"missing-receipt", ReplayBlock{HeaderRLP: []byte{1}, Transactions: [][]byte{{2}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, ok := providerHistoryCollectionRetainBlockObserved(tc.raw, FinalizedHeader{}, providerReplayUnitMeter())
			if ok || !reflect.DeepEqual(out, providerHistoryCollectionBlock{}) {
				t.Fatal("invalid byte shape exposed a partial retained value")
			}
		})
	}
	for _, tc := range []struct {
		name  string
		block providerHistoryCollectionBlock
	}{
		{"missing-header", providerHistoryCollectionBlock{}},
		{"missing-receipt", providerHistoryCollectionBlock{headerRLP: "header", transactionCount: 1, transactions: []string{"tx"}}},
		{"wrong-count", providerHistoryCollectionBlock{headerRLP: "header", transactionCount: 1}},
		{"trace-without-profile", providerHistoryCollectionBlock{headerRLP: "header", gasTrace: "[]"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, ok := tc.block.replayObserved(providerReplayUnitMeter())
			if ok || !reflect.DeepEqual(out, ReplayBlock{}) {
				t.Fatal("invalid retained shape exposed a partial replay value")
			}
		})
	}
}

func TestProviderHistoryCollectionPendingByteDigestsChargeFreshCopies(t *testing.T) {
	want := providerHistoryCollectionBytesRootObserved("abc", 3, providerReplayUnitMeter())
	if !lowerHash(want) {
		t.Fatal("normal local DATA digest precursor")
	}
	historyUnitConversionRefusalPrefixes(t, func(m *providerWorkMeter) (bool, bool) {
		out := providerHistoryCollectionBytesRootObserved("abc", 3, m)
		if out != "" && out != want {
			t.Fatal("fresh digest changed the normal DATA value")
		}
		return out != "", out == "" || out == want
	})
	// The list's bounded JCS preflight can exceed its eventual encoding work.
	// Do not claim the measured work is a sufficient incoming allowance.
	for _, values := range [][]string{{"a", "bc"}, {"a", ""}} {
		m := providerReplayUnitMeter()
		m.upper[10] = 1
		if out := providerHistoryCollectionListRootObserved(values, 2, m); out != "" || !m.failed {
			t.Fatal("unfunded list digest exposed a root")
		}
		prefix := m.actual
		if out := providerHistoryCollectionListRootObserved(values, 2, m); out != "" || m.actual != prefix {
			t.Fatal("failed list digest renewed its original observer")
		}
	}
	if providerHistoryCollectionListRootObserved([]string{"a", ""}, 2, providerReplayUnitMeter()) != "" {
		t.Fatal("missing retained bytes produced a complete list digest")
	}
}
