package registry011

import "strconv"

// Build one closed continuing T/C/S delta under the NEW operation's current observer. The
// caller owns the coordinator ledger and keeps store.mu throughout. No write,
// observer transfer, hold or request dispatch occurs in this construction.
func providerContinuingChangeData(s *providerCustodyStore, before *CandidateEnrollmentRoot, ledger *CandidateAdmissionLedger, indexes *providerLedgerIndexes, prior providerFrontierRecord, kind, operation string, change *CandidateAdmissionLedgerChange, extra []providerIntegrationObjectInput, clock CandidateClockDiagnostic, meter *providerWorkMeter) (*providerFirstOperationChange, error) {
	if !s.valid() || s.config.bounds.firstAdmission == nil || s.frontierFiles == nil || s.frontierReplay == nil || s.frontierReplay.body == nil || meter == nil || meter.owner != meter || meter.closed || meter.failed || !providerContinuingChangeMeter(s, meter) || s.activeMeter.Load() != meter || before == nil || before.digest != s.enrollment || ledger == nil || !ledger.valid() || ledger.meter != meter || change == nil || ledger.current != change || indexes == nil || !indexes.valid() || indexes.meter != meter || indexes.ledger != ledger || indexes.heads != prior.indexes || prior.sequence != s.sequence || prior.commit != s.tip || prior.enrollment != before.digest || prior.objects != s.objectsRoot || prior.admission != ledger.intent.digest || prior.sequence < 2 || !candidateLedgerClockAdvance(prior.clock, clock) || candidateLedgerExpired(clock, prior.deadline) || !lowerHash(operation) || change.admission != ledger.intent.digest || s.frontierReplay.body.ledger == ledger || s.frontierReplay.indexes == indexes {
		return nil, errProviderIntegration
	}
	if !providerContinuingChangeInputs(kind, extra) {
		return nil, errProviderIntegration
	}
	if s.sequence == ^uint64(0) || before.sequence == ^uint64(0) || s.frontierFiles.pin.ordinal == ^uint64(0) || s.sequence >= uint64(s.config.bounds.maximumCommits) {
		return nil, errProviderIntegration
	}
	m, ok := providerReplayJSON(providerReplayPhaseBytes(before.raw, meter), meter).(map[string]any)
	if !ok {
		return nil, errProviderIntegration
	}
	m["sequence"], m["previous_root"], m["budget_root"] = strconv.FormatUint(before.sequence+1, 10), before.digest, change.digest
	rootRaw := providerReplayCanonical(m, candidateLedgerMetadataLimit, meter)
	root, err := parseCandidateEnrollmentRootObserved(s.frontierReplay.body.context, rootRaw, candidateLedgerMetadataLimit, meter)
	if err != nil {
		return nil, errProviderIntegration
	}
	fresh := make([]providerIntegrationObject, 0, len(extra)+6)
	for i, input := range extra {
		maximum := uint64(candidateLedgerMetadataLimit)
		if kind == "settle" && len(extra) == 5 && i < 3 {
			if input.kind != []string{"exchange", "response", "tls"}[i] {
				return nil, errProviderIntegration
			}
			maximum = s.config.bounds.maximumObjectBytes
		} else if kind == "settle" && len(extra) == 5 && i == 3 && input.kind != "wire-binding" {
			return nil, errProviderIntegration
		}
		if kind == "admit" && input.kind == "request" {
			maximum = 16 << 10
		}
		object, objectErr := newProviderIntegrationObjectObserved(input.kind, input.raw, maximum, meter)
		if objectErr != nil {
			return nil, errProviderIntegration
		}
		if kind == "admit" && (input.kind == "request" || input.kind == "params") {
			key, ok := providerReplayRefKey(providerIntegrationObjectRef{kind: object.kind, digest: object.digest}, meter)
			if !ok {
				return nil, errProviderIntegration
			}
			if _, exists := s.objects[key]; exists {
				original, readErr := s.object(object.kind, object.digest)
				if readErr != nil || !providerReplayEqual(original, input.raw, meter) {
					return nil, errProviderIntegration
				}
				continue // Immutable repeated request/params are re-read, not re-created.
			}
		}
		fresh = append(fresh, object)
	}
	inputObjects := len(fresh)
	entry := fresh[len(fresh)-1]
	if entry.digest != operation || len(change.nodes) != 1 {
		return nil, errProviderIntegration
	}
	for _, raw := range change.nodes {
		object, objectErr := newProviderIntegrationObjectObserved("ledger-node", providerReplayPhaseBytes(raw, meter), candidateLedgerMetadataLimit, meter)
		if objectErr != nil {
			return nil, errProviderIntegration
		}
		fresh = append(fresh, object)
	}
	for _, input := range []providerIntegrationObjectInput{{"budget", providerReplayPhaseBytes(change.raw, meter)}, {"enrollment", rootRaw}} {
		object, objectErr := newProviderIntegrationObjectObserved(input.kind, input.raw, candidateLedgerMetadataLimit, meter)
		if objectErr != nil {
			return nil, errProviderIntegration
		}
		fresh = append(fresh, object)
	}
	mutations := 3
	if kind == "consume" {
		mutations = 2 // Consumption changes the ticket and operation indexes only.
	}
	delta, err := indexes.prepare(kind, operation, s.sequence+1, entry, root, ledger)
	if err != nil || len(delta.mutations) != mutations {
		return nil, errProviderIntegration
	}
	heads := delta.previous
	for _, mutation := range delta.mutations {
		if len(mutation.object.raw) > candidateLedgerMetadataLimit {
			return nil, errProviderIntegration
		}
		fresh = append(fresh, mutation.object)
		heads[mutation.chain] = mutation.object.digest
	}
	if len(fresh) != inputObjects+3+mutations || len(fresh) > s.config.bounds.maximumEntries || len(s.objects) > s.config.bounds.maximumObjects-len(fresh) {
		return nil, errProviderIntegration
	}
	inventory, err := newProviderFrontierInventoryObserved(s.objectsRoot, fresh, s.config.bounds.maximumEntries, min(s.config.bounds.maximumStoredBytes, 64<<20), meter)
	previous, previousErr := prior.bytesObserved(meter)
	link := providerFrontierLink{identity: s.config.identity, fence: root.fence, sequence: s.sequence, commit: s.tip, setup: s.setupTip, enrollment: before.digest, objects: s.objectsRoot, frontier: providerReplayDigest("sage-provider-custody-frontier-v1", previous, meter)}
	intent, intentErr := providerFrontierIntentDiagnosticObserved(link, root, inventory, kind, operation, clock, s.config.bounds, meter)
	commit := providerFrontierCommitObserved(intent, meter)
	if err != nil || previousErr != nil || intentErr != nil || len(commit) == 0 {
		return nil, errProviderIntegration
	}
	frontier := prior
	frontier.sequence, frontier.previous, frontier.commit = s.sequence+1, link.frontier, providerReplayDigest("sage-provider-custody-commit-v2", commit, meter)
	frontier.objects, frontier.enrollment, frontier.indexes = inventory.digest, root.digest, heads
	frontier.budget, frontier.clock = change.digest, clock
	raw, err := frontier.bytesObserved(meter)
	after := s.frontierFiles.pin
	after.ordinal, after.sequence = after.ordinal+1, frontier.sequence
	after.previous = providerReplayDigest("sage-provider-custody-highwater-v1", providerReplayPhaseBytes(s.frontierFiles.pinRaw, meter), meter)
	after.commit, after.frontier = frontier.commit, providerReplayDigest("sage-provider-custody-frontier-v1", raw, meter)
	if err != nil || !after.validObserved(s.config, meter) || meter.failed {
		return nil, errProviderIntegration
	}
	return &providerFirstOperationChange{root, delta, fresh, inventory, intent, commit, raw, after}, nil
}

