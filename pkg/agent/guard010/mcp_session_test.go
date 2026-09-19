package guard010

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func sessionFixture(t *testing.T) (string, []byte, []byte, []byte, string, string) {
	t.Helper()
	b, e := os.ReadFile("testdata/guard-rpc.json")
	if e != nil {
		t.Fatal(e)
	}
	var v struct {
		ID                  string
		Input               map[string]any
		Requests, Responses []struct {
			ID   string
			Wire string `json:"wire_hex"`
		}
	}
	if json.Unmarshal(b, &v) != nil {
		t.Fatal("fixture")
	}
	var req, resp []byte
	for _, q := range v.Requests {
		if q.ID == "valid" {
			req, _ = hex.DecodeString(q.Wire)
		}
	}
	for _, q := range v.Responses {
		if q.ID == "valid" {
			resp, _ = hex.DecodeString(q.Wire)
		}
	}
	intent, _ := hex.DecodeString(v.Input["envelope_hex"].(string))
	_, m, _, e := intentEnvelope(intent)
	if e != nil || len(req) == 0 || len(resp) == 0 {
		t.Fatal("fixture")
	}
	return v.ID, req, resp, intent, str(m, "issuer"), str(m, "recipient")
}
func TestMCPSessionIdentityAndExactBinding(t *testing.T) {
	id, req, resp, intent, issuer, recipient := sessionFixture(t)
	if _, e := sessionRequest(MCPVersion, id, req, issuer, recipient); e != nil {
		t.Fatal(e)
	}
	if _, _, e := sessionResult(MCPVersion, id, resp, intent, recipient, issuer); e != nil {
		t.Fatal(e)
	}
	for _, q := range []struct{ version, id, issuer, recipient string }{{"unsupported", id, issuer, recipient}, {MCPVersion, "wrong", issuer, recipient}, {MCPVersion, id, recipient, issuer}, {MCPVersion, id, issuer, issuer}} {
		if _, e := sessionRequest(q.version, q.id, req, q.issuer, q.recipient); e == nil {
			t.Fatal("unbound request")
		}
	}
	for _, q := range []struct{ id, issuer, recipient string }{{"wrong", recipient, issuer}, {id, issuer, recipient}, {id, recipient, recipient}} {
		if _, _, e := sessionResult(MCPVersion, q.id, resp, intent, q.issuer, q.recipient); e == nil {
			t.Fatal("unbound response")
		}
	}
	changed := strings.Replace(string(intent), "public.txt", "other.txt", 1)
	if _, _, e := sessionResult(MCPVersion, id, resp, []byte(changed), recipient, issuer); e == nil {
		t.Fatal("wrong intent")
	}
	if _, e := sessionRequest(MCPVersion, id, append(req, make([]byte, 16349)...), issuer, recipient); e == nil {
		t.Fatal("size")
	}
}
func TestMCPSessionEmptyAndSharedFailurePermit(t *testing.T) {
	var empty MCPSessionCall
	if empty.ID() != "" || empty.Request() != nil {
		t.Fatal("empty")
	}
	if _, e := empty.OpenReply(context.Background(), nil); e == nil {
		t.Fatal("empty permit")
	}
	if _, _, e := SealMCPSessionRequest(context.Background(), nil, MCPVersion, "", nil, 1); e == nil {
		t.Fatal("no session")
	}
	c := MCPSessionCall{&mcpSessionCall{inbound: true}}
	copy := c
	if _, e := c.SealReply(context.Background(), []byte("{}"), 1); e == nil || !copy.state.used {
		t.Fatal("failed reply did not consume shared permit")
	}
	if _, e := copy.SealReply(context.Background(), []byte("{}"), 1); e == nil {
		t.Fatal("reused permit")
	}
}
