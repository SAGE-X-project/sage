package guard010

import (
	"context"
	_ "embed"
	"encoding/json"
	"sync"
)

//go:embed mcp-tool.json
var mcpTool []byte

// MCPTool returns the closed tool input schema for this supported binding.
// Structural schema validation never replaces signed intent and policy checks.
func MCPTool(version string) ([]byte, error) {
	if CheckMCPVersion(version) != nil {
		return nil, ErrInvalid
	}
	return append([]byte(nil), mcpTool...), nil
}
func rpcObject(raw []byte, limit, size, depth int) (map[string]json.RawMessage, error) {
	if _, e := canonicalizeBounds(raw, limit, size, depth); e != nil {
		return nil, ErrInvalid
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil || m == nil {
		return nil, ErrInvalid
	}
	return m, nil
}
func rpcText(m map[string]json.RawMessage, key string) string {
	var s string
	_ = json.Unmarshal(m[key], &s)
	return s
}

// MCPRequest encodes the UUID string request-ID subset of JSON-RPC tools/call.
// It checks structure only. The client must authorize before protected handoff.
func MCPRequest(version, id string, intent []byte) ([]byte, error) {
	if CheckMCPVersion(version) != nil || !uuid.MatchString(id) {
		return nil, ErrInvalid
	}
	_, _, canonical, e := intentEnvelope(intent)
	if e != nil {
		return nil, ErrInvalid
	}
	return encode(map[string]any{"jsonrpc": "2.0", "id": id, "method": "tools/call", "params": map[string]any{"name": "sage_secure_call", "arguments": map[string]any{"envelope": json.RawMessage(canonical)}}}), nil
}

// ParseMCPRequest returns unauthenticated intent bytes. expectedID comes from the
// trusted transport invocation, not this document. Notifications, batches, direct
// tools, extra arguments and alternate ID types are unsupported and fail closed.
func ParseMCPRequest(version, expectedID string, raw []byte) ([]byte, error) {
	if CheckMCPVersion(version) != nil || !uuid.MatchString(expectedID) {
		return nil, ErrInvalid
	}
	m, e := rpcObject(raw, 4110, 2*MaxBytes, 36)
	if e != nil || len(m) != 4 || rpcText(m, "jsonrpc") != "2.0" || rpcText(m, "id") != expectedID || rpcText(m, "method") != "tools/call" {
		return nil, ErrInvalid
	}
	var params, args map[string]json.RawMessage
	if json.Unmarshal(m["params"], &params) != nil || len(params) != 2 || rpcText(params, "name") != "sage_secure_call" || json.Unmarshal(params["arguments"], &args) != nil || len(args) != 1 {
		return nil, ErrInvalid
	}
	_, _, canonical, e := intentEnvelope(args["envelope"])
	if e != nil {
		return nil, ErrInvalid
	}
	return canonical, nil
}
func parseMCPResponse(version, id string, raw []byte) ([]byte, error) {
	if CheckMCPVersion(version) != nil || !uuid.MatchString(id) {
		return nil, ErrInvalid
	}
	m, e := rpcObject(raw, 4110, 9*MaxBytes, 36)
	if e != nil || len(m) != 3 || rpcText(m, "jsonrpc") != "2.0" || rpcText(m, "id") != id {
		return nil, ErrInvalid
	}
	return ParseMCPResult(version, m["result"])
}

// MCPWireSender is trusted protected transport, called once under the client lock.
// It authenticates the peer and entire RPC bytes, binds the UUID, supplies fresh
// outer nonce/sequence, honors deadlines and never reenters or queues duplicates.
type MCPWireSender interface {
	Send(context.Context, string, []byte) error
}
type mcpSender struct {
	version string
	wire    MCPWireSender
}

// NewMCPClientSender rejects unsupported setup before any client handoff.
func NewMCPClientSender(version string, wire MCPWireSender) (ClientSender, error) {
	if CheckMCPVersion(version) != nil || wire == nil {
		return nil, ErrInvalid
	}
	return &mcpSender{version, wire}, nil
}
func (s *mcpSender) Commit(ctx context.Context, id string, intent []byte) error {
	if ctx == nil || ctx.Err() != nil {
		return ErrInvalid
	}
	raw, e := MCPRequest(s.version, id, intent)
	if e != nil {
		return ErrInvalid
	}
	return s.wire.Send(ctx, id, raw)
}

// AcceptMCPResponse consumes mismatched IDs, RPC errors and malformed data as
// unverified local failures. No error response becomes authenticated tool output.
func (c *Client) AcceptMCPResponse(ctx context.Context, t *ClientInvocation, version string, raw []byte) (*ClientDelivery, error) {
	if t == nil {
		return nil, ErrInvalid
	}
	envelope, e := parseMCPResponse(version, t.ID(), raw)
	if e != nil {
		_ = c.Failed(t)
		return nil, ErrInvalid
	}
	return c.Accept(ctx, t, envelope)
}

// MCPEndpoint binds one authenticated transport session to the protected gate.
// Route every protected tool call through it. A separate unguarded route cannot be
// secured by this object. Replay protection across sessions/restarts is transport
// policy; a new endpoint must not reset an existing session's request-ID history.
// At most 1024 attempts are accepted per endpoint; exhaustion requires closing the
// authenticated session. IDs are consumed even when parsing or authorization fails.
type MCPEndpoint struct {
	mu      sync.Mutex
	gate    *DispatchGate
	version string
	seen    map[string]bool
	closed  bool
}
type MCPReceipt struct {
	owner   *MCPEndpoint
	id      string
	intent  []byte
	receipt *DispatchReceipt
}

// NewMCPEndpoint validates locally supported setup; negotiation provenance is trusted host input.
func NewMCPEndpoint(version string, gate *DispatchGate) (*MCPEndpoint, error) {
	if CheckMCPVersion(version) != nil || gate == nil {
		return nil, ErrInvalid
	}
	return &MCPEndpoint{gate: gate, version: version, seen: map[string]bool{}}, nil
}
func (r *MCPReceipt) Created() bool   { return r != nil && r.receipt.Created() }
func (r *MCPReceipt) Committed() bool { return r != nil && r.receipt.Committed() }
func (r *MCPReceipt) State() string {
	if r == nil {
		return ""
	}
	return r.receipt.State()
}
func (r *MCPReceipt) IntentDigest() string {
	if r == nil {
		return ""
	}
	return r.receipt.IntentDigest()
}

// Dispatch serializes one transport attempt and delegates current authentication,
// policy, durable reservation and bounded tool commitment to the existing gate.
func (e *MCPEndpoint) Dispatch(ctx context.Context, id string, raw []byte) (*MCPReceipt, error) {
	if e == nil || ctx == nil {
		return nil, ErrInvalid
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.gate == nil || e.seen == nil || e.closed || !uuid.MatchString(id) || e.seen[id] || len(e.seen) >= 1024 {
		return nil, ErrInvalid
	}
	e.seen[id] = true
	intent, err := ParseMCPRequest(e.version, id, raw)
	if err != nil {
		return nil, ErrInvalid
	}
	receipt, err := e.gate.Dispatch(ctx, intent)
	if err != nil {
		return nil, ErrInvalid
	}
	return &MCPReceipt{e, id, intent, receipt}, nil
}

// Reply preserves the gate's one-response permit and immutable signed result.
// The caller must protect and send these bytes only on the receipt's invocation.
func (e *MCPEndpoint) Reply(ctx context.Context, r *MCPReceipt, s ResultSigner) (raw []byte, err error) {
	if e == nil || ctx == nil || r == nil || r.owner != e {
		return nil, ErrInvalid
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	defer func() {
		if recover() != nil {
			e.closed = true
			raw = nil
			err = ErrInvalid
		}
	}()
	if e.gate == nil || e.seen == nil || e.closed {
		return nil, ErrInvalid
	}
	result, err := e.gate.Reply(ctx, r.receipt, s)
	if err != nil {
		return nil, ErrInvalid
	}
	v, err := VerifyResult(ctx, result, s, acceptedInvocation(r.intent))
	if err != nil {
		return nil, ErrInvalid
	}
	body, err := v.MCPResult(e.version)
	if err != nil {
		return nil, ErrInvalid
	}
	return mcpRPCResponse(r.id, body), nil
}

// Close retires this session boundary without closing the shared execution gate.
func (e *MCPEndpoint) Close() {
	if e != nil {
		e.mu.Lock()
		defer e.mu.Unlock()
		e.closed = true
	}
}

// The endpoint owns both the ID and the generated MCP result. Use the JSON
// encoder and canonicalizer, never string concatenation, to frame the response.
func mcpRPCResponse(id string, body []byte) []byte {
	return encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": json.RawMessage(body)})
}
