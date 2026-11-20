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
