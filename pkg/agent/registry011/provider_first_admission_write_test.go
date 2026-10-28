package registry011

import "testing"

// Each larger decoded envelope is selected before NEW H0. The existing
// smaller-profile refusal remains tested; no live F/B/O ceiling is raised.
func providerFirstWriterFixture(t *testing.T, first providerFirstAdmissionBounds) (*providerCustodyStore, *providerFirstAdmissionOwner) {
	t.Helper()
	s := providerScanEnrollmentOriginalsWithFirst(t, &first)
	x := mustProviderBindingCodec(t, s.config)
	target, err := x.target(providerScanTarget{setup: s.setupTip, custody: s.tip, frontier: s.frontierFiles.pin.frontier, objects: s.objectsRoot, enrollment: s.enrollment})
	if err != nil {
		t.Fatal(err)
	}
	attempt := providerScanAttemptFixture(t, s, s.scanControl.pin)
	attempt.target = x.hash("sage-provider-scan-target-v1", target)
	if s.scanControl.claim(attempt) != nil {
		t.Fatal("fresh candidate hold")
	}
	if _, err := s.scanControl.inspectBeforeAdmission(target); err != nil || s.firstAdmissionOwner == nil {
		t.Fatal("fresh candidate original inspection", err)
	}
	f := s.firstAdmissionOwner
	t.Cleanup(func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		f.abortLocked()
	})
	return s, f
}

func providerFirstWriterBounds() providerFirstAdmissionBounds {
	b := providerFirstAdmissionBoundsFixture()
	b.work[2] = 512 << 20
	return b
}

func TestProviderFirstAdmissionCandidateRetainsOriginalsOnceUnderTheOriginalHold(t *testing.T) {
	s, f := providerFirstWriterFixture(t, providerFirstWriterBounds())
	stored, retained, held, original := s.stored, s.retainedRaw, s.scanControl.held, s.frontierReplay
	before, clocks, upper := f.meter.actual, f.meter.clocks, f.meter.upper
	data, err := s.retainFirstAdmissionCandidate(providerFirstTicketReservationData())
	if err != nil || data == nil {
		t.Fatalf("candidate first E1 failed: %v; state=%d actual=%v clocks=%d plan=%+v physical=%d sequence=%d", err, f.state, f.meter.actual, f.meter.clocks, f.plan, f.physical, s.sequence)
	}
	r := s.frontierReplay
	if f.state != providerFirstAdmissionRetained || !f.replayStarted || !f.custodianStarted || !f.physicalUsed || f.plan == nil || !f.plan.used || s.stored != stored || s.scanControl.held != held || s.last != nil || s.sequence != 2 || s.frontierFiles.pending != nil || s.frontierFiles.pin != data.highwater || r == original || len(s.objects) != 21 || r.body.ledger == nil || r.body.intent == nil || r.current.enrollment != data.enrollment || s.enrollment != data.enrollment || r.body.anchor.digest != r.body.intent.anchor || r.body.anchor.digest == data.enrollment || r.body.ledger.current.digest != data.budget || r.body.ledger.lastTicket.digest != data.ticket || r.body.intent.digest != data.admission {
		t.Fatal("candidate imported coordinator state, repeated a hold or returned authority")
	}
	written := f.meter.actual[9] - before[9]
	if f.physical != written+21*s.config.bounds.fileOverheadBytes || s.retainedRaw != retained+written || f.physical > f.plan.physical || f.meter.upper != upper || f.meter.clocks != clocks+f.plan.clocks || data.work != f.meter.actual || data.physical != f.physical || !f.meter.closed || f.meter.failed || s.activeMeter.Load() != nil || r.meter != f.meter || r.body.meter != f.meter || r.indexes.meter != f.meter || r.body.ledger.meter != f.meter || r.body.ledger.valid() || r.indexes.valid() {
		t.Fatal("prepaid bytes, owned samples or permanent observer closure changed", written, retained, s.retainedRaw, f.meter.clocks-clocks, f.plan.clocks)
	}
	for coordinate, n := range f.meter.actual {
		if n < before[coordinate] || n-before[coordinate] > f.plan.work[coordinate] {
			t.Fatal("candidate exceeded its complete prospective plan", coordinate)
		}
	}
	for _, disk := range []*retentionDisk{s.custody, s.evidence} {
		if disk.checkIdentity() != nil {
			t.Fatal("owned candidate root identity")
		}
	}
	for _, name := range []string{"HEAD", providerCustodyName("highwater", 2), providerCustodyName("frontier", 2), providerCustodyName("commit", 2), providerCustodyName("intent", 2)} {
		if raw, err := s.custody.read(name, int64(s.config.bounds.maximumMetadataPerCommit)); err != nil || len(raw) == 0 {
			t.Fatal("retained DATA preceded original readbacks", name, err)
		}
	}
	for _, ref := range s.objects {
		raw, err := s.evidence.read(providerObjectName(ref.kind, ref.digest), int64(ref.bytes))
		digest, ok := providerIntegrationObjectDigest(ref.kind, raw)
		if err != nil || !ok || digest != ref.digest {
			t.Fatal("independent custodian lacks a referenced original", err)
		}
	}
	actual := f.meter.actual
	if next, err := s.retainFirstAdmissionCandidate(providerFirstTicketReservationData()); err == nil || next != nil || f.startLocked() == nil || f.consumeLocked(1) || f.writingLocked(s) || f.meter.actual != actual || data.UnmarshalJSON(nil) == nil || f.plan.UnmarshalJSON(nil) == nil {
		t.Fatal("retained diagnostics reconstructed a finishing owner or reused its files")
	}
	f.abortLocked()
	if s.failed || f.state != providerFirstAdmissionRetained || s.stored != stored || s.scanControl.held != held {
		t.Fatal("already completed candidate was refunded or converted to failure")
	}
}

