package registry011

// Compare only the delta from the two previously verified live caches.
// The concrete continuing owner must first prove its private old main/G
// prefix and exclusively apply these typed deltas to those two same maps.
// This DATA predicate cannot establish that prerequisite or grant a permit.
// Full initial/final original scans remain separate and mandatory.
func providerContinuingDeltaEqual(s *providerCustodyStore, proposed, retained *providerLedgerIndexes, delta *providerIndexDelta, ticket *CandidateAdmissionTicket, meter *providerWorkMeter) bool {
	if !s.valid() || s.config.bounds.guardian == nil || s.frontierReplay == nil || s.frontierReplay.indexes != retained || s.frontierReplay.body == nil ||
		proposed == nil || retained == nil || proposed == retained || !proposed.valid() || !retained.valid() ||
		delta == nil || delta.owner != proposed || !delta.used || len(delta.mutations) == 0 || len(delta.mutations) > 3 ||
		ticket == nil || ticket.intent == nil || meter == nil || meter.closed || meter.failed || meter.owner != meter ||
		s.activeMeter.Load() != meter || proposed.meter != meter || retained.meter != meter ||
		proposed.identity != retained.identity || proposed.identity != s.config.identity || proposed.fence != retained.fence ||
		proposed.bounds != retained.bounds || proposed.heads != retained.heads || proposed.counts != retained.counts ||
		proposed.schemaBytes != retained.schemaBytes || proposed.admission != retained.admission ||
		len(proposed.tickets) != len(retained.tickets) || len(proposed.operations) != len(retained.operations) || len(proposed.segments) != len(retained.segments) {
		return false
	}
	a, b := proposed.ledger, retained.ledger
	if a == nil || b == nil || a == b || !a.valid() || !b.valid() || a.meter != meter || b.meter != meter ||
		s.frontierReplay.body.ledger != b || a.intent == nil || b.intent == nil || a.intent == b.intent ||
		a.intent.digest != b.intent.digest || a.intent.digest != ticket.intent.digest || a.bounds != b.bounds || a.limits != b.limits ||
		a.live != b.live || a.retained != b.retained || a.initialHeads != b.initialHeads || a.initialBytes != b.initialBytes ||
		a.initialTotals != b.initialTotals || len(a.initialNodes) != 0 || len(b.initialNodes) != 0 || a.providerStart != b.providerStart ||
		len(a.tickets) != len(b.tickets) || len(a.segments) != len(b.segments) || a.lastTicket == nil || b.lastTicket == nil ||
		a.lastTicket == b.lastTicket || a.lastTicket.digest != ticket.digest || b.lastTicket.digest != ticket.digest ||
		a.current == nil || b.current == nil || a.current == b.current {
		return false
	}
	x, y := a.current, b.current
	if x.digest != y.digest || x.admission != y.admission || x.sequence != y.sequence || x.heads != y.heads || x.counts != y.counts ||
		x.totals != y.totals || x.payload != y.payload || x.metadata != y.metadata || x.privatePayload != y.privatePayload ||
		x.privateMetadata != y.privateMetadata || x.clock != y.clock || len(x.nodes) != 1 || len(y.nodes) != 1 ||
		!providerReplayTextEqual(x.raw, y.raw, meter) || !providerReplayTextEqual(x.nodes[0], y.nodes[0], meter) ||
		!providerReplayTextEqual(a.lastTicket.raw, b.lastTicket.raw, meter) {
		return false
	}
	if (a.lastConsumption == nil) != (b.lastConsumption == nil) {
		return false
	}
	if a.lastConsumption != nil && (a.lastConsumption == b.lastConsumption || a.lastConsumption.digest != b.lastConsumption.digest || !providerReplayTextEqual(a.lastConsumption.raw, b.lastConsumption.raw, meter)) {
		return false
	}
	for _, mutation := range delta.mutations {
		switch mutation.chain {
		case 0:
			left, exists := proposed.tickets[mutation.key]
			right, other := retained.tickets[mutation.key]
			if !exists || !other || !bytesEqualProviderValueObserved(left.object(), right.object(), meter) {
				return false
			}
		case 1:
			left, exists := proposed.operations[mutation.key]
			right, other := retained.operations[mutation.key]
			if !exists || !other || !bytesEqualProviderValueObserved(left.object(), right.object(), meter) {
				return false
			}
		case 2:
			left, exists := proposed.segments[mutation.key]
			right, other := retained.segments[mutation.key]
			if !exists || !other || !bytesEqualProviderValueObserved(left.object(), right.object(), meter) {
				return false
			}
		default:
			return false
		}
	}
	left, exists := a.tickets[ticket.digest]
	right, other := b.tickets[ticket.digest]
	if !exists || !other || left.ticket == nil || right.ticket == nil || left.ticket == right.ticket || left.ticket.digest != right.ticket.digest ||
		left.terminal != right.terminal || (left.consumption == nil) != (right.consumption == nil) ||
		left.consumption != nil && (left.consumption == right.consumption || left.consumption.digest != right.consumption.digest) || a.segments[ticket.segment] != b.segments[ticket.segment] {
		return false
	}
	c, d := a.continuation, b.continuation
	if c == nil || d == nil || c == d || !c.valid(a) || !d.valid(b) || c.bounds != d.bounds || c.page != d.page ||
		c.pending != nil || d.pending != nil || c.final || d.final || c.sealedRecords != d.sealedRecords ||
		c.sealedPayload != d.sealedPayload || c.sealedMetadata != d.sealedMetadata || c.extraMetadata != d.extraMetadata ||
		c.sealedExtra != d.sealedExtra || c.auxiliaryMetadata != d.auxiliaryMetadata || c.sealedAuxiliary != d.sealedAuxiliary ||
		c.auxiliaryPayload != d.auxiliaryPayload || c.sealedExternal != d.sealedExternal || c.lastSeal != d.lastSeal ||
		c.lastPage != d.lastPage || c.lastBudget != d.lastBudget || len(c.records) == 0 || len(c.records) != len(d.records) ||
		c.records[len(c.records)-1] != d.records[len(d.records)-1] {
		return false
	}
	return !meter.failed
}
