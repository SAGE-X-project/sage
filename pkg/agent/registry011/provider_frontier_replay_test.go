package registry011

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// This unit reads normal original object files from an isolated owned fixture.
// The surrounding main commits/frontiers are INERT DATA, not actual external
// writes/CAS/current-tip authority. Advancing this fixture cannot yield an ACK,
// start a transport, execute an operation or qualify an operating Source.
type providerFrontierUnit struct {
	s       *providerCustodyStore
	r       *providerFrontierReplay
	indexes *providerLedgerIndexes
	root    *CandidateEnrollmentRoot
	ledger  *CandidateAdmissionLedger
	policy  *ProviderPolicyCandidate
	context *ProviderCandidateContext
	clock   ClockSample
	records []providerFrontierUnitRecord
	head    *providerFrontierHighwater
}

type providerFrontierUnitRecord struct {
	inventory                *providerIntegrationInventory
	fresh                    []providerIntegrationObject
	intent, commit, frontier []byte
}

func providerFrontierUnitFixture(t *testing.T) *providerFrontierUnit {
	return providerFrontierUnitFixtureWithClock(t, nil)
}

func providerFrontierUnitFixtureWithClock(t *testing.T, owned *providerOwnedClock) *providerFrontierUnit {
	return providerFrontierUnitFixtureMode(t, owned, false)
}

func providerFrontierUnitFixtureMode(t *testing.T, owned *providerOwnedClock, highwater bool) *providerFrontierUnit {
	t.Helper()
	model, policy := providerPrivateLedgerFixture(t, 8, 32)
	if owned != nil {
		// Select the existing normal owned-store fixture's five-minute envelope
		// BEFORE its new original clock/setup/I. The one-second arithmetic
		// fixture and all already failed runs remain unchanged, never renewed.
		model, policy = providerPrivateLedgerFixtureWithDeadline(t, 8, 32, 300000)
		if owned.policy != policy.policy {
			t.Fatal("normal owned fixture clock belongs to another policy")
		}
	}
	id := model.intent.identity
	context := &ProviderCandidateContext{id, true}
	rootObject := admissionTestRootObject(t)
	rootObject["policy"] = id.Policy
	root, err := ParseCandidateEnrollmentRoot(context, providerPolicyTestBytes(t, rootObject), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	b, capture := providerFrontierBoundsFixture()
	private := model.continuation.bounds
	b.continuation, b.maximumCommits, b.maximumObjects = &private, int(private.maximumCommits), int(private.maximumObjects)
	b.ledger = CandidateLedgerBounds{int(private.pageRecords), min(private.pagePayloadBytes, 16<<20), min(private.pageMetadataBytes, 64<<20)}
	empty, err := newProviderFrontierInventory("", nil, b.maximumEntries, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	profile := providerIntegrationProfile(b, capture)
	anchor := providerIntegrationJSON(map[string]any{"format": "sage-provider-custody-anchor-v1", "namespace": id.Namespace, "instance_key": id.Instance, "profile": profile, "initial_objects_root": empty.digest})
	anchorDigest := providerCandidateHash("sage-provider-custody-anchor-v1", anchor)
	directory := t.TempDir()
	config := providerCustodyConfig{identity: id, policy: policy, bounds: b, capture: capture, profile: profile, anchor: anchorDigest, expectedTip: anchorDigest, expectedSetupTip: anchorDigest, custodyDirectory: filepath.Join(directory, "custody"), evidenceDirectory: filepath.Join(directory, "evidence")}
	for _, name := range []string{config.custodyDirectory, config.evidenceDirectory} {
		if os.Mkdir(name, 0700) != nil || os.WriteFile(filepath.Join(name, "AUTHORITY.LOCK"), nil, 0600) != nil || os.WriteFile(filepath.Join(name, "ANCHOR"), anchor, 0600) != nil {
			t.Fatal("normal owned replay object fixture")
		}
	}
	custody, err := openRetentionDisk(config.custodyDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = custody.close() })
	evidence, err := openRetentionDisk(config.evidenceDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = evidence.close() })
	d := model.intent.clock
	clock := ClockSample(d) // Inert data; the optional owned backend samples itself.
	if owned != nil {
		clock, err = owned.sample()
		if err != nil {
			t.Fatal(err)
		}
		d = providerClockDiagnostic(clock)
	}
	s := &providerCustodyStore{config: config, custody: custody, evidence: evidence, clock: owned, tip: config.anchor, objectsRoot: empty.digest, setupOrigin: clock, objects: make(map[string]providerIntegrationObjectRef), operations: make(map[string]bool), bindings: make(map[string]string)}
	s.owner = s
	claim, setupCommit := providerCustodySetup(config, clock), []byte(nil)
	setupCommit = providerSetupCommit(claim)
	s.setupTip = providerCandidateHash("sage-provider-authority-setup-commit-v1", setupCommit)
	claimDigest := providerCandidateHash("sage-provider-authority-setup-v1", claim)
	s.setups = []providerSetupReference{{commit: s.setupTip, claim: claimDigest, expectedTip: s.tip, clock: d}}
	if custody.write(providerCustodyName("setup-intent", 0), claim) != nil || evidence.write(providerCustodyName("setup-claim", 0), claim) != nil || custody.write(providerCustodyName("setup-commit", 0), setupCommit) != nil {
		t.Fatal("normal finite setup fixture")
	}
	r, err := newProviderFrontierReplay(config)
	if err != nil {
		t.Fatal(err)
	}
	indexes, anchors, err := newProviderLedgerIndexes(id, 1, private)
	if err != nil {
		t.Fatal(err)
	}
	u := &providerFrontierUnit{s: s, r: r, indexes: indexes, root: root, policy: policy, context: context, clock: clock}
	if highwater {
		h := providerFrontierHighwater{identity: id, fence: 1, profile: config.profile, commit: config.anchor, setup: s.setupTip}
		raw, err := h.bytes(config)
		if err != nil || custody.write(providerCustodyName("highwater", 0), raw) != nil {
			t.Fatal("normal immutable high-water DATA fixture", err)
		}
		u.head = &h
	}
	inputs := []providerIntegrationObjectInput{{"enrollment", root.CanonicalBytes()}, {"setup", claim}, {"setup-commit", setupCommit}}
	for _, anchor := range anchors {
		inputs = append(inputs, providerIntegrationObjectInput{anchor.kind, []byte(anchor.raw)})
	}
	operation := providerCandidateHash("sage-provider-enroll-operation-v1", []byte(config.anchor+id.Instance+id.Policy+id.SourceID))
	u.applyData(t, "enroll", operation, root, "enrollment", root.CanonicalBytes(), inputs, 0)
	return u
}

