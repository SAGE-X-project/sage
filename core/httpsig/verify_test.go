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

package httpsig

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/core/clock"
	"github.com/sage-x-project/sage/core/crypto"
	"github.com/sage-x-project/sage/core/did"
	"github.com/sage-x-project/sage/core/obs"
	"github.com/sage-x-project/sage/core/replay"
)

const (
	respDID = "did:sage:ethereum:0x2222222222222222222222222222222222222222"
	tVerify = int64(1788609600)
)

func settings(t *testing.T) core.Settings {
	t.Helper()
	s, err := core.NewSettings(core.WithSpecVersion(string(core.V0_9_0_Draft1_1)))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// recordFor returns an active record of d listing cs's key with a valid
// proof of possession.
func recordFor(t *testing.T, cs crypto.Signer, d string) did.Record {
	t.Helper()
	parsed, err := did.Parse(d)
	if err != nil {
		t.Fatal(err)
	}
	data := cs.Public().Bytes()
	proof, err := crypto.SignPoP(context.Background(), cs, d, data)
	if err != nil {
		t.Fatal(err)
	}
	typ := did.KeyTypeECDSA
	if cs.Public().Alg() == crypto.AlgEd25519 {
		typ = did.KeyTypeEd25519
	}
	return did.Record{DID: parsed, IsActive: true, Keys: []did.Key{{Type: typ, KeyData: data, Signature: proof.Bytes(), Verified: true}}}
}

// agentFor returns the agent of d whose signing key is cs's.
func agentFor(t *testing.T, cs crypto.Signer, d string) did.Agent {
	t.Helper()
	now := time.Unix(tVerify, 0)
	rec := recordFor(t, cs, d)
	a, err := did.Evaluate(context.Background(), rec.DID, rec, did.Observation{At: now},
		did.Policy{Clock: clock.NewFixed(now), Settings: settings(t)})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// mapResolver resolves the records it holds, observed at its clock's now.
type mapResolver struct {
	mu    sync.Mutex
	clock clock.Clock
	recs  map[string]did.Record
	err   error
	calls int
}

func (r *mapResolver) Resolve(_ context.Context, d did.DID) (did.Record, did.Observation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	if r.err != nil {
		return did.Record{}, did.Observation{}, r.err
	}
	rec, ok := r.recs[d.String()]
	if !ok {
		return did.Record{}, did.Observation{}, errors.New("unknown agent")
	}
	now, _ := r.clock.Now()
	return rec, did.Observation{At: now, Source: "test"}, nil
}

func (r *mapResolver) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

type fixture struct {
	v     *Verifier
	clock *clock.Fixed
	res   *mapResolver
}

// newFixture returns a verifier at tVerify whose replay store was restored
// an hour earlier and whose resolver knows recs.
func newFixture(t *testing.T, recs ...did.Record) fixture {
	t.Helper()
	now := time.Unix(tVerify, 0)
	c := clock.NewFixed(now.Add(-time.Hour))
	store, err := replay.NewMemStore(c, replay.Capacity{PerOwner: 16, PerScope: 64, Total: 64})
	if err != nil {
		t.Fatal(err)
	}
	c.Set(now)
	res := &mapResolver{clock: c, recs: map[string]did.Record{}}
	for _, r := range recs {
		res.recs[r.DID.String()] = r
	}
	v, err := NewVerifier(c, settings(t), store, res)
	if err != nil {
		t.Fatal(err)
	}
	return fixture{v: v, clock: c, res: res}
}

func nonce(i int) string { return "n-0123456789abcd" + string(rune('a'+i)) }

// signedAt signs testRequest (keyid vecDID#key-1) at tVerify.
func signedAt(t *testing.T, s *Signer, n string) Request {
	t.Helper()
	return signed(t, s, testRequest(), Params{Created: tVerify, Nonce: n})
}

func TestNewVerifier(t *testing.T) {
	c := clock.NewFixed(time.Unix(tVerify, 0))
	store, err := replay.NewMemStore(c, replay.Capacity{PerOwner: 1, PerScope: 1, Total: 1})
	if err != nil {
		t.Fatal(err)
	}
	res := &mapResolver{clock: c}
	s := settings(t)
	for name, mk := range map[string]func() (*Verifier, error){
		"no clock":    func() (*Verifier, error) { return NewVerifier(nil, s, store, res) },
		"no store":    func() (*Verifier, error) { return NewVerifier(c, s, nil, res) },
		"no resolver": func() (*Verifier, error) { return NewVerifier(c, s, store, nil) },
		"no settings": func() (*Verifier, error) { return NewVerifier(c, core.Settings{}, store, res) },
	} {
		if v, err := mk(); v != nil || !errors.Is(err, ErrVerifier) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	var zero *Verifier
	if _, err := zero.VerifyRequest(context.Background(), Request{}, Expect{}); !errors.Is(err, ErrVerifier) {
		t.Fatal(err)
	}
	if _, err := (&Verifier{}).VerifyResponse(context.Background(), Response{}, Request{}, Expect{}); !errors.Is(err, ErrVerifier) {
		t.Fatal(err)
	}
}

func TestVerifyRequest(t *testing.T) {
	s, cs := testSigner(t, 1, vecDID+"#key-1")
	f := newFixture(t, recordFor(t, cs, vecDID))
	vr, err := f.v.VerifyRequest(context.Background(), signedAt(t, s, nonce(0)), Expect{})
	if err != nil {
		t.Fatal(err)
	}
	if !vr.Valid() || !vr.Signer().Equal(mustDID(t, vecDID)) || vr.KeyID().AsSent() != vecDID+"#key-1" ||
		!vr.Key().Equal(cs.Public()) || vr.Created().Unix() != tVerify || vr.Label() != Label {
		t.Fatal("verified request")
	}
	var zero VerifiedRequest
	if zero.Valid() || zero.Key() != nil || !zero.Signer().IsZero() || !zero.Created().IsZero() {
		t.Fatal("zero VerifiedRequest")
	}
	// The same request again is a replay (step 8).
	_, err = f.v.VerifyRequest(context.Background(), signedAt(t, s, nonce(0)), Expect{})
	wantReason(t, err, obs.ReasonReplay)
}

func mustDID(t *testing.T, s string) did.DID {
	t.Helper()
	d, err := did.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// TestFailedStepsKeepTheNonce: a request that fails step 3, 4, 5, 6 or 7
// does not consume its nonce: the genuine request with the same nonce is
// accepted afterwards (03 §5 step 8, E03-18). digest-mismatch-keeps-nonce is
// the step 7 case (SN-6, D30).
func TestFailedStepsKeepTheNonce(t *testing.T) {
	s, cs := testSigner(t, 1, vecDID+"#key-1")
	other, _ := testSigner(t, 2, vecDID+"#key-1")
	for i, c := range []struct {
		name   string
		mutate func(f fixture, r *Request, n string) Expect
		want   obs.Reason
	}{
		{"expected DID (step 3)", func(_ fixture, _ *Request, _ string) Expect { return Expect{DID: mustDID(t, respDID)} }, obs.ReasonPeerMismatch},
		{"resolution (step 4)", func(f fixture, _ *Request, _ string) Expect {
			f.res.err = errors.New("rpc down")
			return Expect{}
		}, obs.ReasonResolution},
		{"timing (step 5)", func(f fixture, _ *Request, _ string) Expect {
			f.clock.Set(time.Unix(tVerify+301, 0))
			return Expect{}
		}, obs.ReasonStale},
		{"signature (step 6)", func(_ fixture, r *Request, n string) Expect {
			*r = signedAt(t, other, n)
			return Expect{}
		}, obs.ReasonSignature},
		{"digest (step 7): digest-mismatch-keeps-nonce", func(_ fixture, r *Request, _ string) Expect {
			r.Body = []byte(`{"a":2}`)
			return Expect{}
		}, obs.ReasonDigest},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t, recordFor(t, cs, vecDID))
			n := nonce(i)
			bad := signedAt(t, s, n)
			bad.Header = bad.Header.Clone()
			e := c.mutate(f, &bad, n)
			_, err := f.v.VerifyRequest(context.Background(), bad, e)
			wantReason(t, err, c.want)
			f.res.err = nil
			f.clock.Set(time.Unix(tVerify, 0))
			if _, err := f.v.VerifyRequest(context.Background(), signedAt(t, s, n), Expect{}); err != nil {
				t.Fatalf("the genuine request after the failure: %v", err)
			}
		})
	}
}

// TestVerifyOrder: steps 1 to 3 run before any resolution, and resolution
// (step 4) before timing (step 5), as 03 §5 lists them (D35).
func TestVerifyOrder(t *testing.T) {
	s, cs := testSigner(t, 1, vecDID+"#key-1")
	f := newFixture(t, recordFor(t, cs, vecDID))
	ctx := context.Background()
	r := signedAt(t, s, nonce(0))
	_, err := f.v.VerifyRequest(ctx, r, Expect{DID: mustDID(t, respDID)})
	wantReason(t, err, obs.ReasonPeerMismatch)
	bad := r
	bad.Header = r.Header.Clone()
	bad.Header.Set(FieldSAGEDID, respDID)
	_, err = f.v.VerifyRequest(ctx, bad, Expect{})
	wantReason(t, err, obs.ReasonHeaderMismatch)
	if f.res.count() != 0 {
		t.Fatalf("%d resolutions for messages rejected at step 3", f.res.count())
	}
	// A stale request of an unknown agent fails at resolution, not timing.
	u, _ := testSigner(t, 3, respDID+"#key-1")
	f.clock.Set(time.Unix(tVerify+301, 0))
	ur := testRequest()
	ur.Header = ur.Header.Clone()
	ur.Header.Set(FieldSAGEDID, respDID)
	_, err = f.v.VerifyRequest(ctx, signed(t, u, ur, Params{Created: tVerify, Nonce: nonce(1)}), Expect{})
	wantReason(t, err, obs.ReasonResolution)
	if f.res.count() != 1 {
		t.Fatalf("%d resolutions", f.res.count())
	}
}

func TestCheckSigner(t *testing.T) {
	base := `sig1=("@method" "@target-uri" "@authority");created=1;nonce="n-0123456789abcdef";keyid=`
	alias := "did:sage:eth:0x1111111111111111111111111111111111111111"
	hdr := func(v ...string) http.Header {
		h := http.Header{}
		for _, x := range v {
			h.Add(FieldSAGEDID, x)
		}
		return h
	}
	for _, c := range []struct {
		name  string
		keyid string
		h     http.Header
		e     Expect
		want  obs.Reason
	}{
		{"keyid-bad-did", "did:sage:ethereum:a/b", nil, Expect{}, obs.ReasonIdentifier},
		{"keyid-bad-fragment", vecDID + "#1", nil, Expect{}, obs.ReasonIdentifier},
		{"leading zero", vecDID + "#key-01", nil, Expect{}, obs.ReasonIdentifier},
		{"not a DID", "k", nil, Expect{}, obs.ReasonIdentifier},
		{"x-sage-did-mismatch", vecDID, hdr(respDID), Expect{}, obs.ReasonHeaderMismatch},
		{"x-sage-did empty", vecDID, hdr(""), Expect{}, obs.ReasonHeaderMismatch},
		{"x-sage-did not a DID", vecDID, hdr("x"), Expect{}, obs.ReasonHeaderMismatch},
		{"x-sage-did twice", vecDID, hdr(vecDID, vecDID), Expect{}, obs.ReasonDuplicate},
		{"expected DID", vecDID, nil, Expect{DID: mustDID(t, respDID)}, obs.ReasonPeerMismatch},
		{"expected key id: fragment", vecDID + "#key-1", nil, Expect{KeyID: mustRef(t, vecDID+"#key-2")}, obs.ReasonPeerMismatch},
		{"expected key id: none sent", vecDID, nil, Expect{KeyID: mustRef(t, vecDID+"#key-1")}, obs.ReasonPeerMismatch},
		{"expected key id: DID", vecDID + "#key-1", nil, Expect{KeyID: mustRef(t, respDID+"#key-1")}, obs.ReasonPeerMismatch},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := CheckSigner(parsedOf(t, base+`"`+c.keyid+`"`), c.h, c.e)
			wantReason(t, err, c.want)
		})
	}
	// Canonical comparison (06 §2): an alias of the DID is the same DID.
	ref, err := CheckSigner(parsedOf(t, base+`"`+alias+`#key-1"`), hdr(vecDID),
		Expect{DID: mustDID(t, vecDID), KeyID: mustRef(t, vecDID+"#key-1")})
	if err != nil || ref.AsSent() != alias+"#key-1" {
		t.Fatalf("alias: %v", err)
	}
	_, err = CheckSigner(parsedOf(t, `sig1=("@method");created=1`), nil, Expect{})
	wantReason(t, err, obs.ReasonParameters)
}

