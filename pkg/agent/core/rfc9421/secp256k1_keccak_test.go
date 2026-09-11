package rfc9421

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"net/http"
	"testing"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
)

// A signature produced by KeyPair.Sign (Ethereum convention: Keccak-256,
// r||s||v) must verify on the envelope path under every secp256k1 algorithm
// identifier, and a signature over SHA-256 must not.
func TestEnvelope_Secp256k1UsesKeccak(t *testing.T) {
	kp, err := keys.GenerateSecp256k1KeyPair()
	require.NoError(t, err)
	pub := kp.PublicKey().(*ecdsa.PublicKey)

	for _, alg := range []string{string(AlgorithmES256K), string(AlgorithmECDSASecp256k1), string(AlgorithmECDSA)} {
		v := NewVerifier()
		msg := &Message{
			AgentDID: "did:sage:ethereum:0xabc", MessageID: "m-" + alg, Timestamp: time.Now(), Nonce: "n-" + alg,
			Headers: map[string]string{"content-type": "application/json"}, Body: []byte(`{"x":1}`),
			Algorithm: alg, KeyID: "k1", SignedFields: []string{"agent_did", "message_id", "timestamp", "nonce", "body"},
		}
		base := v.ConstructSignatureBase(msg)
		msg.Signature, err = kp.Sign([]byte(base))
		require.NoError(t, err)
		require.NoError(t, v.VerifySignature(pub, msg, nil), "alg %s", alg)

		// Same bytes signed under SHA-256 (the old convention) must be rejected.
		priv := kp.PrivateKey().(*ecdsa.PrivateKey)
		wrong, err := ethcrypto.Sign(sha256Sum([]byte(base)), priv)
		require.NoError(t, err)
		msg.Signature = wrong
		msg.Nonce = "n2-" + alg
		require.Error(t, v.VerifySignature(pub, msg, nil))
	}
}

// The HTTP path must produce Ethereum-recoverable signatures for secp256k1
// keys and verify them with Keccak-256.
func TestHTTP_Secp256k1SignaturesAreEthereumRecoverable(t *testing.T) {
	kp, err := keys.GenerateSecp256k1KeyPair()
	require.NoError(t, err)
	priv := kp.PrivateKey().(*ecdsa.PrivateKey)
	pub := kp.PublicKey().(*ecdsa.PublicKey)

	v := NewHTTPVerifier()
	defer v.Close()
	body := `{"amount":1}`
	req, err := http.NewRequest(http.MethodPost, "https://agent.example/v1/a2a", bytes.NewBufferString(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Digest", ComputeContentDigest([]byte(body)))
	params := &SignatureInputParams{
		CoveredComponents: []string{`"@method"`, `"@target-uri"`, `"@authority"`, `"content-type"`, `"content-digest"`},
		KeyID:             "did:sage:ethereum:0xabc#key-1", Algorithm: "es256k", Created: time.Now().Unix(), Nonce: "n1",
	}
	require.NoError(t, v.SignRequest(req, "sig1", params, priv))
	require.NoError(t, v.VerifyRequest(req, pub, StrictHTTPVerificationOptions()))

	// Recover the signer's Ethereum address from the wire signature, as a contract would.
	sigs, err := ParseSignature(req.Header.Get("Signature"))
	require.NoError(t, err)
	base, err := v.canonicalizer.BuildSignatureBase(req, "sig1", params)
	require.NoError(t, err)
	recovered, err := ethcrypto.SigToPub(ethcrypto.Keccak256([]byte(base)), sigs["sig1"])
	require.NoError(t, err)
	require.Equal(t, ethcrypto.PubkeyToAddress(*pub), ethcrypto.PubkeyToAddress(*recovered))

	// KeyPair.Sign over the same base is interchangeable with SignRequest.
	direct, err := kp.Sign([]byte(base))
	require.NoError(t, err)
	require.Equal(t, sigs["sig1"], direct)
}

func sha256Sum(b []byte) []byte {
	h := sha256.Sum256(b)
	return h[:]
}
