// SAGE - Secure Agent Guarantee Engine
// Copyright (C) 2025 SAGE-X-project
//
// This file is part of SAGE.
//
// SAGE is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// SAGE is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with SAGE. If not, see <https://www.gnu.org/licenses/>.

package suites

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/core/clock"
	"github.com/sage-x-project/sage/core/did"
	"github.com/sage-x-project/sage/core/httpsig"
	"github.com/sage-x-project/sage/core/obs"
	"github.com/sage-x-project/sage/core/vectors"
)

func publishedRFC9421(t *testing.T) (string, map[string]map[string]any) {
	t.Helper()
	data, err := os.ReadFile(published + "/rfc9421.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Description string           `json:"description"`
		Vectors     []map[string]any `json:"vectors"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatal(err)
	}
	byName := map[string]map[string]any{}
	for _, v := range f.Vectors {
		byName[v["name"].(string)] = v
	}
	return f.Description, byName
}

// TestRFC9421ReproducesPublished regenerates the four published rfc9421
// vectors with httpsig.Signer and requires values identical to the
// published file; the randomised P-256 signature is taken from it and
// verified.
func TestRFC9421ReproducesPublished(t *testing.T) {
	s := RFC9421()
	s.Cases = rfc9421Published()
	files, err := runner(t, s, publishedGenerator).Generate(os.DirFS(published))
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Description string           `json:"description"`
		Vectors     []map[string]any `json:"vectors"`
	}
	if err := json.Unmarshal(files["rfc9421.json"], &got); err != nil {
		t.Fatal(err)
	}
	desc, want := publishedRFC9421(t)
	if got.Description != desc || len(got.Vectors) != len(want) {
		t.Fatalf("description or vector count differs from the published file")
	}
	for _, v := range got.Vectors {
		if !reflect.DeepEqual(v, want[v["name"].(string)]) {
			t.Errorf("%s differs from the published vector:\n got %v\nwant %v", v["name"], v, want[v["name"].(string)])
		}
	}
}

// TestRFC9421SuiteBuilder: the suite's own signature base builder, which
// signs the reject vectors, reproduces the published request and response
// bases and signatures, independently of httpsig.
func TestRFC9421SuiteBuilder(t *testing.T) {
	_, pub := publishedRFC9421(t)
	req, resp := baseRequest(), baseResponse()
	for _, c := range []struct{ name, got, want string }{
		{"request input", req.member(), reqSignatureInput},
		{"request signature", req.sign(), reqSignature},
		{"request base", req.base(nil), pub["request-ed25519"]["output"].(map[string]any)["signature_base"].(string)},
		{"response input", resp.member(), pub["response-ed25519"]["output"].(map[string]any)["signature_input"].(string)},
		{"response signature", resp.sign(), respSignature},
		{"response base", resp.signFor.base(&resp), pub["response-ed25519"]["output"].(map[string]any)["signature_base"].(string)},
	} {
		if c.got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, c.got, c.want)
		}
	}
}

// TestRFC9421BaseMessagesPass: request-ed25519 and response-ed25519 as
// received pass every step the reject vectors run, so each reject vector
// fails only through its mutation.
func TestRFC9421BaseMessagesPass(t *testing.T) {
	if err := rejectHTTPRequest(baseRequest().received()); err != nil {
		t.Fatalf("base request: %v", err)
	}
	if err := rejectHTTPResponse(baseResponse().received()); err != nil {
		t.Fatalf("base response: %v", err)
	}
}

// TestRFC9421RejectsAreResigned: every reject vector that changes the
// signed content carries a signature over its mutated base, so a verifier
// without the rule under test would accept it.
func TestRFC9421RejectsAreResigned(t *testing.T) {
	for _, c := range rfc9421Rejects() {
		if c.Mode != vectors.ModeReject || !strings.Contains(c.Description, "re-signed") {
			continue
		}
		fields := c.Input["fields"].([]any)
		var input, sig string
		for _, f := range fields {
			pair := f.([]any)
			switch pair[0] {
			case "signature-input":
				input = pair[1].(string)
			case "signature":
				sig = pair[1].(string)
			}
		}
		if input == "" || sig == "" || sig == reqSignature {
			t.Errorf("%s: not re-signed", c.Name)
		}
	}
}

func TestRFC9421InputErrors(t *testing.T) {
	for _, in := range []map[string]any{
		{"fields": "x"},
		{"fields": []any{"x"}},
		{"fields": []any{[]any{"Content-Type", "a"}}},
		{"fields": []any{[]any{"a", 1}}},
		{"fields": []any{}, "method": 1},
	} {
		if err := rejectHTTPRequest(in); err == nil || isRejection(err) {
			t.Errorf("%v: %v", in, err)
		}
	}
	if err := rejectHTTPResponse(map[string]any{}); err == nil || isRejection(err) {
		t.Errorf("response without request: %v", err)
	}
}

// TestRFC9421TimingAndReplayRejectsPassEarlierStages: each vector rejected
// at timing (step 5) or replay (step 8) passes steps 1 to 4, 6 and 7
// unchanged, so the timing or replay rule is the only reason it is
// rejected.
func TestRFC9421TimingAndReplayRejectsPassEarlierStages(t *testing.T) {
	n := 0
	for _, c := range rfc9421Rejects() {
		if st := obs.StageOf(c.Reason); st != obs.StageTiming && st != obs.StageReplay {
			continue
		}
		n++
		r, err := receivedRequestOf(c.Input)
		if err != nil {
			t.Fatal(err)
		}
		if err := passesOtherSteps(r, c.Input); err != nil {
			t.Errorf("%s: %v", c.Name, err)
		}
		if err := rejectHTTPRequest(c.Input); !isReason(err, c.Reason) {
			t.Errorf("%s: %v, want %s", c.Name, err, c.Reason)
		}
		if _, ok := c.Input["prior"]; ok {
			// Without the prior request the same input is accepted, so
			// the replay of prior is the reason, not the restore window.
			alone := map[string]any{}
			for k, v := range c.Input {
				alone[k] = v
			}
			delete(alone, "prior")
			if err := rejectHTTPRequest(alone); err != nil {
				t.Errorf("%s without prior: %v", c.Name, err)
			}
		}
	}
	if n != 5 {
		t.Fatalf("%d timing and replay vectors, want 5", n)
	}
}

// TestRFC9421PriorMustBeAccepted: a prior request that is rejected fails
// the vector with an error that is not a rejection.
func TestRFC9421PriorMustBeAccepted(t *testing.T) {
	in := baseRequest().received()
	bad := baseRequest()
	bad.input, bad.signature = bad.member(), bad.sign()
	bad.method = "PUT" // the signature no longer verifies
	in["prior"] = []any{bad.received()}
	if err := rejectHTTPRequest(in); err == nil || isRejection(err) {
		t.Fatalf("rejected prior: %v", err)
	}
	in["prior"] = "x"
	if err := rejectHTTPRequest(in); err == nil || isRejection(err) {
		t.Fatalf("prior not a list: %v", err)
	}
	delete(in, "prior")
	in["verify_at"] = "now"
	if err := rejectHTTPRequest(in); err == nil || isRejection(err) {
		t.Fatalf("verify_at not an integer: %v", err)
	}
}

func isReason(err error, want obs.Reason) bool {
	var r *core.Rejection
	return errors.As(err, &r) && r.Reason() == want
}

// passesOtherSteps runs steps 1, 2, 3, 4, 6 and 7 of 03 §5, without timing
// and replay, on a received request.
func passesOtherSteps(r httpsig.Request, in map[string]any) error {
	p, err := httpsig.Parse(r.Header, "", core.DefaultLimits())
	if err != nil {
		return err
	}
	base, err := httpsig.RequestBase(p, r)
	if err != nil {
		return err
	}
	ref, err := httpsig.CheckSigner(p, r.Header, httpsig.Expect{})
	if err != nil {
		return err
	}
	label, _ := in["key_label"].(string)
	cs, err := labelSigner(label)
	if err != nil {
		return err
	}
	at := time.Unix(reqCreated, 0)
	c := clock.NewFixed(at)
	s, err := core.NewSettings(core.WithSpecVersion(string(core.V0_9_0_Draft1_1)))
	if err != nil {
		return err
	}
	agent, err := did.ResolveAgent(context.Background(), labelResolver{signer: cs, clock: c}, ref.DID(), did.Policy{Clock: c, Settings: s})
	if err != nil {
		return err
	}
	if _, err := httpsig.VerifySignature(base, agent); err != nil {
		return err
	}
	return httpsig.CheckDigest(r.Header, r.Body, core.DefaultLimits())
}

// TestRFC9421W014RejectsOnlyForTheirRule: each W-014 reject vector is its
// base request with one mutation. The base requests are accepted, and the
// es256k vectors carry exactly the r and s of the accepted request-es256k
// signature, so the mutation is the only reason for each rejection.
func TestRFC9421W014RejectsOnlyForTheirRule(t *testing.T) {
	es := baseRequest()
	es.es256kRequest(nil)
	if err := rejectHTTPRequest(es.received()); err != nil {
		t.Fatalf("request-es256k as built: %v", err)
	}
	if err := rejectHTTPRequest(baseRequest().received()); err != nil {
		t.Fatalf("request-ed25519 as built: %v", err)
	}
	sigOf := func(in map[string]any) []byte {
		t.Helper()
		r, err := receivedRequestOf(in)
		if err != nil {
			t.Fatal(err)
		}
		p, err := httpsig.Parse(r.Header, "", core.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		return p.Signature()
	}
	good := sigOf(es.received())
	byName := map[string]map[string]any{}
	for _, c := range rfc9421Rejects() {
		byName[c.Name] = c.Input
	}
	if got := sigOf(byName["es256k-64-byte"]); !bytes.Equal(got, good[:64]) {
		t.Fatal("es256k-64-byte is not the accepted signature without v")
	}
	if got := sigOf(byName["es256k-der"]); !bytes.Equal(got, derOf(good)) {
		t.Fatal("es256k-der is not the accepted signature in DER")
	}
	for _, n := range []string{"keyid-bad-did", "keyid-bad-fragment", "x-sage-did-mismatch", "digest-mismatch",
		"digest-two-members", "digest-sha512-only", "es256k-64-byte", "es256k-der", "alg-key-mismatch"} {
		if byName[n] == nil {
			t.Errorf("missing %s", n)
		}
	}
	// digest-mismatch with the original body is the accepted base request.
	in := map[string]any{}
	for k, v := range byName["digest-mismatch"] {
		in[k] = v
	}
	in["body"] = reqBody
	if err := rejectHTTPRequest(in); err != nil {
		t.Fatalf("digest-mismatch with its body restored: %v", err)
	}
}

// TestRFC9421KeepsNonceNeedsARejection: digest-mismatch-keeps-nonce fails
// when its first request is accepted (it would then consume the nonce).
func TestRFC9421KeepsNonceNeedsARejection(t *testing.T) {
	c := rfc9421KeepsNonce()
	out, err := c.Produce(c.Input)
	if err != nil || out["accepted"] != true {
		t.Fatalf("%v %v", out, err)
	}
	in := baseRequest().received()
	first := baseRequest().received()
	delete(first, "verify_at")
	in["rejected_first"] = []any{first}
	if _, err := c.Produce(in); err == nil || isRejection(err) {
		t.Fatalf("an accepted first request: %v", err)
	}
}
