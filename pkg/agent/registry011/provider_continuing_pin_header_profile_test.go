package registry011

import (
	"context"
	"strconv"
	"testing"
)

func TestProviderContinuingPinHeaderSelectionFundsTheOriginalAdmission(t *testing.T) {
	for _, tc := range []struct {
		name          string
		wide, liveTLS bool
	}{
		{"default", false, false},
		{"wide", true, false},
		{"default-live-tls", false, true},
		{"wide-live-tls", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			selection := &providerScanFixtureSelection{pin: true, pinResidentGiB: true, pinRPCHeader: tc.wide}
			if tc.liveTLS {
				tls := providerTLSBounds{2, 5, 8 << 10}
				selection.tls = &tls
			}
			config, clock := providerGuardianConfigWithSelectionFixture(t, 64<<20, 6, selection)
			wantBody, wantWire, wantBootstrap := 16<<10, 32<<10, uint64(256<<10)
			if tc.wide {
				wantBody, wantWire, wantBootstrap = 96<<10, 160<<10, 1<<20
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

func TestProviderContinuingPinHeaderSelectionCompletesOwnedExchanges(t *testing.T) {
	for _, liveTLS := range []bool{false, true} {
		t.Run(strconv.FormatBool(liveTLS), func(t *testing.T) {
			selection := &providerScanFixtureSelection{pin: true, pinResidentGiB: true, pinRPCHeader: true}
			if liveTLS {
				tls := providerTLSBounds{2, 5, 8 << 10}
				selection.tls = &tls
			}
			// Ordinary owned TLS exercises the first RPC, local BootstrapStart,
			// four pin RPCs and page closures with this SAME fresh profile.
			s, g, digests := providerContinuingPinReadyWithSelectionFixture(t, selection)
			origin, initial := s.setupOrigin, g.integration.firstExchange
			firstE, firstR := g.first.execution.actual, g.first.remaining.actual
			pin, err := newProviderContinuingPinCandidate(s, "example-agent", digests)
			if err != nil {
				t.Fatal("wide profile local pin admission", err)
			}
			value, err := pin.executePinCandidate(context.Background())
			if err != nil || value == nil {
				t.Fatal("wide profile original pin completion", err)
			}
			if g.first.plan.capture.BodyBytes != 96<<10 || g.first.plan.capture.WireBytes != 160<<10 ||
				value.exchanges != digests || value.pin != g.integration.pin || value.selected != admissionTestDigest("a") ||
				s.setupOrigin != origin || g.original != origin || g.integration.firstExchange != initial ||
				g.first.execution.actual != firstE || g.first.remaining.actual != firstR ||
				s.sequence != 25 || g.ordinal != 9 || !g.ready || g.active || g.continuing != nil || s.activeMeter.Load() != nil {
				t.Fatal("wide profile changed original limits or reset its retained lifetime")
			}
		})
	}
}
