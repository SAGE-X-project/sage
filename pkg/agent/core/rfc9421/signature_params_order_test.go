package rfc9421

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestReceivedSignatureParametersKeepTagAndOrder(t *testing.T) {
	seed := sha256.Sum256([]byte("sage RFC 9421 ordered parameters test"))
	private := ed25519.NewKeyFromSeed(seed[:])
	public := private.Public().(ed25519.PublicKey)
	const did = "did:sage:web:agent.example:alice"
	request, err := http.NewRequest(http.MethodPost,
		"https://agent.example/call?x=1", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	digest := sha256.Sum256([]byte("{}"))
	request.Header.Set("Content-Digest", "sha-256=:"+base64.StdEncoding.EncodeToString(digest[:])+":")
	request.Header.Set("X-Sage-Did", did)
	request.Header.Set("X-Sage-Version", "0.10.0")
	created := time.Now().Unix()
	components := []string{`"@method"`, `"@target-uri"`, `"@authority"`,
		`"content-type"`, `"content-digest"`, `"x-sage-did"`, `"x-sage-version"`}
	params := fmt.Sprintf(`sig1=(%s);tag="sage-0.10.0";created=%d;keyid="%s#signing-1";alg="ed25519";expires=%d;nonce="MDEyMzQ1Njc4OWFiY2RlZg"`,
		strings.Join(components, " "), created, did, created+300)
	request.Header.Set("Signature-Input", params)
	parsed, err := ParseSignatureInput(params)
	if err != nil || parsed["sig1"].Tag != "sage-0.10.0" {
		t.Fatalf("tag not parsed: %v", err)
	}
	base := fmt.Sprintf(`"@method": POST
"@target-uri": https://agent.example/call?x=1
"@authority": agent.example
"content-type": application/json
"content-digest": %s
"x-sage-did": %s
"x-sage-version": 0.10.0
"@signature-params": %s`, request.Header.Get("Content-Digest"), did,
		strings.TrimPrefix(params, "sig1="))
	computed, err := NewCanonicalizer().BuildSignatureBase(request, "sig1", parsed["sig1"])
	if err != nil || computed != base {
		t.Fatalf("received parameter order changed: %v\n%s", err, computed)
	}
	request.Header.Set("Signature", "sig1=:"+base64.StdEncoding.EncodeToString(ed25519.Sign(private, []byte(base)))+":")
	verifier := NewHTTPVerifier()
	defer verifier.Close()
	opts := StrictHTTPVerificationOptions()
	opts.ExpectedDID = did
	opts.ExpectedKeyID = did + "#signing-1"
	opts.ExpectedAuthorities = []string{"agent.example"}
	if err := verifier.VerifyRequest(request, public, opts); err != nil {
		t.Fatalf("valid signature with reordered parameters failed: %v", err)
	}

	changed := request.Clone(request.Context())
	changed.Header = request.Header.Clone()
	changed.Header.Set("Signature-Input", strings.Replace(params,
		`;tag="sage-0.10.0";created=`, `;created=`, 1)+`;tag="sage-0.10.0"`)
	second := NewHTTPVerifier()
	defer second.Close()
	if err := second.VerifyRequest(changed, public, opts); err == nil {
		t.Fatal("changed parameter order retained signature authority")
	}
}

func TestSignRequestIncludesTag(t *testing.T) {
	seed := sha256.Sum256([]byte("sage HTTP tag signer test"))
	private := ed25519.NewKeyFromSeed(seed[:])
	request, err := http.NewRequest(http.MethodGet, "https://agent.example/", nil)
	if err != nil {
		t.Fatal(err)
	}
	params := &SignatureInputParams{
		CoveredComponents: []string{`"@method"`}, KeyID: "key-1",
		Algorithm: "ed25519", Created: time.Now().Unix(), Tag: "sage-0.10.0",
	}
	verifier := NewHTTPVerifier()
	defer verifier.Close()
	if err := verifier.SignRequest(request, "sig1", params, private); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(request.Header.Get("Signature-Input"), `;tag="sage-0.10.0"`) {
		t.Fatal("signed request omitted tag")
	}
	if err := verifier.VerifyRequest(request, private.Public(), nil); err != nil {
		t.Fatalf("signed tag was not verified: %v", err)
	}
}
