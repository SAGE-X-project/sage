package registry011

import (
	"context"
	"os"
	"reflect"
)

// This private binding is made ONLY from the two completed original semantic
// replays in dispatch preparation. It is registered in both original owners.
// It is neither a serialized replay proof nor a request/transport permission.
// Each use still re-reads EVERY recorded original independently for each role.
type providerHistoryMemberReplaySeal struct {
	owner                            *providerHistoryMemberReplaySeal
	dispatch                         *providerHistoryGroupDispatch
	recovery                         *providerHistoryRecoveryOwner
	reads                            [2]*providerHistoryReplayReads
	readRoots                        [2]string
	models                           [2]*providerHistoryGroupLedger
	modelRoots                       [2]string
	fixed                            providerHistoryMemberReplayContext
	consumption                      providerHistoryRecoveryCheckpoint
	admission                        providerHistoryRecoveryCheckpoint
	cutoff                           providerHistoryRecoveryCheckpoint
	initial                          *providerHistoryRecoveryCheckpoint
	ancestors                        []providerHistoryMemberAncestor
	fixedComparison, frameComparison uint64
	readContexts                     [2]providerHistoryMemberReadContext
	readComparisons                  [2]uint64
	claims                           [2]providerHistoryMemberClaimInput
	claimComparisons                 [2]uint64
	revoked                          bool
}

type providerHistoryMemberReadContext struct {
	config                   providerCustodyConfig
	program, costs           string
	initial, current, cutoff providerFrontierHighwater
}

func providerHistoryMemberReadContextLocked(p *providerHistoryReplayReads) providerHistoryMemberReadContext {
	return providerHistoryMemberReadContext{p.config, p.program, p.costs, p.initial, p.current, p.cutoffPin}
}

// Only immutable selections are captured. Meter actuals, clock reads, retained
// member count, sent/completed bits and live retention MUST grow normally.
// The C checkpoint's retention is fixed and is distinct from live retention.
type providerHistoryMemberReplayContext struct {
	config            providerCustodyConfig
	policy            ProviderPolicyCandidate
	bounds            providerHistoryProgramBounds
	scan              providerScanBounds
	continuation      providerLedgerBounds
	methods           providerMethodBounds
	tls               providerTLSBounds
	first             providerFirstAdmissionBounds
	limits            providerFirstAdmissionLimits
	dispatch          providerHistoryDispatchBounds
	clock             *providerOwnedClock
	clockSelection    providerHistoryClockSelection
	original          ClockSample
	held, preI        providerScanUpper
	initialRead       uint64
	clockCeiling      uint64
	start             providerHistoryProgramStart
	scope             providerHistoryCustodyScope
	windowStart       candidateWorkVector
	windowUpper       candidateWorkVector
	clockStart        uint64
	clockUpper        uint64
	collectionInput   providerHistoryCollectionInput
	collectionRaw     string
	collectionDigest  string
	collectionCounts  providerHistoryCollectionCounts
	derivation        providerHistoryProgramDerivation
	manifest          providerHistoryCollectionManifest
	frontierFiles     *providerFrontierFiles
	frontierValue     providerFrontierFiles
	groupModels       [2]*providerHistoryGroupLedger
	custody, evidence *retentionDisk
	custodyRoot       *os.Root
	evidenceRoot      *os.Root
	custodyLock       *os.File
	evidenceLock      *os.File
}

func (*providerHistoryMemberReplaySeal) UnmarshalJSON([]byte) error { return errProviderIntegration }

