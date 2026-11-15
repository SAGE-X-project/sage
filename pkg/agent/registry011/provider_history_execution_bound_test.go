package registry011

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestProviderHistoryExecutionReplayMatchesPortableCompleteRows(t *testing.T) {
	w := uint64(strconv.IntSize / 8)
	s := uint64(reflect.TypeFor[types.Transaction]().Size()) + uint64(reflect.TypeFor[types.BlobTx]().Size()) + 2
	j, row, log := uint64(reflect.TypeFor[big.Int]().Size()), uint64(reflect.TypeFor[replayLog]().Size()), uint64(reflect.TypeFor[types.Log]().Size())
	for _, tc := range []struct {
		limits                                     providerHistoryExecutionReplayLimits
		decoded, normal, envelope, word, row, logs uint64
	}{
		{providerHistoryExecutionReplayLimits{4, 1, 1, 1, 1, 1}, 265, 4295141058, 4295207108, 2, 23, 1},
		{providerHistoryExecutionReplayLimits{1024, 1024, 2048, 2, 2, 2048}, 369, 4297782027, 4303090444, 0, 30, 2},
		{providerHistoryExecutionReplayLimits{8192, 2048, 67108864, 40, 30, 65536}, 3921, 4439133765, 4503638426308166, 0, 226, 30},
		{providerHistoryExecutionReplayLimits{8388608, 8388608, 67108864, 65536, 65536, 67108864}, 6815905, 140746780740179, 4644380767914580, 0, 458768, 65536},
		{providerHistoryExecutionReplayLimits{8388608, 67108864, 67108864, 65536, 65536, 67108864}, 6815905, 9007238846777939, 13510872833952340, 0, 458768, 65536},
	} {
		fixed := s + 2*j + tc.word*w + tc.row*row + tc.logs*log
		want := providerHistoryExecutionReplayCost{
			normal:   candidateWorkVector{2: tc.decoded, 5: tc.limits.Frames, 6: tc.limits.Logs, 10: tc.normal + fixed},
			envelope: candidateWorkVector{2: tc.decoded, 5: tc.limits.Frames, 6: tc.limits.Logs, 10: tc.envelope + fixed},
		}
		if got, ok := providerHistoryExecutionReplayUpper(tc.limits); !ok || got != want {
			t.Fatal("independent complete execution row", tc, got, want, ok)
		}
	}
}

func TestProviderHistoryExecutionReplayRejectsInvalidBoundData(t *testing.T) {
	for _, p := range []providerHistoryExecutionReplayLimits{
		{0, 512, 1024, 2, 2, 1024}, {3, 512, 1024, 2, 2, 1024}, {(8 << 20) + 1, 512, 1024, 2, 2, 1024},
		{1024, 0, 1024, 2, 2, 1024}, {1024, 1025, 1024, 2, 2, 1024},
		{1024, 512, 0, 2, 2, 1024}, {1024, 512, (64 << 20) + 1, 2, 2, 1024},
		{1024, 512, 1024, 0, 2, 1024}, {1024, 512, 1024, 65537, 2, 1024},
		{1024, 512, 1024, 2, 0, 1024}, {1024, 512, 1024, 2, 65537, 1024},
		{1024, 512, 1024, 2, 2, 0}, {1024, 512, 1024, 2, 2, (64 << 20) + 1},
		{^uint64(0), 512, 1024, 2, 2, 1024}, {1024, ^uint64(0), 1024, 2, 2, 1024},
		{1024, 512, ^uint64(0), 2, 2, 1024}, {1024, 512, 1024, ^uint64(0), 2, 1024},
		{1024, 512, 1024, 2, ^uint64(0), 1024}, {1024, 512, 1024, 2, 2, ^uint64(0)},
	} {
		if got, ok := providerHistoryExecutionReplayUpper(p); ok || got != (providerHistoryExecutionReplayCost{}) {
			t.Fatal("invalid input published a partial execution pair", p, got, ok)
		}
	}
}

func historyExecutionBoundJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal("finite private JSON precursor", err)
	}
	return raw
}

func historyExecutionDifferentAddress(t *testing.T, raw json.RawMessage) json.RawMessage {
	t.Helper()
	var address string
	if json.Unmarshal(raw, &address) != nil || len(address) != 42 {
		t.Fatal("canonical private address precursor")
	}
	last := "0"
	if address[41] == '0' {
		last = "1"
	}
	return historyRootContextString(t, address[:41]+last)
}

