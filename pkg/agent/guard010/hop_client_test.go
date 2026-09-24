package guard010_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

type hopFixture struct {
	guardFixture
	parentAllowed bool
	parentChecks  int
	denyAt        int
	sends         int
}

func (f *hopFixture) Sample(context.Context) (int64, int64, error) {
	return 1700000000000, 0, nil
}
func (f *hopFixture) Commit(context.Context, string, []byte) error {
	f.sends++
	return nil
}
func (f *hopFixture) Authorized(_ context.Context, _ []byte) error {
	f.parentChecks++
	if !f.parentAllowed || f.parentChecks == f.denyAt {
		return g.ErrInvalid
	}
	return nil
}

func TestHopClientRejectsUnboundCaptureAndReusedIdentities(t *testing.T) {
	incoming, outgoing, parent, child := hopTestInput(t)
	services := g.ClientServices{IntentAuthority: child, Policy: child, ResultAuthority: child, Clock: child, Sender: child,
		ExpectedIssuer: child.s("expected_issuer"), ExpectedRecipient: child.s("expected_recipient")}
	hop := g.HopServices{Authority: parent, Policy: parent, Parent: parent}
	var envelope map[string]any
	if json.Unmarshal(incoming, &envelope) != nil {
		t.Fatal("parent")
	}
	parentIntent := envelope["intent"].(map[string]any)
	f := guardFixture{"envelope_hex": mustJSON(t, hex.EncodeToString(outgoing))}
	for _, tc := range []struct {
		name, field string
		value       any
	}{
		{"unbound capture", "original_digest", parentIntent["original_digest"]},
		{"reused request", "request_id", parentIntent["request_id"]},
		{"reused call", "call_id", parentIntent["call_id"]},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := resign(t, f, tc.field, tc.value, false)
			path := filepath.Join(t.TempDir(), "journal")
			if c, err := g.OpenHopClient(context.Background(), path, true, incoming, changed, services, hop); err == nil || c != nil {
				t.Fatal("invalid hop opened")
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("invalid hop created durable state")
			}
		})
	}
}

