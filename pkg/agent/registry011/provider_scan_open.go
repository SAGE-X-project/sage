package registry011

import (
	"os"
	"strconv"
)

// Fresh V3 metadata open has its own fully observed finite O path. It never
// invokes unobserved V1/V2 setup serializers as a shortcut. Both roots and
// provisioned scan H0/HEAD0 are external candidate inputs, not created here.
// No returned private handle is a qualified root principal or execution ACK.
func openProviderScanCandidate(config providerCustodyConfig, clock *providerOwnedClock) (*providerCustodyStore, error) {
	id, b := config.identity, config.bounds
	if b.scan == nil || !b.valid() || !config.capture.valid() || !b.methods.fits(config.capture) || !providerScanSetupFits(config) || !b.scan.fits(providerScanOpenUpper(config), providerScanUpper{}) || clock == nil || clock.policy != id.Policy || config.attempt != 0 || config.expectedTip != config.anchor || config.expectedSetupTip != config.anchor || !lowerHash(config.anchor) || !lowerHash(config.expectedScanHead) || config.custodyDirectory == "" || config.evidenceDirectory == "" {
		return nil, ErrConfiguration
	}
	s := &providerCustodyStore{config: config, clock: clock, tip: config.anchor, stored: b.setupRetained, objects: make(map[string]providerIntegrationObjectRef), operations: make(map[string]bool), bindings: make(map[string]string)}
	s.owner = s
	setupOwner := &providerScanControl{store: s}
	setupOwner.owner = setupOwner
	meter := &providerWorkMeter{scan: setupOwner, upper: b.setupWork, maximumClocks: b.setupMaximumClockReads}
	meter.owner = meter
	s.activeMeter.Store(meter)
	defer s.activeMeter.Store(nil)
	x, err := newProviderScanCodec(config, meter)
	if err != nil || !s.firstAdmissionLimits(x) || !providerFirstAdmissionCapacity(s, providerScanOpenUpper(config)) {
		return nil, ErrConfiguration
	}
	// The single O is held before any original-root observation. Actual
	// durable root-operation admission remains a separate unqualified boundary.
	initial, err := s.sample()
	if err != nil {
		return nil, errProviderIntegration
	}
	s.setupOrigin = initial
	succeeded := false
	defer func() {
		if !succeeded {
			s.failed = true
			if s.evidence != nil {
				_ = s.evidence.close()
			}
			if s.custody != nil {
				_ = s.custody.close()
			}
		}
	}()
	s.custody, err = openRetentionDisk(config.custodyDirectory)
	if err != nil {
		return nil, errProviderIntegration
	}
	s.evidence, err = openRetentionDisk(config.evidenceDirectory)
	if err != nil || os.SameFile(s.custody.directoryInfo, s.evidence.directoryInfo) || os.SameFile(s.custody.lockInfo, s.evidence.lockInfo) {
		return nil, errProviderIntegration
	}
	for _, disk := range []*retentionDisk{s.custody, s.evidence} {
		names, e := providerDirectoryNamesObserved(disk, 4, meter)
		lock, lockErr := disk.lock.Stat()
		if e != nil || lockErr != nil || lock.Size() != 0 || len(names) != 4 || !names["ANCHOR"] || !names["AUTHORITY.LOCK"] || !names["SCAN-HEAD"] || !names[providerCustodyName("scan-highwater", 0)] {
			return nil, errProviderIntegration
		}
	}
	anchor, a := s.read(s.custody, "ANCHOR", int64(b.maximumMetadataPerCommit))
	local, l := s.read(s.evidence, "ANCHOR", int64(b.maximumMetadataPerCommit))
	emptyRaw, emptyErr := x.encode(map[string]any{"format": "sage-provider-custody-objects-v2", "previous": nil, "entries": []map[string]string{}})
	emptyDigest := x.hash("sage-provider-custody-objects-v2", emptyRaw)
	m := contentObject(x.decode(anchor), "format", "namespace", "instance_key", "profile", "initial_objects_root")
	wantAnchor, canonicalErr := x.encode(m)
	if a != nil || l != nil || emptyErr != nil || canonicalErr != nil || m == nil || !x.equal(anchor, local) || !x.equal(anchor, wantAnchor) || x.hash("sage-provider-custody-anchor-v1", anchor) != config.anchor || m["format"] != "sage-provider-custody-anchor-v1" || m["namespace"] != id.Namespace || m["instance_key"] != id.Instance || m["profile"] != config.profile || m["initial_objects_root"] != emptyDigest || !lowerHash(emptyDigest) {
		return nil, errProviderIntegration
	}
	controlBytes, err := providerScanInitialFiles(s)
	if err != nil {
		return nil, errProviderIntegration
	}
	s.objectsRoot = emptyDigest
	r, err := newProviderFrontierReplayObserved(config, meter)
	if err != nil {
		return nil, errProviderIntegration
	}
	s.frontierReplay = r
	claim, err := x.setup(providerClockDiagnostic(initial))
	if err != nil {
		return nil, errProviderIntegration
	}
	commit, err := x.setupCommit(claim)
	if err != nil {
		return nil, errProviderIntegration
	}
	// Five files will be written: setup intent/claim/commit, main H0 and HEAD.
	// Provisioned two anchors/locks and four scan originals stay charged too.
	files, filesOK := providerUpperProduct(13, b.fileOverheadBytes)
	upper, upperOK := providerUpperSum(uint64(2*len(anchor)+2*len(claim)+len(commit)), controlBytes, 2*candidateLedgerMetadataLimit, files)
	if !filesOK || !upperOK || upper > b.setupRetained || s.write(s.custody, providerCustodyName("setup-intent", 0), claim) != nil || s.write(s.evidence, providerCustodyName("setup-claim", 0), claim) != nil || s.write(s.custody, providerCustodyName("setup-commit", 0), commit) != nil {
		return nil, errProviderIntegration
	}
	checker := &providerScanControl{store: s}
	if !checker.readExact(s.custody, providerCustodyName("setup-intent", 0), claim) || !checker.readExact(s.evidence, providerCustodyName("setup-claim", 0), claim) || !checker.readExact(s.custody, providerCustodyName("setup-commit", 0), commit) {
		return nil, errProviderIntegration
	}
	s.setupTip = x.hash("sage-provider-authority-setup-commit-v1", commit)
	claimHash := x.hash("sage-provider-authority-setup-v1", claim)
	if !lowerHash(s.setupTip) || !lowerHash(claimHash) {
		return nil, errProviderIntegration
	}
	s.setupCommits = map[string]string{s.setupTip: claimHash}
	s.setups = []providerSetupReference{{commit: s.setupTip, claim: claimHash, expectedTip: config.anchor, clock: providerClockDiagnostic(initial)}}
	s.retainedRaw = uint64(2*len(anchor)+2*len(claim)+len(commit)) + controlBytes
	h := providerFrontierHighwater{identity: id, fence: 1, profile: config.profile, commit: config.anchor, setup: s.setupTip}
	head, err := x.encode(map[string]any{"format": "sage-provider-custody-highwater-v1", "namespace": id.Namespace, "instance_key": id.Instance, "generation": strconv.FormatUint(id.Generation, 10), "fence": "1", "ordinal": "0", "previous_highwater": nil, "profile": config.profile, "sequence": "0", "custody_commit": config.anchor, "setup_head": s.setupTip, "frontier": nil})
	if err != nil || s.write(s.custody, providerCustodyName("highwater", 0), head) != nil || s.write(s.custody, providerCustodyName("HEAD.pending", 0), head) != nil {
		return nil, errProviderIntegration
	}
	f := &providerFrontierFiles{store: s}
	f.owner = f
	s.frontierFiles = f
	if f.renameHead(0) != nil || !f.readExact("HEAD", head) || !f.readExact(providerCustodyName("highwater", 0), head) {
		return nil, errProviderIntegration
	}
	f.pin, f.pinRaw, f.initialized = h, string(head), true
	s.retainedRaw += uint64(2 * len(head))
	if _, err := newProviderScanControl(s); err != nil {
		return nil, errProviderIntegration
	}
	for _, entry := range []struct {
		disk     *retentionDisk
		required []string
	}{
		{s.custody, []string{"ANCHOR", "AUTHORITY.LOCK", "SCAN-HEAD", providerCustodyName("scan-highwater", 0), providerCustodyName("setup-intent", 0), providerCustodyName("setup-commit", 0), "HEAD", providerCustodyName("highwater", 0)}},
		{s.evidence, []string{"ANCHOR", "AUTHORITY.LOCK", "SCAN-HEAD", providerCustodyName("scan-highwater", 0), providerCustodyName("setup-claim", 0)}},
	} {
		names, e := providerDirectoryNamesObserved(entry.disk, len(entry.required), meter)
		if e != nil || len(names) != len(entry.required) {
			return nil, errProviderIntegration
		}
		for _, name := range entry.required {
			if !names[name] {
				return nil, errProviderIntegration
			}
		}
	}
	end, err := s.sample()
	if err != nil || !candidateLedgerClockAdvance(providerClockDiagnostic(initial), providerClockDiagnostic(end)) || uint64(end.MonotonicMS-initial.MonotonicMS) > b.setupMaximumDurationMS || meter.failed || s.retainedRaw > s.stored || s.custody.checkIdentity() != nil || s.evidence.checkIdentity() != nil {
		return nil, errProviderIntegration
	}
	succeeded = true
	return s, nil
}