func historyExecutionLogRows(t *testing.T, address json.RawMessage, count, topicCount int) []map[string]any {
	t.Helper()
	var text string
	if json.Unmarshal(address, &text) != nil {
		t.Fatal("private log address precursor")
	}
	if topicCount < 0 || topicCount > 4 {
		t.Fatal("private log topic count")
	}
	topics := make([]string, topicCount)
	for i := range topics {
		topics[i] = "0x" + strings.Repeat(strconv.Itoa(i+1), 64)
	}
	rows := make([]map[string]any, count)
	for i := range rows {
		rows[i] = map[string]any{"address": text, "topics": topics,
			"data": "0x" + hex.EncodeToString([]byte{byte(i), byte(i + 1)}), "position": 0, "index": i}
	}
	return rows
}

// Existing public signature-decoder DATA and finite callTracer JSON only.
// Header/receipts/tree are private checker inputs, not a consensus-valid block,
// valid gas settlement, authenticated RPC response, deployed Source or grant.
func historyExecutionBoundFixture(t *testing.T, mode string, logCount int) (ReplayExecution, replayDecodedBlock, []byte) {
	t.Helper()
	return historyExecutionBoundFixtureTopics(t, mode, logCount, 1)
}

func historyExecutionBoundFixtureTopics(t *testing.T, mode string, logCount, topicCount int) (ReplayExecution, replayDecodedBlock, []byte) {
	t.Helper()
	fields, tx := historyRootContextFixture(t, mode == "create")
	delete(fields, "stateGasReservoir")
	fields["gasUsed"] = historyRootContextString(t, "0x5208")
	fields["logs"] = historyExecutionBoundJSON(t, historyExecutionLogRows(t, fields["to"], logCount, topicCount))
	switch mode {
	case "wrong-from":
		fields["from"] = historyExecutionDifferentAddress(t, fields["from"])
	case "wrong-value":
		fields["value"] = historyRootContextString(t, "0x1")
	case "wrong-input":
		fields["input"] = historyRootContextString(t, "0x0103")
	case "wrong-recipient":
		fields["to"] = historyExecutionDifferentAddress(t, fields["to"])
	case "wrong-kind":
		fields["type"] = historyRootContextString(t, "STATICCALL")
	case "root-error":
		fields["error"] = historyRootContextString(t, "example failure")
	}
	child := mode == "committed-child" || mode == "uncommitted-child" || mode == "static-child" || mode == "wrong-child-from"
	if child {
		callee := make(map[string]json.RawMessage, len(fields))
		for key, value := range fields {
			callee[key] = value
		}
		callee["from"] = fields["to"]
		childLogs := historyExecutionLogRows(t, callee["to"], 1, topicCount)
		childLogs[0]["data"] = "0x7f00"
		callee["logs"] = historyExecutionBoundJSON(t, childLogs)
		switch mode {
		case "uncommitted-child":
			callee["error"] = historyRootContextString(t, "example child failure")
		case "static-child":
			callee["type"] = historyRootContextString(t, "STATICCALL")
			callee["logs"] = json.RawMessage("[]")
		case "wrong-child-from":
			callee["from"] = historyExecutionDifferentAddress(t, callee["from"])
		}
		fields["calls"] = historyExecutionBoundJSON(t, []map[string]json.RawMessage{callee})
	}
	raw := historyExecutionBoundJSON(t, fields)
	frame, err := DecodeCallTracer(context.Background(), raw, replayTestLimits())
	if err != nil {
		t.Fatal("complete private normalized-tree precursor", mode, err)
	}
	header := &types.Header{Number: big.NewInt(2), Difficulty: new(big.Int), Time: sepoliaAmsterdamTime,
		GasLimit: 30000000, GasUsed: 21000}
	headerRaw, err := rlp.EncodeToBytes(header)
	if err != nil || len(headerRaw) == 0 || len(headerRaw) > 65536 {
		t.Fatal("finite canonical header precursor", err)
	}
	// All own logs have position0 and the only child follows them. Construct
	// that known reference order directly, without calling the observed walk.
	ordered := append([]ReplayFrameLog(nil), frame.Logs...)
	if mode == "committed-child" || mode == "wrong-child-from" {
		ordered = append(ordered, frame.Calls[0].Logs...)
	}
	receipt := &types.Receipt{Type: tx.Type(), Status: 1, CumulativeGasUsed: 21000}
	for _, log := range ordered {
		receipt.Logs = append(receipt.Logs, &types.Log{Address: common.HexToAddress(log.Address),
			Topics: append([]common.Hash(nil), log.Topics...), Data: bytes.Clone(log.Data)})
	}
	if mode == "receipt-status" {
		receipt.Status = 0
	}
	e := ReplayExecution{header.Hash().Hex(), tx.Hash().Hex(), 0, frame}
	if mode == "wrong-index" {
		e.TransactionIndex = 1
	}
	return e, replayDecodedBlock{header: header, txs: types.Transactions{tx}, receipts: types.Receipts{receipt}}, raw
}

