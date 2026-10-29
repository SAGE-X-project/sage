package registry011

import "testing"

func TestProviderAdmissionReceiptReplayChecksBothOriginalChainsWithoutRestoringBody(t *testing.T) {
	s, f, _, binding := providerScanBindingFixture(t)
	original, readErr := s.evidence.read(providerScanReceiptName("chunk", binding.lastChunk, nil), int64(s.config.bounds.maximumMetadataPerCommit))
	last, ok := contentJSON(original).(map[string]any)
	objects, objectsOK := providerCandidateUint(last["cumulative_objects"], false)
	bytes, bytesOK := providerCandidateUint(last["cumulative_bytes"], false)
	if readErr != nil || !ok || !objectsOK || !bytesOK {
		t.Fatal("original last-record fixture diagnostics", readErr)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := f.startLocked(); err != nil {
		t.Fatal(err)
	}
	before, clocks, stored, held := f.meter.actual, f.meter.clocks, s.stored, s.scanControl.held
	var prior *providerScanReceiptClosure
	for n := 0; n < 2; n++ {
		closure, err := replayProviderAdmissionReceipts(s, f.meter, binding)
		// The retained record ends before its own encoding/write/readbacks and
		// second catalog. Compare the exact recorded snapshot, while keeping
		// those later operations inside the separate cumulative live B meter.
		if err != nil || closure == nil || closure.records != f.records || closure.objects != objects || closure.bytes != bytes || closure.objects > f.body.objects || closure.bytes > f.body.rawBytes || closure.metadata == 0 || closure.end.ClockID != f.origin.ClockID || closure.end.Epoch != f.origin.Epoch || f.meter.actual[2] <= before[2] || f.meter.actual[10] <= before[10] || f.meter.clocks != clocks+f.records || f.body.active || s.last != nil || s.stored != stored || s.scanControl.held != held || closure.UnmarshalJSON(nil) == nil {
			t.Fatal("original receipt replay reset B, reused measurement or restored an owner", err)
		}
		if prior != nil && *prior != *closure {
			t.Fatal("same original chain reconstructed different diagnostics")
		}
		before, clocks, prior = f.meter.actual, f.meter.clocks, closure
	}
}

func TestProviderAdmissionReceiptReplayRefusesAbsentOriginalsAndUnownedWork(t *testing.T) {
	for _, mode := range []string{"detached", "closed-meter", "profile", "binding", "budget", "missing-chunk", "missing-command", "clock"} {
		t.Run(mode, func(t *testing.T) {
			s, f, _, binding := providerScanBindingFixture(t)
			s.mu.Lock()
			defer s.mu.Unlock()
			if err := f.startLocked(); err != nil {
				t.Fatal(err)
			}
			m := f.meter
			switch mode {
			case "detached":
				s.activeMeter.Store(nil)
			case "closed-meter":
				m.closed = true
			case "profile":
				s.config.profile = admissionTestDigest("b")
			case "binding":
				binding.lastChunk = "missing finite digest"
			case "budget":
				m.actual[2] = f.startWork[2] + f.limits.work[2]
			case "missing-chunk", "missing-command":
				kind, digest := "chunk", binding.lastChunk
				if mode == "missing-command" {
					kind, digest = "command", f.body.receipts.commands[len(f.body.receipts.commands)-1]
				}
				if err := s.evidence.root.Remove(providerScanReceiptName(kind, digest, nil)); err != nil {
					t.Fatal(err)
				}
			case "clock":
				s.clock.mu.Lock()
				s.clock.read = s.clock.maximum
				s.clock.mu.Unlock()
			}
			stored, held := s.stored, s.scanControl.held
			if closure, err := replayProviderAdmissionReceipts(s, m, binding); err == nil || closure != nil || s.stored != stored || s.scanControl.held != held || s.last != nil || f.body.active {
				t.Fatal("missing or unowned original chain supplied a closure", mode)
			}
		})
	}
	if closure, err := replayProviderAdmissionReceipts(nil, nil, providerScanAdmissionBinding{}); err == nil || closure != nil {
		t.Fatal("nil owner supplied a retained receipt closure")
	}
}

func TestProviderAdmissionReceiptCursorDataRejectsSkippedAndInvalidTransitions(t *testing.T) {
	for _, pair := range []struct {
		start, end providerScanCursor
		valid      bool
	}{
		{providerScanCursor{"setup", 0}, providerScanCursor{"setup", 1}, true},
		{providerScanCursor{"setup", 1}, providerScanCursor{"main", 1}, true},
		{providerScanCursor{"main", 1}, providerScanCursor{"main", 2}, true},
		{providerScanCursor{"main", 2}, providerScanCursor{"done", 0}, true},
		{providerScanCursor{"setup", 0}, providerScanCursor{"setup", 2}, false},
		{providerScanCursor{"setup", 0}, providerScanCursor{"done", 0}, false},
		{providerScanCursor{"main", 1}, providerScanCursor{"main", 3}, false},
		{providerScanCursor{"done", 0}, providerScanCursor{"main", 1}, false},
	} {
		if providerScanReceiptStep(pair.start, pair.end) != pair.valid {
			t.Fatal("closed record skipped or changed cursor order", pair)
		}
		if cursor, ok := providerScanReceiptCursor(contentJSON(providerIntegrationJSON(pair.start.object()))); !ok || cursor != pair.start {
			t.Fatal("canonical cursor DATA did not roundtrip")
		}
	}
	for _, data := range []any{nil, map[string]any{"phase": "main", "next_sequence": "0"}, map[string]any{"phase": "done", "next_sequence": "1"}, map[string]any{"phase": "setup", "next_sequence": "00"}, map[string]any{"phase": "setup", "next_sequence": "0", "extra": "inert DATA"}} {
		if _, ok := providerScanReceiptCursor(data); ok {
			t.Fatal("invalid fixed cursor DATA")
		}
	}
	_, _, x, binding := providerScanBindingFixture(t)
	if _, _, err := readProviderAdmissionReceipt(nil, x, binding, "other", "invalid"); err == nil {
		t.Fatal("unknown receipt name began IO")
	}
}

func TestProviderAdmissionReceiptClockDataKeepsOriginalDurationAndObservedOrder(t *testing.T) {
	// Pure finite clock-comparison DATA; no clock source, meter, root or network
	// is constructed or replaced by these values.
	origin := CandidateClockDiagnostic{ClockID: "owned-clock", Epoch: "one-epoch", WallMS: 1000, MonotonicMS: 100}
	start, end := origin, origin
	start.WallMS, start.MonotonicMS = 1010, 110
	end.WallMS, end.MonotonicMS = 1020, 120
	if !providerScanReceiptOriginalTime(origin, start, end, 1100, 10, 20) || !providerScanReceiptObservedTime(origin, end, end, 1100, 20) {
		t.Fatal("exact original time limits were refused")
	}
	for _, mode := range []string{"total", "per-record", "epoch", "before-origin", "before-start", "deadline"} {
		a, b, first, deadline, perRecord, total := start, end, origin, uint64(1100), uint64(10), uint64(20)
		switch mode {
		case "total":
			// Five milliseconds is a valid record, but30 exceeds original20.
			a.WallMS, a.MonotonicMS, b.WallMS, b.MonotonicMS = 1025, 125, 1030, 130
		case "per-record":
			perRecord = 9
		case "epoch":
			b.Epoch = "different-epoch"
		case "before-origin":
			first.WallMS, first.MonotonicMS = 1011, 111
		case "before-start":
			b.WallMS, b.MonotonicMS = 1009, 109
		case "deadline":
			deadline = 1019
		}
		if providerScanReceiptOriginalTime(first, a, b, deadline, perRecord, total) {
			t.Fatal("recorded time escaped original clock/duration/deadline", mode)
		}
	}
	for _, mode := range []string{"future-record", "observed-total", "observed-epoch", "observed-deadline"} {
		recorded, sample, deadline, total := end, end, uint64(1100), uint64(40)
		switch mode {
		case "future-record":
			recorded.WallMS, recorded.MonotonicMS = 1030, 130
			sample.WallMS, sample.MonotonicMS = 1025, 125
			if !providerScanReceiptOriginalTime(origin, start, recorded, deadline, 20, total) {
				t.Fatal("finite future-record scenario must fit the recorded limits")
			}
		case "observed-total":
			total = 19
		case "observed-epoch":
			sample.Epoch = "different-epoch"
		case "observed-deadline":
			deadline = 1019
		}
		if providerScanReceiptObservedTime(origin, recorded, sample, deadline, total) {
			t.Fatal("owned observation preceded a recorded end or renewed limits", mode)
		}
	}
}
