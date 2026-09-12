package vectors

import (
	"bytes"
	"crypto"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sage-x-project/sage/pkg/agent/core/rfc9421"
)

// Fixed request used by every RFC 9421 vector.
const (
	vecMethod    = "POST"
	vecURL       = "https://agent-b.example/mcp/tools/call"
	vecBody      = `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"echo","arguments":{"text":"hello"}}}`
	vecDate      = "Tue, 01 Sep 2026 12:00:00 GMT"
	vecCreated   = int64(1788609600) // 2026-09-05T12:00:00Z
	vecNonce     = "8f0c5b1c-0d1a-4c8e-9a51-6d2e4b7f1a10"
	vecDIDA      = "did:sage:ethereum:0x1111111111111111111111111111111111111111"
	vecDIDB      = "did:sage:ethereum:0x2222222222222222222222222222222222222222"
	vecSigName   = "sig1"
	vecRespBody  = `{"jsonrpc":"2.0","id":7,"result":{"content":[{"type":"text","text":"hello"}]}}`
	vecRespCT    = "application/json"
	vecRequestCT = "application/json"
)

var requestComponents = []string{`"@method"`, `"@target-uri"`, `"@authority"`, `"content-type"`, `"content-digest"`, `"x-sage-did"`, `"date"`}

// responseComponents is what rfc9421.ResponseSigner covers for a request that
// carried Content-Digest and Signature and a response with a body.
var responseComponents = []string{`"@status"`, `"@method";req`, `"@target-uri";req`, `"@authority";req`, `"content-digest";req`, `"signature";req`, `"content-type"`, `"content-digest"`}

func buildRequest() (*http.Request, error) {
	req, err := http.NewRequest(vecMethod, vecURL, strings.NewReader(vecBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", vecRequestCT)
	req.Header.Set("Content-Digest", rfc9421.ComputeContentDigest([]byte(vecBody)))
	req.Header.Set("X-SAGE-DID", vecDIDA)
	req.Header.Set("Date", vecDate)
	return req, nil
}

func buildResponse(req *http.Request) *http.Response {
	resp := &http.Response{
		StatusCode: 200,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(vecRespBody)),
		Request:    req,
	}
	resp.Header.Set("Content-Type", vecRespCT)
	resp.Header.Set("Content-Digest", rfc9421.ComputeContentDigest([]byte(vecRespBody)))
	return resp
}

func requestInput(alg, keyLabel string) map[string]any {
	return map[string]any{
		"method":         vecMethod,
		"url":            vecURL,
		"headers":        map[string]any{"content-type": vecRequestCT, "x-sage-did": vecDIDA, "date": vecDate},
		"body":           vecBody,
		"covered":        requestComponents,
		"signature_name": vecSigName,
		"keyid":          vecDIDA + "#key-1",
		"alg":            alg,
		"created":        vecCreated,
		"nonce":          vecNonce,
		"key_label":      keyLabel,
	}
}

func signerFor(alg, label string) (crypto.Signer, crypto.PublicKey, error) {
	switch alg {
	case "ed25519":
		priv := ed25519FromLabel(label)
		return priv, priv.Public(), nil
	case "es256k":
		priv, err := secp256k1FromLabel(label)
		if err != nil {
			return nil, nil, err
		}
		return priv, &priv.PublicKey, nil
	case "ecdsa-p256-sha256":
		priv, err := p256FromLabel(label)
		if err != nil {
			return nil, nil, err
		}
		return priv, &priv.PublicKey, nil
	}
	return nil, nil, fmt.Errorf("unsupported alg %q", alg)
}

func paramsFrom(in map[string]any) *rfc9421.SignatureInputParams {
	p := &rfc9421.SignatureInputParams{
		CoveredComponents: requestComponents,
		KeyID:             in["keyid"].(string),
		Algorithm:         in["alg"].(string),
		Created:           vecCreated,
		Nonce:             vecNonce,
	}
	return p
}

func signRequestVector(in map[string]any) (map[string]any, error) {
	alg, _ := str(in, "alg")
	label, _ := str(in, "key_label")
	signer, _, err := signerFor(alg, label)
	if err != nil {
		return nil, err
	}
	req, err := buildRequest()
	if err != nil {
		return nil, err
	}
	v := rfc9421.NewHTTPVerifier()
	if err := v.SignRequest(req, vecSigName, paramsFrom(in), signer); err != nil {
		return nil, err
	}
	base, err := rfc9421.NewCanonicalizer().BuildSignatureBase(req, vecSigName, paramsFrom(in))
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"content_digest":  req.Header.Get("Content-Digest"),
		"signature_base":  base,
		"signature_input": req.Header.Get("Signature-Input"),
		"signature":       req.Header.Get("Signature"),
	}, nil
}

