package registry011

// A separate custodian reconstructs this state from its exact retained prefix.
// It never shares the coordinator's provisional ledger or index maps. This
// typed replay is deliberately not an external current-tip selection/CAS, a
// custody ACK, a recovered execution permit or operating policy qualification.
type providerFrontierReplay struct {
	owner        *providerFrontierReplay
	body         *providerCustodyReplay
	indexes      *providerLedgerIndexes
	anchors      []providerIntegrationObject
	current      providerFrontierRecord
	preI         providerFrontierRetention
	meter        *providerWorkMeter
	setupOrdinal int
	highwater    bool
	failed       bool
}

func (*providerFrontierReplay) UnmarshalJSON([]byte) error { return errProviderIntegration }

func newProviderFrontierReplay(config providerCustodyConfig) (*providerFrontierReplay, error) {
	return newProviderFrontierReplayObserved(config, nil)
}

func newProviderFrontierReplayObserved(config providerCustodyConfig, meter *providerWorkMeter) (*providerFrontierReplay, error) {
	if config.bounds.continuation == nil || !config.bounds.valid() || !config.capture.valid() || !config.bounds.methods.fits(config.capture) || config.policy == nil || config.policy.raw == "" || config.identity.Policy != config.policy.policy || config.identity.Instance != config.policy.instance || config.identity.Provider != config.policy.provider {
		return nil, errProviderIntegration
	}
	if meter != nil && !meter.preflight(providerEncodeEvent, config.bounds.maximumMetadataPerCommit) {
		return nil, errProviderIntegration
	}
	profile, domain := providerIntegrationProfileBytes(config.bounds, config.capture)
	if meter != nil && (!meter.event(providerEncodeEvent, uint64(len(profile))) || !meter.event(providerHashEvent, uint64(len(profile)+1+len(domain)))) {
		return nil, errProviderIntegration
	}
	if config.profile != providerCandidateHash(domain, profile) {
		return nil, errProviderIntegration
	}
	parsedPolicy, err := parseProviderPolicyCandidateObserved(providerReplayPhaseBytes(config.policy.raw, meter), 64<<10, meter)
	if err != nil || *parsedPolicy != *config.policy {
		return nil, errProviderIntegration
	}
	indexes, anchors, err := newProviderLedgerIndexesObserved(config.identity, 1, *config.bounds.continuation, meter)
	if err != nil {
		return nil, errProviderIntegration
	}
	r := &providerFrontierReplay{body: &providerCustodyReplay{context: &ProviderCandidateContext{config.identity, true}, policy: config.policy}, indexes: indexes, anchors: anchors, setupOrdinal: -1}
	r.owner = r
	r.body.frontier = r
	return r, nil
}

// Creation-time setup observation must not leave its expired setup meter
// attached to the producer cache. Only this independent pre-I scan instance
// keeps the one live, already claimed body meter for its typed DATA reads.
func newProviderFrontierScanReplay(c *providerScanControl, meter *providerWorkMeter) (*providerFrontierReplay, error) {
	if !c.valid() || c.prepared == nil || meter == nil || meter.owner != meter || meter.scan != c || meter.integration != nil || meter.failed || meter.closed || c.store.activeMeter.Load() != meter || meter.upper != c.prepared.body.work || meter.maximumClocks != c.prepared.body.clocks {
		return nil, errProviderIntegration
	}
	r, err := newProviderFrontierReplayObserved(c.store.config, meter)
	if err != nil {
		return nil, err
	}
	r.meter, r.body.meter, r.indexes.meter = meter, meter, meter
	return r, nil
}

// apply consumes retained data only. The surrounding concrete custodian must
// have checked its independently selected current composite tip before any
// write. After this typed delta it must perform and read back that exact CAS
// before returning a private ACK. This function returns no such credential.
// A failure poisons the replay cache; no partially advanced state is reusable.
func (r *providerFrontierReplay) apply(s *providerCustodyStore, inventory *providerIntegrationInventory, fresh []providerIntegrationObject, intent, commit, frontier []byte) (providerFrontierRetention, error) {
	return r.applyRetained(s, inventory, fresh, intent, commit, frontier, nil)
}

