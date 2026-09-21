package hpke

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
)

type recordRequest010 struct {
	http     *httpContext010
	wire     []byte
	terminal bool
}

// SessionResponse010 is released only after correlation, signature, AEAD and
// transactional replay acceptance. Success/Error describe the authenticated peer
// result, not authorization to execute it. Application validation remains required.
type SessionResponse010 struct {
	MessageID string
	Success   bool
	Error     string
	Data      []byte
}

func responseError010(code string) bool {
	switch code {
	case "authentication_failed", "policy_denied", "operation_failed", "unavailable":
		return true
	}
	return false
}
func retained010(r *recordRequest010) (map[string]json.RawMessage, string, error) {
	if r == nil || r.terminal {
		return nil, "", errCompletion010
	}
	var w map[string]json.RawMessage
	if json.Unmarshal(r.wire, &w) != nil {
		return nil, "", errCompletion010
	}
	h := sha256.Sum256(canon010(w))
	return w, base64.RawURLEncoding.EncodeToString(h[:]), nil
}

// SealResponse emits one terminal signed/encrypted response to a request accepted
// by this exact session. The complete signed request is retained internally. Error
// responses use the same encryption and binding. A retry must reuse returned bytes;
// a second emission is rejected. No caller-provided request hash or verified flag.
func (s *AuthenticatedCompletion010) SealResponse(ctx context.Context, messageID string, data []byte, success bool, code string, ttl int64) ([]byte, error) {
	e := s.endpoint
	e.mu.Lock()
	defer e.mu.Unlock()
	if s.httpTarget != "" || (s.lifetime != nil && s.lifetime.http.Load()) {
		return nil, errCompletion010
	}
	return s.sealResponse010(ctx, messageID, data, success, code, ttl)
}
func (s *AuthenticatedCompletion010) sealResponse010(ctx context.Context, messageID string, data []byte, success bool, code string, ttl int64) ([]byte, error) {
	e := s.endpoint
	start, x := e.sample()
	if x != nil || s.recordLive(start) != nil {
		s.destroy()
		return nil, errCompletion010
	}
	retained := s.received[messageID]
	request, hash, x := retained010(retained)
	if x != nil || (!s.initiator && !s.confirmed) || ttl < 1 || ttl > 300 || len(data) > 16348 || (success && code != "") || (!success && !responseError010(code)) {
		return nil, errCompletion010
	}
	if e.current(ctx, s.a, s.b) != nil {
		s.destroy()
		return nil, errCompletion010
	}
	role := "initiator"
	if !s.initiator {
		role = "responder"
	}
	w, x := envelope010(e.did, str010(request, "did"), e.kid, s.tuple["ctx"], "responder", nil, start.Unix, start.Unix+ttl)
	if x != nil {
		return nil, errCompletion010
	}
	if w["id"] == str010(request, "id") || w["nonce"] == str010(request, "nonce") {
		return nil, errCompletion010
	}
	w["role"] = role
	w["encoding"] = "session"
	w["session_id"] = s.tuple["sid"]
	w["message_id"] = messageID
	w["request_hash"] = hash
	w["success"] = success
	if !success {
		w["error"] = code
	}
	delete(w, "data")
	aad := canon010(w)
	if len(aad) > 4033 {
		return nil, errCompletion010
	}
	record, x := s.records.Seal(data, aad)
	if x != nil {
		s.destroy()
		return nil, errCompletion010
	}
	w["data"] = base64.RawURLEncoding.EncodeToString(record)
	result := sign010(w, "sage-wire-response|0.10.0\n", e.signing)
	end, x := s.recordGate(start, start.Unix+ttl)
	if x != nil {
		return nil, x
	}
	retained.terminal = true
	s.active = end
	s.lifetime.active.Store(end.MonoMS)
	s.lifetime.wall.Store(end.Unix)
	return result, nil
}

// OpenResponse accepts one terminal response to an internally retained sent
// request. Invalid responses do not consume the request or any replay state.
// Terminal acceptance is serialized with ID/nonce/sequence publication; a later
// application rejection never reopens the request. Unsolicited responses fail.
func (s *AuthenticatedCompletion010) OpenResponse(ctx context.Context, raw []byte) (*SessionResponse010, error) {
	e := s.endpoint
	e.mu.Lock()
	defer e.mu.Unlock()
	if s.httpTarget != "" || (s.lifetime != nil && s.lifetime.http.Load()) {
		return nil, errCompletion010
	}
	return s.openResponse010(ctx, raw, nil)
}
func (s *AuthenticatedCompletion010) openResponse010(ctx context.Context, raw []byte, proof *httpProof010) (*SessionResponse010, error) {
	e := s.endpoint
	start, x := e.sample()
	if x != nil || s.recordLive(start) != nil {
		s.destroy()
		return nil, errCompletion010
	}
	if !s.initiator && !s.confirmed {
		return nil, errCompletion010
	}
	if proof != nil {
		start = proof.start
	}
	w, record, x := sessionRecord010(raw, start.Unix, true)
	if x != nil {
		return nil, errCompletion010
	}
	messageID := str010(w, "message_id")
	retained := s.sent[messageID]
	request, hash, x := retained010(retained)
	if x != nil || str010(w, "request_hash") != hash || str010(w, "id") == str010(request, "id") || str010(w, "nonce") == str010(request, "nonce") {
		return nil, errCompletion010
	}
	if proof != nil && s.verifyHTTP010(proof, w) != nil {
		return nil, errCompletion010
	}
	data, x := s.acceptRecord010(ctx, start, w, record, true)
	if x != nil {
		return nil, errCompletion010
	}
	retained.terminal = true
	return &SessionResponse010{MessageID: messageID, Success: string(w["success"]) == "true", Error: str010(w, "error"), Data: data}, nil
}
