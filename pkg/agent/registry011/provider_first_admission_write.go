package registry011

// Retained candidate diagnostics contain no live ledger, execution permit,
// acknowledgment, selected principal or facility for reconstructing F.
type providerFirstAdmissionRetainedData struct {
	admission, ticket, budget, enrollment string
	highwater                             providerFrontierHighwater
	work                                  candidateWorkVector
	physical                              uint64
}

func (*providerFirstAdmissionRetainedData) UnmarshalJSON([]byte) error { return errProviderIntegration }

// A candidate-only first-E1 transaction. The future request reservation is
// DATA, not an independently qualified quote: this entry cannot create an
// integration/Source, dispatch a request or return a custody ACK. Its exact
// physical write is nevertheless prepaid, one-use and independently replayed.
func (s *providerCustodyStore) retainFirstAdmissionCandidate(reservation providerOperationReservation) (*providerFirstAdmissionRetainedData, error) {
	if s == nil {
		return nil, errProviderIntegration
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	f := s.firstAdmissionOwner
	if !s.valid() || f == nil || f.store != s || f.state != providerFirstAdmissionPrepared || f.operation != nil || s.firstOperation != nil {
		return nil, errProviderIntegration
	}
	if f.startLocked() != nil {
		return nil, errProviderIntegration
	}
	success := false
	defer func() {
		if !success {
			f.abortLocked()
		}
	}()
	if !f.planLocked() {
		return nil, errProviderIntegration
	}
	input, err := f.replayEnrollmentLocked()
	if err != nil {
		return nil, errProviderIntegration
	}
	clock, err := f.sampleLocked()
	if err != nil {
		return nil, errProviderIntegration
	}
	p, err := providerFirstAdmissionProposalData(f, input, reservation, providerClockDiagnostic(clock))
	if err != nil {
		return nil, errProviderIntegration
	}
	// The second reader constructs its own original E0 maps, body and indexes.
	// Only those maps can be installed. The proposal's maps remain DATA.
	custodian, err := f.replayCustodianLocked()
	if err != nil || custodian.replay == input.replay || custodian.replay.indexes == p.indexes || custodian.replay.body.ledger != nil {
		return nil, errProviderIntegration
	}
	if _, err := f.sampleLocked(); err != nil || !f.adoptCustodianLocked(custodian) {
		return nil, errProviderIntegration
	}
	f.plan.used = true
	r := s.frontierReplay
	cost, err := r.retainOwnedLocked(s, p.inventory, p.fresh, p.claim, p.commit, p.frontier, p.after, nil, f)
	if err != nil || cost.objects != 14 || !f.physicalUsed || cost.stored != f.physical || s.stored != f.plan.stored || s.scanControl.held != f.plan.held || s.sequence != 2 || s.frontierFiles.pin != p.after || s.last != nil || r.body.intent == nil || r.body.ledger == nil || r.body.anchor == nil || r.body.intent.digest != p.intent.digest || r.body.ledger == p.ledger || r.body.ledger.current.digest != p.ledger.current.digest || r.body.ledger.current.totals != p.ledger.current.totals || r.body.intent.anchor != r.body.anchor.digest || r.current.enrollment != p.root.digest || s.enrollment != r.current.enrollment || r.indexes == p.indexes {
		return nil, errProviderIntegration
	}
	if _, err := f.sampleLocked(); err != nil || f.meter.failed || f.meter.closed {
		return nil, errProviderIntegration
	}
	// The immutable E0 anchor remains the admission's anchor. The newly
	// retained E1 enrollment is the independently derived current frontier.
	data := &providerFirstAdmissionRetainedData{r.body.intent.digest, r.body.ledger.lastTicket.digest, r.body.ledger.current.digest, r.current.enrollment, s.frontierFiles.pin, f.meter.actual, f.physical}
	// Observers deliberately keep the now-closed meter. Clearing/reopening it
	// would allow an unqualified next operation to mutate this retained DATA.
	f.state = providerFirstAdmissionRetained
	f.meter.mu.Lock()
	f.meter.closed = true
	f.meter.mu.Unlock()
	s.activeMeter.Store(nil)
	success = true
	return data, nil
}

func (f *providerFirstAdmissionOwner) adoptCustodianLocked(reader *providerScanReplayReader) bool {
	if f == nil || f.store == nil || f.plan == nil || f.plan.owner != f || f.plan.used || !f.custodianStarted || f.custodianAdopted || reader == nil || reader != f.custodian || !f.fits(candidateWorkVector{}) || !reader.valid() || reader.control != f.control || reader.meter != f.meter || reader.selected != f.mainPin {
		return false
	}
	s, v, r := f.store, reader.view, reader.replay
	if v == s || r == nil || r.owner != r || r.body == nil || r.indexes == nil || r.body.intent != nil || r.body.ledger != nil || r.setupOrdinal != 0 || r.meter != f.meter || r.indexes.meter != f.meter || r.body.meter != f.meter || s.frontierFiles.pin != f.mainPin || s.sequence != v.sequence || s.tip != v.tip || s.enrollment != v.enrollment || s.objectsRoot != v.objectsRoot || s.setupTip != v.setupTip || s.setupOrigin != v.setupOrigin || len(v.objects) != providerInitialObjectCount(s.config.bounds) || len(v.setups) != 1 || s.stored != f.plan.stored || s.scanControl.held != f.plan.held || s.retainedRaw < v.retainedRaw || !providerReplayTextEqual(s.frontierFiles.pinRaw, f.mainRaw, f.meter) {
		return false
	}
	// Keep original physical holds and the measured aggregate retainedRaw,
	// which also includes scan controls/receipts. The custodian's preI remains
	// E0-only; admissionScanSeed independently adds those originals once.
	s.objects, s.bindings, s.operations = v.objects, v.bindings, v.operations
	s.setupCommits, s.setups = v.setupCommits, v.setups
	s.requiredReferences, s.newReferences = nil, nil
	s.frontierReplay, s.replay = r, r.body
	f.custodianAdopted = true
	return true
}

func (f *providerFirstAdmissionOwner) writingLocked(s *providerCustodyStore) bool {
	return f != nil && s != nil && f.store == s && s.firstAdmissionOwner == f && f.state == providerFirstAdmissionStarted && f.plan != nil && f.plan.owner == f && f.plan.used && f.custodianStarted && f.custodianAdopted && f.custodian != nil && !f.physicalUsed && f.fits(candidateWorkVector{}) && s.sequence == 1 && s.frontierFiles.pin == f.mainPin && s.stored == f.plan.stored && s.scanControl.held == f.plan.held && s.frontierReplay == f.custodian.replay && s.frontierReplay.meter == f.meter && providerReplayTextEqual(s.frontierFiles.pinRaw, f.mainRaw, f.meter)
}

func (f *providerFirstAdmissionOwner) consumeLocked(physical uint64) bool {
	if f == nil || !f.writingLocked(f.store) || physical == 0 || physical > f.plan.physical || physical > f.limits.retained {
		return false
	}
	// B already included the full F retained/live upper before inspection.
	// Consume it once, without adding a second charge or refunding its tail.
	f.physicalUsed, f.physical = true, physical
	return true
}
