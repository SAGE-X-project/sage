package hpke

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"golang.org/x/crypto/hkdf"
)

// The 0.11.0 selection manifest (sage-spec a22a275,
// profiles/sage-0.11.0/selection.json) fixes these bytes and values.
func TestProfile011MatchesSelection(t *testing.T) {
	for name, tc := range map[string]struct{ got, wantHex string }{
		"wire_request":  {profile011.wireRequest, "736167652d776972652d726571756573747c302e31312e300a"},
		"wire_response": {profile011.wireResponse, "736167652d776972652d726573706f6e73657c302e31312e300a"},
		"hpke_info":     {profile011.hpkeInfo, "736167652d68706b652d696e666f7c302e31312e300a"},
		"hpke_export":   {profile011.hpkeExport, "736167652d68706b652d6578706f72747c302e31312e300a"},
		"hpke_complete": {profile011.hpkeComplete, "736167652d68706b652d636f6d706c6574657c302e31312e300a"},
		"hpke_combiner": {profile011.combiner, "736167652d68706b652d636f6d62696e65727c302e31312e30"},
		"hpke_ack":      {profile011.ack, "736167652d68706b652d61636b7c302e31312e30"},
		"session":       {profile011.session, "736167652d73657373696f6e7c302e31312e30"},
	} {
		if hex.EncodeToString([]byte(tc.got)) != tc.wantHex {
			t.Errorf("%s is %q", name, tc.got)
		}
	}
	for name, tc := range map[string]struct{ got, want string }{
		"hpke_v": {profile011.version, "0.11.0"}, "hpke_init_task": {profile011.initTask, "hpke/init@0.11.0"},
		"hpke_complete_task": {profile011.completeTask, "hpke/complete@0.11.0"}, "http_signature_tag": {profile011.httpTag, "sage-0.11.0"},
	} {
		if tc.got != tc.want {
			t.Errorf("%s is %q", name, tc.got)
		}
	}
}

func decodeWire(t *testing.T, raw []byte) map[string]json.RawMessage {
	t.Helper()
	var w map[string]json.RawMessage
	if err := json.Unmarshal(raw, &w); err != nil {
		t.Fatal(err)
	}
	return w
}

// A complete 0.11.0 exchange: handshake, records both ways and a response.
func TestCompletion011RoundTrip(t *testing.T) {
	a, b, _ := completionPairFor(t, NewCompletionEndpoint011, NewCompletionEndpoint011)
	if a.Version() != "0.11.0" || b.Version() != "0.11.0" {
		t.Fatalf("versions %q %q", a.Version(), b.Version())
	}
	ctx := context.Background()
	p, q, err := a.Start(ctx, completionBob, completionBob+"#signing-1", 300)
	if err != nil {
		t.Fatal(err)
	}
	w := decodeWire(t, q)
	init, _ := base64.RawURLEncoding.DecodeString(str010(w, "payload"))
	if str010(w, "version") != "0.11.0" || !bytes.Contains(init, []byte(`"task":"hpke/init@0.11.0"`)) || !bytes.Contains(init, []byte(`"v":"0.11.0"`)) {
		t.Fatalf("initiation is not 0.11.0: %s / %s", q, init)
	}
	s, r, err := b.Respond(ctx, q, 300)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	completion, _ := base64.RawURLEncoding.DecodeString(str010(decodeWire(t, r), "data"))
	if !bytes.Contains(completion, []byte(`"task":"hpke/complete@0.11.0"`)) {
		t.Fatalf("completion is not 0.11.0: %s", completion)
	}
	i, err := p.Complete(ctx, r)
	if err != nil {
		t.Fatal(err)
	}
	defer i.Close()
	if i.Tuple()["sid"] != s.Tuple()["sid"] || i.Tuple()["sid"] == "" {
		t.Fatal("session identifiers differ")
	}
	req, err := i.SealRequest(ctx, []byte("pay 5 DUSD"), 300)
	if err != nil {
		t.Fatal(err)
	}
	if str010(decodeWire(t, req), "version") != "0.11.0" {
		t.Fatal("record is not 0.11.0")
	}
	got, err := s.OpenRequest(ctx, req)
	if err != nil || string(got) != "pay 5 DUSD" {
		t.Fatalf("open request: %q %v", got, err)
	}
	resp, err := s.SealResponse(ctx, str010(decodeWire(t, req), "id"), []byte("paid"), true, "", 300)
	if err != nil {
		t.Fatal(err)
	}
	out, err := i.OpenResponse(ctx, resp)
	if err != nil || string(out.Data) != "paid" {
		t.Fatalf("open response: %v", err)
	}
}

