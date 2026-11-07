package registry011

type providerHistoryBinaryCoreCost struct {
	total                                 candidateWorkVector
	passes, windows, blocks, maximumValue uint64
}

// Only successful replayDecodeContent primitives, excluding both trie calls.
// Each canonical raw value of length n pays codec(n), encode(n), equality:
// decoded n and logical2n*n+2n+3. Two root-to-header comparisons add130 per
// block, separately from trie computation. Native decoder/CPU/RAM, outer JSON
// and binary construction, gas/frames, failure/final and factory remain open.
// E/T/L/V are prospective bounds from the SAME original semantic window;
// they are not reported smaller actuals, authority or operating defaults.
func providerHistoryBinaryCoreUpper(l providerHistoryCollectionSemanticLimits, evidence, transactions, logs, valueBytes uint64) (providerHistoryBinaryCoreCost, bool) {
	var empty providerHistoryBinaryCoreCost
	calls, ok := providerHistoryCollectionSemanticCallsBound(l)
	if !ok || evidence == 0 || evidence > 64<<20 || transactions == 0 || transactions > 16384 ||
		logs == 0 || logs > 65536 || valueBytes == 0 || valueBytes > evidence {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryBinaryCoreCost
	// Original acquisition and later semantic decoding use independent
	// canonical values. Neither a cursor copy nor a shared cached decoder
	// replaces either pass. Their successful bytes/counts share the same
	// source bounds even though the acquisition budget is temporary.
	out.passes = a.product(2, a.sum(a.product(14, l.Groups), 1))
	out.windows = a.product(2, calls.windowOpens)
	out.blocks = a.product(2, calls.blockApplies)
	// Headers have their own64KiB cap; a smaller transaction ceiling does
	// not bound them. Every canonical header/value also shares aggregate E.
	out.maximumValue = min(evidence, max(uint64(65536), valueBytes))
	out.total[2] = a.product(out.windows, evidence)
	out.total[3] = out.blocks
	out.total[4] = a.product(out.windows, transactions)
	out.total[6] = a.product(out.windows, logs)
	window := a.sum(a.product(2, a.product(out.maximumValue, evidence)), a.product(2, evidence), a.product(6, transactions))
	// Per block there are three canonical-header event constants plus
	// two32-byte comparisons at65 each. Transaction/receipt constants are
	// already the separate6T window row. Trie observations remain separate.
	out.total[10] = a.sum(a.product(out.windows, window), a.product(133, out.blocks))
	if a.failed {
		return empty, false
	}
	return out, true
}
