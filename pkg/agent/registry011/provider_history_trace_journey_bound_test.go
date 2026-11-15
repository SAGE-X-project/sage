package registry011

import (
	"bytes"
	"context"
	"math/big"
	"reflect"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestProviderHistoryTraceJourneyMatchesIndependentPortableRows(t *testing.T) {
	w := uint64(strconv.IntSize / 8)
	s := uint64(reflect.TypeFor[types.Transaction]().Size()) + uint64(reflect.TypeFor[types.BlobTx]().Size()) + 2
	j := uint64(reflect.TypeFor[big.Int]().Size())
	r := uint64(reflect.TypeFor[replayLog]().Size())
	l := uint64(reflect.TypeFor[types.Log]().Size())
	e := uint64(reflect.TypeFor[ReplayExecution]().Size())
	for _, row := range []struct {
		limits        providerHistoryTraceJourneyLimits
		pools         [6]uint64
		normalization [5]uint64
		execution     [9]uint64
		assembly      [2]uint64
		room          uint64
	}{
		{providerHistoryTraceJourneyLimits{1, 1, 1, 65536, 1024, 512, 2048, 2, 16, 16, 1}, [6]uint64{1, 1, 1, 1024, 512, 65536}, [5]uint64{146432, 16, 16, 146025428, 423888}, [9]uint64{1825, 16, 16, 4296102931, 0, 1, 2, 128, 16}, [2]uint64{1026, 1}, 5308417},
		{providerHistoryTraceJourneyLimits{1, 2, 2, 1, 64, 1, 1, 16, 1, 1, 20}, [6]uint64{2, 1, 20, 2, 1, 1}, [5]uint64{596, 1, 1, 50507, 887}, [9]uint64{265, 1, 1, 4295141238, 2, 1, 2, 23, 1}, [2]uint64{132, 20}, 66050},
		{providerHistoryTraceJourneyLimits{2, 10, 2, 4096, 8192, 1024, 67108864, 4, 8, 8, 40}, [6]uint64{17, 8, 16, 69632, 8192, 8192}, [5]uint64{9614336, 16, 16, 78775111700, 28758992}, [9]uint64{2952, 16, 16, 35456129112, 0, 8, 16, 240, 16}, [2]uint64{139291, 16}, 4503633987174401},
		{providerHistoryTraceJourneyLimits{3, 12, 2, 65536, 4096, 1024, 67108864, 4, 64, 128, 10}, [6]uint64{10, 10, 10, 40960, 10240, 196608}, [5]uint64{5763072, 192, 384, 23194933032, 16936256}, [9]uint64{29258, 192, 384, 44321830014, 0, 10, 20, 2848, 384}, [2]uint64{40980, 10}, 4503633987174401},
		{providerHistoryTraceJourneyLimits{1, 100, 10, 4096, 8192, 2048, 2048, 4, 4, 4, 100}, [6]uint64{5, 4, 44, 20480, 4096, 4096}, [5]uint64{2827520, 4, 4, 23169128456, 8458484}, [9]uint64{1060, 4, 4, 17199571644, 0, 4, 8, 92, 4}, [2]uint64{41009, 44}, 5308417},
		{providerHistoryTraceJourneyLimits{1, 1, 1, 65536, 1024, 1, 67108864, 1, 1, 1, 0}, [6]uint64{0, 0, 0, 0, 0, 0}, [5]uint64{0, 0, 0, 0, 0}, [9]uint64{0, 0, 0, 0, 0, 0, 0, 0, 0}, [2]uint64{0, 0}, 0},
		{providerHistoryTraceJourneyLimits{100, 100, 10, 67108864, 8388608, 8388608, 67108864, 16384, 65536, 65536, 100}, [6]uint64{100, 100, 100, 838860800, 838860800, 6710886400}, [5]uint64{117859942400, 6553600, 6553600, 971089547919360400, 346849280000}, [9]uint64{681590500, 6553600, 6553600, 14074678074017900, 0, 100, 200, 45876800, 6553600}, [2]uint64{838861000, 100}, 4503633987174401},
	} {
		got, ok := providerHistoryTraceJourneyUpper(row.limits)
		n, x := row.normalization, row.execution
		want := providerHistoryTraceJourneyCost{
			normalization:         candidateWorkVector{2: n[0], 5: n[1], 6: n[2], 10: n[3] + n[4]*w},
			execution:             candidateWorkVector{2: x[0], 5: x[1], 6: x[2], 10: x[3] + x[4]*w + x[5]*s + x[6]*j + x[7]*r + x[8]*l},
			assembly:              candidateWorkVector{10: row.assembly[0] + row.assembly[1]*e},
			normalizationAttempts: row.pools[0], executionRoots: row.pools[1], executionSlots: row.pools[2],
			normalizationRaw: row.pools[3], canonicalPool: row.pools[4], receiptPool: row.pools[5],
		}
		for i := range want.normal {
			want.normal[i] = want.normalization[i] + want.execution[i] + want.assembly[i]
		}
		want.envelope = want.normal
		want.envelope[10] += row.room
		if !ok || got != want {
			t.Fatal("independent portable complete trace row", row, got, want, ok)
		}
	}
}

func TestProviderHistoryTraceJourneyRejectsInvalidAndOverflowingData(t *testing.T) {
	valid := providerHistoryTraceJourneyLimits{2, 4, 2, 65536, 4096, 512, 2048, 4, 8, 8, 10}
	for _, mode := range []string{"blocks-zero", "blocks-large", "windows-zero", "windows-large", "pending-zero", "pending-large", "evidence-zero", "evidence-large", "body-small", "body-large", "value-zero", "value-evidence", "value-body", "value-native", "native-zero", "native-large", "transactions-zero", "transactions-large", "frames-zero", "frames-large", "logs-zero", "logs-large", "effects-large", "late-overflow"} {
		p := valid
		switch mode {
		case "blocks-zero":
			p.Blocks = 0
		case "blocks-large":
			p.Blocks = (1 << 20) + 1
		case "windows-zero":
			p.Windows = 0
		case "windows-large":
			p.Windows = p.Blocks + 1
		case "pending-zero":
			p.PendingBlocks = 0
		case "pending-large":
			p.PendingBlocks = 11
		case "evidence-zero":
			p.EvidenceBytes = 0
		case "evidence-large":
			p.EvidenceBytes = (64 << 20) + 1
		case "body-small":
			p.BodyBytes = 3
		case "body-large":
			p.BodyBytes = (8 << 20) + 1
		case "value-zero":
			p.ValueBytes = 0
		case "value-evidence":
			p.EvidenceBytes = p.ValueBytes - 1
		case "value-body":
			p.BodyBytes = p.ValueBytes - 1
		case "value-native":
			p.TransactionBytes = p.ValueBytes - 1
		case "native-zero":
			p.TransactionBytes = 0
		case "native-large":
			p.TransactionBytes = (64 << 20) + 1
		case "transactions-zero":
			p.Transactions = 0
		case "transactions-large":
			p.Transactions = 16385
		case "frames-zero":
			p.Frames = 0
		case "frames-large":
			p.Frames = 65537
		case "logs-zero":
			p.Logs = 0
		case "logs-large":
			p.Logs = 65537
		case "effects-large":
			p.Effects = (1 << 20) + 1
		case "late-overflow":
			p = providerHistoryTraceJourneyLimits{1 << 20, 1 << 20, 10, 64 << 20, 8 << 20, 8 << 20, 64 << 20, 16384, 65536, 65536, 1 << 20}
		}
		if got, ok := providerHistoryTraceJourneyUpper(p); ok || got != (providerHistoryTraceJourneyCost{}) {
			t.Fatal("invalid or late overflow leaked partial trace cost", mode, got, ok)
		}
	}
	for i := range 11 {
		p := valid
		fields := []*uint64{&p.Windows, &p.Blocks, &p.PendingBlocks, &p.EvidenceBytes, &p.BodyBytes, &p.ValueBytes, &p.TransactionBytes, &p.Transactions, &p.Frames, &p.Logs, &p.Effects}
		*fields[i] = ^uint64(0)
		if got, ok := providerHistoryTraceJourneyUpper(p); ok || got != (providerHistoryTraceJourneyCost{}) {
			t.Fatal("MAX field leaked partial trace cost", i, got, ok)
		}
	}
}

func TestProviderHistoryTraceJourneyRecombinesClosedSingleWindowChildren(t *testing.T) {
	p := providerHistoryTraceJourneyLimits{1, 1, 1, 65536, 1024, 512, 2048, 2, 16, 16, 1}
	got, ok := providerHistoryTraceJourneyUpper(p)
	n, a := providerHistoryTraceNormalizationRefusalUpper(providerHistoryTraceNormalizationLimits{1024, 1024, 1, 16, 16})
	x, b := providerHistoryExecutionReplayUpper(providerHistoryExecutionReplayLimits{1024, 512, 2048, 16, 16, 65536})
	if !ok || !a || !b || got.normalization != n || got.execution != x.normal || got.normal[5] != 32 || got.normal[6] != 32 {
		t.Fatal("normalization and semantic passes lost original work", got, n, x)
	}
	if got.envelope[10]-got.normal[10] != 5308417 || x.envelope[10]-x.normal[10] != 5308417 {
		t.Fatal("original current native incoming room changed", got)
	}
	// This private numerical identity is neither source authentication nor a
	// refund of initializer plus SUM FULL admitted stage costs.
}

func TestProviderHistoryTraceJourneyKeepsZeroEffectsEmpty(t *testing.T) {
	p := providerHistoryTraceJourneyLimits{1, 1, 1, 65536, 1024, 1, 64 << 20, 1, 1, 1, 0}
	if got, ok := providerHistoryTraceJourneyUpper(p); !ok || got != (providerHistoryTraceJourneyCost{}) {
		t.Fatal("zero effects fabricated trace work", got, ok)
	}
}

// Finite offline invocation DATA, not a whole admitted original cursor or
// acquisition/chain/Source witness. Native encodings below supply fixtures
// before the target observer. Its original inert selector still uses64MiB.
func historyTraceJourneyFixtureQuote(t *testing.T, raw []byte, block replayDecodedBlock, limits ReplayLimits) (providerHistoryTraceJourneyCost, int) {
	t.Helper()
	var pool, largest uint64
	for _, tx := range block.txs {
		binary, err := tx.MarshalBinary()
		if err != nil || len(binary) == 0 {
			t.Fatal("canonical private transaction precursor", err)
		}
		pool += uint64(len(binary))
		largest = max(largest, uint64(len(binary)))
	}
	header, err := rlp.EncodeToBytes(block.header)
	if err != nil || len(header) > 65536 {
		t.Fatal("canonical private header precursor", err)
	}
	canonical := len(header)
	for i, tx := range block.txs {
		transaction, txErr := tx.MarshalBinary()
		receipt, receiptErr := block.receipts[i].MarshalBinary()
		if txErr != nil || receiptErr != nil {
			t.Fatal("complete private canonical rows", txErr, receiptErr)
		}
		canonical += len(transaction) + len(receipt)
	}
	p := providerHistoryTraceJourneyLimits{1, 1, 1, uint64(limits.MaxEvidenceBytes), max(1024, uint64(len(raw)), pool), max(1, largest), 64 << 20, uint64(limits.MaxTransactions), uint64(limits.MaxFrames), uint64(limits.MaxLogs), 1}
	cost, ok := providerHistoryTraceJourneyUpper(p)
	if !ok || canonical >= limits.MaxEvidenceBytes {
		t.Fatal("finite original private invocation quote", p, ok, canonical)
	}
	return cost, canonical
}

func TestProviderHistoryTraceJourneyRunsNormalizationAndCanonicalExecution(t *testing.T) {
	for _, mode := range []string{"call", "create", "committed-child", "uncommitted-child", "static-child"} {
		original, block, raw := historyExecutionBoundFixture(t, mode, 1)
		before := bytes.Clone(raw)
		frames, logs := 1+len(original.Root.Calls), 1
		if mode == "committed-child" || mode == "uncommitted-child" {
			logs = 2
		}
		limits := replayTestLimits()
		limits.MaxEvidenceBytes = 1 << 20
		limits.MaxTransactions = 1
		limits.MaxFrames = frames
		limits.MaxLogs = logs
		cost, canonical := historyTraceJourneyFixtureQuote(t, raw, block, limits)
		m := providerHistoryContentUnitMeter(cost.envelope)
		normalization := replayBudget{limits: limits, observer: m}
		semantic := replayBudget{limits: limits, bytes: canonical, txs: 1, observer: m}
		if !m.event(providerCopyEvent, uint64(reflect.TypeFor[ReplayExecution]().Size())) {
			t.Fatal("original one-root array initialization")
		}
		executions := make([]ReplayExecution, 1)
		copied := providerReplayBytes(string(raw), m)
		frame, err := decodeCallTracerBudgetObserved(context.Background(), copied, &normalization)
		if err != nil || !reflect.DeepEqual(frame, original.Root) {
			t.Fatal("whole original normalized tree", mode, err)
		}
		executions[0] = ReplayExecution{original.BlockHash, original.TransactionHash, original.TransactionIndex, frame}
		got, err := replayExecutionObserved(context.Background(), &executions[0], block, 0, &semantic)
		if err != nil || len(got) != len(block.receipts[0].Logs) || m.failed || m.clocks != 0 || !bytes.Equal(raw, before) || !bytes.Equal(copied, before) ||
			normalization.frames != frames || semantic.frames != frames || normalization.traceLogs != logs || semantic.traceLogs != logs || semantic.bytes <= canonical ||
			m.actual[5] != 2*uint64(frames) || m.actual[6] != 2*uint64(logs) {
			t.Fatal("normalization and execution lost separate original counters", mode, err, m.actual)
		}
		historyEventQueueWithin(t, m.actual, cost.normal)
	}
}

func TestProviderHistoryTraceJourneyKeepsWideOriginalNativeIncoming(t *testing.T) {
	e, block, raw := historyExecutionBoundFixture(t, "call", 1)
	limits := replayTestLimits()
	limits.MaxEvidenceBytes = 1 << 20
	limits.MaxTransactions = 1
	limits.MaxFrames = 1
	limits.MaxLogs = 1
	cost, canonical := historyTraceJourneyFixtureQuote(t, raw, block, limits)
	if cost.normal[10] >= 4503633987174401 {
		t.Fatal("small private actual-only row unexpectedly funds original wide incoming")
	}
	m := providerHistoryContentUnitMeter(cost.normal)
	normalization := replayBudget{limits: limits, observer: m}
	frame, err := decodeCallTracerBudgetObserved(context.Background(), raw, &normalization)
	if err != nil || normalization.frames != 1 || normalization.traceLogs != 1 {
		t.Fatal("ordinary normalization before native guard", err, m.actual)
	}
	e.Root = frame
	semantic := replayBudget{limits: limits, bytes: canonical, txs: 1, observer: m}
	got, err := replayExecutionObserved(context.Background(), &e, block, 0, &semantic)
	if err == nil || got != nil || !m.failed || m.clocks != 0 || semantic.frames != 0 || m.actual[5] != 1 || m.actual[2] == 0 {
		t.Fatal("actual-only quote bypassed original encoding guard", err, m.actual)
	}
	historyEventQueueWithin(t, m.actual, cost.normal)
}

func TestProviderHistoryTraceJourneyChargesRawCopyBeforeEvidenceRefusal(t *testing.T) {
	raw := historyNormalizationTrace(t, historyNormalizationLeaf(false))
	if len(raw) < 5 {
		t.Fatal("ordinary complete raw precursor")
	}
	limits := replayTestLimits()
	limits.MaxEvidenceBytes = len(raw) - 1
	limits.MaxTransactions = 1
	limits.MaxFrames = 1
	limits.MaxLogs = 1
	p := providerHistoryTraceJourneyLimits{1, 1, 1, uint64(limits.MaxEvidenceBytes), uint64(len(raw)), 1, 64 << 20, 1, 1, 1, 1}
	cost, ok := providerHistoryTraceJourneyUpper(p)
	if !ok || cost.normalizationRaw >= uint64(len(raw)) {
		t.Fatal("raw internal cap does not distinguish earlier copy", cost, ok)
	}
	m := providerHistoryContentUnitMeter(cost.envelope)
	budget := replayBudget{limits: limits, observer: m}
	copied := providerReplayBytes(string(raw), m)
	got, err := decodeCallTracerBudgetObserved(context.Background(), copied, &budget)
	want := candidateWorkVector{10: uint64(len(raw)) + 1}
	if err == nil || !reflect.DeepEqual(got, ReplayFrame{}) || m.failed || m.clocks != 0 || m.actual != want || budget.frames != 0 || budget.traceLogs != 0 || budget.limits != limits || !bytes.Equal(raw, copied) {
		t.Fatal("later evidence guard erased entered copy", err, m.actual, want)
	}
	historyEventQueueWithin(t, m.actual, cost.normal)
}

func TestProviderHistoryTraceJourneyRetainsFinalSharedWindowRawPrefix(t *testing.T) {
	raw := historyNormalizationTrace(t, historyNormalizationLeaf(false))
	limits := replayTestLimits()
	limits.MaxEvidenceBytes = 1 << 20
	limits.MaxTransactions = 2
	limits.MaxFrames = 2
	limits.MaxLogs = 1
	p := providerHistoryTraceJourneyLimits{1, 3, 1, 1 << 20, max(1024, uint64(len(raw))), 1, 64 << 20, 2, 2, 1, 3}
	cost, ok := providerHistoryTraceJourneyUpper(p)
	if !ok || cost.normalizationAttempts != 3 {
		t.Fatal("original two-frame plus terminal raw attempt")
	}
	m := providerHistoryContentUnitMeter(cost.envelope)
	b := replayBudget{limits: limits, observer: m}
	var before candidateWorkVector
	for attempt := 0; attempt < 3; attempt++ {
		if attempt == 2 {
			before = m.actual
		}
		copied := providerReplayBytes(string(raw), m)
		_, err := decodeCallTracerBudgetObserved(context.Background(), copied, &b)
		if (err == nil) != (attempt < 2) || m.failed || !bytes.Equal(raw, copied) {
			t.Fatal("original shared capacity sequence", attempt, err, m.actual)
		}
	}
	n := uint64(len(raw))
	want := candidateWorkVector{2: 2 * n, 10: 2*n*n + 3*n + 5}
	var delta candidateWorkVector
	for i := range delta {
		delta[i] = m.actual[i] - before[i]
	}
	if delta != want || b.frames != 2 || b.traceLogs != 0 || b.limits != limits || m.actual[5] != 2 || m.clocks != 0 {
		t.Fatal("last raw prefix lost earlier original window usage", delta, want, m.actual)
	}
	historyEventQueueWithin(t, m.actual, cost.normal)
	// Stop at this first ordinary error; no failed original S is retried.
}

func TestProviderHistoryTraceJourneyInitializesDeclaredRootsBeforeFirstFailure(t *testing.T) {
	// These existing synthetic deployment-zero cursors cannot pass production
	// admission. This isolated ORIGINAL transition has a broad inert meter;
	// its outer content/reconstruction events are NOT funded by a trace quote.
	control, input, methods := historyCollectionContentInputFixture(t, 2, historyCollectionStreamingInput(t), SepoliaAmsterdamRootExecutionGasProfile)
	control = historyCollectionContentStep(t, control, input, methods)
	control = historyCollectionContentStep(t, control, input, methods)
	control = historyCollectionContentStep(t, control, input, methods)
	if control.pendingApplied != 1 || control.semantic.blocks != 1 {
		t.Fatal("normal first complete ready-block precursor")
	}
	c, in, methods := historyCollectionContentInputFixture(t, 2, historyCollectionStreamingInput(t), SepoliaAmsterdamRootExecutionGasProfile)
	c = historyCollectionContentStep(t, c, in, methods)
	c = historyCollectionContentStep(t, c, in, methods)
	if c.phase != "effects" || len(c.pending) != 2 || len(c.effects) != 40 || c.pendingApplied != 0 || c.semantic.blocks != 0 {
		t.Fatal("ordinary two-block pending precursor")
	}
	table := historyCollectionBoundaryPlan(t, c, methods)
	results := historyCollectionContentResults(t, c, in, table)
	if len(results) != 32 || c.effects[0].number != c.effects[1].number || c.effects[1].number == c.effects[2].number {
		t.Fatal("two first-block roots with complete first group")
	}
	results[0] = historyCollectionBoundaryResult(t, []byte("true"))
	block := c.pending[0]
	if block.gasProfile != "" || block.gasTrace != "" || len(block.transactions) != 2 || len(block.receipts) != 2 {
		t.Fatal("ordinary pre-activation reconstruction precursor")
	}
	rawBytes := uint64(len(block.headerRLP))
	for i := range block.transactions {
		rawBytes += uint64(len(block.transactions[i]) + len(block.receipts[i]))
	}
	w := uint64(strconv.IntSize / 8)
	reconstruction := rawBytes + 12*w + 7
	retained := uint64(len(results)) * (uint64(reflect.TypeFor[string]().Size()) + 1)
	effects := 9 * uint64(len(c.effects))
	declared := 2*uint64(reflect.TypeFor[ReplayExecution]().Size()) + 1
	// copy(true)=5, two complete raw validation codecs/scans=44, then
	// frameObserved1 and nodeObjectObserved scan(true)=5 before refusal.
	want := candidateWorkVector{2: 8, 5: 1, 10: retained + effects + reconstruction + declared + 54}
	m := c.semantic.observer
	before := m.actual
	if c.contentBoundaryObserved(context.Background(), results, m) {
		t.Fatal("ordinary scalar trace unexpectedly parsed as a frame")
	}
	var delta candidateWorkVector
	for i := range delta {
		delta[i] = m.actual[i] - before[i]
	}
	if delta != want || m.failed || m.clocks != 0 || c.effectNext != 32 || c.traces != 32 || c.pendingApplied != 0 || c.semantic.blocks != 0 ||
		c.semantic.window.budget.frames != 0 || c.semantic.window.budget.traceLogs != 0 || c.traceResults[0] != "true" {
		t.Fatal("declared two-root allocation or failed prefix was erased", delta, want)
	}
	// Stop at the first failed private transition. No source state is published.
}

func TestProviderHistoryTraceJourneyRejectsLastReceiptTopicAfterWholeTraversal(t *testing.T) {
	var successful candidateWorkVector
	for _, refuse := range []bool{false, true} {
		original, block, raw := historyExecutionBoundFixtureTopics(t, "call", 17, 4)
		if len(block.receipts[0].Logs) != 17 || len(block.receipts[0].Logs[16].Topics) != 4 {
			t.Fatal("complete receipt/topic precursor")
		}
		if refuse {
			block.receipts[0].Logs[16].Topics[3][0] ^= 1
		}
		limits := replayTestLimits()
		limits.MaxEvidenceBytes = 1 << 20
		limits.MaxTransactions = 1
		limits.MaxFrames = 1
		limits.MaxLogs = 17
		cost, canonical := historyTraceJourneyFixtureQuote(t, raw, block, limits)
		m := providerHistoryContentUnitMeter(cost.envelope)
		normal := replayBudget{limits: limits, observer: m}
		frame, err := decodeCallTracerBudgetObserved(context.Background(), raw, &normal)
		if err != nil {
			t.Fatal("complete raw tree before final receipt comparison", err)
		}
		original.Root = frame
		semantic := replayBudget{limits: limits, bytes: canonical, txs: 1, observer: m}
		got, err := replayExecutionObserved(context.Background(), &original, block, 0, &semantic)
		if (err != nil) != refuse || refuse && got != nil || !refuse && len(got) != 17 || m.failed || m.clocks != 0 || normal.frames != 1 || semantic.frames != 1 || normal.traceLogs != 17 || semantic.traceLogs != 17 || m.actual[5] != 2 || m.actual[6] != 34 {
			t.Fatal("final receipt join did not follow complete traversal", refuse, err, m.actual)
		}
		if !refuse {
			successful = m.actual
		} else if m.actual != successful {
			t.Fatal("last fourth-topic comparison erased an already entered prefix", m.actual, successful)
		}
		historyEventQueueWithin(t, m.actual, cost.normal)
	}
}