func (u *providerFrontierUnit) applyData(t *testing.T, kind, operation string, root *CandidateEnrollmentRoot, entryKind string, entryRaw []byte, inputs []providerIntegrationObjectInput, deltaMS int64) {
	t.Helper()
	entry, err := newProviderIntegrationObject(entryKind, entryRaw, 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	if u.ledger != nil && u.indexes.ledger == nil {
		if err := u.indexes.bindRetainedLedger(u.ledger, u.r.preI); err != nil {
			t.Fatal(err)
		}
	}
	delta, err := u.indexes.prepare(kind, operation, u.s.sequence+1, entry, root, u.ledger)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutation := range delta.mutations {
		inputs = append(inputs, providerIntegrationObjectInput{mutation.object.kind, []byte(mutation.object.raw)})
	}
	fresh := make([]providerIntegrationObject, 0, len(inputs))
	seen := make(map[string]bool)
	for _, input := range inputs {
		object, err := newProviderIntegrationObject(input.kind, input.raw, u.s.config.bounds.maximumObjectBytes)
		if err != nil {
			t.Fatal(err)
		}
		key := object.kind + ":" + object.digest
		if _, old := u.s.objects[key]; old || seen[key] {
			continue
		}
		seen[key] = true
		fresh = append(fresh, object)
		if u.s.evidence.write(providerObjectName(object.kind, object.digest), input.raw) != nil {
			t.Fatal("normal immutable unit object")
		}
		u.s.objects[key] = providerIntegrationObjectRef{object.kind, object.digest, uint64(len(object.raw))}
	}
	inventory, err := newProviderFrontierInventory(u.s.objectsRoot, fresh, u.s.config.bounds.maximumEntries, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	sample := u.clock
	sample.WallMS, sample.MonotonicMS = sample.WallMS+deltaMS, sample.MonotonicMS+deltaMS
	link := providerFrontierLink{identity: root.identity, fence: 1, sequence: u.s.sequence, commit: u.s.tip, setup: u.s.setupTip, enrollment: u.s.enrollment, objects: u.s.objectsRoot}
	previous := ""
	if u.s.sequence != 0 {
		raw, err := u.r.current.bytes()
		if err != nil {
			t.Fatal(err)
		}
		previous = providerCandidateHash("sage-provider-custody-frontier-v1", raw)
		link.frontier = previous
	}
	intent, err := providerFrontierIntent(link, root, inventory, kind, operation, sample, u.s.config.bounds)
	if err != nil {
		t.Fatal(err)
	}
	commit := providerFrontierCommit(intent)
	p := contentJSON([]byte(u.policy.raw)).(map[string]any)
	duration, _ := providerCandidateUint(p["limits"].(map[string]any)["bootstrap"].(map[string]any)["deadline_ms"], true)
	f := providerFrontierRecord{identity: root.identity, fence: 1, sequence: u.s.sequence + 1, previous: previous, commit: providerCandidateHash("sage-provider-custody-commit-v2", commit), setup: u.s.setupTip, enrollment: root.digest, objects: inventory.digest, indexes: delta.previous, initial: providerClockDiagnostic(u.clock), clock: providerClockDiagnostic(sample), deadline: uint64(u.clock.WallMS-u.clock.UncertaintyMS) + duration}
	if u.ledger != nil {
		f.admission, f.budget = u.ledger.intent.digest, u.ledger.current.digest
	}
	f.seal = u.r.current.seal
	if kind == "seal" {
		f.seal = entry.digest
	}
	for _, mutation := range delta.mutations {
		f.indexes[mutation.chain] = mutation.object.digest
	}
	frontier, err := f.bytes()
	if err != nil {
		t.Fatal(err)
	}
	var cost providerFrontierRetention
	if u.head == nil {
		cost, err = u.r.apply(u.s, inventory, fresh, intent, commit, frontier)
	} else {
		before, e := u.head.bytes(u.s.config)
		if e != nil {
			t.Fatal(e)
		}
		after := *u.head
		after.ordinal, after.sequence, after.previous = after.ordinal+1, f.sequence, providerCandidateHash("sage-provider-custody-highwater-v1", before)
		after.commit, after.frontier = f.commit, providerCandidateHash("sage-provider-custody-frontier-v1", frontier)
		cost, err = u.r.applyWithHighwater(u.s, inventory, fresh, intent, commit, frontier, after)
		raw, e := after.bytes(u.s.config)
		if err == nil && (e != nil || u.s.custody.write(providerCustodyName("highwater", after.ordinal), raw) != nil) {
			t.Fatal("normal immutable high-water DATA fixture", e)
		}
		u.head = &after
	}
	if err != nil {
		t.Fatal("independent retained-data reconstruction", kind, err)
	}
	if kind == "enroll" {
		err = u.indexes.apply(delta)
	} else {
		err = u.indexes.applyRetained(delta, cost)
	}
	if err != nil || u.indexes.heads != u.r.indexes.heads {
		t.Fatal("independently reconstructed index heads differ", err)
	}
	u.records = append(u.records, providerFrontierUnitRecord{inventory, fresh, intent, commit, frontier})
	// Advance only inert test metadata. No on-disk main write, high-water/CAS,
	// custody ACK or execution permission is manufactured by this unit.
	u.s.tip, u.s.enrollment, u.s.objectsRoot, u.s.sequence = f.commit, root.digest, inventory.digest, f.sequence
	u.root = root
}

func TestProviderFrontierReplayIndependentlyRebuildsEnrollmentAndFirstTicket(t *testing.T) {
	u := providerFrontierUnitFixture(t)
	if u.r.body.anchor.digest != u.root.digest || u.r.body.intent != nil || u.r.body.ledger != nil || u.r.current.admission != "" || u.r.indexes.counts != [3]uint64{0, 1, 0} || u.r.preI.metadata == 0 {
		t.Fatal("initial reconstruction invented future I/budget")
	}
	ticket := u.firstTicket(t, 1)
	if u.r.body.ledger == u.ledger || u.r.indexes == u.indexes || u.r.body.ledger.intent.digest != u.ledger.intent.digest || u.r.body.ledger.current.digest != u.ledger.current.digest || u.r.body.ledger.current.privatePayload != u.ledger.current.privatePayload || u.r.body.ledger.current.privateMetadata != u.ledger.current.privateMetadata || u.r.current.budget != u.ledger.current.digest || u.r.indexes.tickets[ticket.digest].consumption != "" {
		t.Fatal("replay trusted coordinator maps, lost original bytes or consumed work")
	}
}

func (u *providerFrontierUnit) firstTicket(t *testing.T, deltaMS int64) *CandidateAdmissionTicket {
	t.Helper()
	c := &providerIntegration{store: u.s, policy: u.policy, context: u.context, anchor: u.root}
	if err := c.makeIntent(u.s.tip); err != nil {
		t.Fatal(err)
	}
	request, err := newProviderIntegrationRequest("eth_getBlockByNumber", []any{"finalized", false})
	if err != nil {
		t.Fatal(err)
	}
	_, _, ticketObject := admissionTestTicketFixture(t)
	ticketObject["admission_digest"] = c.intent.digest
	ticketObject["namespace"], ticketObject["instance_key"], ticketObject["source_id"] = c.intent.identity.Namespace, c.intent.identity.Instance, c.intent.identity.SourceID
	ticketObject["segment"], ticketObject["input_digest"], ticketObject["params_digest"] = candidateAdmissionSegment(c.intent.digest, 0), request.inputDigest, request.paramsDigest
	ticketObject["deadline_wall_ms"] = strconv.FormatUint(c.intent.deadline, 10)
	ticket := admissionTestTicket(t, u.policy, c.intent, 0, ticketObject)
	u.ledger, err = newProviderContinuingLedger(u.policy, c.intent, ticket, *u.s.config.bounds.continuation)
	if err != nil {
		t.Fatal(err)
	}
	next := providerIndexRoot(t, u.root, u.ledger)
	inputs := []providerIntegrationObjectInput{{"admission", []byte(c.intent.raw)}, {"ticket", ticket.CanonicalBytes()}, {"request", []byte(request.wire)}, {"params", []byte(request.params)}, {"enrollment", next.CanonicalBytes()}, {"budget", u.ledger.current.BudgetBytes()}}
	for _, node := range u.ledger.current.LedgerNodeBytes() {
		inputs = append(inputs, providerIntegrationObjectInput{"ledger-node", node})
	}
	u.applyData(t, "admit", ticket.digest, next, "ticket", ticket.CanonicalBytes(), inputs, deltaMS)
	return ticket
}

func TestProviderFrontierReplayHistoricalEnrollmentExcludesLaterSetup(t *testing.T) {
	u := providerFrontierUnitFixture(t)
	initialPreI := u.r.preI
	firstSetup := u.s.setupTip
	sample := u.clock
	sample.WallMS++
	sample.MonotonicMS++
	config := u.s.config
	config.attempt, config.expectedTip, config.expectedSetupTip = 1, u.s.tip, firstSetup
	claim := providerCustodySetup(config, sample)
	commit := providerSetupCommit(claim)
	setup := providerSetupReference{commit: providerCandidateHash("sage-provider-authority-setup-commit-v1", commit), claim: providerCandidateHash("sage-provider-authority-setup-v1", claim), expectedTip: u.s.tip, ordinal: 1, clock: providerClockDiagnostic(sample)}
	if u.s.custody.write(providerCustodyName("setup-intent", 1), claim) != nil || u.s.evidence.write(providerCustodyName("setup-claim", 1), claim) != nil || u.s.custody.write(providerCustodyName("setup-commit", 1), commit) != nil {
		t.Fatal("normal later setup fixture")
	}
	u.s.setups, u.s.setupTip = append(u.s.setups, setup), setup.commit
	// A newly reconstructed view sees the complete setup prefix, including the
	// later one, while replaying the historical original E0. It must select the
	// setup belonging to that old main boundary instead of the final setupTip.
	r, err := newProviderFrontierReplay(u.s.config)
	if err != nil {
		t.Fatal(err)
	}
	empty, err := newProviderFrontierInventory("", nil, u.s.config.bounds.maximumEntries, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	view := &providerCustodyStore{config: u.s.config, custody: u.s.custody, evidence: u.s.evidence, tip: u.s.config.anchor, objectsRoot: empty.digest, setupTip: setup.commit, setups: append([]providerSetupReference(nil), u.s.setups...), setupOrigin: u.s.setupOrigin, objects: make(map[string]providerIntegrationObjectRef), bindings: make(map[string]string), operations: make(map[string]bool)}
	for key, value := range u.s.objects {
		view.objects[key] = value
	}
	e0 := u.records[0]
	if _, err := r.apply(view, e0.inventory, e0.fresh, e0.intent, e0.commit, e0.frontier); err != nil || r.current.setup != firstSetup || r.setupOrdinal != 0 || r.preI != initialPreI {
		t.Fatal("future setup changed historical E0 or its pre-I accounting", err)
	}
	view.tip, view.enrollment, view.objectsRoot, view.sequence = r.current.commit, r.current.enrollment, r.current.objects, r.current.sequence
	selected, err := r.setupAtBoundary(view, CandidateClockDiagnostic{ClockID: sample.ClockID, Epoch: sample.Epoch, WallMS: sample.WallMS + 1, MonotonicMS: sample.MonotonicMS + 1, UncertaintyMS: sample.UncertaintyMS})
	if err != nil || selected.commit != setup.commit {
		t.Fatal("next main boundary failed to select its new setup", err)
	}
	added, ok := r.setupBytes(view, selected.ordinal)
	if !ok || added != uint64(2*len(claim)+len(commit)) {
		t.Fatal("later setup raw bytes omitted or charged twice")
	}
	u.firstTicket(t, 2)
	first := u.records[1]
	for _, object := range first.fresh {
		view.objects[object.kind+":"+object.digest] = providerIntegrationObjectRef{object.kind, object.digest, uint64(len(object.raw))}
	}
	if _, err := r.apply(view, first.inventory, first.fresh, first.intent, first.commit, first.frontier); err != nil || r.setupOrdinal != 1 || r.current.setup != setup.commit || r.preI != initialPreI || r.body.ledger.current.privateMetadata != u.ledger.current.privateMetadata || r.body.ledger.current.privatePayload != u.ledger.current.privatePayload {
		t.Fatal("reconstructed later setup changed I/metadata/payload lineage", err)
	}
	if added, ok := r.setupBytes(view, selected.ordinal); !ok || added != 0 {
		t.Fatal("setup metadata was replenished after a main transition")
	}
}

func TestProviderFrontierReplayRefusesUnqualifiedOrMalformedOwner(t *testing.T) {
	u := providerFrontierUnitFixture(t)
	if _, err := newProviderFrontierReplay(providerCustodyConfig{}); err == nil {
		t.Fatal("zero config created replay")
	}
	for _, r := range []*providerFrontierReplay{nil, {}, {owner: u.r, body: u.r.body, indexes: u.r.indexes}} {
		if _, err := r.apply(u.s, nil, nil, nil, nil, nil); err == nil {
			t.Fatal("invalid owner accepted delta")
		}
	}
	if _, err := u.r.apply(u.s, nil, nil, nil, nil, nil); err == nil || !u.r.failed {
		t.Fatal("incomplete retained delta remained reusable")
	}
}

func (u *providerFrontierUnit) transitionData(t *testing.T, kind, entryKind string, raw []byte, delta int64) {
	t.Helper()
	entry, err := newProviderIntegrationObject(entryKind, raw, 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	next := providerIndexRoot(t, u.root, u.ledger)
	inputs := []providerIntegrationObjectInput{{entryKind, raw}, {"enrollment", next.CanonicalBytes()}, {"budget", u.ledger.current.BudgetBytes()}}
	for _, node := range u.ledger.current.LedgerNodeBytes() {
		inputs = append(inputs, providerIntegrationObjectInput{"ledger-node", node})
	}
	u.applyData(t, kind, entry.digest, next, entryKind, raw, inputs, delta)
}

func TestProviderFrontierReplayReadsOldOriginalsThroughConsumptionSettlementAndSeal(t *testing.T) {
	u := providerFrontierUnitFixture(t)
	ticket := u.firstTicket(t, 1)
	consumption := ledgerTestConsumption(t, ticket, nil, 10)
	if _, err := u.ledger.AppendCandidateConsumption(consumption); err != nil {
		t.Fatal(err)
	}
	u.transitionData(t, "consume", "consumption", []byte(consumption.raw), 11)
	settlement := ledgerTestSettlement(t, ticket, "failed", 20)
	if _, err := u.ledger.AppendCandidateSettlement(settlement); err != nil {
		t.Fatal(err)
	}
	u.transitionData(t, "settle", "settlement", settlement.CanonicalBytes(), 21)
	if u.r.body.ledger.current.counts != [3]uint64{1, 1, 1} || u.r.body.ledger.current.totals.bootstrap.charged != ticket.work || u.r.body.ledger.current.totals.bootstrap.outstanding != (candidateWorkVector{}) || u.r.indexes.tickets[ticket.digest].consumption != consumption.digest || u.r.indexes.tickets[ticket.digest].settlement != settlement.digest {
		t.Fatal("independent reconstruction lost full charge or terminal originals")
	}
	heads := u.indexes.heads
	boundary, err := u.ledger.prepareProviderBoundary(true)
	if err != nil {
		t.Fatal(err)
	}
	seal, err := boundary.sealObject(heads)
	if err != nil {
		t.Fatal(err)
	}
	next := providerIndexRoot(t, u.root, u.ledger)
	inputs := []providerIntegrationObjectInput{{"enrollment", next.CanonicalBytes()}, {"budget", u.ledger.current.BudgetBytes()}, {"ledger-page", []byte(boundary.pageObject.raw)}, {"ledger-seal", []byte(seal.raw)}}
	u.applyData(t, "seal", admissionTestDigest("e"), next, "ledger-seal", []byte(seal.raw), inputs, 30)
	if err := u.ledger.applyProviderBoundary(boundary, []byte(boundary.pageObject.raw), []byte(seal.raw), heads); err != nil {
		t.Fatal(err)
	}
	if !u.r.body.ledger.continuation.final || !u.ledger.continuation.final || u.r.current.seal != seal.digest || u.r.body.ledger.current.privateMetadata != u.ledger.current.privateMetadata || u.r.body.ledger.current.privatePayload != u.ledger.current.privatePayload || u.r.current.checkpoint != "" {
		t.Fatal("seal manufactured a checkpoint or changed common accounting")
	}
}

func TestProviderFrontierOldReferenceRefusalUsesInertMapStateOnly(t *testing.T) {
	u := providerFrontierUnitFixture(t)
	anchorKey := "enrollment:" + u.root.digest
	anchorRef := u.s.objects[anchorKey]
	delete(u.s.objects, anchorKey)
	if u.r.checkOldMaterials(u.s, "admit", admissionTestDigest("e")) == nil {
		t.Fatal("first I/T used its cached E0 without the retained original reference")
	}
	u.s.objects[anchorKey] = anchorRef
	if err := u.r.checkOldMaterials(u.s, "admit", admissionTestDigest("e")); err != nil {
		t.Fatal("normal original E0 before the first I/T", err)
	}
	ticket := u.firstTicket(t, 1)
	// Remove only a diagnostic reference from this isolated in-memory unit
	// view. No retained file, real filesystem boundary or transport is attacked.
	key := "ticket:" + ticket.digest
	ref := u.s.objects[key]
	delete(u.s.objects, key)
	if u.r.checkOldMaterials(u.s, "admit", admissionTestDigest("e")) == nil {
		t.Fatal("private cached ticket replaced its missing original reference")
	}
	u.s.objects[key] = ref
	if err := u.r.checkOldMaterials(u.s, "admit", admissionTestDigest("e")); err != nil {
		t.Fatal("normal original references", err)
	}
}
