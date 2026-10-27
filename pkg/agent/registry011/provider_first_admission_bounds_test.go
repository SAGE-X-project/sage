package registry011

import "testing"

func providerFirstAdmissionBoundsFixture() providerFirstAdmissionBounds {
	return providerFirstAdmissionBounds{work: candidateWorkVector{0, 0, 1 << 20, 0, 0, 0, 0, 0, 0, 1 << 20, 1 << 50}, retained: 2 << 20, live: 2 << 20, clocks: 64, durationMS: 300000}
}

func TestProviderFirstAdmissionHasDistinctPreprovisionedProfile(t *testing.T) {
	first := providerFirstAdmissionBoundsFixture()
	config, clock := providerScanProvisionedFixtureWithFirst(t, &first)
	old := config.bounds
	old.firstAdmission = nil
	oldRaw, oldDomain := providerIntegrationProfileBytes(old, config.capture)
	raw, domain := providerIntegrationProfileBytes(config.bounds, config.capture)
	m := contentJSON(raw).(map[string]any)
	store := m["store"].(map[string]any)
	if domain != "sage-provider-store-profile-v4" || oldDomain != "sage-provider-store-profile-v3" || m["format"] != "sage-provider-owned-profile-v4" || store["live_accounting"] != providerFirstAdmissionLiveAccounting || store["first_admission"] == nil || string(raw) == string(oldRaw) || providerIntegrationProfile(old, config.capture) == config.profile {
		t.Fatal("old H0 accounting was silently adopted")
	}
	s, err := openProviderFrontierCandidate(config, clock)
	if err != nil {
		t.Fatal("ordinary fresh V4 DATA open", err)
	}
	t.Cleanup(func() { _ = s.close() })
	if s.firstLimits == nil || !providerFirstAdmissionCapacity(s, providerScanOpenUpper(config)) || s.last != nil || s.sequence != 0 {
		t.Fatal("capacity data became an enrollment or ACK")
	}
	full, err := providerScanBodyUpper(config)
	oldBody, oldErr := providerScanBodyUpper(providerCustodyConfig{bounds: old})
	// V4 additionally funds the two closed receipt decoders, independently of
	// the four original readbacks; old V3 keeps its original output bound.
	oldBody.work[2] += 2 * config.bounds.maximumMetadataPerCommit * config.bounds.scan.maximumChunks
	want, ok := providerScanUpperSum(oldBody, providerScanUpper{first.work, first.retained, first.clocks})
	if err != nil || oldErr != nil || !ok || full != want {
		t.Fatal("first-E1 costs were not held in the complete B before I", err, oldErr)
	}
	if _, err := s.enrollBeforeScan(); err != nil {
		t.Fatal("original O-funded enrollment before inspection", err)
	}
	if err := s.scanControl.claim(providerScanAttemptFixture(t, s, s.scanControl.pin)); err != nil || s.scanControl.held.retained < first.retained || s.last != nil {
		t.Fatal("ordinary fresh claim omitted its finite first-E1 hold", err)
	}
	wrong := config
	wrong.profile = providerIntegrationProfile(old, config.capture)
	if _, err := newProviderScanCodec(wrong, nil); err == nil {
		t.Fatal("old V3 profile accepted as a V4 bridge profile")
	}
}

func TestProviderFirstAdmissionFiniteBoundsRejectIncompleteEnvelopes(t *testing.T) {
	base, _ := providerScanProvisionedFixture(t)
	base.bounds.setupLive = base.bounds.setupRetained
	for _, mode := range []string{"ordinary", "no-scan", "open-live", "zero", "live", "retained", "clocks", "duration", "decoded", "written", "work", "request", "coordinate"} {
		t.Run(mode, func(t *testing.T) {
			b, first := base.bounds, providerFirstAdmissionBoundsFixture()
			switch mode {
			case "no-scan":
				b.scan = nil
			case "open-live":
				b.setupLive++
			case "zero":
				first.retained, first.live = 0, 0
			case "live":
				first.live++
			case "retained":
				first.retained, first.live = b.maximumStoredBytes+1, b.maximumStoredBytes+1
			case "clocks":
				first.clocks = 0
			case "duration":
				first.durationMS = 0
			case "decoded":
				first.work[2] = 0
			case "written":
				first.work[9] = first.retained + 1
			case "work":
				first.work[10] = 0
			case "request":
				first.work[0] = 1
			case "coordinate":
				first.work[10] = b.scan.maximumPreIWork[10] + 1
			}
			b.firstAdmission = &first
			if b.valid() != (mode == "ordinary") {
				t.Fatal("incomplete or unfunded candidate envelope accepted", mode)
			}
		})
	}
}

func TestProviderFirstAdmissionCachesObservedLimitsBeforeAnyClaim(t *testing.T) {
	first := providerFirstAdmissionBoundsFixture()
	config, clock := providerScanProvisionedFixtureWithFirst(t, &first)
	s, err := openProviderFrontierCandidate(config, clock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.close() })
	if _, err := s.enrollBeforeScan(); err != nil {
		t.Fatal("funded candidate enrollment before claim capacity checks", err)
	}
	want := *s.firstLimits
	for _, mode := range []string{"missing", "live", "retained", "work"} {
		t.Run(mode, func(t *testing.T) {
			limits := want
			s.firstLimits = &limits
			switch mode {
			case "missing":
				s.firstLimits = nil
			case "live":
				limits.live = 0
			case "retained":
				limits.retained = 0
			case "work":
				limits.work[10] = 0
			}
			held, stored := s.scanControl.held, s.stored
			if err := s.scanControl.claim(providerScanAttemptFixture(t, s, s.scanControl.pin)); err == nil || s.stored != stored || s.scanControl.held != held || s.scanControl.prepared != nil || s.failed {
				t.Fatal("capacity refusal spent or replenished a claim", mode)
			}
		})
	}
}

func TestProviderFirstAdmissionLimitDecoderRefusesMalformedOrUnfundedData(t *testing.T) {
	first := providerFirstAdmissionBoundsFixture()
	config, _ := providerScanProvisionedFixtureWithFirst(t, &first)
	for _, raw := range []string{`[]`, `{"limits":[]}`, `{"limits":{"bootstrap":{},"live_disk_bytes":"1","retained_bytes":"1"}}`} {
		// Inert parser DATA only: this object never opens roots or creates an I.
		c := config
		c.policy = &ProviderPolicyCandidate{raw: raw}
		s := &providerCustodyStore{config: c}
		x := &providerScanCodec{config: c}
		if s.firstAdmissionLimits(x) || s.firstLimits != nil {
			t.Fatal("invalid limit bytes populated a cached capacity")
		}
	}
	s := &providerCustodyStore{config: config}
	control := &providerScanControl{store: s}
	control.owner = control
	m := &providerWorkMeter{scan: control}
	m.owner = m
	if s.firstAdmissionLimits(&providerScanCodec{config: config, meter: m}) || !m.failed || s.firstLimits != nil {
		t.Fatal("unfunded decoder populated a capacity")
	}
}
