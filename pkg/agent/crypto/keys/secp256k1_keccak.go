package keys

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"fmt"
	"math/big"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"

	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
)

// secp256k1 signatures in SAGE follow the Ethereum convention everywhere:
// the message is hashed with Keccak-256 and signed with RFC 6979 deterministic
// ECDSA, producing r || s || v (65 bytes). Verifiers accept 64-byte r || s as
// well. This is the single implementation every signing and verification path
// (KeyPair, RFC 9421 envelope and HTTP, HPKE, A2A proofs) must use so that a
// signature made by an Ethereum wallet or verifiable by ecrecover on-chain is
// also valid for SAGE, and vice versa.

// IsSecp256k1Curve reports whether c is the secp256k1 curve. It compares the
// curve parameters rather than the name because go-ethereum's cgo-backed
// curve reports an empty name while decred's reports "secp256k1".
func IsSecp256k1Curve(c elliptic.Curve) bool {
	if c == nil {
		return false
	}
	p, ref := c.Params(), Secp256k1Curve().Params()
	return p != nil && p.P.Cmp(ref.P) == 0 && p.N.Cmp(ref.N) == 0 && p.Gx.Cmp(ref.Gx) == 0
}

// SignSecp256k1Keccak signs Keccak-256(message) with an secp256k1 private key
// and returns the 65-byte Ethereum signature r || s || v.
func SignSecp256k1Keccak(priv *ecdsa.PrivateKey, message []byte) ([]byte, error) {
	if priv == nil || !IsSecp256k1Curve(priv.Curve) {
		return nil, fmt.Errorf("%w: secp256k1 private key required", sagecrypto.ErrInvalidKeyType)
	}
	return ethcrypto.Sign(ethcrypto.Keccak256(message), priv)
}

// VerifySecp256k1Keccak verifies a 64- or 65-byte Ethereum-style signature over
// Keccak-256(message) against an secp256k1 public key.
func VerifySecp256k1Keccak(pub *ecdsa.PublicKey, message, signature []byte) error {
	if pub == nil || !IsSecp256k1Curve(pub.Curve) {
		return fmt.Errorf("%w: secp256k1 public key required", sagecrypto.ErrInvalidKeyType)
	}
	switch len(signature) {
	case 65:
		signature = signature[:64]
	case 64:
	default:
		return sagecrypto.ErrInvalidSignature
	}
	// go-ethereum rejects signatures whose s is in the upper half of the
	// order. SAGE signers emit low-S, but signatures made with crypto/ecdsa or
	// other libraries may not; normalising here keeps them verifiable
	// (verification is unaffected: (r, s) and (r, N-s) are both valid).
	signature = EncodeRawECDSASignature(pub.Curve, new(big.Int).SetBytes(signature[:32]), new(big.Int).SetBytes(signature[32:]))
	if !ethcrypto.VerifySignature(ethcrypto.FromECDSAPub(pub), ethcrypto.Keccak256(message), signature) {
		return sagecrypto.ErrInvalidSignature
	}
	return nil
}