// The physical V2 store must use this path, not apply: all immutable high-water
// copies join the SAME common-ledger debit as indexes/custody/page storage.
// This still returns retained DATA only, never a selected current tip or ACK.
// The caller holds store.mu and must separately finish the owned HEAD CAS.
func (r *providerFrontierReplay) applyWithHighwater(s *providerCustodyStore, inventory *providerIntegrationInventory, fresh []providerIntegrationObject, intent, commit, frontier []byte, after providerFrontierHighwater) (providerFrontierRetention, error) {
	return r.applyRetained(s, inventory, fresh, intent, commit, frontier, &after)
}

func (r *providerFrontierReplay) applyRetained(s *providerCustodyStore, inventory *providerIntegrationInventory, fresh []providerIntegrationObject, intent, commit, frontier []byte, after *providerFrontierHighwater) (providerFrontierRetention, error) {
	if r == nil || r.owner != r || r.failed || r.body == nil || r.indexes == nil || !r.indexes.valid() || s == nil || r.meter != r.body.meter || r.meter != r.indexes.meter || r.meter != nil && s.activeMeter.Load() != r.meter || s.config.bounds.continuation == nil || r.indexes.identity != s.config.identity || r.indexes.bounds != *s.config.bounds.continuation || s.sequence != r.current.sequence || s.config.policy != r.body.policy {
		return providerFrontierRetention{}, errProviderIntegration
	}
	r.failed = true
	// Do not graft physical-store accounting onto an earlier unqualified
	// data-only prefix (or drop it from an already physical replay prefix).
	if r.current.sequence != 0 && r.highwater != (after != nil) {
		return providerFrontierRetention{}, errProviderIntegration
	}
	if r.preflight(s, inventory, fresh) != nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	m := providerFrontierIntentObjectObserved(intent, r.meter)
	if m == nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	clock, clockOK := providerStoredClock(m["clock"])
	setup, setupErr := r.setupAtBoundary(s, clock)
	if !clockOK || setupErr != nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	rootRaw, err := s.object("enrollment", contentString(m["new_enrollment_root"]))
	root, parseErr := parseCandidateEnrollmentRootObserved(r.body.context, rootRaw, 64<<10, r.meter)
	if err != nil || parseErr != nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	link := providerFrontierLink{identity: s.config.identity, fence: root.fence, sequence: s.sequence, commit: s.tip, setup: setup.commit, enrollment: s.enrollment, objects: s.objectsRoot}
	if s.sequence != 0 {
		prior, err := r.current.bytesObserved(r.meter)
		if err != nil || r.current.commit != s.tip || r.current.enrollment != s.enrollment || r.current.objects != s.objectsRoot {
			return providerFrontierRetention{}, errProviderIntegration
		}
		link.frontier = providerReplayDigest("sage-provider-custody-frontier-v1", prior, r.meter)
	}
	kind, operation := contentString(m["kind"]), contentString(m["operation"])
	if !checkProviderFrontierIntentObserved(intent, link, root, inventory, kind, operation, s.config.bounds, r.meter) || !providerReplayEqual(commit, providerFrontierCommitObserved(intent, r.meter), r.meter) || providerCheckNewEnrollmentObserved(s, root, kind, r.meter) != nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	claimed, err := parseProviderFrontierRecordObserved(frontier, s.config.identity, r.meter)
	if err != nil || !clockOK || claimed.clock != clock || !candidateLedgerClockAdvance(providerClockDiagnostic(s.setupOrigin), clock) || s.sequence != 0 && !candidateLedgerClockAdvance(r.current.clock, clock) {
		return providerFrontierRetention{}, errProviderIntegration
	}
	if r.checkOldMaterials(s, kind, operation) != nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	entry, boundary, err := r.applyBody(s, root, m, claimed)
	if err != nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	if r.body.ledger != nil && r.indexes.ledger == nil {
		if s.config.bounds.firstAdmission != nil {
			metadata, ok := providerUpperSum(r.preI.metadata, r.body.scanMetadata)
			if !ok || r.body.scanMetadata == 0 || metadata > r.indexes.bounds.maximumMetadataBytes {
				return providerFrontierRetention{}, errProviderIntegration
			}
			r.preI.metadata = metadata
		}
		if err := r.indexes.bindRetainedLedger(r.body.ledger, r.preI); err != nil {
			return providerFrontierRetention{}, errProviderIntegration
		}
	}
	delta, err := r.indexes.prepare(kind, operation, s.sequence+1, entry, root, r.body.ledger)
	if err != nil || r.checkIndexObjects(s, delta, kind == "enroll") != nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	derived, err := r.deriveFrontier(s, root, delta, commit, claimed, clock, setup.commit)
	if err != nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	raw, err := derived.bytesObserved(r.meter)
	if err != nil || !providerReplayEqual(raw, frontier, r.meter) || s.sequence != 0 && !checkProviderFrontierAdvanceObserved(r.current, derived, r.meter) {
		return providerFrontierRetention{}, errProviderIntegration
	}
	cost, err := providerFrontierRetentionCostObserved(s.config.identity, inventory, fresh, intent, commit, frontier, s.config.bounds, r.meter)
	if err != nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	if after != nil {
		headCost, err := r.highwaterRetention(s, *after, setup, intent, commit, frontier)
		metadata, a := providerUpperSum(cost.metadata, headCost.metadata)
		stored, b := providerUpperSum(cost.stored, headCost.stored)
		if err != nil || !a || !b || metadata > r.indexes.bounds.maximumMetadataBytes || stored > s.config.bounds.maximumStoredBytes {
			return providerFrontierRetention{}, errProviderIntegration
		}
		cost.metadata, cost.stored = metadata, stored
	}
	setupBytes, ok := r.setupBytes(s, setup.ordinal)
	metadata, sumOK := providerUpperSum(cost.metadata, setupBytes)
	if !ok || !sumOK || metadata > r.indexes.bounds.maximumMetadataBytes {
		return providerFrontierRetention{}, errProviderIntegration
	}
	// Setup files have their own finite physical retention charge when written.
	// Their exact raw bytes enter the common ledger once at their first main
	// boundary. They are not written/charged as new main files a second time.
	cost.metadata = metadata
	if kind == "enroll" {
		// No I exists yet. Charge this exact retained prefix once when the first
		// independently parsed I/T ledger is constructed. Setup/open charges are
		// separately derived from the already retained setup prefix here.
		if r.indexes.apply(delta) != nil {
			return providerFrontierRetention{}, errProviderIntegration
		}
		r.preI = cost
	} else if r.indexes.applyRetained(delta, cost) != nil {
		return providerFrontierRetention{}, errProviderIntegration
	}
	if boundary != nil {
		priorHeads := delta.previous
		if r.body.ledger.applyProviderBoundary(boundary, providerReplayPhaseBytes(boundary.pageObject.raw, r.meter), providerReplayPhaseBytes(entry.raw, r.meter), priorHeads) != nil {
			return providerFrontierRetention{}, errProviderIntegration
		}
	}
	// Every newly retained inventory entry must have been semantically required
	// by the independent typed reconstruction, not merely hashed/read once.
	for _, object := range fresh {
		key, ok := providerReplayRefKey(providerIntegrationObjectRef{kind: object.kind, digest: object.digest}, r.meter)
		if !ok || !s.requiredReferences[key] {
			return providerFrontierRetention{}, errProviderIntegration
		}
	}
	r.current, r.setupOrdinal, r.failed = derived, setup.ordinal, false
	r.highwater = after != nil
	return cost, nil
}

