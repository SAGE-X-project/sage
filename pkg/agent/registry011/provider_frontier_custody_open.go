package registry011

import (
	"bytes"
	"os"
)

// Initial-only PRIVATE candidate opener. The two directories/locks/anchors
// already exist and are selected configuration; it provisions no environment.
// Its exact empty view is a bounded filesystem observation, never independent
// authentication of absence, principal qualification or an operating Source.
// Reopen/recovery is inspection-only and is deliberately not this constructor.
func openProviderFrontierCandidate(config providerCustodyConfig, clock *providerOwnedClock) (*providerCustodyStore, error) {
	if config.bounds.scan != nil {
		return openProviderScanCandidate(config, clock)
	}
	if config.expectedScanHead != "" {
		return nil, ErrConfiguration
	}
	r, err := newProviderFrontierReplay(config)
	id := config.identity
	if err != nil || clock == nil || clock.policy != id.Policy || config.attempt != 0 || config.expectedTip != config.anchor || config.expectedSetupTip != config.anchor || !lowerHash(config.anchor) || !lowerHash(id.Namespace) || !lowerHash(id.Instance) || !lowerHash(id.Policy) || !lowerHash(id.SourceID) || !lowerHash(id.Provider) || id.Generation != 0 || config.custodyDirectory == "" || config.evidenceDirectory == "" {
		return nil, ErrConfiguration
	}
	clock.mu.Lock()
	startRead := clock.read
	clock.mu.Unlock()
	if !clock.claimLegacy(id.Policy) {
		return nil, ErrConfiguration
	}
	custody, err := openRetentionDisk(config.custodyDirectory)
	if err != nil {
		return nil, errProviderIntegration
	}
	evidence, err := openRetentionDisk(config.evidenceDirectory)
	if err != nil {
		_ = custody.close()
		return nil, errProviderIntegration
	}
	s := &providerCustodyStore{config: config, clock: clock, custody: custody, evidence: evidence, tip: config.anchor, frontierReplay: r, objects: make(map[string]providerIntegrationObjectRef), operations: make(map[string]bool), bindings: make(map[string]string)}
	s.owner = s
	succeeded := false
	defer func() {
		if !succeeded {
			s.failed, r.failed = true, true
			_ = evidence.close()
			_ = custody.close()
		}
	}()
	if os.SameFile(custody.directoryInfo, evidence.directoryInfo) || os.SameFile(custody.lockInfo, evidence.lockInfo) {
		return nil, errProviderIntegration
	}
	for _, disk := range []*retentionDisk{custody, evidence} {
		names, err := providerDirectoryNames(disk, 2)
		lock, lockErr := disk.lock.Stat()
		if err != nil || len(names) != 2 || !names["ANCHOR"] || !names["AUTHORITY.LOCK"] || lockErr != nil || lock.Size() != 0 {
			return nil, errProviderIntegration
		}
	}
	anchor, a := s.read(custody, "ANCHOR", int64(config.bounds.maximumMetadataPerCommit))
	local, b := s.read(evidence, "ANCHOR", int64(config.bounds.maximumMetadataPerCommit))
	empty, emptyErr := newProviderFrontierInventory("", nil, config.bounds.maximumEntries, min(config.bounds.maximumStoredBytes, 64<<20))
	value := contentObject(contentJSON(anchor), "format", "namespace", "instance_key", "profile", "initial_objects_root")
	if a != nil || b != nil || emptyErr != nil || value == nil || !bytes.Equal(anchor, local) || providerCandidateHash("sage-provider-custody-anchor-v1", anchor) != config.anchor || value["format"] != "sage-provider-custody-anchor-v1" || value["namespace"] != id.Namespace || value["instance_key"] != id.Instance || value["profile"] != config.profile || value["initial_objects_root"] != empty.digest || !bytes.Equal(anchor, providerIntegrationJSON(value)) {
		return nil, errProviderIntegration
	}
	s.objectsRoot = empty.digest
	initial, err := s.sample()
	if err != nil {
		return nil, errProviderIntegration
	}
	claim := providerCustodySetup(config, initial)
	completed := providerSetupCommit(claim)
	// Fund the entire initial claim, both anchors/locks and BOTH future H0
	// copies before the first setup write. H0's bounded upper is not actual.
	allocation, allocOK := providerUpperProduct(9, config.bounds.fileOverheadBytes)
	upper, upperOK := providerUpperSum(uint64(2*len(anchor)+2*len(claim)+len(completed)), 2*candidateLedgerMetadataLimit, allocation)
	if !allocOK || !upperOK || upper > config.bounds.setupRetained {
		return nil, errProviderIntegration
	}
	s.stored, s.retainedRaw = config.bounds.setupRetained, uint64(2*len(anchor))
	if s.writeSetup(initial, uint64(2*len(anchor))) != nil {
		return nil, errProviderIntegration
	}
	now, err := s.sample()
	if err != nil {
		return nil, errProviderIntegration
	}
	custodyNames, a := providerDirectoryNames(custody, 4)
	localNames, b := providerDirectoryNames(evidence, 3)
	if a != nil || b != nil || s.loadSetupPrefix(custodyNames, localNames, now) != nil || len(custodyNames) != 2 || len(localNames) != 2 {
		return nil, errProviderIntegration
	}
	f, err := newProviderFrontierFiles(s)
	if err != nil {
		return nil, errProviderIntegration
	}
	if _, err := f.initialize(); err != nil {
		return nil, errProviderIntegration
	}
	end, err := s.sample()
	clock.mu.Lock()
	endRead := clock.read
	clock.mu.Unlock()
	if err != nil || !candidateLedgerClockAdvance(providerClockDiagnostic(initial), providerClockDiagnostic(end)) || uint64(end.MonotonicMS-initial.MonotonicMS) > config.bounds.setupMaximumDurationMS || endRead < startRead || endRead-startRead > config.bounds.setupMaximumClockReads || !s.valid() {
		return nil, errProviderIntegration
	}
	succeeded = true
	return s, nil // Metadata handle only; no qualification/dispatch acknowledgment.
}