func (x *providerScanCodec) setup(clock CandidateClockDiagnostic) ([]byte, error) {
	c, b, id := x.config, x.config.bounds, x.config.identity
	operation := x.hash("sage-provider-setup-operation-v1", []byte(c.anchor+id.Instance+id.Policy+id.SourceID))
	if !lowerHash(operation) {
		return nil, errProviderIntegration
	}
	return x.encode(map[string]any{"format": "sage-provider-authority-setup-v1", "namespace": id.Namespace, "instance_key": id.Instance, "policy": id.Policy, "source_id": id.SourceID, "generation": strconv.FormatUint(id.Generation, 10), "anchor": c.anchor, "expected_tip": c.expectedTip, "previous_setup_commit": c.expectedSetupTip, "operation": operation, "profile": c.profile, "attempt": strconv.Itoa(c.attempt), "clock": providerDiagnosticClockObject(clock), "work": candidateLedgerWorkObject(b.setupWork), "live_upper": strconv.FormatUint(b.setupLive, 10), "retained_upper": strconv.FormatUint(b.setupRetained, 10), "maximum_clock_reads": strconv.FormatUint(b.setupMaximumClockReads, 10), "maximum_duration_ms": strconv.FormatUint(b.setupMaximumDurationMS, 10)})
}

func (x *providerScanCodec) setupCommit(claim []byte) ([]byte, error) {
	m, ok := x.decode(claim).(map[string]any)
	if !ok || m["format"] != "sage-provider-authority-setup-v1" {
		return nil, errProviderIntegration
	}
	digest := x.hash("sage-provider-authority-setup-v1", claim)
	if !lowerHash(digest) {
		return nil, errProviderIntegration
	}
	return x.encode(map[string]any{"format": "sage-provider-authority-setup-commit-v1", "anchor": m["anchor"], "namespace": m["namespace"], "instance_key": m["instance_key"], "attempt": m["attempt"], "previous_setup_commit": m["previous_setup_commit"], "operation": m["operation"], "claim_digest": digest})
}
