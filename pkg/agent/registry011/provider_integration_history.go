package registry011

// This is an early lower-bound refusal, not a promise that the complete batch
// fits. All subsequent tickets still enforce the complete cumulative vector,
// metadata/capacity and original deadlines; no cap is reset per content batch.
func providerHistoryCapacity(bounds CandidateLedgerBounds, counts [3]uint64, base, selected FinalizedHeader) bool {
	if !candidateLedgerBoundsValid(bounds) || !validHeader(base) || !validHeader(selected) || selected.Number < base.Number {
		return false
	}
	used, ok := providerUpperSum(counts[0], counts[1], counts[2])
	if !ok || used > uint64(bounds.MaximumRecords) {
		return false
	}
	minimum, ok := providerUpperProduct(selected.Number-base.Number, 3)
	if !ok {
		return false
	}
	// One additional local collection needs its own T/C/S even for zero gap.
	minimum, ok = providerUpperSum(minimum, 3)
	return ok && minimum <= uint64(bounds.MaximumRecords)-used
}

func providerCollectionScope(reads *PinnedAgentReads) bool {
	if reads == nil || reads.rpc == nil || reads.rpc.acquisition == nil {
		return false
	}
	c := reads.rpc.acquisition.integration
	if !c.valid() || c.bootstrap == "" || c.pin == "" || c.ledger == nil || reads.header != c.selected {
		return false
	}
	m := c.store.activeMeter.Load()
	if m == nil || m.integration != c || m.authority || m.closed || m.failed || m.parent != nil {
		return false
	}
	entry, exists := c.ledger.tickets[m.ticket]
	return exists && entry.ticket != nil && entry.ticket.kind == "local-work" && entry.consumption != nil && entry.terminal == ""
}
