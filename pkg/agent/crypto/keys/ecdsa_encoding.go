package keys

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"errors"
	"fmt"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// Fixed-length encodings for ECDSA keys. Go 1.26 deprecates direct use of the
// D, X and Y fields; the standard library offers Bytes()/ParseRawPrivateKey for
// NIST curves only, so secp256k1 goes through go-ethereum's encoders. All
// results are left-padded to the curve size, which is also what JWK (RFC 7518)
// and the Ethereum address derivation require.

// ECDSAPrivateScalar returns the private scalar d, left-padded to the curve size.
func ECDSAPrivateScalar(priv *ecdsa.PrivateKey) ([]byte, error) {
	if priv == nil {
		return nil, errors.New("nil private key")
	}
	if IsSecp256k1Curve(priv.Curve) {
		return ethcrypto.FromECDSA(priv), nil
	}
	return priv.Bytes()
}

// ECDSAPublicUncompressed returns the SEC 1 uncompressed point 0x04 || X || Y.
func ECDSAPublicUncompressed(pub *ecdsa.PublicKey) ([]byte, error) {
	if pub == nil {
		return nil, errors.New("nil public key")
	}
	if IsSecp256k1Curve(pub.Curve) {
		return ethcrypto.FromECDSAPub(pub), nil
	}
	return pub.Bytes()
}

// ECDSAPublicCoordinates returns the affine coordinates X and Y, each
// left-padded to the curve size.
func ECDSAPublicCoordinates(pub *ecdsa.PublicKey) (x, y []byte, err error) {
	raw, err := ECDSAPublicUncompressed(pub)
	if err != nil {
		return nil, nil, err
	}
	if len(raw) < 3 || raw[0] != 0x04 || (len(raw)-1)%2 != 0 {
		return nil, nil, fmt.Errorf("unexpected public key encoding (%d bytes)", len(raw))
	}
	n := (len(raw) - 1) / 2
	return raw[1 : 1+n], raw[1+n:], nil
}

// ParseP256PrivateKey builds a P-256 private key from its raw scalar.
func ParseP256PrivateKey(d []byte) (*ecdsa.PrivateKey, error) {
	return ecdsa.ParseRawPrivateKey(elliptic.P256(), d)
}

// ParseECDSAPublicKey builds a public key on curve from affine coordinates of
// any length (they are left-padded to the curve size) and validates that the
// point is on the curve.
func ParseECDSAPublicKey(curve elliptic.Curve, x, y []byte) (*ecdsa.PublicKey, error) {
	if curve == nil {
		return nil, errors.New("nil curve")
	}
	size := (curve.Params().BitSize + 7) / 8
	if len(x) > size || len(y) > size {
		return nil, fmt.Errorf("coordinate longer than curve size (%d bytes)", size)
	}
	raw := make([]byte, 1+2*size)
	raw[0] = 0x04
	copy(raw[1+size-len(x):1+size], x)
	copy(raw[1+2*size-len(y):], y)
	if IsSecp256k1Curve(curve) {
		return ethcrypto.UnmarshalPubkey(raw)
	}
	return ecdsa.ParseUncompressedPublicKey(curve, raw)
}
