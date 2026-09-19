package guard010_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
	"os"
	"testing"
)

func TestMCPIndependentVerification(t *testing.T) {
	raw, e := os.ReadFile("testdata/guard-mcp.json")
	if e != nil {
		t.Fatal(e)
	}
	var suite struct {
		Cases []struct {
			ID     string
			Input  guardFixture
			Accept bool
		}
	}
	if json.Unmarshal(raw, &suite) != nil || len(suite.Cases) != 32 {
		t.Fatal("fixture")
	}
	for _, c := range suite.Cases {
		t.Run(c.ID, func(t *testing.T) {
			f := c.Input
			wire, _ := hex.DecodeString(f.s("wire_hex"))
			raw, e := g.ParseMCPResult(f.s("mcp_version"), wire)
			var v *g.VerifiedResult
			if e == nil {
				v, e = g.VerifyResult(context.Background(), raw, f, f)
			}
			if (e == nil) != c.Accept {
				t.Fatalf("accept=%v error=%v", c.Accept, e)
			}
			if e != nil {
				return
			}
			success, code, e := v.Carriage()
			if e != nil || success != (v.Status() == "completed") || code != map[string]string{"completed": "", "pending": "unavailable", "unknown": "operation_failed", "rejected": "policy_denied"}[v.Status()] {
				t.Fatal("carriage")
			}
			encoded, e := v.MCPResult(g.MCPVersion)
			if e != nil {
				t.Fatal(e)
			}
			decoded, e := g.ParseMCPResult(g.MCPVersion, encoded)
			if e != nil || !bytes.Equal(decoded, v.Canonical()) {
				t.Fatal("roundtrip")
			}
		})
	}
	var zero g.VerifiedResult
	if _, e := zero.MCPResult(g.MCPVersion); e == nil {
		t.Fatal("zero")
	}
}
func TestMCPClientFailureConsumesInvocation(t *testing.T) {
	c, _, _, s := clientSetup(t)
	ctx := context.Background()
	ticket, e := c.Begin(ctx, outer(1))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.AcceptMCP(ctx, ticket, g.MCPVersion, []byte(`{"isError":true}`)); e == nil {
		t.Fatal("unsigned error accepted")
	}
	if _, e = c.Accept(ctx, ticket, clientRaw(s, "completed")); e == nil {
		t.Fatal("failed invocation reused")
	}
}
func TestMCPClientVerifiedDelivery(t *testing.T) {
	c, _, _, s := clientSetup(t)
	ctx := context.Background()
	ticket, e := c.Begin(ctx, outer(1))
	if e != nil {
		t.Fatal(e)
	}
	raw := clientRaw(s, "completed")
	wire, _ := json.Marshal(map[string]any{"structuredContent": json.RawMessage(raw), "content": []any{map[string]any{"type": "text", "text": string(raw)}}, "isError": false})
	d, e := c.AcceptMCP(ctx, ticket, g.MCPVersion, wire)
	if e != nil || !d.FirstTerminal() || d.Status() != "completed" {
		t.Fatal("delivery", e)
	}
	if _, e = c.AcceptMCP(ctx, ticket, g.MCPVersion, wire); e == nil {
		t.Fatal("duplicate accepted")
	}
}

func TestMCPEnvelopeSizeBoundary(t *testing.T) {
	base := map[string]any{"result": map[string]any{"status": "completed", "created": 1700000000, "expires": 1700000300, "output": map[string]any{"text": ""}}, "proof": "test-only"}
	b, _ := json.Marshal(base)
	canonical, e := g.Canonicalize(b)
	if e != nil {
		t.Fatal(e)
	}
	body := base["result"].(map[string]any)
	output := body["output"].(map[string]any)
	for _, extra := range []int{0, 1} {
		output["text"] = string(bytes.Repeat([]byte("a"), g.MaxBytes-len(canonical)+extra))
		b, _ = json.Marshal(base)
		// ASCII sorted source canonicalization above is only needed at the valid limit.
		var text []byte
		if extra == 0 {
			text, e = g.Canonicalize(b)
			if e != nil {
				t.Fatal(e)
			}
		} else {
			text = b
		}
		wire, _ := json.Marshal(map[string]any{"structuredContent": json.RawMessage(b), "content": []any{map[string]any{"type": "text", "text": string(text)}}, "isError": false})
		_, e = g.ParseMCPResult(g.MCPVersion, wire)
		if (e == nil) != (extra == 0) {
			t.Fatalf("size %d: %v", extra, e)
		}
	}
}