func historyExecutionBoundQuote(t *testing.T, block replayDecodedBlock, raw []byte, frames, logs int) providerHistoryExecutionReplayCost {
	t.Helper()
	transaction, err := block.txs[0].MarshalBinary()
	if err != nil || len(transaction) == 0 {
		t.Fatal("finite canonical transaction precursor", err)
	}
	receipt, err := block.receipts[0].MarshalBinary()
	if err != nil || len(receipt) == 0 {
		t.Fatal("finite canonical receipt precursor", err)
	}
	p := providerHistoryExecutionReplayLimits{max(4, uint64(len(raw))), uint64(len(transaction)), 64 << 20, uint64(frames), uint64(logs), uint64(len(receipt))}
	cost, ok := providerHistoryExecutionReplayUpper(p)
	if !ok {
		t.Fatal("finite complete execution bound")
	}
	return cost
}

func TestProviderHistoryExecutionReplayPreservesCallCreateAndChildLogJoins(t *testing.T) {
	for _, mode := range []string{"call", "create", "committed-child", "uncommitted-child", "static-child"} {
		e, block, raw := historyExecutionBoundFixture(t, mode, 1)
		before := bytes.Clone(raw)
		frames, logs := 1+len(e.Root.Calls), 1
		if mode == "committed-child" || mode == "uncommitted-child" {
			logs = 2
		}
		cost := historyExecutionBoundQuote(t, block, raw, frames, logs)
		m := providerHistoryContentUnitMeter(cost.envelope)
		limits := replayTestLimits()
		limits.MaxFrames, limits.MaxLogs = frames, logs
		b := replayBudget{limits: limits, observer: m}
		got, err := replayExecutionObserved(context.Background(), &e, block, 0, &b)
		if err != nil || m.failed || m.clocks != 0 || len(got) != len(block.receipts[0].Logs) ||
			b.frames != frames || b.traceLogs != logs || !bytes.Equal(before, raw) || m.actual[5] != uint64(frames) || m.actual[6] != uint64(logs) {
			t.Fatal("whole execution or committed log coverage changed", mode, err, m.actual, b.frames, b.traceLogs)
		}
		for i, value := range got {
			want := block.receipts[0].Logs[i]
			wantFrame := &e.Root
			if mode == "committed-child" && i >= len(e.Root.Logs) {
				wantFrame = &e.Root.Calls[0]
			}
			if value.log == nil || value.log.Address != want.Address || !bytes.Equal(value.log.Data, want.Data) || !reflect.DeepEqual(value.log.Topics, want.Topics) || value.eligible != (mode != "create") || value.frame != wantFrame {
				t.Fatal("root/child log order, bytes or eligibility changed", mode, i)
			}
		}
		historyEventQueueWithin(t, m.actual, cost.normal)
	}
}

func TestProviderHistoryExecutionReplayFundsCompleteOutputGrowth(t *testing.T) {
	for _, count := range []int{0, 1, 16, 17, 32, 33} {
		e, block, raw := historyExecutionBoundFixture(t, "call", count)
		maximum := max(1, count)
		cost := historyExecutionBoundQuote(t, block, raw, 1, maximum)
		m := providerHistoryContentUnitMeter(cost.envelope)
		limits := replayTestLimits()
		limits.MaxFrames, limits.MaxLogs = 1, maximum
		b := replayBudget{limits: limits, observer: m}
		got, err := replayExecutionObserved(context.Background(), &e, block, 0, &b)
		if err != nil || m.failed || m.clocks != 0 || len(got) != count || m.actual[5] != 1 || m.actual[6] != uint64(count) {
			t.Fatal("whole log buffer growth exceeded the source row", count, err, m.actual)
		}
		for i := range got {
			if !bytes.Equal(got[i].log.Data, []byte{byte(i), byte(i + 1)}) || got[i].frame != &e.Root || !got[i].eligible {
				t.Fatal("ordered grown output lost original frame/data", count, i)
			}
		}
		historyEventQueueWithin(t, m.actual, cost.normal)
	}
}

