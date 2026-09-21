package guard010

import (
	"bytes"
	"encoding/json"
)

// Setup codecs validate plaintext only, after the signed session carrier has
// authenticated it. They never release metadata to a model, create readiness,
// verify outer success/correlation, or enter the Guard execution ledger.
func mcpSetupObject(raw []byte) (map[string]json.RawMessage, error) {
	return rpcObject(raw, 16348, 16348, 36)
}
func mcpNested(raw json.RawMessage) (map[string]json.RawMessage, bool) {
	var m map[string]json.RawMessage
	e := json.Unmarshal(raw, &m)
	return m, e == nil && m != nil
}
func mcpString(raw json.RawMessage) bool {
	var s string
	return len(raw) > 0 && raw[0] == '"' && json.Unmarshal(raw, &s) == nil
}
func mcpInfo(raw json.RawMessage) bool {
	m, ok := mcpNested(raw)
	if !ok || !mcpString(m["name"]) || !mcpString(m["version"]) {
		return false
	}
	if title, exists := m["title"]; exists && !mcpString(title) {
		return false
	}
	return true
}
func mcpMeta(m map[string]json.RawMessage) bool {
	raw, exists := m["_meta"]
	if !exists {
		return true
	}
	_, ok := mcpNested(raw)
	return ok
}
func validateMCPInitialize(raw []byte, id string, response bool) error {
	m, e := mcpSetupObject(raw)
	if e != nil || !uuid.MatchString(id) || rpcText(m, "id") != id || rpcText(m, "jsonrpc") != "2.0" {
		return ErrInvalid
	}
	key, info := "params", "clientInfo"
	if response {
		key, info = "result", "serverInfo"
		if len(m) != 3 {
			return ErrInvalid
		}
	} else if len(m) != 4 || rpcText(m, "method") != "initialize" {
		return ErrInvalid
	}
	body, ok := mcpNested(m[key])
	if !ok || rpcText(body, "protocolVersion") != MCPVersion || !mcpInfo(body[info]) || !mcpMeta(body) {
		return ErrInvalid
	}
	caps, ok := mcpNested(body["capabilities"])
	if !ok {
		return ErrInvalid
	}
	if response {
		tools, ok := mcpNested(caps["tools"])
		if !ok || len(caps) != 1 || len(tools) != 0 {
			return ErrInvalid
		}
		if instructions, exists := body["instructions"]; exists && !mcpString(instructions) {
			return ErrInvalid
		}
	} else {
		if len(caps) != 0 {
			return ErrInvalid
		}
		// Initialize params inherit request metadata; a progress token may be a
		// string or number but does not enable progress notifications in this profile.
		if meta, exists := body["_meta"]; exists {
			mm, _ := mcpNested(meta)
			if token, exists := mm["progressToken"]; exists {
				var n json.Number
				if !mcpString(token) && (bytes.Equal(token, []byte("null")) || json.Unmarshal(token, &n) != nil) {
					return ErrInvalid
				}
			}
		}
	}
	return nil
}
func validateMCPInitialized(raw []byte, ack bool) error {
	if ack {
		if bytes.Equal(raw, []byte("{}")) {
			return nil
		}
		return ErrInvalid
	}
	m, e := mcpSetupObject(raw)
	if e != nil || len(m) != 2 || rpcText(m, "jsonrpc") != "2.0" || rpcText(m, "method") != "notifications/initialized" {
		return ErrInvalid
	}
	return nil
}
func validateMCPDiscovery(raw []byte, id string, response bool) error {
	m, e := mcpSetupObject(raw)
	if e != nil || !uuid.MatchString(id) || rpcText(m, "id") != id || rpcText(m, "jsonrpc") != "2.0" || len(m) != 3 {
		return ErrInvalid
	}
	if !response {
		if rpcText(m, "method") == "tools/list" {
			return nil
		}
		return ErrInvalid
	}
	result, ok := mcpNested(m["result"])
	if !ok || len(result) != 1 {
		return ErrInvalid
	}
	var tools []json.RawMessage
	if json.Unmarshal(result["tools"], &tools) != nil || len(tools) != 1 {
		return ErrInvalid
	}
	got, e := canonicalizeBounds(tools[0], 16348, 16348, 36)
	if e != nil {
		return ErrInvalid
	}
	want, e := canonicalizeBounds(mcpTool, 16348, 16348, 36)
	if e != nil || !bytes.Equal(got, want) {
		return ErrInvalid
	}
	return nil
}
