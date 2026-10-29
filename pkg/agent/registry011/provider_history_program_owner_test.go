package registry011

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// Scalar-only selection tests: the preprovisioned roots retain their ORIGINAL
// old profile and are NEVER opened with this fabricated derivation witness.
// This test-only witness proves no source-cost row, ACK or operating choice.
// Production deliberately has no derivation factory while costs remain open.
func providerHistoryParentScalarFixture(t *testing.T) *providerCustodyStore {
	t.Helper()
	first := providerFirstAdmissionBoundsFixture()
	config, clock := providerScanProvisionedFixtureWithFirst(t, &first)
	scan := *config.bounds.scan
	scan.maximumPreIRetainedBytes /= 2
	scan.maximumPreIWork[2] /= 2
	scan.maximumPreIWork[9] /= 2
	scan.maximumPreIWork[10] /= 2
	scan.maximumClockReads, scan.maximumPreIClockReads = 256, 1024
	config.bounds.scan = &scan
	work := candidateWorkVector{}
	for i := range work {
		work[i] = 1
	}
	work[10] = 128
	h := &providerHistoryProgramBounds{
		program: admissionTestDigest("a"), costs: admissionTestDigest("b"),
		future:  providerHistoryGroupTotals{work: work, physical: 4096, retainedRaw: 1024, payload: 64, metadata: 64, objects: 3, files: 10, clocks: 32},
		startup: providerScanUpper{work: work, retained: 1024, clocks: 11},
		stages:  1, records: 3, commits: 3, catalogEntries: 10, catalogNameBytes: 2550, resident: 1 << 20, durationMS: 300000,
	}
	h.derivation = &providerHistoryProgramDerivation{}
	h.derivation.owner = h.derivation
	h.derivation.snapshot = *h
	config.bounds.history = h
	config.profile = providerIntegrationProfile(config.bounds, config.capture)
	if !config.bounds.valid() {
		t.Fatal("inert complete scalar shape")
	}
	held, ok := providerHistoryWholeHold(config.bounds)
	if !ok {
		t.Fatal("inert disjoint sum")
	}
	limits := &providerFirstAdmissionLimits{work: held.work, operation: work, segment: work, live: held.retained, retained: held.retained}
	s := &providerCustodyStore{config: config, clock: clock, stored: held.retained, firstLimits: limits}
	s.owner = s
	control := &providerScanControl{store: s}
	control.owner = control
	meter := &providerWorkMeter{scan: control, upper: config.bounds.setupWork, maximumClocks: config.bounds.setupMaximumClockReads}
	meter.owner = meter
	s.activeMeter.Store(meter)
	return s
}

func providerHistoryParentBoundScalarFixture(t *testing.T) (*providerCustodyStore, *providerHistoryProgramOwner) {
	t.Helper()
	s := providerHistoryParentScalarFixture(t)
	p, err := newProviderHistoryProgramBeforeOpenCandidate(s)
	if err != nil || p == nil || s.clock.read != 0 {
		t.Fatal("hold before any native sample", err)
	}
	sample, err := s.samplePrimitive()
	if err != nil || !p.bindOriginalLocked(sample) {
		t.Fatal("single original backend sample", err)
	}
	return s, p
}

func TestProviderHistoryProgramRefusesNumbersAndJSONBeforeObservation(t *testing.T) {
	s := providerHistoryParentScalarFixture(t)
	s.config.bounds.history.derivation = nil
	if s.config.bounds.valid() {
		t.Fatal("numbers supplied a source derivation")
	}
	if p, err := openProviderScanCandidate(s.config, s.clock); err == nil || p != nil || s.clock.read != 0 || s.clock.history != nil {
		t.Fatal("unfinished profile reached roots or clock")
	}
	for _, value := range []any{&providerHistoryProgramBounds{}, &providerHistoryProgramDerivation{}, &providerHistoryProgramOwner{}} {
		if json.Unmarshal([]byte(`{}`), value) == nil {
			t.Fatal("JSON reconstructed a private original")
		}
	}
}

