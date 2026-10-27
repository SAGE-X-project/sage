package registry011

// This fixed-shape private marker belongs only to the same continuously live
// guardian. The initial marker must follow the complete first-R comparison;
// later markers follow both known typed deltas and their bounded comparison.
// It is not reconstructed from files, snapshots, digests, or candidate JSON.
// It summarizes roots/counts; it never imports or enumerates either old map.
type providerContinuingPrefix struct {
	owner                        *providerGuardianOwner
	coordinator, retained        *providerLedgerIndexes
	coordinatorLedger, replay    *CandidateAdmissionLedger
	main                         providerFrontierHighwater
	enrollment, ticket, consumed string
	heads                        [3]string
	counts                       [3]uint64
	mapSizes                     [7]int
	schemaBytes                  uint64
	stored                       uint64
	budget                       providerContinuingBudgetPin
	page                         providerContinuingPagePin
}

type providerContinuingBudgetPin struct {
	digest, admission           string
	sequence                    uint64
	heads                       [3]string
	counts                      [3]uint64
	totals                      candidateLedgerTotals
	payload, metadata           uint64
	privatePayload, privateMeta uint64
	clock                       CandidateClockDiagnostic
}

type providerContinuingPagePin struct {
	page, sealedRecords, sealedPayload, sealedMetadata uint64
	extraMetadata, sealedExtra, auxiliaryMetadata      uint64
	sealedAuxiliary, auxiliaryPayload, sealedExternal  uint64
	lastSeal, lastPage, lastBudget                     string
	records                                            int
}

func (*providerContinuingPrefix) UnmarshalJSON([]byte) error { return errProviderIntegration }

// Snapshot scalars only after the caller has independently established full
// first-prefix equality or exact typed-delta equality. This helper alone does
// not establish that prerequisite, install a marker, or grant permission.
func providerContinuingPrefixData(g *providerGuardianOwner) (providerContinuingPrefix, bool) {
	var empty providerContinuingPrefix
	if g == nil || !g.selectedLocked() || g.integration == nil || g.indexes == nil || g.store.frontierFiles == nil || g.store.frontierReplay == nil || g.store.frontierReplay.body == nil {
		return empty, false
	}
	s, c, proposed, retained := g.store, g.integration, g.indexes, g.store.frontierReplay.indexes
	if !c.valid() || !proposed.valid() || retained == nil || retained == proposed || !retained.valid() || proposed.ledger != c.ledger || retained.ledger == c.ledger || retained.ledger != s.frontierReplay.body.ledger || c.root == nil || c.root.digest != s.enrollment {
		return empty, false
	}
	a, b := c.ledger, retained.ledger
	if a == nil || b == nil || a.current == nil || b.current == nil || a.current == b.current || a.lastTicket == nil || b.lastTicket == nil || a.lastTicket == b.lastTicket || a.lastTicket.digest != b.lastTicket.digest || (a.lastConsumption == nil) != (b.lastConsumption == nil) {
		return empty, false
	}
	budget := providerContinuingBudgetData(a.current)
	page, ok := providerContinuingPageData(a.continuation)
	otherPage, otherOK := providerContinuingPageData(b.continuation)
	if budget != providerContinuingBudgetData(b.current) || !ok || !otherOK || page != otherPage || proposed.heads != retained.heads || proposed.counts != retained.counts || proposed.schemaBytes != retained.schemaBytes || len(proposed.tickets) != len(retained.tickets) || len(proposed.operations) != len(retained.operations) || len(proposed.segments) != len(retained.segments) || len(a.tickets) != len(b.tickets) || len(a.segments) != len(b.segments) {
		return empty, false
	}
	consumed := ""
	if a.lastConsumption != nil {
		if a.lastConsumption == b.lastConsumption || a.lastConsumption.digest != b.lastConsumption.digest {
			return empty, false
		}
		consumed = a.lastConsumption.digest
	}
	return providerContinuingPrefix{owner: g, coordinator: proposed, retained: retained, coordinatorLedger: a, replay: b, main: s.frontierFiles.pin, enrollment: s.enrollment, ticket: a.lastTicket.digest, consumed: consumed, heads: proposed.heads, counts: proposed.counts, mapSizes: [7]int{len(proposed.tickets), len(proposed.operations), len(proposed.segments), len(a.tickets), len(a.segments), len(s.objects), len(s.bindings)}, schemaBytes: proposed.schemaBytes, stored: s.stored, budget: budget, page: page}, true
}

func providerContinuingBudgetData(value *CandidateAdmissionLedgerChange) providerContinuingBudgetPin {
	return providerContinuingBudgetPin{digest: value.digest, admission: value.admission, sequence: value.sequence, heads: value.heads, counts: value.counts, totals: value.totals, payload: value.payload, metadata: value.metadata, privatePayload: value.privatePayload, privateMeta: value.privateMetadata, clock: value.clock}
}

func providerContinuingPageData(value *providerLedgerContinuation) (providerContinuingPagePin, bool) {
	if value == nil || value.pending != nil || value.final || len(value.records) == 0 {
		return providerContinuingPagePin{}, false
	}
	return providerContinuingPagePin{page: value.page, sealedRecords: value.sealedRecords, sealedPayload: value.sealedPayload, sealedMetadata: value.sealedMetadata, extraMetadata: value.extraMetadata, sealedExtra: value.sealedExtra, auxiliaryMetadata: value.auxiliaryMetadata, sealedAuxiliary: value.sealedAuxiliary, auxiliaryPayload: value.auxiliaryPayload, sealedExternal: value.sealedExternal, lastSeal: value.lastSeal, lastPage: value.lastPage, lastBudget: value.lastBudget, records: len(value.records)}, true
}

// This check follows observer attachment under the newly consumed G. No raw
// or mutable cache is imported, and a mismatch permanently fails its owner.
func (g *providerGuardianOwner) continuingPrefixMatchesLocked() bool {
	if g == nil || g.prefix.owner != g {
		return false
	}
	value, ok := providerContinuingPrefixData(g)
	return ok && value == g.prefix
}
