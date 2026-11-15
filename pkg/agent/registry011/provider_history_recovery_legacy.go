package registry011

import "context"

// Original immutable strings are retained separately from returned walk DATA.
// Slot observations are logical work, not an allocator/live-RAM certificate.
type providerHistoryRecoveryInventoryFrame struct {
	owner     *providerHistoryRecoveryInventoryFrame
	walk      *providerHistoryRecoveryInventoryWalk
	sequence  uint64
	inventory *providerIntegrationInventory
	objects   []providerIntegrationObject
	complete  bool
}

type providerHistoryRecoveryLegacyRecord struct {
	before, after               providerFrontierHighwater
	root                        *CandidateEnrollmentRoot
	frontier                    providerFrontierRecord
	kind, operation             string
	intent, commit, rawFrontier string
}

// A fresh typed control/enrollment prefix, not a completed budget/response/
// index semantic model. The latter must be derived before recovery acceptance.
type providerHistoryRecoveryLegacyPrefix struct {
	owner    *providerHistoryRecoveryLegacyPrefix
	recovery *providerHistoryRecoveryOwner
	source   *providerHistoryRecoveryInventoryWalk
	scan     *providerHistoryRecoveryScanWalk
	pass     int
	pin      providerFrontierHighwater
	records  [4]providerHistoryRecoveryLegacyRecord
	setups   []providerSetupReference
	bytes    uint64
	// Exact lengths of originals validated by this pass. These are historical
	// logical ledger inputs, distinct from the recovery reader's input count.
	anchorBytes, setupBytesTotal uint64
	setupOriginalBytes           []uint64
	highwaterOriginalBytes       []uint64
}

func (*providerHistoryRecoveryLegacyPrefix) UnmarshalJSON([]byte) error {
	return errProviderIntegration
}

func (p *providerHistoryRecoveryLegacyPrefix) selectedLocked(r *providerHistoryRecoveryOwner) bool {
	return p != nil && p.owner == p && r != nil && p.recovery == r && p.pass >= 0 && p.pass < len(r.legacyPrefixes) && r.legacyPrefixes[p.pass] == p &&
		p.source != nil && p.source == r.inventoryWalks[p.pass] && p.source.framesReadyLocked(r) && p.scan != nil && p.scan == r.scanWalks[p.pass] &&
		p.scan.selectedLocked(r) && p.scan.complete && p.scan.source == p.source && p.pin == r.initialDescriptor.main && p.records[3].root != nil &&
		uint64(len(p.setups)) == r.initialDescriptor.setups && len(p.setupOriginalBytes) == len(p.setups) &&
		uint64(len(p.highwaterOriginalBytes)) == p.pin.ordinal+1 && p.anchorBytes != 0 && p.setupBytesTotal != 0
}

func (w *providerHistoryRecoveryInventoryWalk) framesReadyLocked(r *providerHistoryRecoveryOwner) bool {
	if !w.selectedLocked(r) || !w.complete || w.current != nil || w.remaining != 0 {
		return false
	}
	var count uint64
	for i, f := range w.frames {
		if f == nil || f.owner != f || f.walk != w || f.sequence != uint64(i+1) || !f.complete || f.inventory == nil || len(f.objects) == 0 ||
			len(f.objects) != len(f.inventory.entries) || cap(f.objects) != len(f.inventory.entries) || len(f.objects) > r.config.bounds.maximumEntries {
			return false
		}
		var ok bool
		count, ok = providerUpperSum(count, uint64(len(f.objects)))
		if !ok || count > w.descriptor.objectCount {
			return false
		}
	}
	return count == w.descriptor.objectCount
}

// This is bounded immutable DATA lookup over the actual replay's own frames.
// It does not call a failed store reader, producer map or caller callback.
func (w *providerHistoryRecoveryInventoryWalk) frameObjectObserved(kind, digest string, through uint64) []byte {
	r := w.recovery
	if !w.framesReadyLocked(r) || through == 0 || through > uint64(len(w.frames)) || !lowerHash(digest) {
		return nil
	}
	for i := uint64(0); i < through; i++ {
		for _, object := range w.frames[i].objects {
			if !r.meter.event(providerCompareEvent, uint64(len(object.kind)+len(kind))) {
				return nil
			}
			if object.kind != kind {
				continue
			}
			if !r.meter.event(providerCompareEvent, uint64(len(object.digest)+len(digest))) {
				return nil
			}
			if object.digest == digest {
				return providerReplayPhaseBytes(object.raw, r.meter)
			}
		}
	}
	return nil
}

