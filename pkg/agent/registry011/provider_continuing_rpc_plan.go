package registry011

// Every coefficient belongs to the separately recorded closed small/large
// call table. The two sizes are not replaced by a single global response cap.
func providerContinuingRPCPlan(g *providerGuardianOwner, request *providerIntegrationRequest, capture ProviderCaptureLimits) (providerContinuingPlan, bool) {
	var empty providerContinuingPlan
	if g == nil || !g.selectedLocked() || !g.initialized || !g.admitted || g.integration == nil || g.integration.ledger == nil || g.first == nil ||
		request == nil || !capture.valid() || g.config.bounds.methods == nil || g.config.bounds.tls == nil || !lowerHash(g.integration.bootstrap) {
		return empty, false
	}
	s, b, ledger := g.store, g.config.bounds, g.integration.ledger
	page := ledger.continuation
	if !ledger.valid() || page == nil || !page.valid(ledger) || page.pending != nil || page.final || page.bounds.pageRecords < 3 ||
		uint64(len(page.records)) > page.bounds.pageRecords-3 || ledger.providerStart != g.integration.bootstrap ||
		s.firstLimits == nil || s.sequence > ^uint64(0)-3 || s.sequence+3 > uint64(b.maximumCommits) || len(s.objects) > b.maximumObjects-32 {
		return empty, false
	}
	for _, value := range ledger.current.totals.bootstrap.outstanding {
		if value != 0 {
			return empty, false
		}
	}
	plan, storage, maximum, ok := providerContinuingRPCResourceUpper(b, capture, uint64(len(request.wire)))
	if !ok {
		return empty, false
	}
	for i, value := range plan.reservation.work {
		if value > s.firstLimits.operation[i] || value > s.firstLimits.segment[i] {
			return empty, false
		}
	}
	memory, memoryOK := providerContinuingRPCMemoryUpper(g, request, capture, storage, plan.metadataRaw, maximum)
	if !memoryOK {
		return empty, false
	}
	plan.memory = memory
	return plan, true
}
