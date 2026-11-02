package registry011

import (
	"math/bits"
	"reflect"
)

// One successful lifetime-index root's explicit logical events only. Entries
// is an original prospective count, including zero for an empty index; it is
// not an observed map or a caller's claimed actual. The target's type sizes
// describe the same copy events as the source, not native allocation or RAM.
// Repeated root multiplicities, state serialization/cloning and the complete
// semantic/program/native/failure/final envelope remain separate obligations.
func providerHistorySemanticIndexUpper(entries uint64, claims bool) (candidateWorkVector, bool) {
	return providerHistorySemanticIndexCostUpper(entries, claims, true)
}

// Entered-work DATA only. The original standalone function above still funds
// its unchanged encode8192 preflight. A complete enclosing scope must preserve
// every pointwise primitive/composite headroom before selecting this row.
func providerHistorySemanticIndexActualUpper(entries uint64, claims bool) (candidateWorkVector, bool) {
	return providerHistorySemanticIndexCostUpper(entries, claims, false)
}

func providerHistorySemanticIndexCostUpper(entries uint64, claims, includeHeadroom bool) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	if entries > 1<<20 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	slot := uint64(reflect.TypeFor[providerHistorySemanticIndexEntry]().Size())
	header := uint64(reflect.TypeFor[[]providerHistorySemanticIndexEntry]().Size())
	rounds := uint64(0)
	if entries > 1 {
		rounds = uint64(bits.Len64(entries - 1))
	}
	keyBytes, escapedKey := uint64(64), uint64(6*64)
	population := a.sum(2*(64+1), slot, 1)
	if claims {
		// Two actor text events, one hex64 encoding, a107-byte joined
		// actor/digest key and one entry put. The digest is always32 bytes.
		keyBytes, escapedKey = 107, 6*42+1+64
		population = a.sum(2*(42+1), 64*64+1, 107+1, slot, 1)
	}
	buffer := a.sum(a.product(a.product(2, entries), slot), 1)
	populate := a.product(entries, population)
	compare := a.sum(a.product(2, keyBytes), 1)
	// At most n comparisons and exactly n puts per merge pass. The two
	// slice descriptors are copied once after each complete pass.
	merge := a.product(rounds, a.sum(a.product(entries, a.sum(compare, slot, 1)), a.product(2, header), 1))
	// The closed schemas contain four/eight fields. Count/ordinal need at
	// most seven decimal bytes, block text20, previous digest66 and flag5.
	// ASCII source bytes escape to at most six JSON bytes; the claim key's
	// separator and fixed hex suffix do not escape. These schema uppers are
	// 112 and570/637 at the current domain, not the8192 metadata ceiling.
	metadata := uint64(candidateLedgerMetadataLimit)
	domain := uint64(len(providerHistoryCollectionStateDomain))
	initialBytes := a.sum(domain, uint64(len("semantic-index")), 57)
	entryBytes := a.sum(domain, uint64(len("semantic-index-entry")), 192, escapedKey)
	if a.failed || initialBytes > metadata || entryBytes > metadata {
		return empty, false
	}
	initialRoot := a.sum(a.product(initialBytes, initialBytes), initialBytes, domain, uint64(len("semantic-index")), 4)
	entryRoots := a.product(entries, a.sum(a.product(entryBytes, entryBytes), entryBytes, domain, uint64(len("semantic-index-entry")), 4))
	// Canonicalization still preflights encode(M) on every invocation. Keep
	// its maximum gap over a schema-sized encode once for this whole root.
	// The gap is not charged/refunded, so it remains available before each
	// encode; each invocation's actual upper and all later events stay funded.
	smallest := min(initialBytes, entryBytes)
	padding := a.difference(a.product(metadata, metadata), a.product(smallest, smallest))
	decimal := a.product(entries, 20*20+1)
	adjacent := uint64(0)
	if entries > 0 {
		adjacent = a.product(entries-1, compare)
	}
	units := a.sum(buffer, populate, merge, initialRoot, entryRoots, decimal, adjacent)
	if includeHeadroom {
		units = a.sum(units, padding)
	}
	if a.failed {
		return empty, false
	}
	var work candidateWorkVector
	work[10] = units
	return work, true
}
