package registry011

// Immutable selected DATA for independent reconstruction of the initial legacy
// and scan prefix. Provenance pointers must never supply a live reader, cached
// catalog/model or renewed clock. Actual retained-byte replay is still required.
type providerHistoryRecoveryInitialDescriptor struct {
	owner                                                    *providerHistoryRecoveryInitialDescriptor
	recovery                                                 *providerHistoryRecoveryOwner
	checkpoint                                               *providerHistoryRecoveryCheckpoint
	start                                                    *providerHistoryProgramStart
	first                                                    *providerFirstAdmissionOwner
	main                                                     providerFrontierHighwater
	admission, ticket, budget, enrollment, objects, exchange string
	scan                                                     providerScanHighwater
	attempt                                                  providerScanAttempt
	scanMain                                                 providerFrontierHighwater
	scanMainRaw, targetRaw, lastChunk                        string
	records, setups, objectCount                             uint64
}

func (*providerHistoryRecoveryInitialDescriptor) UnmarshalJSON([]byte) error {
	return errProviderIntegration
}

// Called by the original healthy initializer before its final native sample.
// The two variable-length strings are copied with its original observer. All
// other DATA are value structs, bounded hashes or immutable small strings.
func (r *providerHistoryRecoveryOwner) prepareInitialDescriptorLocked(c *providerHistoryRecoveryCheckpoint) *providerHistoryRecoveryInitialDescriptor {
	if r == nil || c == nil || c.owner != c || c.recovery != r || c.group == nil || c.group != r.originalGroup || c.group.parent != r.parent ||
		c.group.phase != "initializing" || !c.group.activeLocked(r.originalMeter) || c.descriptor != nil || r.initialDescriptor != nil ||
		r.parent.start == nil || r.parent.firstAdmission == nil || r.store.config.bounds.scan == nil {
		return nil
	}
	s, start, first := r.store, r.parent.start, r.parent.firstAdmission
	if start != c.group.start || start.owner != start || start.parent != r.parent || first != s.firstAdmissionOwner || first.state != providerFirstAdmissionRetained ||
		first.body != s.scanBody || first.control == nil || first.control != s.scanControl || first.control.pin != first.scanPin ||
		first.control.prepared == nil || *first.control.prepared != first.attempt || first.mainPin != r.parent.firstMain ||
		first.mainRaw == "" || first.targetRaw == "" || uint64(len(first.mainRaw)) > s.config.bounds.maximumMetadataPerCommit || uint64(len(first.targetRaw)) > s.config.bounds.maximumMetadataPerCommit ||
		first.records == 0 || first.records > s.config.bounds.scan.maximumChunks || first.mainPin.ordinal != first.records-1 || !lowerHash(first.lastChunk) ||
		first.scanPin.ordinal == 0 || first.scanPin.ordinal != first.attempt.ordinal || first.scanPin.ordinal > s.config.bounds.scan.maximumAttempts ||
		len(s.setups) == 0 || len(s.setups) > s.config.bounds.maximumSetupRecoveries+1 || len(s.objects) > s.config.bounds.maximumObjects {
		return nil
	}
	n := &providerHistoryRecoveryInitialDescriptor{recovery: r, checkpoint: c, start: start, first: first, main: start.main,
		admission: start.admission, ticket: start.ticket, budget: start.budget, enrollment: start.enrollment, objects: start.objects, exchange: start.exchange,
		scan: first.scanPin, attempt: first.attempt, scanMain: first.mainPin, lastChunk: first.lastChunk, records: first.records,
		setups: uint64(len(s.setups)), objectCount: uint64(len(s.objects))}
	n.owner = n
	n.scanMainRaw = providerHistoryRecoveryCloneObserved(first.mainRaw, r.originalMeter)
	n.targetRaw = providerHistoryRecoveryCloneObserved(first.targetRaw, r.originalMeter)
	if n.scanMainRaw == "" || n.targetRaw == "" {
		return nil
	}
	return n
}

// Scalar provenance and finite shape only. In particular, do not compare with
// mutable first-body/cache DATA after failure or call a legacy active reader.
func (r *providerHistoryRecoveryOwner) initialDescriptorShapeLocked(g *providerHistoryGroupOwner) bool {
	if r == nil || r.initialDescriptor == nil || r.initial == nil || r.parent == nil || r.store == nil || g == nil || g != r.originalGroup || r.config.bounds.scan == nil {
		return false
	}
	n, b := r.initialDescriptor, r.config.bounds
	return n.owner == n && n.recovery == r && n.checkpoint == r.initial && r.initial.descriptor == n &&
		n.start != nil && n.start == g.start && n.start == r.parent.start && n.first != nil && n.first == r.parent.firstAdmission &&
		n.main == r.initial.pin && n.main == r.initial.scope.initial && n.main.sequence == 4 &&
		n.main.ordinal <= uint64(b.maximumCommits+b.maximumSetupRecoveries) &&
		lowerHash(n.admission) && lowerHash(n.ticket) && lowerHash(n.budget) && lowerHash(n.enrollment) && lowerHash(n.objects) && lowerHash(n.exchange) &&
		n.exchange == r.initial.scope.exchange && n.records > 0 && n.records <= b.scan.maximumChunks && n.scanMain.ordinal == n.records-1 &&
		n.scan.ordinal > 0 && n.scan.ordinal <= b.scan.maximumAttempts && n.attempt.ordinal == n.scan.ordinal && lowerHash(n.lastChunk) &&
		n.scanMainRaw != "" && uint64(len(n.scanMainRaw)) <= b.maximumMetadataPerCommit && n.targetRaw != "" && uint64(len(n.targetRaw)) <= b.maximumMetadataPerCommit &&
		n.setups > 0 && n.setups <= uint64(b.maximumSetupRecoveries+1) && n.objectCount <= uint64(b.maximumObjects)
}
