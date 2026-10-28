package registry011

// This one-use live owner preserves the exact open-O meter through E0. It
// returns retained candidate DATA only, never a principal, ACK or Source.
type providerBootstrapOwner struct {
	store                       *providerCustodyStore
	meter                       *providerWorkMeter
	setupPhysical, planned      uint64
	physical                    uint64
	before                      string
	origin                      ClockSample
	started, used, done, failed bool
}

func (*providerBootstrapOwner) UnmarshalJSON([]byte) error { return errProviderIntegration }

// The finite E0 has seven fresh objects: enrollment, three empty indexes,
// original setup/commit and one operations-index mutation. Its other seven
// files are inventory, intent/pending/commit/frontier and two head copies.
// The upper includes a HEAD.pending file even when its outcome is unknown.
// It is compared with the old O; no increased envelope is installed here.
func providerBootstrapUpper(b providerIntegrationBounds) (candidateWorkVector, uint64, bool) {
	var work candidateWorkVector
	if b.firstAdmission == nil || !b.valid() || b.maximumEntries < 7 || b.maximumObjects < 7 {
		return work, 0, false
	}
	metadata, a := providerUpperProduct(b.maximumMetadataPerCommit, 5)
	fixed, c := providerUpperSum(3*4096, 3*candidateLedgerMetadataLimit, 3*(64<<10))
	bytes, d := providerUpperSum(metadata, fixed)
	allocation, e := providerUpperProduct(b.fileOverheadBytes, 14)
	physical, f := providerUpperSum(bytes, allocation)
	// Conservative full-schema equivalents cover root/index construction64,
	// inventory/order/readbacks128, independent typed replay128, original
	// setup/high-water checks128, and bounded filename/scalar copies64.
	// These are held logical uppers, never an actual invocation count.
	n := max(b.maximumMetadataPerCommit, uint64(64<<10))
	square, g := providerUpperProduct(n, n)
	units, h := providerUpperSum(square, n, 16)
	units, i := providerUpperProduct(units, 512)
	decoded, j := providerUpperProduct(n, 512)
	work[2], work[9], work[10] = decoded, bytes, units
	if b.guardian != nil {
		extra, extraPhysical, extraOK := providerGuardianBootstrapExtraUpper(b)
		if !extraOK || b.maximumEntries < 8 || b.maximumObjects < 8 {
			return candidateWorkVector{}, 0, false
		}
		for k, amount := range extra {
			var ok bool
			work[k], ok = providerUpperSum(work[k], amount)
			if !ok {
				return candidateWorkVector{}, 0, false
			}
		}
		var ok bool
		physical, ok = providerUpperSum(physical, extraPhysical)
		if !ok {
			return candidateWorkVector{}, 0, false
		}
	}
	return work, physical, a && c && d && e && f && g && h && i && j
}

func (o *providerBootstrapOwner) writing(s *providerCustodyStore) bool {
	return o != nil && o.store == s && s.bootstrapOwner == o && o.started && !o.used && !o.done && !o.failed && o.meter != nil && o.meter.owner == o.meter && !o.meter.closed && !o.meter.failed && s.activeMeter.Load() == o.meter && s.sequence == 0 && s.enrollment == "" && providerGuardianStoredMatches(s.config.bounds, s.stored) && s.frontierFiles != nil && s.frontierFiles.pinRaw == o.before && s.setupOrigin == o.origin
}

func (o *providerBootstrapOwner) consume(bytes uint64) bool {
	if o == nil || o.store == nil || !o.writing(o.store) || bytes == 0 || bytes > o.planned {
		return false
	}
	total, ok := providerUpperSum(o.setupPhysical, bytes)
	if !ok || total > o.store.config.bounds.setupRetained {
		return false
	}
	o.used, o.physical = true, total
	return true
}

