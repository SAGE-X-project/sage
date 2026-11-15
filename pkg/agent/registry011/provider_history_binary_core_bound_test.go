package registry011

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
)

type historyBinaryCoreShape struct {
	work                                    candidateWorkVector
	bytes, transactions, logs, maximumValue uint64
}

// Independent original-byte dimensions and native receipt log counts; no
// quote, observed actual vector or live operating profile supplies this row.
func historyBinaryCoreOriginalShape(t *testing.T, raw ReplayBlock) historyBinaryCoreShape {
	t.Helper()
	if len(raw.HeaderRLP) == 0 || len(raw.Transactions) != len(raw.Receipts) {
		t.Fatal("ordinary complete original block required")
	}
	var out historyBinaryCoreShape
	add := func(n uint64) {
		out.bytes += n
		out.work[2] += n
		out.work[10] += 2*n*n + 2*n + 3
	}
	add(uint64(len(raw.HeaderRLP)))
	out.transactions = uint64(len(raw.Transactions))
	for i, tx := range raw.Transactions {
		receipt := raw.Receipts[i]
		add(uint64(len(tx)))
		add(uint64(len(receipt)))
		out.maximumValue = max(out.maximumValue, uint64(len(tx)), uint64(len(receipt)))
		var parsed types.Receipt
		if parsed.UnmarshalBinary(receipt) != nil {
			t.Fatal("independent original receipt")
		}
		out.logs += uint64(len(parsed.Logs))
	}
	out.work[3], out.work[4], out.work[6] = 1, out.transactions, out.logs
	out.work[10] += 130 // Two32-byte root-to-header comparisons,65 each.
	return out
}

func TestProviderHistoryBinaryCoreUpperMatchesIndependentWholeVectors(t *testing.T) {
	for _, tc := range []struct {
		g, r, blocks, windows, evidence, txs, logs, value uint64
		maximum, passes, windowPasses, blockPasses        uint64
		work                                              candidateWorkVector
	}{
		{1, 1, 1, 1, 1024, 1, 1, 1024, 1024, 30, 30, 30, candidateWorkVector{2: 30720, 3: 30, 4: 30, 6: 30, 10: 62980170}},
		{2, 3, 5, 2, 4096, 2, 10, 1024, 4096, 58, 116, 290, candidateWorkVector{2: 475136, 3: 290, 4: 232, 6: 1160, 10: 3893304346}},
		{1, 32, 3, 3, 64 << 20, 16384, 65536, 8 << 20, 8 << 20, 30, 90, 90, candidateWorkVector{2: 6039797760, 3: 90, 4: 1474560, 6: 5898240, 10: 101331003704291010}},
		{1, 1, 1, 1, 131072, 1, 1, 1, 65536, 30, 30, 30, candidateWorkVector{2: 3932160, 3: 30, 4: 30, 6: 30, 10: 515403944010}},
		{1, 1, 1, 1, 131072, 1, 1, 65537, 65537, 30, 30, 30, candidateWorkVector{2: 3932160, 3: 30, 4: 30, 6: 30, 10: 515411808330}},
	} {
		got, ok := providerHistoryBinaryCoreUpper(providerHistoryCollectionSemanticLimits{
			Groups: tc.g, Requests: tc.r, Blocks: tc.blocks, Windows: tc.windows,
		}, tc.evidence, tc.txs, tc.logs, tc.value)
		if !ok || got.maximumValue != tc.maximum || got.passes != tc.passes || got.windows != tc.windowPasses ||
			got.blocks != tc.blockPasses || got.total != tc.work {
			t.Fatal("independent complete binary-core counts and polynomial", got, ok)
		}
	}
}

