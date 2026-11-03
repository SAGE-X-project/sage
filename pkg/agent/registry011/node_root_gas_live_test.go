//go:build registry_live

package registry011

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// This is an independent qualification trial of the EXISTING explicitly
// selected top-only callTracer adapter. It never reads the strict trial's
// result, retries it, replaces its failure or activates an operating policy.
func TestSepoliaRootGasCandidateEvidenceLive(t *testing.T) {
	parent := os.Getenv("SAGE_NODE_EVIDENCE_DIR")
	if parent == "" {
		t.Fatal("explicit evidence directory required")
	}
	dir := filepath.Join(parent, "root-gas-candidate")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	report := map[string]any{
		"scope": "one existing deployment block through the explicitly selected root-call gas candidate; no complete history, independent consensus, Source qualification or policy activation",
		"rpc":   SepoliaRPCURL, "started_utc": time.Now().UTC().Format(time.RFC3339Nano),
		"paid_transactions": 0, "source_ready": false, "policy_active": false,
		"gas_acquisition": "callTracer block roots, onlyTopCall=true, withLog=false, includePrecompiles=true; selected before acquisition",
		"strict_fallback": false, "candidate_status": "NOT_CHECKED",
	}
	defer func() {
		report["finished_utc"] = time.Now().UTC().Format(time.RFC3339Nano)
		raw, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			t.Error(err)
			return
		}
		if err := os.WriteFile(filepath.Join(dir, "live-result.json"), append(raw, '\n'), 0644); err != nil {
			t.Error(err)
		}
	}()
	identity, _, err := nodeIdentityManifest(nodeIdentityJSON, nodeABIJSON)
	if err != nil {
		t.Fatal("original deployment identity required", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	client, err := NewSepoliaReadRPC(RPCTransportLimits{20 * time.Second, 4 << 20, 128})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	transport, ok := client.client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("owned transport required for passive diagnostics")
	}
	client.client.Transport = &liveCaptureTransport{base: transport, dir: dir, t: t, remaining: 32 << 20}
	reads, err := client.PinAgentReads(ctx, SepoliaRegistryAddress, "sage-deployment-check")
	if err != nil {
		report["candidate_error"] = err.Error()
		t.Fatal("candidate pin", err)
	}
	limits := nodeTestLimits()
	limits.BlockEncoding = EthereumJSONBlocks
	limits.Replay.MaxTransactions = 1024
	limits.Replay.MaxFrames = 256
	limits.Replay.MaxLogs = 4096
	limits.Replay.MaxEvidenceBytes = 8 << 20
	report["block_encoding"] = EthereumJSONBlocks
	report["consensus_bytes_basis"] = "derived canonical RLP from retained standard JSON, not original RLP response bytes"
	// This fixed entry never calls ReadReplayBlock or changes the strict
	// stateGasTracer trial. Both test roots run regardless of the other's result.
	block, err := reads.ReadReplayBlockWithRootGas(ctx, sepoliaDeploymentBlock, limits)
	if err != nil {
		report["candidate_error"] = err.Error()
		t.Fatal("explicit root-gas candidate evidence", err)
	}
	if block == nil || block.Header.Number != sepoliaDeploymentBlock || block.Header.Hash != identity.Deployment.BlockHash ||
		block.Block.Gas == nil || block.Block.Gas.BlockHash != block.Header.Hash ||
		(block.Block.Gas.Profile != SepoliaAmsterdamRootRegularGasProfile && block.Block.Gas.Profile != SepoliaAmsterdamRootExecutionGasProfile) {
		t.Fatal("same deployment and explicit candidate profile required")
	}
	raw, err := json.MarshalIndent(block, "", "  ")
	if err != nil || len(raw) > 32<<20 {
		t.Fatal("retained candidate bundle bound", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "candidate-replay-block.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	pins := make([]map[string]any, 0, len(block.Exchanges))
	for index, exchange := range block.Exchanges {
		name := fmt.Sprintf("candidate-response-%03d.json", index)
		if err := os.WriteFile(filepath.Join(dir, name), exchange.Response, 0644); err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(exchange.Response)
		pins = append(pins, map[string]any{"file": name, "method": exchange.Method, "params": exchange.Params, "bytes": len(exchange.Response),
			"sha256": hex.EncodeToString(digest[:]), "rpc": SepoliaRPCURL, "accepted_http_status": 200})
	}
	if t.Failed() {
		t.Fatal("complete candidate evidence retention required")
	}
	report["anchor"] = reads.header
	report["deployment"] = block.Header
	report["client_version_claim"] = block.ClientVersion
	report["gas"] = block.Gas
	report["gas_profile"] = block.Block.Gas.Profile
	report["requests"] = len(block.Exchanges)
	report["bundle_bytes"] = len(raw)
	report["response_files"] = pins
	report["response_status_basis"] = "ReadRPC admits HTTP200 application/json; passive diagnostics are not accepted raw HTTP-header proof"
	report["derived_gas_witness_basis"] = "Gas.Trace is the candidate adapter's derived transaction-bound witness; original responses are retained separately"
	report["candidate_status"] = "CHECKED: trusted-provider gas observations joined to committed block and receipts; independent qualification trial only"
}