func providerHistoryMemberReplayContextLocked(o *providerHistoryGroupOwner) providerHistoryMemberReplayContext {
	p := o.parent
	f := providerHistoryMemberReplayContext{config: p.config, policy: p.policy, bounds: p.bounds,
		scan: p.scan, continuation: p.continuation, methods: p.methods, tls: p.tls, first: p.first,
		limits: p.limits, dispatch: p.dispatchBounds, clock: p.clock, clockSelection: p.clockSelection,
		original: p.original, held: p.held, preI: p.preI, initialRead: p.initialRead, clockCeiling: p.clockCeiling,
		start: *p.start, scope: o.scope, windowStart: o.windowStart, windowUpper: o.windowUpper,
		clockStart: o.clockStart, clockUpper: o.clockUpper, collectionInput: p.collectionInput,
		collectionRaw: p.collectionRaw, collectionDigest: p.collectionDigest, collectionCounts: p.collectionCounts,
		frontierFiles: o.store.frontierFiles, frontierValue: *o.store.frontierFiles,
		groupModels: [2]*providerHistoryGroupLedger{o.coordinator, o.custodian},
		custody:     o.store.custody, evidence: o.store.evidence, custodyRoot: o.store.custody.root,
		evidenceRoot: o.store.evidence.root, custodyLock: o.store.custody.lock, evidenceLock: o.store.evidence.lock}
	if p.bounds.derivation != nil {
		f.derivation = *p.bounds.derivation
	}
	if p.bounds.collection != nil {
		f.manifest = *p.bounds.collection
	}
	return f
}

// Original-pointer/numeric selection only: raw-bearing value comparisons live
// in explicit observed boundaries, outside scalar meter selection and its lock.
func (q *providerHistoryMemberReplaySeal) contextMatchesLocked(d *providerHistoryGroupDispatch) bool {
	if q == nil || q.owner != q || q.revoked || d == nil || d.owner != d || q.dispatch != d || d.group == nil ||
		d.group.parent == nil || d.group.parent.start == nil || d.group.parent.recovery != q.recovery ||
		d.acknowledgment == nil || d.group.store == nil || d.group.store.custody == nil || d.group.store.evidence == nil ||
		q.recovery == nil || q.recovery.owner != q.recovery || q.recovery.originalGroup != d.group ||
		q.recovery.originalMeter != d.meter || d.meter != d.group.meter ||
		q.recovery.consumption == nil || q.recovery.confirmed != q.recovery.consumption ||
		q.recovery.consumption != q.consumption.owner || q.consumption.ack != d.acknowledgment ||
		q.consumption.claim != d.acknowledgment.claim || q.consumption.previous != q.admission.owner ||
		d.group.stageBefore == nil || d.group.stageBefore != q.cutoff.owner ||
		d.group.parent.start != q.fixed.start.owner || d.group.parent.bounds.derivation != q.fixed.bounds.derivation ||
		d.group.parent.bounds.collection != q.fixed.bounds.collection || !d.group.window {
		return false
	}
	return d.group.windowStart == q.fixed.windowStart && d.group.windowUpper == q.fixed.windowUpper &&
		d.group.clockStart == q.fixed.clockStart && d.group.clockUpper == q.fixed.clockUpper &&
		d.group.start == d.group.parent.start && d.group.store.frontierFiles == q.fixed.frontierFiles &&
		[2]*providerHistoryGroupLedger{d.group.coordinator, d.group.custodian} == q.fixed.groupModels &&
		d.group.admission == q.admission.ack && d.group.last == q.consumption.ack &&
		d.coordinator == q.models[0] && d.custodian == q.models[1] && q.models[0] != q.models[1] &&
		d.replayReads == q.reads && q.reads[0] != nil && q.reads[1] != nil && q.reads[0] != q.reads[1]
}

func (q *providerHistoryMemberReplaySeal) fixedContextMatchesObservedLocked(m *providerWorkMeter) bool {
	return q != nil && q.contextMatchesLocked(q.dispatch) &&
		providerHistoryMemberEqualObserved(providerHistoryMemberReplayContextLocked(q.dispatch.group), q.fixed, q.fixedComparison, m)
}

func (d *providerHistoryGroupDispatch) memberReplaySelectedLocked() bool {
	if d == nil || d.memberReplaySeal == nil || d.group == nil || d.group.parent == nil {
		return false
	}
	q := d.memberReplaySeal
	return q.contextMatchesLocked(d) && q.initial != nil && q.recovery.initial == q.initial && len(q.ancestors) > 0 &&
		q.recovery.originalDispatch == d && q.recovery.memberReplaySeal == q
}

