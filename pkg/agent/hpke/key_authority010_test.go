package hpke

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

// This Source is a prevalidated static fixture, not a Registry proof checker.
// Rotation-shaped version3 retains kem-1's immutable bytes in revoked history.
type authorityRead010 struct {
	Endpoint     string                `json:"endpoint"`
	RequestedDID string                `json:"requested_did"`
	Error        string                `json:"error,omitempty"`
	Snapshot     *registry010.Snapshot `json:"snapshot,omitempty"`
}
type authoritySource010 struct {
	control  *completionControl
	endpoint string
	rotated  bool
	reads    *[]authorityRead010
}

func (s authoritySource010) Read(ctx context.Context, did string) (registry010.Snapshot, error) {
	snapshot, err := s.control.Read(ctx, did)
	entry := authorityRead010{Endpoint: s.endpoint, RequestedDID: did}
	if err != nil {
		entry.Error = err.Error()
	} else {
		if s.rotated && did == completionBob {
			if len(snapshot.Keys) != 2 || snapshot.Keys[0].Name != "kem-1" {
				return registry010.Snapshot{}, errors.New("fixture layout")
			}
			snapshot.Keys[0].State = "revoked"
			key, e := ecdh.X25519().NewPrivateKey(bytes.Repeat([]byte{7}, 32))
			if e != nil {
				return registry010.Snapshot{}, e
			}
			fresh := registry010.Key{Name: "kem-2", Alg: "x25519", Material: hex.EncodeToString(key.PublicKey().Bytes()), State: "accepted"}
			snapshot.Keys = append(snapshot.Keys[:1:1], append([]registry010.Key{fresh}, snapshot.Keys[1:]...)...)
			snapshot.Version = "3"
		}
		// Keep another accepted signing key in every static observation. Revoking
		// the selected signing-1 is not revoking the last usable signing key.
		seed := byte(8)
		if did == completionBob {
			seed = 9
		}
		alternate := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{seed}, 32)).Public().(ed25519.PublicKey)
		snapshot.Keys = append(snapshot.Keys, registry010.Key{Name: "signing-2", Alg: "ed25519", Material: hex.EncodeToString(alternate), State: "accepted"})
		digest := sha256.Sum256(canon010(snapshot.Keys))
		snapshot.Digest = hex.EncodeToString(digest[:])

		copy := snapshot
		copy.Keys = append([]registry010.Key(nil), snapshot.Keys...)
		entry.Snapshot = &copy
	}
	*s.reads = append(*s.reads, entry)
	return snapshot, err
}
func authorityPair010(t *testing.T, rotated, matching bool) (*CompletionEndpoint010, *CompletionEndpoint010, *completionControl, *[]authorityRead010) {
	t.Helper()
	a, b, c := completionPair(t)
	trace := new([]authorityRead010)
	for name, e := range map[string]*CompletionEndpoint010{"client": a, "server": b} {
		journal, err := registry010.OpenJournal(filepath.Join(t.TempDir(), "authority"), true)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = journal.Close() })
		gate, err := registry010.NewGate(registry010.Config{Source: "fixture-authority", Registry: completionRegistry, Network: "local"}, authoritySource010{c, name, rotated, trace}, c, journal)
		if err != nil {
			t.Fatal(err)
		}
		e.registry = gate
	}
	if rotated && matching {
		zeroBytes(b.kem)
		b.kem = bytes.Repeat([]byte{7}, 32)
	}
	return a, b, c, trace
}
func authorityWitness010(t *testing.T, kind string, request []byte, trace []authorityRead010, accepted bool, replay int, extra ...map[string]any) {
	t.Helper()
	if len(request) == 0 || len(request) > 32768 {
		t.Fatal("unbounded public request")
	}
	digest := sha256.Sum256(request)
	witness := map[string]any{
		"schema": "sage-core-key-authority/1", "case": kind,
		"scope": "PREVALIDATED_STATIC_SOURCE_MEMORY_UNIT", "source_reads": trace,
		"request_sha256": hex.EncodeToString(digest[:]), "request_b64": base64.RawURLEncoding.EncodeToString(request), "session_returned": accepted,
		"server_replay_entries": replay, "peer_document_parser": "NOT_EXPOSED",
		"operating_registry": "NOT_BOUND", "deployed_custody": "NOT_BOUND",
	}
	for _, fields := range extra {
		for key, value := range fields {
			if _, exists := witness[key]; exists {
				t.Fatal("duplicate witness field")
			}
			witness[key] = value
		}
	}
	raw, err := json.Marshal(witness)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("handshake-key-authority-witness " + string(raw))
}
func authorityServerReads010(trace []authorityRead010) int {
	n := 0
	for _, r := range trace {
		if r.Endpoint == "server" {
			n++
		}
	}
	return n
}
func authorityPositive010(t *testing.T, rotated bool) {
	t.Helper()
	a, b, _, trace := authorityPair010(t, rotated, true)
	ctx := context.Background()
	pending, request, err := a.Start(ctx, completionBob, completionBob+"#signing-1", 300)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pending.Close)
	server, response, err := b.Respond(ctx, request, 300)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(server.Close)
	client, err := pending.Complete(ctx, response)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.Close)
	handshakeTrace := append([]authorityRead010(nil), (*trace)...)
	handshakeReplay := len(b.replay.(*completionReplay).seen)
	if handshakeReplay != 3 {
		t.Fatal("wrong handshake replay scope")
	}
	expectedKEM := completionBob + "#kem-1"
	if rotated {
		expectedKEM = completionBob + "#kem-2"
	}
	for _, session := range []*AuthenticatedCompletion010{client, server} {
		tuple := session.Tuple()
		if tuple["initDid"] != completionAlice || tuple["respDid"] != completionBob || tuple["initKid"] != completionAlice+"#signing-1" || tuple["respKid"] != completionBob+"#signing-1" || tuple["kemKid"] != expectedKEM {
			t.Fatalf("unexpected tuple %v", tuple)
		}
		if err := session.BindHTTP("https://agent.example/messages"); err != nil {
			t.Fatal(err)
		}
	}
	if !bytes.Equal(canon010(client.Tuple()), canon010(server.Tuple())) {
		t.Fatal("authenticated tuple mismatch")
	}
	payload := []byte("inert core authority request")
	message, err := client.SealHTTPRequest(ctx, payload, 300)
	if err != nil {
		t.Fatal(err)
	}
	got, err := server.OpenHTTPRequest(ctx, message)
	if err != nil || !bytes.Equal(got, payload) {
		t.Fatalf("normal authenticated request: %v", err)
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(message.Body, &object) != nil {
		t.Fatal("request JSON")
	}
	reply, err := server.SealHTTPResponse(ctx, str010(object, "id"), []byte("inert core authority reply"), true, "", 300, 200)
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.OpenHTTPResponse(ctx, reply)
	if err != nil || result == nil || !result.Success || result.MessageID != str010(object, "id") || !bytes.Equal(result.Data, []byte("inert core authority reply")) {
		t.Fatalf("normal authenticated response: %v", err)
	}
	if replayed, err := server.OpenHTTPRequest(ctx, message); err == nil || replayed != nil {
		t.Fatal("request replay accepted")
	}
	if replayed, err := client.OpenHTTPResponse(ctx, reply); err == nil || replayed != nil {
		t.Fatal("terminal response reused")
	}
	if authorityServerReads010(handshakeTrace) == 0 {
		t.Fatal("missing actual server Source read")
	}
	kind := "normal"
	if rotated {
		kind = "matched-kem-2"
	}
	authorityWitness010(t, kind, request, handshakeTrace, true, len(b.replay.(*completionReplay).seen), map[string]any{
		"completion_b64": base64.RawURLEncoding.EncodeToString(response), "tuple": client.Tuple(),
		"application_request_b64": base64.RawURLEncoding.EncodeToString(message.Body), "application_response_b64": base64.RawURLEncoding.EncodeToString(reply.Body),
		"handshake_server_replay_entries": handshakeReplay, "application_request_opened": 1, "application_response_opened": 1, "request_replay_refused": true, "response_reuse_refused": true,
	})
}

