package guard010

import (
	"encoding/json"
	"strings"
	"testing"
)

func setupInitialize(response bool) []byte {
	body := map[string]any{"protocolVersion": MCPVersion, "capabilities": map[string]any{}, "clientInfo": map[string]any{"name": "test", "version": "1", "title": "display only"}}
	msg := map[string]any{"jsonrpc": "2.0", "id": ownerID1, "method": "initialize", "params": body}
	if response {
		delete(body, "clientInfo")
		body["serverInfo"] = map[string]any{"name": "test", "version": "1"}
		body["capabilities"] = map[string]any{"tools": map[string]any{}}
		body["instructions"] = "untrusted descriptive text"
		msg = map[string]any{"jsonrpc": "2.0", "id": ownerID1, "result": body}
	}
	return encode(msg)
}
func TestMCPSetupInitializeValidation(t *testing.T) {
	for _, response := range []bool{false, true} {
		raw := setupInitialize(response)
		if e := validateMCPInitialize(raw, ownerID1, response); e != nil {
			t.Fatal(e)
		}
		for _, bad := range [][]byte{
			[]byte(strings.Replace(string(raw), MCPVersion, "unsupported", 1)),
			[]byte(strings.Replace(string(raw), `"version":"1"`, `"version":null`, 1)),
			[]byte(strings.Replace(string(raw), `"capabilities":`, `"capabilities":null,"capabilities":`, 1)),
			append(append([]byte(nil), raw...), []byte(" {}")...),
			[]byte(`[]`), []byte(`null`), append(raw, make([]byte, 16349)...),
			[]byte(strings.Replace(string(raw), `"jsonrpc":"2.0"`, `"jsonrpc":"1.0"`, 1)),
		} {
			if e := validateMCPInitialize(bad, ownerID1, response); e == nil {
				t.Fatal("invalid setup accepted")
			}
		}
		if e := validateMCPInitialize(raw, ownerID2, response); e == nil {
			t.Fatal("wrong ID")
		}
		var m map[string]any
		_ = json.Unmarshal(raw, &m)
		key := "params"
		if response {
			key = "result"
		}
		b := m[key].(map[string]any)
		b["capabilities"] = map[string]any{"tools": map[string]any{}, "sampling": map[string]any{}}
		if e := validateMCPInitialize(encode(m), ownerID1, response); e == nil {
			t.Fatal("extra capability")
		}
	}
}
func TestMCPSetupMetadataIsTypedButNeverReturned(t *testing.T) {
	var m map[string]any
	_ = json.Unmarshal(setupInitialize(false), &m)
	body := m["params"].(map[string]any)
	for _, v := range []any{"token", float64(42)} {
		body["_meta"] = map[string]any{"progressToken": v}
		if e := validateMCPInitialize(encode(m), ownerID1, false); e != nil {
			t.Fatal(e)
		}
	}
	for _, v := range []any{nil, true, []any{}, map[string]any{}} {
		body["_meta"] = map[string]any{"progressToken": v}
		if e := validateMCPInitialize(encode(m), ownerID1, false); e == nil {
			t.Fatal("invalid token")
		}
	}
}
func TestMCPSetupNotificationAndExactAcknowledgement(t *testing.T) {
	if e := validateMCPInitialized([]byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`), false); e != nil {
		t.Fatal(e)
	}
	for _, raw := range []string{`{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}`, `{"jsonrpc":"2.0","method":"notifications/initialized","id":"x"}`, `{"jsonrpc":"2.0","method":"ping"}`} {
		if e := validateMCPInitialized([]byte(raw), false); e == nil {
			t.Fatal("invalid notification")
		}
	}
	if e := validateMCPInitialized([]byte("{}"), true); e != nil {
		t.Fatal(e)
	}
	for _, raw := range []string{" {}", "{ }", "{}\n", "null", `{"result":{}}`} {
		if e := validateMCPInitialized([]byte(raw), true); e == nil {
			t.Fatal("nonexact ack")
		}
	}
}
func TestMCPSetupPinnedDiscovery(t *testing.T) {
	req := encode(map[string]any{"jsonrpc": "2.0", "id": ownerID2, "method": "tools/list"})
	if e := validateMCPDiscovery(req, ownerID2, false); e != nil {
		t.Fatal(e)
	}
	reply := func(tool any, extra bool) []byte {
		result := map[string]any{"tools": []any{tool}}
		if extra {
			result["nextCursor"] = "more"
		}
		return encode(map[string]any{"jsonrpc": "2.0", "id": ownerID2, "result": result})
	}
	if e := validateMCPDiscovery(reply(json.RawMessage(mcpTool), false), ownerID2, true); e != nil {
		t.Fatal(e)
	}
	var tool map[string]any
	_ = json.Unmarshal(mcpTool, &tool)
	tool["description"] = "extra"
	for _, raw := range [][]byte{reply(tool, false), reply(json.RawMessage(mcpTool), true), []byte(strings.Replace(string(req), `"method":"tools/list"`, `"method":"tools/list","params":{}`, 1))} {
		response := strings.Contains(string(raw), `"result"`)
		if e := validateMCPDiscovery(raw, ownerID2, response); e == nil {
			t.Fatal("changed descriptor/discovery shape")
		}
	}
}
