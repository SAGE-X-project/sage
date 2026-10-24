//go:build registry_live

package registry011

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	report := map[string]any{"scope": "thirteen runtime identities, one existing Registry creation, and provider-observed gas consistency for its complete block; no independent EVM/consensus validation, complete history, mutation lifecycle or Source qualification", "rpc": SepoliaRPCURL, "started_utc": time.Now().UTC().Format(time.RFC3339Nano), "paid_transactions": 0, "source_ready": false}
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
	report["gas"] = got.Gas
	report["gas_accounting_status"] = "UNVALIDATED: legacy equality is not fork or consensus qualification"
	if !got.Gas.LegacyGasCompatible {
		report["gas_accounting_status"] = "UNSUPPORTED for strict command replay"
	}
	report["result"] = "OBSERVED within declared content scope"
	strict, err := r.ReadReplayBlock(ctx, sepoliaDeploymentBlock, limits)
	if err != nil {
		report["strict_gas_error"] = err.Error()
		t.Fatal("strict block gas evidence", err)
	}
	if strict.Header != got.Deployment || strict.Block.Gas == nil {
		t.Fatal("same selected block and complete gas witness required")
	}
	b, err = json.MarshalIndent(strict, "", "  ")
	if err != nil || len(b) > 32<<20 {
		t.Fatal("strict bundle bound", err)
	}
	if err = os.WriteFile(filepath.Join(dir, "strict-replay-block.json"), append(b, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	report["strict_gas_profile"] = strict.Block.Gas.Profile
	report["strict_gas_status"] = "CHECKED: trusted-provider observations joined to committed contents; not independent consensus validation"
	report["strict_requests"] = len(strict.Exchanges)
	// A prettified bundle cannot preserve JSON whitespace. Retain every response
	// independently, byte for byte, before publishing its digest in the report.
	for _, group := range []struct {
		name      string
		exchanges []NodeExchange
	}{{"deployment", got.Exchanges}, {"strict", strict.Exchanges}} {
		pins := make([]map[string]any, 0, len(group.exchanges))
		for i, exchange := range group.exchanges {
			name := fmt.Sprintf("%s-response-%03d.json", group.name, i)
			if err := os.WriteFile(filepath.Join(dir, name), exchange.Response, 0644); err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(exchange.Response)
			pins = append(pins, map[string]any{"file": name, "method": exchange.Method, "params": exchange.Params, "bytes": len(exchange.Response), "sha256": hex.EncodeToString(digest[:]), "rpc": SepoliaRPCURL, "accepted_http_status": 200})
		}
		report[group.name+"_response_files"] = pins
	}
	report["response_status_basis"] = "ReadRPC admits only HTTP 200 application/json responses; no raw HTTP-header capture"
	report["request_evidence_scope"] = "fixed methods and serialized params; not a capture of raw HTTP or full JSON-RPC request bytes"
}