func verifyRequestVector(in, out map[string]any) error {
	alg, _ := str(in, "alg")
	label, _ := str(in, "key_label")
	_, pub, err := signerFor(alg, label)
	if err != nil {
		return err
	}
	req, err := buildRequest()
	if err != nil {
		return err
	}
	for _, h := range []string{"signature_input", "signature"} {
		if _, err := str(out, h); err != nil {
			return err
		}
	}
	req.Header.Set("Signature-Input", out["signature_input"].(string))
	req.Header.Set("Signature", out["signature"].(string))
	opts := &rfc9421.HTTPVerificationOptions{
		SignatureName:        vecSigName,
		MaxAge:               0, // vectors carry a fixed created timestamp
		RequiredComponents:   []string{"@method", "@target-uri", "@authority", "content-digest"},
		RequireContentDigest: true,
		RequireNonce:         true,
		DisableReplayCheck:   true,
		ExpectedDID:          vecDIDA,
	}
	return rfc9421.NewHTTPVerifier().VerifyRequest(req, pub, opts)
}

func rfc9421Suite() Suite {
	s := Suite{
		Name: "rfc9421",
		Description: "HTTP Message Signatures profile: covered components, signature base, Signature-Input/Signature headers, " +
			"Content-Digest (sha-256), keyid = DID#fragment, and response signatures bound to the request with ;req.",
	}
	for _, c := range []struct {
		alg, label string
		mode       Mode
	}{
		{"ed25519", labelEd25519A, ModeDeterministic},
		{"es256k", labelSecp256k1A, ModeDeterministic},
		{"ecdsa-p256-sha256", labelP256A, ModeVerify},
	} {
		s.Cases = append(s.Cases, Case{
			Name: "request-" + c.alg, Mode: c.mode,
			Description: "Signed POST request with alg=" + c.alg + ". Verify with MaxAge disabled (fixed created), nonce required, ExpectedDID bound to keyid.",
			Input:       requestInput(c.alg, c.label),
			Produce:     signRequestVector,
			Verify:      verifyRequestVector,
		})
	}
	s.Cases = append(s.Cases, Case{
		Name: "response-ed25519", Mode: ModeDeterministic,
		Description: "Response signed by agent B over @status plus the request's @method/@target-uri/@authority/content-digest/signature with ;req, " +
			"and the response content-type/content-digest. The request carries the ed25519 request signature vector.",
		Input: map[string]any{
			"request":        requestInput("ed25519", labelEd25519A),
			"status":         200,
			"headers":        map[string]any{"content-type": vecRespCT},
			"body":           vecRespBody,
			"covered":        responseComponents,
			"signature_name": vecSigName,
			"keyid":          vecDIDB + "#key-1",
			"alg":            "ed25519",
			"created":        vecCreated + 1,
			"nonce":          vecNonce + "-resp",
			"key_label":      labelEd25519B,
		},
		Produce: func(in map[string]any) (map[string]any, error) {
			reqIn := in["request"].(map[string]any)
			signerA, _, err := signerFor("ed25519", labelEd25519A)
			if err != nil {
				return nil, err
			}
			req, err := buildRequest()
			if err != nil {
				return nil, err
			}
			if err := rfc9421.NewHTTPVerifier().SignRequest(req, vecSigName, paramsFrom(reqIn), signerA); err != nil {
				return nil, err
			}
			resp := buildResponse(req)
			respParams := &rfc9421.SignatureInputParams{
				CoveredComponents: responseComponents,
				KeyID:             vecDIDB + "#key-1",
				Algorithm:         "ed25519",
				Created:           vecCreated + 1,
				Nonce:             vecNonce + "-resp",
			}
			if err := rfc9421.NewHTTPVerifier().SignResponse(resp, req, vecSigName, respParams, ed25519FromLabel(labelEd25519B)); err != nil {
				return nil, err
			}
			params, err := rfc9421.ParseSignatureInput(resp.Header.Get("Signature-Input"))
			if err != nil {
				return nil, err
			}
			base, err := rfc9421.NewCanonicalizer().BuildResponseSignatureBase(resp, req, vecSigName, params[vecSigName])
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"request_signature_input": req.Header.Get("Signature-Input"),
				"request_signature":       req.Header.Get("Signature"),
				"content_digest":          resp.Header.Get("Content-Digest"),
				"signature_base":          base,
				"signature_input":         resp.Header.Get("Signature-Input"),
				"signature":               resp.Header.Get("Signature"),
			}, nil
		},
		Verify: func(in, out map[string]any) error {
			req, err := buildRequest()
			if err != nil {
				return err
			}
			req.Header.Set("Signature-Input", out["request_signature_input"].(string))
			req.Header.Set("Signature", out["request_signature"].(string))
			resp := buildResponse(req)
			resp.Body = io.NopCloser(bytes.NewReader([]byte(vecRespBody)))
			resp.Header.Set("Signature-Input", out["signature_input"].(string))
			resp.Header.Set("Signature", out["signature"].(string))
			opts := &rfc9421.HTTPVerificationOptions{
				SignatureName:         vecSigName,
				MaxAge:                0,
				RequiredComponents:    []string{"@status", "content-digest"},
				RequireContentDigest:  true,
				RequireNonce:          true,
				DisableReplayCheck:    true,
				RequireRequestBinding: true,
				ExpectedDID:           vecDIDB,
			}
			pubB := ed25519FromLabel(labelEd25519B).Public()
			return rfc9421.NewHTTPVerifier().VerifyResponse(resp, req, pubB, opts)
		},
	})
	return s
}
