package registry011

import (
	"strconv"
	"testing"
)

func TestProviderContinuingPinHeaderSelectionFundsTheOriginalAdmission(t *testing.T) {
	for _, wide := range []bool{false, true} {
		t.Run(strconv.FormatBool(wide), func(t *testing.T) {
			selection := &providerScanFixtureSelection{pin: true, pinResidentGiB: true, pinRPCHeader: wide}
			config, clock := providerGuardianConfigWithSelectionFixture(t, 64<<20, 6, selection)
			wantBody, wantWire, wantBootstrap := 16<<10, 32<<10, uint64(256<<10)
			if wide {
				wantBody, wantWire, wantBootstrap = 128<<10, 192<<10, 1<<20
			}
			if config.bounds.methods.header != (providerMethodLimit{wantBody, wantWire}) ||
				config.capture.BodyBytes != 128<<10 || config.capture.WireBytes != 256<<10 ||
				config.bounds.maximumObjectBytes != 256<<10 {
				t.Fatal("new header class changed an unrelated envelope or old default")
			}
			policy := contentJSON(config.policy.CanonicalBytes()).(map[string]any)
			limits := policy["limits"].(map[string]any)
			for _, level := range []string{"operation", "segment", "bootstrap"} {
				want := uint64(256 << 10)
				if level == "bootstrap" {
					want = wantBootstrap
				}
				got, ok := providerCandidateUint(limits[level].(map[string]any)["wire_bytes"], true)
				if !ok || got != want {
					t.Fatal("pre-H0 policy did not fund the exact selected lifetime", level, got, want)
				}
			}
			s, err := openProviderFrontierCandidate(config, clock)
			if err != nil || s == nil {
				t.Fatal("original fresh profile open", err)
			}
			t.Cleanup(func() { _ = s.close() })
			if _, err := s.enrollBeforeScan(); err != nil {
				t.Fatal("original enrollment", err)
			}
			codec := mustProviderBindingCodec(t, s.config)
			target, err := codec.target(providerScanTarget{setup: s.setupTip, custody: s.tip, frontier: s.frontierFiles.pin.frontier, objects: s.objectsRoot, enrollment: s.enrollment})
			if err != nil {
				t.Fatal(err)
			}
			attempt := providerScanAttemptFixture(t, s, s.scanControl.pin)
			attempt.target = codec.hash("sage-provider-scan-target-v1", target)
			if err := s.scanControl.claim(attempt); err != nil {
				t.Fatal("original initial scan claim", err)
			}
			if _, err := s.scanControl.inspectBeforeAdmission(target); err != nil {
				t.Fatal("complete original scan", err)
			}
			first, err := newProviderFirstOperationCandidate(s)
			if err != nil || first == nil || first.plan.capture.BodyBytes != wantBody || first.plan.capture.WireBytes != wantWire {
				t.Fatal("same original admission cannot fund its selected header", err)
			}
			// Do not dispatch: this UNIT validates the genuinely retained
			// preflight. The existing official-RPC runtime exercises transport.
		})
	}
}
