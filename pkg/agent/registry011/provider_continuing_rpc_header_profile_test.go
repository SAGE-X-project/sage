package registry011

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
)

// Select the existing isolated header profile before its new anchor and clock.
// Ordinary scalar requests retain their smaller method class. This test input
// neither changes a production limit nor retries an already refused owner.
func providerContinuingRPCHeaderSelectionFixture(capture *ProviderCaptureLimits, certificates *providerTLSBounds) *providerScanFixtureSelection {
	return &providerScanFixtureSelection{capture: capture, tls: certificates, continuingWire: true, pin: true, pinResidentGiB: true, pinRPCHeader: true}
}

func TestProviderContinuingRPCHeaderProfileRetainsLargeFirstResponseAndOriginalLifetime(t *testing.T) {
	t.Parallel()
	transactions := make([]string, 300)
	for i := range transactions {
		transactions[i] = fmt.Sprintf("0x%064x", i+1)
	}
	response, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "result": map[string]any{
		"hash": admissionTestDigest("a"), "parentHash": admissionTestDigest("b"),
		"number": "0x" + strconv.FormatUint(sepoliaDeploymentBlock+1, 16), "timestamp": "0x1", "transactions": transactions,
	}})
	if err != nil || len(response) <= 16<<10 || len(response) >= 32<<10 {
		t.Fatal("bounded ordinary finalized response fixture", err)
	}
	for _, wide := range []bool{false, true} {
		t.Run(strconv.FormatBool(wide), func(t *testing.T) {
			selection := providerContinuingRPCHeaderSelectionFixture(nil, nil)
			selection.pinRPCHeader = wide
			config, clock := providerGuardianConfigWithSelectionFixture(t, 64<<20, 6, selection)
			wantHeader := providerMethodLimit{16 << 10, 32 << 10}
			if wide {
				wantHeader = providerMethodLimit{96 << 10, 160 << 10}
			}
			if config.bounds.methods.header != wantHeader || config.bounds.methods.scalar != (providerMethodLimit{1024, 16 << 10}) ||
				config.capture.BodyBytes != 128<<10 || config.capture.WireBytes != 256<<10 || config.bounds.maximumObjectBytes != 256<<10 {
				t.Fatal("header selection changed a scalar, global or object limit")
			}
			s, err := openProviderFrontierCandidate(config, clock)
			if err != nil {
				t.Fatal("fresh isolated header profile", err)
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
				t.Fatal("original initial-scan claim", err)
			}
			if _, err := s.scanControl.inspectBeforeAdmission(target); err != nil {
				t.Fatal("complete original scan", err)
			}
			first, err := newProviderFirstOperationCandidate(s)
			if err != nil || first == nil || first.plan.capture.BodyBytes != wantHeader.body || first.plan.capture.WireBytes != wantHeader.wire {
				t.Fatal("original admission did not retain its selected header", err)
			}
			origin := s.setupOrigin
			connection := providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, string(response))
			initial, err := first.executeCandidateWithConnection(context.Background(), connection)
			if !wide {
				if err == nil || initial != nil || first.diagnostic.Tag != providerFirstFailureBody || first.diagnostic.Reason != providerFirstReasonLimit ||
					first.diagnostic.Detail != (16<<10)+1 || !first.consumption || !first.settlement || s.sequence != 4 || s.setupOrigin != origin {
					t.Fatal("old finite header class did not retain its bounded refusal", err)
				}
				return
			}
			if err != nil || initial == nil || len(initial.result) <= 16<<10 || !first.consumption || !first.settlement {
				t.Fatal("selected header could not retain the complete ordinary response", err)
			}
			firstExchange, execution, remaining := initial.exchange, first.execution.actual, first.remaining.actual
			bootstrap, err := newProviderContinuingBootstrapCandidate(s)
			if err != nil {
				t.Fatal("same-lifetime bootstrap admission", err)
			}
			if _, err := bootstrap.executeBootstrapCandidate(context.Background()); err != nil {
				t.Fatal("same-lifetime bootstrap completion", err)
			}
			if _, err := sealProviderContinuingPageCandidate(s); err != nil {
				t.Fatal("first original page closure", err)
			}
			methods := []string{"eth_chainId", "web3_clientVersion", "eth_chainId"}
			results := []string{`"0xaa36a7"`, `"SAGE fixture"`, `"0xaa36a7"`}
			for i, method := range methods {
				if i == 2 {
					if _, err := sealProviderContinuingPageCandidate(s); err != nil {
						t.Fatal("next original page closure", err)
					}
				}
				operation, err := newProviderContinuingRPCCandidate(s, method, []any{})
				if err != nil {
					t.Fatal("retained scalar admission", method, err)
				}
				body := `{"jsonrpc":"2.0","id":1,"result":` + results[i] + `}`
				connection := providerIntegrationOwnedTLS(t, operation.request.body, body)
				value, err := operation.executeRPCCandidateWithConnection(context.Background(), connection)
				if err != nil || value == nil || string(value.result) != results[i] || operation.capture.BodyBytes != 1024 ||
					operation.state != providerContinuingDone || !operation.consumption || !operation.settlement || !operation.execution.closed || !operation.remaining.closed {
					t.Fatal("retained scalar completion", method, err)
				}
			}
			g := s.guardian
			if s.setupOrigin != origin || g.original != origin || g.integration.firstExchange != firstExchange ||
				s.frontierReplay.body.firstExchange != firstExchange || first.execution.actual != execution || first.remaining.actual != remaining ||
				s.sequence != 18 || g.ordinal != 6 || g.integration.ledger.continuation.page != 2 || !g.ready || g.active || s.activeMeter.Load() != nil {
				t.Fatal("large first header replaced its original lifetime or escaped retention")
			}
		})
	}
}
