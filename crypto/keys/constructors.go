package keys

import (
	"crypto"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	sagecrypto "github.com/sage-x-project/sage/crypto"
)

// NewEd25519KeyPair creates a new Ed25519 key pair from an existing private key
func NewEd25519KeyPair(privateKey ed25519.PrivateKey, id string) (sagecrypto.KeyPair, error) {
	publicKey := privateKey.Public().(ed25519.PublicKey)
	
	// Use provided ID or generate from public key
	if id == "" {
		hash := sha256.Sum256(publicKey)
		id = hex.EncodeToString(hash[:8])
	}
	
	return &ed25519KeyPair{
		privateKey: privateKey,
		publicKey:  publicKey,
		id:         id,
	}, nil
}

// NewSecp256k1KeyPair creates a new Secp256k1 key pair from an existing private key
func NewSecp256k1KeyPair(privateKey *secp256k1.PrivateKey, id string) (sagecrypto.KeyPair, error) {
	publicKey := privateKey.PubKey()
	
	// Use provided ID or generate from public key
	if id == "" {
		pubKeyBytes := publicKey.SerializeCompressed()
		hash := sha256.Sum256(pubKeyBytes)
		id = hex.EncodeToString(hash[:8])
	}
	
	return &secp256k1KeyPair{
		privateKey: privateKey,
		publicKey:  publicKey,
		id:         id,
	}, nil
}

// PublicKeyOnlyEd25519 wraps an Ed25519 public key for verification only
type publicKeyOnlyEd25519 struct {
	publicKey ed25519.PublicKey
	id        string
}

func (pk *publicKeyOnlyEd25519) PublicKey() crypto.PublicKey {
	return pk.publicKey
}

func (pk *publicKeyOnlyEd25519) PrivateKey() crypto.PrivateKey {
	return nil
}

func (pk *publicKeyOnlyEd25519) Type() sagecrypto.KeyType {
	return sagecrypto.KeyTypeEd25519
}

func (pk *publicKeyOnlyEd25519) Sign(message []byte) ([]byte, error) {
	return nil, errors.New("cannot sign with public key only")
}

func (pk *publicKeyOnlyEd25519) Verify(message, signature []byte) error {
	if !ed25519.Verify(pk.publicKey, message, signature) {
		return sagecrypto.ErrInvalidSignature
	}
	return nil
}

func (pk *publicKeyOnlyEd25519) ID() string {
	return pk.id
}