func (p *providerHistoryRecoveryLegacyPrefix) readLocked(ctx context.Context, kind providerHistoryRecoveryFileKind, ordinal uint64) ([]byte, error) {
	r := p.recovery
	out, err := r.readSelectedInputLocked(ctx, providerHistoryRecoveryInputSelection{checkpoint: r.initial, kind: kind, index: int(ordinal)})
	if err != nil || out == nil || out.state != providerHistoryRecoveryWholeInput {
		return nil, errProviderIntegration
	}
	bytes, ok := providerUpperSum(p.bytes, uint64(len(out.raw)))
	if !ok || bytes > r.config.bounds.maximumStoredBytes {
		return nil, errProviderIntegration
	}
	p.bytes = bytes
	return out.raw, nil
}

func (r *providerHistoryRecoveryOwner) replayLegacyPrefixCandidate(ctx context.Context, pass int) (*providerHistoryRecoveryLegacyPrefix, error) {
	if r == nil || r.store == nil {
		return nil, errProviderIntegration
	}
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if !r.activeLocked(r.meter) || pass < 0 || pass >= len(r.legacyPrefixes) || r.legacyPrefixes[pass] != nil {
		return nil, errProviderIntegration
	}
	source, scan := r.inventoryWalks[pass], r.scanWalks[pass]
	if source == nil || !source.framesReadyLocked(r) || scan == nil || !scan.selectedLocked(r) || !scan.complete || scan.source != source ||
		pass == 1 && (r.legacyPrefixes[0] == nil || !r.legacyPrefixes[0].selectedLocked(r)) {
		return nil, errProviderIntegration
	}
	accepted := false
	defer func() {
		if !accepted {
			r.stopLocked(true)
		}
	}()
	slots, slotsOK := providerUpperSum(2*r.initialDescriptor.setups, r.initialDescriptor.main.ordinal, 2)
	if !r.inputReadyLocked(ctx) || !slotsOK || !r.meter.event(providerCopyEvent, slots) {
		return nil, errProviderIntegration
	}
	p := &providerHistoryRecoveryLegacyPrefix{recovery: r, source: source, scan: scan, pass: pass, setups: make([]providerSetupReference, 0, int(r.initialDescriptor.setups)),
		setupOriginalBytes: make([]uint64, int(r.initialDescriptor.setups)), highwaterOriginalBytes: make([]uint64, int(r.initialDescriptor.main.ordinal+1))}
	p.owner = p
	empty, err := newProviderFrontierInventoryObserved("", nil, r.config.bounds.maximumEntries, min(r.config.bounds.maximumStoredBytes, 64<<20), r.meter)
	if err != nil || empty == nil {
		return nil, errProviderIntegration
	}
	anchor, a := p.readLocked(ctx, providerHistoryRecoveryAnchorCustody, 0)
	local, b := p.readLocked(ctx, providerHistoryRecoveryAnchorEvidence, 0)
	expectedAnchor := providerReplayCanonical(map[string]any{"format": "sage-provider-custody-anchor-v1", "namespace": r.config.identity.Namespace,
		"instance_key": r.config.identity.Instance, "profile": r.config.profile, "initial_objects_root": empty.digest}, r.config.bounds.maximumMetadataPerCommit, r.meter)
	if a != nil || b != nil || !providerReplayEqual(anchor, local, r.meter) || !providerReplayEqual(anchor, expectedAnchor, r.meter) ||
		!providerReplayTextEqual(providerReplayDigest("sage-provider-custody-anchor-v1", anchor, r.meter), r.config.anchor, r.meter) {
		return nil, errProviderIntegration
	}
	p.anchorBytes = uint64(len(anchor) + len(local))
	setup, err := p.setupLocked(ctx, 0, r.config.anchor, r.config.anchor)
	if err != nil || setup.clock != r.initialDescriptor.attempt.clock {
		return nil, errProviderIntegration
	}
	if !r.meter.event(providerCopyEvent, 1) {
		return nil, errProviderIntegration
	}
	p.setups = append(p.setups, setup)
	raw, err := p.readLocked(ctx, providerHistoryRecoveryLegacyHighwater, 0)
	h, parseErr := parseProviderFrontierHighwaterObserved(raw, r.config, r.meter)
	initial := providerFrontierHighwater{identity: r.config.identity, fence: 1, profile: r.config.profile, commit: r.config.anchor, setup: setup.commit}
	if err != nil || parseErr != nil || h != initial {
		return nil, errProviderIntegration
	}
	p.highwaterOriginalBytes[0] = uint64(len(raw))
	objectsRoot, enrollment := empty.digest, ""
	lastClock := setup.clock
	for ordinal := uint64(1); ordinal <= r.initialDescriptor.main.ordinal; ordinal++ {
		raw, err = p.readLocked(ctx, providerHistoryRecoveryLegacyHighwater, ordinal)
		after, parseErr := parseProviderFrontierHighwaterObserved(raw, r.config, r.meter)
		if err != nil || parseErr != nil || after.ordinal != ordinal {
			return nil, errProviderIntegration
		}
		p.highwaterOriginalBytes[ordinal] = uint64(len(raw))
		if after.sequence == h.sequence {
			index := uint64(len(p.setups))
			if index >= r.initialDescriptor.setups {
				return nil, errProviderIntegration
			}
			setup, claim, commit, err := p.setupOriginalsLocked(ctx, index, h.commit, h.setup)
			if err != nil || !checkProviderHighwaterSetupObserved(h, after, r.config, claim, commit, r.meter) || !candidateLedgerClockAdvance(lastClock, setup.clock) {
				return nil, errProviderIntegration
			}
			if !r.meter.event(providerCopyEvent, 1) {
				return nil, errProviderIntegration
			}
			p.setups, lastClock = append(p.setups, setup), setup.clock
		} else {
			if after.sequence != h.sequence+1 || after.sequence == 0 || after.sequence > uint64(len(p.records)) {
				return nil, errProviderIntegration
			}
			record, err := p.mainLocked(ctx, h, after, objectsRoot, enrollment)
			if err != nil || !candidateLedgerClockAdvance(lastClock, record.frontier.clock) || record.frontier.initial != r.initialDescriptor.attempt.clock {
				return nil, errProviderIntegration
			}
			if !r.meter.event(providerCopyEvent, 1) {
				return nil, errProviderIntegration
			}
			p.records[after.sequence-1] = record
			objectsRoot, enrollment, lastClock = record.frontier.objects, record.root.digest, record.frontier.clock
		}
		h = after
	}
	n, final := r.initialDescriptor, p.records[3].frontier
	if h != n.main || uint64(len(p.setups)) != n.setups || p.records[3].root == nil {
		return nil, errProviderIntegration
	}
	for _, pair := range [][2]string{{objectsRoot, n.objects}, {enrollment, n.enrollment}, {final.admission, n.admission}, {final.budget, n.budget}} {
		if !providerReplayTextEqual(pair[0], pair[1], r.meter) {
			return nil, errProviderIntegration
		}
	}
	if !r.inputReadyLocked(ctx) {
		return nil, errProviderIntegration
	}
	p.pin, r.legacyPrefixes[pass], accepted = h, p, true
	return p, nil
}

