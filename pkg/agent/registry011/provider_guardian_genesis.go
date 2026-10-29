package registry011

import "os"

// The caller retains store.mu and the original O meter until both E0 and the
// separate guardian genesis have exact original readbacks. No atomic pair is
// assumed and a partial outcome permanently quarantines this live handle.
func (g *providerGuardianOwner) installGenesisLocked(o *providerBootstrapOwner) error {
	if !g.selectedLocked() || g.initialized || g.genesis != "" || g.origin == "" ||
		o == nil || o.store != g.store || g.store.bootstrapOwner != o || !o.started || !o.used || o.done || o.failed ||
		o.meter == nil || o.meter.closed || o.meter.failed || g.store.activeMeter.Load() != o.meter ||
		g.store.sequence != 1 || g.store.frontierFiles == nil || g.store.frontierReplay == nil || g.store.frontierFiles.pin.sequence != 1 ||
		g.store.frontierReplay.failed || !g.store.frontierReplay.highwater || g.store.frontierReplay.body == nil ||
		g.store.frontierReplay.body.anchor == nil || g.store.frontierReplay.body.intent != nil || g.store.frontierReplay.body.ledger != nil ||
		g.store.frontierReplay.body.anchor.digest != g.store.enrollment || g.store.frontierReplay.current.enrollment != g.store.enrollment ||
		len(g.store.objects) != 8 || g.store.frontierReplay.current.commit != g.store.tip || g.store.frontierFiles.pin.commit != g.store.tip {
		return errProviderIntegration
	}
	s, m := g.store, o.meter
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
	original, err := s.object("guardian-origin", g.origin)
	expected, originErr := x.guardianOrigin(g.h0, s.setupTip, providerClockDiagnostic(s.setupOrigin))
	deadline, _, clockOK := x.deadline(providerClockDiagnostic(s.setupOrigin))
	if err != nil || originErr != nil || !clockOK || !x.equal(original, expected) || x.hash(providerGuardianOriginDomain, original) != g.origin {
		return errProviderIntegration
	}
	value := providerGuardianHeadData{origin: g.origin, e0: s.tip, frontier: s.frontierFiles.pin.frontier, h0: g.h0, setup: s.setupTip, deadline: deadline, clock: providerClockDiagnostic(s.setupOrigin)}
	raw, err := x.guardianHead(value)
	allocation, a := providerUpperProduct(2, s.config.bounds.fileOverheadBytes)
	metadata, b := providerUpperProduct(2, uint64(len(raw)))
	physical, c := providerUpperSum(metadata, allocation)
	wholePhysical, wholeOK := providerUpperSum(o.physical, physical)
	wholeRaw, rawOK := providerUpperSum(s.retainedRaw, metadata)
	objects, objectsOK := providerUpperSum(uint64(len(s.objects)), 1)
	if err != nil || !a || !b || !c || o.physical < o.setupPhysical ||
		!wholeOK || !rawOK || !objectsOK || wholePhysical > s.config.bounds.setupRetained ||
		wholeRaw > s.config.bounds.maximumStoredBytes || wholeRaw > s.config.bounds.continuation.maximumMetadataBytes || objects > g.bounds.objects ||
		!candidateCapacity(0, o.physical-o.setupPhysical, physical, o.planned) {
		return errProviderIntegration
	}
	// The selected initial namespace has no guardian HEAD. This finite owned
	// directory check is not independent authenticated-absence qualification.
	if s.custody.checkIdentity() != nil || s.evidence.checkIdentity() != nil || !s.event(providerReadEvent, 0) {
		return errProviderIntegration
	}
	if _, err := s.custody.root.Lstat("GUARDIAN-HEAD"); !os.IsNotExist(err) {
		return errProviderIntegration
	}
	if s.custody.checkIdentity() != nil || s.evidence.checkIdentity() != nil {
		return errProviderIntegration
	}
	if s.write(s.custody, s.name("guardian-highwater", 0), raw) != nil ||
		s.write(s.custody, s.name("GUARDIAN-HEAD.pending", 0), raw) != nil ||
		g.renameHeadLocked(0) != nil || !g.readExactLocked("GUARDIAN-HEAD", raw) ||
		!g.readExactLocked(s.name("guardian-highwater", 0), raw) {
		return errProviderIntegration
	}
	retained := providerReplayString(raw, m)
	digest := x.hash(providerGuardianHeadDomain, raw)
	if retained == "" || !lowerHash(digest) || m.failed {
		return errProviderIntegration
	}
	g.pin, g.pinRaw, g.genesis, g.main = value, retained, digest, s.frontierFiles.pin
	g.genesisMetadata, g.initialized = metadata, true
	o.physical = wholePhysical // O was already held; no new s.stored charge.
	s.retainedRaw = wholeRaw
	success = true
	return nil
}

func (g *providerGuardianOwner) readExactLocked(name string, expected []byte) bool {
	s := g.store
	raw, err := s.read(s.custody, name, candidateLedgerMetadataLimit)
	return err == nil && providerReplayEqual(raw, expected, s.activeMeter.Load())
}

func (g *providerGuardianOwner) renameHeadLocked(ordinal uint64) error {
	s := g.store
	if s.custody.checkIdentity() != nil || s.evidence.checkIdentity() != nil ||
		!s.event(providerRenameEvent, 0) || s.custody.root.Rename(s.name("GUARDIAN-HEAD.pending", ordinal), "GUARDIAN-HEAD") != nil ||
		!s.event(providerSyncEvent, 0) || s.custody.sync() != nil ||
		s.custody.checkIdentity() != nil || s.evidence.checkIdentity() != nil {
		return errProviderIntegration
	}
	return nil
}