func TestProviderHistoryBinaryCoreMeasuresCanonicalRowsAndSeparateRootComputations(t *testing.T) {
	in := replayTestInput(t)
	for _, raw := range in.Blocks[:3] {
		shape := historyBinaryCoreOriginalShape(t, raw)
		m := providerHistorySemanticUnitMeter()
		before := m.actual
		if _, err := providerHistoryContentRootObserved(context.Background(), raw.Transactions, m); err != nil {
			t.Fatal("ordinary independent original transaction root", err)
		}
		if _, err := providerHistoryContentRootObserved(context.Background(), raw.Receipts, m); err != nil {
			t.Fatal("ordinary independent original receipt root", err)
		}
		roots := historyEventQueueDelta(t, m, before)
		before = m.actual
		budget := replayBudget{limits: replayTestLimits(), observer: m}
		decoded, err := replayDecodeContent(context.Background(), raw, &budget)
		want := shape.work
		for i, n := range roots {
			want[i] += n
		}
		if err != nil || decoded.header == nil || len(decoded.txs) != len(raw.Transactions) || len(decoded.receipts) != len(raw.Receipts) ||
			historyEventQueueDelta(t, m, before) != want || budget.bytes != int(shape.bytes) || budget.txs != int(shape.transactions) || budget.logs != int(shape.logs) {
			t.Fatal("actual canonical rows omit root comparisons or original charges", err, m.actual, want)
		}
	}
}

func TestProviderHistoryBinaryCoreUpperFundsBothCorePassesOnTheSameObserver(t *testing.T) {
	blocks := replayTestInput(t).Blocks[:3]
	var evidence, txs, logs, value uint64
	for _, raw := range blocks {
		s := historyBinaryCoreOriginalShape(t, raw)
		evidence += s.bytes
		txs += s.transactions
		logs += s.logs
		value = max(value, s.maximumValue)
	}
	limits := providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 3, Windows: 1}
	core, ok := providerHistoryBinaryCoreUpper(limits, evidence, txs, logs, value)
	roots, rootsOK := providerHistoryCollectionTrieUpper(limits, evidence, txs, value, EthereumJSONBlocks)
	if !ok || !rootsOK {
		t.Fatal("finite ordinary original DATA ceilings")
	}
	upper := core.total
	for i, n := range roots.total {
		upper[i] += n
	}
	m := providerHistoryContentUnitMeter(upper)
	l := replayTestLimits()
	l.MaxEvidenceBytes, l.MaxTransactions, l.MaxLogs = int(evidence), int(txs), int(logs)
	// Acquisition has a temporary budget per block. The later semantic
	// pass retains ONE original whole-window budget. Both use the same
	// inert UNIT observer; no live owner, allowance or clock is reset.
	for _, raw := range blocks {
		budget := replayBudget{limits: l, observer: m}
		if _, err := replayDecodeContent(context.Background(), raw, &budget); err != nil {
			t.Fatal("ordinary acquisition core", err)
		}
	}
	semantic := replayBudget{limits: l, observer: m}
	for _, raw := range blocks {
		if _, err := replayDecodeContent(context.Background(), raw, &semantic); err != nil {
			t.Fatal("ordinary same-window semantic core", err)
		}
	}
	if m.failed || semantic.bytes != int(evidence) || semantic.txs != int(txs) || semantic.logs != int(logs) ||
		m.actual[2] != 2*evidence || m.actual[3] != 6 || m.actual[4] != 2*txs || m.actual[6] != 2*logs {
		t.Fatal("two original core passes lost shared bytes, counts or observer", m.actual)
	}
	historyEventQueueWithin(t, m.actual, upper)
}