func (p *providerHistoryRecoveryLegacyPrefix) setupLocked(ctx context.Context, ordinal uint64, tip, previous string) (providerSetupReference, error) {
	s, _, _, err := p.setupOriginalsLocked(ctx, ordinal, tip, previous)
	return s, err
}

func (p *providerHistoryRecoveryLegacyPrefix) setupOriginalsLocked(ctx context.Context, ordinal uint64, tip, previous string) (providerSetupReference, []byte, []byte, error) {
	var zero providerSetupReference
	r := p.recovery
	claim, a := p.readLocked(ctx, providerHistoryRecoverySetupClaim, ordinal)
	intent, b := p.readLocked(ctx, providerHistoryRecoverySetupIntent, ordinal)
	commit, c := p.readLocked(ctx, providerHistoryRecoverySetupCommit, ordinal)
	fields, ok := providerReplayJSON(claim, r.meter).(map[string]any)
	clock, valid := providerStoredClock(fields["clock"])
	copy := r.config
	copy.attempt, copy.expectedTip, copy.expectedSetupTip = int(ordinal), tip, previous
	expected := providerCustodySetupDiagnosticObserved(copy, clock, r.meter)
	if a != nil || b != nil || c != nil || !ok || !valid || !providerReplayEqual(claim, intent, r.meter) || !providerReplayEqual(claim, expected, r.meter) || !providerReplayEqual(commit, providerSetupCommitObserved(claim, r.meter), r.meter) {
		return zero, nil, nil, errProviderIntegration
	}
	commitDigest, claimDigest := providerReplayDigest("sage-provider-authority-setup-commit-v1", commit, r.meter), providerReplayDigest("sage-provider-authority-setup-v1", claim, r.meter)
	if !lowerHash(commitDigest) || !lowerHash(claimDigest) {
		return zero, nil, nil, errProviderIntegration
	}
	if ordinal >= uint64(len(p.setupOriginalBytes)) || p.setupOriginalBytes[ordinal] != 0 {
		return zero, nil, nil, errProviderIntegration
	}
	count := uint64(len(claim) + len(intent) + len(commit))
	total, totalOK := providerUpperSum(p.setupBytesTotal, count)
	if !totalOK || total > r.config.bounds.continuation.maximumMetadataBytes || !r.inputReadyLocked(ctx) {
		return zero, nil, nil, errProviderIntegration
	}
	p.setupOriginalBytes[ordinal], p.setupBytesTotal = count, total
	return providerSetupReference{commit: commitDigest, claim: claimDigest, expectedTip: tip, ordinal: int(ordinal), clock: clock}, claim, commit, nil
}

