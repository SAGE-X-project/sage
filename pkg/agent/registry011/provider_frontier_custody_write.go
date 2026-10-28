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
	return r.retainLocked(s, inventory, fresh, intent, commit, frontier, after, nil)
}

// The caller holds store.mu. Only the original live bootstrap owner can spend
// its previously held O on E0; legacy V2/V3 retains the separate charge path.
func (r *providerFrontierReplay) retainLocked(s *providerCustodyStore, inventory *providerIntegrationInventory, fresh []providerIntegrationObject, intent, commit, frontier []byte, after providerFrontierHighwater, prepaid *providerBootstrapOwner) (providerFrontierRetention, error) {
	return r.retainOwnedLocked(s, inventory, fresh, intent, commit, frontier, after, prepaid, nil)
}

// The two closed prepaid roles cannot substitute for each other. O retains
// initial E0 only; F retains the first E1 only, from its original B hold.
func (r *providerFrontierReplay) retainOwnedLocked(s *providerCustodyStore, inventory *providerIntegrationInventory, fresh []providerIntegrationObject, intent, commit, frontier []byte, after providerFrontierHighwater, prepaid *providerBootstrapOwner, first *providerFirstAdmissionOwner) (providerFrontierRetention, error) {
	return r.retainOperationOwnedLocked(s, inventory, fresh, intent, commit, frontier, after, prepaid, first, nil)
}

// The third role spends only a separately pre-I-held first C/S allocation.
// It cannot retain E0/E1, enable a generic V4 writer or adopt retained DATA.
func (r *providerFrontierReplay) retainOperationOwnedLocked(s *providerCustodyStore, inventory *providerIntegrationInventory, fresh []providerIntegrationObject, intent, commit, frontier []byte, after providerFrontierHighwater, prepaid *providerBootstrapOwner, first *providerFirstAdmissionOwner, operation *providerFirstOperationOwner) (providerFrontierRetention, error) {
	return r.retainAllOwnedLocked(s, inventory, fresh, intent, commit, frontier, after, prepaid, first, operation, nil)
}

// The fourth closed role can spend G only for its bound T and its separate
// newly held P only for that T's C/S. It cannot borrow any older E/R owner.
func (r *providerFrontierReplay) retainContinuingOwnedLocked(s *providerCustodyStore, inventory *providerIntegrationInventory, fresh []providerIntegrationObject, intent, commit, frontier []byte, after providerFrontierHighwater, operation *providerContinuingOwner) (providerFrontierRetention, error) {
	return r.retainAllOwnedLocked(s, inventory, fresh, intent, commit, frontier, after, nil, nil, nil, operation)
}

func (r *providerFrontierReplay) retainAllOwnedLocked(s *providerCustodyStore, inventory *providerIntegrationInventory, fresh []providerIntegrationObject, intent, commit, frontier []byte, after providerFrontierHighwater, prepaid *providerBootstrapOwner, first *providerFirstAdmissionOwner, operation *providerFirstOperationOwner, continuing *providerContinuingOwner) (providerFrontierRetention, error) {
	return r.retainClosedOwnedLocked(s, inventory, fresh, intent, commit, frontier, after, prepaid, first, operation, continuing, nil)
}

// The fifth role spends G metadata only for one closed page-seal transition.
func (r *providerFrontierReplay) retainSealingOwnedLocked(s *providerCustodyStore, inventory *providerIntegrationInventory, fresh []providerIntegrationObject, intent, commit, frontier []byte, after providerFrontierHighwater, sealing *providerPageSealOwner) (providerFrontierRetention, error) {
	return r.retainClosedOwnedLocked(s, inventory, fresh, intent, commit, frontier, after, nil, nil, nil, nil, sealing)
}

