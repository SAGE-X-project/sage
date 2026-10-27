package registry011

import "bytes"

// A claim is a private, one-use allocation record under store.mu. It cannot be
// reconstructed from JSON or serve as an operating principal/dispatch permit.
type providerFrontierWriteClaim struct {
	owner                    *providerFrontierFiles
	before, after, kind      string
	intent, commit, frontier string
	stored                   uint64
	headCost                 providerFrontierRetention
	used                     bool
}

func (*providerFrontierWriteClaim) UnmarshalJSON([]byte) error { return errProviderIntegration }

// This candidate writer returns only the independently derived retained DATA.
// There is no execution ACK/qualified Source constructor here. The actual
// principal/open/full-scan/selection boundary remains a separate prerequisite.
// The caller supplies the coordinator's provisional delta, never its maps.
func (r *providerFrontierReplay) retain(s *providerCustodyStore, inventory *providerIntegrationInventory, fresh []providerIntegrationObject, intent, commit, frontier []byte, after providerFrontierHighwater) (providerFrontierRetention, error) {
	if s == nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	f := s.frontierFiles
	if !s.valid() || f == nil || !f.valid() || !f.initialized || f.pending != nil || r == nil || s.frontierReplay != r || r.owner != r || r.failed || r.current.sequence != s.sequence || r.indexes == nil || !r.indexes.valid() || s.sequence != 0 && !r.highwater || f.pin.sequence != s.sequence || f.pin.commit != s.tip || f.pin.setup != s.setupTip || inventory == nil || inventory.previous != s.objectsRoot || !checkProviderHighwaterMain(f.pin, after, s.config, intent, commit, frontier) {
		return providerFrontierRetention{}, errProviderIntegration
	}
	// Once a valid transaction starts, any later uncertainty quarantines BOTH
	// owners. There is no rollback, refund, retry or new-pin adoption.
	succeeded := false
	defer func() {
		if !succeeded {
			s.failed, f.failed, r.failed = true, true, true
		}
	}()
	if len(fresh) == 0 || len(fresh) > s.config.bounds.maximumEntries || len(s.objects) > s.config.bounds.maximumObjects-len(fresh) || !f.readExact("HEAD", []byte(f.pinRaw)) || !f.readExact(providerCustodyName("highwater", f.pin.ordinal), []byte(f.pinRaw)) {
		return providerFrontierRetention{}, errProviderIntegration
	}
	var rawBytes uint64
	for _, object := range fresh {
		if len(object.raw) == 0 || uint64(len(object.raw)) > s.config.bounds.maximumObjectBytes {
			return providerFrontierRetention{}, errProviderIntegration
		}
		var ok bool
		rawBytes, ok = providerUpperSum(rawBytes, uint64(len(object.raw)))
		if !ok || rawBytes > 64<<20 {
			return providerFrontierRetention{}, errProviderIntegration
		}
	}
	scratch, scratchOK := providerUpperSum(rawBytes, 4*s.config.bounds.maximumMetadataPerCommit, 64<<10)
	tickets, segments := uint64(0), uint64(0)
	if r.body == nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	if r.body.ledger != nil {
		tickets, segments = uint64(len(r.body.ledger.tickets)), uint64(len(r.body.ledger.segments))
	}
	if _, ok := r.indexes.bounds.residentUpper(tickets+1, s.sequence+1, segments+1, uint64(len(s.objects)+len(fresh)), s.retainedRaw, scratch); !scratchOK || !ok {
		return providerFrontierRetention{}, errProviderIntegration
	}
	seen := make(map[string]bool, len(fresh))
	for _, object := range fresh {
		key := object.kind + ":" + object.digest
		if _, exists := s.objects[key]; exists || seen[key] {
			return providerFrontierRetention{}, errProviderIntegration
		}
		domain, known := providerIntegrationDomain(object.kind)
		if !known || !s.event(providerHashEvent, uint64(len(object.raw)+len(domain))) || !s.event(providerCopyEvent, uint64(len(object.raw))) || domain != "" && !s.event(providerCodecEvent, uint64(len(object.raw))) {
			return providerFrontierRetention{}, errProviderIntegration
		}
		derived, err := newProviderIntegrationObject(object.kind, []byte(object.raw), s.config.bounds.maximumObjectBytes)
		if err != nil || derived != object {
			return providerFrontierRetention{}, errProviderIntegration
		}
		seen[key] = true
	}
	base, err := providerFrontierRetentionCost(s.config.identity, inventory, fresh, intent, commit, frontier, s.config.bounds)
	raw, rawErr := after.bytes(s.config)
	headCost, headOK := providerHighwaterFileCost(uint64(len(raw)), s.config.bounds)
	total, totalOK := providerUpperSum(base.stored, headCost.stored)
	metadataRaw, metaOK := providerUpperSum(rawBytes, uint64(len(inventory.raw)), uint64(2*len(intent)+len(commit)+len(frontier)))
	retained, retainedOK := providerUpperSum(s.retainedRaw, metadataRaw, headCost.metadata)
	if err != nil || rawErr != nil || !headOK || !totalOK || !metaOK || !retainedOK || !candidateCapacity(0, s.stored, total, s.config.bounds.maximumStoredBytes) || r.preflight(s, inventory, fresh) != nil || r.preflightRetention(s, inventory, fresh, intent, commit, frontier, after) != nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	provisional, parseErr := parseProviderFrontierRecord(frontier, s.config.identity)
	if parseErr != nil || !providerFrontierClockValid(s, provisional) {
		return providerFrontierRetention{}, errProviderIntegration
	}
	// The complete main+head physical charge is held BEFORE the first write.
	// The file step consumes this exact claim without another charge/refund.
	s.stored += total
	claim := &providerFrontierWriteClaim{owner: f, before: f.pinRaw, after: string(raw), kind: "main", intent: string(intent), commit: string(commit), frontier: string(frontier), stored: s.stored, headCost: headCost}
	f.pending = claim
	next := s.sequence + 1
	if s.write(s.custody, providerCustodyName("intent", next), intent) != nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	for _, object := range fresh {
		if s.write(s.evidence, providerObjectName(object.kind, object.digest), []byte(object.raw)) != nil {
			return providerFrontierRetention{}, errProviderIntegration
		}
		s.objects[object.kind+":"+object.digest] = providerIntegrationObjectRef{object.kind, object.digest, uint64(len(object.raw))}
	}
	if s.write(s.evidence, "inventory-"+inventory.digest[2:], []byte(inventory.raw)) != nil || s.write(s.evidence, providerCustodyName("pending", next), intent) != nil || s.write(s.custody, providerCustodyName("commit", next), commit) != nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	// Independently read every new original object through object(), then derive
	// typed ledger/index state and one combined metadata debit from those bytes.
	if !s.exactFrontierMain(next, inventory, intent, commit) {
		return providerFrontierRetention{}, errProviderIntegration
	}
	cost, err := r.applyWithHighwater(s, inventory, fresh, intent, commit, frontier, after)
	if err != nil || cost.stored != total || s.write(s.custody, providerCustodyName("frontier", next), frontier) != nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	// Reconfirm main/frontier bytes and the owned clock before and after the
	// physical head replacement. Late/uncertain state never returns success.
	if !s.exactFrontierMain(next, inventory, intent, commit) || !s.exactFrontierFile(providerCustodyName("frontier", next), frontier) || !r.currentClockValid(s) {
		return providerFrontierRetention{}, errProviderIntegration
	}
	if _, err := f.advanceClaimedLocked(after, "main", intent, commit, frontier, claim); err != nil || !claim.used || !r.currentClockValid(s) || !s.exactFrontierMain(next, inventory, intent, commit) || !s.exactFrontierFile(providerCustodyName("frontier", next), frontier) {
		return providerFrontierRetention{}, errProviderIntegration
	}
	s.sequence, s.tip, s.enrollment, s.objectsRoot = next, after.commit, r.current.enrollment, inventory.digest
	s.retainedRaw = retained
	succeeded = true
	return cost, nil
}

