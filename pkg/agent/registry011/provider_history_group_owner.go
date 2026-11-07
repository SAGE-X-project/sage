package registry011

// One live controller consumes the original first-exchange bridge exactly once.
// Its observer belongs to the ALREADY held future lifetime, not a new per-group
// allowance. Initializing these models grants no T/C/S ACK, capture permission,
// recovered operation or Source. The production derivation factory is absent.
type providerHistoryGroupOwner struct {
	owner               *providerHistoryGroupOwner
	parent              *providerHistoryProgramOwner
	store               *providerCustodyStore
	start               *providerHistoryProgramStart
	meter               *providerWorkMeter
	pin                 providerFrontierHighwater
	scope               providerHistoryCustodyScope
	coordinator         *providerHistoryGroupLedger
	custodian           *providerHistoryGroupLedger
	phase               string
	failed              bool
	retention           providerHistoryStorageCharge
	inventory           string
	stageBefore         *providerHistoryRecoveryCheckpoint
	stageRetentionStart providerHistoryStorageCharge
	last                *providerHistoryGroupAcknowledgment
	admission           *providerHistoryGroupAcknowledgment
	order               *providerHistoryGroupOrder
	dispatch            *providerHistoryGroupDispatch
	window              bool
	windowStart         candidateWorkVector
	windowUpper         candidateWorkVector
	clockStart          uint64
	clockUpper          uint64
	advanceStart        candidateWorkVector
	advanceClockStart   uint64
	firstPReading       bool
	firstPStart         candidateWorkVector
}

func (*providerHistoryGroupOwner) UnmarshalJSON([]byte) error { return errProviderIntegration }

// This derives the legacy control-only initialization envelope. A collection
// profile additionally preflights two explicit original-read rows at the caller;
// those numeric sublimits are not a source-derived sufficient-cost proof. For
// each of two control passes: 4 reads, 1 decode, 3 encodes, 4 hashes, 3 comparisons,
// one selected-pin byte copy and 3 bounded filename copies, plus four copies
// sealing the initial recovery checkpoint and scan descriptor. Two native samples
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
	// Four additional bounded copies seal the checkpoint and scan descriptor.
	linear, g := providerUpperProduct(34, linear)
	units, h := providerUpperSum(quadratic, linear, 18)
	decoded, i := providerUpperProduct(2, n)
	if !a || !c || !d || !e || !f || !g || !h || !i {
		return work, false
	}
	work[2], work[10] = decoded, units
	return work, true
}

// The original constructor and irrevocable admission accounting must reserve
// the SAME initialization envelope. Collection adds two complete first-reader
// sublimits once; it neither borrows a later stage nor replaces each reader's
// own original sublimit. These remain numeric DATA pending full derivation.
func providerHistoryGroupInitializationForProfileUpper(b providerIntegrationBounds) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	upper, ok := providerHistoryGroupInitializationUpper(b)
	if !ok {
		return empty, false
	}
	if b.history == nil || b.history.collection == nil {
		return upper, true
	}
	manifest := b.history.collection
	if !manifest.validSnapshot() || !manifest.input.firstRead.valid() {
		return empty, false
	}
	for i, n := range manifest.input.firstRead.work {
		both, doubled := providerUpperProduct(2, n)
		value, added := providerUpperSum(upper[i], both)
		if !doubled || !added {
			return empty, false
		}
		upper[i] = value
	}
	return upper, true
}

