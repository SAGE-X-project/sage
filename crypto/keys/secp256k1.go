package keys

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"math/big"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	sagecrypto "github.com/sage-x-project/sage/crypto"
)

// secp256k1KeyPair implements the KeyPair interface for Secp256k1 keys
type secp256k1KeyPair struct {
	privateKey *secp256k1.PrivateKey
	publicKey  *secp256k1.PublicKey
	id         string
}

// GenerateSecp256k1KeyPair generates a new Secp256k1 key pair
func GenerateSecp256k1KeyPair() (sagecrypto.KeyPair, error) {
	privateKey, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}

	publicKey := privateKey.PubKey()

	// Generate ID from public key hash
	pubKeyBytes := publicKey.SerializeCompressed()
	hash := sha256.Sum256(pubKeyBytes)
	id := hex.EncodeToString(hash[:8])

	return &secp256k1KeyPair{
		privateKey: privateKey,
		publicKey:  publicKey,
		id:         id,
	}, nil
}

// PublicKey returns the public key
func (kp *secp256k1KeyPair) PublicKey() crypto.PublicKey {
	return kp.publicKey.ToECDSA()
}

// PrivateKey returns the private key
func (kp *secp256k1KeyPair) PrivateKey() crypto.PrivateKey {
	return kp.privateKey.ToECDSA()
}

// Type returns the key type
func (kp *secp256k1KeyPair) Type() sagecrypto.KeyType {
	return sagecrypto.KeyTypeSecp256k1
}

// Sign signs the given message
func (kp *secp256k1KeyPair) Sign(message []byte) ([]byte, error) {
	// Hash the message using SHA256
	hash := sha256.Sum256(message)
	
	// Sign the hash
	r, s, err := ecdsa.Sign(rand.Reader, kp.privateKey.ToECDSA(), hash[:])
	if err != nil {
		return nil, err
	}
	
	// Serialize the signature
	return serializeSignature(r, s), nil
}

// Verify verifies the signature
func (kp *secp256k1KeyPair) Verify(message, signature []byte) error {
	// Hash the message using SHA256
	hash := sha256.Sum256(message)
	
	// Deserialize the signature
	r, s, err := deserializeSignature(signature)
	if err != nil {
		return sagecrypto.ErrInvalidSignature
	}
	
	// Verify the signature
	verified := ecdsa.Verify(kp.publicKey.ToECDSA(), hash[:], r, s)
	if !verified {
		return sagecrypto.ErrInvalidSignature
	}
	
	return nil
}

// ID returns a unique identifier for this key pair
func (kp *secp256k1KeyPair) ID() string {
	return kp.id
}

// serializeSignature serializes an ECDSA signature
func serializeSignature(r, s *big.Int) []byte {
	// Ensure r and s are 32 bytes each
	rBytes := r.Bytes()
	sBytes := s.Bytes()
	
	signature := make([]byte, 64)
	
	// Pad with zeros if necessary
	copy(signature[32-len(rBytes):32], rBytes)
	copy(signature[64-len(sBytes):64], sBytes)
	
	return signature
}

// deserializeSignature deserializes an ECDSA signature
func deserializeSignature(data []byte) (*big.Int, *big.Int, error) {
	if len(data) != 64 {
		return nil, nil, sagecrypto.ErrInvalidSignature
	}
	
	r := new(big.Int).SetBytes(data[:32])
	s := new(big.Int).SetBytes(data[32:])
	
	return r, s, nil
}