func (r *providerFrontierReplay) preflight(s *providerCustodyStore, inventory *providerIntegrationInventory, fresh []providerIntegrationObject) error {
	if inventory == nil || len(fresh) == 0 || len(fresh) > s.config.bounds.maximumEntries || len(fresh) != len(inventory.entries) || inventory.previous != s.objectsRoot || len(s.objects) > s.config.bounds.maximumObjects || len(s.setups) == 0 || len(s.setups) > s.config.bounds.maximumSetupRecoveries+1 {
		return errProviderIntegration
	}
	var scratch uint64
	for _, object := range fresh {
		if len(object.raw) == 0 || uint64(len(object.raw)) > s.config.bounds.maximumObjectBytes {
			return errProviderIntegration
		}
		var ok bool
		scratch, ok = providerUpperSum(scratch, uint64(len(object.raw)))
		if !ok || scratch > 64<<20 {
			return errProviderIntegration
		}
	}
	scratch, ok := providerUpperSum(scratch, 4*s.config.bounds.maximumMetadataPerCommit, 64<<10)
	if !ok {
		return errProviderIntegration
	}
	tickets, segments := uint64(0), uint64(0)
	if r.body.ledger != nil {
		tickets, segments = uint64(len(r.body.ledger.tickets)), uint64(len(r.body.ledger.segments))
	}
	if _, ok := r.indexes.bounds.residentUpper(min(tickets+1, r.indexes.bounds.maximumRecords), s.sequence+1, min(segments+1, r.indexes.bounds.maximumRecords), uint64(len(s.objects)), s.retainedRaw, scratch); !ok {
		return errProviderIntegration
	}
	s.requiredReferences = make(map[string]bool)
	if s.config.bounds.guardian != nil {
		// The bound inventory is the only finite incoming reference slice.
		// Every original still has to be semantically consumed below.
		s.newReferences = inventory.entries
	}
	return nil
}

