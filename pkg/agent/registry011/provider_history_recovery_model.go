package registry011

import "context"

// This owns a fresh DATA model of the four initial main records. It has no
// store facade, live observer installation, producer cache or execution role.
// The later group/member replay, catalog and sufficient-cost gates remain
// separate prerequisites for a completed recovery result.
type providerHistoryRecoveryInitialModel struct {
	owner             *providerHistoryRecoveryInitialModel
	recovery          *providerHistoryRecoveryOwner
	prefix            *providerHistoryRecoveryLegacyPrefix
	frontier          *providerFrontierReplay
	root              *CandidateEnrollmentRoot
	used              [4][]bool
	scanReadCounts    [2]providerHistoryRecoveryReadCount
	scanReadsVerified bool
	through           uint64
	complete          bool
	firstFinalized    FinalizedHeader
	finalizedSelected bool
}

func (*providerHistoryRecoveryInitialModel) UnmarshalJSON([]byte) error {
	return errProviderIntegration
}

func (m *providerHistoryRecoveryInitialModel) selectedLocked(r *providerHistoryRecoveryOwner) bool {
	return m != nil && m.owner == m && r != nil && m.recovery == r && m.prefix != nil && m.prefix.selectedLocked(r) &&
		r.initialModels[m.prefix.pass] == m && m.complete && m.scanReadsVerified && m.through == 4 && m.root != nil && m.frontier != nil &&
		m.frontier.owner == m.frontier && !m.frontier.failed && m.frontier.current.sequence == 4 &&
		m.frontier.body != nil && m.frontier.body.ledger != nil && m.frontier.body.ledger.valid() &&
		m.frontier.meter == r.meter && m.frontier.body.meter == r.meter && m.frontier.indexes != nil &&
		m.frontier.indexes.meter == r.meter && m.frontier.indexes.ledger == m.frontier.body.ledger
}

// Lookup is restricted to this pass's actual immutable frames and current
// main boundary. Every successful lookup marks semantic use independently of
// the earlier hash/reference walk. No caller-supplied resolver is accepted.
func (m *providerHistoryRecoveryInitialModel) object(kind, digest string) ([]byte, error) {
	if m == nil || m.owner != m || m.complete || m.recovery == nil || !m.recovery.activeLocked(m.recovery.meter) ||
		m.prefix == nil || !m.prefix.selectedLocked(m.recovery) || m.through == 0 || m.through > 4 || !lowerHash(digest) {
		return nil, errProviderIntegration
	}
	r := m.recovery
	for i := uint64(0); i < m.through; i++ {
		frame := m.prefix.source.frames[i]
		if len(m.used[i]) != len(frame.objects) {
			return nil, errProviderIntegration
		}
		for j, original := range frame.objects {
			if !r.meter.event(providerCompareEvent, uint64(len(kind)+len(original.kind))) {
				return nil, errProviderIntegration
			}
			if kind != original.kind {
				continue
			}
			if !r.meter.event(providerCompareEvent, uint64(len(digest)+len(original.digest))) {
				return nil, errProviderIntegration
			}
			if digest != original.digest {
				continue
			}
			if len(original.raw) == 0 || uint64(len(original.raw)) > r.config.bounds.maximumObjectBytes ||
				kind == "request" && len(original.raw) > 16<<10 || kind == "params" && len(original.raw) > 8<<10 {
				return nil, errProviderIntegration
			}
			raw := providerReplayPhaseBytes(original.raw, r.meter)
			if len(raw) == 0 || !r.meter.event(providerCopyEvent, 1) {
				return nil, errProviderIntegration
			}
			m.used[i][j] = true
			return raw, nil
		}
	}
	return nil, errProviderIntegration
}

