package registry011

import "testing"

func TestProviderFirstAdmissionReplaysOriginalEnrollmentWithFreshMaps(t *testing.T) {
	s, f := providerFirstAdmissionOwnerFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := f.startLocked(); err != nil {
		t.Fatal(err)
	}
	old := f.body.reader
	before, clocks, stored, held := f.meter.actual, f.meter.clocks, s.stored, s.scanControl.held
	// Ordinary provisional caches are deliberately absent. The independent
	// reader must reconstruct its inputs from the retained owned originals.
	s.objects = map[string]providerIntegrationObjectRef{"unused-provisional-data": {}}
	s.operations = map[string]bool{"unused-provisional-data": true}
	s.setupCommits, s.setups = nil, nil
	s.frontierReplay.body.anchor = nil
	r, err := f.replayEnrollmentLocked()
	if err != nil || r == nil || r == old || r.view == s || r.replay == s.frontierReplay || r.replay.body.anchor == nil || r.replay.body.anchor.digest != s.enrollment || r.replay.body.anchor.budget != "" || r.replay.body.intent != nil || r.replay.body.ledger != nil || r.selected != f.mainPin || r.outputUpper != f.readerOutput || len(r.view.objects) == 0 || len(r.view.setups) != 1 || !r.lineage() {
		t.Fatal("first finishing adopted old caches or skipped original E0", err)
	}
	if f.body.reader != nil || f.body.receipts != nil || f.body.required != nil || f.body.observed != nil || f.body.targetRaw != "" || f.body.active || !f.replayStarted || f.state != providerFirstAdmissionStarted || f.meter.closed || f.meter.failed || f.meter.actual[2] <= before[2] || f.meter.actual[10] <= before[10] || f.meter.clocks != clocks+f.mainPin.ordinal+3 || s.stored != stored || s.scanControl.held != held || s.last != nil {
		t.Fatal("finishing reset B, refunded a hold or returned authority")
	}
	actual := f.meter.actual
	if again, err := f.replayEnrollmentLocked(); err == nil || again != nil || f.meter.actual != actual || f.body.active {
		t.Fatal("finishing restarted the closed inspection")
	}
}

func TestProviderFirstAdmissionOriginalReplayRefusesIncompleteOrUnownedInputs(t *testing.T) {
	for _, mode := range []string{"missing", "foreign", "prepared", "detached", "main-pin", "body", "missing-cache", "output", "target", "decoded", "clocks", "clock-owner", "end-clock", "missing-setup", "missing-enrollment", "memory"} {
		t.Run(mode, func(t *testing.T) {
			s, f := providerFirstAdmissionOwnerFixture(t)
			s.mu.Lock()
			defer s.mu.Unlock()
			candidate := f
			if mode != "prepared" {
				if err := f.startLocked(); err != nil {
					t.Fatal(err)
				}
			}
			switch mode {
			case "missing":
				candidate = nil
			case "foreign":
				copy := *f
				candidate = &copy
			case "prepared":
			case "detached":
				s.activeMeter.Store(nil)
			case "main-pin":
				s.frontierFiles.pinRaw = "unselected finite pin DATA"
			case "body":
				f.body.active = true
			case "missing-cache":
				f.body.reader = nil
			case "output":
				f.readerOutput = 0
			case "target":
				f.targetRaw = "inert invalid target DATA"
			case "decoded":
				f.meter.actual[2] = f.startWork[2] + f.limits.work[2]
			case "clocks":
				f.meter.clocks = f.startClocks + f.limits.clocks
			case "clock-owner":
				s.clock.mu.Lock()
				s.clock.read = s.clock.maximum
				s.clock.mu.Unlock()
			case "end-clock":
				s.clock.mu.Lock()
				s.clock.maximum = s.clock.read + f.mainPin.ordinal + 2
				s.clock.mu.Unlock()
			case "missing-setup":
				if err := s.evidence.root.Remove(providerCustodyName("setup-claim", 0)); err != nil {
					t.Fatal(err)
				}
			case "missing-enrollment":
				if err := s.evidence.root.Remove(providerObjectName("enrollment", s.enrollment)); err != nil {
					t.Fatal(err)
				}
			case "memory":
				f.readerOutput = ^uint64(0)
			}
			stored, held := s.stored, s.scanControl.held
			if reader, err := candidate.replayEnrollmentLocked(); err == nil || reader != nil || s.stored != stored || s.scanControl.held != held || s.last != nil {
				t.Fatal("unowned or incomplete original E0 became first-I DATA", mode)
			}
			if mode != "missing" && mode != "foreign" && mode != "prepared" && (!s.failed || !f.meter.closed || f.state != providerFirstAdmissionFailed) {
				t.Fatal("uncertain finishing retained a reusable owner", mode)
			}
		})
	}
}
