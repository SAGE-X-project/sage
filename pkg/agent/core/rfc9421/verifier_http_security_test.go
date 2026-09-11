package rfc9421

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// signedRequest builds a POST request with a JSON body signed under params.
func signedRequest(t *testing.T, v *HTTPVerifier, priv ed25519.PrivateKey, params *SignatureInputParams, body string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "https://agent.example/v1/a2a", bytes.NewBufferString(body))
	require.NoError(t, err)
	req.Header.Set("Host", "agent.example")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Digest", ComputeContentDigest([]byte(body)))
	require.NoError(t, v.SignRequest(req, "sig1", params, priv))
	return req
}

func newTestKey(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	return pub, priv
}

func baseParams(nonce string) *SignatureInputParams {
	return &SignatureInputParams{
		CoveredComponents: []string{`"@method"`, `"@target-uri"`, `"@authority"`, `"content-type"`, `"content-digest"`},
		KeyID:             "did:sage:ethereum:0xabc#key-1",
		Algorithm:         "ed25519",
		Created:           time.Now().Unix(),
		Nonce:             nonce,
	}
}

func TestVerifyRequest_RejectsReplayedNonce(t *testing.T) {
	pub, priv := newTestKey(t)
	v := NewHTTPVerifier()
	defer v.Close()

	req := signedRequest(t, v, priv, baseParams("nonce-1"), `{"a":1}`)
	require.NoError(t, v.VerifyRequest(req, pub, nil), "first presentation must verify")

	err := v.VerifyRequest(req, pub, nil)
	require.Error(t, err, "second presentation of the same nonce must be rejected")
	assert.Contains(t, err.Error(), "replay")
}

func TestVerifyRequest_ReplayCheckCanBeDisabled(t *testing.T) {
	pub, priv := newTestKey(t)
	v := NewHTTPVerifier()
	defer v.Close()

	req := signedRequest(t, v, priv, baseParams("nonce-2"), `{"a":1}`)
	opts := DefaultHTTPVerificationOptions()
	opts.DisableReplayCheck = true
	require.NoError(t, v.VerifyRequest(req, pub, opts))
	require.NoError(t, v.VerifyRequest(req, pub, opts))
}

func TestVerifyRequest_RequireNonce(t *testing.T) {
	pub, priv := newTestKey(t)
	v := NewHTTPVerifier()
	defer v.Close()

	req := signedRequest(t, v, priv, baseParams(""), `{"a":1}`)
	opts := DefaultHTTPVerificationOptions()
	opts.RequireNonce = true
	err := v.VerifyRequest(req, pub, opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonce")
}

func TestVerifyRequest_RejectsCreatedInTheFuture(t *testing.T) {
	pub, priv := newTestKey(t)
	v := NewHTTPVerifier()
	defer v.Close()

	params := baseParams("nonce-3")
	params.Created = time.Now().Add(24 * time.Hour).Unix()
	req := signedRequest(t, v, priv, params, `{"a":1}`)
	err := v.VerifyRequest(req, pub, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "future")
}

func TestVerifyRequest_AllowsSmallForwardSkew(t *testing.T) {
	pub, priv := newTestKey(t)
	v := NewHTTPVerifier()
	defer v.Close()

	params := baseParams("nonce-4")
	params.Created = time.Now().Add(20 * time.Second).Unix()
	req := signedRequest(t, v, priv, params, `{"a":1}`)
	require.NoError(t, v.VerifyRequest(req, pub, nil))
}

func TestVerifyRequest_EnforcesRequiredComponents(t *testing.T) {
	pub, priv := newTestKey(t)
	v := NewHTTPVerifier()
	defer v.Close()

	params := baseParams("nonce-5")
	params.CoveredComponents = []string{`"@method"`, `"content-type"`} // no digest, no target
	req := signedRequest(t, v, priv, params, `{"a":1}`)

	opts := DefaultHTTPVerificationOptions()
	opts.RequiredComponents = []string{"@target-uri", "content-digest"}
	err := v.VerifyRequest(req, pub, opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "@target-uri")
}

func TestVerifyRequest_StrictOptionsRejectBodySwap(t *testing.T) {
	pub, priv := newTestKey(t)
	v := NewHTTPVerifier()
	defer v.Close()

	// Signed without content-digest coverage: strict options must refuse it outright.
	loose := baseParams("nonce-6")
	loose.CoveredComponents = []string{`"@method"`, `"@target-uri"`, `"@authority"`, `"content-type"`}
	req := signedRequest(t, v, priv, loose, `{"amount":1}`)
	err := v.VerifyRequest(req, pub, StrictHTTPVerificationOptions())
	require.Error(t, err, "strict verification must require content-digest for a request with a body")

	// Signed with content-digest coverage: swapping the body must fail verification.
	req = signedRequest(t, v, priv, baseParams("nonce-7"), `{"amount":1}`)
	req.Body = http.NoBody
	req.Body = readCloser(`{"amount":1000000}`)
	err = v.VerifyRequest(req, pub, StrictHTTPVerificationOptions())
	require.Error(t, err)
}

func TestVerifyRequest_DeterministicSignatureSelection(t *testing.T) {
	pub, priv := newTestKey(t)
	_, otherPriv := newTestKey(t)
	v := NewHTTPVerifier()
	defer v.Close()

	// Two signatures on one request: "sig1" by an unrelated key, "b" by the verified key.
	req := signedRequest(t, v, otherPriv, baseParams("nonce-8"), `{"a":1}`)
	inputA, sigA := req.Header.Get("Signature-Input"), req.Header.Get("Signature")
	require.NoError(t, v.SignRequest(req, "b", baseParams("nonce-9"), priv))
	req.Header.Set("Signature-Input", inputA+", "+req.Header.Get("Signature-Input"))
	req.Header.Set("Signature", sigA+", "+req.Header.Get("Signature"))

	opts := DefaultHTTPVerificationOptions()
	opts.DisableReplayCheck = true
	var results []bool
	for i := 0; i < 20; i++ {
		results = append(results, v.VerifyRequest(req, pub, opts) == nil)
	}
	for _, r := range results[1:] {
		assert.Equal(t, results[0], r, "verification outcome must not depend on map iteration order")
	}
	// The lexicographically first label ("b" < "sig1") is chosen when no name is given.
	assert.True(t, results[0])
	opts.SignatureName = "sig1"
	assert.Error(t, v.VerifyRequest(req, pub, opts), "explicitly selecting the foreign signature must fail")
}