// All fallible creation and original C authentication precede dispatch's final
// native sample. Publishing the two registration pointers afterwards is scalar.
func newProviderHistoryMemberReplaySealLocked(d *providerHistoryGroupDispatch) (*providerHistoryMemberReplaySeal, error) {
	if d == nil || d.group == nil || d.group.parent == nil || d.group.parent.recovery == nil ||
		d.group.phase != "preparing-dispatch" || !d.group.activeLocked(d.meter) ||
		d.group.parent.recovery.originalDispatch != nil || d.group.parent.recovery.memberReplaySeal != nil ||
		d.group.parent.start == nil || d.group.parent.bounds.derivation == nil {
		return nil, errProviderIntegration
	}
	o, m, r := d.group, d.meter, d.group.parent.recovery
	if r.consumption == nil || r.consumption.previous == nil || o.stageBefore == nil ||
		!m.event(providerCopyEvent, uint64(reflect.TypeFor[providerHistoryMemberReplaySeal]().Size())) {
		return nil, errProviderIntegration
	}
	q := &providerHistoryMemberReplaySeal{dispatch: d, recovery: r, reads: d.replayReads,
		models: [2]*providerHistoryGroupLedger{d.coordinator, d.custodian}, fixed: providerHistoryMemberReplayContextLocked(o),
		consumption: *r.consumption, admission: *r.consumption.previous, cutoff: *o.stageBefore}
	q.owner = q
	var ok bool
	q.fixedComparison, ok = providerHistoryMemberComparisonWeightObserved(q.fixed, m)
	if !ok {
		return nil, errProviderIntegration
	}
	q.frameComparison, ok = providerHistoryMemberComparisonWeightObserved(q.consumption.frame, m)
	if !ok {
		return nil, errProviderIntegration
	}
	for role, p := range q.reads {
		q.readContexts[role] = providerHistoryMemberReadContext{q.fixed.config, q.fixed.bounds.program, q.fixed.bounds.costs,
			q.fixed.start.main, q.consumption.pin, q.cutoff.pin}
		q.readComparisons[role], ok = providerHistoryMemberComparisonWeightObserved(q.readContexts[role], m)
		if !ok || !q.readMatchesLocked(p, role) ||
			!providerHistoryMemberEqualObserved(providerHistoryMemberReadContextLocked(p), q.readContexts[role], q.readComparisons[role], m) ||
			!q.models[role].validObserved(m) {
			return nil, errProviderIntegration
		}
		q.readRoots[role], q.modelRoots[role] = p.root, p.model
		if !lowerHash(p.root) || !providerReplayTextEqual(p.root, p.dataRootObserved(m), m) ||
			!providerReplayTextEqual(p.model, q.models[role].head, m) {
			return nil, errProviderIntegration
		}
	}
	if !q.contextMatchesLocked(d) || !q.fixedContextMatchesObservedLocked(m) || !q.captureClaimsObservedLocked(m) || !q.captureAncestorsObservedLocked(m) || q.modelRoots[0] != q.modelRoots[1] {
		return nil, errProviderIntegration
	}
	root, err := q.currentConsumptionRootLocked()
	if err != nil || root != q.modelRoots[0] || m.failed || m.closed {
		return nil, errProviderIntegration
	}
	return q, nil
}

func (q *providerHistoryMemberReplaySeal) readMatchesLocked(p *providerHistoryReplayReads, role int) bool {
	if q == nil || q.dispatch == nil || q.dispatch.group == nil || role < 0 || role > 1 ||
		p == nil || p.owner != p || !p.complete || p.failed || p.context != nil {
		return false
	}
	o := q.dispatch.group
	return p == q.reads[role] && int(p.role) == role && p.group == o && p.meter == q.dispatch.meter &&
		p.cutoff == o.stageBefore && p.ack == q.consumption.ack
}

