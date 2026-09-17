package hpke

// This module authenticates the bounded, metadata-free plain handshake carriage.
// HTTP binding, durable replay storage and application dispatch
// are separate dependencies/integrations; this is not full WireTransport conformance.
import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"sync"

	"github.com/google/uuid"
	"github.com/sage-x-project/sage/pkg/agent/crypto/jcs"
	"github.com/sage-x-project/sage/pkg/agent/registry010"
	"github.com/sage-x-project/sage/pkg/agent/session"
)

var errCompletion010 = errors.New("authentication failed")

// Replay010 describes a fully authenticated plain envelope. Reserve must atomically
// reject reused ID or nonce in the sender/recipient scope, retain both through
// Expires+30, reject reused nonempty Context per sender, fail closed on capacity/clock/storage failure, and apply restart
// recovery or 360-second quarantine. It must not return success before durability.
type Replay010 struct {
	Sender, Recipient, ID, Nonce, Context string
	Expires                               int64
}

// ReplayStore010 is a required trusted deployment dependency, never a peer flag.
type ReplayStore010 interface{ Reserve(Replay010) error }

// CompletionEndpoint010 serializes one endpoint's handshake operations. Each
// endpoint must use the same durable replay store across transports and restarts.
// Registry and Clock are trusted local dependencies; no verified booleans enter.
type CompletionEndpoint010 struct {
	mu       sync.Mutex
	registry *registry010.Gate
	clock    registry010.Clock
	replay   ReplayStore010
	did, kid string
	signing  ed25519.PrivateKey
	kem      []byte
	last     *registry010.Stamp
}

