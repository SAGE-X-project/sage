package registry011

import "testing"

func providerFirstAdmissionOwnerFixture(t *testing.T) (*providerCustodyStore, *providerFirstAdmissionOwner) {
	t.Helper()
	s, target := providerScanReceiptStoreFixture(t)
	result, err := s.scanControl.inspectBeforeAdmission(target)
	if err != nil || result == nil || s.firstAdmissionOwner == nil {
		t.Fatal("ordinary retained inspection did not prepare its finishing owner", err)
	}
	f := s.firstAdmissionOwner
	t.Cleanup(func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		f.abortLocked()
	})
	return s, f
}

func TestProviderFirstAdmissionHandoffRetainsTheExactInspectionMeter(t *testing.T) {
	s, f := providerFirstAdmissionOwnerFixture(t)
	m, b := f.meter, f.body
	before, upper, clocks := m.actual, m.upper, m.clocks
	stored, held := s.stored, s.scanControl.held
	if f.state != providerFirstAdmissionPrepared || f.startWork != before || f.upper != upper || f.startClocks != clocks || f.origin != s.setupOrigin || f.targetRaw != b.targetRaw || f.lastChunk != b.receipts.chunks[len(b.receipts.chunks)-1] || f.records != uint64(len(b.receipts.chunks)) || b.active || b.failed || m.closed || m.failed || s.activeMeter.Load() != nil {
		t.Fatal("inspection DATA reset, closed or replaced the original B")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := f.startLocked(); err != nil || f.state != providerFirstAdmissionStarted || s.activeMeter.Load() != m || m.firstAdmission != f || m.actual != before || m.upper != upper || m.clocks != clocks || b.active || b.valid() || b.receipts.valid() {
		t.Fatal("finishing start renewed B or resurrected the inspection body", err)
	}
	if _, err := f.sampleLocked(); err != nil || !m.event(providerCopyEvent, 3) || m.actual[10] <= before[10] || m.clocks != clocks+1 || m.upper != upper || s.stored != stored || s.scanControl.held != held || s.last != nil {
		t.Fatal("finishing did not use the cumulative meter or invented an ACK", err)
	}
	actual := m.actual
	if f.startLocked() == nil || f.UnmarshalJSON([]byte(`{}`)) == nil || m.actual != actual || b.active {
		t.Fatal("used owner restarted or restored the old body")
	}
	f.abortLocked()
	if f.state != providerFirstAdmissionFailed || !s.failed || !s.scanControl.failed || !m.failed || !m.closed || s.activeMeter.Load() != nil || s.stored != stored || s.scanControl.held != held || f.startLocked() == nil {
		t.Fatal("failure refunded a hold or retained a reusable finishing owner")
	}
}

// Fixed local state refusals only. None supplies a JSON/reopen factory, a
// caller-selected operating meter or an ACK from the cooperating-root fixture.
func TestProviderFirstAdmissionRejectsChangedOrUnownedPreparedState(t *testing.T) {
	for _, mode := range []string{"missing", "foreign", "spent", "work", "clocks", "upper", "clock-upper", "active", "body", "scan-pin", "main-pin", "origin", "limits", "closed-meter"} {
		t.Run(mode, func(t *testing.T) {
			s, f := providerFirstAdmissionOwnerFixture(t)
			m := f.meter
			s.mu.Lock()
			defer s.mu.Unlock()
			candidate := f
			switch mode {
			case "missing":
				candidate = nil
			case "foreign":
				copy := *f
				candidate = &copy
			case "spent":
				f.state = providerFirstAdmissionStarted
			case "work":
				m.actual[10]++
			case "clocks":
				m.clocks++
			case "upper":
				m.upper[10]--
			case "clock-upper":
				m.maximumClocks--
			case "active":
				s.activeMeter.Store(m)
			case "body":
				f.body.active = true
			case "scan-pin":
				f.scanPin.intent = "unselected DATA"
			case "main-pin":
				f.mainRaw = "unselected DATA"
			case "origin":
				f.origin.Epoch = "unselected epoch"
			case "limits":
				f.limits.clocks--
			case "closed-meter":
				m.closed = true
			}
			before, stored, held := m.actual, s.stored, s.scanControl.held
			if candidate.startLocked() == nil || m.actual != before || s.stored != stored || s.scanControl.held != held || s.last != nil || m.firstAdmission != nil {
				t.Fatal("changed state began finishing or altered the original holds", mode)
			}
		})
	}
}

func TestProviderFirstAdmissionKeepsFinishingAndGlobalCeilings(t *testing.T) {
	for _, mode := range []string{"decoded", "written", "work", "global", "clock", "clock-owner", "upper-change", "detached", "body-active"} {
		t.Run(mode, func(t *testing.T) {
			s, f := providerFirstAdmissionOwnerFixture(t)
			s.mu.Lock()
			defer s.mu.Unlock()
			if err := f.startLocked(); err != nil {
				t.Fatal(err)
			}
			m := f.meter
			kind, n := providerCopyEvent, uint64(0)
			switch mode {
			case "decoded":
				m.actual[2] = f.startWork[2] + f.limits.work[2]
				kind, n = providerCodecEvent, 1
			case "written":
				m.actual[9] = f.startWork[9] + f.limits.work[9]
				kind, n = providerWriteEvent, 1
			case "work":
				m.actual[10] = f.startWork[10] + f.limits.work[10]
			case "global":
				m.actual[10] = m.upper[10]
			case "clock":
				m.clocks = f.startClocks + f.limits.clocks
				kind = providerClockEvent
			case "clock-owner":
				s.clock.mu.Lock()
				s.clock.read = s.clock.maximum
				s.clock.mu.Unlock()
			case "upper-change":
				m.upper[10]--
			case "detached":
				s.activeMeter.Store(nil)
			case "body-active":
				f.body.active = true
			}
			before, stored, held := m.actual, s.stored, s.scanControl.held
			if mode == "clock-owner" {
				if _, err := f.sampleLocked(); err == nil || !s.failed || !m.closed || s.activeMeter.Load() != nil {
					t.Fatal("unknown owned sample renewed a finishing phase")
				}
			} else if m.event(kind, n) || !m.failed || m.actual != before {
				t.Fatal("finishing bypassed a sublimit or cumulative ceiling", mode)
			}
			if s.stored != stored || s.scanControl.held != held || s.last != nil || f.startLocked() == nil {
				t.Fatal("exhaustion refunded an original hold or produced an ACK")
			}
		})
	}
}

func TestProviderFirstAdmissionHasNoUnownedSampleOrFailureFactory(t *testing.T) {
	var missing *providerFirstAdmissionOwner
	if missing.startLocked() == nil || missing.fits(candidateWorkVector{}) || missing.UnmarshalJSON(nil) == nil {
		t.Fatal("missing owner constructed authority")
	}
	if _, err := missing.sampleLocked(); err == nil {
		t.Fatal("missing owner sampled a clock")
	}
	missing.abortLocked()
	s, f := providerFirstAdmissionOwnerFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := f.sampleLocked(); err == nil || f.state != providerFirstAdmissionPrepared || f.fits(candidateWorkVector{}) {
		t.Fatal("PREPARED DATA started a finishing primitive")
	}
	if owner, err := newProviderFirstAdmissionOwner(nil, nil, ClockSample{}); err == nil || owner != nil {
		t.Fatal("missing completed inspection produced a live owner")
	}
}
