package registry011

// One live controller consumes the original first-exchange bridge exactly once.
// Its observer belongs to the ALREADY held future lifetime, not a new per-group
// allowance. Initializing these models grants no T/C/S ACK, capture permission,
// recovered operation or Source. The production derivation factory is absent.
type providerHistoryGroupOwner struct {
	owner       *providerHistoryGroupOwner
	parent      *providerHistoryProgramOwner
	store       *providerCustodyStore
	start       *providerHistoryProgramStart
	meter       *providerWorkMeter
	pin         providerFrontierHighwater
	scope       providerHistoryCustodyScope
	coordinator *providerHistoryGroupLedger
	custodian   *providerHistoryGroupLedger
	phase       string
	failed      bool
	retention   providerHistoryStorageCharge
	inventory   string
	last        *providerHistoryGroupAcknowledgment
	window      bool
	windowStart candidateWorkVector
	windowUpper candidateWorkVector
	clockStart  uint64
	clockUpper  uint64
}

func (*providerHistoryGroupOwner) UnmarshalJSON([]byte) error { return errProviderIntegration }

// This derives only the closed INITIALIZATION invocation envelope. For each of
// two independent passes: 4 reads, 1 decode, 3 encodes, 4 hashes, 3 comparisons,
// one selected-pin byte copy and 3 bounded filename copies. Two native samples
// bracket the original/model reconstruction. Two identity passes each perform four
// metadata reads for each root. All byte primitives fit 2*n+101; encode and
// decode use n*n+1. This is not a whole-program or allocator certificate.
func providerHistoryGroupInitializationUpper(b providerIntegrationBounds) (candidateWorkVector, bool) {
	var work candidateWorkVector
	n := max(b.maximumMetadataPerCommit, uint64(64<<10))
	q, a := providerUpperProduct(n, n)
	q, c := providerUpperSum(q, 1)
	quadratic, d := providerUpperProduct(8, q)
	linear, e := providerUpperProduct(2, n)
	linear, f := providerUpperSum(linear, 101)
	linear, g := providerUpperProduct(30, linear)
	units, h := providerUpperSum(quadratic, linear, 18)
	decoded, i := providerUpperProduct(2, n)
	if !a || !c || !d || !e || !f || !g || !h || !i {
		return work, false
	}
	work[2], work[10] = decoded, units
	return work, true
}