func (s *providerCustodyStore) exactFrontierMain(sequence uint64, inventory *providerIntegrationInventory, intent, commit []byte) bool {
	local, err := s.read(s.evidence, "inventory-"+inventory.digest[2:], 64<<10)
	pending, pendingErr := s.read(s.evidence, providerCustodyName("pending", sequence), int64(s.config.bounds.maximumMetadataPerCommit))
	return err == nil && pendingErr == nil && bytes.Equal(local, []byte(inventory.raw)) && bytes.Equal(pending, intent) && s.exactFrontierFile(providerCustodyName("intent", sequence), intent) && s.exactFrontierFile(providerCustodyName("commit", sequence), commit)
}

func (s *providerCustodyStore) exactFrontierFile(name string, expected []byte) bool {
	raw, err := s.read(s.custody, name, int64(s.config.bounds.maximumMetadataPerCommit))
	return err == nil && s.event(providerCompareEvent, uint64(len(raw)+len(expected))) && bytes.Equal(raw, expected)
}

func (r *providerFrontierReplay) currentClockValid(s *providerCustodyStore) bool {
	return providerFrontierClockValid(s, r.current)
}

func providerFrontierClockValid(s *providerCustodyStore, frontier providerFrontierRecord) bool {
	now, err := s.sample()
	return err == nil && candidateLedgerClockAdvance(frontier.clock, providerClockDiagnostic(now)) && !candidateLedgerExpired(providerClockDiagnostic(now), frontier.deadline)
}
