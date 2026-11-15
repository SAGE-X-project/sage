package registry011

import "strconv"

// The closed ASCII spine schema has fifteen literal/value fragments totalling
// at most402 bytes. Append copies cost402+15, three decimal encodes cost
// (20^2+1)+(16^2+1)+(7^2+1), kind-copy costs6 and the original hash costs445.
// This bounds actual logical work, not native CPU/RAM or an entire allowance.
const providerHistoryTypedSpineMaximumWork uint64 = 1576

// Every invocation freshly validates and encodes every original field. Fixed
// keys are already in JCS order; validated hashes and decimal strings contain
// no escaping characters. There is no cached root, tree, authority or result.
// The old512-byte encode preflight remains before the closed allocation.
func providerHistoryCollectionTypedSpineDigestObserved(previous string, ordinal int, header FinalizedHeader, meter *providerWorkMeter) string {
	if meter == nil || ordinal < 0 || ordinal >= 1<<20 || ordinal == 0 && previous != "" ||
		ordinal > 0 && !lowerHash(previous) || !validHeader(header) {
		return ""
	}
	if !meter.event(providerCopyEvent, 5) || !meter.preflight(providerEncodeEvent, providerHistoryCollectionSpineBytes) {
		return ""
	}
	number, timestamp, index := strconv.FormatUint(header.Number, 10), strconv.FormatUint(header.Timestamp, 10), strconv.Itoa(ordinal)
	parts := [...]string{
		`{"format":"`, providerHistoryCollectionStateDomain,
		`","header":{"hash":"`, header.Hash,
		`","number":"`, number,
		`","parent_hash":"`, header.ParentHash,
		`","timestamp":"`, timestamp,
		`"},"kind":"spine","ordinal":"`, index,
		`","previous":"`, previous, `"}`,
	}
	// Batch only this fixed pure primitive's actual local events. Initial
	// copy/encode checks retain recovery permissions; add rechecks the SAME
	// owner, every global/window coordinate and sticky failure. There is no
	// I/O, callback, clock, mutation of input or released role between them.
	var actual candidateWorkVector
	for _, text := range [...]string{number, timestamp, index} {
		n := uint64(len(text)) // Validated uint64/53-bit/20-bit decimal values.
		actual[10] += n*n + 1
	}
	raw := make([]byte, 0, providerHistoryCollectionSpineBytes)
	for _, part := range parts {
		actual[10] += uint64(len(part)) + 1
		raw = append(raw, part...)
	}
	if uint64(len(raw)) > providerHistoryCollectionSpineBytes || !meter.add(actual) {
		return ""
	}
	return providerReplayDigest(providerHistoryCollectionStateDomain, raw, meter)
}