func (r *providerFrontierReplay) applyBody(s *providerCustodyStore, root *CandidateEnrollmentRoot, m map[string]any, claimed providerFrontierRecord) (providerIntegrationObject, *providerLedgerBoundary, error) {
	kind, operation := contentString(m["kind"]), contentString(m["operation"])
	if kind == "seal" {
		ledger := r.body.ledger
		if ledger == nil || claimed.seal == "" || root.budget != ledger.current.digest {
			return providerIntegrationObject{}, nil, errProviderIntegration
		}
		raw, err := s.object("ledger-seal", claimed.seal)
		value, ok := providerReplayJSON(raw, r.meter).(map[string]any)
		final, boolOK := value["final"].(bool)
		if err != nil || !ok || !boolOK {
			return providerIntegrationObject{}, nil, errProviderIntegration
		}
		boundary, err := ledger.prepareProviderBoundary(final)
		if err != nil {
			return providerIntegrationObject{}, nil, errProviderIntegration
		}
		seal, err := boundary.sealObject(r.indexes.heads)
		page, pageErr := s.object("ledger-page", boundary.pageObject.digest)
		budget, budgetErr := s.object("budget", root.budget)
		if err != nil || pageErr != nil || budgetErr != nil || !providerReplayEqual(raw, providerReplayPhaseBytes(seal.raw, r.meter), r.meter) || !providerReplayEqual(page, providerReplayPhaseBytes(boundary.pageObject.raw, r.meter), r.meter) || !providerReplayEqual(budget, providerReplayPhaseBytes(ledger.current.raw, r.meter), r.meter) {
			return providerIntegrationObject{}, nil, errProviderIntegration
		}
		return seal, boundary, nil
	}
	if r.body.apply(s, m) != nil {
		return providerIntegrationObject{}, nil, errProviderIntegration
	}
	entryKind, entryDigest := "", operation
	switch kind {
	case "enroll":
		entryKind, entryDigest = "enrollment", root.digest
	case "admit":
		entryKind = "ticket"
	case "consume":
		entryKind = "consumption"
	case "settle":
		entryKind = "settlement"
	default:
		return providerIntegrationObject{}, nil, errProviderIntegration
	}
	raw, err := s.object(entryKind, entryDigest)
	if err != nil {
		return providerIntegrationObject{}, nil, errProviderIntegration
	}
	entry, err := newProviderIntegrationObjectObserved(entryKind, raw, 64<<10, r.meter)
	return entry, nil, err
}

