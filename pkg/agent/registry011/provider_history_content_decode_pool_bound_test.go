package registry011

import (
	"context"
	"math/big"
	"testing"
)

func TestProviderHistoryContentDecodePoolMatchesIndependentRows(t *testing.T) {
	for _, tc := range []struct {
		limits                            providerHistoryContentDecodePoolLimits
		bytes, blocks, txs, logs, logical uint64
	}{
		{providerHistoryContentDecodePoolLimits{1, 1, 1, 1, 1, 1, 1}, 3, 2, 3, 3, 10235793},
		{providerHistoryContentDecodePoolLimits{1, 2, 2, 1024, 128, 2, 2}, 4096, 4, 8, 8, 40025316},
		{providerHistoryContentDecodePoolLimits{2, 5, 2, 4096, 1024, 4, 10}, 24576, 10, 24, 60, 384759186},
		{providerHistoryContentDecodePoolLimits{3, 12, 2, 65536, 1024, 4, 10}, 524288, 24, 32, 80, 103207270392},
		{providerHistoryContentDecodePoolLimits{1, 2, 2, 67108864, 8388608, 16384, 65536}, 268435456, 4, 65536, 262144, 6755632700326452},
		{providerHistoryContentDecodePoolLimits{100, 100, 10, 67108864, 67108864, 16384, 65536}, 14092861440, 200, 3440640, 13762560, 2837280011355123240},
	} {
		want := candidateWorkVector{2: tc.bytes, 3: tc.blocks, 4: tc.txs, 6: tc.logs, 10: tc.logical}
		got, ok := providerHistoryContentDecodePoolUpper(tc.limits)
		if !ok || got.normal != want || got.envelope != want {
			t.Fatal("independent decoder pending-frontier pool", tc, got, want, ok)
		}
	}
}

func TestProviderHistoryContentDecodePoolRejectsInvalidData(t *testing.T) {
	for _, p := range []providerHistoryContentDecodePoolLimits{
		{0, 1, 1, 1024, 128, 1, 1}, {1, 0, 1, 1024, 128, 1, 1}, {2, 1, 1, 1024, 128, 1, 1},
		{1, (1 << 20) + 1, 1, 1024, 128, 1, 1}, {1, 1, 0, 1024, 128, 1, 1}, {1, 1, 11, 1024, 128, 1, 1},
		{1, 1, 1, 0, 128, 1, 1}, {1, 1, 1, (64 << 20) + 1, 128, 1, 1},
		{1, 1, 1, 1024, 0, 1, 1}, {1, 1, 1, 1024, 1025, 1, 1},
		{1, 1, 1, 1024, 128, 0, 1}, {1, 1, 1, 1024, 128, 16385, 1},
		{1, 1, 1, 1024, 128, 1, 0}, {1, 1, 1, 1024, 128, 1, 65537},
		{^uint64(0), 1, 1, 1024, 128, 1, 1}, {1, ^uint64(0), 1, 1024, 128, 1, 1},
		{1, 1, ^uint64(0), 1024, 128, 1, 1}, {1, 1, 1, ^uint64(0), 128, 1, 1},
		{1, 1, 1, 1024, ^uint64(0), 1, 1}, {1, 1, 1, 1024, 128, ^uint64(0), 1},
		{1, 1, 1, 1024, 128, 1, ^uint64(0)},
	} {
		if got, ok := providerHistoryContentDecodePoolUpper(p); ok || got != (providerHistoryControlPair{}) {
			t.Fatal("invalid pooled DATA leaked a partial pair", p, got, ok)
		}
	}
}

func TestProviderHistoryContentDecodePoolRefusesLateLogicalOverflow(t *testing.T) {
	large, ok := new(big.Int).SetString("27156822965827620240", 10)
	if !ok || large.BitLen() <= 64 {
		t.Fatal("independent late-overflow literal")
	}
	// Decoded134888816640/block2000/tx32931840/log131727360 fit;
	// the complete logical cost alone exceeds uint64 and must erase all rows.
	p := providerHistoryContentDecodePoolLimits{1000, 1000, 10, 67108864, 67108864, 16384, 65536}
	if got, ok := providerHistoryContentDecodePoolUpper(p); ok || got != (providerHistoryControlPair{}) {
		t.Fatal("late pooled overflow published earlier finite coordinates", got, ok)
	}
}