func TestProviderHistoryBinaryCoreLastHeaderComparisonRefusalKeepsEnteredWork(t *testing.T) {
	raw := replayTestInput(t).Blocks[0]
	shape := historyBinaryCoreOriginalShape(t, raw)
	h := uint64(len(raw.HeaderRLP))
	// Header codec+encode have entered. The final2h+1 equality lacks one
	// unit; no transaction decoder or trie computation has been entered.
	m := providerHistoryContentUnitMeter(candidateWorkVector{2: h, 3: 1, 10: 2*h*h + 2*h + 2})
	budget := replayBudget{limits: replayTestLimits(), observer: m}
	out, err := replayDecodeContent(context.Background(), raw, &budget)
	want := candidateWorkVector{2: h, 3: 1, 10: 2*h*h + 2}
	if err == nil || out.header != nil || out.txs != nil || out.receipts != nil || !m.failed || m.actual != want ||
		budget.bytes != int(shape.bytes) || budget.txs != int(shape.transactions) || budget.logs != 0 {
		t.Fatal("last header equality refusal lost original charged prefix", err, m.actual, want)
	}
	if _, retry := replayDecodeContent(context.Background(), raw, &budget); retry == nil || m.actual != want {
		t.Fatal("failed original observer resumed or refunded its entered work")
	}
}

func TestProviderHistoryBinaryCoreWholeWindowByteLimitIsNeverRenewed(t *testing.T) {
	blocks := replayTestInput(t).Blocks[:2]
	shape := historyBinaryCoreOriginalShape(t, blocks[0])
	l := replayTestLimits()
	l.MaxEvidenceBytes = int(shape.bytes)
	m := providerHistorySemanticUnitMeter()
	budget := replayBudget{limits: l, observer: m}
	if _, err := replayDecodeContent(context.Background(), blocks[0], &budget); err != nil {
		t.Fatal("normal first complete original block", err)
	}
	before, allowance := m.actual, m.upper
	for range 2 {
		out, err := replayDecodeContent(context.Background(), blocks[1], &budget)
		if err == nil || out.header != nil || out.txs != nil || out.receipts != nil || m.actual != before || m.upper != allowance ||
			budget.bytes != int(shape.bytes) || budget.txs != int(shape.transactions) {
			t.Fatal("next ordinary block renewed the exhausted original window", err, m.actual)
		}
	}
}

func TestProviderHistoryBinaryCoreUpperInvalidDomainsReturnWholeZero(t *testing.T) {
	base := providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1}
	for _, mode := range []string{"groups", "requests", "blocks", "windows", "evidence-zero", "evidence-wide", "tx-zero", "tx-wide", "logs-zero", "logs-wide", "value-zero", "value-wide"} {
		l, evidence, txs, logs, value := base, uint64(1024), uint64(1), uint64(1), uint64(1)
		switch mode {
		case "groups":
			l.Groups = 0
		case "requests":
			l.Requests = 33
		case "blocks":
			l.Blocks = 0
		case "windows":
			l.Windows = 2
		case "evidence-zero":
			evidence = 0
		case "evidence-wide":
			evidence = 64<<20 + 1
		case "tx-zero":
			txs = 0
		case "tx-wide":
			txs = 16385
		case "logs-zero":
			logs = 0
		case "logs-wide":
			logs = 65537
		case "value-zero":
			value = 0
		case "value-wide":
			value = evidence + 1
		}
		if got, ok := providerHistoryBinaryCoreUpper(l, evidence, txs, logs, value); ok || got != (providerHistoryBinaryCoreCost{}) {
			t.Fatal("invalid binary-core domain returned a partial quote", mode, got, ok)
		}
	}
}

func TestProviderHistoryBinaryCoreUpperLateLogicalOverflowReturnsWholeZero(t *testing.T) {
	logical, ok := new(big.Int).SetString("27021598167171471000", 10)
	if !ok || logical.BitLen() <= 64 {
		t.Fatal("independent logical overflow precursor")
	}
	// All other coordinates are finite:2=201326592000,3=3000,
	//4=49152000,6=196608000. None can survive a late logical refusal.
	got, admitted := providerHistoryBinaryCoreUpper(providerHistoryCollectionSemanticLimits{
		Groups: 1, Requests: 1, Blocks: 100, Windows: 100,
	}, 64<<20, 16384, 65536, 64<<20)
	if admitted || got != (providerHistoryBinaryCoreCost{}) {
		t.Fatal("late overflow leaked finite decoded/count fields", got, admitted)
	}
}