func (r *providerHistoryRecoveryOwner) replayInitialModelCandidate(ctx context.Context, pass int) (*providerHistoryRecoveryInitialModel, error) {
	if r == nil || r.store == nil {
		return nil, errProviderIntegration
	}
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if !r.activeLocked(r.meter) || pass < 0 || pass >= len(r.initialModels) || r.initialModels[pass] != nil {
		return nil, errProviderIntegration
	}
	p := r.legacyPrefixes[pass]
	if p == nil || !p.selectedLocked(r) || pass == 1 && (r.initialModels[0] == nil || !r.initialModels[0].selectedLocked(r)) {
		return nil, errProviderIntegration
	}
	accepted := false
	defer func() {
		if !accepted {
			r.stopLocked(true)
		}
	}()
	if !r.inputReadyLocked(ctx) {
		return nil, errProviderIntegration
	}
	f, err := newProviderFrontierReplayObserved(r.config, r.meter)
	if err != nil || f == nil {
		return nil, errProviderIntegration
	}
	// This constructor created private arithmetic state only. Explicitly bind
	// its observations to the existing read-only row, never store.activeMeter.
	f.meter, f.body.meter, f.indexes.meter = r.meter, r.meter, r.meter
	policy, err := parseProviderPolicyCandidateObserved(providerReplayPhaseBytes(r.config.policy.raw, r.meter), 64<<10, r.meter)
	if err != nil || *policy != *r.config.policy {
		return nil, errProviderIntegration
	}
	f.body.policy = policy
	m := &providerHistoryRecoveryInitialModel{recovery: r, prefix: p, frontier: f}
	m.owner = m
	for i, frame := range p.source.frames {
		if !r.meter.event(providerCopyEvent, uint64(len(frame.objects))) {
			return nil, errProviderIntegration
		}
		m.used[i] = make([]bool, len(frame.objects))
	}
	for i, record := range p.records {
		m.through = uint64(i + 1)
		if !r.inputReadyLocked(ctx) || m.apply(record) != nil || !r.inputReadyLocked(ctx) {
			return nil, errProviderIntegration
		}
	}
	last := f.current
	for _, pair := range [][2]string{{last.commit, p.pin.commit}, {last.enrollment, r.initialDescriptor.enrollment},
		{last.objects, r.initialDescriptor.objects}, {f.body.intent.digest, r.initialDescriptor.admission}, {f.body.ledger.lastTicket.digest, r.initialDescriptor.ticket},
		{f.body.ledger.current.digest, r.initialDescriptor.budget}, {f.body.firstExchange, r.initialDescriptor.exchange}} {
		if !providerReplayTextEqual(pair[0], pair[1], r.meter) {
			return nil, errProviderIntegration
		}
	}
	if !m.verifyScanReadCounts(ctx) || !r.inputReadyLocked(ctx) {
		return nil, errProviderIntegration
	}
	m.complete, r.initialModels[pass], accepted = true, m, true
	return m, nil
}

func (m *providerHistoryRecoveryInitialModel) apply(record providerHistoryRecoveryLegacyRecord) error {
	r, f := m.recovery, m.frontier
	frame := m.prefix.source.frames[m.through-1]
	if record.after.sequence != m.through || f.current.sequence+1 != m.through || frame.sequence != m.through {
		return errProviderIntegration
	}
	intent, commit, rawFrontier := providerReplayPhaseBytes(record.intent, r.meter), providerReplayPhaseBytes(record.commit, r.meter), providerReplayPhaseBytes(record.rawFrontier, r.meter)
	v := providerFrontierIntentObjectObserved(intent, r.meter)
	rootRaw, err := m.object("enrollment", contentString(v["new_enrollment_root"]))
	root, parseErr := parseCandidateEnrollmentRootObserved(f.body.context, rootRaw, 64<<10, r.meter)
	if err != nil || parseErr != nil || root.digest != record.root.digest || record.kind != contentString(v["kind"]) || record.operation != contentString(v["operation"]) {
		return errProviderIntegration
	}
	if m.root == nil {
		if record.kind != "enroll" || root.sequence != 0 || root.status != "enrolling" || root.fence != 1 {
			return errProviderIntegration
		}
	} else {
		kind := "update-enrolling"
		if record.kind == "admit" {
			kind = "install-budget"
		}
		if CheckCandidateEnrollmentChange(m.root, root, kind) != nil {
			return errProviderIntegration
		}
	}
	setup, err := m.setup(record)
	if err != nil || m.through == 1 && !m.scanState(0) || m.oldMaterials(record) != nil {
		return errProviderIntegration
	}
	entry, err := m.applyBody(record, root, setup)
	if err != nil {
		return errProviderIntegration
	}
	if f.body.ledger != nil && f.indexes.ledger == nil {
		metadata, ok := providerUpperSum(f.preI.metadata, f.body.scanMetadata)
		if !ok || f.body.scanMetadata == 0 || metadata > f.indexes.bounds.maximumMetadataBytes {
			return errProviderIntegration
		}
		f.preI.metadata = metadata
		if f.indexes.bindRetainedLedger(f.body.ledger, f.preI) != nil {
			return errProviderIntegration
		}
	}
	delta, err := f.indexes.prepare(record.kind, record.operation, m.through, entry, root, f.body.ledger)
	if err != nil {
		return errProviderIntegration
	}
	if record.kind == "enroll" {
		for _, anchor := range f.anchors {
			if m.exactObject(anchor) != nil {
				return errProviderIntegration
			}
		}
	}
	for _, mutation := range delta.mutations {
		if m.exactObject(mutation.object) != nil {
			return errProviderIntegration
		}
	}
	derived, err := m.deriveFrontier(record, root, delta, commit, setup)
	if err != nil {
		return errProviderIntegration
	}
	canonical, err := derived.bytesObserved(r.meter)
	if err != nil || !providerReplayEqual(canonical, rawFrontier, r.meter) || m.through > 1 && !checkProviderFrontierAdvanceObserved(f.current, derived, r.meter) {
		return errProviderIntegration
	}
	cost, err := providerFrontierRetentionCostObserved(r.config.identity, frame.inventory, frame.objects, intent, commit, rawFrontier, r.config.bounds, r.meter)
	if err != nil {
		return errProviderIntegration
	}
	cost, err = m.retention(record, setup, cost)
	if err != nil {
		return errProviderIntegration
	}
	if record.kind == "enroll" {
		if f.indexes.apply(delta) != nil {
			return errProviderIntegration
		}
		f.preI = cost
	} else if f.indexes.applyRetained(delta, cost) != nil {
		return errProviderIntegration
	}
	for _, used := range m.used[m.through-1] {
		if !r.meter.event(providerCompareEvent, 1) || !used {
			return errProviderIntegration
		}
	}
	f.current, f.setupOrdinal, f.highwater, m.root = derived, setup.ordinal, true, root
	if m.through == 1 && !m.scanState(1) {
		return errProviderIntegration
	}
	return nil
}

