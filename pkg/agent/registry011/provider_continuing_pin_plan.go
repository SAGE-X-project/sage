package registry011

// This prospective local command funds both independent complete child
// replays. The original RPC reservations/closed observers do not pay it.
func providerContinuingPinPlan(g *providerGuardianOwner, command *providerContinuingPinCommand) (providerContinuingPlan, bool) {
	var empty providerContinuingPlan
	if g == nil || !g.selectedLocked() || !g.initialized || !g.admitted || g.integration == nil || g.integration.ledger == nil || command == nil || command.parent == nil || g.continuing != command.parent || command.parent.pin != command || command.input.kind != "pin-input" || !lowerHash(command.input.digest) || !lowerHash(command.selected) || g.config.bounds.methods == nil || g.config.bounds.tls == nil || g.store.firstLimits == nil {
		return empty, false
	}
	s, b, ledger := g.store, g.config.bounds, g.integration.ledger
	page := ledger.continuation
	if !ledger.valid() || page == nil || !page.valid(ledger) || page.pending != nil || page.final || page.bounds.pageRecords < 3 || uint64(len(page.records)) > page.bounds.pageRecords-3 || !lowerHash(g.integration.bootstrap) || ledger.providerStart != g.integration.bootstrap || s.sequence > ^uint64(0)-3 || s.sequence+3 > uint64(b.maximumCommits) || len(s.objects) > b.maximumObjects-32 {
		return empty, false
	}
	for _, value := range ledger.current.totals.bootstrap.outstanding {
		if value != 0 {
			return empty, false
		}
	}
	plan, storage, maximum, sizes, ok := providerContinuingPinResourceUpper(b, g.config.capture, command.requests, command.captures)
	if !ok {
		return empty, false
	}
	for coordinate, amount := range plan.reservation.work {
		if amount > s.firstLimits.operation[coordinate] || amount > s.firstLimits.segment[coordinate] {
			return empty, false
		}
	}
	memory, memoryOK := providerContinuingPinMemoryUpper(g, command, storage, plan.metadataRaw, maximum, sizes)
	if !memoryOK {
		return empty, false
	}
	plan.memory = memory
	return plan, true
}
