package registry011

// Compare two separately owned first-operation models after the independently
// retained delta has also been charged to the provisional coordinator. This
// is a closed equality check, not permission to import a provisional map or
// reconstruct an owner. The caller holds store.mu and the current observer.
func providerFirstOperationModelsEqual(s *providerCustodyStore, proposed, retained *providerLedgerIndexes, meter *providerWorkMeter) bool {
	if s == nil || !s.valid() || s.config.bounds.firstAdmission == nil || s.frontierReplay == nil || s.frontierReplay.body == nil || s.frontierReplay.indexes != retained || proposed == nil || retained == nil || proposed == retained || !proposed.valid() || !retained.valid() || meter == nil || meter.owner != meter || meter.closed || meter.failed || s.activeMeter.Load() != meter || proposed.meter != meter || retained.meter != meter || proposed.ledger == nil || retained.ledger == nil || s.frontierReplay.body.ledger != retained.ledger || proposed.ledger == retained.ledger || proposed.identity != s.config.identity || proposed.identity != retained.identity || proposed.fence != retained.fence || proposed.bounds != retained.bounds || proposed.heads != retained.heads || proposed.counts != retained.counts || proposed.schemaBytes != retained.schemaBytes || proposed.admission != retained.admission || len(proposed.tickets) != 1 || len(retained.tickets) != 1 || len(proposed.operations) < 2 || len(proposed.operations) > 4 || len(proposed.operations) != len(retained.operations) || len(proposed.segments) != 1 || len(retained.segments) != 1 {
		return false
	}
	a, b := proposed.ledger, retained.ledger
	if !a.valid() || !b.valid() || a.meter != meter || b.meter != meter || a.intent == nil || b.intent == nil || a.intent == b.intent || a.intent.digest != b.intent.digest || a.bounds != b.bounds || a.limits != b.limits || a.live != b.live || a.retained != b.retained || a.initialHeads != b.initialHeads || a.initialBytes != b.initialBytes || a.initialTotals != b.initialTotals || len(a.initialNodes) != 0 || len(b.initialNodes) != 0 || a.providerStart != "" || b.providerStart != "" || len(a.tickets) != 1 || len(b.tickets) != 1 || len(a.segments) != 1 || len(b.segments) != 1 || a.lastTicket == nil || b.lastTicket == nil || a.lastTicket.sequence != 0 || b.lastTicket.sequence != 0 || a.lastTicket.segment != 0 || b.lastTicket.segment != 0 || a.lastTicket.digest != b.lastTicket.digest || a.current == nil || b.current == nil || a.current == b.current {
		return false
	}
	x, y := a.current, b.current
	if x.digest != y.digest || x.admission != y.admission || x.sequence != y.sequence || x.sequence > 2 || x.heads != y.heads || x.counts != y.counts || x.totals != y.totals || x.payload != y.payload || x.metadata != y.metadata || x.privatePayload != y.privatePayload || x.privateMetadata != y.privateMetadata || x.clock != y.clock || len(x.nodes) == 0 || len(x.nodes) > 4 || len(x.nodes) != len(y.nodes) {
		return false
	}
	// At most eight raw comparisons here: policy, intent, ticket, budget and
	// four first-admission nodes. Later C/S each has only one new node. These
	// are native text comparisons with no unobserved temporary byte copies.
	for _, pair := range [][2]string{{a.policy, b.policy}, {a.intent.raw, b.intent.raw}, {a.lastTicket.raw, b.lastTicket.raw}, {x.raw, y.raw}} {
		if !providerReplayTextEqual(pair[0], pair[1], meter) {
			return false
		}
	}
	for i, raw := range x.nodes {
		if !providerReplayTextEqual(raw, y.nodes[i], meter) {
			return false
		}
	}
	if (a.lastConsumption == nil) != (b.lastConsumption == nil) {
		return false
	}
	if a.lastConsumption != nil && (a.lastConsumption == b.lastConsumption || a.lastConsumption.digest != b.lastConsumption.digest || !providerReplayTextEqual(a.lastConsumption.raw, b.lastConsumption.raw, meter)) {
		return false
	}
	for key, value := range proposed.tickets {
		other, ok := retained.tickets[key]
		if !ok || value != other {
			return false
		}
	}
	for key, value := range proposed.operations {
		other, ok := retained.operations[key]
		if !ok || value != other {
			return false
		}
	}
	for key, value := range proposed.segments {
		other, ok := retained.segments[key]
		if !ok || value != other {
			return false
		}
	}
	for key, value := range a.tickets {
		other, ok := b.tickets[key]
		if !ok || value.ticket == nil || other.ticket == nil || value.ticket.digest != other.ticket.digest || value.terminal != other.terminal || (value.consumption == nil) != (other.consumption == nil) || value.consumption != nil && value.consumption.digest != other.consumption.digest {
			return false
		}
	}
	for key, value := range a.segments {
		other, ok := b.segments[key]
		if !ok || value != other {
			return false
		}
	}
	c, d := a.continuation, b.continuation
	if c == nil || d == nil || c == d || !c.valid(a) || !d.valid(b) || c.bounds != d.bounds || c.page != 0 || d.page != 0 || c.pending != nil || d.pending != nil || c.final || d.final || c.sealedRecords != d.sealedRecords || c.sealedPayload != d.sealedPayload || c.sealedMetadata != d.sealedMetadata || c.extraMetadata != d.extraMetadata || c.sealedExtra != d.sealedExtra || c.auxiliaryMetadata != d.auxiliaryMetadata || c.sealedAuxiliary != d.sealedAuxiliary || c.auxiliaryPayload != d.auxiliaryPayload || c.sealedExternal != d.sealedExternal || c.lastSeal != d.lastSeal || c.lastPage != d.lastPage || c.lastBudget != d.lastBudget || len(c.records) == 0 || len(c.records) > 3 || len(c.records) != len(d.records) {
		return false
	}
	for i, value := range c.records {
		if value != d.records[i] {
			return false
		}
	}
	return !meter.failed
}