// The initial control contract selects setup H0 followed by E0. Compare the
// original receipt's nullable state and all three roots with this fresh model,
// rather than accepting hash-shaped roots or the producer's cached snapshot.
func (m *providerHistoryRecoveryInitialModel) scanState(index uint64) bool {
	scan, f, meter := m.prefix.scan, m.frontier, m.recovery.meter
	if len(scan.records) != 2 || scan.result.records != 2 || index > 1 {
		return false
	}
	v := scan.records[1-index] // Actual receipt chain was read backwards.
	wantStart, wantEnd := providerScanCursor{"setup", 0}, providerScanCursor{"main", 1}
	if index == 1 {
		wantStart, wantEnd = wantEnd, providerScanCursor{"done", 0}
	}
	if v.index != index || v.start != wantStart || v.end != wantEnd || !providerReplayTextEqual(v.state, f.current.replay, meter) {
		return false
	}
	for i, root := range [3]string{v.tickets, v.operations, v.segments} {
		if !providerReplayTextEqual(root, f.indexes.heads[i], meter) {
			return false
		}
	}
	return true
}

func (m *providerHistoryRecoveryInitialModel) exactObject(want providerIntegrationObject) error {
	raw, err := m.object(want.kind, want.digest)
	if err != nil || !providerReplayEqual(raw, providerReplayPhaseBytes(want.raw, m.recovery.meter), m.recovery.meter) {
		return errProviderIntegration
	}
	return nil
}

func (m *providerHistoryRecoveryInitialModel) setup(record providerHistoryRecoveryLegacyRecord) (providerSetupReference, error) {
	f := m.frontier
	for _, setup := range m.prefix.setups {
		if !m.recovery.meter.event(providerCompareEvent, uint64(len(setup.commit)+len(record.before.setup))) {
			return providerSetupReference{}, errProviderIntegration
		}
		if setup.commit != record.before.setup {
			continue
		}
		if setup.ordinal < f.setupOrdinal || !candidateLedgerClockAdvance(setup.clock, record.frontier.clock) {
			return providerSetupReference{}, errProviderIntegration
		}
		for ordinal := f.setupOrdinal + 1; ordinal <= setup.ordinal; ordinal++ {
			v := m.prefix.setups[ordinal]
			if v.ordinal != ordinal || v.expectedTip != record.before.commit || !candidateLedgerClockAdvance(v.clock, record.frontier.clock) ||
				m.through > 1 && !candidateLedgerClockAdvance(f.current.clock, v.clock) {
				return providerSetupReference{}, errProviderIntegration
			}
		}
		return setup, nil
	}
	return providerSetupReference{}, errProviderIntegration
}

