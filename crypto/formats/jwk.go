package formats

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	sagecrypto "github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/keys"
)

// JWK represents a JSON Web Key
type JWK struct {
	Kty string `json:"kty"`           // Key Type
	Crv string `json:"crv,omitempty"` // Curve (for EC and OKP)
	X   string `json:"x,omitempty"`   // X coordinate (for EC) or public key (for OKP)
	Y   string `json:"y,omitempty"`   // Y coordinate (for EC)
	D   string `json:"d,omitempty"`   // Private key
	Kid string `json:"kid,omitempty"` // Key ID
	Use string `json:"use,omitempty"` // Key use
	Alg string `json:"alg,omitempty"` // Algorithm
}

// jwkExporter implements KeyExporter for JWK format
type jwkExporter struct{}

// NewJWKExporter creates a new JWK exporter
func NewJWKExporter() sagecrypto.KeyExporter {
	return &jwkExporter{}
}

// Export exports the key pair in JWK format
func (e *jwkExporter) Export(keyPair sagecrypto.KeyPair, format sagecrypto.KeyFormat) ([]byte, error) {
	if format != sagecrypto.KeyFormatJWK {
		return nil, sagecrypto.ErrInvalidKeyFormat
	}

	jwk := &JWK{
		Kid: keyPair.ID(),
		Use: "sig",
	}

	switch keyPair.Type() {
	case sagecrypto.KeyTypeEd25519:
		privateKey, ok := keyPair.PrivateKey().(ed25519.PrivateKey)
		if !ok {
			return nil, errors.New("invalid Ed25519 private key type")
		}
		publicKey := privateKey.Public().(ed25519.PublicKey)

		jwk.Kty = "OKP"
		jwk.Crv = "Ed25519"
		jwk.X = base64.RawURLEncoding.EncodeToString(publicKey)
		jwk.D = base64.RawURLEncoding.EncodeToString(privateKey.Seed())
		jwk.Alg = "EdDSA"

	case sagecrypto.KeyTypeSecp256k1:
		privateKey, ok := keyPair.PrivateKey().(*ecdsa.PrivateKey)
		if !ok {
			return nil, errors.New("invalid Secp256k1 private key type")
		}

		jwk.Kty = "EC"
		jwk.Crv = "secp256k1"
		jwk.X = base64.RawURLEncoding.EncodeToString(privateKey.X.Bytes())
		jwk.Y = base64.RawURLEncoding.EncodeToString(privateKey.Y.Bytes())
		jwk.D = base64.RawURLEncoding.EncodeToString(privateKey.D.Bytes())
		jwk.Alg = "ES256K"

	default:
		return nil, sagecrypto.ErrInvalidKeyType
	}

	return json.Marshal(jwk)
}

// ExportPublic exports only the public key in JWK format
func (e *jwkExporter) ExportPublic(keyPair sagecrypto.KeyPair, format sagecrypto.KeyFormat) ([]byte, error) {
	if format != sagecrypto.KeyFormatJWK {
		return nil, sagecrypto.ErrInvalidKeyFormat
	}

	jwk := &JWK{
		Kid: keyPair.ID(),
		Use: "sig",
	}

	switch keyPair.Type() {
	case sagecrypto.KeyTypeEd25519:
		publicKey, ok := keyPair.PublicKey().(ed25519.PublicKey)
		if !ok {
			return nil, errors.New("invalid Ed25519 public key type")
		}

		jwk.Kty = "OKP"
		jwk.Crv = "Ed25519"
		jwk.X = base64.RawURLEncoding.EncodeToString(publicKey)
		jwk.Alg = "EdDSA"

	case sagecrypto.KeyTypeSecp256k1:
		publicKey, ok := keyPair.PublicKey().(*ecdsa.PublicKey)
		if !ok {
			return nil, errors.New("invalid Secp256k1 public key type")
		}

		jwk.Kty = "EC"
		jwk.Crv = "secp256k1"
		jwk.X = base64.RawURLEncoding.EncodeToString(publicKey.X.Bytes())
		jwk.Y = base64.RawURLEncoding.EncodeToString(publicKey.Y.Bytes())
		jwk.Alg = "ES256K"

	default:
		return nil, sagecrypto.ErrInvalidKeyType
	}

	return json.Marshal(jwk)
}

