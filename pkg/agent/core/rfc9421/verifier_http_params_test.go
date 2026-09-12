package rfc9421

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// A verifier must build the "@signature-params" line from the received
// Signature-Input member, so a conformant signer that orders parameters
// differently or adds tag still verifies (RFC 9421 section 2.3).
func TestVerifyRequest_UsesReceivedSignatureParams(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	req, err := http.NewRequest("GET", "https://example.com/items?x=1", nil)
	require.NoError(t, err)
	req.Header.Set("Date", "Tue, 01 Sep 2026 12:00:00 GMT")

	// Third-party serialisation: tag first, created before keyid, extra unknown parameter.
	member := `("@method" "@request-target" "date");tag="third-party";created=1788609600;keyid="did:sage:ethereum:0xabc#key-1";alg="ed25519";nonce="n-1";x-custom=1`
	base := strings.Join([]string{
		`"@method": GET`,
		`"@request-target": /items?x=1`,
		`"date": Tue, 01 Sep 2026 12:00:00 GMT`,
		`"@signature-params": ` + member,
	}, "\n")
	sig := ed25519.Sign(priv, []byte(base))
	req.Header.Set("Signature-Input", "sig1="+member)
	req.Header.Set("Signature", "sig1=:"+base64.StdEncoding.EncodeToString(sig)+":")

	opts := &HTTPVerificationOptions{SignatureName: "sig1", MaxAge: 0, RequiredComponents: []string{"@method"}}
	v := NewHTTPVerifier()
	require.NoError(t, v.VerifyRequest(req, pub, opts))

	params, err := ParseSignatureInput(req.Header.Get("Signature-Input"))
	require.NoError(t, err)
	require.Equal(t, "third-party", params["sig1"].Tag)
	require.Equal(t, member, params["sig1"].Raw)
}

// Signers serialise in the specification's order and include tag.
func TestFormatSignatureParams_Order(t *testing.T) {
	p := &SignatureInputParams{
		CoveredComponents: []string{`"@method"`},
		KeyID:             "k", Algorithm: "ed25519", Created: 1, Expires: 2, Nonce: "n", Tag: "t",
	}
	require.Equal(t, `("@method");keyid="k";alg="ed25519";created=1;expires=2;nonce="n";tag="t"`, FormatSignatureParams(p))
}
