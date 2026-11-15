package registry011

// The original scan-control and receipt subgraph, interpreted afresh in both
// reconstruction passes. Files, names, anchored inventory loads, semantic root
// checks and native resources belong to other rows. This numeric DATA cannot
// install a recovery observer, renew its allowance or grant execution.
type providerHistoryReconstructionScanCost struct {
	encodes, decodes, hashes, equalities, textEqualities uint64
	copies                                               candidateWorkVector
	normal, incoming, envelope                           candidateWorkVector
}

func providerHistoryReconstructionScanUpper(l providerHistoryLegacyControlLimits, policyBytes uint64) (providerHistoryReconstructionScanCost, bool) {
	var empty providerHistoryReconstructionScanCost
	if policyBytes == 0 || policyBytes > 64<<10 {
		return empty, false
	}
	if _, ok := providerHistoryLegacyControlUpper(l); !ok {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryReconstructionScanCost
	// Per fresh pass, constructor+binding use two encodes and two decodes.
	// Initial/final controls add eight encodes and five hashes. Each prior
	// attempt adds six encodes, one decode, four hashes and five equalities.
	// A receipt pair for BOTH kinds decodes/encodes four originals; expected
	// command construction adds two encodes and its own fifth decode.
	out.encodes = a.product(2, a.sum(10, a.product(6, l.attempts), a.product(6, l.chunks)))
	out.decodes = a.product(2, a.sum(2, l.attempts, a.product(5, l.chunks)))
	out.hashes = a.product(2, a.sum(6, a.product(4, l.attempts), a.product(4, l.chunks)))
	out.equalities = a.product(2, a.sum(6, a.product(5, l.attempts), a.product(7, l.chunks)))
	out.textEqualities = a.product(2, a.sum(10, a.product(20, l.chunks)))
	// Binding and policy bytes are copied before their codec length checks.
	// Include the selected full object allowance, even on early rejection.
	// Each receipt retains two strings and one record, separately from the
	// initial records-capacity copy. The input/file row owns filename copies.
	out.copies[10] = a.product(2, a.sum(l.objectBytes, 1, policyBytes, 1, l.chunks, 1,
		a.product(l.chunks, a.sum(a.product(2, l.metadata), 4))))
	codec := a.sum(a.product(l.metadata, l.metadata), 1)
	domain := uint64(max(len("sage-provider-store-profile-v6"), len(providerScanIntentDomain), len(providerScanCommitDomain),
		len(providerScanHighwaterDomain), len(providerScanHeadDomain), len(providerScanChunkV2), len(providerScanCommandV2)))
	hash := a.sum(l.metadata, domain, 2)
	equal := a.sum(a.product(2, l.metadata), 1)
	out.normal[2] = a.product(out.decodes, l.metadata)
	out.normal[10] = a.sum(a.product(a.sum(out.encodes, out.decodes), codec), a.product(out.hashes, hash),
		a.product(out.equalities, equal), a.product(out.textEqualities, 133), out.copies[10])
	// An original encode preflight retains the full selected metadata square,
	// regardless of a smaller successful result. A rejected field may be D
	// bytes rather than a valid66-byte digest. Keep ONE outstanding maximum,
	// including the pre-decode binding/policy copies, across both passes.
	out.incoming[2] = l.metadata
	out.incoming[10] = max(codec, hash, equal, a.sum(l.metadata, 67), a.sum(l.objectBytes, 1), a.sum(policyBytes, 1), a.sum(l.chunks, 1))
	for coordinate := range out.normal {
		out.envelope[coordinate] = a.sum(out.normal[coordinate], out.incoming[coordinate])
	}
	if a.failed || !providerHistoryReadWorkDomain(out.envelope) {
		return empty, false
	}
	return out, true
}