func (r *providerFrontierReplay) checkIndexObjects(s *providerCustodyStore, delta *providerIndexDelta, initial bool) error {
	if initial {
		for _, expected := range r.anchors {
			raw, err := s.object(expected.kind, expected.digest)
			if err != nil || !providerReplayEqual(raw, providerReplayPhaseBytes(expected.raw, r.meter), r.meter) {
				return errProviderIntegration
			}
		}
	}
	for _, mutation := range delta.mutations {
		expected := mutation.object
		raw, err := s.object(expected.kind, expected.digest)
		if err != nil || !providerReplayEqual(raw, providerReplayPhaseBytes(expected.raw, r.meter), r.meter) {
			return errProviderIntegration
		}
	}
	return nil
}

func (r *providerFrontierReplay) setupAtBoundary(s *providerCustodyStore, clock CandidateClockDiagnostic) (providerSetupReference, error) {
	selected, found := s.setupForBoundary(s.tip)
	if !found && s.sequence != 0 {
		for _, setup := range s.setups {
			if setup.commit == r.current.setup {
				selected, found = setup, true
				break
			}
		}
	}
	if !found || selected.ordinal < 0 || selected.ordinal < r.setupOrdinal || selected.ordinal >= len(s.setups) || !candidateLedgerClockAdvance(selected.clock, clock) {
		return providerSetupReference{}, errProviderIntegration
	}
	for ordinal := r.setupOrdinal + 1; ordinal <= selected.ordinal; ordinal++ {
		setup := s.setups[ordinal]
		if setup.ordinal != ordinal || setup.expectedTip != s.tip || !candidateLedgerClockAdvance(setup.clock, clock) || s.sequence != 0 && !candidateLedgerClockAdvance(r.current.clock, setup.clock) {
			return providerSetupReference{}, errProviderIntegration
		}
	}
	return selected, nil
}

func (r *providerFrontierReplay) setupBytes(s *providerCustodyStore, through int) (uint64, bool) {
	// Immutable selected anchor exists in both stores; setup claim/commit are
	// actual retained original files, not a multiplied reservation-as-actual.
	if through < r.setupOrdinal || through >= len(s.setups) {
		return 0, false
	}
	var total uint64
	if r.setupOrdinal < 0 {
		anchor, err := s.read(s.evidence, "ANCHOR", int64(s.config.bounds.maximumMetadataPerCommit))
		if err != nil {
			return 0, false
		}
		total = uint64(2 * len(anchor))
	}
	for ordinal := r.setupOrdinal + 1; ordinal <= through; ordinal++ {
		setup := s.setups[ordinal]
		claim, a := s.read(s.evidence, s.name("setup-claim", uint64(ordinal)), int64(s.config.bounds.maximumMetadataPerCommit))
		intent, b := s.read(s.custody, s.name("setup-intent", uint64(ordinal)), int64(s.config.bounds.maximumMetadataPerCommit))
		commit, c := s.read(s.custody, s.name("setup-commit", uint64(ordinal)), int64(s.config.bounds.maximumMetadataPerCommit))
		previous := s.config.anchor
		if ordinal != 0 {
			previous = s.setups[ordinal-1].commit
		}
		config := s.config
		config.attempt, config.expectedTip, config.expectedSetupTip = ordinal, setup.expectedTip, previous
		clock := ClockSample{ClockID: setup.clock.ClockID, Epoch: setup.clock.Epoch, WallMS: setup.clock.WallMS, MonotonicMS: setup.clock.MonotonicMS, UncertaintyMS: setup.clock.UncertaintyMS}
		if a != nil || b != nil || c != nil || setup.ordinal != ordinal || !providerReplayEqual(claim, intent, r.meter) || !providerReplayEqual(claim, providerCustodySetupDiagnosticObserved(config, providerClockDiagnostic(clock), r.meter), r.meter) || !providerReplayEqual(commit, providerSetupCommitObserved(claim, r.meter), r.meter) || providerReplayDigest("sage-provider-authority-setup-v1", claim, r.meter) != setup.claim || providerReplayDigest("sage-provider-authority-setup-commit-v1", commit, r.meter) != setup.commit {
			return 0, false
		}
		var ok bool
		total, ok = providerUpperSum(total, uint64(len(claim)+len(intent)+len(commit)))
		if !ok {
			return 0, false
		}
	}
	return total, true
}

