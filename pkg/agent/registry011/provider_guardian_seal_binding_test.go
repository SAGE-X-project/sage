package registry011

import "testing"

// Pure candidate codec DATA, with no open store, live G, filesystem or RPC.
func TestProviderGuardianSealBindingRequiresExactBoundaryAndCanonicalBytes(t *testing.T) {
	t.Parallel()
	config, _ := providerGuardianConfigWithClaimFixture(t, 64<<20)
	x := mustProviderBindingCodec(t, config)
	h := admissionTestDigest("a")
	v := providerGuardianSealBinding{ordinal: 1, pageIndex: 0, records: config.bounds.continuation.pageRecords,
		claim: h, head: h, previous: h, main: h, frontier: h, setup: h, admission: h, operation: h, budget: h,
		page: h, seal: h, indexes: [3]string{h, h, h}, clock: CandidateClockDiagnostic{ClockID: "unit-clock", Epoch: "unit-epoch", WallMS: 100, MonotonicMS: 100}}
	var err error
	v.target, err = x.guardianSealTarget(v)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := x.guardianSealBinding(v)
	parsed, parseErr := x.parseGuardianSealBinding(raw)
	if err != nil || parseErr != nil || parsed != v {
		t.Fatal("canonical closed binding DATA", err, parseErr)
	}
	for _, mode := range []string{"ordinal", "claim", "clock", "page", "records", "budget", "target"} {
		t.Run(mode, func(t *testing.T) {
			changed := v
			switch mode {
			case "ordinal":
				changed.ordinal = 0
			case "claim":
				changed.claim = ""
			case "clock":
				changed.clock.Epoch = ""
			case "page":
				changed.pageIndex = config.bounds.continuation.maximumPages
			case "records":
				changed.records--
			case "budget":
				changed.budget = ""
			case "target":
				changed.target = admissionTestDigest("f")
			}
			if _, err := x.guardianSealBinding(changed); err == nil {
				t.Fatal("different or incomplete boundary produced binding DATA", mode)
			}
		})
	}
	for _, mode := range []string{"noncanonical", "purpose", "number", "extension"} {
		t.Run(mode, func(t *testing.T) {
			value := contentJSON(raw).(map[string]any)
			switch mode {
			case "purpose":
				value["purpose"] = "rpc-admission"
			case "number":
				value["page_index"] = "00"
			case "extension":
				value["unknown"] = true
			}
			candidate := providerIntegrationJSON(value)
			if mode == "noncanonical" {
				candidate = append(candidate, '\n')
			}
			if _, err := x.parseGuardianSealBinding(candidate); err == nil {
				t.Fatal("unagreed or noncanonical binding DATA accepted", mode)
			}
		})
	}
	if _, err := (*providerScanCodec)(nil).guardianSealTarget(v); err == nil {
		t.Fatal("missing codec produced target DATA")
	}
	if _, err := (*providerScanCodec)(nil).parseGuardianSealBinding(raw); err == nil {
		t.Fatal("missing codec parsed binding DATA")
	}
	if _, err := providerGuardianSealBindingForBoundary(x, nil, v); err == nil {
		t.Fatal("missing retained G original produced binding DATA")
	}
}
