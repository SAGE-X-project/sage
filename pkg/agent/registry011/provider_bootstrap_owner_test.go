package registry011

import "testing"

func providerBootstrapOwnerFixture(t *testing.T) *providerCustodyStore {
	t.Helper()
	first := providerFirstAdmissionBoundsFixture()
	config, clock := providerScanProvisionedFixtureWithFirst(t, &first)
	s, err := openProviderFrontierCandidate(config, clock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.close() })
	return s
}

func TestProviderBootstrapRetainsEnrollmentUsingTheExactOriginalOpenMeter(t *testing.T) {
	s := providerBootstrapOwnerFixture(t)
	o := s.bootstrapOwner
	m := o.meter
	before, upper, origin := m.actual, m.upper, s.setupOrigin
	stored, held := s.stored, s.scanControl.held
	root, err := s.enrollBeforeScan()
	if err != nil || root == nil {
		t.Fatal("ordinary original O through E0", err)
	}
	if o.meter != m || !o.started || !o.used || !o.done || o.failed || !m.closed || m.failed || m.upper != upper || m.actual[2] <= before[2] || m.actual[9] <= before[9] || m.actual[10] <= before[10] || s.setupOrigin != origin || s.stored != stored || s.scanControl.held != held || o.physical > stored || o.physical <= o.setupPhysical || s.sequence != 1 || s.enrollment != root.digest || len(s.objects) != 7 || s.frontierReplay.meter != nil || s.frontierReplay.body.meter != nil || s.frontierReplay.indexes.meter != nil || s.activeMeter.Load() != nil || s.last != nil {
		t.Fatal("E0 renewed O, double-charged files, retained an expired observer or produced an ACK")
	}
	finished := m.actual
	if again, err := s.enrollBeforeScan(); err == nil || again != nil || m.actual != finished || s.stored != stored {
		t.Fatal("E0 completion recreated a first-enrollment phase")
	}
	if err := s.scanControl.claim(providerScanAttemptFixture(t, s, s.scanControl.pin)); err != nil {
		t.Fatal("completed E0 did not permit its separately funded first inspection", err)
	}
}

func TestProviderBootstrapRefusesScanAndUnfundedWriterBeforeEnrollment(t *testing.T) {
	s := providerBootstrapOwnerFixture(t)
	held, stored := s.scanControl.held, s.stored
	if err := s.scanControl.claim(providerScanAttemptFixture(t, s, s.scanControl.pin)); err == nil || s.scanControl.held != held || s.stored != stored || s.scanControl.prepared != nil {
		t.Fatal("scan began before original O-funded E0")
	}
	if _, err := s.frontierReplay.retain(s, nil, nil, nil, nil, nil, providerFrontierHighwater{}); err == nil || s.failed || s.sequence != 0 || s.stored != stored || s.bootstrapOwner.started {
		t.Fatal("legacy writer bypassed the bootstrap owner")
	}
}

func TestProviderBootstrapMissingOrSpentOwnersDoNotStartWork(t *testing.T) {
	var absent *providerCustodyStore
	if root, err := absent.enrollBeforeScan(); err == nil || root != nil {
		t.Fatal("nil store reconstructed an initial owner")
	}
	for _, mode := range []string{"missing", "foreign", "started", "done", "failed", "closed-meter", "stale-pin", "active-meter"} {
		t.Run(mode, func(t *testing.T) {
			s := providerBootstrapOwnerFixture(t)
			o := s.bootstrapOwner
			switch mode {
			case "missing":
				s.bootstrapOwner = nil
			case "foreign":
				o.store = nil
			case "started":
				o.started = true
			case "done":
				o.done = true
			case "failed":
				o.failed = true
			case "closed-meter":
				o.meter.closed = true
			case "stale-pin":
				o.before = "unselected finite DATA"
			case "active-meter":
				s.activeMeter.Store(o.meter)
				t.Cleanup(func() { s.activeMeter.Store(nil) })
			}
			before, stored := o.meter.actual, s.stored
			if root, err := s.enrollBeforeScan(); err == nil || root != nil || s.sequence != 0 || o.meter.actual != before || s.stored != stored || s.last != nil {
				t.Fatal("missing or already spent owner performed bootstrap work", mode)
			}
		})
	}
}

