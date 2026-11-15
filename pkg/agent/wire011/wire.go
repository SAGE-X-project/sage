package wire011

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"
)

// Clock returns trusted UTC time. An error means time is unavailable or
// uncertain, and every check that needs time then fails.
type Clock interface{ Now() (time.Time, error) }

// KeyResolver returns the Ed25519 public key named by a full key URL only if
// that key is currently active, proven and authorized for HTTP signing. Any
// error, including an unknown key or an unavailable source, refuses the message.
type KeyResolver interface {
	ResolveSigningKey(ctx context.Context, kid string) (ed25519.PublicKey, error)
}

// ReplayStore atomically reserves a (sender, recipient, nonce) triple and the
// message id in the same scope until the given time. It returns an error if
// either is already reserved or the store cannot accept more state.
type ReplayStore interface {
	Reserve(ctx context.Context, sender, recipient, nonce, id string, until time.Time) error
}

// Event is one trust-boundary record (decision D22). It names the stage, the
// party before and after, and the internal reason of a refusal. It never
// carries keys, signatures or payload bytes.
type Event struct {
	Stage       string
	Correlation string
	From, To    string
	Decision    string
	Reason      string
}

// Tracer receives trust-boundary events. A nil Tracer discards them.
type Tracer interface {
	Trace(ctx context.Context, e Event)
}

// SlogTracer writes events to a slog logger at debug level.
type SlogTracer struct{ Logger *slog.Logger }

func (t SlogTracer) Trace(ctx context.Context, e Event) {
	if t.Logger == nil {
		return
	}
	t.Logger.LogAttrs(ctx, slog.LevelDebug, "sage.wire011", slog.String("stage", e.Stage),
		slog.String("correlation", e.Correlation), slog.String("from", e.From), slog.String("to", e.To),
		slog.String("decision", e.Decision), slog.String("reason", e.Reason))
}

func trace(ctx context.Context, t Tracer, e Event) {
	if t != nil {
		t.Trace(ctx, e)
	}
}

// Signer signs requests and responses as one SAGE agent key.
type Signer struct {
	DID   string
	KID   string
	Key   ed25519.PrivateKey
	Clock Clock
	// TTL is the message lifetime in seconds, at most 300.
	TTL int64
}

func (s *Signer) times() (int64, int64, error) {
	if s == nil || s.Clock == nil || kidDID(s.KID) != s.DID || len(s.Key) != ed25519.PrivateKeySize || s.TTL <= 0 || s.TTL > maxLifetime {
		return 0, 0, fmt.Errorf("wire011: invalid signer")
	}
	now, err := s.Clock.Now()
	if err != nil {
		return 0, 0, fmt.Errorf("wire011: clock: %w", err)
	}
	created := now.Unix()
	return created, created + s.TTL, nil
}

// RequestOptions are the optional authenticated request members.
type RequestOptions struct {
	ContextID, TaskID string
	Metadata          map[string]string
}

// SentRequest is the sender's retained state for verifying the one response.
type SentRequest struct {
	envelope Request
	hash     string
	http     httpContext

	mu       sync.Mutex
	answered bool
}

// Envelope returns the signed request envelope.
func (s *SentRequest) Envelope() Request { return s.envelope }

