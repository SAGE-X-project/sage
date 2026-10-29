package registry011

import "strconv"

// Only proposed DATA: no hold, clock sample, write or execution is created.
// Page/seal bytes were derived from the coordinator's pre-binding boundary.
func providerPageSealChangeData(o *providerPageSealOwner, page, seal, binding providerIntegrationObject, clock CandidateClockDiagnostic) (*providerFirstOperationChange, error) {
	if !o.selectedLocked() || o.change != nil || o.boundary == nil || o.boundary.final || !o.boundary.valid(o.boundary.owner) {
		return nil, errProviderIntegration
	}
	g := o.guardian
	s, c, m := g.store, g.integration, g.meter
	prior := s.frontierReplay.current
	if g.phase != providerGuardianRecorded || s.frontierFiles.pin != o.before || s.sequence != prior.sequence ||
		s.sequence == ^uint64(0) || c.root.sequence == ^uint64(0) || o.before.ordinal == ^uint64(0) ||
		s.sequence >= uint64(s.config.bounds.maximumCommits) || c.root.digest != s.enrollment ||
		prior.commit != s.tip || prior.objects != s.objectsRoot || prior.enrollment != c.root.digest ||
		prior.budget != c.ledger.current.digest || g.indexes.heads != prior.indexes ||
		!candidateLedgerClockAdvance(prior.clock, clock) || candidateLedgerExpired(clock, prior.deadline) ||
		page != o.boundary.pageObject || seal.kind != "ledger-seal" || seal.digest != o.binding.seal ||
		binding.kind != "guardian-seal-binding" || o.binding.page != page.digest {
		return nil, errProviderIntegration
	}
	expected, err := o.boundary.sealObject(g.indexes.heads)
	if err != nil || expected != seal {
		return nil, errProviderIntegration
	}
	value, ok := providerReplayJSON(providerReplayPhaseBytes(c.root.raw, m), m).(map[string]any)
	if !ok {
		return nil, errProviderIntegration
	}
	value["sequence"], value["previous_root"], value["budget_root"] = strconv.FormatUint(c.root.sequence+1, 10), c.root.digest, c.ledger.current.digest
	rootRaw := providerReplayCanonical(value, candidateLedgerMetadataLimit, m)
	root, err := parseCandidateEnrollmentRootObserved(s.frontierReplay.body.context, rootRaw, candidateLedgerMetadataLimit, m)
	rootObject, rootErr := newProviderIntegrationObjectObserved("enrollment", rootRaw, candidateLedgerMetadataLimit, m)
	if err != nil || rootErr != nil {
		return nil, errProviderIntegration
	}
	delta, err := g.indexes.prepare("seal", seal.digest, s.sequence+1, seal, root, c.ledger)
	if err != nil || len(delta.mutations) != 1 || delta.mutations[0].chain != 1 {
		return nil, errProviderIntegration
	}
	fresh := []providerIntegrationObject{page, seal, binding, rootObject, delta.mutations[0].object}
	if len(fresh) > s.config.bounds.maximumEntries || len(s.objects) > s.config.bounds.maximumObjects-len(fresh) {
		return nil, errProviderIntegration
	}
	inventory, err := newProviderFrontierInventoryObserved(s.objectsRoot, fresh, s.config.bounds.maximumEntries, min(s.config.bounds.maximumStoredBytes, 64<<20), m)
	previous, previousErr := prior.bytesObserved(m)
	link := providerFrontierLink{identity: s.config.identity, fence: root.fence, sequence: s.sequence, commit: s.tip, setup: s.setupTip, enrollment: c.root.digest, objects: s.objectsRoot, frontier: providerReplayDigest("sage-provider-custody-frontier-v1", previous, m)}
	intent, intentErr := providerFrontierIntentDiagnosticObserved(link, root, inventory, "seal", seal.digest, clock, s.config.bounds, m)
	commit := providerFrontierCommitObserved(intent, m)
	if err != nil || previousErr != nil || intentErr != nil || len(commit) == 0 {
		return nil, errProviderIntegration
	}
	frontier := prior
	frontier.sequence, frontier.previous, frontier.commit = s.sequence+1, link.frontier, providerReplayDigest("sage-provider-custody-commit-v2", commit, m)
	frontier.objects, frontier.enrollment = inventory.digest, root.digest
	frontier.indexes[1], frontier.seal, frontier.clock = delta.mutations[0].object.digest, seal.digest, clock
	raw, err := frontier.bytesObserved(m)
	after := o.before
	after.ordinal, after.sequence = after.ordinal+1, frontier.sequence
	after.previous = providerReplayDigest("sage-provider-custody-highwater-v1", providerReplayPhaseBytes(s.frontierFiles.pinRaw, m), m)
	after.commit, after.frontier = frontier.commit, providerReplayDigest("sage-provider-custody-frontier-v1", raw, m)
	if err != nil || !after.validObserved(s.config, m) || m.failed {
		return nil, errProviderIntegration
	}
	return &providerFirstOperationChange{root, delta, fresh, inventory, intent, commit, raw, after}, nil
}
