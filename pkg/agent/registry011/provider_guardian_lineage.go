package registry011

// The pre-I E0 reader has no borrowed guardian owner. Derive origin/genesis
// from its own originals and close BOTH exact genesis copies in this target.
// Later histories use their own typed G chain, never substitute current HEAD
// for this immutable genesis or demand that a later HEAD still equal G0.
func (i *providerScanReplayReader) guardianInitialLineage() bool {
	if i == nil || i.view == nil || i.replay == nil || i.replay.body == nil || i.view.sequence != 1 || i.replay.body.intent != nil || i.view.config.bounds.guardian == nil {
		return false
	}
	v, m := i.view, i.meter
	data, err := providerGuardianInitialOriginals(v, m, v.tip, i.selected.frontier, i.selected.setup, providerClockDiagnostic(v.setupOrigin))
	x, codecErr := newProviderScanCodec(v.config, m)
	if err != nil || codecErr != nil {
		return false
	}
	_, expected, err := providerGuardianExpectedInitial(x, v.tip, i.selected.frontier, i.selected.setup, providerClockDiagnostic(v.setupOrigin))
	head, readErr := v.read(v.custody, "GUARDIAN-HEAD", candidateLedgerMetadataLimit)
	if err != nil || readErr != nil || !x.equal(head, expected) || x.hash(providerGuardianHeadDomain, head) != data.genesis {
		return false
	}
	if !i.guardianGenesis {
		raw, ok := providerUpperSum(v.retainedRaw, data.metadata)
		if !ok || raw > v.config.bounds.maximumStoredBytes || raw > v.config.bounds.continuation.maximumMetadataBytes {
			return false
		}
		v.retainedRaw, i.guardianGenesis = raw, true
	}
	return true
}