func TestProviderHistoryContentDecodePoolFundsIndependentAcquisitionAndSemanticPasses(t *testing.T) {
	blocks := replayTestInput(t).Blocks[:2]
	var bytes, value, txs, logs uint64
	for _, raw := range blocks {
		s := historyBinaryCoreOriginalShape(t, raw)
		bytes += s.bytes
		value, txs, logs = max(value, s.maximumValue), txs+s.transactions, logs+s.logs
	}
	p := providerHistoryContentDecodePoolLimits{1, 2, 2, bytes, value, txs, logs}
	cost, ok := providerHistoryContentDecodePoolUpper(p)
	if !ok {
		t.Fatal("finite two-pass pool precursor")
	}
	m := providerHistoryContentUnitMeter(cost.normal)
	limits := replayTestLimits()
	limits.MaxEvidenceBytes, limits.MaxTransactions, limits.MaxLogs = int(bytes), int(txs), int(logs)
	for _, raw := range blocks {
		b := replayBudget{limits: limits, observer: m}
		if _, err := replayDecodeContent(context.Background(), raw, &b); err != nil {
			t.Fatal("original independent acquisition budget", err)
		}
	}
	semantic := replayBudget{limits: limits, observer: m}
	for _, raw := range blocks {
		if _, err := replayDecodeContent(context.Background(), raw, &semantic); err != nil {
			t.Fatal("original shared semantic budget", err)
		}
	}
	if m.failed || m.clocks != 0 || m.actual[2] != 2*bytes || m.actual[3] != 4 || m.actual[4] != 2*txs || m.actual[6] != 2*logs || semantic.bytes != int(bytes) {
		t.Fatal("independent original passes lost work or renewed bytes", m.actual, semantic.bytes)
	}
	historyEventQueueWithin(t, m.actual, cost.normal)
}

func TestProviderHistoryContentDecodePoolRetainsPendingWorkBeforeFailedSemanticCharge(t *testing.T) {
	blocks := replayTestInput(t).Blocks[:2]
	shapes := []historyBinaryCoreShape{historyBinaryCoreOriginalShape(t, blocks[0]), historyBinaryCoreOriginalShape(t, blocks[1])}
	// Decoder DATA only: choose the larger canonical block first so the
	// same semantic byte budget exhausts exactly there. This is not a claim
	// that this reordered pair passes the separate chain/semantic caller.
	if shapes[0].bytes < shapes[1].bytes {
		blocks[0], blocks[1] = blocks[1], blocks[0]
		shapes[0], shapes[1] = shapes[1], shapes[0]
	}
	evidence := shapes[0].bytes
	p := providerHistoryContentDecodePoolLimits{1, 2, 2, evidence, max(shapes[0].maximumValue, shapes[1].maximumValue),
		max(shapes[0].transactions, shapes[1].transactions), max(shapes[0].logs, shapes[1].logs)}
	cost, ok := providerHistoryContentDecodePoolUpper(p)
	if !ok || shapes[1].bytes == 0 {
		t.Fatal("finite pending canonical-block precursor")
	}
	m := providerHistoryContentUnitMeter(cost.normal)
	limits := replayTestLimits()
	limits.MaxEvidenceBytes, limits.MaxTransactions, limits.MaxLogs = int(p.EvidenceBytes), int(p.Transactions), int(p.Logs)
	for _, raw := range blocks {
		b := replayBudget{limits: limits, observer: m}
		if _, err := replayDecodeContent(context.Background(), raw, &b); err != nil {
			t.Fatal("pending acquisition must precede semantic admission", err)
		}
	}
	semantic := replayBudget{limits: limits, observer: m}
	if _, err := replayDecodeContent(context.Background(), blocks[0], &semantic); err != nil {
		t.Fatal("first shared semantic block", err)
	}
	before := m.actual
	if _, err := replayDecodeContent(context.Background(), blocks[1], &semantic); err == nil || m.failed || m.actual != before || semantic.bytes != int(evidence) {
		t.Fatal("late semantic byte refusal renewed or erased pending work", err, m.actual, before)
	}
	if m.clocks != 0 || m.actual[2] != 2*evidence+shapes[1].bytes || m.actual[2] <= 2*evidence || m.actual[3] != 3 {
		t.Fatal("pending original decoder work was hidden in the shared semantic pool", m.actual)
	}
	historyEventQueueWithin(t, m.actual, cost.normal)
}

