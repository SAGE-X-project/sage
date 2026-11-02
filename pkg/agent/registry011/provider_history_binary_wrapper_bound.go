package registry011

import (
	"reflect"
	"strconv"
)

type providerHistoryBinaryWrapperCost struct {
	common, transaction, retention, builder candidateWorkVector
	preflight, actual, total                candidateWorkVector
	journeys, windows, blocks, native       uint64
	binaryBytes, headerBytes                uint64
	bodyPool, headerPool                    uint64
}

// Successful common binary acquisition and immutable retain/replay copies.
// JSON/RPC parsing, trie computation, replayDecodeContent, gas/frame/effect
// callers, native CPU/allocator/RAM, failure/final and factory stay separate.
// E/T/V come from the original semantic window, C from its selected capture,
// native from the original whole manifest. Numeric DATA grants no authority.
func providerHistoryBinaryWrapperUpper(l providerHistoryCollectionSemanticLimits, evidence, transactions, bodyBytes, valueBytes, native uint64, encoding string) (providerHistoryBinaryWrapperCost, bool) {
	out, ok := providerHistoryBinaryWrapperActual(l, evidence, transactions, bodyBytes, valueBytes, native, encoding)
	if !ok {
		return providerHistoryBinaryWrapperCost{}, false
	}
	var a providerHistoryReadArithmetic
	maximum := max(bodyBytes, uint64(65536))
	out.preflight[10] = a.product(out.windows, a.sum(a.product(maximum, maximum), 1))
	for coordinate := range out.total {
		out.total[coordinate] = a.sum(out.actual[coordinate], out.preflight[coordinate])
	}
	if a.failed {
		return providerHistoryBinaryWrapperCost{}, false
	}
	return out, true
}

// Actual-only numerical DATA for the enclosing original-scope compiler.
// Preflight and total stay zero: this is not a standalone admission bound,
// and it cannot replace any original scope's primitive/composite checks.
func providerHistoryBinaryWrapperActual(l providerHistoryCollectionSemanticLimits, evidence, transactions, bodyBytes, valueBytes, native uint64, encoding string) (providerHistoryBinaryWrapperCost, bool) {
	var empty providerHistoryBinaryWrapperCost
	calls, ok := providerHistoryCollectionSemanticCallsBound(l)
	// The original final header encoder preflights65536 against E, even
	// when its actual header is smaller. Impossible smaller E is refused.
	if !ok || evidence < 65536 || evidence > 64<<20 || transactions == 0 || transactions > 16384 ||
		bodyBytes == 0 || bodyBytes > 8<<20 || valueBytes == 0 || valueBytes > min(evidence, bodyBytes) || native > l.Blocks {
		return empty, false
	}
	withdrawals := uint64(16)
	json := false
	switch encoding {
	case EthereumJSONBlocks:
		json = true
	case "debug-raw-rlp-v1":
		withdrawals = 16384 // Original successful trie guard, not JSON's16.
	default:
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryBinaryWrapperCost
	out.journeys = a.sum(a.product(14, l.Groups), 1)
	out.windows, out.blocks, out.native = calls.windowOpens, calls.blockApplies, a.product(out.journeys, native)
	out.binaryBytes = min(evidence, bodyBytes)
	out.headerBytes = min(out.binaryBytes, uint64(65536))
	out.bodyPool = a.product(out.binaryBytes, l.Blocks)
	if json {
		// Header+canonical tx bytes share E/T. A typed transaction adds at
		// most five RLP-prefix bytes in a block. The three enclosing prefixes,
		// empty uncles and at most16 canonical49-byte withdrawals add<=800.
		pool := a.sum(a.product(l.Windows, a.sum(evidence, a.product(5, transactions))), a.product(800, l.Blocks))
		out.bodyPool = min(out.bodyPool, pool)
	}
	out.headerPool = min(a.product(out.headerBytes, l.Blocks), a.product(evidence, l.Windows))
	word := uint64(strconv.IntSize / 8)
	gas := uint64(reflect.TypeFor[ReplayGasEvidence]().Size())
	body := uint64(reflect.TypeFor[providerHistoryNativeBlockBody]().Size())
	// Three whole-body codecs, one body-payload codec and one header codec;
	// canonical body/uncle/header encodes and the selected-header hash/text
	// comparisons are distinct. Withdrawal encodes are<=49^2+1 each.
	out.common[2] = a.product(out.journeys, a.sum(a.product(4, out.bodyPool), out.headerPool))
	perBlock := a.sum(a.product(2, bodyBytes), 9128, a.product(withdrawals, a.sum(2402, a.product(3, word))))
	out.common[10] = a.product(out.journeys, a.sum(
		a.product(6, a.product(out.binaryBytes, out.bodyPool)), a.product(4, a.product(out.headerBytes, out.headerPool)),
		a.product(3, out.bodyPool), a.product(3, out.headerPool), a.product(l.Blocks, perBlock)))
	// Actual canonical transaction materialization, with one output slice.
	// The slice event's per-block constant is already in common's9128.
	out.transaction[10] = a.product(out.windows, a.sum(a.product(valueBytes, evidence), a.product(a.sum(a.product(3, word), 1), transactions)))
	// Retain and replay each own their strings/bytes and array headers.
	// Original gas checking charges trace bytes to the SAME semantic E.
	out.retention[10] = a.sum(a.product(2, a.product(out.windows, evidence)),
		a.product(a.sum(a.product(10, word), 4), a.product(out.windows, transactions)),
		a.product(6, out.blocks), a.product(a.sum(gas, 3), out.native))
	if json {
		out.builder[10] = a.product(out.journeys, a.sum(a.product(out.binaryBytes, out.bodyPool), a.product(l.Blocks, a.sum(body, 2))))
	}
	for c := range out.actual {
		out.actual[c] = a.sum(out.common[c], out.transaction[c], out.retention[c], out.builder[c])
	}
	if a.failed {
		return empty, false
	}
	return out, true
}
