package registry011

import (
	"context"
	"os"
	"testing"
)

// Stop a fresh normal candidate after its owned E0 write and before G0.
// Only tests can reach this split point; no fault callback or alternate
// constructor is added to the production path.
func providerGuardianBeforeGenesisFixture(t *testing.T) (*providerCustodyStore, *providerGuardianOwner, *providerBootstrapOwner) {
	t.Helper()
	config, clock := providerGuardianConfigFixture(t)
	s, err := openProviderFrontierCandidate(config, clock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.close() })
	o, r := s.bootstrapOwner, s.frontierReplay
	_, physical, ok := providerBootstrapUpper(s.config.bounds)
	if !ok {
		t.Fatal("finite bootstrap quote")
	}
	o.started, o.planned = true, physical
	s.activeMeter.Store(o.meter)
	r.meter, r.body.meter, r.indexes.meter = o.meter, o.meter, o.meter
	if _, err := s.buildInitialEnrollment(o.meter); err != nil {
		t.Fatal("owned E0 split-point fixture", err)
	}
	return s, s.guardian, o
}

func TestProviderGuardianConstructorRefusesIncompletePreOpenCapacity(t *testing.T) {
	if next, err := newProviderGuardianBeforeOpenCandidate(nil); err == nil || next != nil {
		t.Fatal("missing store created G")
	}
	for _, mode := range []string{"stored", "work"} {
		t.Run(mode, func(t *testing.T) {
			config, clock := providerGuardianConfigFixture(t)
			limits := &providerFirstAdmissionLimits{live: ^uint64(0), retained: ^uint64(0)}
			for i := range limits.work {
				limits.work[i] = ^uint64(0)
			}
			s := &providerCustodyStore{config: config, clock: clock, firstLimits: limits}
			s.owner = s
			s.stored, _ = providerGuardianInitialStored(config.bounds)
			if mode == "stored" {
				s.stored--
			} else {
				limits.work[2] = config.bounds.setupWork[2]
			}
			if next, err := newProviderGuardianBeforeOpenCandidate(s); err == nil || next != nil || clock.read != 0 || s.custody != nil || s.evidence != nil {
				t.Fatal("incomplete pre-open hold observed roots or created G", mode)
			}
		})
	}
}

func TestProviderGuardianLineageRefusesMissingOrUnboundedOriginals(t *testing.T) {
	var absent *providerScanReplayReader
	if absent.guardianInitialLineage() {
		t.Fatal("missing original reader")
	}
	for _, mode := range []string{"genesis", "head", "retained-overflow"} {
		t.Run(mode, func(t *testing.T) {
			s, _, o := providerGuardianBeforeGenesisFixture(t)
			s.mu.Lock()
			defer s.mu.Unlock()
			if mode != "genesis" {
				if err := s.guardian.installGenesisLocked(o); err != nil {
					t.Fatal("normal owned G0", err)
				}
			}
			if mode == "head" {
				if err := s.custody.root.Remove("GUARDIAN-HEAD"); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "retained-overflow" {
				s.retainedRaw = ^uint64(0)
			}
			// Negative predicate DATA only; this does not construct or qualify
			// an independent scan owner, nor claim a successful scan.
			i := &providerScanReplayReader{view: s, replay: s.frontierReplay, meter: o.meter, selected: s.frontierFiles.pin}
			if i.guardianInitialLineage() || i.guardianGenesis {
				t.Fatal("missing original or overflow became lineage", mode)
			}
		})
	}
}

func TestProviderGuardianOpenRefusesAnAlreadyUsedOriginalClock(t *testing.T) {
	config, clock := providerGuardianConfigFixture(t)
	clock.mu.Lock()
	clock.read = 1
	clock.mu.Unlock()
	if next, err := openProviderScanCandidate(config, clock); err == nil || next != nil || clock.read != 1 {
		t.Fatal("reused G origin took another sample or opened a candidate")
	}
}

func TestProviderGuardianBootstrapRefusesAnUnselectedOrigin(t *testing.T) {
	config, clock := providerGuardianConfigFixture(t)
	s, err := openProviderFrontierCandidate(config, clock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.close() })
	s.guardian.original.Epoch = admissionTestDigest("f")
	stored := s.stored
	if next, err := s.enrollBeforeScan(); err == nil || next != nil || !s.failed || s.stored != stored || s.last != nil {
		t.Fatal("different original G clock enrolled a usable candidate")
	}
}

func TestProviderGuardianFirstAdmissionCannotBindAnIncompleteGuardian(t *testing.T) {
	s := providerGuardianPreparedFixture(t)
	s.guardian.initialized = false
	if next, err := newProviderFirstOperationCandidate(s); err == nil || next != nil || !s.failed || s.last != nil {
		t.Fatal("incomplete G supplied a first operation")
	}
}