// A 0.11.0 endpoint never accepts 0.10.0 messages and the reverse; there is
// no fallback in either direction.
func TestProfilesNeverMix(t *testing.T) {
	ctx := context.Background()
	for name, pair := range map[string][2]endpointConstructor{
		"0.10.0 initiator to 0.11.0 responder": {NewCompletionEndpoint010, NewCompletionEndpoint011},
		"0.11.0 initiator to 0.10.0 responder": {NewCompletionEndpoint011, NewCompletionEndpoint010},
	} {
		t.Run(name, func(t *testing.T) {
			a, b, _ := completionPairFor(t, pair[0], pair[1])
			p, q, err := a.Start(ctx, completionBob, completionBob+"#signing-1", 300)
			if err != nil {
				t.Fatal(err)
			}
			defer p.Close()
			if _, _, err := b.Respond(ctx, q, 300); err == nil {
				t.Fatal("initiation of the other version accepted")
			}
		})
	}
	// A response produced by a 0.11.0 responder cannot complete a 0.10.0 pending.
	a10, b10, _ := completionPairFor(t, NewCompletionEndpoint010, NewCompletionEndpoint010)
	a11, b11, _ := completionPairFor(t, NewCompletionEndpoint011, NewCompletionEndpoint011)
	p10, _, err := a10.Start(ctx, completionBob, completionBob+"#signing-1", 300)
	if err != nil {
		t.Fatal(err)
	}
	_, q11, err := a11.Start(ctx, completionBob, completionBob+"#signing-1", 300)
	if err != nil {
		t.Fatal(err)
	}
	_, r11, err := b11.Respond(ctx, q11, 300)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p10.Complete(ctx, r11); err == nil {
		t.Fatal("0.11.0 completion accepted by a 0.10.0 pending")
	}
	_ = b10
}

