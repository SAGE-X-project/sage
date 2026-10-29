package registry011

import (
	"bytes"
	"os"
)

// This is the concrete filesystem portion of the CANDIDATE protocol only. Its
// private pin is not a selected qualified retaining principal or a work ACK.
// The actual principal/factory must first establish the independently selected
// anchor, complete prefix, typed maps and qualified host/FS/clock/claim profile.
// Every method below returns metadata/error only, never protected-use authority.
type providerFrontierFiles struct {
	owner               *providerFrontierFiles
	store               *providerCustodyStore
	pin                 providerFrontierHighwater
	pinRaw              string
	pending             *providerFrontierWriteClaim
	historyPending      *providerHistoryWriteClaim
	initialized, failed bool
}

func (*providerFrontierFiles) UnmarshalJSON([]byte) error { return errProviderIntegration }

func newProviderFrontierFiles(s *providerCustodyStore) (*providerFrontierFiles, error) {
	if s == nil {
		return nil, errProviderIntegration
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.owner != s || s.frontierFiles != nil || s.closed || s.failed || s.custody == nil || s.evidence == nil || s.config.bounds.continuation == nil || !s.config.bounds.valid() || !s.config.capture.valid() || !s.config.bounds.methods.fits(s.config.capture) || s.config.profile != providerIntegrationProfile(s.config.bounds, s.config.capture) || s.stored > s.config.bounds.maximumStoredBytes || s.retainedRaw > s.stored || s.custody.checkIdentity() != nil || s.evidence.checkIdentity() != nil || os.SameFile(s.custody.directoryInfo, s.evidence.directoryInfo) || os.SameFile(s.custody.lockInfo, s.evidence.lockInfo) {
		return nil, errProviderIntegration
	}
	f := &providerFrontierFiles{store: s}
	f.owner = f
	s.frontierFiles = f
	return f, nil
}

func (f *providerFrontierFiles) valid() bool {
	// The caller holds store.mu. File state, store counters, close and the
	// permanent failure latch all share this one serialization boundary.
	return f != nil && f.owner == f && !f.failed && f.store != nil && f.store.owner == f.store && f.store.frontierFiles == f && !f.store.closed && !f.store.failed && f.store.custody != nil && f.store.evidence != nil
}

// Initial H0 is derived only after the caller's authenticated empty-state
// setup0. Exact original anchor/setup bytes and the complete small empty main
// directory view are rechecked here. This bounded view is not itself that
// independently selected principal or authenticated-absence qualification.
func (f *providerFrontierFiles) initialize() (providerFrontierRetention, error) {
	if f == nil || f.store == nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	f.store.mu.Lock()
	defer f.store.mu.Unlock()
	return f.initializeLocked()
}

// initializeLocked is also the outer store transaction's entry point. The
// caller must hold store.mu for the complete transaction, including this step.
func (f *providerFrontierFiles) initializeLocked() (providerFrontierRetention, error) {
	if !f.valid() || f.initialized || f.historyPending != nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	f.failed = true
	succeeded := false
	defer func() {
		if !succeeded {
			f.store.failed = true
		}
	}()
	s, config := f.store, f.store.config
	if config.attempt != 0 || s.sequence != 0 || s.tip != config.anchor || s.enrollment != "" || config.expectedTip != config.anchor || config.expectedSetupTip != config.anchor || len(s.setups) != 1 || s.setups[0].ordinal != 0 || s.setups[0].expectedTip != config.anchor || s.setupTip != s.setups[0].commit || s.stored != config.bounds.setupRetained {
		return providerFrontierRetention{}, errProviderIntegration
	}
	if f.checkInitialPrefix() != nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	h := providerFrontierHighwater{identity: config.identity, fence: 1, profile: config.profile, commit: config.anchor, setup: s.setupTip}
	raw, err := h.bytes(config)
	cost, costOK := providerHighwaterFileCost(uint64(len(raw)), config.bounds)
	if err != nil || !costOK || !s.event(providerEncodeEvent, uint64(len(raw))) {
		return providerFrontierRetention{}, errProviderIntegration
	}
	// The setup reservation was charged before opening/validating this view.
	// It includes H0's two copies, and is not charged as a new main operation.
	upper, ok := providerUpperSum(s.retainedRaw, cost.metadata)
	files, mulOK := providerUpperProduct(9, config.bounds.fileOverheadBytes)
	upper, sumOK := providerUpperSum(upper, files)
	if !ok || !mulOK || !sumOK || upper > config.bounds.setupRetained {
		return providerFrontierRetention{}, errProviderIntegration
	}
	if s.write(s.custody, s.name("highwater", 0), raw) != nil || s.write(s.custody, s.name("HEAD.pending", 0), raw) != nil || f.renameHead(0) != nil || !f.readExact("HEAD", raw) || !f.readExact(s.name("highwater", 0), raw) {
		return providerFrontierRetention{}, errProviderIntegration
	}
	f.pin, f.pinRaw, f.initialized, f.failed = h, string(raw), true, false
	s.retainedRaw += cost.metadata // Full upper was checked before writes.
	succeeded = true
	return cost, nil
}

// Main/setup relation checks do NOT replace the separate complete typed delta
// replay. The caller must already have independently validated and retained the
// new original commit/frontier or setup prefix before invoking this file step.
// Unknown rename/readback state permanently poisons this live candidate handle.
func (f *providerFrontierFiles) advance(after providerFrontierHighwater, kind string, input, commit, frontier []byte) (providerFrontierRetention, error) {
	if f == nil || f.store == nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	f.store.mu.Lock()
	defer f.store.mu.Unlock()
	return f.advanceLocked(after, kind, input, commit, frontier)
}

// advanceLocked shares the outer store transaction's mutex, so close and any
// other store writer cannot interleave with its comparison/write/readback.
func (f *providerFrontierFiles) advanceLocked(after providerFrontierHighwater, kind string, input, commit, frontier []byte) (providerFrontierRetention, error) {
	return f.advanceClaimedLocked(after, kind, input, commit, frontier, nil)
}

func (f *providerFrontierFiles) advanceClaimedLocked(after providerFrontierHighwater, kind string, input, commit, frontier []byte, claim *providerFrontierWriteClaim) (providerFrontierRetention, error) {
	if !f.valid() || !f.initialized || f.pending != claim || f.historyPending != nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	if f.store.config.bounds.history != nil && claim == nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	f.failed = true
	succeeded := false
	defer func() {
		if !succeeded {
			f.store.failed = true
		}
	}()
	s := f.store
	meter := s.activeMeter.Load()
	valid := false
	switch kind {
	case "main":
		valid = checkProviderHighwaterMainObserved(f.pin, after, s.config, input, commit, frontier, meter)
	case "setup":
		valid = len(frontier) == 0 && checkProviderHighwaterSetupObserved(f.pin, after, s.config, input, commit, meter)
	}
	raw, err := after.bytesObserved(s.config, meter)
	cost, ok := providerHighwaterFileCost(uint64(len(raw)), s.config.bounds)
	retained, retainedOK := providerUpperSum(s.retainedRaw, cost.metadata)
	prepaid := claim != nil && claim.owner == f && !claim.used && claim.stored == s.stored && claim.headCost == cost && claim.kind == kind
	if prepaid {
		if s.config.bounds.firstAdmission != nil {
			prepaid = meter != nil && providerReplayPhaseTextEqual(claim.before, f.pinRaw, meter) && providerReplayPhaseRawTextEqual(raw, claim.after, meter) && providerReplayPhaseRawTextEqual(input, claim.intent, meter) && providerReplayPhaseRawTextEqual(commit, claim.commit, meter) && providerReplayPhaseRawTextEqual(frontier, claim.frontier, meter)
		} else {
			prepaid = claim.before == f.pinRaw && claim.after == string(raw) && claim.intent == string(input) && claim.commit == string(commit) && claim.frontier == string(frontier)
		}
	}
	if !valid || err != nil || !ok || !retainedOK || claim != nil && !prepaid || !f.readExact("HEAD", providerReplayBytes(f.pinRaw, meter)) || !f.readExact(s.name("highwater", f.pin.ordinal), providerReplayBytes(f.pinRaw, meter)) || !prepaid && !candidateCapacity(0, s.stored, cost.stored, s.config.bounds.maximumStoredBytes) {
		return providerFrontierRetention{}, errProviderIntegration
	}
	// Retain the full allocation charge after any uncertain write. Neither
	// rename nor replacement of the old HEAD refunds earlier lifetime charges.
	if prepaid {
		claim.used = true
	} else {
		s.stored += cost.stored
	}
	if s.write(s.custody, s.name("highwater", after.ordinal), raw) != nil || s.write(s.custody, s.name("HEAD.pending", after.ordinal), raw) != nil || !f.readExact("HEAD", providerReplayBytes(f.pinRaw, meter)) || f.renameHead(after.ordinal) != nil || !f.readExact("HEAD", raw) || !f.readExact(s.name("highwater", after.ordinal), raw) {
		return providerFrontierRetention{}, errProviderIntegration
	}
	retainedRaw := providerReplayString(raw, meter)
	if retainedRaw == "" {
		return providerFrontierRetention{}, errProviderIntegration
	}
	f.pin, f.pinRaw, f.failed = after, retainedRaw, false
	f.pending = nil
	s.retainedRaw = retained
	succeeded = true
	return cost, nil
}

func providerHighwaterFileCost(rawBytes uint64, bounds providerIntegrationBounds) (providerFrontierRetention, bool) {
	if bounds.continuation == nil || !bounds.valid() || rawBytes == 0 || rawBytes > candidateLedgerMetadataLimit {
		return providerFrontierRetention{}, false
	}
	metadata, a := providerUpperProduct(rawBytes, 2)
	allocation, b := providerUpperProduct(bounds.fileOverheadBytes, 2)
	stored, c := providerUpperSum(metadata, allocation)
	return providerFrontierRetention{metadata: metadata, stored: stored}, a && b && c && stored <= bounds.maximumStoredBytes
}

func (f *providerFrontierFiles) readExact(name string, expected []byte) bool {
	s := f.store
	raw, err := s.read(s.custody, name, candidateLedgerMetadataLimit)
	return err == nil && s.event(providerCompareEvent, uint64(len(raw)+len(expected))) && bytes.Equal(raw, expected)
}

func (f *providerFrontierFiles) renameHead(ordinal uint64) error {
	s := f.store
	pending := s.name("HEAD.pending", ordinal)
	if pending == "" || s.custody.checkIdentity() != nil || s.evidence.checkIdentity() != nil || !s.event(providerRenameEvent, 0) || s.custody.root.Rename(pending, "HEAD") != nil || !s.event(providerSyncEvent, 0) || s.custody.sync() != nil || s.custody.checkIdentity() != nil || s.evidence.checkIdentity() != nil {
		if meter := s.activeMeter.Load(); meter != nil {
			meter.refuse()
		}
		return errProviderIntegration
	}
	return nil
}

func (f *providerFrontierFiles) checkInitialPrefix() error {
	s, config := f.store, f.store.config
	for _, disk := range []*retentionDisk{s.custody, s.evidence} {
		lock, err := disk.lock.Stat()
		if err != nil || lock.Size() != 0 {
			return errProviderIntegration
		}
	}
	for _, item := range []struct {
		disk  *retentionDisk
		names []string
	}{
		{s.custody, []string{"ANCHOR", "AUTHORITY.LOCK", s.name("setup-intent", 0), s.name("setup-commit", 0)}},
		{s.evidence, []string{"ANCHOR", "AUTHORITY.LOCK", s.name("setup-claim", 0)}},
	} {
		names, err := providerDirectoryNames(item.disk, len(item.names))
		if err != nil || len(names) != len(item.names) {
			return errProviderIntegration
		}
		for _, name := range item.names {
			if !names[name] {
				return errProviderIntegration
			}
		}
	}
	anchor, a := s.read(s.custody, "ANCHOR", int64(config.bounds.maximumMetadataPerCommit))
	local, b := s.read(s.evidence, "ANCHOR", int64(config.bounds.maximumMetadataPerCommit))
	empty, err := newProviderFrontierInventory("", nil, config.bounds.maximumEntries, 64<<20)
	m := contentObject(contentJSON(anchor), "format", "namespace", "instance_key", "profile", "initial_objects_root")
	if a != nil || b != nil || err != nil || m == nil || !bytes.Equal(anchor, local) || providerCandidateHash("sage-provider-custody-anchor-v1", anchor) != config.anchor || m["format"] != "sage-provider-custody-anchor-v1" || m["namespace"] != config.identity.Namespace || m["instance_key"] != config.identity.Instance || m["profile"] != config.profile || m["initial_objects_root"] != empty.digest || s.objectsRoot != empty.digest {
		return errProviderIntegration
	}
	claim, a := s.read(s.evidence, s.name("setup-claim", 0), int64(config.bounds.maximumMetadataPerCommit))
	intent, b := s.read(s.custody, s.name("setup-intent", 0), int64(config.bounds.maximumMetadataPerCommit))
	commit, c := s.read(s.custody, s.name("setup-commit", 0), int64(config.bounds.maximumMetadataPerCommit))
	if a != nil || b != nil || c != nil || !bytes.Equal(claim, intent) || !bytes.Equal(claim, providerCustodySetup(config, s.setupOrigin)) || !bytes.Equal(commit, providerSetupCommit(claim)) || providerCandidateHash("sage-provider-authority-setup-v1", claim) != s.setups[0].claim || providerCandidateHash("sage-provider-authority-setup-commit-v1", commit) != s.setupTip {
		return errProviderIntegration
	}
	// Derive actual original prefix lengths; never trust retainedRaw supplied by
	// a fixture/caller to make the setup envelope appear large enough.
	s.retainedRaw = uint64(len(anchor) + len(local) + len(claim) + len(intent) + len(commit))
	return nil
}