// SignRequest builds and signs a POST to the recipient's canonical https endpoint.
func (s *Signer) SignRequest(recipient, target string, payload []byte, opts RequestOptions) (*HTTPRequest, *SentRequest, error) {
	created, expires, err := s.times()
	if err != nil {
		return nil, nil, err
	}
	target, authority, err := endpoint(target)
	if err != nil {
		return nil, nil, err
	}
	if len(payload) > maxPayload {
		return nil, nil, fmt.Errorf("wire011: payload too large")
	}
	id, err := newUUID()
	if err != nil {
		return nil, nil, err
	}
	nonce, err := newNonce()
	if err != nil {
		return nil, nil, err
	}
	env := Request{Version: Version, ID: id, DID: s.DID, Recipient: recipient, KID: s.KID, Created: created, Expires: expires,
		Nonce: nonce, Encoding: "plain", Payload: b64(payload), ContextID: opts.ContextID, TaskID: opts.TaskID, Metadata: copyMap(opts.Metadata)}
	if err := validCommon(env.Version, env.ID, env.DID, env.Recipient, env.KID, env.Created, env.Expires, env.Nonce, env.Encoding,
		"", env.ContextID, env.TaskID, env.Metadata); err != nil {
		return nil, nil, fmt.Errorf("wire011: request: %w", err)
	}
	if env.Signature, err = signEnvelope(requestDomain, env.unsigned(), s.Key); err != nil {
		return nil, nil, err
	}
	body, err := json.Marshal(env)
	if err != nil {
		return nil, nil, err
	}
	h := map[string]string{"content-type": "application/json", "content-digest": contentDigest(body),
		"x-sage-did": s.DID, "x-sage-version": Version}
	input := signatureInput(requestComponents, s.KID, created, expires, nonce)
	sig := ed25519.Sign(s.Key, requestBase("POST", target, authority, h, input))
	h["signature-input"] = "sig1=" + input
	h["signature"] = "sig1=:" + base64Std(sig) + ":"
	hash, err := requestHash(env)
	if err != nil {
		return nil, nil, err
	}
	req := &HTTPRequest{Method: "POST", TargetURI: target, Authority: authority, Body: body, Fields: fieldsOf(h)}
	sent := &SentRequest{envelope: env, hash: hash, http: httpContext{"POST", target, authority, h["content-digest"],
		h["signature"], Version}}
	return req, sent, nil
}

// Receiver verifies requests addressed to one agent at one endpoint.
type Receiver struct {
	DID      string // this agent; requests for another recipient are refused
	Endpoint string // canonical https target this agent serves
	Keys     KeyResolver
	Replay   ReplayStore
	Clock    Clock
	Trace    Tracer
}

// VerifiedRequest is an accepted request and the state its response must bind.
type VerifiedRequest struct {
	Envelope Request
	Payload  []byte
	hash     string
	http     httpContext
}

// VerifyRequest follows MSG-04: bounded field checks, fixed version, coverage
// and timing, body digest, key resolution, both signatures, envelope and
// header agreement, then one atomic replay reservation. Nothing is returned
// before every check passes.
func (r *Receiver) VerifyRequest(ctx context.Context, req HTTPRequest) (*VerifiedRequest, error) {
	if r == nil {
		return nil, ErrAuthentication
	}
	v, reason := r.verifyRequest(ctx, req)
	if reason != "" {
		trace(ctx, r.Trace, Event{Stage: "request.refused", To: r.DID, Decision: "refused", Reason: reason})
		return nil, ErrAuthentication
	}
	trace(ctx, r.Trace, Event{Stage: "request.accepted", Correlation: v.Envelope.ID, From: v.Envelope.DID, To: r.DID, Decision: "accepted"})
	return v, nil
}