func TestProviderFirstAdmissionCandidateRefusesInsufficientOriginalProfilesBeforeAnyIO(t *testing.T) {
	for _, mode := range []string{"original-decoded", "physical", "clock", "work", "clock-backend"} {
		t.Run(mode, func(t *testing.T) {
			first := providerFirstWriterBounds()
			switch mode {
			case "original-decoded":
				first = providerFirstAdmissionBoundsFixture()
			case "physical":
				first.live, first.retained, first.work[9] = 128<<10, 128<<10, 128<<10
			case "clock":
				first.clocks = 1
			case "work":
				first.work[10] = 1
			}
			s, f := providerFirstWriterFixture(t, first)
			if mode == "clock-backend" {
				// Finite exhaustion only; the original clock/epoch is retained.
				s.clock.mu.Lock()
				s.clock.maximum = s.clock.read
				s.clock.mu.Unlock()
			}
			actual, clocks, stored, retained, held, pin := f.meter.actual, f.meter.clocks, s.stored, s.retainedRaw, s.scanControl.held, s.frontierFiles.pin
			if data, err := s.retainFirstAdmissionCandidate(providerFirstTicketReservationData()); err == nil || data != nil || f.meter.actual != actual || f.meter.clocks != clocks || f.replayStarted || f.custodianStarted || f.physicalUsed || f.plan != nil || s.stored != stored || s.retainedRaw != retained || s.scanControl.held != held || s.frontierFiles.pin != pin || s.sequence != 1 || !s.failed || !f.meter.closed || f.state != providerFirstAdmissionFailed || s.last != nil {
				t.Fatal("insufficient original profile performed IO or invented a larger hold", mode)
			}
		})
	}
}

func TestProviderFirstAdmissionCandidateRefusesAClosedPreparedMeterBeforeIO(t *testing.T) {
	s, f := providerFirstWriterFixture(t, providerFirstWriterBounds())
	f.meter.mu.Lock()
	f.meter.closed = true
	f.meter.mu.Unlock()
	actual, clocks, stored, held := f.meter.actual, f.meter.clocks, s.stored, s.scanControl.held
	if data, err := s.retainFirstAdmissionCandidate(providerFirstTicketReservationData()); err == nil || data != nil || f.meter.actual != actual || f.meter.clocks != clocks || s.stored != stored || s.scanControl.held != held || !s.failed || f.state != providerFirstAdmissionFailed || s.last != nil {
		t.Fatal("a closed prepared meter was restarted, sampled or refunded")
	}
}

