package registry011

// Reconstructed original metadata, not an ACK, permission, current selection
// or recoverable G role. The caller supplies an independently retained prior
// G state and the immediate old main boundary from its own typed replay.
type providerGuardianClaimData struct {
	attempt  providerGuardianAttemptData
	head     providerGuardianHeadData
	metadata uint64
}

func providerGuardianClaimOriginals(s *providerCustodyStore, m *providerWorkMeter, before providerGuardianHeadData, main providerFrontierHighwater, admission, target, purpose string) (*providerGuardianClaimData, error) {
	if !s.valid() || s.config.bounds.guardian == nil || m == nil || m.closed || m.failed || s.activeMeter.Load() != m || before.ordinal >= s.config.bounds.guardian.attempts || !lowerHash(admission) || !lowerHash(target) || !providerGuardianPurpose(purpose) || !main.validObserved(s.config, m) {
		return nil, errProviderIntegration
	}
	x, err := newProviderScanCodec(s.config, m)
	if err != nil {
		return nil, errProviderIntegration
	}
	previous, err := x.guardianHead(before)
	prior, priorErr := s.read(s.custody, s.name("guardian-highwater", before.ordinal), candidateLedgerMetadataLimit)
	if err != nil || priorErr != nil || !x.equal(previous, prior) {
		return nil, errProviderIntegration
	}
	n := before.ordinal + 1
	intent, readErr := s.read(s.custody, s.name("guardian-intent", n), candidateLedgerMetadataLimit)
	v, parseErr := x.parseGuardianAttempt(intent)
	if readErr != nil || parseErr != nil || v.ordinal != n || v.origin != before.origin || v.previous != x.hash(providerGuardianHeadDomain, previous) ||
		v.main != main.commit || v.frontier != main.frontier || v.setup != main.setup || v.admission != admission || v.target != target || v.purpose != purpose ||
		v.clock != before.clock || v.deadline != before.deadline || before.ordinal != 0 && !candidateLedgerClockAdvance(before.lastStart, v.start) {
		return nil, errProviderIntegration
	}
	claim, err := x.guardianClaim(intent, v)
	commit, commitErr := x.guardianCommit(intent, claim, v)
	after := before
	after.ordinal, after.previous, after.admission, after.lastStart = n, v.previous, admission, v.start
	after.claim, after.commit = x.hash(providerGuardianClaimDomain, claim), x.hash(providerGuardianCommitDomain, commit)
	head, headErr := x.guardianHead(after)
	if err != nil || commitErr != nil || headErr != nil {
		return nil, errProviderIntegration
	}
	for _, row := range []struct {
		disk *retentionDisk
		name string
		raw  []byte
	}{
		{s.evidence, s.name("guardian-claim", n), claim},
		{s.custody, s.name("guardian-commit", n), commit},
		{s.custody, s.name("guardian-highwater", n), head},
	} {
		raw, err := s.read(row.disk, row.name, candidateLedgerMetadataLimit)
		if err != nil || !x.equal(raw, row.raw) {
			return nil, errProviderIntegration
		}
	}
	// HEAD replacement never refunds its old allocation. Its exact old byte
	// length is derived from the immutable same-content high-water original.
	metadata, ok := providerUpperSum(uint64(len(intent)), uint64(len(claim)), uint64(len(commit)), uint64(len(head)), uint64(len(head)))
	if !ok || metadata > s.config.bounds.guardian.claim.metadata || metadata > s.config.bounds.guardian.claim.raw {
		return nil, errProviderIntegration
	}
	return &providerGuardianClaimData{attempt: v, head: after, metadata: metadata}, nil
}
