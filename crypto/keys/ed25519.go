package keys

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"

	sagecrypto "github.com/sage-x-project/sage/crypto"
)

// ed25519KeyPair implements the KeyPair interface for Ed25519 keys
type ed25519KeyPair struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	id         string
}

// GenerateEd25519KeyPair generates a new Ed25519 key pair
func GenerateEd25519KeyPair() (sagecrypto.KeyPair, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	// Generate ID from public key hash
	hash := sha256.Sum256(publicKey)
	id := hex.EncodeToString(hash[:8])

	return &ed25519KeyPair{
		privateKey: privateKey,
		publicKey:  publicKey,
		id:         id,
	}, nil
}

// PublicKey returns the public key
func (kp *ed25519KeyPair) PublicKey() crypto.PublicKey {
	return kp.publicKey
}

// PrivateKey returns the private key
func (kp *ed25519KeyPair) PrivateKey() crypto.PrivateKey {
	return kp.privateKey
}

// Type returns the key type
func (kp *ed25519KeyPair) Type() sagecrypto.KeyType {
	return sagecrypto.KeyTypeEd25519
}

// Sign signs the given message
func (kp *ed25519KeyPair) Sign(message []byte) ([]byte, error) {
	signature := ed25519.Sign(kp.privateKey, message)
	return signature, nil
}

// Verify verifies the signature
func (kp *ed25519KeyPair) Verify(message, signature []byte) error {
	if !ed25519.Verify(kp.publicKey, message, signature) {
		return sagecrypto.ErrInvalidSignature
	}
	return nil
}

// ID returns a unique identifier for this key pair
func (kp *ed25519KeyPair) ID() string {
	return kp.id
}