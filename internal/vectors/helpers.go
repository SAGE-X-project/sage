package vectors

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
)

// Fixed key material. Every private key is derived from a labelled SHA-256
// so the vectors are reproducible from this source alone; the labels are
// part of the published inputs.
const (
	labelEd25519A   = "sage-spec/vectors/ed25519/agent-a"
	labelEd25519B   = "sage-spec/vectors/ed25519/agent-b"
	labelSecp256k1A = "sage-spec/vectors/secp256k1/agent-a"
	labelP256A      = "sage-spec/vectors/p256/agent-a"
	labelX25519A    = "sage-spec/vectors/x25519/agent-a"
	labelX25519B    = "sage-spec/vectors/x25519/agent-b"
)

func seed32(label string) []byte {
	h := sha256.Sum256([]byte(label))
	return h[:]
}

func hx(b []byte) string { return hex.EncodeToString(b) }

func unhex(in map[string]any, key string) ([]byte, error) {
	s, ok := in[key].(string)
	if !ok {
		return nil, fmt.Errorf("%s: missing or not a string", key)
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", key, err)
	}
	return b, nil
}

func str(in map[string]any, key string) (string, error) {
	s, ok := in[key].(string)
	if !ok {
		return "", fmt.Errorf("%s: missing or not a string", key)
	}
	return s, nil
}

func ed25519FromLabel(label string) ed25519.PrivateKey {
	return ed25519.NewKeyFromSeed(seed32(label))
}

func secp256k1FromLabel(label string) (*ecdsa.PrivateKey, error) {
	return ethcrypto.ToECDSA(seed32(label))
}

func p256FromLabel(label string) (*ecdsa.PrivateKey, error) {
	d := new(big.Int).SetBytes(seed32(label))
	d.Mod(d, elliptic.P256().Params().N)
	if d.Sign() == 0 {
		return nil, fmt.Errorf("zero scalar for %s", label)
	}
	scalar := make([]byte, 32)
	d.FillBytes(scalar)
	return keys.ParseP256PrivateKey(scalar)
}

func scalarBytes(priv *ecdsa.PrivateKey) (string, error) {
	b, err := keys.ECDSAPrivateScalar(priv)
	if err != nil {
		return "", err
	}
	return hx(b), nil
}

func p256PublicBytes(pub *ecdsa.PublicKey) []byte {
	return elliptic.Marshal(pub.Curve, pub.X, pub.Y) //nolint:staticcheck // uncompressed SEC1 point is the documented encoding
}

func secp256k1PublicBytes(pub *ecdsa.PublicKey) []byte {
	return ethcrypto.FromECDSAPub(pub)
}

func compressedSecp256k1(pub *ecdsa.PublicKey) []byte {
	return ethcrypto.CompressPubkey(pub)
}

func ethAddress(pub *ecdsa.PublicKey) (string, error) {
	return keys.EthereumAddress(pub)
}

func secp256k1PublicFromBytes(b []byte) (*ecdsa.PublicKey, error) {
	return ethcrypto.UnmarshalPubkey(b)
}

func p256PublicFromBytes(b []byte) (*ecdsa.PublicKey, error) {
	curve := elliptic.P256()
	x, y := elliptic.Unmarshal(curve, b) //nolint:staticcheck // uncompressed SEC1 point is the documented encoding
	if x == nil {
		return nil, fmt.Errorf("invalid P-256 public key")
	}
	return keys.ParseECDSAPublicKey(curve, x.Bytes(), y.Bytes())
}

func isLowS(curve elliptic.Curve, s []byte) bool {
	v := new(big.Int).SetBytes(s)
	return keys.LowS(curve, v).Cmp(v) == 0
}
