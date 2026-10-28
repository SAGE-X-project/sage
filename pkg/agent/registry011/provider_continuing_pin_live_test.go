//go:build provider_capture_live

package registry011

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Fresh isolated test inputs, ordinary owned retention and the one official
// RPC only. This does not select operating custody, a policy or an Agent key.
func TestProviderCaptureContinuingPinRetainsSameBlockRegistryIdentity(t *testing.T) {
	t.Parallel()
	dir := os.Getenv("SAGE_PROVIDER_CAPTURE_DIR")
	if dir == "" {
		t.Fatal("explicit evidence directory required")
	}
	dir = filepath.Join(dir, "connected-pin")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	_, capture := providerFrontierBoundsFixture()
	capture.Timeout = 20 * time.Second // Same existing official-read ceiling, chosen before H0.
	// A separate fresh TLS test subject admits the two verified paths seen
	// in the original official-origin sample. It neither changes the old 1/4
	// refusal subject nor treats a prior sample as this connection's identity.
	tls := providerTLSBounds{2, 5, 8 << 10}
	config, clock := providerGuardianConfigWithSelectionFixture(t, 64<<20, 6, &providerScanFixtureSelection{capture: &capture, tls: &tls, pin: true})
	s, err := openProviderFrontierCandidate(config, clock)
	if err != nil {
		t.Fatal("fresh isolated retained pipeline", err)
	}
	t.Cleanup(func() { _ = s.close() })
	defer providerRetainConnectedFixture(t, dir, s)
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
		t.Fatal("full initial original scan", err)
	}
	first, err := newProviderFirstOperationCandidate(s)
	if err != nil {
		t.Fatal("original first request", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	initial, err := first.executeCandidate(ctx)
	if err != nil || initial == nil {
		t.Fatal("official retained first finalized response", err)
	}
	bootstrap, err := newProviderContinuingBootstrapCandidate(s)
	if err != nil {
		t.Fatal("next retained bootstrap admission", err)
	}
	start, err := bootstrap.executeBootstrapCandidate(ctx)
	if err != nil || start == nil {
		t.Fatal("retained bootstrap", err)
	}
	if _, err := sealProviderContinuingPageCandidate(s); err != nil {
		t.Fatal("first original page closure", err)
	}
	g := s.guardian
	origin, firstExchange := s.setupOrigin, g.integration.firstExchange
	requests, err := providerPinRequests(start.selected.Hash)
	if err != nil {
		t.Fatal(err)
	}
	var digests [4]string
	for i, request := range requests {
		if i == 2 {
			if _, err := sealProviderContinuingPageCandidate(s); err != nil {
				t.Fatal("next original page closure", err)
			}
		}
		var params any
		if json.Unmarshal([]byte(request.params), &params) != nil {
			t.Fatal("fixed read params")
		}
		o, err := newProviderContinuingRPCCandidate(s, request.method, params)
		if err != nil {
			t.Fatalf("official child %d admission: %v G=%+v", i, err, g.meter)
		}
		value, err := o.executeRPCCandidate(ctx)
		if err != nil || value == nil {
			t.Fatalf("official child %d: %v E=%+v R=%+v", i, err, o.execution, o.remaining)
		}
		digests[i] = value.exchange
	}
	if _, err := sealProviderContinuingPageCandidate(s); err != nil {
		t.Fatal("page before local pin", err)
	}
	pin, err := newProviderContinuingPinCandidate(s, "example-agent", digests)
	if err != nil {
		t.Fatalf("official pin admission: %v G=%+v", err, g.meter)
	}
	value, err := pin.executePinCandidate(ctx)
	if err != nil || value == nil {
		t.Fatalf("official complete original pin: %v E=%+v R=%+v", err, pin.execution, pin.remaining)
	}
	if s.setupOrigin != origin || g.original != origin || g.integration.firstExchange != firstExchange || value.selected != start.selected.Hash || value.exchanges != digests || g.integration.pin != value.pin || s.sequence != 25 || g.ordinal != 9 || !g.ready || g.active || s.activeMeter.Load() != nil {
		t.Fatal("identity pin reset lifetime or escaped original settlement")
	}
	diagnostic, err := g.integration.ledger.current.Diagnostic()
	if err != nil {
		t.Fatal(err)
	}
	report := map[string]any{"scope": "isolated V5 fixed-origin same-block Registry identity pin; no registered Agent record, complete history, final original inspection or qualified operating Source", "endpoint": SepoliaRPCURL, "registry": SepoliaRegistryAddress, "chain_id": "11155111", "rpc_exchanges": 5, "local_commands": 2, "page_closures": 3, "source_ready": false, "policy_activated": false, "paid_transactions": 0, "initial_exchange": firstExchange, "pin_exchanges": digests, "selected": start.selected, "pin": value.pin, "ledger": diagnostic, "custody_sequence": s.sequence, "custody_commit": s.tip}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "report.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
