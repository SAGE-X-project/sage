package guard010

import (
	"bytes"
	"context"
	"encoding/json"
)

// MCPVersion is the explicitly supported structured-result binding. A host must
// authenticate negotiation and reject unsupported versions before protected calls.
const MCPVersion = "2025-06-18"

// CheckMCPVersion checks local support, not the authenticity of negotiation.
func CheckMCPVersion(version string) error {
	if version != MCPVersion {
		return ErrInvalid
	}
	return nil
}

// ParseMCPResult checks representation and status mapping only. Returned bytes
// are UNAUTHENTICATED: use VerifyResult or Client.AcceptMCP before any consumption.
// The wrapper permits up to 8 MiB for JSON string escaping of a 1 MiB envelope;
// the enclosed envelope retains its own 1 MiB, depth 32 and 4096-member bounds.
// Unknown fields (including unsigned annotations) are rejected, never forwarded.
func ParseMCPResult(version string, raw []byte) ([]byte, error) {
	if CheckMCPVersion(version) != nil {
		return nil, ErrInvalid
	}
	if _, e := canonicalizeBounds(raw, 4104, 8*MaxBytes, 34); e != nil {
		return nil, ErrInvalid
	}
	var root map[string]json.RawMessage
	if json.Unmarshal(raw, &root) != nil || len(root) != 3 {
		return nil, ErrInvalid
	}
	var flag bool
	if !bytes.Equal(root["isError"], []byte("true")) && !bytes.Equal(root["isError"], []byte("false")) {
		return nil, ErrInvalid
	}
	if json.Unmarshal(root["isError"], &flag) != nil {
		return nil, ErrInvalid
	}
	var blocks []map[string]json.RawMessage
	if json.Unmarshal(root["content"], &blocks) != nil || len(blocks) != 1 || len(blocks[0]) != 2 {
		return nil, ErrInvalid
	}
	var kind, text string
	if json.Unmarshal(blocks[0]["type"], &kind) != nil || kind != "text" || json.Unmarshal(blocks[0]["text"], &text) != nil {
		return nil, ErrInvalid
	}
	e, canonical, err := object(root["structuredContent"])
	if err != nil || !closed(e, "result proof") || !bytes.Equal(canonical, []byte(text)) {
		return nil, ErrInvalid
	}
	m, ok := e["result"].(map[string]any)
	if !ok {
		return nil, ErrInvalid
	}
	// Check original protocol integers before canonicalization can round them.
	if _, ok = number(m, "created"); !ok {
		return nil, ErrInvalid
	}
	if _, ok = number(m, "expires"); !ok {
		return nil, ErrInvalid
	}
	success, _, err := resultMapping(str(m, "status"))
	if err != nil || flag == success {
		return nil, ErrInvalid
	}
	return canonical, nil
}

func resultMapping(status string) (bool, string, error) {
	switch status {
	case "completed":
		return true, "", nil
	case "pending":
		return false, "unavailable", nil
	case "unknown":
		return false, "operation_failed", nil
	case "rejected":
		return false, "policy_denied", nil
	default:
		return false, "", ErrInvalid
	}
}

// MCPResult formats an authenticated snapshot without signing or extending its
// lifetime. Publishers must still use their one-response permit and protected
// transport; recipients must freshly verify and consume their own invocation.
func (v *VerifiedResult) MCPResult(version string) ([]byte, error) {
	if v == nil || len(v.canonical) == 0 || CheckMCPVersion(version) != nil {
		return nil, ErrInvalid
	}
	// encode canonicalizes owned JSON after safe string encoding, preserving the
	// inner envelope spelling without manually assembling JSON fragments.
	raw := encode(map[string]any{"structuredContent": json.RawMessage(v.canonical), "content": []any{map[string]any{"type": "text", "text": string(v.canonical)}}, "isError": v.status != "completed"})

	if _, e := ParseMCPResult(version, raw); e != nil {
		return nil, ErrInvalid
	}
	return raw, nil
}

// Carriage maps authenticated status to chapter 08 success/error. This snapshot
// is not authorization to poll, execute, or consume a result again.
func (v *VerifiedResult) Carriage() (bool, string, error) {
	if v == nil || len(v.canonical) == 0 {
		return false, "", ErrInvalid
	}
	return resultMapping(v.status)
}

// AcceptMCP consumes the outstanding invocation even on malformed MCP data or
// unsupported version, then uses the durable single-terminal acceptance path.
func (c *Client) AcceptMCP(ctx context.Context, t *ClientInvocation, version string, raw []byte) (*ClientDelivery, error) {
	envelope, e := ParseMCPResult(version, raw)
	if e != nil {
		_ = c.Failed(t)
		return nil, ErrInvalid
	}
	return c.Accept(ctx, t, envelope)
}