func TestProviderHistoryProgramHoldsEveryCoordinateBeforeFirstSample(t *testing.T) {
	for row := 0; row < 13; row++ {
		s := providerHistoryParentScalarFixture(t)
		if row < 11 {
			s.firstLimits.work[row]--
		} else if row == 11 {
			s.firstLimits.live--
		} else {
			s.firstLimits.retained--
		}
		stored := s.stored
		if p, err := newProviderHistoryProgramBeforeOpenCandidate(s); err == nil || p != nil || s.history != nil || s.clock.history != nil || s.clock.read != 0 || s.stored != stored {
			t.Fatal("partial or unfunded pre-H0 reservation", row)
		}
	}
	s := providerHistoryParentScalarFixture(t)
	if !s.clock.claimLegacy(s.config.identity.Policy) {
		t.Fatal("ordinary backend claim")
	}
	if p, err := newProviderHistoryProgramBeforeOpenCandidate(s); err == nil || p != nil || s.clock.read != 0 || s.history != nil {
		t.Fatal("backend already handed to a legacy store was reserved")
	}
}

func TestProviderHistoryProgramClockIsExclusiveAndCannotRebind(t *testing.T) {
	s := providerHistoryParentScalarFixture(t)
	p, err := newProviderHistoryProgramBeforeOpenCandidate(s)
	if err != nil {
		t.Fatal(err)
	}
	copy := *p
	if _, err := s.clock.sample(); err == nil {
		t.Fatal("legacy caller spent the parent clock")
	}
	if !s.event(providerClockEvent, 0) {
		t.Fatal("one clock event before its native claim")
	}
	claim := p.issueClockClaimLocked(s.activeMeter.Load())
	if claim == nil {
		t.Fatal("exact original O clock claim")
	}
	if _, err := s.clock.sampleForHistory(&copy, claim); err == nil || s.clock.read != 0 || s.clock.claimLegacy(s.config.identity.Policy) {
		t.Fatal("copy or second store acquired the backend")
	}
	sample, err := s.clock.sampleForHistory(p, claim)
	if err != nil || !p.bindOriginalLocked(sample) || p.bindOriginalLocked(sample) || s.clock.read != 1 || !p.matchesLocked() {
		t.Fatal("original bound more than once", err)
	}
	p.failLocked()
	if _, err := s.clock.sampleForHistory(p, claim); err == nil || !s.clock.historyRevoked || s.clock.read != 1 || p.matchesLocked() {
		t.Fatal("failure refunded or restored the clock")
	}
}

func TestProviderHistoryProgramRefusesReplacementOriginalAndSelectedInputs(t *testing.T) {
	s := providerHistoryParentScalarFixture(t)
	p, err := newProviderHistoryProgramBeforeOpenCandidate(s)
	if err != nil {
		t.Fatal(err)
	}
	sample, err := s.samplePrimitive()
	if err != nil {
		t.Fatal(err)
	}
	sample.MonotonicMS++
	if p.bindOriginalLocked(sample) || !p.failed || !s.clock.historyRevoked || p.original != (ClockSample{}) {
		t.Fatal("caller metadata replaced the actual original")
	}
	for _, mode := range []string{"policy", "scan", "method", "tls", "limits", "stored", "bounds", "epoch"} {
		t.Run(mode, func(t *testing.T) {
			local, parent := providerHistoryParentBoundScalarFixture(t)
			switch mode {
			case "policy":
				local.config.policy.policy = admissionTestDigest("f")
			case "scan":
				local.config.bounds.scan.maximumAttempts--
			case "method":
				local.config.bounds.methods = &providerMethodBounds{}
			case "tls":
				local.config.bounds.tls = &providerTLSBounds{}
			case "limits":
				local.firstLimits.live--
			case "stored":
				local.stored--
			case "bounds":
				local.config.bounds.history.future.metadata--
			case "epoch":
				local.clock.epoch = admissionTestDigest("f")
			}
			if parent.matchesLocked() {
				t.Fatal("changed selected input preserved the original hold")
			}
		})
	}
}