func TestProviderHistoryExecutionReplayKeepsAllWorkBeforeLaterReceiptRefusal(t *testing.T) {
	const count = 32
	var first candidateWorkVector
	for _, bad := range []int{0, count - 1} {
		e, block, raw := historyExecutionBoundFixture(t, "call", count)
		block.receipts[0].Logs[bad].Data[0] ^= 1
		cost := historyExecutionBoundQuote(t, block, raw, 1, count)
		m := providerHistoryContentUnitMeter(cost.envelope)
		limits := replayTestLimits()
		limits.MaxFrames, limits.MaxLogs = 1, count
		b := replayBudget{limits: limits, observer: m}
		got, err := replayExecutionObserved(context.Background(), &e, block, 0, &b)
		if err == nil || got != nil || m.failed || m.clocks != 0 || m.actual[5] != 1 || m.actual[6] != count {
			t.Fatal("receipt mismatch returned partial execution or lost full walk", bad, err, m.actual)
		}
		if bad == 0 {
			first = m.actual
		} else if m.actual[2] != first[2] || m.actual[10] != first[10]+111*(count-1) {
			t.Fatal("later receipt comparison erased work or replayed the tree", m.actual, first)
		}
		historyEventQueueWithin(t, m.actual, cost.normal)
	}
}

func TestProviderHistoryExecutionReplayRetainsOrdinaryRootAndChildRefusals(t *testing.T) {
	for _, mode := range []string{"wrong-index", "receipt-status", "wrong-from", "wrong-value", "wrong-input", "wrong-recipient", "wrong-kind", "root-error", "wrong-child-from"} {
		count := 1
		if mode == "receipt-status" {
			count = 0
		}
		e, block, raw := historyExecutionBoundFixture(t, mode, count)
		frames, logs := 1+len(e.Root.Calls), max(1, count)
		if mode == "wrong-child-from" {
			logs++
		}
		cost := historyExecutionBoundQuote(t, block, raw, frames, logs)
		m := providerHistoryContentUnitMeter(cost.envelope)
		limits := replayTestLimits()
		limits.MaxFrames, limits.MaxLogs = frames, logs
		b := replayBudget{limits: limits, observer: m}
		got, err := replayExecutionObserved(context.Background(), &e, block, 0, &b)
		if err == nil || got != nil || m.failed || m.clocks != 0 || m.actual[10] == 0 {
			t.Fatal("original ordinary semantic guard was bypassed", mode, err, m.actual)
		}
		if mode == "wrong-child-from" && (m.actual[5] != 2 || m.actual[6] != 1) {
			t.Fatal("late child refusal lost earlier frame/log prefix", m.actual)
		}
		historyEventQueueWithin(t, m.actual, cost.normal)
	}
}

func TestProviderHistoryExecutionReplayCannotRenewOriginalSharedQuotas(t *testing.T) {
	for _, mode := range []string{"frames", "logs"} {
		count, frames, logs := 0, 1, 1
		if mode == "logs" {
			count, frames = 1, 2
		}
		e, block, raw := historyExecutionBoundFixture(t, "call", count)
		cost := historyExecutionBoundQuote(t, block, raw, frames, logs)
		upper := cost.normal
		for i := range upper {
			upper[i] *= 2
		}
		upper[10] += cost.envelope[10] - cost.normal[10]
		m := providerHistoryContentUnitMeter(upper)
		limits := replayTestLimits()
		limits.MaxFrames, limits.MaxLogs = frames, logs
		b := replayBudget{limits: limits, observer: m}
		if _, err := replayExecutionObserved(context.Background(), &e, block, 0, &b); err != nil || m.failed {
			t.Fatal("positive original quota precursor", mode, err, m.actual)
		}
		before := m.actual
		if got, err := replayExecutionObserved(context.Background(), &e, block, 0, &b); err == nil || got != nil || m.failed || m.clocks != 0 || m.actual[10] <= before[10] || b.limits != limits {
			t.Fatal("original shared quota was reset or earlier native work vanished", mode, err, m.actual, before)
		}
		if b.frames != frames || b.traceLogs != count || m.actual[5] != uint64(frames) || m.actual[6] != uint64(count) {
			t.Fatal("failed later walk changed original quota accounting", mode, b.frames, b.traceLogs, m.actual)
		}
		historyEventQueueWithin(t, m.actual, upper)
	}
}