func mustRef(t *testing.T, s string) did.KeyRef {
	t.Helper()
	r, err := did.ParseKeyRef(s)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestExpectedAuthorities(t *testing.T) {
	s, cs := testSigner(t, 1, vecDID+"#key-1")
	f := newFixture(t, recordFor(t, cs, vecDID))
	ctx := context.Background()
	_, err := f.v.VerifyRequest(ctx, signedAt(t, s, nonce(0)), Expect{Authorities: []string{"agent-a.example", "AGENT-B.example"}})
	wantReason(t, err, obs.ReasonPeerMismatch)
	if f.res.count() != 0 {
		t.Fatal("resolved a request for another authority")
	}
	if _, err := f.v.VerifyRequest(ctx, signedAt(t, s, nonce(0)), Expect{Authorities: []string{"agent-b.example"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.v.VerifyResponse(ctx, Response{}, Request{}, Expect{Authorities: []string{"x"}}); !errors.Is(err, ErrExpect) {
		t.Fatal(err)
	}
}

func digestOf(b []byte) string {
	sum := sha256.Sum256(b)
	return base64.StdEncoding.EncodeToString(sum[:])
}

func TestCheckDigest(t *testing.T) {
	body := []byte(`{"a":1}`)
	good := "sha-256=:" + digestOf(body) + ":"
	short := base64.StdEncoding.EncodeToString(make([]byte, 31))
	lim := core.DefaultLimits()
	h := func(v ...string) http.Header {
		out := http.Header{}
		for _, x := range v {
			out.Add(FieldContentDigest, x)
		}
		return out
	}
	if err := CheckDigest(h(good), body, lim); err != nil {
		t.Fatal(err)
	}
	if err := CheckDigest(h(), nil, lim); err != nil {
		t.Fatalf("no body, no digest: %v", err)
	}
	if err := CheckDigest(h(ContentDigest(nil)), nil, lim); err != nil {
		t.Fatalf("empty body with its digest: %v", err)
	}
	for name, c := range map[string]struct {
		h    http.Header
		body []byte
		want obs.Reason
	}{
		"digest-mismatch":     {h(good), []byte(`{"a":2}`), obs.ReasonDigest},
		"digest-two-members":  {h(good + ", sha-512=:" + base64.StdEncoding.EncodeToString(make([]byte, 64)) + ":"), body, obs.ReasonDigest},
		"digest-sha512-only":  {h("sha-512=:" + base64.StdEncoding.EncodeToString(make([]byte, 64)) + ":"), body, obs.ReasonDigest},
		"31-byte member":      {h("sha-256=:" + short + ":"), body, obs.ReasonDigest},
		"other name":          {h("sha-384=:" + digestOf(body) + ":"), body, obs.ReasonDigest},
		"duplicate member":    {h(good + ", " + good), body, obs.ReasonDigest},
		"parameter":           {h(good + ";a=1"), body, obs.ReasonDigest},
		"string value":        {h(`sha-256="` + digestOf(body) + `"`), body, obs.ReasonDigest},
		"inner list":          {h("sha-256=(:" + digestOf(body) + ":)"), body, obs.ReasonDigest},
		"malformed":           {h("sha-256=:%%:"), body, obs.ReasonDigest},
		"empty value":         {h(""), body, obs.ReasonDigest},
		"missing with a body": {h(), body, obs.ReasonDigest},
		"two field lines":     {h(good, good), body, obs.ReasonDuplicate},
	} {
		t.Run(name, func(t *testing.T) { wantReason(t, CheckDigest(c.h, c.body, lim), c.want) })
	}
	// 16 MiB is the limit: at the limit accepted, one byte more rejected
	// before the digest is computed.
	big := make([]byte, core.SpecMaxDigestBodyBytes)
	if err := CheckDigest(h("sha-256=:"+digestOf(big)+":"), big, lim); err != nil {
		t.Fatalf("16 MiB: %v", err)
	}
	big = append(big, 0)
	wantReason(t, CheckDigest(h("sha-256=:"+digestOf(big)+":"), big, lim), obs.ReasonTooLarge)
	if err := CheckDigest(h(good), body, core.Limits{}); err == nil {
		t.Fatal("zero limits accepted")
	}
}

func TestVerifyRequestBodyLimit(t *testing.T) {
	s, cs := testSigner(t, 1, vecDID+"#key-1")
	f := newFixture(t, recordFor(t, cs, vecDID))
	r := testRequest()
	r.Body = make([]byte, core.SpecMaxDigestBodyBytes+1)
	r = signed(t, s, r, Params{Created: tVerify, Nonce: nonce(0)})
	_, err := f.v.VerifyRequest(context.Background(), r, Expect{})
	wantReason(t, err, obs.ReasonTooLarge)
	if f.res.count() != 0 {
		t.Fatal("resolved an oversize request")
	}
}

func TestVerifyResolution(t *testing.T) {
	s, cs := testSigner(t, 1, vecDID+"#key-1")
	ctx := context.Background()
	inactive := recordFor(t, cs, vecDID)
	inactive.IsActive = false
	f := newFixture(t, inactive)
	_, err := f.v.VerifyRequest(ctx, signedAt(t, s, nonce(0)), Expect{})
	wantReason(t, err, obs.ReasonInactive)
	// A resolver that answers with another agent's record lends no key.
	f = newFixture(t)
	f.res.recs[mustDID(t, vecDID).String()] = recordFor(t, cs, respDID)
	_, err = f.v.VerifyRequest(ctx, signedAt(t, s, nonce(0)), Expect{})
	wantReason(t, err, obs.ReasonResolution)
	// A record whose key is another key: the signature does not verify.
	_, otherKey := testSigner(t, 2, vecDID)
	f = newFixture(t, recordFor(t, otherKey, vecDID))
	_, err = f.v.VerifyRequest(ctx, signedAt(t, s, nonce(0)), Expect{})
	wantReason(t, err, obs.ReasonSignature)
}

// TestVerifyES256K covers E03-23 and E01-11 with a secp256k1 agent:
// es256k-64-byte, es256k-der and alg-key-mismatch.
func TestVerifyES256K(t *testing.T) {
	k1, err := crypto.NewSecp256k1Signer(secretScalar(t))
	if err != nil {
		t.Fatal(err)
	}
	ref := mustRef(t, vecDID+"#key-1")
	s, err := NewSigner(ref, k1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	f := newFixture(t, recordFor(t, k1, vecDID))
	if _, err := f.v.VerifyRequest(ctx, signedAt(t, s, nonce(0)), Expect{}); err != nil {
		t.Fatal(err)
	}
	reencode := func(n string, f func([]byte) []byte) Request {
		r := signedAt(t, s, n)
		p, err := Parse(r.Header, "", core.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		r.Header = r.Header.Clone()
		r.Header.Set(FieldSignature, Label+"=:"+base64.StdEncoding.EncodeToString(f(p.Signature()))+":")
		return r
	}
	_, err = f.v.VerifyRequest(ctx, reencode(nonce(1), func(b []byte) []byte { return b[:64] }), Expect{})
	wantReason(t, err, obs.ReasonSignature)
	_, err = f.v.VerifyRequest(ctx, reencode(nonce(2), func(b []byte) []byte {
		return append([]byte{0x30, 0x44, 0x02, 0x20}, append(append([]byte(nil), b[:32]...), append([]byte{0x02, 0x20}, b[32:64]...)...)...)
	}), Expect{})
	wantReason(t, err, obs.ReasonSignature)
	// alg-key-mismatch: an Ed25519 signer that claims es256k.
	es, edKey := testSigner(t, 1, vecDID+"#key-1")
	f = newFixture(t, recordFor(t, edKey, vecDID))
	r := signedAt(t, es, nonce(3))
	r.Header = r.Header.Clone()
	in := r.Header.Get(FieldSignatureInput)
	mutated := strings.Replace(in, `alg="ed25519"`, `alg="es256k"`, 1)
	if mutated == in {
		mutated = strings.Replace(in, `;created=`, `;alg="es256k";created=`, 1)
	}
	r.Header.Set(FieldSignatureInput, mutated)
	_, err = f.v.VerifyRequest(ctx, r, Expect{})
	wantReason(t, err, obs.ReasonAlgorithm)
}

// TestVerifyResponse: a signed response verifies for its signed request; an
// unsigned response, as a responder whose signer fails sends, is a failure
// whose body the client never gets (E03-20).
func TestVerifyResponse(t *testing.T) {
	reqSigner, reqKey := testSigner(t, 1, vecDID+"#key-1")
	respSigner, respKey := testSigner(t, 2, respDID+"#key-1")
	req := signedAt(t, reqSigner, nonce(0))
	f := newFixture(t, recordFor(t, reqKey, vecDID), recordFor(t, respKey, respDID))
	ctx := context.Background()
	resp := Response{Status: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: []byte(`{"ok":true}`)}
	fields, err := respSigner.SignResponse(ctx, resp, req, Params{Created: tVerify, Nonce: nonce(1)})
	if err != nil {
		t.Fatal(err)
	}
	resp.Header = resp.Header.Clone()
	fields.Apply(resp.Header)
	vr, err := f.v.VerifyResponse(ctx, resp, req, Expect{DID: mustDID(t, respDID)})
	if err != nil {
		t.Fatal(err)
	}
	if !vr.Valid() || !vr.Signer().Equal(mustDID(t, respDID)) || !vr.Key().Equal(respKey.Public()) || vr.KeyID().Fragment() != "key-1" ||
		vr.Created().Unix() != tVerify || vr.Label() != Label {
		t.Fatal("verified response")
	}
	var zero VerifiedResponse
	if zero.Valid() || zero.Key() != nil || zero.Label() != "" || !zero.Signer().IsZero() || !zero.KeyID().DID().IsZero() || !zero.Created().IsZero() {
		t.Fatal("zero VerifiedResponse")
	}
	// A response is not reserved (DP-4): the same response verifies again.
	if _, err := f.v.VerifyResponse(ctx, resp, req, Expect{}); err != nil {
		t.Fatal(err)
	}
	_, err = f.v.VerifyResponse(ctx, resp, req, Expect{DID: mustDID(t, vecDID)})
	wantReason(t, err, obs.ReasonPeerMismatch)
	tampered := resp
	tampered.Body = []byte(`{"ok":false}`)
	_, err = f.v.VerifyResponse(ctx, tampered, req, Expect{})
	wantReason(t, err, obs.ReasonDigest)

	// A test server whose response signer fails answers with an unsigned
	// error; the client's verification fails and the body is not used.
	failing, _ := NewSigner(mustRef(t, respDID+"#key-1"), failingSigner{respKey})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		out := Response{Status: 200, Header: http.Header{}, Body: []byte(`{"result":"secret"}`)}
		sig, err := failing.SignResponse(context.Background(), out, req, Params{Created: tVerify, Nonce: nonce(2)})
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		sig.Apply(w.Header())
		_, _ = w.Write(out.Body)
	}))
	defer srv.Close()
	got, err := clientCall(ctx, f.v, srv.URL, req)
	if err == nil || got != nil {
		t.Fatalf("client used an unsigned error response: %q, %v", got, err)
	}
}

// clientCall is a client of E03-20: it returns the body only of a response
// that verifies for req.
func clientCall(ctx context.Context, v *Verifier, url string, req Request) ([]byte, error) {
	hr, err := http.NewRequestWithContext(ctx, req.Method, url, bytes.NewReader(req.Body))
	if err != nil {
		return nil, err
	}
	hr.Header = req.Header.Clone()
	res, err := http.DefaultClient.Do(hr)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if _, err := v.VerifyResponse(ctx, Response{Status: res.StatusCode, Header: res.Header, Body: body}, req, Expect{}); err != nil {
		return nil, err
	}
	return body, nil
}

type failingSigner struct{ crypto.Signer }

func (failingSigner) Sign(context.Context, crypto.Purpose, []byte) (crypto.Signature, error) {
	return crypto.Signature{}, errors.New("key store unavailable")
}

func TestTargetFrom(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/mcp/tools/call?x=1", nil)
	r.Host = "internal:8080"
	got, err := TargetFrom("https://agent-b.example", r)
	if err != nil || got != "https://agent-b.example/mcp/tools/call?x=1" {
		t.Fatalf("%q, %v", got, err)
	}
	if got, err := TargetFrom("http://agent-b.example:8080", r); err != nil || got != "http://agent-b.example:8080/mcp/tools/call?x=1" {
		t.Fatalf("%q, %v", got, err)
	}
	for _, origin := range []string{"", "agent-b.example", "https://agent-b.example/", "https://agent-b.example/base",
		"https://Agent-B.example", "https://agent-b.example:443", "ftp://agent-b.example", "https://u@agent-b.example", "https://agent-b.example?q"} {
		if _, err := TargetFrom(origin, r); !errors.Is(err, ErrOrigin) {
			t.Errorf("origin %q: %v", origin, err)
		}
	}
	abs := httptest.NewRequest(http.MethodGet, "/", nil)
	abs.RequestURI = "http://evil.example/x"
	for _, x := range []*http.Request{nil, abs} {
		if _, err := TargetFrom("https://agent-b.example", x); !errors.Is(err, ErrOrigin) {
			t.Error(err)
		}
	}
}

// TestWriteAuthenticationFailure: every failure gets the same answer
// (E03-17).
func TestWriteAuthenticationFailure(t *testing.T) {
	write := func(prior string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		w.Header().Set("X-Reason", prior)
		if err := WriteAuthenticationFailure(w, `SAGE-RFC9421 realm="agents"`); err != nil {
			t.Fatal(err)
		}
		return w
	}
	a, b := write("stale"), write("digest")
	if a.Code != http.StatusUnauthorized || a.Body.String() != "authentication failed" ||
		a.Header().Get("WWW-Authenticate") != `SAGE-RFC9421 realm="agents"` || a.Header().Get("X-Reason") != "" {
		t.Fatalf("%d %q %v", a.Code, a.Body.String(), a.Header())
	}
	if a.Body.String() != b.Body.String() || len(a.Header()) != len(b.Header()) {
		t.Fatal("answers differ")
	}
	for _, c := range []string{"", "a\r\nX: y", "ä"} {
		if err := WriteAuthenticationFailure(httptest.NewRecorder(), c); !errors.Is(err, ErrChallenge) {
			t.Errorf("%q: %v", c, err)
		}
	}
}