// Authenticate the actual current C bytes, rather than trusting a matching
// cache/model root. The independently retained checkpoint fixes the expected
// pin, original scope and model. No new baseline is adopted from current bytes.
func (q *providerHistoryMemberReplaySeal) currentConsumptionRootLocked() (string, error) {
	if q == nil || q.dispatch == nil || q.dispatch.group == nil {
		return "", errProviderIntegration
	}
	d, o := q.dispatch, q.dispatch.group
	s, m := o.store, d.meter
	if !q.contextMatchesLocked(d) || !o.activeLocked(m) {
		return "", errProviderIntegration
	}
	c := q.consumption
	maximum := int64(candidateLedgerMetadataLimit)
	head, a := s.read(s.custody, "HEAD", maximum)
	immutable, b := s.read(s.custody, s.name("highwater", c.pin.ordinal), maximum)
	before, e := s.read(s.custody, s.name("highwater", q.admission.pin.ordinal), maximum)
	if a != nil || b != nil || e != nil || !providerReplayTextEqual(providerReplayString(head, m), c.head, m) ||
		!providerReplayEqual(head, immutable, m) {
		return "", errProviderIntegration
	}
	currentPin, err := parseProviderFrontierHighwaterObserved(head, q.fixed.config, m)
	beforePin, beforeErr := parseProviderFrontierHighwaterObserved(before, q.fixed.config, m)
	if err != nil || beforeErr != nil || currentPin != c.pin || beforePin != q.admission.pin {
		return "", errProviderIntegration
	}
	intent, a := s.read(s.custody, s.name("intent", c.pin.sequence), providerHistoryCustodyFrameBytes)
	pending, b := s.read(s.evidence, s.name("history-pending", c.pin.sequence), providerHistoryCustodyFrameBytes)
	commit, e := s.read(s.custody, s.name("commit", c.pin.sequence), providerHistoryCustodyFrameBytes)
	frontier, f := s.read(s.custody, s.name("frontier", c.pin.sequence), providerHistoryCustodyFrameBytes)
	frame, err := parseProviderHistoryCustodyFrameObserved(q.fixed.config, q.fixed.scope, q.admission.pin, c.pin, intent, commit, frontier, m)
	if a != nil || b != nil || e != nil || f != nil || err != nil || !providerReplayEqual(intent, pending, m) ||
		!providerHistoryMemberEqualObserved(*frame, c.frame, q.frameComparison, m) || frame.phase != "consume" || frame.stage != d.coordinator.ticket.table.stage ||
		frame.model != c.model || frame.previousModel != q.admission.model ||
		frame.previousInventory != q.admission.inventory.digest || frame.inventory != c.inventory.digest ||
		frame.operation != d.coordinator.consumption.digest {
		return "", errProviderIntegration
	}
	return frame.model, nil
}

// This is the ONLY semantic replay substitution, in the two member preparation
// calls for the SAME live C. Collision resistance of the existing SHA-256
// primitive binds every fresh raw input to the ordered original full pass.
// All fixed behavioral context and actual C model are checked separately.
// S, failed S, advance, recovery and final scan retain full semantic replay.
func (d *providerHistoryGroupDispatch) replayMemberEquivalentLocked(ctx context.Context, role int) (*providerHistoryGroupLedger, error) {
	if ctx == nil || ctx.Err() != nil || role < 0 || role > 1 || !d.memberReplaySelectedLocked() {
		return nil, errProviderIntegration
	}
	q, o, m := d.memberReplaySeal, d.group, d.meter
	p := q.reads[role]
	if o.phase != "preparing-member" || !o.activeLocked(m) || m.replayReads != nil || !q.readMatchesLocked(p, role) ||
		!o.identityLocked() || !d.memberCatalogLocked() ||
		!q.fixedContextMatchesObservedLocked(m) || !q.claimsMatchObservedLocked(m) || !q.ancestorsMatchObservedLocked(m) ||
		!providerHistoryMemberEqualObserved(providerHistoryMemberReadContextLocked(p), q.readContexts[role], q.readComparisons[role], m) ||
		!providerReplayTextEqual(p.root, q.readRoots[role], m) || !providerReplayTextEqual(p.dataRootObserved(m), q.readRoots[role], m) {
		return nil, errProviderIntegration
	}
	for _, row := range p.rows {
		if ctx.Err() != nil || row.maximum > uint64(^uint64(0)>>1) {
			return nil, errProviderIntegration
		}
		disk := o.store.custody
		if row.root == 1 {
			disk = o.store.evidence
		}
		raw, err := o.store.read(disk, row.name, int64(row.maximum))
		if err != nil || uint64(len(raw)) != row.bytes ||
			!providerReplayTextEqual(providerReplayDigest("", raw, m), row.digest, m) {
			return nil, errProviderIntegration
		}
	}
	root, err := q.currentConsumptionRootLocked()
	model := q.models[role]
	if err != nil || root != q.modelRoots[role] || !model.validObserved(m) || model.head != root ||
		!o.coordinator.validObserved(m) || !o.custodian.validObserved(m) ||
		!providerReplayTextEqual(root, o.coordinator.head, m) || !providerReplayTextEqual(root, o.custodian.head, m) ||
		!providerReplayTextEqual(p.dataRootObserved(m), q.readRoots[role], m) || !q.readMatchesLocked(p, role) {
		return nil, errProviderIntegration
	}
	copy := providerHistoryMemberReplayCopyObserved(model, m)
	if copy == nil || copy.head != root || ctx.Err() != nil || !d.memberReplaySelectedLocked() ||
		!q.fixedContextMatchesObservedLocked(m) || !q.claimsMatchObservedLocked(m) || !q.ancestorsMatchObservedLocked(m) ||
		!providerHistoryMemberEqualObserved(providerHistoryMemberReadContextLocked(p), q.readContexts[role], q.readComparisons[role], m) ||
		!o.identityLocked() || !d.memberCatalogLocked() ||
		!o.store.frontierFiles.readExact("HEAD", providerReplayBytes(q.consumption.head, m)) || m.failed || m.closed {
		return nil, errProviderIntegration
	}
	return copy, nil
}