// jwkImporter implements KeyImporter for JWK format
type jwkImporter struct{}

// NewJWKImporter creates a new JWK importer
func NewJWKImporter() sagecrypto.KeyImporter {
	return &jwkImporter{}
}

// Import imports a key pair from JWK format
func (i *jwkImporter) Import(data []byte, format sagecrypto.KeyFormat) (sagecrypto.KeyPair, error) {
	if format != sagecrypto.KeyFormatJWK {
		return nil, sagecrypto.ErrInvalidKeyFormat
	}

	var jwk JWK
	if err := json.Unmarshal(data, &jwk); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JWK: %w", err)
	}

	switch jwk.Kty {
	case "OKP":
		if jwk.Crv != "Ed25519" {
			return nil, fmt.Errorf("unsupported OKP curve: %s", jwk.Crv)
		}
		return i.importEd25519(&jwk)

	case "EC":
		if jwk.Crv != "secp256k1" {
			return nil, fmt.Errorf("unsupported EC curve: %s", jwk.Crv)
		}
		return i.importSecp256k1(&jwk)

	default:
		return nil, fmt.Errorf("unsupported key type: %s", jwk.Kty)
	}
}

// ImportPublic imports only a public key from JWK format
func (i *jwkImporter) ImportPublic(data []byte, format sagecrypto.KeyFormat) (crypto.PublicKey, error) {
	if format != sagecrypto.KeyFormatJWK {
		return nil, sagecrypto.ErrInvalidKeyFormat
	}

	var jwk JWK
	if err := json.Unmarshal(data, &jwk); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JWK: %w", err)
	}

	switch jwk.Kty {
	case "OKP":
		if jwk.Crv != "Ed25519" {
			return nil, fmt.Errorf("unsupported OKP curve: %s", jwk.Crv)
		}
		publicKeyBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
		if err != nil {
			return nil, fmt.Errorf("failed to decode public key: %w", err)
		}
		return ed25519.PublicKey(publicKeyBytes), nil

	case "EC":
		if jwk.Crv != "secp256k1" {
			return nil, fmt.Errorf("unsupported EC curve: %s", jwk.Crv)
		}
		xBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
		if err != nil {
			return nil, fmt.Errorf("failed to decode X coordinate: %w", err)
		}
		yBytes, err := base64.RawURLEncoding.DecodeString(jwk.Y)
		if err != nil {
			return nil, fmt.Errorf("failed to decode Y coordinate: %w", err)
		}
		
		pubKey := &ecdsa.PublicKey{
			Curve: secp256k1.S256(),
			X:     new(big.Int).SetBytes(xBytes),
			Y:     new(big.Int).SetBytes(yBytes),
		}
		return pubKey, nil

	default:
		return nil, fmt.Errorf("unsupported key type: %s", jwk.Kty)
	}
}

func (i *jwkImporter) importEd25519(jwk *JWK) (sagecrypto.KeyPair, error) {
	if jwk.D == "" {
		return nil, errors.New("missing private key component")
	}

	seedBytes, err := base64.RawURLEncoding.DecodeString(jwk.D)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	privateKey := ed25519.NewKeyFromSeed(seedBytes)
	return keys.NewEd25519KeyPair(privateKey, jwk.Kid)
}

func (i *jwkImporter) importSecp256k1(jwk *JWK) (sagecrypto.KeyPair, error) {
	if jwk.D == "" {
		return nil, errors.New("missing private key component")
	}

	dBytes, err := base64.RawURLEncoding.DecodeString(jwk.D)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	privateKey := secp256k1.PrivKeyFromBytes(dBytes)
	return keys.NewSecp256k1KeyPair(privateKey, jwk.Kid)
}