func TestProviderHistoryProgramSubclaimsDoNotDebitOrRefundParent(t *testing.T) {
	s, p := providerHistoryParentBoundScalarFixture(t)
	stored := s.stored
	addition := providerScanUpper{work: candidateWorkVector{0, 0, 1, 0, 0, 0, 0, 0, 0, 1, 1}, retained: 2, clocks: 1}
	if !p.claimInspectionLocked(providerScanUpper{}, addition, providerScanUpper{}) || p.claimInspectionLocked(providerScanUpper{}, addition, providerScanUpper{}) || p.inspection != addition || s.stored != stored {
		t.Fatal("inspection prefix was duplicated or double debited")
	}
	tooMuch := addition
	tooMuch.work[0] = 1
	if p.claimInspectionLocked(addition, tooMuch, providerScanUpper{}) || p.inspection != addition || s.stored != stored {
		t.Fatal("inspection borrowed a request from future work")
	}
	plan := providerFirstOperationPlan{reservation: providerOperationReservation{work: p.bounds.startup.work}, storage: providerFirstOperationStorage{physical: p.bounds.startup.retained}}
	wrong := plan
	wrong.storage.physical++
	if p.claimStartupLocked(wrong) || p.startupClaimed || !p.claimStartupLocked(plan) || !p.startupClaimed || p.claimStartupLocked(plan) || p.claimInspectionLocked(addition, addition, providerScanUpper{}) || s.stored != stored {
		t.Fatal("first-P hold was resized, duplicated or refunded")
	}
}

func TestProviderHistoryProgramCapacityIncludesUnspentFutureHold(t *testing.T) {
	s := providerHistoryParentScalarFixture(t)
	open := providerScanOpenUpper(s.config)
	if !providerFirstAdmissionCapacity(s, open) {
		t.Fatal("one disjoint original whole hold did not fit")
	}
	for row := 0; row < 13; row++ {
		limits := *s.firstLimits
		local := &providerCustodyStore{config: s.config, firstLimits: &limits}
		if row < 11 {
			limits.work[row]--
		} else if row == 11 {
			limits.live--
		} else {
			limits.retained--
		}
		if providerFirstAdmissionCapacity(local, open) {
			t.Fatal("small actual prefix discounted the future hold", row)
		}
	}
	outside := providerHistoryPreIUpper(s.config.bounds)
	outside.work[0] = 1
	if providerFirstAdmissionCapacity(s, outside) {
		t.Fatal("pre-I inspection consumed a future request")
	}
}