func TestProviderGuardianFirstSettlementCannotPublishWithoutItsOneUseHandoff(t *testing.T) {
	s, o := providerGuardianFirstFixture(t)
	s.guardian.ready = true // Inert invalid handoff state, not a runtime peer.
	conn := providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, providerFirstOperationResponse())
	if result, err := o.executeCandidateWithConnection(context.Background(), conn); err == nil || result != nil || !s.failed || !s.guardian.failed || s.last != nil {
		t.Fatal("invalid handoff published a result")
	}
}

func TestProviderGuardianSampleRoutesOnlyTheSelectedStore(t *testing.T) {
	s, g := providerGuardianSettledFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := g.startClaimLocked("bootstrap-admission"); err != nil {
		t.Fatal(err)
	}
	if now, err := s.sample(); err != nil || now.ClockID != g.original.ClockID || now.Epoch != g.original.Epoch {
		t.Fatal("selected G lost its original clock")
	}
	other := &providerCustodyStore{}
	other.activeMeter.Store(g.meter)
	reads := s.clock.read
	if _, err := other.sample(); err == nil || s.clock.read != reads {
		t.Fatal("different store borrowed the G clock")
	}
}

func TestProviderGuardianGenesisRefusesIncompleteOwnedStorageAndQuotas(t *testing.T) {
	for _, mode := range []string{"codec-quota", "missing-origin", "physical", "closed-root", "present-head", "sync"} {
		t.Run(mode, func(t *testing.T) {
			s, g, o := providerGuardianBeforeGenesisFixture(t)
			s.mu.Lock()
			defer s.mu.Unlock()
			stored := s.stored
			switch mode {
			case "codec-quota":
				o.meter.upper[10] = o.meter.actual[10]
			case "missing-origin":
				if err := s.evidence.root.Remove(providerObjectName("guardian-origin", g.origin)); err != nil {
					t.Fatal(err)
				}
			case "physical":
				o.planned = o.physical - o.setupPhysical
			case "closed-root":
				if err := s.custody.root.Close(); err != nil {
					t.Fatal(err)
				}
			case "present-head":
				if err := os.WriteFile(s.config.custodyDirectory+"/GUARDIAN-HEAD", []byte("incomplete isolated test DATA"), 0600); err != nil {
					t.Fatal(err)
				}
			case "sync":
				s.custody.syncFile = func(*os.File) error { return errProviderIntegration }
			}
			if err := g.installGenesisLocked(o); err == nil || !g.failed || !s.failed || g.initialized || s.stored != stored || s.activeMeter.Load() != nil || s.last != nil {
				t.Fatal("incomplete G0 restored a candidate owner", mode, err)
			}
		})
	}
}

func TestProviderGuardianConsumedClaimClosesAfterMissingOriginalOrWriteFailure(t *testing.T) {
	for _, mode := range []string{"original", "write", "end-clock", "start-clock"} {
		t.Run(mode, func(t *testing.T) {
			s, g := providerGuardianSettledFixture(t)
			s.mu.Lock()
			defer s.mu.Unlock()
			stored := s.stored
			if mode == "start-clock" {
				s.clock.mu.Lock()
				s.clock.uncertainty = 0 // Finite unavailable-backend unit only.
				s.clock.mu.Unlock()
				if g.startClaimLocked("bootstrap-admission") == nil || !g.failed || s.stored != stored || g.ordinal != 1 {
					t.Fatal("unavailable first sample refunded G")
				}
				return
			}
			if g.startClaimLocked("bootstrap-admission") != nil {
				t.Fatal("fresh metadata claim")
			}
			switch mode {
			case "original":
				if err := s.custody.root.Remove("HEAD"); err != nil {
					t.Fatal(err)
				}
			case "write":
				s.custody.syncFile = func(*os.File) error { return errProviderIntegration }
			case "end-clock":
				// Simulate exhaustion of the owned finite read allowance at a
				// normal write boundary; no clock callback enters production.
				s.custody.syncFile = func(file *os.File) error {
					s.clock.mu.Lock()
					s.clock.read = s.clock.maximum
					s.clock.mu.Unlock()
					return file.Sync()
				}
			}
			if g.retainClaimLocked(g.integration.firstExchange) == nil || !g.failed || !s.failed || !g.meter.closed || s.activeMeter.Load() != nil || s.stored != stored || s.sequence != 4 || s.last != nil {
				t.Fatal("uncertain consumed metadata remained usable", mode)
			}
		})
	}
}