func newProviderHistoryGroupOwnerCandidate(s *providerCustodyStore) (*providerHistoryGroupOwner, error) {
	if s == nil {
		return nil, errProviderIntegration
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.history
	if p == nil || p.group != nil || !p.startReadyLocked() {
		return nil, errProviderIntegration
	}
	o := &providerHistoryGroupOwner{parent: p, store: s, start: p.start, pin: p.start.main, phase: "initializing"}
	o.owner = o
	m := newProviderWorkMeter(nil, providerOperationReservation{work: p.bounds.future.work, execution: p.bounds.future.work})
	m.historyGroup, m.maximumClocks, m.authority = o, p.bounds.future.clocks, true
	o.meter, p.group = m, o
	// Install the exact closed role BEFORE the first sample, read or codec.
	// A failed initialization permanently consumes this live bridge.
	s.activeMeter.Store(m)
	succeeded := false
	defer func() {
		if !succeeded {
			o.failLocked()
		}
	}()
	upper, ok := providerHistoryGroupInitializationUpper(s.config.bounds)
	if !ok || m.maximumClocks < 2 || !m.preflightWork(upper) || !o.identityLocked() {
		return nil, errProviderIntegration
	}
	begin, err := s.samplePrimitive()
	if err != nil || !o.clockFits(begin) {
		return nil, errProviderIntegration
	}
	a, scope, err := o.initialModelLocked()
	if err != nil {
		return nil, errProviderIntegration
	}
	b, other, err := o.initialModelLocked()
	if err != nil || a == b || a.head != b.head || scope != other || !o.identityLocked() {
		return nil, errProviderIntegration
	}
	end, err := s.samplePrimitive()
	if err != nil || !o.clockFits(end) || !candidateLedgerClockAdvance(providerClockDiagnostic(begin), providerClockDiagnostic(end)) ||
		uint64(end.MonotonicMS-begin.MonotonicMS) > 300000 || m.failed || m.closed {
		return nil, errProviderIntegration
	}
	o.coordinator, o.custodian, o.scope, o.phase = a, b, scope, "prepared"
	// Retain the same cumulative observer for future closed phases; idle code
	// cannot sample it or attach a replacement and renew its lifetime counters.
	s.activeMeter.Store(nil)
	succeeded = true
	return o, nil
}

func (o *providerHistoryGroupOwner) activeLocked(m *providerWorkMeter) bool {
	common := o != nil && o.owner == o && !o.failed && o.parent != nil && o.parent.group == o &&
		o.store != nil && o.store.owner == o.store && o.store.history == o.parent && !o.store.closed && !o.store.failed &&
		o.start == o.parent.start && o.start != nil && o.start.owner == o.start && o.start.parent == o.parent &&
		o.meter == m && m != nil && m.owner == m && m.historyGroup == o && m.integration == nil && m.scan == nil &&
		m.firstAdmission == nil && m.firstOperation == nil && m.guardian == nil && m.continuing == nil && m.parent == nil && m.authority &&
		o.store.activeMeter.Load() == m && o.store.config == o.parent.config && o.store.stored == o.parent.held.retained &&
		m.upper == o.parent.bounds.future.work && m.maximumClocks == o.parent.bounds.future.clocks &&
		o.store.frontierFiles != nil && o.store.frontierFiles.valid() && o.store.frontierFiles.initialized &&
		o.store.frontierFiles.pending == nil && o.store.frontierFiles.pin == o.pin &&
		o.pin == o.start.main && o.store.sequence == o.pin.sequence && o.store.tip == o.pin.commit && o.store.setupTip == o.pin.setup
	if !common {
		return false
	}
	pending := o.store.frontierFiles.historyPending
	switch o.phase {
	case "initializing", "preparing-admit":
		return pending == nil
	case "writing-admit":
		return pending != nil && pending.owner == pending && pending.group == o && pending.meter == m && !pending.used &&
			pending.frame != nil && pending.frame.before == o.pin && pending.frame.stage == 1 && pending.frame.phase == "admit"
	}
	return false
}

func (o *providerHistoryGroupOwner) fits(m *providerWorkMeter) bool {
	return o.activeLocked(m) && o.parent.matchesLocked()
}

// Called with the same cumulative meter mutex held. A group is a sublimit,
// never a replacement observer or a reset of the pre-H0 lifetime actuals.
func (o *providerHistoryGroupOwner) fitsWork(m *providerWorkMeter, incoming candidateWorkVector) bool {
	if !o.fits(m) {
		return false
	}
	if o.phase == "initializing" {
		return !o.window
	}
	if !o.window || m.clocks < o.clockStart || m.clocks-o.clockStart > o.clockUpper {
		return false
	}
	for i, n := range incoming {
		if m.actual[i] < o.windowStart[i] || !candidateCapacity(0, m.actual[i]-o.windowStart[i], n, o.windowUpper[i]) {
			return false
		}
	}
	return true
}

func (o *providerHistoryGroupOwner) clockFits(clock ClockSample) bool {
	return o.parent.original != (ClockSample{}) && candidateLedgerClockAdvance(providerClockDiagnostic(o.parent.original), providerClockDiagnostic(clock)) &&
		uint64(clock.MonotonicMS-o.parent.original.MonotonicMS) <= o.parent.bounds.durationMS
}

// The opened root alone is insufficient: it could still name old originals
// after the selected path or lock identity changed. Match both selected roots
// before and after the independent reads. checkIdentity performs exactly four
// metadata reads even on error; each invocation is funded before those calls.
func (o *providerHistoryGroupOwner) identityLocked() bool {
	if o == nil || !o.activeLocked(o.meter) || o.store.custody == nil || o.store.evidence == nil ||
		o.store.custody.directory != o.store.config.custodyDirectory || o.store.evidence.directory != o.store.config.evidenceDirectory {
		return false
	}
	for _, disk := range []*retentionDisk{o.store.custody, o.store.evidence} {
		if disk.root == nil || disk.lock == nil {
			return false
		}
		var metadata candidateWorkVector
		metadata[10] = 4
		if !o.meter.preflightWork(metadata) {
			return false
		}
		for range 4 {
			if !o.meter.event(providerReadEvent, 0) {
				return false
			}
		}
		if disk.checkIdentity() != nil {
			return false
		}
	}
	return true
}

func (o *providerHistoryGroupOwner) initialModelLocked() (*providerHistoryGroupLedger, providerHistoryCustodyScope, error) {
	var empty providerHistoryCustodyScope
	if !o.activeLocked(o.meter) {
		return nil, empty, errProviderIntegration
	}
	s, m, p := o.store, o.meter, o.parent
	maximum := int64(s.config.bounds.maximumMetadataPerCommit)
	head, a := s.read(s.custody, "HEAD", maximum)
	immutable, b := s.read(s.custody, s.name("highwater", o.pin.ordinal), maximum)
	frontier, c := s.read(s.custody, s.name("frontier", o.pin.sequence), maximum)
	commit, d := s.read(s.custody, s.name("commit", o.pin.sequence), maximum)
	expected := providerReplayPhaseBytes(s.frontierFiles.pinRaw, m)
	if a != nil || b != nil || c != nil || d != nil || !providerReplayEqual(head, expected, m) || !providerReplayEqual(immutable, expected, m) ||
		providerReplayDigest("sage-provider-custody-frontier-v1", frontier, m) != o.pin.frontier ||
		providerReplayDigest("sage-provider-custody-commit-v2", commit, m) != o.pin.commit {
		return nil, empty, errProviderIntegration
	}
	f, err := parseProviderFrontierRecordObserved(frontier, s.config.identity, m)
	if err != nil || f.sequence != o.pin.sequence || f.commit != o.pin.commit || f.setup != o.pin.setup || f.budget != o.start.budget ||
		f.enrollment != o.start.enrollment || f.objects != o.start.objects || f.admission != o.start.admission {
		return nil, empty, errProviderIntegration
	}
	canonical, err := f.bytesObserved(m)
	if err != nil || !providerReplayEqual(canonical, frontier, m) {
		return nil, empty, errProviderIntegration
	}
	initial := providerReplayCanonical(map[string]any{
		"format": "sage-provider-history-initial-model-v1", "program": p.bounds.program, "profile": s.config.profile, "cost_revision": p.bounds.costs,
		"initial_frontier": o.pin.frontier, "first_exchange": o.start.exchange, "budget": f.budget, "inventory": f.objects,
	}, candidateLedgerMetadataLimit, m)
	if len(initial) == 0 {
		return nil, empty, errProviderIntegration
	}
	initialRoot := providerReplayDigest("sage-provider-history-initial-model-v1", initial, m)
	if initialRoot == "" {
		return nil, empty, errProviderIntegration
	}
	model, err := newProviderHistoryGroupLedgerObserved(p.bounds.program, s.config.profile, p.bounds.costs, initialRoot, p.bounds.stages, p.bounds.future, m)
	if err != nil {
		return nil, empty, errProviderIntegration
	}
	return model, providerHistoryCustodyScope{initial: o.pin, exchange: o.start.exchange, origin: providerClockDiagnostic(p.original)}, nil
}

func (o *providerHistoryGroupOwner) failLocked() {
	if o == nil || o.owner != o || o.parent == nil || o.parent.group != o {
		return
	}
	o.failed, o.phase = true, "failed"
	o.parent.failLocked()
	o.store.failed = true
	if o.meter != nil && o.meter.owner == o.meter {
		o.meter.mu.Lock()
		o.meter.failed, o.meter.closed = true, true
		o.meter.mu.Unlock()
		if o.store.activeMeter.Load() == o.meter {
			o.store.activeMeter.Store(nil)
		}
	}
}