func (r *providerFrontierReplay) deriveFrontier(s *providerCustodyStore, root *CandidateEnrollmentRoot, delta *providerIndexDelta, commit []byte, claimed providerFrontierRecord, clock CandidateClockDiagnostic, setup string) (providerFrontierRecord, error) {
	initial := providerClockDiagnostic(s.setupOrigin)
	policy, policyOK := providerReplayJSON(providerReplayPhaseBytes(r.body.policy.raw, r.meter), r.meter).(map[string]any)
	if !policyOK {
		return providerFrontierRecord{}, errProviderIntegration
	}
	limits := policy["limits"].(map[string]any)
	duration, ok := providerCandidateUint(limits["bootstrap"].(map[string]any)["deadline_ms"], true)
	if !ok || initial.WallMS < initial.UncertaintyMS {
		return providerFrontierRecord{}, errProviderIntegration
	}
	deadline, ok := providerUpperSum(uint64(initial.WallMS-initial.UncertaintyMS), duration)
	if !ok {
		return providerFrontierRecord{}, errProviderIntegration
	}
	completed, commitOK := providerReplayJSON(commit, r.meter).(map[string]any)
	commitDigest := providerReplayDigest("sage-provider-custody-commit-v2", commit, r.meter)
	if !commitOK || commitDigest == "" {
		return providerFrontierRecord{}, errProviderIntegration
	}
	f := providerFrontierRecord{identity: s.config.identity, fence: root.fence, sequence: s.sequence + 1, commit: commitDigest, setup: setup, objects: contentString(completed["objects_root"]), enrollment: root.digest, indexes: delta.previous, initial: initial, clock: clock, deadline: deadline}
	if s.sequence != 0 {
		raw, err := r.current.bytesObserved(r.meter)
		if err != nil {
			return providerFrontierRecord{}, errProviderIntegration
		}
		f.previous = providerReplayDigest("sage-provider-custody-frontier-v1", raw, r.meter)
		f.seal, f.replay, f.checkpoint = r.current.seal, r.current.replay, r.current.checkpoint
	}
	if r.body.intent != nil {
		if r.body.intent.deadline != deadline || r.body.intent.clock != initial || r.body.ledger == nil || !candidateLedgerClockAdvance(r.body.ledger.current.clock, clock) {
			return providerFrontierRecord{}, errProviderIntegration
		}
		f.admission, f.budget = r.body.intent.digest, r.body.ledger.current.digest
	}
	if delta.kind == "seal" {
		f.seal = delta.entry.digest
	}
	// Until the closed checkpoint/receipt replay exists, a claimed nonnull new
	// replay/checkpoint cannot sneak through a presence-only frontier parser.
	if claimed.replay != f.replay || claimed.checkpoint != f.checkpoint {
		return providerFrontierRecord{}, errProviderIntegration
	}
	for _, node := range delta.mutations {
		f.indexes[node.chain] = node.object.digest
	}
	return f, nil
}
