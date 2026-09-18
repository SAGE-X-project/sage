package hpke

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/sage-x-project/sage/pkg/agent/crypto/jcs"
	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

// RecordReplayStore010 extends the mandatory trusted replay store with a
// transaction boundary. ReserveRecord must check both scoped ID and nonce, stage
// durable denial state, then call validate exactly once at final publication.
// All fallible/blocking storage work must precede validate. Failure publishes no
// acceptance or plaintext; staged durable denial may remain to reject retries.
// Success requires both denial keys durable.
// It must not reenter the endpoint. The endpoint lock spans this transaction,
// sequence consumption and provisional confirmation. A crash discards sessions;
// durable replay denial remains. There is no session restoration protocol.
// Implementations unable to provide this contract must not enable record receive.
type RecordReplayStore010 interface {
	ReserveRecord(Replay010, func() error) error
}

func (s *AuthenticatedCompletion010) recordLive(t registry010.Stamp) error {
	if s.closed || s.records == nil || t.MonoMS < s.created.MonoMS || t.MonoMS < s.active.MonoMS || t.Unix < s.created.Unix || t.MonoMS-s.created.MonoMS >= 3600000 || t.MonoMS-s.active.MonoMS >= 600000 || (!s.initiator && !s.confirmed && !pendingLive010(t, s.created, s.expires)) {
		return errCompletion010
	}
	return nil
}
func (s *AuthenticatedCompletion010) recordGate(start registry010.Stamp, expires int64) (registry010.Stamp, error) {
	t, x := s.endpoint.sample()
	if x != nil || s.recordLive(t) != nil || t.MonoMS-start.MonoMS > 5000 || t.Unix >= expires || !pinnedLive010(t.Unix, s.a, s.b) {
		s.destroy()
		return t, errCompletion010
	}
	return t, nil
}
func requestAAD010(m map[string]json.RawMessage) []byte {
	a := map[string]json.RawMessage{}
	for k, v := range m {
		if k != "payload" && k != "data" && k != "signature" {
			a[k] = v
		}
	}
	return canon010(a)
}
func sessionRequest010(raw []byte, now int64) (map[string]json.RawMessage, []byte, error) {
	return sessionRecord010(raw, now, false)
}
func sessionRecord010(raw []byte, now int64, response bool) (map[string]json.RawMessage, []byte, error) {
	// This bounded subset deliberately shares the 32 KiB envelope admission cap.
	if len(raw) > 32768 {
		return nil, nil, errCompletion010
	}
	if _, err := jcs.Canonicalize(raw); err != nil {
		return nil, nil, errCompletion010
	}
	names := append(append([]string{}, wireFields010...), "session_id")
	field := "payload"
	if response {
		field = "data"
		names = append(names, "message_id", "request_hash", "success")
		var probe map[string]json.RawMessage
		if json.Unmarshal(raw, &probe) != nil {
			return nil, nil, errCompletion010
		}
		if string(probe["success"]) == "false" {
			names = append(names, "error")
		} else if string(probe["success"]) != "true" {
			return nil, nil, errCompletion010
		}
	}
	names = append(names, field)
	m, x := rawObject010(raw, names)
	if x != nil || str010(m, "encoding") != "session" || str010(m, "version") != "0.10.0" || !uuid010.MatchString(str010(m, "id")) {
		return nil, nil, errCompletion010
	}
	created, x := int010(m, "created")
	expires, y := int010(m, "expires")
	if x != nil || y != nil || created < 0 || expires > 9007199254740991 || expires <= created || expires-created > 300 || created > now+30 || now >= expires {
		return nil, nil, errCompletion010
	}
	for k, n := range map[string]int{"nonce": 16, "signature": 64, "session_id": 16} {
		if _, x = binary010(str010(m, k), n); x != nil {
			return nil, nil, errCompletion010
		}
	}
	if response {
		if !uuid010.MatchString(str010(m, "message_id")) {
			return nil, nil, errCompletion010
		}
		if _, x = binary010(str010(m, "request_hash"), 32); x != nil {
			return nil, nil, errCompletion010
		}
		if string(m["success"]) == "false" && !responseError010(str010(m, "error")) {
			return nil, nil, errCompletion010
		}
	}
	encoded := str010(m, field)
	body, x := base64.RawURLEncoding.Strict().DecodeString(encoded)
	if x != nil || len(body) < 36 || len(body) > 16384 || base64.RawURLEncoding.EncodeToString(body) != encoded || len(requestAAD010(m)) > 4033 {
		return nil, nil, errCompletion010
	}
	return m, body, nil
}

