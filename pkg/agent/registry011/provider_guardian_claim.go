package registry011

// startClaimLocked consumes a fresh slice of the ORIGINAL pre-H0 G hold.
// The caller holds store.mu. It cannot restore a failed or completed claim,
// import a producer pin, reopen R, or obtain a protected-work permit here.
// Target formatting happens only after this irreversible transition.
func (g *providerGuardianOwner) startClaimLocked(purpose string) error {
	if !g.selectedLocked() || !providerGuardianPurpose(purpose) || !g.initialized || !g.admitted || !g.ready || g.active ||
		g.claim != nil || g.meter != nil || g.first == nil || g.first.state != providerFirstOperationDone ||
		g.ordinal != g.pin.ordinal || g.ordinal >= g.bounds.attempts || g.integration == nil || g.indexes == nil || g.continuing != nil || g.sealing != nil || g.prefix.owner != g {
		return errProviderIntegration
	}
	s, c := g.store, g.integration
	if s.activeMeter.Load() != nil || s.frontierFiles == nil || s.frontierReplay == nil || s.frontierReplay.body == nil ||
		s.frontierFiles.pin != g.main || s.frontierFiles.pending != nil || s.sequence != g.main.sequence ||
		s.tip != g.main.commit || s.setupTip != g.main.setup || s.last != nil || c.store != s || c.ledger != g.indexes.ledger ||
		c.ledger == nil || c.intent == nil || c.root == nil || c.root.digest != s.enrollment ||
		s.frontierReplay.indexes == g.indexes || s.frontierReplay.body.ledger == c.ledger ||
		s.frontierReplay.body.ledger == nil || c.ledger.current == nil || s.frontierReplay.body.ledger.current == nil ||
		c.ledger.current.digest != s.frontierReplay.body.ledger.current.digest || c.ledger.continuation == nil ||
		c.ledger.continuation.pending != nil || c.ledger.continuation.final {
		return errProviderIntegration
	}
	next := g.ordinal + 1 // attempts <= 1<<30; this cannot overflow.
	if !g.claimRoomLocked(next) {
		return errProviderIntegration
	}
	// Only scalar checks preceded this debit. All maps, encodes, hashing,
	// clock samples and filesystem observations below spend this new role.
	g.ordinal, g.before, g.beforeRaw = next, g.pin, g.pinRaw
	g.active, g.ready, g.phase = true, false, providerGuardianMetadata
	g.claimPhysical, g.claimRaw = 0, 0
	g.claim = &providerGuardianAttemptData{origin: g.origin, purpose: purpose, main: g.main.commit, frontier: g.main.frontier, setup: g.main.setup, admission: c.intent.digest, ordinal: next, deadline: g.pin.deadline, clock: providerClockDiagnostic(g.original)}
	m := &providerWorkMeter{guardian: g, upper: g.bounds.claim.work, maximumClocks: g.bounds.claim.clocks}
	m.owner, g.meter = m, m
	s.activeMeter.Store(m)
	if _, err := g.sampleLocked(); err != nil {
		g.failLocked()
		return errProviderIntegration
	}
	return nil
}

// Preflight the cumulative fixed-vector debits and original backend capacity
// without constructing usage maps or taking an uncharged sample.
func (g *providerGuardianOwner) claimRoomLocked(next uint64) bool {
	if g == nil || g.clock == nil || next == 0 || next > g.bounds.attempts {
		return false
	}
	for i, n := range g.bounds.claim.work {
		used, ok := providerUpperProduct(n, next)
		if !ok || used > g.bounds.lifetime.work[i] {
			return false
		}
	}
	for _, pair := range [][2]uint64{
		{g.bounds.claim.retained, g.bounds.lifetime.retained}, {g.bounds.claim.live, g.bounds.lifetime.live},
		{g.bounds.claim.clocks, g.bounds.lifetime.clocks}, {g.bounds.claim.records, g.bounds.lifetime.records},
		{g.bounds.claim.objects, g.bounds.lifetime.objects}, {g.bounds.claim.raw, g.bounds.lifetime.raw},
		{g.bounds.claim.metadata, g.bounds.lifetime.metadata},
	} {
		used, ok := providerUpperProduct(pair[0], next)
		if !ok || used > pair[1] {
			return false
		}
	}
	clock := g.clock
	clock.mu.Lock()
	defer clock.mu.Unlock()
	return clock.owner == clock && !clock.failed && clock.id == g.original.ClockID && clock.epoch == g.original.Epoch &&
		clock.policy == g.policy.policy && candidateCapacity(0, clock.read, g.bounds.claim.clocks, clock.maximum)
}