func (r *providerFrontierReplay) retainClosedOwnedLocked(s *providerCustodyStore, inventory *providerIntegrationInventory, fresh []providerIntegrationObject, intent, commit, frontier []byte, after providerFrontierHighwater, prepaid *providerBootstrapOwner, first *providerFirstAdmissionOwner, operation *providerFirstOperationOwner, continuing *providerContinuingOwner, sealing *providerPageSealOwner) (providerFrontierRetention, error) {
	if s == nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	f := s.frontierFiles
	held := prepaid != nil || first != nil || operation != nil || continuing != nil || sealing != nil
	roles := 0
	for _, present := range [5]bool{prepaid != nil, first != nil, operation != nil, continuing != nil, sealing != nil} {
		if present {
			roles++
		}
	}
	if roles > 1 || s.config.bounds.firstAdmission != nil && (prepaid == nil || !prepaid.writing(s)) && (first == nil || !first.writingLocked(s)) && (operation == nil || !operation.writingLocked(s)) && (continuing == nil || !continuing.writingLocked(s)) && (sealing == nil || !sealing.writingLocked(s)) || s.config.bounds.firstAdmission == nil && held {
		return providerFrontierRetention{}, errProviderIntegration
	}
	if !s.valid() || f == nil || !f.valid() || !f.initialized || f.pending != nil || r == nil || s.frontierReplay != r || r.owner != r || r.failed || r.current.sequence != s.sequence || r.indexes == nil || !r.indexes.valid() || s.sequence != 0 && !r.highwater || f.pin.sequence != s.sequence || f.pin.commit != s.tip || f.pin.setup != s.setupTip || inventory == nil || inventory.previous != s.objectsRoot || !checkProviderHighwaterMainObserved(f.pin, after, s.config, intent, commit, frontier, r.meter) {
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
	var selected *providerFirstOperationChange
	if operation != nil {
		selected = operation.change
	}
	if continuing != nil {
		selected = continuing.change
	}
	if sealing != nil {
		selected = sealing.change
	}
	if operation != nil || continuing != nil || sealing != nil {
		if selected == nil || selected.inventory != inventory || selected.after != after || len(selected.fresh) != len(fresh) || !providerReplayEqual(selected.intent, intent, r.meter) || !providerReplayEqual(selected.commit, commit, r.meter) || !providerReplayEqual(selected.frontier, frontier, r.meter) {
			return providerFrontierRetention{}, errProviderIntegration
		}
		for i, original := range selected.fresh {
			if original.kind != fresh[i].kind || original.digest != fresh[i].digest || !providerReplayTextEqual(original.raw, fresh[i].raw, r.meter) {
				return providerFrontierRetention{}, errProviderIntegration
			}
		}
	}
	if len(fresh) == 0 || len(fresh) > s.config.bounds.maximumEntries || len(s.objects) > s.config.bounds.maximumObjects-len(fresh) || !f.readExact("HEAD", providerReplayBytes(f.pinRaw, r.meter)) || !f.readExact(s.name("highwater", f.pin.ordinal), providerReplayBytes(f.pinRaw, r.meter)) {
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
		key, keyOK := providerReplayRefKey(providerIntegrationObjectRef{kind: object.kind, digest: object.digest}, r.meter)
		if _, exists := s.objects[key]; !keyOK || exists || seen[key] {
			return providerFrontierRetention{}, errProviderIntegration
		}
		derived, err := newProviderIntegrationObjectObserved(object.kind, providerReplayBytes(object.raw, r.meter), s.config.bounds.maximumObjectBytes, r.meter)
		if err != nil || derived != object {
			return providerFrontierRetention{}, errProviderIntegration
		}
		seen[key] = true
	}
	base, err := providerFrontierRetentionCostObserved(s.config.identity, inventory, fresh, intent, commit, frontier, s.config.bounds, r.meter)
	raw, rawErr := after.bytesObserved(s.config, r.meter)
	headCost, headOK := providerHighwaterFileCost(uint64(len(raw)), s.config.bounds)
	total, totalOK := providerUpperSum(base.stored, headCost.stored)
	metadataRaw, metaOK := providerUpperSum(rawBytes, uint64(len(inventory.raw)), uint64(2*len(intent)+len(commit)+len(frontier)))
	retained, retainedOK := providerUpperSum(s.retainedRaw, metadataRaw, headCost.metadata)
	if err != nil || rawErr != nil || !headOK || !totalOK || !metaOK || !retainedOK || !held && !candidateCapacity(0, s.stored, total, s.config.bounds.maximumStoredBytes) || r.preflight(s, inventory, fresh) != nil || r.preflightRetention(s, inventory, fresh, intent, commit, frontier, after) != nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	provisional, parseErr := parseProviderFrontierRecordObserved(frontier, s.config.identity, r.meter)
	if parseErr != nil || !providerFrontierClockValid(s, provisional) {
		return providerFrontierRetention{}, errProviderIntegration
	}
	// The complete main+head physical charge is held BEFORE the first write.
	// The file step consumes this exact claim without another charge/refund.
	if prepaid != nil {
		if !prepaid.consume(total) {
			return providerFrontierRetention{}, errProviderIntegration
		}
	} else if first != nil {
		if !first.consumeLocked(total) {
			return providerFrontierRetention{}, errProviderIntegration
		}
	} else if operation != nil {
		if !operation.consumeStorageLocked(total) {
			return providerFrontierRetention{}, errProviderIntegration
		}
	} else if continuing != nil {
		if !continuing.consumeStorageLocked(total) {
			return providerFrontierRetention{}, errProviderIntegration
		}
	} else if sealing != nil {
		if !sealing.consumeStorageLocked(total) {
			return providerFrontierRetention{}, errProviderIntegration
		}
	} else {
		s.stored += total
	}
	claim := &providerFrontierWriteClaim{owner: f, before: f.pinRaw, after: providerReplayString(raw, r.meter), kind: "main", intent: providerReplayString(intent, r.meter), commit: providerReplayString(commit, r.meter), frontier: providerReplayString(frontier, r.meter), stored: s.stored, headCost: headCost}
	if claim.after == "" || claim.intent == "" || claim.commit == "" || claim.frontier == "" {
		return providerFrontierRetention{}, errProviderIntegration
	}
	f.pending = claim
	next := s.sequence + 1
	if s.write(s.custody, s.name("intent", next), intent) != nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	for _, object := range fresh {
		if s.write(s.evidence, s.objectName(object.kind, object.digest), providerReplayBytes(object.raw, r.meter)) != nil {
			return providerFrontierRetention{}, errProviderIntegration
		}
		key, ok := providerReplayRefKey(providerIntegrationObjectRef{kind: object.kind, digest: object.digest}, r.meter)
		if !ok {
			return providerFrontierRetention{}, errProviderIntegration
		}
		s.objects[key] = providerIntegrationObjectRef{object.kind, object.digest, uint64(len(object.raw))}
	}
	if s.write(s.evidence, s.inventoryName(inventory.digest), providerReplayBytes(inventory.raw, s.activeMeter.Load())) != nil || s.write(s.evidence, s.name("pending", next), intent) != nil || s.write(s.custody, s.name("commit", next), commit) != nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	// Independently read every new original object through object(), then derive
	// typed ledger/index state and one combined metadata debit from those bytes.
	if !s.exactFrontierMain(next, inventory, intent, commit) {
		return providerFrontierRetention{}, errProviderIntegration
	}
	// The live append must derive the same wire-binding index as a full
	// original scan. A producer's in-memory exchange is not that index.
	if operation != nil && operation.state == providerFirstOperationSettlement && len(fresh) == 11 && operation.bindRetainedExchangeLocked(s, fresh) != nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	cost, err := r.applyWithHighwater(s, inventory, fresh, intent, commit, frontier, after)
	if err != nil || cost.stored != total || s.write(s.custody, s.name("frontier", next), frontier) != nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	// Reconfirm main/frontier bytes and the owned clock before and after the
	// physical head replacement. Late/uncertain state never returns success.
	if !s.exactFrontierMain(next, inventory, intent, commit) || !s.exactFrontierFile(s.name("frontier", next), frontier) || !r.currentClockValid(s) {
		return providerFrontierRetention{}, errProviderIntegration
	}
	if _, err := f.advanceClaimedLocked(after, "main", intent, commit, frontier, claim); err != nil || !claim.used || !r.currentClockValid(s) || !s.exactFrontierMain(next, inventory, intent, commit) || !s.exactFrontierFile(s.name("frontier", next), frontier) {
		return providerFrontierRetention{}, errProviderIntegration
	}
	s.sequence, s.tip, s.enrollment, s.objectsRoot = next, after.commit, r.current.enrollment, inventory.digest
	s.retainedRaw = retained
	succeeded = true
	return cost, nil
}

func (s *providerCustodyStore) exactFrontierMain(sequence uint64, inventory *providerIntegrationInventory, intent, commit []byte) bool {
	local, err := s.read(s.evidence, s.inventoryName(inventory.digest), 64<<10)
	pending, pendingErr := s.read(s.evidence, s.name("pending", sequence), int64(s.config.bounds.maximumMetadataPerCommit))
	return err == nil && pendingErr == nil && providerReplayEqual(local, providerReplayBytes(inventory.raw, s.activeMeter.Load()), s.activeMeter.Load()) && providerReplayEqual(pending, intent, s.activeMeter.Load()) && s.exactFrontierFile(s.name("intent", sequence), intent) && s.exactFrontierFile(s.name("commit", sequence), commit)
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