// These are fixed counter-exhaustion units, not measurements or a way to
// inject an operating clock, disk allowance or replacement bootstrap meter.
func TestProviderBootstrapUnfundedOrUncertainStartKeepsTheWholeOriginalHold(t *testing.T) {
	for _, mode := range []string{"work", "physical", "clock-room", "owned-clock-exhausted", "closed-root"} {
		t.Run(mode, func(t *testing.T) {
			s := providerBootstrapOwnerFixture(t)
			o, m := s.bootstrapOwner, s.bootstrapOwner.meter
			switch mode {
			case "work":
				m.actual[2] = m.upper[2]
			case "physical":
				o.setupPhysical = s.stored
			case "clock-room":
				m.clocks = m.maximumClocks
			case "owned-clock-exhausted":
				s.clock.mu.Lock()
				s.clock.read = s.clock.maximum
				s.clock.mu.Unlock()
			case "closed-root":
				_ = s.evidence.root.Close()
			}
			stored, retained, held := s.stored, s.retainedRaw, s.scanControl.held
			if root, err := s.enrollBeforeScan(); err == nil || root != nil || !o.started || !o.failed || !s.failed || !m.closed || !m.failed || s.activeMeter.Load() != nil || s.stored != stored || s.retainedRaw != retained || s.scanControl.held != held || s.last != nil {
				t.Fatal("failed original bootstrap renewed or refunded its hold", mode)
			}
			if root, err := s.enrollBeforeScan(); err == nil || root != nil {
				t.Fatal("failed bootstrap retried with a new envelope")
			}
		})
	}
}

func TestProviderBootstrapPrepaidPhaseIsOneUseAndBoundedByTheOldHold(t *testing.T) {
	for _, mode := range []string{"ordinary", "zero", "phase-upper", "store-upper", "overflow"} {
		t.Run(mode, func(t *testing.T) {
			s := providerBootstrapOwnerFixture(t)
			o := s.bootstrapOwner
			o.started, o.planned = true, 2
			s.activeMeter.Store(o.meter)
			t.Cleanup(func() { s.activeMeter.Store(nil) })
			n := uint64(1)
			switch mode {
			case "zero":
				n = 0
			case "phase-upper":
				n = 3
			case "store-upper":
				o.setupPhysical = s.stored
			case "overflow":
				o.setupPhysical = ^uint64(0)
			}
			stored := s.stored
			if o.consume(n) != (mode == "ordinary") || s.stored != stored || o.UnmarshalJSON([]byte(`{}`)) == nil {
				t.Fatal("phase did not honor its original retained/live hold", mode)
			}
			if mode == "ordinary" && (o.consume(1) || !o.used || o.physical != o.setupPhysical+1) {
				t.Fatal("the same prepaid physical phase was consumed twice")
			}
		})
	}
	var missing *providerBootstrapOwner
	if missing.consume(1) {
		t.Fatal("nil prepaid phase")
	}
}

func TestProviderBootstrapWriterRefusesAnInsufficientPrepaidPhaseBeforeFiles(t *testing.T) {
	s := providerBootstrapOwnerFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	o, r := s.bootstrapOwner, s.frontierReplay
	m := o.meter
	stored, held, pin := s.stored, s.scanControl.held, s.frontierFiles.pinRaw
	// A fixed deficient phase inside the genuine old O. No new meter or
	// operating quota is injected, and the normal builder still derives cost.
	o.started, o.planned = true, 1
	s.activeMeter.Store(m)
	r.meter, r.body.meter, r.indexes.meter = m, m, m
	defer func() {
		m.refuse()
		m.mu.Lock()
		m.closed = true
		m.mu.Unlock()
		s.activeMeter.Store(nil)
		r.meter, r.body.meter, r.indexes.meter = nil, nil, nil
	}()
	if root, err := s.buildInitialEnrollment(m); err == nil || root != nil || !s.failed || !r.failed || !s.frontierFiles.failed || o.used || s.sequence != 0 || s.stored != stored || s.scanControl.held != held || s.frontierFiles.pinRaw != pin || s.frontierFiles.pending != nil || len(s.objects) != 0 || s.last != nil {
		t.Fatal("writer exceeded its deficient prepaid phase or wrote an original", err)
	}
}

func TestProviderBootstrapPlanRequiresItsFreshFiniteProfile(t *testing.T) {
	s := providerBootstrapOwnerFixture(t)
	work, physical, ok := providerBootstrapUpper(s.config.bounds)
	if !ok || work[2] == 0 || work[9] == 0 || work[10] == 0 || physical <= work[9] || s.bootstrapOwner.clock(ClockSample{}) {
		t.Fatal("incomplete prospective original-file plan")
	}
	for _, mode := range []string{"legacy", "entries", "objects"} {
		bounds := s.config.bounds
		switch mode {
		case "legacy":
			bounds.firstAdmission = nil
		case "entries":
			bounds.maximumEntries = 1
		case "objects":
			bounds.maximumObjects = 1
		}
		if _, _, ok := providerBootstrapUpper(bounds); ok {
			t.Fatal("unfunded or legacy profile acquired E0 bootstrap semantics", mode)
		}
	}
}