// The derivation uses the 0.11.0 labels: recompute the session identifier and
// the ACK from the transcript with the selection's byte strings.
func TestDerivation011UsesTheSelectedLabels(t *testing.T) {
	kem, _ := ecdh.X25519().NewPrivateKey(bytes.Repeat([]byte{7}, 32))
	binding := map[string]string{"v": "0.11.0", "ctx": "3b1b2a8e-2f5c-4c4e-9c1a-8d3f6e7a9b10", "initDid": completionAlice, "respDid": completionBob,
		"initKid": completionAlice + "#signing-1", "respKid": completionBob + "#signing-1", "kemKid": completionBob + "#kem-1",
		"suite": "hpke-base+x25519+hkdf-sha256", "combiner": "e2e-x25519-hkdf-v1", "nonce": "AAAAAAAAAAAAAAAAAAAAAA"}
	raw := canon010(binding)
	dom, err := BuildDomains011(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(dom.Info, []byte("sage-hpke-info|0.11.0\n")) || !bytes.HasPrefix(dom.ExportContext, []byte("sage-hpke-export|0.11.0\n")) {
		t.Fatal("domains are not 0.11.0")
	}
	if _, err := BuildDomains010(raw); err == nil {
		t.Fatal("0.11.0 binding accepted as 0.10.0")
	}
	initiator, init, err := StartInitiator011(raw, kem.PublicKey().Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RespondFresh010(init, kem.Bytes()); err == nil {
		t.Fatal("0.11.0 initiation derived as 0.10.0")
	}
	responder, err := RespondFresh011(init, kem.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	derived, err := initiator.Derive(responder.Transcript)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(derived.Seed, responder.Seed) || !bytes.Equal(derived.AckTag, responder.AckTag) {
		t.Fatal("initiator and responder disagree")
	}
	if !strings.Contains(string(responder.Transcript), `"task":"hpke/init@0.11.0"`) {
		t.Fatal("transcript is not 0.11.0")
	}
	th := sha256.Sum256(responder.Transcript)
	sid := sha256.Sum256(append([]byte("sage-session|0.11.0"), th[:]...))
	if responder.SID != base64.RawURLEncoding.EncodeToString(sid[:16]) {
		t.Fatal("session identifier does not use sage-session|0.11.0")
	}
	ackKey := make([]byte, 32)
	if _, err := io.ReadFull(hkdf.Expand(sha256.New, responder.Seed, append([]byte("sage-hpke-ack|0.11.0"), th[:]...)), ackKey); err != nil {
		t.Fatal(err)
	}
	mac := hmac.New(sha256.New, ackKey)
	mac.Write(th[:])
	if !bytes.Equal(mac.Sum(nil), responder.AckTag) || !VerifyAckTag011(responder.Seed, th[:], responder.AckTag) || VerifyAckTag010(responder.Seed, th[:], responder.AckTag) {
		t.Fatal("ACK does not use sage-hpke-ack|0.11.0")
	}
	if _, err := DeriveResponder011(init, kem.Bytes(), bytes.Repeat([]byte{9}, 32), "3b1b2a8e-2f5c-4c4e-9c1a-8d3f6e7a9b11"); err != nil {
		t.Fatal(err)
	}
	seed, err := CombineSecrets011(bytes.Repeat([]byte{1}, 32), bytes.Repeat([]byte{2}, 32), th[:])
	other, _ := CombineSecrets010(bytes.Repeat([]byte{1}, 32), bytes.Repeat([]byte{2}, 32), th[:])
	if err != nil || bytes.Equal(seed, other) {
		t.Fatal("combiner label not separated")
	}
	if tag, err := MakeAckTag011(seed, th[:]); err != nil || len(tag) != 32 {
		t.Fatal("ACK tag")
	}
}

// The HTTP binding carries X-Sage-Version 0.11.0 and tag sage-0.11.0, and the
// framing of one version refuses messages of the other.
func TestHTTPHandshake011(t *testing.T) {
	a, b, _ := completionPairFor(t, NewCompletionEndpoint011, NewCompletionEndpoint011)
	ctx := context.Background()
	if a.BindHTTP(handshakeTarget010) != nil || b.BindHTTP(handshakeTarget010) != nil {
		t.Fatal("bind")
	}
	p, q, err := a.StartHTTP(ctx, completionBob, completionBob+"#signing-1", 300)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	wire, err := EncodeHTTP011(q, handshakeTarget010)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(wire, []byte("X-Sage-Version: 0.11.0")) && !bytes.Contains(wire, []byte("x-sage-version: 0.11.0")) || !bytes.Contains(wire, []byte(`tag="sage-0.11.0"`)) {
		t.Fatalf("HTTP request is not 0.11.0:\n%s", wire)
	}
	if _, err := ParseHTTP010(wire, handshakeTarget010, false); err == nil {
		t.Fatal("0.11.0 HTTP message parsed as 0.10.0")
	}
	if _, err := EncodeHTTP010(q, handshakeTarget010); err == nil {
		t.Fatal("0.11.0 HTTP message encoded as 0.10.0")
	}
	parsed, err := ParseHTTP011(wire, handshakeTarget010, false)
	if err != nil {
		t.Fatal(err)
	}
	s, r, err := b.RespondHTTP(ctx, parsed, 300)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	i, err := p.CompleteHTTP(ctx, r)
	if err != nil {
		t.Fatal(err)
	}
	defer i.Close()
	m, err := i.SealHTTPRequest(ctx, []byte("over HTTP"), 300)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := s.OpenHTTPRequest(ctx, m); err != nil || string(got) != "over HTTP" {
		t.Fatalf("open HTTP request: %v", err)
	}
}

func TestEndpoint011Constructors(t *testing.T) {
	var nilEndpoint *CompletionEndpoint010
	if nilEndpoint.Version() != "" {
		t.Fatal("nil endpoint has a version")
	}
	if _, err := NewCompletionEndpoint011("bad", "bad", nil, nil, nil, nil, nil); err == nil {
		t.Fatal("invalid configuration accepted")
	}
	if _, err := NewCustodyCompletionEndpoint011(context.Background(), "bad", "bad", nil, nil, nil, nil, nil); err == nil {
		t.Fatal("invalid custody configuration accepted")
	}
	if _, err := NewProtectedCompletionEndpoint011(context.Background(), "bad", "bad", nil, nil, nil, nil, nil); err == nil {
		t.Fatal("invalid protected configuration accepted")
	}
}