// SealRequest emits one signed session request, using fixed participant bindings
// and JCS envelope AAD. It is bounded to 16348 plaintext bytes, with no metadata,
// task fields or HTTP binding. Return is the emission boundary; retries reuse
// exact returned bytes. A provisional responder cannot send. No application
// authorization or protected execution is implied by this cryptographic API.
func (s *AuthenticatedCompletion010) SealRequest(ctx context.Context, plaintext []byte, ttl int64) ([]byte, error) {
	e := s.endpoint
	e.mu.Lock()
	defer e.mu.Unlock()
	if s.httpTarget != "" {
		return nil, errCompletion010
	}
	return s.sealRequest010(ctx, plaintext, ttl)
}
func (s *AuthenticatedCompletion010) sealRequest010(ctx context.Context, plaintext []byte, ttl int64) ([]byte, error) {
	e := s.endpoint
	start, x := e.sample()
	if x != nil || s.recordLive(start) != nil {
		s.destroy()
		return nil, errCompletion010
	}
	if (!s.initiator && !s.confirmed) || ttl < 1 || ttl > 300 || len(plaintext) > 16348 {
		return nil, errCompletion010
	}
	if e.current(ctx, s.a, s.b) != nil {
		s.destroy()
		return nil, errCompletion010
	}
	peer, role := s.tuple["respDid"], "initiator"
	if !s.initiator {
		peer, role = s.tuple["initDid"], "responder"
	}
	w, x := envelope010(e.did, peer, e.kid, s.tuple["ctx"], "initiator", nil, start.Unix, start.Unix+ttl)
	if x != nil {
		return nil, errCompletion010
	}
	w["role"] = role
	w["encoding"] = "session"
	w["session_id"] = s.tuple["sid"]
	delete(w, "payload")
	aad := canon010(w)
	if len(aad) > 4033 {
		return nil, errCompletion010
	}
	wire, x := s.records.Seal(plaintext, aad)
	if x != nil {
		s.destroy()
		return nil, errCompletion010
	}
	w["payload"] = base64.RawURLEncoding.EncodeToString(wire)
	result := sign010(w, "sage-wire-request|0.10.0\n", e.signing)
	end, x := s.recordGate(start, start.Unix+ttl)
	if x != nil {
		return nil, x
	}
	s.active = end
	if s.sent == nil {
		s.sent = map[string]*recordRequest010{}
	}
	s.sent[w["id"].(string)] = &recordRequest010{wire: append([]byte(nil), result...)}
	return result, nil
}

// OpenRequest authenticates the complete signed session request and atomically
// consumes transport ID/nonce, record sequence and responder confirmation before
// releasing plaintext. Application validation/authorization follows this call;
// rejection there must not undo this cryptographic acceptance. This is not dispatch.
func (s *AuthenticatedCompletion010) OpenRequest(ctx context.Context, raw []byte) ([]byte, error) {
	e := s.endpoint
	e.mu.Lock()
	defer e.mu.Unlock()
	if s.httpTarget != "" {
		return nil, errCompletion010
	}
	return s.openRequest010(ctx, raw, nil)
}
func (s *AuthenticatedCompletion010) openRequest010(ctx context.Context, raw []byte, proof *httpProof010) ([]byte, error) {
	e := s.endpoint
	start, x := e.sample()
	if x != nil || s.recordLive(start) != nil {
		s.destroy()
		return nil, errCompletion010
	}
	if proof != nil {
		start = proof.start
	}
	w, wire, x := sessionRequest010(raw, start.Unix)
	if x != nil {
		return nil, errCompletion010
	}
	if proof != nil && s.verifyHTTP010(proof, w) != nil {
		return nil, errCompletion010
	}
	plaintext, x := s.acceptRecord010(ctx, start, w, wire, false)
	if x != nil {
		return nil, x
	}
	if s.received == nil {
		s.received = map[string]*recordRequest010{}
	}
	s.received[str010(w, "id")] = &recordRequest010{wire: canon010(w)}
	return plaintext, nil
}
func (s *AuthenticatedCompletion010) acceptRecord010(ctx context.Context, start registry010.Stamp, w map[string]json.RawMessage, wire []byte, response bool) ([]byte, error) {
	e := s.endpoint
	did, recipient, kid, role, key := s.tuple["initDid"], s.tuple["respDid"], s.tuple["initKid"], "initiator", s.a.Signing()
	if s.initiator {
		did, recipient, kid, role, key = s.tuple["respDid"], s.tuple["initDid"], s.tuple["respKid"], "responder", s.b.Signing()
	}
	for k, v := range map[string]string{"did": did, "recipient": recipient, "kid": kid, "role": role, "context_id": s.tuple["ctx"], "session_id": s.tuple["sid"]} {
		if str010(w, k) != v {
			return nil, errCompletion010
		}
	}
	if e.current(ctx, s.a, s.b) != nil {
		s.destroy()
		return nil, errCompletion010
	}
	if verifyWire010(w, response, key) != nil {
		return nil, errCompletion010
	}
	store, ok := e.replay.(RecordReplayStore010)
	if !ok {
		return nil, errCompletion010
	}
	entry := reservation010(w)
	entry.Context = "" // Only initiation reserves a context.
	var accepted registry010.Stamp
	gateFailed := false
	// Do not Close the record owner inside OpenChecked: its mutex is held.
	plaintext, x := s.records.OpenChecked(wire, requestAAD010(w), func() error {
		called := false
		err := store.ReserveRecord(entry, func() error {
			if called {
				return errCompletion010
			}
			called = true
			var err error
			accepted, err = e.sample()
			if err != nil || s.recordLive(accepted) != nil || accepted.MonoMS-start.MonoMS > 5000 || accepted.Unix >= entry.Expires || !pinnedLive010(accepted.Unix, s.a, s.b) {
				gateFailed = true
				return errCompletion010
			}
			return nil
		})
		if err != nil || !called || gateFailed {
			return errCompletion010
		}
		return nil
	})
	if x != nil {
		if gateFailed {
			s.destroy()
		}
		return nil, errCompletion010
	}
	s.confirmed = true
	s.active = accepted
	return plaintext, nil
}
