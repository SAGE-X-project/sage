package registry011

import "strconv"

type providerHistoryJSONShapeCost struct {
	fields                                                        [7]candidateWorkVector
	jsonPreflight, stringValue, hexData, quantity, value, literal candidateWorkVector
	array, quantityRoom, valueRoom                                candidateWorkVector
	maximum, slots                                                uint64
}

// Successful observations of the original nonnil JSON primitives only.
// Fields are ordered block/transaction/access/authorization/withdrawal/receipt/log.
// These source-derived rows are alternatives, not one complete parser cost.
// Typed canonical JSON, hashes/signatures, caller multiplicities, native CPU
// and physical RAM remain separate. Numerical DATA supplies no authority.
func providerHistoryJSONShapeUpper(maximum, elements uint64) (providerHistoryJSONShapeCost, bool) {
	if maximum < 4 || maximum > 64<<20 || elements > 65536 {
		return providerHistoryJSONShapeCost{}, false
	}
	// The closed domain makes even the largest row less than2^54. No
	// unchecked caller-sized product, reported output length or allocation
	// is accepted. Go's supported word sizes are four or eight bytes.
	w := uint64(strconv.IntSize / 8)
	n, square := maximum, maximum*maximum
	out := providerHistoryJSONShapeCost{maximum: maximum, slots: min(elements, (maximum-1)/2)}
	out.jsonPreflight = candidateWorkVector{2: n, 10: square + 2*n + 3}
	out.stringValue = candidateWorkVector{2: n, 10: square + n + 2}
	out.hexData = candidateWorkVector{2: 2*n - 4, 10: square + (n-4)*(n-4) + n + 1}
	out.quantity = candidateWorkVector{2: 16, 10: 566}
	out.value = candidateWorkVector{2: 64, 10: 4164}
	// Only the original null/false literals (length at most five) are covered.
	out.literal = candidateWorkVector{10: 2*n + 7}
	out.array = candidateWorkVector{2: n, 10: square + (6*w+1)*out.slots + 2}
	// For the original distinct allowed/required names: A fields, S total
	// name bytes, R mandatory fields. Keys are a subset of allowed names;
	// mandatory RawMessage values are disjoint and sum to at most n.
	// Constants below are2*A*S+A*A+6*R, not producer-supplied schema sizes.
	for i, constant := range [7]uint64{18287, 10296, 88, 408, 296, 5891, 2094} {
		out.fields[i] = candidateWorkVector{2: n, 10: square + 3*n + 2 + constant}
	}
	// Additional unconsumed room for each selected primitive's original
	// preflight. It is neither recorded actual nor a renewed allowance.
	out.quantityRoom = candidateWorkVector{10: 257}
	out.valueRoom = candidateWorkVector{10: 4097}
	return out, true
}

// Original JSON atoms and the root-context seven-call prefix only. The first
// seven schemas retain the existing successful rows; the rest are gas row,
// execution/regular dimensions, classification, root context, frame and log.
// Keys/strings must come from the original UTF8+surrogate-checked response.
// These numeric parts grant no source/observer/native/RAM/factory authority.
type providerHistoryJSONControlCost struct {
	fields                                                      [14]providerHistoryControlPair
	jsonPreflight, stringValue, hexData, quantity, index, value providerHistoryControlPair
	literal, array, signedQuantity, rootContextPrefix           providerHistoryControlPair
}

func providerHistoryJSONControlUpper(maximum, elements uint64) (providerHistoryJSONControlCost, bool) {
	var empty providerHistoryJSONControlCost
	normal, ok := providerHistoryJSONShapeUpper(maximum, elements)
	if !ok {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryJSONControlCost
	n, square := maximum, a.product(maximum, maximum)
	pair := func(v candidateWorkVector) providerHistoryControlPair { return providerHistoryControlPair{v, v} }
	out.jsonPreflight, out.stringValue, out.hexData = pair(normal.jsonPreflight), pair(normal.stringValue), pair(normal.hexData)
	out.value, out.literal, out.array = pair(normal.value), pair(normal.literal), pair(normal.array)
	// A/S/R: distinct allowed names, total name bytes, required names. These
	// are inspected fixed source schemas, never producer supplied counts.
	for i, schema := range [14][3]uint64{
		{29, 298, 27}, {24, 201, 12}, {2, 18, 2}, {6, 28, 6},
		{4, 32, 4}, {17, 163, 10}, {10, 97, 9}, {2, 12, 2},
		{4, 44, 4}, {4, 42, 4}, {5, 58, 0}, {15, 121, 0},
		{17, 130, 6}, {5, 30, 0},
	} {
		allowed, names, required := schema[0], schema[1], schema[2]
		linear := n
		if required != 0 {
			linear = a.product(3, n)
		}
		constant := a.sum(a.product(a.product(2, allowed), names), a.product(allowed, allowed), a.product(6, required))
		v := candidateWorkVector{2: n, 10: a.sum(square, linear, 2, constant)}
		out.fields[i] = pair(v)
		// Map iteration may visit valid names before its FIRST unknown name.
		// That decoded key can be n bytes and is compared to every allowed
		// name before refusal. Mandatory-value checks are then skipped, but
		// retaining their old normal room is a conservative complete envelope.
		out.fields[i].envelope[10] = a.sum(v[10], a.product(allowed, n), names, allowed)
	}
	out.quantity = pair(normal.quantity)
	// ParseUint's uint64 refusal is AFTER the lowercase scan and input-sized
	// codec. Its successful16-digit bound cannot erase a long failed token.
	out.quantity.envelope[2] = max(normal.quantity[2], n-2)
	out.quantity.envelope[10] = max(normal.quantity[10], a.sum(a.product(n-2, n-2), n, 2))
	// Decimal index20: initial string scan21, copy21, codec401, encode401,
	// equality41. A failed string JSON parse may continue to decimal fallback
	// on the same observer; its two codec inputs can each contain20 bytes.
	out.index.normal[2] = a.sum(n, 16)
	out.index.normal[10] = max(a.sum(normal.stringValue[10], normal.quantity[10]), uint64(885))
	out.index.envelope[2] = max(out.index.normal[2], a.product(2, min(n, uint64(20))), a.sum(n, out.quantity.envelope[2]))
	out.index.envelope[10] = max(out.index.normal[10], a.sum(normal.stringValue[10], out.quantity.envelope[10]))
	// Signed diagnostic text is at most19 bytes on success. Its raw string
	// scan and quantity codec still precede the late int64 range decision.
	out.signedQuantity.normal = candidateWorkVector{2: a.sum(n, 16), 10: a.sum(normal.stringValue[10], 586)}
	out.signedQuantity.envelope[2] = max(out.signedQuantity.normal[2], a.sum(n, out.quantity.envelope[2]))
	out.signedQuantity.envelope[10] = max(out.signedQuantity.normal[10], a.sum(normal.stringValue[10], n, 1, out.quantity.envelope[10]))
	// Four string values and input hex come from FIVE DISTINCT fields of
	// one source object: from/type/gas/input/value. Their raw JSON lengths
	// sum<=n. Input and gas inner-codec lengths also share that same pool.
	// ALL seven original calls precede the combined bool check, including
	// input/value after a quantity refusal. Never apply first-false per atom.
	out.rootContextPrefix.normal = candidateWorkVector{2: a.sum(a.product(2, n), 80),
		10: a.sum(a.product(2, square), n, 4739)}
	out.rootContextPrefix.envelope = candidateWorkVector{2: out.rootContextPrefix.normal[2],
		10: a.sum(a.product(2, square), a.product(2, n), 4739)}
	if a.failed {
		return empty, false
	}
	return out, true
}