// Fixed adversarial bytes stay in this unit; there is no destination or CLI.
// A correctly signed closed-object violation is not a DID-document parse test.
func TestHandshakeKeyAuthorityBoundary010(t *testing.T) {
	t.Run("normal", func(t *testing.T) { authorityPositive010(t, false) })
	t.Run("matched-kem-2", func(t *testing.T) { authorityPositive010(t, true) })
	extras := []struct {
		name  string
		value any
	}{
		{"didDocument", map[string]any{"id": completionAlice, "verificationMethod": []any{}, "keyAgreement": []any{}}},
		{"alsoKnownAs", []string{"did:example:display"}},
		{"equivalentId", []string{completionBob}},
		{"canonicalId", "did:example:display"},
	}
	for _, site := range []string{"outer", "initiation"} {
		for _, extra := range extras {
			t.Run(site+"/"+extra.name, func(t *testing.T) {
				a, b, _, trace := authorityPair010(t, false, false)
				ctx := context.Background()
				pending, request, err := a.Start(ctx, completionBob, completionBob+"#signing-1", 300)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(pending.Close)
				var wire map[string]any
				if json.Unmarshal(request, &wire) != nil {
					t.Fatal("fixture wire")
				}
				if site == "outer" {
					wire[extra.name] = extra.value
				} else {
					body, err := base64.RawURLEncoding.DecodeString(wire["payload"].(string))
					if err != nil {
						t.Fatal(err)
					}
					var init map[string]any
					if json.Unmarshal(body, &init) != nil {
						t.Fatal("fixture initiation")
					}
					init[extra.name] = string(canon010(extra.value))
					wire["payload"] = base64.RawURLEncoding.EncodeToString(canon010(init))
				}
				delete(wire, "signature")
				changed := signKey010(wire, "sage-wire-request|0.10.0\n", ed25519.NewKeyFromSeed(bytes.Repeat([]byte{1}, 32)))
				var raw map[string]json.RawMessage
				if json.Unmarshal(changed, &raw) != nil {
					t.Fatal("signed fixture")
				}
				snapshot, err := a.registry.Select(ctx, completionAlice, completionAlice+"#signing-1", false)
				if err != nil || verifyWire010(raw, false, snapshot.Signing()) != nil {
					t.Fatal("signature calibration failed")
				}
				before := len(b.replay.(*completionReplay).seen)
				server, response, err := b.Respond(ctx, changed, 300)
				if err == nil || server != nil || response != nil {
					t.Fatal("closed-field violation completed")
				}
				if len(b.replay.(*completionReplay).seen) != before || authorityServerReads010(*trace) != 0 {
					t.Fatal("closed-field refusal reached Source or replay publication")
				}
				authorityWitness010(t, site+"/"+extra.name, changed, *trace, false, before)
			})
		}
	}
	for _, mode := range []string{"local-kem-mismatch", "revoke-init", "revoke-resp", "source-error", "local-signing-mismatch"} {
		t.Run(mode, func(t *testing.T) {
			a, b, c, trace := authorityPair010(t, mode == "local-kem-mismatch", false)
			ctx := context.Background()
			pending, request, err := a.Start(ctx, completionBob, completionBob+"#signing-1", 300)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(pending.Close)
			var outer map[string]json.RawMessage
			if json.Unmarshal(request, &outer) != nil {
				t.Fatal("fixture wire")
			}
			body, err := base64.RawURLEncoding.DecodeString(str010(outer, "payload"))
			if err != nil {
				t.Fatal(err)
			}
			init, _, err := initiation010(body)
			if err != nil {
				t.Fatal(err)
			}
			expectedKEM := completionBob + "#kem-1"
			if mode == "local-kem-mismatch" {
				expectedKEM = completionBob + "#kem-2"
			}
			if init["kemKid"] != expectedKEM {
				t.Fatal("wrong authoritative KEM selection")
			}
			if mode == "local-signing-mismatch" {
				zeroBytes(b.signing)
				b.signing = ed25519.NewKeyFromSeed(bytes.Repeat([]byte{9}, 32))
				b.public = append(ed25519.PublicKey(nil), b.signing.Public().(ed25519.PublicKey)...)
			} else if mode != "local-kem-mismatch" {
				c.mode = mode
			}
			before := len(b.replay.(*completionReplay).seen)
			server, response, err := b.Respond(ctx, request, 300)
			if err == nil || server != nil || response != nil || len(b.replay.(*completionReplay).seen) != before {
				t.Fatal("unavailable current authority completed")
			}
			if authorityServerReads010(*trace) == 0 {
				t.Fatal("no actual Source attempt")
			}
			authorityWitness010(t, mode, request, *trace, false, before)
		})
	}
}