// NewCompletionEndpoint010 copies a local Ed25519 seed and optional X25519 private
// key. Exact correspondence with current registered public bytes is checked on use.
func NewCompletionEndpoint010(did, kid string, signingSeed, kem []byte, g *registry010.Gate, c registry010.Clock, r ReplayStore010) (*CompletionEndpoint010, error) {
	if !did010(did) || !key010(kid, did) || len(signingSeed) != 32 || (len(kem) != 0 && len(kem) != 32) || g == nil || c == nil || r == nil {
		return nil, errCompletion010
	}
	return &CompletionEndpoint010{registry: g, clock: c, replay: r, did: did, kid: kid, signing: ed25519.NewKeyFromSeed(signingSeed), kem: append([]byte(nil), kem...)}, nil
}
func (e *CompletionEndpoint010) sample() (registry010.Stamp, error) {
	if len(e.signing) != 64 {
		return registry010.Stamp{}, errCompletion010
	}
	t, x := e.clock.Now()
	if x != nil || t.MonoMS < 0 || t.Unix < 0 || t.Unix > 9007199254740691 {
		return t, errCompletion010
	}
	if e.last != nil && (t.MonoMS < e.last.MonoMS || t.Unix < e.last.Unix) {
		return t, errCompletion010
	}
	e.last = &t
	return t, nil
}
func pinnedLive010(now int64, a, b *registry010.Pinned) bool {
	keys := []registry010.Key{a.Signing(), b.Signing()}
	if k := b.KEM(); k != nil {
		keys = append(keys, *k)
	}
	for _, k := range keys {
		if k.Expires != nil && now >= *k.Expires {
			return false
		}
	}
	return true
}
func (e *CompletionEndpoint010) current(ctx context.Context, a, b *registry010.Pinned) error {
	if e.registry.CheckPinned(ctx, a) != nil || e.registry.CheckPinned(ctx, b) != nil {
		return errCompletion010
	}
	return nil
}
func (e *CompletionEndpoint010) selected(ctx context.Context, m map[string]string) (*registry010.Pinned, *registry010.Pinned, error) {
	a, x := e.registry.Select(ctx, m["initDid"], m["initKid"], false)
	if x != nil {
		return nil, nil, errCompletion010
	}
	b, x := e.registry.Select(ctx, m["respDid"], m["respKid"], true)
	if x != nil || b.KEM() == nil || b.DID()+"#"+b.KEM().Name != m["kemKid"] {
		return nil, nil, errCompletion010
	}
	local := a
	if e.did == m["respDid"] {
		local = b
	}
	pub, _ := hex.DecodeString(local.Signing().Material)
	if local.DID() != e.did || local.DID()+"#"+local.Signing().Name != e.kid || !bytes.Equal(pub, e.signing.Public().(ed25519.PublicKey)) {
		return nil, nil, errCompletion010
	}
	return a, b, nil
}
func rawObject010(raw []byte, names []string) (map[string]json.RawMessage, error) {
	if len(raw) > 32768 {
		return nil, errCompletion010
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	t, x := d.Token()
	if x != nil || t != json.Delim('{') {
		return nil, errCompletion010
	}
	m := map[string]json.RawMessage{}
	for d.More() {
		t, x = d.Token()
		if x != nil {
			return nil, errCompletion010
		}
		k, ok := t.(string)
		if !ok {
			return nil, errCompletion010
		}
		if _, ok = m[k]; ok {
			return nil, errCompletion010
		}
		var v json.RawMessage
		if d.Decode(&v) != nil || bytes.Equal(v, []byte("null")) {
			return nil, errCompletion010
		}
		m[k] = v
	}
	if t, x = d.Token(); x != nil || t != json.Delim('}') {
		return nil, errCompletion010
	}
	if _, x = d.Token(); x != io.EOF {
		return nil, errCompletion010
	}
	if len(m) != len(names) {
		return nil, errCompletion010
	}
	for _, n := range names {
		if _, ok := m[n]; !ok {
			return nil, errCompletion010
		}
	}
	return m, nil
}

var wireFields010 = []string{"version", "id", "did", "recipient", "kid", "created", "expires", "nonce", "encoding", "context_id", "role", "signature"}

func str010(m map[string]json.RawMessage, k string) string {
	var s string
	if json.Unmarshal(m[k], &s) != nil {
		return ""
	}
	for _, c := range s {
		if c > 127 {
			return ""
		}
	}
	return s
}
func int010(m map[string]json.RawMessage, k string) (int64, error) {
	return strconv.ParseInt(string(m[k]), 10, 64)
}
func canon010(v any) []byte { b, _ := jcs.Marshal(v); return b }
func sign010(m map[string]any, domain string, sk ed25519.PrivateKey) []byte {
	m["signature"] = base64.RawURLEncoding.EncodeToString(ed25519.Sign(sk, append([]byte(domain), canon010(m)...)))
	return canon010(m)
}
func wire010(raw []byte, response bool, now int64) (map[string]json.RawMessage, []byte, error) {
	names := append([]string{}, wireFields010...)
	field := "payload"
	role := "initiator"
	if response {
		field = "data"
		role = "responder"
		names = append(names, "message_id", "request_hash", "success")
	}
	names = append(names, field)
	m, x := rawObject010(raw, names)
	if x != nil {
		return nil, nil, x
	}
	if str010(m, "version") != "0.10.0" || str010(m, "encoding") != "plain" || str010(m, "role") != role || !uuid010.MatchString(str010(m, "id")) || !uuid010.MatchString(str010(m, "context_id")) || !did010(str010(m, "did")) || !did010(str010(m, "recipient")) || !key010(str010(m, "kid"), str010(m, "did")) {
		return nil, nil, errCompletion010
	}
	created, x := int010(m, "created")
	expires, y := int010(m, "expires")
	if x != nil || y != nil || created < 0 || expires > 9007199254740991 || expires <= created || expires-created > 300 || created > now+30 || now >= expires {
		return nil, nil, errCompletion010
	}
	for k, n := range map[string]int{"nonce": 16, "signature": 64} {
		if _, x = binary010(str010(m, k), n); x != nil {
			return nil, nil, errCompletion010
		}
	}
	if response {
		if string(m["success"]) != "true" || !uuid010.MatchString(str010(m, "message_id")) {
			return nil, nil, errCompletion010
		}
		if _, x = binary010(str010(m, "request_hash"), 32); x != nil {
			return nil, nil, errCompletion010
		}
	}
	body, x := base64.RawURLEncoding.Strict().DecodeString(str010(m, field))
	if x != nil || len(body) > 16384 || base64.RawURLEncoding.EncodeToString(body) != str010(m, field) {
		return nil, nil, errCompletion010
	}
	return m, body, nil
}
func verifyWire010(m map[string]json.RawMessage, response bool, k registry010.Key) error {
	sig, x := binary010(str010(m, "signature"), 64)
	pub, y := hex.DecodeString(k.Material)
	if x != nil || y != nil || len(pub) != 32 || k.Alg != "ed25519" {
		return errCompletion010
	}
	unsigned := map[string]json.RawMessage{}
	for n, v := range m {
		if n != "signature" {
			unsigned[n] = v
		}
	}
	domain := "sage-wire-request|0.10.0\n"
	if response {
		domain = "sage-wire-response|0.10.0\n"
	}
	if !ed25519.Verify(pub, append([]byte(domain), canon010(unsigned)...), sig) {
		return errCompletion010
	}
	return nil
}
func envelope010(did, recipient, kid, ctx, role string, payload []byte, now, expires int64) (map[string]any, error) {
	var nonce [16]byte
	if _, x := rand.Read(nonce[:]); x != nil {
		return nil, errCompletion010
	}
	id, x := uuid.NewRandom()
	if x != nil {
		return nil, errCompletion010
	}
	field := "payload"
	if role == "responder" {
		field = "data"
	}
	return map[string]any{"version": "0.10.0", "id": id.String(), "did": did, "recipient": recipient, "kid": kid, "context_id": ctx, "role": role, "created": now, "expires": expires, "nonce": base64.RawURLEncoding.EncodeToString(nonce[:]), "encoding": "plain", field: base64.RawURLEncoding.EncodeToString(payload)}, nil
}
func reservation010(m map[string]json.RawMessage) Replay010 {
	x, _ := int010(m, "expires")
	context := ""
	if str010(m, "role") == "initiator" {
		context = str010(m, "context_id")
	}
	return Replay010{str010(m, "did"), str010(m, "recipient"), str010(m, "id"), str010(m, "nonce"), context, x}
}

// PendingCompletion010 owns one emitted initiation. Close on abandonment; invalid
// completion consumes it. The return of Start is the local emission boundary.
type PendingCompletion010 struct {
	endpoint *CompletionEndpoint010
	state    *Initiator010
	request  []byte
	init     map[string]string
	a, b     *registry010.Pinned
	emitted  registry010.Stamp
	expires  int64
	closed   bool
}

// Start chooses the current KEM, generates fresh context/nonce/ephemerals and signs
// the exact request retained for completion binding. A delayed send requires a new
// handshake; this API's return is the local emission boundary.
func (e *CompletionEndpoint010) Start(ctx context.Context, recipient, respKid string, ttl int64) (*PendingCompletion010, []byte, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	start, x := e.sample()
	if x != nil || ttl < 1 || ttl > 300 || len(e.signing) != 64 {
		return nil, nil, errCompletion010
	}
	a, x := e.registry.Select(ctx, e.did, e.kid, false)
	if x != nil {
		return nil, nil, errCompletion010
	}
	b, x := e.registry.Select(ctx, recipient, respKid, true)
	if x != nil || b.KEM() == nil {
		return nil, nil, errCompletion010
	}
	pub, _ := hex.DecodeString(a.Signing().Material)
	if !bytes.Equal(pub, e.signing.Public().(ed25519.PublicKey)) {
		return nil, nil, errCompletion010
	}
	contextID, x := uuid.NewRandom()
	if x != nil {
		return nil, nil, errCompletion010
	}
	var nonce [16]byte
	if _, x = rand.Read(nonce[:]); x != nil {
		return nil, nil, errCompletion010
	}
	m := map[string]string{"v": "0.10.0", "ctx": contextID.String(), "initDid": e.did, "respDid": recipient, "initKid": e.kid, "respKid": respKid, "kemKid": recipient + "#" + b.KEM().Name, "suite": "hpke-base+x25519+hkdf-sha256", "combiner": "e2e-x25519-hkdf-v1", "nonce": base64.RawURLEncoding.EncodeToString(nonce[:])}
	kem, _ := hex.DecodeString(b.KEM().Material)
	state, init, x := StartInitiator010(canon010(m), kem)
	if x != nil {
		return nil, nil, errCompletion010
	}
	wire, x := envelope010(e.did, recipient, e.kid, m["ctx"], "initiator", init, start.Unix, start.Unix+ttl)
	if x != nil {
		state.Close()
		return nil, nil, x
	}
	wire["nonce"] = m["nonce"]
	request := sign010(wire, "sage-wire-request|0.10.0\n", e.signing)
	end, x := e.sample()
	if x != nil || end.MonoMS-start.MonoMS > 5000 || !pinnedLive010(end.Unix, a, b) || end.Unix >= start.Unix+ttl {
		state.Close()
		return nil, nil, errCompletion010
	}
	original, _, _ := initiation010(init)
	p := &PendingCompletion010{e, state, request, original, a, b, end, start.Unix + ttl, false}
	return p, append([]byte(nil), request...), nil
}
func (p *PendingCompletion010) destroy() {
	if p.state != nil {
		p.state.Close()
		p.state = nil
	}
	p.closed = true
}

// Close permanently destroys pending material.
func (p *PendingCompletion010) Close() {
	p.endpoint.mu.Lock()
	defer p.endpoint.mu.Unlock()
	p.destroy()
}
func pendingLive010(now, start registry010.Stamp, expires int64) bool {
	return now.MonoMS >= start.MonoMS && now.Unix >= start.Unix && now.MonoMS-start.MonoMS < 300000 && now.Unix < expires
}

// AuthenticatedCompletion010 owns a pinned authenticated transcript and secret
// record state. No public constructor or seed export exists. It is not a dispatch
// API. Responders remain RESPONSE_SENT until OpenRequest atomically confirms them.
type AuthenticatedCompletion010 struct {
	endpoint          *CompletionEndpoint010
	a, b              *registry010.Pinned
	tuple             map[string]string
	created           registry010.Stamp
	expires           int64
	initiator, closed bool
	confirmed         bool
	active            registry010.Stamp
	records           *session.RecordSession010
}

// Tuple returns a copy of public authenticated bindings, never a grant or secret.
func (s *AuthenticatedCompletion010) Tuple() map[string]string {
	s.endpoint.mu.Lock()
	defer s.endpoint.mu.Unlock()
	m := map[string]string{}
	for k, v := range s.tuple {
		m[k] = v
	}
	return m
}

// State reports local lifecycle; Check must run for a current operation.
func (s *AuthenticatedCompletion010) State() string {
	s.endpoint.mu.Lock()
	defer s.endpoint.mu.Unlock()
	if s.closed {
		return "CLOSED"
	}
	if s.initiator || s.confirmed {
		return "ESTABLISHED"
	}
	return "RESPONSE_SENT"
}
func (s *AuthenticatedCompletion010) destroy() {
	if s.records != nil {
		s.records.Close()
	}
	s.closed = true
}

// Close erases the owned seed and retires the result.
func (s *AuthenticatedCompletion010) Close() {
	s.endpoint.mu.Lock()
	defer s.endpoint.mu.Unlock()
	s.destroy()
}

// Check revalidates both pinned signing keys and the KEM, closing on failure.
// It does not confirm provisional state or permit record dispatch.
func (s *AuthenticatedCompletion010) Check(ctx context.Context) error {
	e := s.endpoint
	e.mu.Lock()
	defer e.mu.Unlock()
	start, x := e.sample()
	if x != nil || s.recordLive(start) != nil || e.current(ctx, s.a, s.b) != nil {
		s.destroy()
		return errCompletion010
	}
	end, x := e.sample()
	if x != nil || end.MonoMS-start.MonoMS > 5000 || !pinnedLive010(end.Unix, s.a, s.b) || s.recordLive(end) != nil {
		s.destroy()
		return errCompletion010
	}
	return nil
}
func owned010(e *CompletionEndpoint010, d *Derivation010, a, b *registry010.Pinned, now registry010.Stamp, expires int64, initiator bool) *AuthenticatedCompletion010 {
	var t map[string]string
	_ = json.Unmarshal(d.Transcript, &t)
	tuple := map[string]string{}
	for _, k := range []string{"v", "ctx", "initDid", "respDid", "initKid", "respKid", "kemKid", "suite", "combiner", "kid"} {
		tuple[k] = t[k]
	}
	tuple["th"] = base64.RawURLEncoding.EncodeToString(d.TH)
	tuple["sid"] = d.SID
	seed := d.Seed
	d.Seed = nil
	zeroBytes(d.AckTag)
	records, _ := session.NewRecordSession010(seed, d.TH, initiator)
	zeroBytes(seed)
	return &AuthenticatedCompletion010{endpoint: e, a: a, b: b, tuple: tuple, created: now, active: now, expires: expires, initiator: initiator, records: records}
}

// Complete verifies both signatures, exact request hash and echoed initiation,
// current pinned keys, ACK and both clocks before consuming pending state.
func (p *PendingCompletion010) Complete(ctx context.Context, response []byte) (*AuthenticatedCompletion010, error) {
	e := p.endpoint
	e.mu.Lock()
	defer e.mu.Unlock()
	if p.closed {
		return nil, errCompletion010
	}
	defer p.destroy()
	start, x := e.sample()
	if x != nil || !pendingLive010(start, p.emitted, p.expires) {
		return nil, errCompletion010
	}
	w, body, x := wire010(response, true, start.Unix)
	if x != nil {
		return nil, errCompletion010
	}
	request, _, _ := wire010(p.request, false, p.emitted.Unix)
	h := sha256.Sum256(canon010(request))
	if str010(w, "message_id") != str010(request, "id") || str010(w, "request_hash") != base64.RawURLEncoding.EncodeToString(h[:]) || str010(w, "did") != p.init["respDid"] || str010(w, "recipient") != p.init["initDid"] || str010(w, "kid") != p.init["respKid"] || str010(w, "context_id") != p.init["ctx"] || str010(w, "id") == str010(request, "id") || str010(w, "nonce") == str010(request, "nonce") {
		return nil, errCompletion010
	}
	c, x := rawObject010(body, []string{"v", "task", "transcript", "ackTagB64", "sigB64"})
	if x != nil || str010(c, "v") != "0.10.0" || str010(c, "task") != "hpke/complete@0.10.0" {
		return nil, errCompletion010
	}
	transcript, x := fields010(c["transcript"], append(append([]string{}, bindingFields010...), "task", "enc", "ephC", "ephS", "kid"))
	if x != nil {
		return nil, errCompletion010
	}
	for k, v := range p.init {
		if transcript[k] != v {
			return nil, errCompletion010
		}
	}
	if !bytes.Equal(body, canon010(c)) {
		return nil, errCompletion010
	}
	sig, x := binary010(str010(c, "sigB64"), 64)
	ack, y := binary010(str010(c, "ackTagB64"), 32)
	if x != nil || y != nil {
		return nil, errCompletion010
	}
	delete(c, "sigB64")
	if e.current(ctx, p.a, p.b) != nil || verifyWire010(w, true, p.b.Signing()) != nil {
		return nil, errCompletion010
	}
	pub, _ := hex.DecodeString(p.b.Signing().Material)
	if !ed25519.Verify(pub, append([]byte("sage-hpke-complete|0.10.0\n"), canon010(c)...), sig) {
		return nil, errCompletion010
	}
	d, x := p.state.Derive(c["transcript"])
	if x != nil {
		return nil, errCompletion010
	}
	defer func() { zeroBytes(d.Seed); zeroBytes(d.AckTag) }()
	if !VerifyAckTag010(d.Seed, d.TH, ack) {
		return nil, errCompletion010
	}
	end, x := e.sample()
	expires, _ := int010(w, "expires")
	if x != nil || end.MonoMS-start.MonoMS > 5000 || !pinnedLive010(end.Unix, p.a, p.b) || !pendingLive010(end, p.emitted, p.expires) || end.Unix >= expires {
		return nil, errCompletion010
	}
	if e.replay.Reserve(reservation010(w)) != nil {
		return nil, errCompletion010
	}
	// The trusted replay store may block. A consumed replay entry remains a denial
	// if the operation expires during durable commit; no session is created.
	end, x = e.sample()
	if x != nil || end.MonoMS-start.MonoMS > 5000 || !pinnedLive010(end.Unix, p.a, p.b) || !pendingLive010(end, p.emitted, p.expires) || end.Unix >= expires {
		return nil, errCompletion010
	}
	return owned010(e, d, p.a, p.b, end, 0, true), nil
}

// Respond authenticates the complete signed request and produces a separately
// signed completion payload inside an exact-request-bound signed response. The
// returned result is provisional and cannot send before first-record confirmation.
func (e *CompletionEndpoint010) Respond(ctx context.Context, request []byte, ttl int64) (*AuthenticatedCompletion010, []byte, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	start, x := e.sample()
	if x != nil || ttl < 1 || ttl > 300 || len(e.signing) != 64 || len(e.kem) != 32 {
		return nil, nil, errCompletion010
	}
	w, body, x := wire010(request, false, start.Unix)
	if x != nil {
		return nil, nil, errCompletion010
	}
	m, _, x := initiation010(body)
	if x != nil || !bytes.Equal(body, canon010(m)) || m["respDid"] != e.did || m["respKid"] != e.kid || str010(w, "did") != m["initDid"] || str010(w, "kid") != m["initKid"] || str010(w, "recipient") != m["respDid"] || str010(w, "context_id") != m["ctx"] || str010(w, "nonce") != m["nonce"] {
		return nil, nil, errCompletion010
	}
	a, b, x := e.selected(ctx, m)
	if x != nil || verifyWire010(w, false, a.Signing()) != nil {
		return nil, nil, errCompletion010
	}
	sk, x := ecdh.X25519().NewPrivateKey(e.kem)
	kem, _ := hex.DecodeString(b.KEM().Material)
	if x != nil || !bytes.Equal(sk.PublicKey().Bytes(), kem) {
		return nil, nil, errCompletion010
	}
	d, x := RespondFresh010(body, e.kem)
	if x != nil {
		return nil, nil, errCompletion010
	}
	defer func() { zeroBytes(d.Seed); zeroBytes(d.AckTag) }()
	completion := map[string]any{"v": "0.10.0", "task": "hpke/complete@0.10.0", "transcript": json.RawMessage(d.Transcript), "ackTagB64": base64.RawURLEncoding.EncodeToString(d.AckTag)}
	completion["sigB64"] = base64.RawURLEncoding.EncodeToString(ed25519.Sign(e.signing, append([]byte("sage-hpke-complete|0.10.0\n"), canon010(completion)...)))
	reqExpires, _ := int010(w, "expires")
	expires := reqExpires
	if start.Unix+ttl < expires {
		expires = start.Unix + ttl
	}
	response, x := envelope010(e.did, m["initDid"], e.kid, m["ctx"], "responder", canon010(completion), start.Unix, expires)
	if x != nil {
		return nil, nil, errCompletion010
	}
	if response["id"] == str010(w, "id") || response["nonce"] == str010(w, "nonce") {
		return nil, nil, errCompletion010
	}
	h := sha256.Sum256(canon010(w))
	response["message_id"] = str010(w, "id")
	response["request_hash"] = base64.RawURLEncoding.EncodeToString(h[:])
	response["success"] = true
	raw := sign010(response, "sage-wire-response|0.10.0\n", e.signing)
	end, x := e.sample()
	if x != nil || end.MonoMS-start.MonoMS > 5000 || !pinnedLive010(end.Unix, a, b) || end.Unix >= expires {
		return nil, nil, errCompletion010
	}
	if e.replay.Reserve(reservation010(w)) != nil {
		return nil, nil, errCompletion010
	}
	end, x = e.sample()
	if x != nil || end.MonoMS-start.MonoMS > 5000 || !pinnedLive010(end.Unix, a, b) || end.Unix >= expires {
		return nil, nil, errCompletion010
	}
	return owned010(e, d, a, b, end, expires, false), raw, nil
}

// Close retires the endpoint's local private-key copies. Previously returned
// objects must also be closed by their owner; no keys are restored on restart.
func (e *CompletionEndpoint010) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	zeroBytes(e.signing)
	zeroBytes(e.kem)
	e.signing = nil
	e.kem = nil
}

// State reports whether pending cryptographic material is still retained.
func (p *PendingCompletion010) State() string {
	p.endpoint.mu.Lock()
	defer p.endpoint.mu.Unlock()
	if p.closed {
		return "CLOSED"
	}
	return "INIT_SENT"
}