// Owned DATA only. Fresh ABI reconstruction is deliberately performed on the
// copied run, never on the immutable full-pass semantic baseline.
func providerHistoryMemberReplayCopyObserved(model *providerHistoryGroupLedger, m *providerWorkMeter) *providerHistoryGroupLedger {
	if model == nil || m == nil || !m.event(providerCopyEvent, uint64(reflect.TypeFor[providerHistoryGroupLedger]().Size())) {
		return nil
	}
	copy := providerHistoryLedgerCopyObserved(model, m)
	if copy == nil || copy == model || model.ticket != nil && copy.ticket == model.ticket || model.consumption != nil && copy.consumption == model.consumption {
		return nil
	}
	// ABI maps are mutable behavioral fields outside the semantic commitment.
	// Never trust/share the cached maps: rebuild on this role's owned copy from
	// the same immutable literals with the original primitive observer.
	if copy.collection != nil && copy.collection.semantic != nil && copy.collection.semantic.window != nil {
		w := copy.collection.semantic.window
		if !w.selected(w.run) {
			return nil
		}
		contract, outcomes, abiErr := replayABIsObserved(m)
		if abiErr != nil {
			return nil
		}
		w.run.contract, w.run.outcomes, w.run.steps = contract, outcomes, nil
	}
	if !copy.validObserved(m) || copy.head != model.head {
		return nil
	}
	return copy
}

// Scalar revocation also drops the potentially large transcript arrays. Old S
// archives retain original models/ACKs/bits, but never a growing prefix cache.
func (o *providerHistoryGroupOwner) releaseMemberReplayLocked() {
	if o == nil || o.parent == nil || o.parent.recovery == nil {
		return
	}
	r := o.parent.recovery
	if r.originalGroup != o {
		return
	}
	registered := r.memberReplaySeal
	var registeredDispatch *providerHistoryGroupDispatch
	if registered != nil && registered.recovery == r {
		registeredDispatch = registered.dispatch
		registered.revoked, registered.reads, registered.models, registered.ancestors = true, [2]*providerHistoryReplayReads{}, [2]*providerHistoryGroupLedger{}, nil
		registered.claims = [2]providerHistoryMemberClaimInput{}
		registered.fixed.groupModels = [2]*providerHistoryGroupLedger{}
	}
	for _, d := range [3]*providerHistoryGroupDispatch{r.originalDispatch, registeredDispatch, o.dispatch} {
		if d == nil || d.group != o || d.meter != o.meter {
			continue
		}
		q := d.memberReplaySeal
		if q != nil && q.dispatch == d && q.recovery == r {
			q.revoked, q.reads, q.models, q.ancestors = true, [2]*providerHistoryReplayReads{}, [2]*providerHistoryGroupLedger{}, nil
			q.claims = [2]providerHistoryMemberClaimInput{}
			q.fixed.groupModels = [2]*providerHistoryGroupLedger{}
		}
		d.memberReplaySeal, d.replayReads = nil, [2]*providerHistoryReplayReads{}
	}
	r.memberReplaySeal = nil
}
