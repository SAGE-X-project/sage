package rfc9421

import (
	"bytes"
	"crypto"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func responseParams(covered ...string) *SignatureInputParams {
	return &SignatureInputParams{
		CoveredComponents: covered,
		KeyID:             "did:sage:ethereum:0xtool#key-1",
		Algorithm:         "ed25519",
		Created:           time.Now().Unix(),
	}
}

var boundComponents = []string{`"@status"`, `"@method";req`, `"@target-uri";req`, `"@authority";req`, `"content-digest";req`, `"content-type"`, `"content-digest"`}

// signedExchange returns a signed request, a response to it signed by a fresh
// tool key, and that key's public half.
func signedExchange(t *testing.T, v *HTTPVerifier, reqBody, respBody string) (*http.Request, *http.Response, crypto.PublicKey) {
	t.Helper()
	_, agentPriv := newTestKey(t)
	req := signedRequest(t, v, agentPriv, baseParams("req-nonce-"+respBody), reqBody)

	_, toolPriv := newTestKey(t)
	resp := &http.Response{
		StatusCode:    http.StatusOK,
		Header:        http.Header{"Content-Type": {"application/json"}, "Content-Digest": {ComputeContentDigest([]byte(respBody))}},
		Body:          io.NopCloser(bytes.NewBufferString(respBody)),
		ContentLength: int64(len(respBody)),
		Request:       req,
	}
	require.NoError(t, v.SignResponse(resp, req, "sig1", responseParams(boundComponents...), toolPriv))
	return req, resp, toolPriv.Public()
}

func TestVerifyResponse_BoundToRequest(t *testing.T) {
	v := NewHTTPVerifier()
	defer v.Close()
	req, resp, pub := signedExchange(t, v, `{"op":"add"}`, `{"result":3}`)

	require.NoError(t, v.VerifyResponse(resp, req, pub, StrictHTTPResponseVerificationOptions()))

	// Body still readable after digest validation.
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, `{"result":3}`, string(body))

	// Same response presented for a different request must fail: @target-uri;req differs.
	other, err := http.NewRequest(http.MethodPost, "https://agent.example/v1/other", bytes.NewBufferString(`{"op":"add"}`))
	require.NoError(t, err)
	other.Header = req.Header.Clone()
	require.Error(t, v.VerifyResponse(resp, other, pub, StrictHTTPResponseVerificationOptions()))

	// A request with a different body (digest) must fail too.
	swapped, err := http.NewRequest(http.MethodPost, req.URL.String(), bytes.NewBufferString(`{"op":"mul"}`))
	require.NoError(t, err)
	swapped.Header = req.Header.Clone()
	swapped.Header.Set("Content-Digest", ComputeContentDigest([]byte(`{"op":"mul"}`)))
	require.Error(t, v.VerifyResponse(resp, swapped, pub, StrictHTTPResponseVerificationOptions()))
}

func TestVerifyResponse_RejectsTamperedResultAndStatus(t *testing.T) {
	v := NewHTTPVerifier()
	defer v.Close()
	req, resp, pub := signedExchange(t, v, `{"op":"add"}`, `{"result":3}`)

	// Swapped tool result: digest header no longer matches the body.
	resp.Body = io.NopCloser(bytes.NewBufferString(`{"result":1000000}`))
	err := v.VerifyResponse(resp, req, pub, StrictHTTPResponseVerificationOptions())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content-digest mismatch")

	// Swapped result with a recomputed digest: signature over content-digest fails.
	resp.Header.Set("Content-Digest", ComputeContentDigest([]byte(`{"result":1000000}`)))
	require.Error(t, v.VerifyResponse(resp, req, pub, StrictHTTPResponseVerificationOptions()))

	// Restore body, change status: @status is covered.
	resp.Header.Set("Content-Digest", ComputeContentDigest([]byte(`{"result":3}`)))
	resp.Body = io.NopCloser(bytes.NewBufferString(`{"result":3}`))
	resp.StatusCode = http.StatusAccepted
	require.Error(t, v.VerifyResponse(resp, req, pub, StrictHTTPResponseVerificationOptions()))
}