// enrollBeforeScan is initial-only. Reopening files or losing the live owner
// cannot recreate O, an authenticated absence witness or permission to retry.
func (s *providerCustodyStore) enrollBeforeScan() (*CandidateEnrollmentRoot, error) {
	if s == nil {
		return nil, errProviderIntegration
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	o, r := s.bootstrapOwner, s.frontierReplay
	if !s.valid() || o == nil || o.store != s || o.started || o.done || o.failed || o.meter == nil || o.meter.owner != o.meter || o.meter.closed || o.meter.failed || o.meter.upper != s.config.bounds.setupWork || o.meter.maximumClocks != s.config.bounds.setupMaximumClockReads || s.activeMeter.Load() != nil || s.scanControl == nil || s.scanControl.prepared != nil || s.scanControl.pin.ordinal != 0 || s.sequence != 0 || r == nil || r.body == nil || r.body.ledger != nil || r.meter != nil || r.indexes == nil || r.indexes.meter != nil || s.frontierFiles == nil || s.frontierFiles.pinRaw != o.before || s.setupOrigin != o.origin {
		return nil, errProviderIntegration
	}
	o.started = true // No primitive may precede this irreversible transition.
	m := o.meter
	s.activeMeter.Store(m)
	defer s.activeMeter.Store(nil)
	r.meter, r.body.meter, r.indexes.meter = m, m, m
	defer func() { r.meter, r.body.meter, r.indexes.meter = nil, nil, nil }()
	success := false
	defer func() {
		m.mu.Lock()
		m.closed = true
		m.mu.Unlock()
		if !success {
			o.failed, s.failed = true, true
			m.refuse()
		}
	}()
	work, physical, ok := providerBootstrapUpper(s.config.bounds)
	whole, physicalOK := providerUpperSum(o.setupPhysical, physical)
	if !ok || !physicalOK || whole > s.config.bounds.setupRetained || o.setupPhysical > s.config.bounds.setupRetained || m.clocks > m.maximumClocks || m.maximumClocks-m.clocks < 8 {
		return nil, errProviderIntegration
	}
	for k, n := range work {
		if !candidateCapacity(0, m.actual[k], n, m.upper[k]) {
			return nil, errProviderIntegration
		}
	}
	if _, room := s.config.bounds.continuation.residentUpper(1, 1, 1, uint64(providerInitialObjectCount(s.config.bounds)), physical, 2*physical); !room {
		return nil, errProviderIntegration
	}
	o.planned = physical
	start, err := s.sample()
	if err != nil || !o.clock(start) || !s.scanControl.readHighwater(s.scanControl.pin) || !s.scanControl.checkMainPin() {
		return nil, errProviderIntegration
	}
	root, err := s.buildInitialEnrollment(m)
	if err != nil {
		return nil, errProviderIntegration
	}
	if s.guardian != nil && s.guardian.installGenesisLocked(o) != nil {
		return nil, errProviderIntegration
	}
	end, err := s.sample()
	if err != nil || !o.clock(end) || !o.used || m.failed || !providerGuardianStoredMatches(s.config.bounds, s.stored) || s.retainedRaw > o.physical {
		return nil, errProviderIntegration
	}
	o.done, success = true, true
	return root, nil
}

func (o *providerBootstrapOwner) clock(now ClockSample) bool {
	return candidateLedgerClockAdvance(providerClockDiagnostic(o.origin), providerClockDiagnostic(now)) && uint64(now.MonotonicMS-o.origin.MonotonicMS) <= o.store.config.bounds.setupMaximumDurationMS
}

// Coordinator state is new; the custodian's independent indexes and replay
// are not copied into this proposal. Every serializer uses the current O.
func (s *providerCustodyStore) buildInitialEnrollment(m *providerWorkMeter) (*CandidateEnrollmentRoot, error) {
	id := s.config.identity
	raw := providerReplayCanonical(map[string]any{"format": "sage-provider-enrollment-v2", "namespace": id.Namespace, "instance_key": id.Instance, "policy": id.Policy, "source_id": id.SourceID, "generation": "0", "sequence": "0", "previous_root": nil, "status": "enrolling", "checkpoint": nil, "budget_root": nil, "transition_root": nil, "fence": "1"}, 64<<10, m)
	root, err := parseCandidateEnrollmentRootObserved(&ProviderCandidateContext{id, true}, raw, 64<<10, m)
	indexes, anchors, indexErr := newProviderLedgerIndexesObserved(id, 1, *s.config.bounds.continuation, m)
	entry, entryErr := newProviderIntegrationObjectObserved("enrollment", raw, 64<<10, m)
	joined := providerReplayJoin([]string{s.config.anchor, id.Instance, id.Policy, id.SourceID}, m)
	operation := providerReplayDigest("sage-provider-enroll-operation-v1", providerReplayBytes(joined, m), m)
	if err != nil || indexErr != nil || entryErr != nil || joined == "" || operation == "" {
		return nil, errProviderIntegration
	}
	indexes.meter = m // Creation alone deliberately does not retain an observer.
	delta, err := indexes.prepare("enroll", operation, 1, entry, root, nil)
	if err != nil || len(delta.mutations) != 1 || delta.mutations[0].chain != 1 {
		return nil, errProviderIntegration
	}
	fresh := make([]providerIntegrationObject, 0, 7)
	fresh = append(fresh, entry)
	fresh = append(fresh, anchors...)
	for _, input := range []struct {
		kind, name string
		disk       *retentionDisk
	}{{"setup", s.name("setup-intent", 0), s.custody}, {"setup-commit", s.name("setup-commit", 0), s.custody}} {
		original, readErr := s.read(input.disk, input.name, int64(s.config.bounds.maximumMetadataPerCommit))
		object, objectErr := newProviderIntegrationObjectObserved(input.kind, original, s.config.bounds.maximumMetadataPerCommit, m)
		if readErr != nil || objectErr != nil {
			return nil, errProviderIntegration
		}
		fresh = append(fresh, object)
	}
	fresh = append(fresh, delta.mutations[0].object)
	if s.guardian != nil {
		origin, err := s.guardian.originBeforeEnrollmentLocked(m)
		if err != nil {
			return nil, errProviderIntegration
		}
		fresh = append(fresh, origin)
	}
	inventory, err := newProviderFrontierInventoryObserved(s.objectsRoot, fresh, s.config.bounds.maximumEntries, min(s.config.bounds.maximumStoredBytes, 64<<20), m)
	clock, clockErr := s.sample()
	if err != nil || clockErr != nil || !s.bootstrapOwner.clock(clock) {
		return nil, errProviderIntegration
	}
	link := providerFrontierLink{identity: id, fence: 1, commit: s.tip, setup: s.setupTip, objects: s.objectsRoot}
	intent, err := providerFrontierIntentDiagnosticObserved(link, root, inventory, "enroll", operation, providerClockDiagnostic(clock), s.config.bounds, m)
	commit := providerFrontierCommitObserved(intent, m)
	x, codecErr := newProviderScanCodec(s.config, m)
	if err != nil || codecErr != nil {
		return nil, errProviderIntegration
	}
	deadline, _, ok := x.deadline(providerClockDiagnostic(s.setupOrigin))
	heads := delta.previous
	heads[1] = delta.mutations[0].object.digest
	f := providerFrontierRecord{identity: id, fence: 1, sequence: 1, commit: providerReplayDigest("sage-provider-custody-commit-v2", commit, m), setup: s.setupTip, enrollment: root.digest, objects: inventory.digest, indexes: heads, initial: providerClockDiagnostic(s.setupOrigin), clock: providerClockDiagnostic(clock), deadline: deadline}
	frontier, err := f.bytesObserved(m)
	after := s.frontierFiles.pin
	after.ordinal, after.sequence, after.previous = 1, 1, providerReplayDigest("sage-provider-custody-highwater-v1", providerReplayBytes(s.frontierFiles.pinRaw, m), m)
	after.commit, after.frontier = f.commit, providerReplayDigest("sage-provider-custody-frontier-v1", frontier, m)
	if !ok || err != nil {
		return nil, errProviderIntegration
	}
	if _, err = s.frontierReplay.retainLocked(s, inventory, fresh, intent, commit, frontier, after, s.bootstrapOwner); err != nil {
		return nil, errProviderIntegration
	}
	return root, nil
}