func (r *Receiver) verifyRequest(ctx context.Context, req HTTPRequest) (*VerifiedRequest, string) {
	if r.Keys == nil || r.Replay == nil || r.Clock == nil || !validDID(r.DID) {
		return nil, "receiver not configured"
	}
	target, authority, err := endpoint(r.Endpoint)
	if err != nil {
		return nil, "receiver endpoint"
	}
	if req.Method != "POST" || req.TargetURI != target || req.Authority != authority {
		return nil, "route"
	}
	h, err := checkFields(req.Fields, req.Body)
	if err != nil {
		return nil, "fields: " + err.Error()
	}
	if host, ok := h["host"]; ok && host != authority {
		return nil, "host"
	}
	p, err := parseSignature(h, false)
	if err != nil {
		return nil, "signature-input: " + err.Error()
	}
	if reason := r.checkTime(p.created, p.expires); reason != "" {
		return nil, reason
	}
	var env Request
	if err := decodeClosed(req.Body, requestMembers, &env); err != nil {
		return nil, "envelope: " + err.Error()
	}
	if err := validCommon(env.Version, env.ID, env.DID, env.Recipient, env.KID, env.Created, env.Expires, env.Nonce,
		env.Encoding, env.Signature, env.ContextID, env.TaskID, env.Metadata); err != nil || env.Signature == "" {
		return nil, "envelope: " + fmt.Sprint(err)
	}
	payload, err := decodeB64(env.Payload, -1)
	if err != nil {
		return nil, "payload"
	}
	switch {
	case env.Recipient != r.DID:
		return nil, "recipient"
	case h["x-sage-did"] != env.DID:
		return nil, "x-sage-did does not match the envelope"
	case p.keyid != env.KID || p.created != env.Created || p.expires != env.Expires || p.nonce != env.Nonce:
		return nil, "http parameters do not match the envelope"
	}
	if reason := checkProjections(h, env.ID, env.ContextID, env.TaskID); reason != "" {
		return nil, reason
	}
	key, err := r.Keys.ResolveSigningKey(ctx, env.KID)
	if err != nil || len(key) != ed25519.PublicKeySize {
		return nil, "key resolution"
	}
	trace(ctx, r.Trace, Event{Stage: "key.resolved", Correlation: env.ID, From: env.KID, To: r.DID, Decision: "resolved"})
	if !ed25519.Verify(key, requestBase(req.Method, req.TargetURI, req.Authority, h, p.input), p.signature) {
		return nil, "http signature"
	}
	if !verifyEnvelope(requestDomain, env.unsigned(), env.Signature, key) {
		return nil, "envelope signature"
	}
	if err := r.Replay.Reserve(ctx, env.DID, r.DID, env.Nonce, env.ID, time.Unix(env.Expires+clockSkew, 0)); err != nil {
		return nil, "replay"
	}
	hash, err := requestHash(env)
	if err != nil {
		return nil, "request hash"
	}
	return &VerifiedRequest{Envelope: env, Payload: payload, hash: hash,
		http: httpContext{req.Method, req.TargetURI, req.Authority, h["content-digest"], h["signature"], h["x-sage-version"]}}, ""
}

func (r *Receiver) checkTime(created, expires int64) string {
	return checkTime(r.Clock, created, expires)
}

func checkTime(clock Clock, created, expires int64) string {
	now, err := clock.Now()
	if err != nil {
		return "clock unavailable"
	}
	t := now.Unix()
	switch {
	case expires <= created || expires-created > maxLifetime:
		return "lifetime"
	case created > t+clockSkew:
		return "created in the future"
	case t >= expires+clockSkew:
		return "expired"
	}
	return ""
}

// checkProjections enforces TRANSPORT-05: optional id headers must match the body.
func checkProjections(h map[string]string, id, contextID, taskID string) string {
	for name, want := range map[string]string{"x-sage-message-id": id, "x-sage-context-id": contextID, "x-sage-task-id": taskID} {
		// A projection without a body counterpart is a mismatch, even if empty.
		if got, ok := h[name]; ok && (want == "" || got != want) {
			return name + " does not match the envelope"
		}
	}
	return ""
}

