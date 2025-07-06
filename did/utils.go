package did

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

// MarshalPublicKey converts a public key to bytes for storage
func MarshalPublicKey(publicKey interface{}) ([]byte, error) {
	switch pk := publicKey.(type) {
	case ed25519.PublicKey:
		return pk, nil
	case *secp256k1.PublicKey:
		return pk.SerializeCompressed(), nil
	default:
		// Try to marshal as generic public key using x509
		return x509.MarshalPKIXPublicKey(publicKey)
	}
}

// UnmarshalPublicKey converts bytes back to a public key
func UnmarshalPublicKey(data []byte, keyType string) (interface{}, error) {
	switch keyType {
	case "ed25519":
		if len(data) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("invalid Ed25519 public key size: %d", len(data))
		}
		return ed25519.PublicKey(data), nil
		
	case "secp256k1":
		pk, err := secp256k1.ParsePubKey(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse secp256k1 public key: %w", err)
		}
		return pk, nil
		
	default:
		// Try to unmarshal as generic public key
		block, _ := pem.Decode(data)
		if block != nil {
			data = block.Bytes
		}
		return x509.ParsePKIXPublicKey(data)
	}
}