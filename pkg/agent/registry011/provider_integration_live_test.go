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

// Five normal read-only requests to the one selected official TLS origin.
// These isolated test-owned stores/policy remain explicitly unqualified and
// never authorize a protected message, real agent registration or transaction.
func TestProviderCaptureConnectedAdmissionAndRegistryPin(t *testing.T) {
	dir := os.Getenv("SAGE_PROVIDER_CAPTURE_DIR")
	if dir == "" {
		t.Fatal("explicit evidence directory required")
	}
	dir = filepath.Join(dir, "connected")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	// The existing official capture uses the same20-second request ceiling.
	// This separate subject chooses smaller byte caps before opening/enrolling;
	// no live profile, original I, request deadline or failed attempt is renewed.
	store, policy, candidate, primitive := providerStoreFixtureWithCapture(t, ProviderCaptureLimits{Timeout: 20 * time.Second, HeaderBytes: 8 << 10, BodyBytes: 64 << 10, WireBytes: 128 << 10, JSONDepth: 64})
	defer providerRetainConnectedFixture(t, dir, store)
	c, err := newProviderIntegration(store, policy, candidate, primitive)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	client, err := c.readClient()
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.call(ctx, "eth_getBlockByNumber", []any{"finalized", false}); err != nil {
		t.Fatal("admitted first finalized read", err)
	}
	if err := c.installBootstrap(); err != nil {
		t.Fatal("original retained start", err)
	}
	reads, err := c.pinAgent(ctx, "sage-owned-registry-read")
	if err != nil {
		t.Fatal("closed official Registry pin", err)
	}
	defer reads.rpc.Close()
	if c.pin == "" || c.bootstrap == "" || c.selected != reads.header || store.replay.start != c.bootstrap || c.ledger.current.counts != [3]uint64{7, 7, 7} || store.sequence != 22 || c.root.status != "enrolling" || c.root.checkpoint != "" {
		t.Fatal("pin skipped admission or manufactured readiness")
	}
	diagnostic, err := c.ledger.current.Diagnostic()
	if err != nil {
		t.Fatal(err)
	}
	report := map[string]any{"scope": "isolated candidate admission/first-finalized/bootstrap/closed Registry pin; no complete history/store qualification or protected-message authority", "endpoint": SepoliaRPCURL, "registry": SepoliaRegistryAddress, "chain_id": "11155111", "rpc_exchanges": 5, "local_commands": 2, "source_ready": false, "policy_activated": false, "paid_transactions": 0, "bootstrap": c.bootstrap, "pin": c.pin, "selected": c.selected, "ledger": diagnostic, "custody_sequence": store.sequence, "custody_commit": store.tip}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "report.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}

func providerRetainConnectedFixture(t *testing.T, directory string, store *providerCustodyStore) {
	t.Helper()
	// These are bounded, ordinary files belonging only to this isolated test.
	// Original metadata/request/response bytes are copied, never rewritten.
	for name, source := range map[string]string{"custody": store.config.custodyDirectory, "evidence": store.config.evidenceDirectory} {
		target := filepath.Join(directory, name)
		if err := os.MkdirAll(target, 0755); err != nil {
			t.Error(err)
			return
		}
		entries, err := os.ReadDir(source)
		if err != nil || len(entries) > store.config.bounds.maximumObjects+3*store.config.bounds.maximumCommits+32 {
			t.Error("bounded owned evidence listing", err)
			return
		}
		for _, entry := range entries {
			if !entry.Type().IsRegular() {
				t.Error("nonregular owned evidence")
				return
			}
			raw, err := os.ReadFile(filepath.Join(source, entry.Name()))
			if err != nil || uint64(len(raw)) > store.config.bounds.maximumObjectBytes {
				t.Error("bounded owned evidence file", err)
				return
			}
			if err := os.WriteFile(filepath.Join(target, entry.Name()), raw, 0644); err != nil {
				t.Error(err)
				return
			}
		}
	}
}