func (m *providerHistoryRecoveryInitialModel) retention(record providerHistoryRecoveryLegacyRecord, setup providerSetupReference, cost providerFrontierRetention) (providerFrontierRetention, error) {
	r, p, f := m.recovery, m.prefix, m.frontier
	newHead, err := record.after.bytesObserved(r.config, r.meter)
	head, ok := providerHighwaterFileCost(uint64(len(newHead)), r.config.bounds)
	if err != nil || !ok {
		return providerFrontierRetention{}, errProviderIntegration
	}
	var priorBytes uint64
	if f.setupOrdinal < 0 {
		priorBytes, ok = providerUpperSum(p.anchorBytes, 2*p.highwaterOriginalBytes[0])
		if !ok {
			return providerFrontierRetention{}, errProviderIntegration
		}
	}
	for ordinal := f.setupOrdinal + 1; ordinal <= setup.ordinal; ordinal++ {
		priorBytes, ok = providerUpperSum(priorBytes, p.setupOriginalBytes[ordinal])
		if !ok {
			return providerFrontierRetention{}, errProviderIntegration
		}
	}
	for ordinal := max(f.setupOrdinal, 0) + 1; ordinal <= setup.ordinal; ordinal++ {
		index, sumOK := providerUpperSum(m.through-1, uint64(ordinal))
		if !sumOK || index >= uint64(len(p.highwaterOriginalBytes)) || p.highwaterOriginalBytes[index] == 0 {
			return providerFrontierRetention{}, errProviderIntegration
		}
		priorBytes, ok = providerUpperSum(priorBytes, 2*p.highwaterOriginalBytes[index])
		if !ok {
			return providerFrontierRetention{}, errProviderIntegration
		}
	}
	metadata, a := providerUpperSum(cost.metadata, head.metadata, priorBytes)
	stored, b := providerUpperSum(cost.stored, head.stored)
	if !a || !b || metadata > r.config.bounds.continuation.maximumMetadataBytes || stored > r.config.bounds.maximumStoredBytes {
		return providerFrontierRetention{}, errProviderIntegration
	}
	cost.metadata, cost.stored = metadata, stored
	return cost, nil
}

func (m *providerHistoryRecoveryInitialModel) deriveFrontier(record providerHistoryRecoveryLegacyRecord, root *CandidateEnrollmentRoot, delta *providerIndexDelta, commit []byte, setup providerSetupReference) (providerFrontierRecord, error) {
	r, replay := m.recovery, m.frontier
	p, policyOK := providerReplayJSON(providerReplayPhaseBytes(replay.body.policy.raw, r.meter), r.meter).(map[string]any)
	limits, limitsOK := p["limits"].(map[string]any)
	bootstrap, bootstrapOK := limits["bootstrap"].(map[string]any)
	duration, durationOK := providerCandidateUint(bootstrap["deadline_ms"], true)
	initial := r.initialDescriptor.attempt.clock
	if !policyOK || !limitsOK || !bootstrapOK || !durationOK || initial.WallMS < initial.UncertaintyMS {
		return providerFrontierRecord{}, errProviderIntegration
	}
	deadline, ok := providerUpperSum(uint64(initial.WallMS-initial.UncertaintyMS), duration)
	if !ok {
		return providerFrontierRecord{}, errProviderIntegration
	}
	f := providerFrontierRecord{identity: r.config.identity, fence: root.fence, sequence: m.through,
		commit: providerReplayDigest("sage-provider-custody-commit-v2", commit, r.meter), setup: setup.commit,
		objects: m.prefix.source.frames[m.through-1].inventory.digest, enrollment: root.digest, indexes: delta.previous,
		initial: initial, clock: record.frontier.clock, deadline: deadline}
	if m.through > 1 {
		raw, err := replay.current.bytesObserved(r.meter)
		if err != nil {
			return providerFrontierRecord{}, errProviderIntegration
		}
		f.previous = providerReplayDigest("sage-provider-custody-frontier-v1", raw, r.meter)
		f.seal, f.replay, f.checkpoint = replay.current.seal, replay.current.replay, replay.current.checkpoint
	}
	if replay.body.intent != nil {
		if replay.body.intent.deadline != deadline || replay.body.intent.clock != initial || replay.body.ledger == nil || !candidateLedgerClockAdvance(replay.body.ledger.current.clock, f.clock) {
			return providerFrontierRecord{}, errProviderIntegration
		}
		f.admission, f.budget = replay.body.intent.digest, replay.body.ledger.current.digest
	}
	if record.frontier.replay != f.replay || record.frontier.checkpoint != f.checkpoint {
		return providerFrontierRecord{}, errProviderIntegration
	}
	for _, mutation := range delta.mutations {
		f.indexes[mutation.chain] = mutation.object.digest
	}
	return f, nil
}