func TestProviderHistoryContentDecodePoolKeepsOriginalPendingBatchOutsideSemanticBytes(t *testing.T) {
	c, in, methods := historyCollectionContentInputFixture(t, 2, historyCollectionStreamingInput(t), SepoliaAmsterdamRootExecutionGasProfile)
	c = historyCollectionContentStep(t, c, in, methods) // Original open.
	c = historyCollectionContentStep(t, c, in, methods) // Two acquired blocks, forty pending effects.
	if c.phase != "effects" || len(c.pending) != 2 || c.pendingApplied != 0 || c.semantic.blocks != 0 || c.semantic.window.budget.bytes != 0 {
		t.Fatal("original pending bytes were prematurely charged to semantic decoding")
	}
	for _, block := range c.pending {
		if block.headerRLP == "" || len(block.transactions) == 0 || len(block.receipts) == 0 {
			t.Fatal("pending batch lost retained original bytes")
		}
	}
	m := c.semantic.observer
	before := m.actual
	if c.readContentObserved(context.Background(), nil, m) || m.actual != before {
		t.Fatal("second content batch entered before original pending batch cleared")
	}
	c = historyCollectionContentStep(t, c, in, methods) // First thirty-two traces apply only block zero.
	if c.phase != "effects" || c.pendingApplied != 1 || c.semantic.blocks != 1 || c.semantic.window.budget.bytes == 0 ||
		c.pending[0].headerRLP != "" || c.pending[1].headerRLP == "" {
		t.Fatal("partial original application lost remaining pending bytes")
	}
	m = c.semantic.observer
	before = m.actual
	if c.readContentObserved(context.Background(), nil, m) || m.actual != before {
		t.Fatal("another content batch bypassed the partially applied original batch")
	}
}

func TestProviderHistoryContentDecodePoolKeepsIndependentTransactionAndLogCaps(t *testing.T) {
	raw := replayTestInput(t).Blocks[0]
	shape := historyBinaryCoreOriginalShape(t, raw)
	if shape.transactions != 1 || shape.logs != 1 {
		t.Fatal("one-transaction/one-log precursor")
	}
	for _, mode := range []string{"transactions", "logs"} {
		limits := replayTestLimits()
		limits.MaxEvidenceBytes, limits.MaxTransactions, limits.MaxLogs = int(2*shape.bytes), 1, 1
		if mode == "logs" {
			limits.MaxTransactions = 2
		}
		p := providerHistoryContentDecodePoolLimits{1, 2, 2, uint64(limits.MaxEvidenceBytes), shape.maximumValue, uint64(limits.MaxTransactions), 1}
		cost, ok := providerHistoryContentDecodePoolUpper(p)
		if !ok {
			t.Fatal("finite count-isolated original budget")
		}
		m := providerHistoryContentUnitMeter(cost.normal)
		b := replayBudget{limits: limits, observer: m}
		if _, err := replayDecodeContent(context.Background(), raw, &b); err != nil || m.failed {
			t.Fatal("first original count precursor", mode, err)
		}
		before := m.actual
		out, err := replayDecodeContent(context.Background(), raw, &b)
		if err == nil || m.failed || m.clocks != 0 || b.logs != 1 || b.limits != limits {
			t.Fatal("original count cap bypassed", mode, err, m.actual)
		}
		switch mode {
		case "transactions":
			// Header scalar charge happens BEFORE the remaining-transaction guard;
			// byte room remains, but no second header codec/block event is entered.
			if out.header != nil || m.actual != before || b.bytes != int(shape.bytes)+len(raw.HeaderRLP) || b.txs != 1 {
				t.Fatal("transaction cap lost its exact scalar prefix", m.actual, b.bytes, b.txs)
			}
		case "logs":
			// The second header/tx/receipt rows complete, then the remaining-log
			// guard refuses BEFORE any second log observation or trie root.
			prefix := shape.work
			prefix[6], prefix[10] = 0, prefix[10]-130
			want := before
			for i, n := range prefix {
				want[i] += n
			}
			if out.header == nil || out.txs[0] != nil || out.receipts[0] != nil || m.actual != want || b.bytes != int(2*shape.bytes) || b.txs != 2 {
				t.Fatal("log cap lost entered canonical rows", m.actual, want, b.bytes, b.txs)
			}
		}
		historyEventQueueWithin(t, m.actual, cost.normal)
	}
}
