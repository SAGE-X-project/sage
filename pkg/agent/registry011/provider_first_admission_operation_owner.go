package registry011

const (
	providerFirstOperationPrepared = iota + 1
	providerFirstOperationConsumption
	providerFirstOperationConsumed
	providerFirstOperationUsed
	providerFirstOperationSettlement
	providerFirstOperationDone
	providerFirstOperationFailed
)

// This private candidate owner is selected before I/T0. It cannot be restored
// from retained DATA or returned as an operating Source/Agent permission.
type providerFirstOperationOwner struct {
	owner                    *providerFirstOperationOwner
	store                    *providerCustodyStore
	first                    *providerFirstAdmissionOwner
	integration              *providerIntegration
	indexes                  *providerLedgerIndexes
	plan                     providerFirstOperationPlan
	config                   providerCustodyConfig
	policy                   ProviderPolicyCandidate
	continuation             providerLedgerBounds
	methods                  providerMethodBounds
	tls                      providerTLSBounds
	scan                     providerScanBounds
	firstBounds              providerFirstAdmissionBounds
	limits                   providerFirstAdmissionLimits
	clock                    *providerOwnedClock
	origin                   ClockSample
	stored, physical, clocks uint64
	execution, remaining     *providerWorkMeter
	change                   *providerFirstOperationChange
	permit                   *providerIntegrationPermit
	request                  *providerIntegrationRequest
	consumption, settlement  bool
	state                    int
}

type providerFirstOperationResultData struct {
	admission, ticket, exchange, enrollment string
	highwater                               providerFrontierHighwater
	execution, remaining                    candidateWorkVector
	physical                                uint64
	response, result                        []byte
}

func (*providerFirstOperationOwner) UnmarshalJSON([]byte) error      { return errProviderIntegration }
func (*providerFirstOperationResultData) UnmarshalJSON([]byte) error { return errProviderIntegration }

func (o *providerFirstOperationOwner) selectedLocked() bool {
	if o == nil || o.owner != o || o.store == nil || o.first == nil || o.store.firstOperation != o || o.first.operation != o || o.clock == nil || o.store.clock != o.clock || o.store.setupOrigin != o.origin || o.store.config != o.config || o.store.stored != o.stored || o.config.bounds.continuation == nil || o.config.bounds.methods == nil || o.config.bounds.tls == nil || o.config.bounds.scan == nil || o.config.bounds.firstAdmission == nil {
		return false
	}
	b := o.store.config.bounds
	return o.store.valid() && b.firstAdmission != nil && o.config.policy != nil && *o.config.policy == o.policy && o.store.firstLimits != nil && *o.store.firstLimits == o.limits && *b.continuation == o.continuation && *b.methods == o.methods && *b.tls == o.tls && *b.scan == o.scan && *b.firstAdmission == o.firstBounds && o.state >= providerFirstOperationPrepared && o.state < providerFirstOperationDone && o.physical <= o.plan.storage.physical && o.clocks <= 11
}

// Called with the observer mutex and the concrete store transaction held.
// Neither a second meter nor the remaining phase replenishes this owner.
func (o *providerFirstOperationOwner) fits(meter *providerWorkMeter) bool {
	if !o.selectedLocked() || o.first.state != providerFirstAdmissionRetained || o.integration == nil || o.integration.failed || o.integration.store != o.store || meter == nil || meter.firstOperation != o || meter.integration != o.integration || meter.scan != nil || o.store.activeMeter.Load() != meter || o.integration.firstTicket == nil || meter.ticket != o.integration.firstTicket.digest {
		return false
	}
	if meter.authority {
		return meter == o.remaining && o.state == providerFirstOperationSettlement && meter.upper == o.plan.reservation.remaining && meter.maximumClocks == 5
	}
	return meter == o.execution && o.state >= providerFirstOperationPrepared && o.state <= providerFirstOperationUsed && meter.upper == o.plan.reservation.execution && meter.maximumClocks == 7
}

func (o *providerFirstOperationOwner) writingLocked(s *providerCustodyStore) bool {
	if !o.selectedLocked() || o.store != s || o.change == nil || o.change.delta == nil || s.frontierReplay == nil || o.change.after.ordinal != s.frontierFiles.pin.ordinal+1 || o.change.after.sequence != s.sequence+1 || o.change.inventory == nil || o.change.inventory.previous != s.objectsRoot || o.change.root == nil || o.change.root.digest == s.enrollment || s.last != nil {
		return false
	}
	if o.state == providerFirstOperationConsumption {
		return !o.consumption && !o.settlement && s.sequence == 2 && len(o.change.fresh) == 6 && len(o.change.delta.mutations) == 2 && o.execution == s.activeMeter.Load()
	}
	return o.state == providerFirstOperationSettlement && o.consumption && !o.settlement && s.sequence == 3 && (len(o.change.fresh) == 11 || len(o.change.fresh) == 7) && len(o.change.delta.mutations) == 3 && o.remaining == s.activeMeter.Load()
}

func (o *providerFirstOperationOwner) consumeStorageLocked(physical uint64) bool {
	if o == nil || !o.writingLocked(o.store) || physical == 0 || !candidateCapacity(0, o.physical, physical, o.plan.storage.physical) {
		return false
	}
	raw, files := o.plan.storage.consumption, uint64(14)
	if o.state == providerFirstOperationSettlement {
		raw, files = o.plan.storage.settlement, 18
	}
	overhead, ok := providerUpperProduct(files, o.store.config.bounds.fileOverheadBytes)
	upper, sumOK := providerUpperSum(raw, overhead)
	if !ok || !sumOK || physical > upper {
		return false
	}
	// P was already held before I. This only consumes its one-use private
	// role; the store's retained/live upper never changes or gets refunded.
	o.physical += physical
	if o.state == providerFirstOperationConsumption {
		o.consumption = true
	} else {
		o.settlement = true
	}
	return true
}

func (o *providerFirstOperationOwner) failLocked() {
	if o == nil || o.owner != o || o.store == nil || o.store.firstOperation != o || o.state == providerFirstOperationDone {
		return
	}
	o.state, o.store.failed = providerFirstOperationFailed, true
	if o.integration != nil {
		o.integration.failed = true
	}
	if o.first != nil {
		o.first.abortLocked()
	}
	if g := o.store.guardian; g != nil && g.first == o {
		g.failLocked()
	}
	if r := o.store.frontierReplay; r != nil {
		r.failed = true
	}
	if f := o.store.frontierFiles; f != nil {
		f.failed = true
	}
	for _, meter := range []*providerWorkMeter{o.execution, o.remaining} {
		if meter != nil {
			meter.mu.Lock()
			meter.closed, meter.failed = true, true
			meter.mu.Unlock()
		}
	}
	o.store.activeMeter.Store(nil)
}

// Observation routing is explicit only for the newly bound first-operation
// metric. The original DATA-only F and every legacy metric stay unchanged.
func providerFirstOperationObserved(meter *providerWorkMeter) bool {
	return meter != nil && (meter.continuing != nil || meter.guardian != nil && meter.guardian.continuing != nil || meter.firstOperation != nil || meter.firstAdmission != nil && meter.firstAdmission.operation != nil)
}