func TestVerifyResponse_StrictRequiresBindingAndBody(t *testing.T) {
	v := NewHTTPVerifier()
	defer v.Close()
	_, agentPriv := newTestKey(t)
	req := signedRequest(t, v, agentPriv, baseParams("n-unbound"), `{"op":"add"}`)
	_, toolPriv := newTestKey(t)
	body := `{"result":3}`
	resp := &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(bytes.NewBufferString(body)), ContentLength: int64(len(body)), Request: req}

	// Signed without request binding and without content-digest.
	require.NoError(t, v.SignResponse(resp, req, "sig1", responseParams(`"@status"`, `"content-type"`), toolPriv))
	err := v.VerifyResponse(resp, req, toolPriv.Public(), StrictHTTPResponseVerificationOptions())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not covered")

	// Lenient options accept it.
	require.NoError(t, v.VerifyResponse(resp, req, toolPriv.Public(), DefaultHTTPVerificationOptions()))
}

func TestCanonicalizer_ReqParameterOnlyInResponses(t *testing.T) {
	v := NewHTTPVerifier()
	defer v.Close()
	_, priv := newTestKey(t)
	req, err := http.NewRequest(http.MethodGet, "https://agent.example/x", nil)
	require.NoError(t, err)
	err = v.SignRequest(req, "sig1", responseParams(`"@method";req`), priv)
	require.Error(t, err)

	resp := &http.Response{StatusCode: 204, Header: http.Header{}, Request: req}
	// @method without ;req is not a response component.
	require.Error(t, v.SignResponse(resp, req, "sig1", responseParams(`"@method"`), priv))
	// With ;req it is, and the base line spells the identifier as given.
	base, err := v.canonicalizer.BuildResponseSignatureBase(resp, req, "sig1", responseParams(`"@status"`, `"@method";req`))
	require.NoError(t, err)
	assert.Contains(t, base, "\"@status\": 204\n\"@method\";req: GET\n")
}

func TestIsComponentCovered_NormalizesParameters(t *testing.T) {
	covered := []string{`"@method";req`, `"Content-Digest"`, ` "@status" `}
	assert.True(t, IsComponentCovered(covered, "@method;req"))
	assert.True(t, IsComponentCovered(covered, `"@method";req`))
	assert.False(t, IsComponentCovered(covered, "@method"))
	assert.True(t, IsComponentCovered(covered, "content-digest"))
	assert.True(t, IsComponentCovered(covered, "@status"))
}

func TestResponseSigner_MiddlewareSignsToolResults(t *testing.T) {
	v := NewHTTPVerifier()
	defer v.Close()
	_, toolPriv := newTestKey(t)
	signer := &ResponseSigner{Verifier: v, PrivateKey: toolPriv, KeyID: "did:sage:ethereum:0xtool#key-1", Algorithm: "ed25519"}

	handler := signer.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"result":3}`))
	}))
	srv := httptest.NewServer(handler)
	defer srv.Close()

	_, agentPriv := newTestKey(t)
	client := NewHTTPVerifier()
	defer client.Close()
	body := `{"op":"add"}`
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/tools/calc", bytes.NewBufferString(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Digest", ComputeContentDigest([]byte(body)))
	params := baseParams("mw-nonce")
	params.Algorithm = "ed25519"
	require.NoError(t, client.SignRequest(req, "sig1", params, agentPriv))

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	inputs, err := ParseSignatureInput(resp.Header.Get("Signature-Input"))
	require.NoError(t, err)
	assert.True(t, IsComponentCovered(inputs["sig1"].CoveredComponents, `"signature";req`), "response must be bound to the request signature")
	assert.True(t, IsComponentCovered(inputs["sig1"].CoveredComponents, `"content-digest";req`))

	// The client verifies with the request it sent (resp.Request is set by net/http).
	require.NoError(t, client.VerifyResponse(resp, req, toolPriv.Public(), StrictHTTPResponseVerificationOptions()))
	got, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, `{"result":3}`, string(got))
}
