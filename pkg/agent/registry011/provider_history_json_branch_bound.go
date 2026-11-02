package registry011

import "strconv"

type providerHistoryJSONBranchCost struct {
	actual, preflight, total       candidateWorkVector
	journeys, windows, blocks      uint64
	headerBytes, arraySlots        uint64
	extremalBlock, extremalReceipt uint64
}

// Successful original JSON block/transaction/receipt/log parsing and the
// alternative debug RPC branch. Common binary/retain, core replay, indexed
// tries, gas/effect/native CPU/RAM and failure/final/factory stay separate.
// E/T/L are shared original semantic-window maxima, C selected capture bytes,
// V an independently derived canonical transaction/receipt value ceiling.
// These numerical rows grant no collection or execution authority.
func providerHistoryJSONBranchUpper(l providerHistoryCollectionSemanticLimits, evidence, transactions, logs, bodyBytes, valueBytes uint64, encoding string) (providerHistoryJSONBranchCost, bool) {
	out, ok := providerHistoryJSONBranchActual(l, evidence, transactions, logs, bodyBytes, valueBytes, encoding)
	if !ok {
		return providerHistoryJSONBranchCost{}, false
	}
	var a providerHistoryReadArithmetic
	if encoding == EthereumJSONBlocks {
		out.preflight[10] = a.product(out.windows, 4503633987174401)
	}
	for coordinate := range out.total {
		out.total[coordinate] = a.sum(out.actual[coordinate], out.preflight[coordinate])
	}
	if a.failed {
		return providerHistoryJSONBranchCost{}, false
	}
	return out, true
}

// Successful entered-work DATA only. Preflight and total remain zero so an
// enclosing source derivation must separately fund every original scope.
// In particular this result cannot authorize a JSON decoder or native hash.
func providerHistoryJSONBranchActual(l providerHistoryCollectionSemanticLimits, evidence, transactions, logs, bodyBytes, valueBytes uint64, encoding string) (providerHistoryJSONBranchCost, bool) {
	var empty providerHistoryJSONBranchCost
	calls, ok := providerHistoryCollectionSemanticCallsBound(l)
	if !ok || evidence < 65536 || evidence > 64<<20 || transactions == 0 || transactions > 16384 || logs > 65536 ||
		bodyBytes < 4 || bodyBytes > 8<<20 || valueBytes == 0 || valueBytes > min(evidence, bodyBytes) {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	out := providerHistoryJSONBranchCost{journeys: a.sum(a.product(14, l.Groups), 1), windows: calls.windowOpens, blocks: calls.blockApplies}
	w := uint64(strconv.IntSize / 8)
	c := bodyBytes
	wt, wl := a.product(l.Windows, transactions), a.product(l.Windows, logs)
	switch encoding {
	case EthereumJSONBlocks:
		b := min(evidence, c)
		out.headerBytes = min(b, uint64(65536))
		// Every temporary acquisition charges BOTH original results to E.
		// These are an extremal linear allocation, not separate bounds on
		// measured block/receipt bytes: the larger block coefficient is filled
		// first, then the remaining E (up to C) is assigned to receipts.
		out.extremalBlock = a.product(b, l.Blocks)
		out.extremalReceipt = a.product(min(b, evidence-b), l.Blocks)
		out.actual[2] = a.product(out.journeys, a.sum(a.product(24, out.extremalBlock),
			a.product(12, out.extremalReceipt), a.product(35136, l.Blocks), a.product(2305, wt), a.product(48, wl)))
		out.actual[3], out.actual[4], out.actual[6] = out.blocks, a.product(out.windows, transactions), a.product(out.windows, logs)
		blockCoefficient := a.sum(a.product(25, c), a.product(18, w), 13875)
		receiptCoefficient := a.sum(a.product(12, c), a.product(12, w), 19)
		perBlock := a.sum(53813875, a.product(128, w))
		h := out.headerBytes
		// Canonical tx/receipt/header byte pools share E. Four transaction
		// encoders/hashes plus signing give5V+261; three block header hashes
		// give3H+3. A further header hash occurs for EACH transaction.
		canonicalCoefficient := max(a.sum(a.product(5, valueBytes), 261), a.sum(a.product(3, h), 3))
		perTransaction := a.sum(a.product(h, h), h, a.product(256, valueBytes), a.product(14, w), 3283839)
		perLog := a.sum(14895, a.product(24, w))
		window := a.sum(a.product(canonicalCoefficient, evidence), a.product(perTransaction, transactions), a.product(perLog, logs))
		out.actual[10] = a.product(out.journeys, a.sum(a.product(blockCoefficient, out.extremalBlock),
			a.product(receiptCoefficient, out.extremalReceipt), a.product(perBlock, l.Blocks), a.product(l.Windows, window)))
	case "debug-raw-rlp-v1":
		out.arraySlots = min(transactions, (c-1)/2)
		out.actual[2] = a.product(out.journeys, a.product(a.difference(a.product(5, c), 4), l.Blocks))
		// Block hex, the receipt array, receipt hex codecs and the output
		// slice are included. Common native header/body work is not repeated.
		perBlock := a.sum(a.difference(a.product(5, a.product(c, c)), a.product(6, c)), 20,
			a.product(a.sum(a.product(6, w), 1), out.arraySlots))
		out.actual[10] = a.product(out.journeys, a.sum(a.product(perBlock, l.Blocks), a.product(a.sum(a.product(3, w), 1), wt)))
	default:
		return empty, false
	}
	if a.failed {
		return empty, false
	}
	return out, true
}
