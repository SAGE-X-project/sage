package guard010

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"github.com/sage-x-project/sage/pkg/agent/crypto/jcs"
	"os"
	"strings"
	"testing"
)

// These are codec-only snapshots, not authentication observations. Large content
// intentionally changes the frozen proof's body; no dispatch or verification is claimed.
func TestMCPWirePreservesLargeCanonicalFragments(t *testing.T) {
	raw, e := os.ReadFile("testdata/guard-records.json")
	if e != nil {
		t.Fatal(e)
	}
	var suite struct {
		Cases []struct {
			ID    string
			Input map[string]json.RawMessage
		}
	}
	if json.Unmarshal(raw, &suite) != nil {
		t.Fatal("fixture")
	}
	envelope := func(id string) map[string]any {
		for _, c := range suite.Cases {
			if c.ID == id {
				var h string
				_ = json.Unmarshal(c.Input["envelope_hex"], &h)
				b, _ := hex.DecodeString(h)
				var v map[string]any
				_ = json.Unmarshal(b, &v)
				return v
			}
		}
		t.Fatal("fixture missing")
		return nil
	}
	canonical := func(v any) []byte {
		b, _ := json.Marshal(v)
		b, e = jcs.Canonicalize(b)
		if e != nil || len(b) > MaxBytes {
			t.Fatal("fixture bounds", e)
		}
		return b
	}
	for _, value := range []string{strings.Repeat("<>&", 75000), strings.Repeat("\u2028\u2029", 100000)} {
		intent := envelope("intent-valid")
		intent["intent"].(map[string]any)["arguments"] = map[string]any{"path": value}
		inner := canonical(intent)
		id := "00000000-0000-4000-8000-000000000101"
		request, e := MCPRequest(MCPVersion, id, inner)
		if e != nil {
			t.Fatal(e)
		}
		parsed, e := ParseMCPRequest(MCPVersion, id, request)
		if e != nil || !bytes.Equal(inner, parsed) {
			t.Fatal("request expansion", e)
		}
		result := envelope("result-completed-valid")
		result["result"].(map[string]any)["output"] = map[string]any{"text": value}
		inner = canonical(result)
		snapshot := &VerifiedResult{canonical: inner, status: "completed"}
		body, e := snapshot.MCPResult(MCPVersion)
		if e != nil {
			t.Fatal("result expansion", e)
		}
		response := mcpRPCResponse(id, body)
		parsed, e = parseMCPResponse(MCPVersion, id, response)
		if e != nil || !bytes.Equal(inner, parsed) {
			t.Fatal("response expansion", e)
		}
	}
}