func TestHopClientBindsDeclaredParentIDToHopPath(t *testing.T) {
	incoming, outgoing, parent, child := hopTestInput(t)
	var upstream map[string]any
	if json.Unmarshal(incoming, &upstream) != nil {
		t.Fatal("parent envelope")
	}
	parentID := upstream["intent"].(map[string]any)["call_id"]
	f := guardFixture{"envelope_hex": mustJSON(t, hex.EncodeToString(outgoing))}
	linked := resign(t, f, "parent_call_id", parentID, false)
	services := g.ClientServices{IntentAuthority: child, Policy: child, ResultAuthority: child, Clock: child, Sender: child,
		ExpectedIssuer: child.s("expected_issuer"), ExpectedRecipient: child.s("expected_recipient")}
	hop := g.HopServices{Authority: parent, Policy: parent, Parent: parent}
	rootPath := filepath.Join(t.TempDir(), "root")
	if c, err := g.OpenClient(context.Background(), rootPath, true, linked, services); err == nil || c != nil {
		t.Fatal("declared hop opened through root client")
	}
	if _, err := os.Stat(rootPath); !os.IsNotExist(err) {
		t.Fatal("root denial created a journal")
	}
	wrong := resign(t, f, "parent_call_id", "00000000-0000-4000-8000-000000000099", false)
	wrongPath := filepath.Join(t.TempDir(), "wrong")
	if c, err := g.OpenHopClient(context.Background(), wrongPath, true, incoming, wrong, services, hop); err == nil || c != nil {
		t.Fatal("wrong declared parent opened a hop")
	}
	if _, err := os.Stat(wrongPath); !os.IsNotExist(err) {
		t.Fatal("wrong parent created a journal")
	}
	c, err := g.OpenHopClient(context.Background(), filepath.Join(t.TempDir(), "linked"), true, incoming, linked, services, hop)
	if err != nil {
		t.Fatal("matching parent ID was denied", err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestHopClientRechecksParentAfterJournalReservation(t *testing.T) {
	incoming, outgoing, parent, child := hopTestInput(t)
	parent.denyAt = 3 // Open, first Begin check, then post-journal check.
	services := g.ClientServices{IntentAuthority: child, Policy: child, ResultAuthority: child, Clock: child, Sender: child,
		ExpectedIssuer: child.s("expected_issuer"), ExpectedRecipient: child.s("expected_recipient")}
	c, err := g.OpenHopClient(context.Background(), filepath.Join(t.TempDir(), "journal"), true, incoming, outgoing, services,
		g.HopServices{Authority: parent, Policy: parent, Parent: parent})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	if _, err := c.Begin(context.Background(), "00000000-0000-4000-8000-000000000020"); err == nil || child.sends != 0 || parent.parentChecks != 3 {
		t.Fatal("parent loss after reservation reached transport")
	}
}

func hopTestInput(t *testing.T) ([]byte, []byte, *hopFixture, *hopFixture) {
	t.Helper()
	s := clientVectors(t)
	incoming, err := hex.DecodeString(s.Input.s("envelope_hex"))
	if err != nil {
		t.Fatal(err)
	}
	var env map[string]any
	if json.Unmarshal(incoming, &env) != nil {
		t.Fatal("incoming envelope")
	}
	parent := env["intent"].(map[string]any)
	child := map[string]any{}
	for k, v := range parent {
		child[k] = v
	}
	b := s.Input.s("expected_recipient")
	c := s.Input.s("expected_issuer")
	d, err := g.OriginalCommitment([][]byte{incoming})
	if err != nil {
		t.Fatal(err)
	}
	child["issuer"], child["recipient"], child["keyid"] = b, c, b+"#signing-1"
	child["request_id"] = "00000000-0000-4000-8000-000000000011"
	child["call_id"] = "00000000-0000-4000-8000-000000000012"
	child["original_digest"] = d
	child["nonce"] = base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{2}, 16))
	var policy map[string]any
	if json.Unmarshal(s.Input["approved_policy"], &policy) != nil {
		t.Fatal("policy")
	}
	policy["issuer"] = b
	policyRaw, _ := json.Marshal(policy)
	child["policy_digest"], err = g.PolicyCommitment(policyRaw)
	if err != nil {
		t.Fatal(err)
	}
	seed := sha256.Sum256([]byte("public Guard fixture issuer"))
	key := ed25519.NewKeyFromSeed(seed[:])
	makeChild := func() []byte {
		canonical, e := g.Canonicalize(mustJSON(t, child))
		if e != nil {
			t.Fatal(e)
		}
		return mustJSON(t, map[string]any{
			"intent": child,
			"proof":  base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, append([]byte("sage-execution-intent|0.10.0\x00"), canonical...))),
		})
	}
	upstream := &hopFixture{guardFixture: s.Input, parentAllowed: true}
	out := guardFixture{}
	for k, v := range s.Input {
		out[k] = append([]byte(nil), v...)
	}
	out["expected_issuer"] = mustJSON(t, b)
	out["expected_recipient"] = mustJSON(t, c)
	out["original_digest"] = mustJSON(t, d)
	out["approved_policy"] = policyRaw
	out["public_key_hex"] = mustJSON(t, hex.EncodeToString(key.Public().(ed25519.PublicKey)))
	return incoming, makeChild(), upstream, &hopFixture{guardFixture: out, parentAllowed: true}
}

func TestHopClientCaptureAndParentGate(t *testing.T) {
	incoming, outgoing, parent, child := hopTestInput(t)
	services := g.ClientServices{IntentAuthority: child, Policy: child, ResultAuthority: child, Clock: child, Sender: child,
		ExpectedIssuer: child.s("expected_issuer"), ExpectedRecipient: child.s("expected_recipient")}
	hop := g.HopServices{Authority: parent, Policy: parent, Parent: parent}
	path := filepath.Join(t.TempDir(), "journal")
	parent.parentAllowed = false
	if c, err := g.OpenHopClient(context.Background(), path, true, incoming, outgoing, services, hop); err == nil || c != nil {
		t.Fatal("unapproved parent opened downstream client")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("denied parent created durable state")
	}
	parent.parentAllowed = true
	c, err := g.OpenHopClient(context.Background(), path, true, incoming, outgoing, services, hop)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	parent.parentAllowed = false
	if _, err := c.Begin(context.Background(), "00000000-0000-4000-8000-000000000020"); err == nil || child.sends != 0 {
		t.Fatal("revoked parent reached transport")
	}
	parent.parentAllowed = true
	if _, err := c.Begin(context.Background(), "00000000-0000-4000-8000-000000000021"); err != nil || child.sends != 1 {
		t.Fatalf("authorized hop failed: %v", err)
	}
}
