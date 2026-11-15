package registry011

type providerHistoryContentDecodePoolLimits struct {
	Windows, Blocks, PendingBlocks uint64
	EvidenceBytes, ValueBytes      uint64
	Transactions, Logs             uint64
}

// Conditional decoder-only work for ONE original cursor journey, including
// ordinary S cuts and the one aborting local transition. Already cleared
// acquisition batches have passed the same cumulative semantic budgets. One
// current pending/local batch can precede that admission and must retain K
// temporary E/T/L allowances. There is no second unconsumed content batch.
// Thus acquisition has (W+K) pools, semantics W pools, and at most2B calls.
//
// Every parameter must be bound to the same original selected range/caps.
// V bounds RAW tx/receipt lengths before decoding; use V=E absent a tighter
// original source proof. This replaces overlapping binary-core/two-pass trie
// rows only. Other JSON/body/withdrawal/gas/execution/copy work, independent
// reader journeys and full role/stage/native/resource/factory costs stay open.
// A finite scalar result grants no source, live budget or operating authority.
func providerHistoryContentDecodePoolUpper(p providerHistoryContentDecodePoolLimits) (providerHistoryControlPair, bool) {
	var empty providerHistoryControlPair
	if p.Windows == 0 || p.Blocks == 0 || p.Blocks > 1<<20 || p.Windows > p.Blocks || p.PendingBlocks == 0 || p.PendingBlocks > 10 ||
		p.EvidenceBytes == 0 || p.EvidenceBytes > 64<<20 || p.ValueBytes == 0 || p.ValueBytes > p.EvidenceBytes ||
		p.Transactions == 0 || p.Transactions > 16384 || p.Logs == 0 || p.Logs > 65536 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	factor := a.sum(a.product(2, p.Windows), p.PendingBlocks)
	pool := a.product(factor, p.EvidenceBytes)
	txs := a.product(factor, p.Transactions)
	u := min(p.EvidenceBytes, max(uint64(65536), p.ValueBytes))
	var out providerHistoryControlPair
	out.normal[2], out.normal[3], out.normal[4], out.normal[6] = pool, a.product(2, p.Blocks), txs, a.product(factor, p.Logs)
	out.normal[10] = a.sum(a.product(3, a.product(u, pool)), a.product(36, pool),
		a.product(a.sum(a.product(2, providerHistoryTrieNodeLogical), 6), txs), a.product(282, p.Blocks))
	out.envelope = out.normal
	if a.failed {
		return empty, false
	}
	return out, true
}