func providerContinuingChangeMeter(s *providerCustodyStore, m *providerWorkMeter) bool {
	if s == nil || m == nil || m.scan != nil || m.firstAdmission != nil || m.firstOperation != nil {
		return false
	}
	if m.guardian != nil {
		g := m.guardian
		return m.integration == nil && g.store == s && g.phase == providerGuardianRecorded && g.fits(m) &&
			g.continuing != nil && g.continuing.state == providerContinuingAdmission && g.continuing.selectedLocked()
	}
	return m.integration != nil && m.integration.store == s && m.continuing != nil && m.continuing.store == s && m.continuing.fits(m)
}

func providerContinuingChangeInputs(kind string, extra []providerIntegrationObjectInput) bool {
	switch kind {
	case "admit":
		if len(extra) == 2 {
			return extra[0].kind == "guardian-admission-binding" && extra[1].kind == "ticket"
		}
		if len(extra) == 3 {
			return extra[0].kind == "pin-input" && extra[1].kind == "guardian-admission-binding" && extra[2].kind == "ticket"
		}
		return len(extra) == 4 && extra[0].kind == "request" && extra[1].kind == "params" && extra[2].kind == "guardian-admission-binding" && extra[3].kind == "ticket"
	case "consume":
		return len(extra) == 1 && extra[0].kind == "consumption"
	case "settle":
		if len(extra) == 1 {
			return extra[0].kind == "settlement"
		}
		if len(extra) == 2 {
			return (extra[0].kind == "bootstrap" || extra[0].kind == "pin") && extra[1].kind == "settlement"
		}
		return len(extra) == 5 && extra[0].kind == "exchange" && extra[1].kind == "response" && extra[2].kind == "tls" && extra[3].kind == "wire-binding" && extra[4].kind == "settlement"
	}
	return false
}
