package registry011

// The fixed four-record initial index subgraph, independently reconstructed
// in both read passes. Ledger body transitions, semantic object lookup/use,
// retention inputs, native resources and authority are separate components.
type providerHistoryReconstructionIndexCost struct {
	anchors, preparation, ledgerNodes candidateWorkVector
	segments, mutations, validation   candidateWorkVector
	values                            candidateWorkVector
	preparations, generated, checked  uint64
	segmentKeys, valueComparisons     uint64
	normal, incoming, envelope        candidateWorkVector
}

func providerHistoryReconstructionIndexUpper() providerHistoryReconstructionIndexCost {
	// All constants are fixed by this source's enroll/admit/consume/settle
	// grammar, not caller counters or an asserted current object count. Their
	// entire sum is below2^40; no selected input enters this arithmetic.
	const small, node, entry uint64 = 4096, candidateLedgerMetadataLimit, 64 << 10
	q := func(n uint64) uint64 { return n*n + 1 }
	entryDomain := uint64(max(len("sage-provider-enrollment-v2"), len("sage-provider-budget-ticket-v2"),
		len("sage-provider-budget-consumption-v1"), len("sage-provider-budget-settlement-v2")))
	rootDomain := uint64(len("sage-provider-enrollment-v2"))
	mutationDomain := uint64(len("sage-provider-index-mutation-v1"))
	var out providerHistoryReconstructionIndexCost
	// One original prepare and one independent prepare INSIDE apply, for
	// each of four records in each pass. Mutation counts are1/3/2/3.
	out.preparations, out.generated, out.checked = 16, 36, 18
	// Per pass: six ticket-value segment keys, four prepare-ledger segment
	// keys and two checkMutation segment keys. Only the original read owner
	// selects the ticket-value observer; inert DATA callers may use less.
	out.segmentKeys, out.valueComparisons = 24, 90
	// Each constructor builds three independent empty index anchors. Their
	// original4KiB encoder guard survives before the raw decode/hash/string.
	out.anchors[2] = 6 * small
	out.anchors[10] = 6 * (2*q(small) + 2*small + uint64(len("sage-provider-index-empty-v1")) + 3)
	// Entry phase-copy/hash, root phase-copy/decode/hash/retained string.
	// prepare checks entry<=64KiB and root<=8192 BEFORE these observations.
	out.preparation[2] = out.preparations * node
	out.preparation[10] = out.preparations * (q(node) + 2*entry + 3*node + entryDomain + rootDomain + 7)
	// The settle entry is decoded by both preparations. Every non-enroll
	// preparation separately decodes its latest ordinary8192-byte ledger node.
	out.ledgerNodes[2] = 4*entry + 12*node
	out.ledgerNodes[10] = 4*(entry+1+q(entry)) + 12*(node+1+q(node))
	out.segments[10] = out.segmentKeys * (q(small) + small + uint64(len("sage-provider-budget-segment-v1")) + 2)
	// Each fresh mutation is canonically encoded, then decoded/hashed and
	// retained by its object constructor. apply reconstructs it, not a cache.
	out.mutations[2] = out.generated * node
	out.mutations[10] = out.generated * (2*q(node) + 2*node + mutationDomain + 3)
	// Each accepted node is parsed and independently rehashed after two
	// distinct phase copies. Five value comparisons/node independently encode
	// both sides: two expected-delta, two wire-node and one old-map check.
	out.validation[2] = out.checked * node
	out.validation[10] = out.checked * (q(node) + 3*node + mutationDomain + 4)
	out.values[10] = out.valueComparisons * (2*q(node) + 2*node + 1)
	for coordinate := range out.normal {
		out.normal[coordinate] = out.anchors[coordinate] + out.preparation[coordinate] + out.ledgerNodes[coordinate] +
			out.segments[coordinate] + out.mutations[coordinate] + out.validation[coordinate] + out.values[coordinate]
	}
	// One still-outstanding original primitive, not one per node or pass.
	// prepare's entry raw guard is the largest codec here; all unchanged
	// canonical encoders use8192 or4096 and their preflights remain intact.
	out.incoming = candidateWorkVector{2: entry, 10: q(entry)}
	for coordinate := range out.envelope {
		out.envelope[coordinate] = out.normal[coordinate] + out.incoming[coordinate]
	}
	return out
}