func (p *providerHistoryRecoveryLegacyPrefix) mainLocked(ctx context.Context, before, after providerFrontierHighwater, objects, enrollment string) (providerHistoryRecoveryLegacyRecord, error) {
	var zero providerHistoryRecoveryLegacyRecord
	r, sequence := p.recovery, after.sequence
	if sequence == 0 || sequence > uint64(len(p.records)) {
		return zero, errProviderIntegration
	}
	intent, a := p.readLocked(ctx, providerHistoryRecoveryLegacyIntent, sequence)
	commit, b := p.readLocked(ctx, providerHistoryRecoveryLegacyCommit, sequence)
	frontier, c := p.readLocked(ctx, providerHistoryRecoveryLegacyFrontier, sequence)
	pending, d := p.readLocked(ctx, providerHistoryRecoveryLegacyPending, sequence)
	if a != nil || b != nil || c != nil || d != nil || !providerReplayEqual(intent, pending, r.meter) || !checkProviderHighwaterMainObserved(before, after, r.config, intent, commit, frontier, r.meter) {
		return zero, errProviderIntegration
	}
	m := providerFrontierIntentObjectObserved(intent, r.meter)
	f, err := parseProviderFrontierRecordObserved(frontier, r.config.identity, r.meter)
	rootRaw := p.source.frameObjectObserved("enrollment", contentString(m["new_enrollment_root"]), sequence)
	root, rootErr := parseCandidateEnrollmentRootObserved(&ProviderCandidateContext{identity: r.config.identity, bound: true}, rootRaw, 64<<10, r.meter)
	frame := p.source.frames[sequence-1]
	link := providerFrontierLink{identity: r.config.identity, fence: before.fence, sequence: before.sequence, commit: before.commit, frontier: before.frontier, setup: before.setup, enrollment: enrollment, objects: objects}
	kind, operation := contentString(m["kind"]), contentString(m["operation"])
	wantKind := [4]string{"enroll", "admit", "consume", "settle"}[sequence-1]
	if m == nil || err != nil || rootErr != nil || kind != wantKind || frame == nil || !checkProviderFrontierIntentObserved(intent, link, root, frame.inventory, kind, operation, r.config.bounds, r.meter) {
		return zero, errProviderIntegration
	}
	if sequence == 1 {
		joined := providerReplayJoin([]string{r.config.anchor, r.config.identity.Instance, r.config.identity.Policy, r.config.identity.SourceID}, r.meter)
		if joined == "" || !providerReplayTextEqual(operation, providerReplayDigest("sage-provider-enroll-operation-v1", providerReplayPhaseBytes(joined, r.meter), r.meter), r.meter) {
			return zero, errProviderIntegration
		}
	} else {
		objectKind := [4]string{"", "ticket", "consumption", "settlement"}[sequence-1]
		if len(p.source.frameObjectObserved(objectKind, operation, sequence)) == 0 {
			return zero, errProviderIntegration
		}
	}
	i, k, v := providerReplayString(intent, r.meter), providerReplayString(commit, r.meter), providerReplayString(frontier, r.meter)
	if i == "" || k == "" || v == "" {
		return zero, errProviderIntegration
	}
	return providerHistoryRecoveryLegacyRecord{before: before, after: after, root: root, frontier: f, kind: kind, operation: operation, intent: i, commit: k, rawFrontier: v}, nil
}