// The target is a digest produced by the closed internal builder under THIS
// meter. A retained claim acknowledges metadata only; a later exact T and its
// composite admission binding still have to be independently retained.
func (g *providerGuardianOwner) retainClaimLocked(target string) error {
	if g == nil || !g.fits(g.meter) || g.phase != providerGuardianMetadata || !lowerHash(target) || g.claim.target != "" || g.claim.operation != "" {
		if g != nil && g.owner == g && g.store != nil && g.store.guardian == g && g.active {
			g.failLocked() // A consumed claim cannot recover from invalid state.
		}
		return errProviderIntegration
	}
	s, m := g.store, g.meter
	success := false
	defer func() {
		if !success {
			g.failLocked()
		}
	}()
	x, err := newProviderScanCodec(s.config, m)
	if err != nil {
		return errProviderIntegration
	}
	g.claim.previous = x.hash(providerGuardianHeadDomain, providerReplayBytes(g.beforeRaw, m))
	g.claim.target = target
	g.claim.operation, err = x.guardianOperation(*g.claim)
	intent, a := x.guardianAttempt(*g.claim)
	claim, b := x.guardianClaim(intent, *g.claim)
	commit, d := x.guardianCommit(intent, claim, *g.claim)
	after := g.before
	after.ordinal, after.previous, after.admission, after.lastStart = g.ordinal, g.claim.previous, g.claim.admission, g.claim.start
	after.claim, after.commit = x.hash(providerGuardianClaimDomain, claim), x.hash(providerGuardianCommitDomain, commit)
	head, e := x.guardianHead(after)
	physical, raw, ok := g.claimStorageLocked(intent, claim, commit, head)
	if err != nil || a != nil || b != nil || d != nil || e != nil || !ok || !g.checkSelectedOriginalsLocked(x) {
		return errProviderIntegration
	}
	// Both roots stay locked. Immutable originals are retained before the
	// high-water CAS. Pending copy and all readbacks remain in this G hold.
	for _, row := range []struct {
		disk *retentionDisk
		name string
		raw  []byte
	}{
		{s.custody, s.name("guardian-intent", g.ordinal), intent},
		{s.evidence, s.name("guardian-claim", g.ordinal), claim},
		{s.custody, s.name("guardian-commit", g.ordinal), commit},
		{s.custody, s.name("guardian-highwater", g.ordinal), head},
		{s.custody, s.name("GUARDIAN-HEAD.pending", g.ordinal), head},
	} {
		if s.write(row.disk, row.name, row.raw) != nil {
			return errProviderIntegration
		}
		original, readErr := s.read(row.disk, row.name, int64(s.config.bounds.maximumMetadataPerCommit))
		if readErr != nil || !x.equal(original, row.raw) {
			return errProviderIntegration
		}
	}
	if !g.checkSelectedOriginalsLocked(x) || g.renameHeadLocked(g.ordinal) != nil || !g.readExactLocked("GUARDIAN-HEAD", head) || !g.readExactLocked(s.name("guardian-highwater", g.ordinal), head) {
		return errProviderIntegration
	}
	text := providerReplayString(head, m)
	if text == "" || m.failed {
		return errProviderIntegration
	}
	g.pin, g.pinRaw, g.phase = after, text, providerGuardianRecorded
	g.physical += physical // The full G lifetime was held once before H0.
	g.raw += raw
	g.metadata += raw
	g.auxiliaryMetadata += raw
	g.claimPhysical, g.claimRaw = physical, raw
	s.retainedRaw += raw
	if _, err := g.sampleLocked(); err != nil {
		return errProviderIntegration
	}
	success = true
	return nil
}

func (g *providerGuardianOwner) checkSelectedOriginalsLocked(x *providerScanCodec) bool {
	s, f := g.store, g.store.frontierFiles
	if !g.fits(g.meter) || g.phase != providerGuardianMetadata || f == nil || f.pin != g.main ||
		!f.readExact("HEAD", providerReplayBytes(f.pinRaw, x.meter)) || !f.readExact(s.name("highwater", g.main.ordinal), providerReplayBytes(f.pinRaw, x.meter)) ||
		!g.readExactLocked("GUARDIAN-HEAD", providerReplayBytes(g.beforeRaw, x.meter)) || !g.readExactLocked(s.name("guardian-highwater", g.before.ordinal), providerReplayBytes(g.beforeRaw, x.meter)) {
		return false
	}
	h, raw, err := s.originalHighwaterObserved(g.main.ordinal, x.meter)
	return err == nil && h == g.main && x.equal(raw, providerReplayBytes(f.pinRaw, x.meter)) && x.hash(providerGuardianHeadDomain, providerReplayBytes(g.beforeRaw, x.meter)) == g.claim.previous
}

// All exact-byte and allocation sums are checked BEFORE the first write.
// Five files include the pending HEAD copy; replacement is never a refund.
func (g *providerGuardianOwner) claimStorageLocked(intent, claim, commit, head []byte) (uint64, uint64, bool) {
	var raw uint64
	for _, value := range [][]byte{intent, claim, commit, head, head} {
		if len(value) == 0 || uint64(len(value)) > g.store.config.bounds.maximumMetadataPerCommit || uint64(len(value)) > candidateLedgerMetadataLimit {
			return 0, 0, false
		}
		var ok bool
		raw, ok = providerUpperSum(raw, uint64(len(value)))
		if !ok {
			return 0, 0, false
		}
	}
	files, a := providerUpperProduct(5, g.store.config.bounds.fileOverheadBytes)
	physical, b := providerUpperSum(raw, files)
	allPhysical, c := providerUpperSum(g.physical, physical)
	allRaw, d := providerUpperSum(g.raw, raw)
	allMetadata, e := providerUpperSum(g.metadata, raw)
	storedRaw, f := providerUpperSum(g.store.retainedRaw, raw)
	return physical, raw, a && b && c && d && e && f && physical <= g.bounds.claim.retained && raw <= g.bounds.claim.raw &&
		raw <= g.bounds.claim.metadata && allPhysical <= g.bounds.lifetime.retained && allRaw <= g.bounds.lifetime.raw &&
		allMetadata <= g.bounds.lifetime.metadata && storedRaw <= g.store.stored && storedRaw <= g.store.config.bounds.continuation.maximumMetadataBytes
}
