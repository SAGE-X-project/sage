//go:build registry_live

package registry011

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// This opt-in test makes bounded read-only requests to the sole approved RPC.
// It accesses no keys and cannot submit transactions or use another provider.
func TestSepoliaNodeEvidenceLive(t *testing.T) {
	dir := os.Getenv("SAGE_NODE_EVIDENCE_DIR")
	if dir == "" {
		t.Fatal("explicit evidence directory required")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	report := map[string]any{"scope": "thirteen runtime identities at one provider-finalized anchor and one existing Registry creation; no complete history, mutation lifecycle or Source qualification", "rpc": SepoliaRPCURL, "started_utc": time.Now().UTC().Format(time.RFC3339Nano), "paid_transactions": 0, "source_ready": false}
	defer func() {
		report["finished_utc"] = time.Now().UTC().Format(time.RFC3339Nano)
		b, e := json.MarshalIndent(report, "", "  ")
		if e != nil {
			t.Error(e)
			return
		}
		if e = os.WriteFile(filepath.Join(dir, "live-result.json"), append(b, '\n'), 0644); e != nil {
			t.Error(e)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	c, err := NewSepoliaReadRPC(RPCTransportLimits{20 * time.Second, 4 << 20, 128})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	r, err := c.PinAgentReads(ctx, SepoliaRegistryAddress, "sage-deployment-check")
	if err != nil {
		report["error"] = err.Error()
		t.Fatal("pin", err)
	}
	limits := nodeTestLimits()
	limits.Replay.MaxTransactions = 1024
	limits.Replay.MaxFrames = 256
	limits.Replay.MaxLogs = 4096
	limits.Replay.MaxEvidenceBytes = 8 << 20
	got, err := r.ObserveDeployment(ctx, limits)
	if err != nil {
		report["error"] = err.Error()
		t.Fatal("deployment evidence", err)
	}
	b, err := json.MarshalIndent(got, "", "  ")
	if err != nil || len(b) > 32<<20 {
		t.Fatal("retained bundle bound", err)
	}
	if err = os.WriteFile(filepath.Join(dir, "deployment-evidence.json"), append(b, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	report["anchor"] = got.Anchor
	report["deployment"] = got.Deployment
	report["client_version_claim"] = got.ClientVersion
	report["runtime_code_identities"] = len(got.CodeHashes)
	report["requests_in_observation"] = len(got.Exchanges)
	report["bundle_bytes"] = len(b)
	report["result"] = "PASS within declared scope"
}