// SignResponse answers an accepted request. success=false needs an errCode
// from TRANSPORT-03; success=true must not have one.
func (s *Signer) SignResponse(v *VerifiedRequest, status int, success bool, data []byte, errCode string) (*HTTPResponse, error) {
	if s == nil || v == nil || v.Envelope.Recipient != s.DID {
		return nil, fmt.Errorf("wire011: response signer is not the request recipient")
	}
	if status < 200 || status > 599 || status == 204 || success == (errCode != "") || errCode != "" && !responseErrors[errCode] {
		return nil, fmt.Errorf("wire011: invalid response outcome")
	}
	created, expires, err := s.times()
	if err != nil {
		return nil, err
	}
	id, err := newUUID()
	if err != nil {
		return nil, err
	}
	nonce, err := newNonce()
	if err != nil {
		return nil, err
	}
	env := Response{Version: Version, ID: id, DID: s.DID, Recipient: v.Envelope.DID, KID: s.KID, Created: created,
		Expires: expires, Nonce: nonce, Encoding: "plain", MessageID: v.Envelope.ID, RequestHash: v.hash, Success: success,
		Data: b64(data), Error: errCode, ContextID: v.Envelope.ContextID, TaskID: v.Envelope.TaskID}
	if env.Signature, err = signEnvelope(responseDomain, env.unsigned(), s.Key); err != nil {
		return nil, err
	}
	body, err := json.Marshal(env)
	if err != nil {
		return nil, err
	}
	h := map[string]string{"content-type": "application/json", "content-digest": contentDigest(body),
		"x-sage-did": s.DID, "x-sage-version": Version}
	input := signatureInput(responseComponents, s.KID, created, expires, nonce)
	sig := ed25519.Sign(s.Key, responseBase(status, v.http, h, input))
	h["signature-input"] = "sig1=" + input
	h["signature"] = "sig1=:" + base64Std(sig) + ":"
	return &HTTPResponse{Status: status, Fields: fieldsOf(h), Body: body}, nil
}

// VerifiedResponse is the accepted terminal response of a sent request.
type VerifiedResponse struct {
	Envelope Response
	Data     []byte
}

// ResponseVerifier checks responses with the sender's key source and replay state.
type ResponseVerifier struct {
	Keys   KeyResolver
	Replay ReplayStore
	Clock  Clock
	Trace  Tracer
}

// VerifyResponse accepts at most one terminal response per sent request. The
// signer must be the request recipient and the response must bind this exact request.
func (rv *ResponseVerifier) VerifyResponse(ctx context.Context, sent *SentRequest, resp HTTPResponse) (*VerifiedResponse, error) {
	if rv == nil {
		return nil, ErrAuthentication
	}
	v, reason := rv.verifyResponse(ctx, sent, resp)
	if reason != "" {
		trace(ctx, rv.Trace, Event{Stage: "response.refused", Decision: "refused", Reason: reason})
		return nil, ErrAuthentication
	}
	trace(ctx, rv.Trace, Event{Stage: "response.accepted", Correlation: v.Envelope.MessageID, From: v.Envelope.DID,
		To: v.Envelope.Recipient, Decision: "accepted"})
	return v, nil
}

