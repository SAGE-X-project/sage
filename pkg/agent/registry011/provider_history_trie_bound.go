package registry011

// The closed indexed-key trie has at most seven nodes per value: recursive
// depth strictly increases and an RLP(index) key has at most six nibbles.
// Each internal encoding is at most532 bytes. This source-derived coefficient
// includes node/key/path encodes, comparisons, field copies and hashes; it is
// logical observation work, not native Keccak CPU or backing/peak/lifetime RAM.
const providerHistoryTrieNodeLogical = 1705896

// Prospective bounds for successful canonical values, never reported actuals
// or an admission capability. Each nonempty value has at most valueBytes;
// all values together have at most bytes. Empty actual roots still cost four.
// No caller receives an observer, reservation, operating policy or factory.
func providerHistoryContentTrieUpper(items, bytes, valueBytes uint64) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	if items > 16384 || bytes > 64<<20 || valueBytes > 64<<20 ||
		items == 0 && (bytes != 0 || valueBytes != 0) || items != 0 && (bytes == 0 || valueBytes == 0) {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	value := min(valueBytes, bytes)
	pool := min(bytes, a.product(items, value))
	work := candidateWorkVector{10: a.sum(4, a.product(value, pool), a.product(34, pool), a.product(providerHistoryTrieNodeLogical, items))}
	if a.failed {
		return empty, false
	}
	return work, true
}

type providerHistoryCollectionTrieCost struct {
	content, withdrawals, total             candidateWorkVector
	journeys, contentRoots, withdrawalRoots uint64
	windowBytes                             uint64
}

// Both acquisition and semantic replay remain independent. Per original
// block, JSON acquisition can derive transaction/withdrawal roots, complete
// binary acquisition derives withdrawal/transaction/receipt roots, and the
// semantic window derives transaction/receipt roots again: five content roots
// and two withdrawal roots. Debug-RLP omits the two JSON roots, but its native
// withdrawal reader has the trie guard16384, not JSON's separate16-item guard.
// Original same-window transaction+receipt bytes share E; their cardinalities
// each share T. Their weights are3 and2, so use3*V*E,102*E,5*C*T, not five
// independent maximum-sized arrays. Every root also reserves its empty four.
// Initial/cursor/model work, JSON/RLP codecs, gas/frames, native CPU/RAM,
// rejected prefixes/final/factory and all other work coordinates are separate.
func providerHistoryCollectionTrieUpper(l providerHistoryCollectionSemanticLimits, evidence, transactions, valueBytes uint64, encoding string) (providerHistoryCollectionTrieCost, bool) {
	var empty providerHistoryCollectionTrieCost
	calls, ok := providerHistoryCollectionSemanticCallsBound(l)
	// A pinned Withdrawal has three uint64 fields and a20-byte address. Its
	// canonical RLP is at most3*9+21+1=49 bytes in either original reader.
	weight, roots, withdrawalRoots, withdrawalItems := uint64(3), uint64(5), uint64(2), uint64(16)
	switch encoding {
	case EthereumJSONBlocks:
	case "debug-raw-rlp-v1":
		weight, roots, withdrawalRoots, withdrawalItems = 2, 4, 1, 16384
	default:
		return empty, false
	}
	withdrawal, withdrawalOK := providerHistoryContentTrieUpper(withdrawalItems, withdrawalItems*49, 49)
	if !ok || !withdrawalOK || evidence == 0 || evidence > 64<<20 || transactions == 0 || transactions > 16384 ||
		valueBytes == 0 || valueBytes > evidence {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryCollectionTrieCost
	out.journeys = a.sum(a.product(14, l.Groups), 1)
	blocks := a.product(out.journeys, l.Blocks)
	out.contentRoots, out.withdrawalRoots = a.product(roots, blocks), a.product(withdrawalRoots, blocks)
	out.windowBytes = min(evidence, a.product(2, a.product(transactions, valueBytes)))
	pool := a.sum(a.product(weight, a.product(valueBytes, out.windowBytes)), a.product(a.product(34, weight), out.windowBytes),
		a.product(roots, a.product(providerHistoryTrieNodeLogical, transactions)))
	out.content[10] = a.sum(a.product(calls.windowOpens, pool), a.product(4, out.contentRoots))
	out.withdrawals[10] = a.product(out.withdrawalRoots, withdrawal[10])
	out.total[10] = a.sum(out.content[10], out.withdrawals[10])
	if a.failed {
		return empty, false
	}
	return out, true
}