func TestProviderGuardianOriginalHelpersRefuseMissingGenesisAndInvalidData(t *testing.T) {
	s, g, o := providerGuardianBeforeGenesisFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	m := o.meter
	clock := providerClockDiagnostic(s.setupOrigin)
	if _, err := providerGuardianOriginOriginal(nil, m, s.setupTip, clock); err == nil {
		t.Fatal("missing original owner")
	}
	if _, err := providerGuardianOriginOriginal(s, m, admissionTestDigest("f"), clock); err == nil {
		t.Fatal("different selected H0 setup")
	}
	if _, err := providerGuardianInitialOriginals(s, m, s.tip, s.frontierFiles.pin.frontier, s.setupTip, clock); err == nil {
		t.Fatal("absent genesis accepted")
	}
	x := mustProviderBindingCodec(t, s.config)
	if _, _, err := providerGuardianExpectedInitial(nil, s.tip, g.pin.frontier, s.setupTip, clock); err == nil {
		t.Fatal("missing codec")
	}
	invalidClock := clock
	invalidClock.WallMS = 0
	if _, _, err := providerGuardianExpectedInitial(x, s.tip, s.frontierFiles.pin.frontier, s.setupTip, invalidClock); err == nil {
		t.Fatal("invalid original deadline")
	}
	if _, _, err := providerGuardianExpectedInitial(x, "", s.frontierFiles.pin.frontier, s.setupTip, clock); err == nil {
		t.Fatal("missing exact E0")
	}
	var absent *providerGuardianOwner
	if _, err := absent.sampleLocked(); err == nil || absent.retainClaimLocked(admissionTestDigest("f")) == nil {
		t.Fatal("missing guardian handle")
	}
	absent.failLocked()
}

func TestProviderGuardianInitialOriginalsRefuseAnInvalidExactMain(t *testing.T) {
	s, _, o := providerGuardianBeforeGenesisFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	if data, err := providerGuardianInitialOriginals(s, o.meter, "", s.frontierFiles.pin.frontier, s.setupTip, providerClockDiagnostic(s.setupOrigin)); err == nil || data != nil {
		t.Fatal("invalid exact main created genesis DATA")
	}
}

func TestProviderGuardianClaimRoomRejectsEachCumulativeDebit(t *testing.T) {
	s, g := providerGuardianSettledFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, mode := range []string{"work", "retained", "live", "clocks", "records", "objects", "raw", "metadata", "overflow"} {
		t.Run(mode, func(t *testing.T) {
			bounds := g.bounds
			defer func() { g.bounds = bounds }()
			switch mode {
			case "work":
				g.bounds.lifetime.work[2] = 1
			case "retained":
				g.bounds.lifetime.retained = 1
			case "live":
				g.bounds.lifetime.live = 1
			case "clocks":
				g.bounds.lifetime.clocks = 1
			case "records":
				g.bounds.lifetime.records = 1
			case "objects":
				g.bounds.lifetime.objects = 1
			case "raw":
				g.bounds.lifetime.raw = 1
			case "metadata":
				g.bounds.lifetime.metadata = 1
			case "overflow":
				g.bounds.claim.work[10] = ^uint64(0)
			}
			reads, stored := s.clock.read, s.stored
			if g.claimRoomLocked(2) || g.ordinal != 0 || g.active || s.clock.read != reads || s.stored != stored {
				t.Fatal("insufficient cumulative DATA quote observed or changed a resource", mode)
			}
		})
	}
}

func TestProviderGuardianSampleRejectsAChangedPerClaimStart(t *testing.T) {
	s, g := providerGuardianSettledFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	if g.startClaimLocked("bootstrap-admission") != nil {
		t.Fatal("fresh metadata start")
	}
	g.claim.start.Epoch = admissionTestDigest("f") // Inert inconsistent DATA.
	if _, err := g.sampleLocked(); err == nil || !g.failed || !s.failed || !g.meter.closed {
		t.Fatal("per-claim start replaced original epoch")
	}
}

func TestProviderGuardianOriginalClaimRefusesAnAbsentImmutableClaim(t *testing.T) {
	s, g := providerGuardianSettledFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	if g.startClaimLocked("bootstrap-admission") != nil || g.retainClaimLocked(g.integration.firstExchange) != nil {
		t.Fatal("fresh metadata claim")
	}
	if err := s.evidence.root.Remove(s.name("guardian-claim", g.ordinal)); err != nil {
		t.Fatal(err)
	}
	if data, err := providerGuardianClaimOriginals(s, g.meter, g.before, g.main, g.integration.intent.digest, g.integration.firstExchange, "bootstrap-admission"); err == nil || data != nil {
		t.Fatal("missing immutable original became a claim")
	}
}
