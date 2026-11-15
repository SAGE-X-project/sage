package registry011

type providerHistoryContentDecodeLimits struct {
	EvidenceBytes, ValueBytes, Transactions, Logs uint64
}

// Numeric DATA for ONE complete original replayDecodeContent, including both
// indexed tries and their final header comparisons. Q is the original evidence
// cap; V bounds each original raw transaction/receipt BEFORE native decoding.
// The header separately has its original 64KiB guard. Original whole-window
// transaction/log quotas are retained, never renewed by this quote. A caller
// without a tighter authenticated source bound must use V=Q.
//
// All raw evidence is charged before decoding; U=min(Q,max(65536,V)). The raw
// codec/encode/equality prefixes cost at most2UQ+2Q+6T+3. Two canonical indexed
// tries share the disjoint tx/receipt Q pool and add UQ+34Q+2*node*T+8. Their
// two32-byte header equalities add130. Every original incoming primitive is
// already a pending event in those rows, so N==E for this complete scope. This
// does not remove any native preflight or collapse broader role/stage holds.
//
// Native decoding/CPU/backing/peak/lifetime RAM, gas, acquisition, execution,
// enclosing role multiplicities, all-eleven feasibility and the factory are
// separate. Neither this struct nor a positive quote authenticates raw input.
func providerHistoryContentDecodeUpper(p providerHistoryContentDecodeLimits) (providerHistoryControlPair, bool) {
	var empty providerHistoryControlPair
	if p.EvidenceBytes == 0 || p.EvidenceBytes > 64<<20 || p.ValueBytes == 0 || p.ValueBytes > p.EvidenceBytes ||
		p.Transactions == 0 || p.Transactions > 16384 || p.Logs == 0 || p.Logs > 65536 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	u := min(p.EvidenceBytes, max(uint64(65536), p.ValueBytes))
	var out providerHistoryControlPair
	out.normal[2], out.normal[3], out.normal[4], out.normal[6] = p.EvidenceBytes, 1, p.Transactions, p.Logs
	out.normal[10] = a.sum(a.product(3, a.product(u, p.EvidenceBytes)), a.product(36, p.EvidenceBytes),
		a.product(a.sum(a.product(2, providerHistoryTrieNodeLogical), 6), p.Transactions), 141)
	out.envelope = out.normal
	if a.failed {
		return empty, false
	}
	return out, true
}