func TestProviderHistoryProgramGenericEntrypointsRefuseBeforeEffects(t *testing.T) {
	s, _ := providerHistoryParentBoundScalarFixture(t)
	// Satisfy valid()'s structural preconditions. These inert handles are not
	// original-read/ACK proof and must never reach a filesystem operation.
	s.custody, s.evidence = &retentionDisk{}, &retentionDisk{}
	s.objects, s.operations = make(map[string]providerIntegrationObjectRef), make(map[string]bool)
	s.activeMeter.Store(nil)
	c := &providerIntegration{store: s, clock: s.clock, policy: s.config.policy, intent: &CandidateAdmissionIntent{}, root: &CandidateEnrollmentRoot{}, primitive: providerPrimitiveBounds{capture: s.config.capture}}
	c.owner = c
	if !c.valid() {
		t.Fatal("generic refusal fixture did not pass the existing structural guard")
	}
	beforeRead, beforeStored := s.clock.read, s.stored
	if c.prepare() == nil || c.legacyAllowed() {
		t.Fatal("generic preparation accepted whole-program mode")
	}
	if _, err := c.admit("rpc-capture", admissionTestDigest("a"), "", "", providerOperationReservation{}, nil); err == nil {
		t.Fatal("generic admission reached the nil generic cache")
	}
	if _, _, err := c.execute(context.Background(), "eth_chainId", nil, 1); err == nil {
		t.Fatal("generic execution accepted the parent")
	}
	if _, err := c.readClient(); err == nil {
		t.Fatal("generic client escaped")
	}
	if _, err := c.attachMeter(providerOperationReservation{}); err == nil {
		t.Fatal("generic meter replaced a closed owner")
	}
	if _, err := c.beginLocal(admissionTestDigest("a")); err == nil {
		t.Fatal("generic local command accepted the parent")
	}
	if _, err := c.currentClock(); err == nil || c.use(nil) {
		t.Fatal("generic command spent the first-owner clock")
	}
	c.restoreMeter(nil)
	c.failLocal(nil)
	if c.failed || c.preparations != 0 || c.meters != nil || s.activeMeter.Load() != nil || s.clock.read != beforeRead || s.stored != beforeStored {
		t.Fatal("refusal mutated the reserved owner or resources")
	}
}

func TestProviderHistoryProgramUnclaimedHeadRefusesBeforeNativeIO(t *testing.T) {
	s, _ := providerHistoryParentBoundScalarFixture(t)
	// Deliberately inert handles: reaching any filesystem primitive is an
	// invalid test execution, not a substitute for real original readbacks.
	s.custody, s.evidence = &retentionDisk{}, &retentionDisk{}
	f := &providerFrontierFiles{store: s, initialized: true}
	f.owner, s.frontierFiles = f, f
	stored, reads := s.stored, s.clock.read
	if _, err := f.advanceClaimedLocked(providerFrontierHighwater{}, "main", nil, nil, nil, nil); err == nil || f.failed || s.failed || s.stored != stored || s.clock.read != reads {
		t.Fatal("unclaimed HEAD mutation reached encoding, IO or debit")
	}
}

func TestProviderHistoryProgramClockEnforcesLifetimeAndSelectedBackend(t *testing.T) {
	for _, mode := range []string{"ceiling", "origin", "window", "uncertainty", "maximum"} {
		t.Run(mode, func(t *testing.T) {
			s, p := providerHistoryParentBoundScalarFixture(t)
			if !s.event(providerClockEvent, 0) {
				t.Fatal("prepaid scalar clock event")
			}
			claim := p.issueClockClaimLocked(s.activeMeter.Load())
			if claim == nil {
				t.Fatal("one-use metered clock claim")
			}
			before := s.clock.read
			switch mode {
			case "ceiling":
				s.clock.read = p.clockCeiling
				before = s.clock.read
			case "origin":
				s.clock.origin = s.clock.origin.Add(-time.Second)
			case "window":
				s.clock.readWindow++
			case "uncertainty":
				s.clock.uncertainty--
			case "maximum":
				s.clock.maximum++
			}
			if _, err := s.clock.sampleForHistory(p, claim); err == nil || s.clock.read != before {
				t.Fatal("changed backend or exhausted lifetime reached a sample")
			}
		})
	}
	s := providerHistoryParentScalarFixture(t)
	h := s.config.bounds.history
	h.durationMS = 1
	h.derivation.snapshot = *h // Test-only finite deadline, no source derivation.
	p, err := newProviderHistoryProgramBeforeOpenCandidate(s)
	if err != nil {
		t.Fatal(err)
	}
	sample, err := s.samplePrimitive()
	if err != nil || !p.bindOriginalLocked(sample) {
		t.Fatal("original finite-clock sample", err)
	}
	time.Sleep(5 * time.Millisecond)
	if _, err := s.samplePrimitive(); err == nil || !s.clock.failed || !s.clock.historyRevoked || s.clock.read != 2 {
		t.Fatal("expired whole-program deadline was renewed")
	}
}