func TestProviderFirstAdmissionCandidateClosesOnMissingOriginalsOrUnavailableWrite(t *testing.T) {
	for _, mode := range []string{"enrollment", "setup", "control", "receipt", "destination"} {
		t.Run(mode, func(t *testing.T) {
			s, f := providerFirstWriterFixture(t, providerFirstWriterBounds())
			switch mode {
			case "enrollment":
				if s.evidence.root.Remove(providerObjectName("enrollment", s.enrollment)) != nil {
					t.Fatal("owned missing original")
				}
			case "setup":
				if s.evidence.root.Remove(providerCustodyName("setup-claim", 0)) != nil {
					t.Fatal("owned missing setup")
				}
			case "control":
				if s.custody.root.Remove(providerCustodyName("scan-highwater", 0)) != nil {
					t.Fatal("owned missing control")
				}
			case "receipt":
				if s.evidence.root.Remove(providerScanReceiptName("chunk", f.lastChunk, nil)) != nil {
					t.Fatal("owned missing receipt")
				}
			case "destination":
				if s.custody.root.Mkdir(providerCustodyName("intent", 2), 0700) != nil {
					t.Fatal("owned unavailable destination")
				}
			}
			stored, held, pin := s.stored, s.scanControl.held, s.frontierFiles.pin
			if data, err := s.retainFirstAdmissionCandidate(providerFirstTicketReservationData()); err == nil || data != nil || !s.failed || !s.scanControl.failed || !f.meter.failed || !f.meter.closed || s.activeMeter.Load() != nil || f.state != providerFirstAdmissionFailed || s.stored != stored || s.scanControl.held != held || s.frontierFiles.pin != pin || s.sequence != 1 || s.last != nil {
				t.Fatal("uncertain candidate write returned success, released its hold or renewed the owner", mode)
			}
			if mode == "destination" && (!f.physicalUsed || f.physical == 0 || f.plan == nil || !f.plan.used) {
				t.Fatal("uncertain first write lost its entire prepaid physical claim")
			}
			if data, err := s.retainFirstAdmissionCandidate(providerFirstTicketReservationData()); err == nil || data != nil {
				t.Fatal("failed candidate was retried automatically")
			}
		})
	}
}

func TestProviderFirstAdmissionCandidateCannotStartFromMissingOrSpentData(t *testing.T) {
	var missing *providerCustodyStore
	if data, err := missing.retainFirstAdmissionCandidate(providerOperationReservation{}); err == nil || data != nil {
		t.Fatal("missing store created retained DATA")
	}
	var owner *providerFirstAdmissionOwner
	if owner.planLocked() || owner.adoptCustodianLocked(nil) || owner.writingLocked(nil) || owner.consumeLocked(1) {
		t.Fatal("missing live owner manufactured a plan or prepaid write")
	}
	s, f := providerFirstWriterFixture(t, providerFirstWriterBounds())
	if reader, err := f.replayCustodianLocked(); err == nil || reader != nil {
		t.Fatal("prepared DATA began the custodian pass")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if f.startLocked() != nil || !f.planLocked() || f.planLocked() || f.adoptCustodianLocked(nil) || f.consumeLocked(0) || f.consumeLocked(1) {
		t.Fatal("pre-write plan became a reusable or unbound physical claim")
	}
	input, err := f.replayEnrollmentLocked()
	if err != nil {
		t.Fatal(err)
	}
	reader, err := f.replayCustodianLocked()
	if err != nil || input == reader || input.replay == reader.replay || reader.replay.indexes == input.replay.indexes || !f.adoptCustodianLocked(reader) {
		t.Fatal("second pass reused the first reader or could not bind originals", err)
	}
	if _, err := f.replayCustodianLocked(); err == nil || f.adoptCustodianLocked(input) {
		t.Fatal("custodian pass repeated or coordinator reader became custodian authority")
	}
}

func TestProviderFirstAdmissionPrepaidBoundaryUsesNativeObservedPinComparison(t *testing.T) {
	s, f := providerFirstWriterFixture(t, providerFirstWriterBounds())
	s.mu.Lock()
	defer s.mu.Unlock()
	if f.startLocked() != nil || !f.planLocked() {
		t.Fatal("original candidate plan")
	}
	if _, err := f.replayEnrollmentLocked(); err != nil {
		t.Fatal(err)
	}
	reader, err := f.replayCustodianLocked()
	if err != nil || !f.adoptCustodianLocked(reader) {
		t.Fatal("independent custodian", err)
	}
	f.plan.used = true
	before, clocks := f.meter.actual, f.meter.clocks
	if !f.writingLocked(s) {
		t.Fatal("closed prepaid role refused its original main pin")
	}
	var want candidateWorkVector
	want[10] = uint64(len(s.frontierFiles.pinRaw)+len(f.mainRaw)) + 1
	for coordinate, n := range f.meter.actual {
		if n < before[coordinate] || n-before[coordinate] != want[coordinate] {
			t.Fatal("pin boundary added unplanned raw conversions or codec/write work", coordinate)
		}
	}
	if f.meter.clocks != clocks || f.physicalUsed || s.sequence != 1 || s.last != nil {
		t.Fatal("pin comparison sampled, spent or dispatched work")
	}
}
