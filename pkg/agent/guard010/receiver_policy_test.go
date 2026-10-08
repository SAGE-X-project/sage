package guard010_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/guard010"
)

// testMapping is receiver administration for one provisioned commitment. It
// knows the policy and manifest descriptors but never the original request.
type testMapping struct {
	issuer, digest   string
	policy, manifest []byte
	deny, retired    bool
	seenIssuer       string
	seenDigest       string
}

func (m *testMapping) Approved(ctx context.Context, issuer, digest string) ([]byte, []byte, error) {
	m.seenIssuer, m.seenDigest = issuer, digest
	if ctx.Err() != nil || m.retired || issuer != m.issuer || digest != m.digest {
		return nil, nil, errors.New("not provisioned")
	}
	return m.policy, m.manifest, nil
}

func (m *testMapping) Authorize(_ context.Context, issuer, tool string, args []byte) error {
	if m.deny || issuer != m.issuer || tool == "" || len(args) == 0 {
		return errors.New("denied")
	}
	return nil
}

func validIntent(t *testing.T) (guardFixture, []byte) {
	t.Helper()
	for _, c := range cases(t) {
		if c.ID == "intent-valid" {
			f := guardFixture{}
			if err := json.Unmarshal(c.Input, &f); err != nil {
				t.Fatal(err)
			}
			raw, err := hex.DecodeString(f.s("envelope_hex"))
			if err != nil {
				t.Fatal(err)
			}
			return f, raw
		}
	}
	t.Fatal("intent-valid vector missing")
	return nil, nil
}

func mappingFor(t *testing.T, f guardFixture) *testMapping {
	t.Helper()
	policy := []byte(f["approved_policy"])
	canonical, err := guard010.Canonicalize(policy)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := guard010.PolicyCommitment(canonical)
	if err != nil {
		t.Fatal(err)
	}
	return &testMapping{issuer: f.s("expected_issuer"), digest: digest, policy: canonical, manifest: []byte(f["approved_manifest"])}
}

func receiverPolicy(t *testing.T, m guard010.ReceiverMapping) *guard010.ReceiverPolicy {
	t.Helper()
	p, err := guard010.NewReceiverPolicy(m)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// A receiver accepts the signed intent from its provisioned (issuer,
// policy_digest) mapping without knowing the original request.
func TestReceiverMappingVerifiesWithoutOriginal(t *testing.T) {
	f, raw := validIntent(t)
	m := mappingFor(t, f)
	v, err := guard010.VerifyReceivedIntent(context.Background(), raw, f.s("expected_recipient"), f, receiverPolicy(t, m))
	if err != nil {
		t.Fatal(err)
	}
	if m.seenIssuer != f.s("expected_issuer") || m.seenDigest != m.digest || v.Digest() == "" {
		t.Fatal("mapping not consulted with the signed issuer and policy digest")
	}
	// The ordinary policy path is unchanged at a receiver.
	if _, err = guard010.VerifyReceivedIntent(context.Background(), raw, f.s("expected_recipient"), f, f); err != nil {
		t.Fatal(err)
	}
}

func TestReceiverMappingRefusals(t *testing.T) {
	f, raw := validIntent(t)
	recipient := f.s("expected_recipient")
	otherEpoch := func(m *testMapping) {
		var p map[string]any
		_ = json.Unmarshal(m.policy, &p)
		p["epoch"] = "00000000-0000-4000-8000-0000000000ff"
		b, _ := json.Marshal(p)
		m.policy, _ = guard010.Canonicalize(b)
	}
	for name, change := range map[string]func(*testMapping){
		"unknown-commitment": func(m *testMapping) { m.digest = "00" + m.digest[2:] },
		"retired":            func(m *testMapping) { m.retired = true },
		"other-issuer":       func(m *testMapping) { m.issuer = "did:sage:web:other.example:x" },
		"other-descriptor":   otherEpoch,
		"other-manifest":     func(m *testMapping) { m.manifest = []byte(`{"files":[],"version":"0.10.0"}`) },
		"denied-arguments":   func(m *testMapping) { m.deny = true },
	} {
		t.Run(name, func(t *testing.T) {
			m := mappingFor(t, f)
			change(m)
			if _, err := guard010.VerifyReceivedIntent(context.Background(), raw, recipient, f, receiverPolicy(t, m)); err == nil {
				t.Fatal("receiver accepted")
			}
		})
	}
	m := mappingFor(t, f)
	p := receiverPolicy(t, m)
	if _, err := guard010.VerifyReceivedIntent(context.Background(), raw, "did:sage:web:agent.example:other", f, p); err == nil {
		t.Fatal("wrong recipient accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := guard010.VerifyReceivedIntent(ctx, raw, recipient, f, p); err == nil {
		t.Fatal("cancelled verification accepted")
	}
	tampered := append([]byte(nil), raw...)
	tampered[len(tampered)/2] ^= 1
	if _, err := guard010.VerifyReceivedIntent(context.Background(), tampered, recipient, f, p); err == nil {
		t.Fatal("tampered intent accepted")
	}
	if _, err := guard010.VerifyReceivedIntent(context.Background(), raw, recipient, f, &guard010.ReceiverPolicy{}); err == nil {
		t.Fatal("empty receiver policy accepted")
	}
	var missing *testMapping
	for _, bad := range []guard010.ReceiverMapping{nil, missing} {
		if p, err := guard010.NewReceiverPolicy(bad); err == nil || p != nil {
			t.Fatal("absent mapping accepted")
		}
	}
}

// A receiver mapping never substitutes for the original commitment on paths
// that hold the original, and cannot act as an issuance or Client policy.
func TestReceiverPolicyIsReceiverOnly(t *testing.T) {
	f, raw := validIntent(t)
	p := receiverPolicy(t, mappingFor(t, f))
	if _, err := guard010.VerifyIntent(context.Background(), raw, f.s("expected_recipient"), f, p); err == nil {
		t.Fatal("VerifyIntent accepted a receiver mapping")
	}
	if original, policy, manifest, err := p.Bindings(context.Background(), f.s("expected_issuer"), "x"); err == nil || original != "" || policy != nil || manifest != nil {
		t.Fatal("receiver policy produced bindings")
	}
	var nilPolicy *guard010.ReceiverPolicy
	if nilPolicy.Authorize(context.Background(), "", "", nil) == nil {
		t.Fatal("nil receiver policy authorized")
	}
	// Strict comparison remains for an ordinary policy at a receiver.
	wrong := guardFixture{}
	for k, v := range f {
		wrong[k] = v
	}
	wrong["original_digest"] = json.RawMessage(`"` + "00" + f.s("original_digest")[2:] + `"`)
	if _, err := guard010.VerifyReceivedIntent(context.Background(), raw, f.s("expected_recipient"), f, wrong); err == nil {
		t.Fatal("ordinary receiver policy ignored the original commitment")
	}
}
