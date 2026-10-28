package registry011

import "testing"

func providerScanBindingFixture(t *testing.T) (*providerCustodyStore, *providerFirstAdmissionOwner, *providerScanCodec, providerScanAdmissionBinding) {
	t.Helper()
	s, f := providerFirstAdmissionOwnerFixture(t)
	x, err := newProviderScanCodec(s.config, nil)
	if err != nil {
		t.Fatal(err)
	}
	total, ok := providerScanUpperSum(providerScanOpenUpper(s.config), f.scanPin.held)
	if !ok {
		t.Fatal("finite fixture held sum")
	}
	r := f.body.receipts
	v := providerScanAdmissionBinding{e0: f.mainPin.commit, frontier: f.mainPin.frontier, setup: f.mainPin.setup, head: r.head, highwater: r.highwater, attempt: r.attempt, target: f.attempt.target, lastChunk: f.lastChunk, admission: admissionTestDigest("a"), ordinal: f.scanPin.ordinal, deadline: f.attempt.deadline, live: total.retained, clock: providerClockDiagnostic(f.origin), open: providerScanOpenUpper(s.config), held: f.scanPin.held, total: total}
	return s, f, x, v
}

func TestProviderScanAdmissionBindingIsClosedBackwardOnlyData(t *testing.T) {
	s, f, x, v := providerScanBindingFixture(t)
	before, stored, held := f.meter.actual, s.stored, s.scanControl.held
	raw, err := x.admissionBinding(v)
	got, parseErr := x.parseAdmissionBinding(raw)
	if err != nil || parseErr != nil || got != v || f.state != providerFirstAdmissionPrepared || f.meter.actual != before || s.stored != stored || s.scanControl.held != held || s.activeMeter.Load() != nil || s.last != nil || v.UnmarshalJSON(raw) == nil {
		t.Fatal("canonical binding DATA selected a current owner or lost a prior hold", err, parseErr)
	}
	m := contentJSON(raw).(map[string]any)
	for key := range m {
		t.Run("missing-"+key, func(t *testing.T) {
			copy := contentJSON(raw).(map[string]any)
			delete(copy, key)
			if _, err := x.parseAdmissionBinding(providerIntegrationJSON(copy)); err == nil {
				t.Fatal("binding admitted a missing closed field", key)
			}
		})
	}
	m["future_checkpoint"] = admissionTestDigest("b")
	if _, err := x.parseAdmissionBinding(providerIntegrationJSON(m)); err == nil {
		t.Fatal("future or optional unknown binding field")
	}
}

func TestProviderScanAdmissionBindingRefusesInconsistentOriginalBounds(t *testing.T) {
	_, _, x, base := providerScanBindingFixture(t)
	for _, mode := range []string{"legacy", "ordinal-zero", "ordinal-upper", "empty-hold", "digest", "open", "sum", "live", "clock", "deadline", "unfunded", "identity"} {
		t.Run(mode, func(t *testing.T) {
			v, codec := base, *x
			switch mode {
			case "legacy":
				codec.config.bounds.firstAdmission = nil
			case "ordinal-zero":
				v.ordinal = 0
			case "ordinal-upper":
				v.ordinal = codec.config.bounds.scan.maximumAttempts + 1
			case "empty-hold":
				v.held = providerScanUpper{}
			case "digest":
				v.lastChunk = "unselected finite DATA"
			case "open":
				v.open.work[10]--
			case "sum":
				v.total.work[2]--
			case "live":
				v.live--
			case "clock":
				v.clock.MonotonicMS = -1
			case "deadline":
				v.deadline++
			case "unfunded":
				s, r, _ := providerScanReceiptObserverUnit(t)
				codec.config, codec.meter = s.config, r.body.meter
				codec.meter.upper[10] = codec.meter.actual[10]
			case "identity":
				raw, err := codec.admissionBinding(v)
				if err != nil {
					t.Fatal(err)
				}
				m := contentJSON(raw).(map[string]any)
				m["profile"] = admissionTestDigest("b")
				if _, err := codec.parseAdmissionBinding(providerIntegrationJSON(m)); err == nil {
					t.Fatal("binding was adopted into a different original identity")
				}
				return
			}
			if _, err := codec.admissionBinding(v); err == nil {
				t.Fatal("binding changed original holds, clock or profile", mode)
			}
		})
	}
}

func TestProviderScanAdmissionControlsDeriveSeedFromOriginalsEachTime(t *testing.T) {
	s, f, _, binding := providerScanBindingFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := f.startLocked(); err != nil {
		t.Fatal(err)
	}
	before, stored, held := f.meter.actual, s.stored, s.scanControl.held
	for n := 0; n < 2; n++ {
		seed, err := replayProviderAdmissionControls(s, f.meter, binding)
		if err != nil || seed == nil || seed.work != binding.total.work || seed.retained != binding.total.retained || f.meter.actual[2] <= before[2] || f.meter.actual[10] <= before[10] || s.stored != stored || s.scanControl.held != held || s.last != nil || f.body.active {
			t.Fatal("control seed used a previous measurement, reset B or became an ACK", err)
		}
		before = f.meter.actual
	}
	// A different provisional current pin/totals cannot replace the historical
	// originals. This result is DATA, not proof that the new current pin agrees.
	s.scanControl.pin, s.scanControl.held = providerScanHighwater{}, providerScanUpper{}
	seed, err := replayProviderAdmissionControls(s, f.meter, binding)
	if err != nil || seed == nil || seed.retained != binding.total.retained || seed.work != binding.total.work || s.last != nil {
		t.Fatal("historical original seed was replaced by a later provisional pin", err)
	}
}

func TestProviderScanAdmissionControlsRefuseMissingAndMismatchedEvidence(t *testing.T) {
	for _, mode := range []string{"detached", "budget", "profile", "initial-head", "root", "missing-intent", "missing-commit", "missing-highwater", "binding", "target", "highwater", "attempt", "setup", "custody", "sum"} {
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
			case "budget":
				m.actual[10] = m.upper[10]
			case "profile":
				s.config.profile = admissionTestDigest("b")
			case "initial-head":
				s.config.expectedScanHead = admissionTestDigest("b")
			case "root":
				_ = s.evidence.root.Close()
			case "missing-intent", "missing-commit", "missing-highwater":
				kind, disk := "scan-intent", s.custody
				if mode == "missing-commit" {
					kind = "scan-commit"
				}
				if mode == "missing-highwater" {
					kind, disk = "scan-highwater", s.evidence
				}
				// Ordinary absent-file unit, limited to this owned fixture root.
				if err := disk.root.Remove(providerCustodyName(kind, binding.ordinal)); err != nil {
					t.Fatal(err)
				}
			case "binding":
				binding.lastChunk = "inert missing digest"
			case "target":
				binding.target = admissionTestDigest("b")
			case "highwater":
				binding.highwater = admissionTestDigest("b")
			case "attempt":
				binding.attempt = admissionTestDigest("b")
			case "setup":
				binding.setup = admissionTestDigest("b")
			case "custody":
				binding.e0 = admissionTestDigest("b")
			case "sum":
				binding.total.retained--
			}
			stored, held := s.stored, s.scanControl.held
			if seed, err := replayProviderAdmissionControls(s, m, binding); err == nil || seed != nil || s.stored != stored || s.scanControl.held != held || s.last != nil {
				t.Fatal("missing or mismatched originals supplied an initial seed", mode)
			}
		})
	}
	if seed, err := replayProviderAdmissionControls(nil, nil, providerScanAdmissionBinding{}); err == nil || seed != nil {
		t.Fatal("unowned seed reconstruction")
	}
}