func newProviderHistoryGroupOwnerCandidate(s *providerCustodyStore) (*providerHistoryGroupOwner, error) {
	if s == nil {
		return nil, errProviderIntegration
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.history
	if p == nil || p.group != nil || p.recovery == nil || p.recovery.originalGroup != nil || p.recovery.originalMeter != nil || !p.startReadyLocked() {
		return nil, errProviderIntegration
	}
	o := &providerHistoryGroupOwner{parent: p, store: s, start: p.start, pin: p.start.main, phase: "initializing"}
	o.owner = o
	m := newProviderWorkMeter(nil, providerOperationReservation{work: p.bounds.future.work, execution: p.bounds.future.work})
	m.historyGroup, m.maximumClocks, m.authority = o, p.bounds.future.clocks, true
	m.historyInputGroup = o
	o.meter, p.group = m, o
	// Bind the exact execution instance before its first native operation.
	p.recovery.originalGroup, p.recovery.originalMeter = o, m
	// Install the exact closed role BEFORE the first sample, read or codec.
	// A failed initialization permanently consumes this live bridge.
	s.activeMeter.Store(m)
	succeeded := false
	defer func() {
		if !succeeded {
			o.failLocked()
		}
	}()
	upper, ok := providerHistoryGroupInitializationForProfileUpper(s.config.bounds)
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
	checkpoint := p.recovery.prepareInitialCheckpointLocked(o, scope, a.head)
	if checkpoint == nil {
		return nil, errProviderIntegration
	}
	end, err := s.samplePrimitive()
	if err != nil || !o.clockFits(end) || !candidateLedgerClockAdvance(providerClockDiagnostic(begin), providerClockDiagnostic(end)) ||
		uint64(end.MonotonicMS-begin.MonotonicMS) > 300000 || m.failed || m.closed {
		return nil, errProviderIntegration
	}
	o.coordinator, o.custodian, o.scope, o.phase = a, b, scope, "prepared"
	p.recovery.initial, p.recovery.confirmed = checkpoint, checkpoint
	o.stageBefore, o.inventory = checkpoint, o.start.objects
	p.recovery.initialDescriptor = checkpoint.descriptor
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
		o.meter == m && m != nil && m.owner == m && m.historyGroup == o && m.historyInputGroup == o && m.integration == nil && m.scan == nil &&
		m.historyRecovery == nil && m.scanBody == nil && m.firstAdmission == nil && m.firstOperation == nil && m.guardian == nil && m.continuing == nil && m.parent == nil && m.authority &&
		o.store.activeMeter.Load() == m && o.store.config == o.parent.config && o.store.stored == o.parent.held.retained &&
		m.upper == o.parent.bounds.future.work && m.maximumClocks == o.parent.bounds.future.clocks &&
		o.store.frontierFiles != nil && o.store.frontierFiles.valid() && o.store.frontierFiles.initialized &&
		o.store.frontierFiles.pending == nil && o.store.frontierFiles.pin == o.pin &&
		o.store.sequence == o.pin.sequence && o.store.tip == o.pin.commit && o.store.setupTip == o.pin.setup
	if !common {
		return false
	}
	pending := o.store.frontierFiles.historyPending
	switch o.phase {
	case "initializing":
		return o.pin == o.start.main && pending == nil
	case "preparing-admit":
		return pending == nil && o.admissionBaseShapeLocked()
	case "advancing":
		return pending == nil && o.successfulSettlementShapeLocked()
	case "writing-admit":
		return o.admissionBaseShapeLocked() && pending != nil && pending.owner == pending && pending.group == o && pending.meter == m && !pending.used &&
			pending.frame != nil && pending.frame.before == o.pin && pending.frame.stage == o.nextStageLocked() && pending.frame.phase == "admit"
	case "preparing-consume":
		return pending == nil && o.admissionShapeLocked()
	case "preparing-dispatch":
		return pending == nil && o.dispatch == nil && o.consumptionShapeLocked()
	case "preparing-member", "capturing-member", "retaining-member":
		return pending == nil && o.committedConsumptionShapeLocked() && o.dispatch != nil && o.dispatch.activeMemberShapeLocked()
	case "preparing-settle":
		return pending == nil && o.dispatch != nil && o.dispatch.completedMemberShapeLocked()
	case "writing-settle":
		return pending != nil && pending.owner == pending && pending.group == o && pending.meter == m && !pending.used &&
			pending.dispatch == o.dispatch && pending.consumed == o.last && pending.frame != nil && pending.frame.before == o.pin &&
			pending.frame.stage == o.nextStageLocked() && pending.frame.phase == "settle" && o.dispatch != nil && o.dispatch.completedMemberShapeLocked()
	case "preparing-failed-settle":
		return pending == nil && o.dispatch != nil && o.dispatch.failedMemberShapeLocked()
	case "writing-failed-settle":
		return pending != nil && pending.owner == pending && pending.group == o && pending.meter == m && !pending.used &&
			pending.dispatch == o.dispatch && pending.consumed == o.last && pending.frame != nil && pending.frame.before == o.pin &&
			pending.frame.stage == o.nextStageLocked() && pending.frame.phase == "settle" && o.dispatch != nil && o.dispatch.failedMemberShapeLocked()
	case "writing-consume":
		return o.admissionShapeLocked() && pending != nil && pending.owner == pending && pending.group == o && pending.meter == m && !pending.used &&
			pending.admission == o.admission && pending.frame != nil && pending.frame.before == o.pin && pending.frame.stage == o.nextStageLocked() && pending.frame.phase == "consume"
	}
	return false
}

func (o *providerHistoryGroupOwner) fits(m *providerWorkMeter) bool {
	return o.activeLocked(m) && o.parent.matchesLocked()
}

// Called with the same cumulative meter mutex held. A group is a sublimit,
// never a replacement observer or a reset of the pre-H0 lifetime actuals.
func (o *providerHistoryGroupOwner) fitsWork(m *providerWorkMeter, incoming candidateWorkVector) bool {
	return o.activeLocked(m) && o.fitsWorkAfterOwnerCheck(m, incoming)
}

// Callers reusing a prior owner check hold store.mu and meter.mu throughout
// that same pure primitive. Standalone fitsWork validates the owner itself
// immediately before this call. Every subsequent parent/first-P/global-window/
// own-advance limit remains here; no authorization is cached across calls.
func (o *providerHistoryGroupOwner) fitsWorkAfterOwnerCheck(m *providerWorkMeter, incoming candidateWorkVector) bool {
	if o == nil || o.parent == nil || !o.parent.matchesLocked() || !o.firstPWorkFits(m, incoming) {
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
	if o.phase == "advancing" {
		rows := o.parent.dispatchBounds
		if !rows.advanceValid() || m.clocks < o.advanceClockStart || m.clocks-o.advanceClockStart > rows.advanceClocks {
			return false
		}
		for i, n := range incoming {
			if m.actual[i] < o.advanceStart[i] || !candidateCapacity(0, m.actual[i]-o.advanceStart[i], n, rows.advance[i]) {
				return false
			}
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
	if o == nil || o.store == nil || o.store.frontierFiles == nil {
		return nil, providerHistoryCustodyScope{}, errProviderIntegration
	}
	return o.initialModelAtLocked(o.pin, o.store.frontierFiles.pinRaw, o.store.frontierFiles.pinRaw)
}

// An immutable initial pin is replay DATA. Selecting it never moves the live
// pin, reinstalls an earlier role, or resets an observer/clock.
func (o *providerHistoryGroupOwner) initialModelAtLocked(pin providerFrontierHighwater, initialRaw, currentRaw string) (*providerHistoryGroupLedger, providerHistoryCustodyScope, error) {
	var empty providerHistoryCustodyScope
	if !o.activeLocked(o.meter) || pin != o.start.main {
		return nil, empty, errProviderIntegration
	}
	s, m, p := o.store, o.meter, o.parent
	maximum := int64(s.config.bounds.maximumMetadataPerCommit)
	head, a := s.read(s.custody, "HEAD", maximum)
	immutable, b := s.read(s.custody, s.name("highwater", pin.ordinal), maximum)
	frontier, c := s.read(s.custody, s.name("frontier", pin.sequence), maximum)
	commit, d := s.read(s.custody, s.name("commit", pin.sequence), maximum)
	expected := providerReplayPhaseBytes(initialRaw, m)
	current := expected
	if o.phase != "initializing" {
		current = providerReplayPhaseBytes(currentRaw, m)
	}
	if a != nil || b != nil || c != nil || d != nil || !providerReplayEqual(head, current, m) || !providerReplayEqual(immutable, expected, m) ||
		providerReplayDigest("sage-provider-custody-frontier-v1", frontier, m) != pin.frontier ||
		providerReplayDigest("sage-provider-custody-commit-v2", commit, m) != pin.commit {
		return nil, empty, errProviderIntegration
	}
	f, err := parseProviderFrontierRecordObserved(frontier, s.config.identity, m)
	if err != nil || f.sequence != pin.sequence || f.commit != pin.commit || f.setup != pin.setup || f.budget != o.start.budget ||
		f.enrollment != o.start.enrollment || f.objects != o.start.objects || f.admission != o.start.admission {
		return nil, empty, errProviderIntegration
	}
	canonical, err := f.bytesObserved(m)
	if err != nil || !providerReplayEqual(canonical, frontier, m) {
		return nil, empty, errProviderIntegration
	}
	var initialRoot string
	var cursor *providerHistoryCollectionCursor
	if p.bounds.collection != nil {
		var readErr error
		initialRoot, cursor, readErr = o.firstFinalizedInitialRootLocked(f)
		if readErr != nil {
			return nil, empty, errProviderIntegration
		}
	} else {
		initialRoot = providerHistoryInitialRootObserved(p.bounds, s.config.profile, pin, o.start.exchange, f.budget, f.objects, nil, m)
	}
	if initialRoot == "" {
		return nil, empty, errProviderIntegration
	}
	model, err := newProviderHistoryCollectionLedgerObserved(p.bounds.program, s.config.profile, p.bounds.costs, initialRoot, p.bounds.stages, p.bounds.future, cursor, m)
	if err != nil {
		return nil, empty, errProviderIntegration
	}
	return model, providerHistoryCustodyScope{initial: pin, exchange: o.start.exchange, origin: providerClockDiagnostic(p.original)}, nil
}

func (o *providerHistoryGroupOwner) failLocked() {
	if o == nil || o.owner != o || o.parent == nil || o.parent.group != o {
		return
	}
	o.releaseMemberReplayLocked()
	o.failed, o.phase = true, "failed"
	if o.order != nil {
		o.order.refuse() // Preserve every already consumed bit.
	}
	if o.store != nil && o.store.frontierFiles != nil {
		c := o.store.frontierFiles.historyPending
		if c != nil && c.owner == c && c.group == o && c.meter == o.meter && c.order != nil {
			c.order.refuse() // Also close the original when a cached pointer changed.
		}
	}
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
