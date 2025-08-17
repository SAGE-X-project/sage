package formats

import (
	"crypto"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	sagecrypto "github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/keys"
)

// JWK represents a JSON Web Key
type JWK struct {
	Kty string `json:"kty"`           // Key Type
	Crv string `json:"crv,omitempty"` // Curve (for EC and OKP)
	N   string `json:"n,omitempty"`   // Modulus for RSA
    E   string `json:"e,omitempty"`   // Exponent for RSA
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
	
	case sagecrypto.KeyTypeX25519:
        // X25519 key agreement keys
        privKey, ok := keyPair.PrivateKey().(*ecdh.PrivateKey)
        if !ok {
            return nil, errors.New("invalid X25519 private key type")
        }
        pubKey := privKey.Public().(*ecdh.PublicKey)
		
		jwk.Use = "enc"     // key agreement / encryption use
        jwk.Kty = "OKP"
        jwk.Crv = "X25519"
        jwk.X = base64.RawURLEncoding.EncodeToString(pubKey.Bytes())
        jwk.D = base64.RawURLEncoding.EncodeToString(privKey.Bytes())
        jwk.Alg = "ECDH-ES" // or other appropriate alg identifier

	case sagecrypto.KeyTypeRSA:
        priv, ok := keyPair.PrivateKey().(*rsa.PrivateKey)
        if !ok {
            return nil, errors.New("invalid RSA private key")
        }
        jwk.Kty = "RSA" 
		jwk.Alg = "RS256"
        nBytes := priv.N.Bytes()
        eBytes := big.NewInt(int64(priv.E)).Bytes()
        jwk.N = base64.RawURLEncoding.EncodeToString(nBytes)
        jwk.E = base64.RawURLEncoding.EncodeToString(eBytes)
        // include private exponent
        jwk.D = base64.RawURLEncoding.EncodeToString(priv.D.Bytes())
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
	
	case sagecrypto.KeyTypeX25519:
        pubKey, ok := keyPair.PublicKey().(*ecdh.PublicKey)
        if !ok {
            return nil, errors.New("invalid X25519 public key type")
        }

		jwk.Use = "enc"     // key agreement / encryption use
        jwk.Kty = "OKP"
        jwk.Crv = "X25519"
        jwk.X = base64.RawURLEncoding.EncodeToString(pubKey.Bytes())
        jwk.Alg = "ECDH-ES"
		
	case sagecrypto.KeyTypeRSA:
        pub, ok := keyPair.PublicKey().(*rsa.PublicKey)
        if !ok {
            return nil, errors.New("invalid RSA public key")
        }
        jwk.Kty = "RSA" 
		jwk.Alg = "RS256"
        nBytes := pub.N.Bytes()
        eBytes := big.NewInt(int64(pub.E)).Bytes()
        jwk.N = base64.RawURLEncoding.EncodeToString(nBytes)
        jwk.E = base64.RawURLEncoding.EncodeToString(eBytes)

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
		switch jwk.Crv {
		case "Ed25519":
			return i.importEd25519(&jwk)
		case "X25519":
			return i.importX25519(&jwk)
		default:
			return nil, fmt.Errorf("unsupported OKP curve: %s", jwk.Crv)
		}

	case "EC":
		if jwk.Crv != "secp256k1" {
			return nil, fmt.Errorf("unsupported EC curve: %s", jwk.Crv)
		}
		return i.importSecp256k1(&jwk)
	
	case "RSA":
        _, err := base64.RawURLEncoding.DecodeString(jwk.N)
        if err != nil {
            return nil, err
        }
        _, err = base64.RawURLEncoding.DecodeString(jwk.E)
        if err != nil {
            return nil, err
        }
        return i.importRSA(&jwk)

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
		switch jwk.Crv {
        case "Ed25519":
            publicKeyBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
            if err != nil {
                return nil, fmt.Errorf("failed to decode public key: %w", err)
            }
            return ed25519.PublicKey(publicKeyBytes), nil
            
        case "X25519":
            publicKeyBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
            if err != nil {
                return nil, fmt.Errorf("failed to decode X25519 public key: %w", err)
            }
            publicKey, err := ecdh.X25519().NewPublicKey(publicKeyBytes)
            if err != nil {
                return nil, fmt.Errorf("failed to create X25519 public key: %w", err)
            }
            return publicKey, nil

		default:
            return nil, fmt.Errorf("unsupported OKP curve: %s", jwk.Crv)
        }

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
	
	case "RSA":
        nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
        if err != nil {
            return nil, err
        }
        eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
        if err != nil {
            return nil, err
        }
        eInt := new(big.Int).SetBytes(eBytes).Int64()
        return &rsa.PublicKey{
            N: new(big.Int).SetBytes(nBytes),
            E: int(eInt),
        }, nil
		
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

func (i *jwkImporter) importX25519(jwk *JWK) (sagecrypto.KeyPair, error) {
    if jwk.D == "" {
        return nil, errors.New("missing private key component")
    }

    privateKeyBytes, err := base64.RawURLEncoding.DecodeString(jwk.D)
    if err != nil {
        return nil, fmt.Errorf("failed to decode X25519 private key: %w", err)
    }

    privateKey, err := ecdh.X25519().NewPrivateKey(privateKeyBytes)
    if err != nil {
        return nil, fmt.Errorf("failed to create X25519 private key: %w", err)
    }

    return keys.NewX25519KeyPair(privateKey, jwk.Kid)
}

func (i *jwkImporter) importRSA(jwk *JWK) (sagecrypto.KeyPair, error) {
    dBytes, err := base64.RawURLEncoding.DecodeString(jwk.D)
    if err != nil {
        return nil, err
    }
    nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
    if err != nil {
        return nil, err
    }
    eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
    if err != nil {
        return nil, err
    }
    eInt := int(new(big.Int).SetBytes(eBytes).Int64())
    priv := &rsa.PrivateKey{
        PublicKey: rsa.PublicKey{
            N: new(big.Int).SetBytes(nBytes),
            E: eInt,
        },
        D: new(big.Int).SetBytes(dBytes),
        // CRTValues omitted for simplicity; not required for verification
    }
    return keys.NewRSAKeyPair(priv, jwk.Kid)
}

// ComputeKeyIDRFC9421 generate kid based on RFC-9421/7638 thumbprint
func(jwk JWK) ComputeKeyIDRFC9421() (string, error) {
    m := map[string]string{
        "kty": jwk.Kty,
    }
    if jwk.Crv != "" {
        m["crv"] = jwk.Crv
    }
    if jwk.X != "" {
        m["x"] = jwk.X
    }
    if jwk.Y != "" {
        m["y"] = jwk.Y
    }

    keys := make([]string, 0, len(m))
    for k := range m {
        keys = append(keys, k)
    }
    sort.Strings(keys)

    buf := []byte{'{'}
    for i, k := range keys {
        if i > 0 {
            buf = append(buf, ',')
        }
        valueJSON, err := json.Marshal(m[k])
        if err != nil {
            return "", fmt.Errorf("failed to marshal JWK thumbprint value: %w", err)
        }
        buf = append(buf, fmt.Sprintf("%q:%s", k, valueJSON)...)
    }
    buf = append(buf, '}')

    sum := sha256.Sum256(buf)

    kid := base64.RawURLEncoding.EncodeToString(sum[:])
    return kid, nil
}