func TestProviderHistoryExecutionReplayCannotReplaceOrRenewNativePreflight(t *testing.T) {
	e, block, raw := historyExecutionBoundFixture(t, "call", 1)
	cost := historyExecutionBoundQuote(t, block, raw, 1, 1)
	if cost.normal[10] >= 4503599627370497 {
		t.Fatal("finite fixture must distinguish actual upper from original native room")
	}
	header, err := rlp.EncodeToBytes(block.header)
	if err != nil {
		t.Fatal(err)
	}
	n := uint64(len(header))
	want := candidateWorkVector{10: n*n + n + 2}
	m := providerHistoryContentUnitMeter(cost.normal)
	limits := replayTestLimits()
	limits.MaxFrames, limits.MaxLogs = 1, 1
	b := replayBudget{limits: limits, observer: m}
	got, err := replayExecutionObserved(context.Background(), &e, block, 0, &b)
	if err == nil || got != nil || !m.failed || m.actual != want || m.clocks != 0 {
		t.Fatal("original native preflight was bypassed or lost header prefix", err, m.actual, want)
	}
	before := m.actual
	if got, err := replayExecutionObserved(context.Background(), &e, block, 0, &b); err == nil || got != nil || m.actual != before || !m.failed || m.clocks != 0 {
		t.Fatal("failed original execution observer was renewed", err, m.actual, before)
	}
}

// Topic construction copies each 32-byte word twice (64 units), then the
// final receipt join compares two words plus its event constant (65 units).
func TestProviderHistoryExecutionReplayFundsAllReceiptTopicWidths(t *testing.T) {
	const count = 17
	var previous candidateWorkVector
	for topics := 0; topics <= 4; topics++ {
		e, block, raw := historyExecutionBoundFixtureTopics(t, "call", count, topics)
		cost := historyExecutionBoundQuote(t, block, raw, 1, count)
		m := providerHistoryContentUnitMeter(cost.envelope)
		limits := replayTestLimits()
		limits.MaxFrames, limits.MaxLogs = 1, count
		b := replayBudget{limits: limits, observer: m}
		got, err := replayExecutionObserved(context.Background(), &e, block, 0, &b)
		if err != nil || m.failed || m.clocks != 0 || len(got) != count || m.actual[5] != 1 || m.actual[6] != count {
			t.Fatal("complete topic-width execution", topics, err, m.actual)
		}
		for i := range got {
			if len(got[i].log.Topics) != topics || got[i].frame != &e.Root {
				t.Fatal("topic count or frame binding changed", topics, i)
			}
			for j, topic := range got[i].log.Topics {
				if topic != block.receipts[0].Logs[i].Topics[j] {
					t.Fatal("topic identity changed", topics, i, j)
				}
			}
		}
		if topics > 0 && (m.actual[2] != previous[2] || m.actual[10] != previous[10]+129*count) {
			t.Fatal("per-topic copy and receipt comparison work missing", topics, m.actual, previous)
		}
		historyEventQueueWithin(t, m.actual, cost.normal)
		previous = m.actual
	}
}

func TestProviderHistoryExecutionReplayRetainsLaterTopicComparisonWork(t *testing.T) {
	const count = 17
	for topics := 1; topics <= 4; topics++ {
		var first candidateWorkVector
		for _, bad := range []int{0, count - 1} {
			e, block, raw := historyExecutionBoundFixtureTopics(t, "call", count, topics)
			block.receipts[0].Logs[bad].Topics[topics-1][31] ^= 1
			cost := historyExecutionBoundQuote(t, block, raw, 1, count)
			m := providerHistoryContentUnitMeter(cost.envelope)
			limits := replayTestLimits()
			limits.MaxFrames, limits.MaxLogs = 1, count
			b := replayBudget{limits: limits, observer: m}
			got, err := replayExecutionObserved(context.Background(), &e, block, 0, &b)
			if err == nil || got != nil || m.failed || m.clocks != 0 || m.actual[5] != 1 || m.actual[6] != count {
				t.Fatal("topic mismatch returned partial execution or lost full walk", topics, bad, err, m.actual)
			}
			if bad == 0 {
				first = m.actual
			} else if m.actual[2] != first[2] || m.actual[10] != first[10]+uint64((46+65*topics)*(count-1)) {
				t.Fatal("later topic refusal lost preceding receipt comparisons", topics, m.actual, first)
			}
			historyEventQueueWithin(t, m.actual, cost.normal)
		}
	}
}
