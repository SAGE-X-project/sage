package guard010

import (
	"context"
	"encoding/json"
	"github.com/sage-x-project/sage/pkg/agent/hpke"
	"sync"
)

// MCPSessionCall binds one RPC invocation to one authenticated record request.
// Copies share their one-response permit. The host must exclusively route protected
// MCP traffic through this binding and supply authenticated version negotiation.
// This non-HTTP carriage does not claim chapter 08 HTTP payload mapping or host isolation.
type MCPSessionCall struct{ state *mcpSessionCall }
type mcpSessionCall struct {
	mu                                  sync.Mutex
	session                             *hpke.AuthenticatedCompletion010
	version, id, messageID, local, peer string
	request, intent                     []byte
	inbound, used                       bool
}

func (c *MCPSessionCall) ID() string {
	if c == nil || c.state == nil {
		return ""
	}
	return c.state.id
}

// Request returns a copy of exact authenticated RPC bytes for endpoint dispatch.
func (c *MCPSessionCall) Request() []byte {
	if c == nil || c.state == nil || !c.state.inbound {
		return nil
	}
	return append([]byte(nil), c.state.request...)
}
func sessionRequest(version, id string, raw []byte, issuer, recipient string) ([]byte, error) {
	if len(raw) > 16348 {
		return nil, ErrInvalid
	}
	intent, e := ParseMCPRequest(version, id, raw)
	if e != nil {
		return nil, ErrInvalid
	}
	_, m, _, e := intentEnvelope(intent)
	if e != nil || str(m, "issuer") != issuer || str(m, "recipient") != recipient {
		return nil, ErrInvalid
	}
	return intent, nil
}
func sessionResult(version, id string, raw, intent []byte, issuer, recipient string) (bool, string, error) {
	if len(raw) > 16348 {
		return false, "", ErrInvalid
	}
	result, e := parseMCPResponse(version, id, raw)
	if e != nil {
		return false, "", ErrInvalid
	}
	envelope, _, e := object(result)
	if e != nil {
		return false, "", ErrInvalid
	}
	m, ok := envelope["result"].(map[string]any)
	if !ok {
		return false, "", ErrInvalid
	}
	_, request, _, e := intentEnvelope(intent)
	if e != nil || str(m, "issuer") != issuer || str(m, "recipient") != recipient || str(m, "intent_digest") != hash(intent) || str(m, "request_id") != str(request, "request_id") || str(m, "call_id") != str(request, "call_id") {
		return false, "", ErrInvalid
	}
	switch str(m, "status") {
	case "completed":
		return true, "", nil
	case "pending":
		return false, "unavailable", nil
	case "rejected":
		return false, "policy_denied", nil
	case "unknown":
		return false, "operation_failed", nil
	}
	return false, "", ErrInvalid
}
func wireField(raw []byte, key string) string {
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return ""
	}
	return rpcText(m, key)
}

// SealMCPSessionRequest protects exact RPC bytes with the existing signed AEAD
// record protocol. Call only from the client's authorized protected handoff.
// The session limit (16348 plaintext bytes) applies; no fragmentation is introduced.
func SealMCPSessionRequest(ctx context.Context, s *hpke.AuthenticatedCompletion010, version, id string, raw []byte, ttl int64) ([]byte, *MCPSessionCall, error) {
	if ctx == nil || ctx.Err() != nil || s == nil {
		return nil, nil, ErrInvalid
	}
	local, peer, e := s.Participants()
	if e != nil {
		return nil, nil, ErrInvalid
	}
	if len(raw) > 16348 {
		return nil, nil, ErrInvalid
	}
	// Validate and encrypt the same owned snapshot of caller bytes.
	raw = append([]byte(nil), raw...)
	intent, e := sessionRequest(version, id, raw, local, peer)
	if e != nil {
		return nil, nil, ErrInvalid
	}
	wire, e := s.SealRequest(ctx, raw, ttl)
	if e != nil {
		return nil, nil, ErrInvalid
	}
	return wire, &MCPSessionCall{&mcpSessionCall{session: s, version: version, id: id, messageID: wireField(wire, "id"), local: local, peer: peer, request: raw, intent: intent}}, nil
}

// OpenMCPSessionRequest authenticates identity, complete RPC bytes and fresh outer
// replay before inspecting the inner intent. Rejection never rolls back acceptance.
func OpenMCPSessionRequest(ctx context.Context, s *hpke.AuthenticatedCompletion010, version string, wire []byte) (*MCPSessionCall, error) {
	if ctx == nil || ctx.Err() != nil || s == nil || len(wire) > 32768 || CheckMCPVersion(version) != nil {
		return nil, ErrInvalid
	}
	local, peer, e := s.Participants()
	if e != nil {
		return nil, ErrInvalid
	}
	wire = append([]byte(nil), wire...)
	raw, e := s.OpenRequest(ctx, wire)
	if e != nil {
		return nil, ErrInvalid
	}
	id := wireField(raw, "id")
	intent, e := sessionRequest(version, id, raw, peer, local)
	if e != nil {
		return nil, ErrInvalid
	}
	return &MCPSessionCall{&mcpSessionCall{session: s, version: version, id: id, messageID: wireField(wire, "id"), local: local, peer: peer, request: raw, intent: intent, inbound: true}}, nil
}

// SealReply consumes this incoming invocation once, even on application failure.
// Use the existing endpoint's signed result. Transport binding is not Guard result verification.
func (c *MCPSessionCall) SealReply(ctx context.Context, raw []byte, ttl int64) ([]byte, error) {
	if c == nil || c.state == nil || ctx == nil {
		return nil, ErrInvalid
	}
	p := c.state
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.inbound || p.used {
		return nil, ErrInvalid
	}
	p.used = true
	raw = append([]byte(nil), raw...)
	success, code, e := sessionResult(p.version, p.id, raw, p.intent, p.local, p.peer)
	if e != nil {
		return nil, ErrInvalid
	}
	wire, e := p.session.SealResponse(ctx, p.messageID, raw, success, code, ttl)
	if e != nil {
		return nil, ErrInvalid
	}
	return wire, nil
}

// OpenReply releases exact RPC bytes only for this outgoing invocation. The caller
// must then call Client.AcceptMCPResponse to verify the Guard signature and consume
// output. A correlated authenticated malformed response consumes this permit.
func (c *MCPSessionCall) OpenReply(ctx context.Context, wire []byte) ([]byte, error) {
	if c == nil || c.state == nil || ctx == nil {
		return nil, ErrInvalid
	}
	p := c.state
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.inbound || p.used || len(wire) > 32768 {
		return nil, ErrInvalid
	}
	wire = append([]byte(nil), wire...)
	// Untrusted ID is only an early rejection filter, never an authentication claim.
	if wireField(wire, "message_id") != p.messageID {
		return nil, ErrInvalid
	}
	r, e := p.session.OpenResponse(ctx, wire)
	if e != nil {
		return nil, ErrInvalid
	}
	p.used = true
	success, code, e := sessionResult(p.version, p.id, r.Data, p.intent, p.peer, p.local)
	if e != nil || r.MessageID != p.messageID || r.Success != success || r.Error != code {
		return nil, ErrInvalid
	}
	return append([]byte(nil), r.Data...), nil
}