func (rv *ResponseVerifier) verifyResponse(ctx context.Context, sent *SentRequest, resp HTTPResponse) (*VerifiedResponse, string) {
	if rv.Keys == nil || rv.Replay == nil || rv.Clock == nil || sent == nil {
		return nil, "verifier not configured"
	}
	if resp.Status < 200 || resp.Status > 599 || resp.Status == 204 {
		return nil, "status"
	}
	h, err := checkFields(resp.Fields, resp.Body)
	if err != nil {
		return nil, "fields: " + err.Error()
	}
	p, err := parseSignature(h, true)
	if err != nil {
		return nil, "signature-input: " + err.Error()
	}
	if reason := checkTime(rv.Clock, p.created, p.expires); reason != "" {
		return nil, reason
	}
	var env Response
	if err := decodeClosed(resp.Body, responseMembers, &env); err != nil {
		return nil, "envelope: " + err.Error()
	}
	if err := validCommon(env.Version, env.ID, env.DID, env.Recipient, env.KID, env.Created, env.Expires, env.Nonce,
		env.Encoding, env.Signature, env.ContextID, env.TaskID, env.Metadata); err != nil || env.Signature == "" {
		return nil, "envelope: " + fmt.Sprint(err)
	}
	req := sent.envelope
	switch {
	case env.DID != req.Recipient || env.Recipient != req.DID:
		return nil, "response is not from the request recipient"
	case env.MessageID != req.ID || env.RequestHash != sent.hash:
		return nil, "response does not bind this request"
	case env.ContextID != req.ContextID || env.TaskID != req.TaskID:
		return nil, "response context differs"
	case env.Success == (env.Error != "") || env.Error != "" && !responseErrors[env.Error]:
		return nil, "response outcome"
	case h["x-sage-did"] != env.DID:
		return nil, "x-sage-did does not match the envelope"
	case p.keyid != env.KID || p.created != env.Created || p.expires != env.Expires || p.nonce != env.Nonce:
		return nil, "http parameters do not match the envelope"
	}
	if reason := checkProjections(h, env.ID, env.ContextID, env.TaskID); reason != "" {
		return nil, reason
	}
	data, err := decodeB64(env.Data, -1)
	if err != nil {
		return nil, "data"
	}
	key, err := rv.Keys.ResolveSigningKey(ctx, env.KID)
	if err != nil || len(key) != ed25519.PublicKeySize {
		return nil, "key resolution"
	}
	if !ed25519.Verify(key, responseBase(resp.Status, sent.http, h, p.input), p.signature) {
		return nil, "http signature"
	}
	if !verifyEnvelope(responseDomain, env.unsigned(), env.Signature, key) {
		return nil, "envelope signature"
	}
	sent.mu.Lock()
	defer sent.mu.Unlock()
	if sent.answered {
		return nil, "second terminal response"
	}
	if err := rv.Replay.Reserve(ctx, env.DID, env.Recipient, env.Nonce, env.ID, time.Unix(env.Expires+clockSkew, 0)); err != nil {
		return nil, "replay"
	}
	sent.answered = true
	return &VerifiedResponse{Envelope: env, Data: data}, ""
}

// Quarantine is how long an endpoint without durable replay state refuses
// protected messages after it starts: the 300-second lifetime plus both
// 30-second allowances (MSG-05).
const Quarantine = 360 * time.Second

// MemoryReplay is a bounded in-memory ReplayStore. It refuses new entries
// when full instead of evicting live ones. Its state is lost on restart, so
// it refuses every reservation until Started+Quarantine, and it fails closed
// if its clock moves backwards. Give it the same Clock as the Receiver.
type MemoryReplay struct {
	Capacity int
	Clock    Clock
	// Started is when this process began accepting traffic. The zero value refuses.
	Started time.Time

	mu      sync.Mutex
	entries map[string]time.Time
	highest time.Time
}

func (m *MemoryReplay) Reserve(_ context.Context, sender, recipient, nonce, id string, until time.Time) error {
	if m == nil || m.Capacity <= 0 || m.Clock == nil || m.Started.IsZero() {
		return errors.New("wire011: replay store not configured")
	}
	now, err := m.Clock.Now()
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if now.Before(m.highest) {
		return errors.New("wire011: clock moved backwards")
	}
	m.highest = now
	if now.Before(m.Started.Add(Quarantine)) {
		return errors.New("wire011: replay state quarantine")
	}
	if m.entries == nil {
		m.entries = map[string]time.Time{}
	}
	for k, exp := range m.entries {
		if !now.Before(exp) {
			delete(m.entries, k)
		}
	}
	scope := strconv.Quote(sender) + " " + strconv.Quote(recipient)
	nonceKey, idKey := scope+" nonce "+nonce, scope+" id "+id
	if _, used := m.entries[nonceKey]; used {
		return errors.New("wire011: nonce replayed")
	}
	if _, used := m.entries[idKey]; used {
		return errors.New("wire011: message id replayed")
	}
	if len(m.entries)+2 > m.Capacity {
		return errors.New("wire011: replay store full")
	}
	m.entries[nonceKey], m.entries[idKey] = until, until
	return nil
}

func copyMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func fieldsOf(h map[string]string) []Field {
	order := []string{"content-type", "content-digest", "x-sage-did", "x-sage-version", "signature-input", "signature"}
	out := make([]Field, 0, len(order))
	for _, k := range order {
		out = append(out, Field{Name: k, Value: h[k]})
	}
	return out
}
