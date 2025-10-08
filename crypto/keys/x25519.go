// SAGE - Secure Agent Guarantee Engine
// Copyright (C) 2025 SAGE-X-project
//
// This file is part of SAGE.
//
// SAGE is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// SAGE is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with SAGE. If not, see <https://www.gnu.org/licenses/>.


package keys

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"

	"filippo.io/edwards25519"
	sagecrypto "github.com/sage-x-project/sage/crypto"
	"golang.org/x/crypto/hkdf"

	"github.com/cloudflare/circl/hpke"
)

// X25519KeyPair holds an X25519 private key and its corresponding public key bytes.
type X25519KeyPair struct {
	privateKey *ecdh.PrivateKey
	publicKey  *ecdh.PublicKey
	id      string
}

// GenerateX25519KeyPair generates a new ephemeral X25519 key pair.
// It returns an X25519KeyPair containing the private key and the public key bytes.
func GenerateX25519KeyPair() (sagecrypto.KeyPair, error) {
	privateKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ephemeral ECDH key: %w", err)
	}
	publicKey := privateKey.PublicKey()

	// Generate ID from public key hash
	pubKeyBytes := publicKey.Bytes()
	hash := sha256.Sum256(pubKeyBytes)
	id := hex.EncodeToString(hash[:8])

	return &X25519KeyPair{
		privateKey: privateKey, 
		publicKey: publicKey,
		id: id,
	}, nil
}

// PublicKey returns the public key
func (kp *X25519KeyPair) PublicKey() crypto.PublicKey {
	return kp.publicKey
}

// PublicBytesKey returns the public bytes key
func (kp *X25519KeyPair) PublicBytesKey() []byte {
	return kp.publicKey.Bytes()
}

// PrivateKey returns the private key
func (kp *X25519KeyPair) PrivateKey() crypto.PrivateKey {
	return kp.privateKey
}

// Type returns the key type
func (kp *X25519KeyPair) Type() sagecrypto.KeyType {
	return sagecrypto.KeyTypeX25519
}

// ID returns a unique identifier for this key pair
func (kp *X25519KeyPair) ID() string {
	return kp.id
}

// Sign returns an error as X25519 is a key agreement algorithm and does not support signing operations.
// X25519 keys are designed exclusively for Elliptic Curve Diffie-Hellman (ECDH) key exchange.
// For digital signatures, use Ed25519 keys instead.
func (kp *X25519KeyPair) Sign(message []byte) ([]byte, error) {
    return nil, sagecrypto.ErrSignNotSupported
}

// Verify returns an error as X25519 is a key agreement algorithm and does not support signature verification.
// X25519 keys are designed exclusively for Elliptic Curve Diffie-Hellman (ECDH) key exchange.
// For signature verification, use Ed25519 keys instead
func (kp *X25519KeyPair) Verify(message, signature []byte) error {
    return sagecrypto.ErrVerifyNotSupported
}


// DeriveSharedSecret computes a 32-byte session key from an X25519 ECDH exchange.
// Given our private key and peer's public key bytes, it returns
// SHA-256 of the raw 32-byte ECDH shared secret.
func (kp *X25519KeyPair) DeriveSharedSecret(peerPubBytes []byte) ([]byte, error) {
	curve := ecdh.X25519()
	peerPub, err := curve.NewPublicKey(peerPubBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse peer public key: %w", err)
	}

	shared, err := kp.privateKey.ECDH(peerPub)
	if err != nil {
		return nil, fmt.Errorf("failed to compute shared secret: %w", err)
	}

	sum := sha256.Sum256(shared)
	return sum[:], nil
}

// Encrypt performs ECIES-like encryption using X25519 ECDH.
// It generates an ephemeral key pair, derives a shared key with recipientPub,
// and encrypts plaintext using AES-256-GCM.
// Returns ephemeral public key, random nonce, and ciphertext.
func (kp *X25519KeyPair) Encrypt(recipientPub []byte, plaintext []byte) (nonce, ciphertext []byte, err error) {
	key, err := kp.DeriveSharedSecret(recipientPub)
	if err != nil {
		return nil, nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	ciphertext = aead.Seal(nil, nonce, plaintext, nil)
	return nonce, ciphertext, nil
}

// DecryptWithX25519 decrypts data produced by EncryptWithX25519.
// It takes recipient's private key, ephemeral public key, nonce, and ciphertext.
// Returns the original plaintext or an error on failure.
func (kp *X25519KeyPair) DecryptWithX25519(ephPub, nonce, ciphertext []byte) ([]byte, error) {
	key, err := kp.DeriveSharedSecret(ephPub)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	pt, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return pt, nil
}

// EncryptWithEd25519Peer performs an Ephemeral-Static encryption using Ed25519 keys.
// It converts peer's Ed25519 public key to X25519, does ECDH, runs HKDF, and AES-GCM.
// Returns payload = ephPub||nonce||ciphertext.
func EncryptWithEd25519Peer(edPeerPub crypto.PublicKey, plaintext []byte) ([]byte, error) {
	kp, err := GenerateX25519KeyPair()
	if err != nil {
		return nil, err
	}
	
	peerX, err := convertEd25519PubToX25519(edPeerPub)
	if err != nil {
		return nil, err
	}

	peerPubKey, err := ecdh.X25519().NewPublicKey(peerX)
	if err != nil {
		return nil, err
	}

	privKey, ok := kp.PrivateKey().(*ecdh.PrivateKey)
	if !ok {
		return nil, err
	}

	raw, err := sharedSecret(privKey.ECDH(peerPubKey))
	if err != nil {
		return nil, err
	}

	pubKey, ok := kp.PublicKey().(*ecdh.PublicKey)
	if !ok {
		return nil, err
	}
	transcript := appendPrefix(pubKey.Bytes(), peerX)
	key, err := deriveHKDFKey(raw, transcript)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ct := aead.Seal(nil, nonce, plaintext, transcript)

	return appendPrefix(pubKey.Bytes(), nonce, ct), nil
}

// DecryptWithEd25519Peer reverses EncryptToEd25519Peer.
// It parses ephPub||nonce||ciphertext, converts own Ed25519 key to X25519, does ECDH,
// runs HKDF and opens AES-GCM.
func DecryptWithEd25519Peer(privateKey crypto.PrivateKey, packet []byte) ([]byte, error) {
	ePubLen := 32
	if len(packet) < ePubLen+12 {
		return nil, fmt.Errorf("packet too short")
	}
	ePubBytes := packet[:ePubLen]
	nonce := packet[ePubLen : ePubLen+12]
	ct := packet[ePubLen+12:]

	ePubKey, err := ecdh.X25519().NewPublicKey(ePubBytes)
    if err != nil {
        return nil, fmt.Errorf("invalid ephemeral public key: %w", err)
    }

	selfXPrivBytes, err := convertEd25519PrivToX25519(privateKey)
	if err != nil {
		return nil, err
	}

	selfXPrivKey, err := ecdh.X25519().NewPrivateKey(selfXPrivBytes)
    if err != nil {
        return nil, err
    }

	raw, err := sharedSecret(selfXPrivKey.ECDH(ePubKey))
	if err != nil {
		return nil, err
	}

	selfXPub := selfXPrivKey.PublicKey() 
	transcript := appendPrefix(ePubBytes, selfXPub.Bytes())
	key, err := deriveHKDFKey(raw, transcript)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return aead.Open(nil, nonce, ct, transcript)
}

// deriveHKDFKey derives a 32-byte AES key using HKDF-SHA256.
// The transcript is used as both salt and info string.
// deriveHKDFKey  ➜  raw DH → HKDF-SHA-256(salt = transcript) → 32B AES key
func deriveHKDFKey(raw, transcript []byte) ([]byte, error) {
	h := hkdf.New(sha256.New, raw, transcript, []byte("Noise-IK-AES256GCM"))
	key := make([]byte, 32)
	if _, err := io.ReadFull(h, key); err != nil {
		return nil, fmt.Errorf("hkdf: %w", err)
	}
	return key, nil
}


// convertEd25519PrivToX25519 turns an Ed25519 private key into the X25519 scalar.
func convertEd25519PrivToX25519(privKey crypto.PrivateKey) ([]byte, error) {
	edPriv, ok := privKey.(ed25519.PrivateKey)
    if !ok {
        return nil, fmt.Errorf("expected ed25519.PrivateKey, got %T", privKey)
    }
	
	if l := len(edPriv); l != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("bad Ed25519 priv length: %d", l)
	}
	seed := edPriv.Seed()          // 32-byte seed
	h := sha512.Sum512(seed)       // RFC8032 §5.1.5
	h[0] &= 248
	h[31] &= 127
	h[31] |= 64

	var xPriv [32]byte
	copy(xPriv[:], h[:32])
	return xPriv[:], nil
}

// convertEd25519PubToX25519 turns an Ed25519 public key into the X25519 public key.
func convertEd25519PubToX25519(pubKey crypto.PublicKey) ([]byte, error) {
	edPub, ok := pubKey.(ed25519.PublicKey)
    if !ok {
        return nil, fmt.Errorf("expected ed25519.PublicKey, got %T", pubKey)
    }

	if l := len(edPub); l != ed25519.PublicKeySize {
		return nil, fmt.Errorf("bad Ed25519 pub length: %d", l)
	}
	// Decompress Ed25519 point
	P, err := new(edwards25519.Point).SetBytes(edPub)
	if err != nil {
		return nil, fmt.Errorf("invalid Ed25519 pub: %w", err)
	}
	return P.BytesMontgomery(), nil
}

func sharedSecret(dh []byte, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	var zero [32]byte
	if subtle.ConstantTimeCompare(dh, zero[:]) == 1 {
		return nil, fmt.Errorf("x25519: low-order or identity point")
	}
	return dh, nil
}

// appendPrefix concatenates multiple byte slices into one.
func appendPrefix(parts ...[]byte) []byte {
	return bytes.Join(parts, nil)
}

// HPKEDeriveSharedSecretToX25519Peer establishes an HPKE Base context to the recipient's
// X25519 public key and returns (enc, exporterSecret). The exporterSecret is suitable
// to feed into your session layer as a sessionSeed or shared secret.
// Both parties MUST use identical 'info' and 'exportCtx' to derive the same bytes.
func HPKEDeriveSharedSecretToX25519Peer(
    peer *ecdh.PublicKey,
    info []byte,
    exportCtx []byte,
    exportLen int,
) (enc []byte, exporterSecret []byte, err error) {
    suite := hpke.NewSuite(
        hpke.KEM_X25519_HKDF_SHA256,
        hpke.KDF_HKDF_SHA256,
        hpke.AEAD_ChaCha20Poly1305,
    )

    kem := hpke.KEM_X25519_HKDF_SHA256.Scheme()
    rp, err := kem.UnmarshalBinaryPublicKey(peer.Bytes())
    if err != nil {
        return nil, nil, fmt.Errorf("hpke unmarshal pub: %w", err)
    }

    sender, err := suite.NewSender(rp, info)
    if err != nil {
        return nil, nil, fmt.Errorf("hpke new sender: %w", err)
    }

    enc, sealer, err := sender.Setup(rand.Reader)
    if err != nil {
        return nil, nil, fmt.Errorf("hpke setup: %w", err)
    }

    // Export a shared secret without necessarily encrypting application data.
    secret := sealer.Export(exportCtx, uint(exportLen))
    return enc, secret, nil
}

// HPKEOpenSharedSecretWithX25519Priv takes the recipient's X25519 private key and the 'enc'
// (encapsulated key) from the sender, and reproduces the same exporterSecret.
// 'info' and 'exportCtx' MUST match the sender's values.
func HPKEOpenSharedSecretWithX25519Priv(
    priv *ecdh.PrivateKey,
    enc []byte,
    info []byte,
    exportCtx []byte,
    exportLen int,
) (exporterSecret []byte, err error) {
    suite := hpke.NewSuite(
        hpke.KEM_X25519_HKDF_SHA256,
        hpke.KDF_HKDF_SHA256,
        hpke.AEAD_ChaCha20Poly1305,
    )

    kem := hpke.KEM_X25519_HKDF_SHA256.Scheme()
    skR, err := kem.UnmarshalBinaryPrivateKey(priv.Bytes())
    if err != nil {
        return nil, fmt.Errorf("hpke unmarshal priv: %w", err)
    }

    receiver, err := suite.NewReceiver(skR, info)
    if err != nil {
        return nil, fmt.Errorf("hpke new receiver: %w", err)
    }

    opener, err := receiver.Setup(enc)
    if err != nil {
        return nil, fmt.Errorf("hpke receiver setup: %w", err)
    }

    secret := opener.Export(exportCtx, uint(exportLen))
    return secret, nil
}

// Convenience wrappers that accept crypto.PublicKey / crypto.PrivateKey
// and do type assertions to *ecdh.PublicKey / *ecdh.PrivateKey.

func HPKEDeriveSharedSecretToPeer(
    pub crypto.PublicKey,
    info []byte,
    exportCtx []byte,
    exportLen int,
) (enc []byte, exporterSecret []byte, err error) {
    p, ok := pub.(*ecdh.PublicKey)
    if !ok {
        return nil, nil, fmt.Errorf("expected *ecdh.PublicKey, got %T", pub)
    }

	if p.Curve() != ecdh.X25519() {
		return nil, nil, fmt.Errorf("unsupported KEM curve: want X25519")
	}
    return HPKEDeriveSharedSecretToX25519Peer(p, info, exportCtx, exportLen)
}

func HPKEOpenSharedSecretWithPriv(
    priv crypto.PrivateKey,
    enc []byte,
    info []byte,
    exportCtx []byte,
    exportLen int,
) (exporterSecret []byte, err error) {
    p, ok := priv.(*ecdh.PrivateKey)
    if !ok {
        return nil, fmt.Errorf("expected *ecdh.PrivateKey, got %T", priv)
    }

	if p.Curve() != ecdh.X25519() {
		return nil, fmt.Errorf("unsupported KEM curve: want X25519")
	}
    return HPKEOpenSharedSecretWithX25519Priv(p, enc, info, exportCtx, exportLen)
}

// OPTIONAL: If you still want to encrypt a handshake payload while also deriving the shared secret,
// you can use the seal/open helpers below. They output (packet = enc||ct) and exporterSecret.

func HPKESealAndExportToX25519Peer(
    peer crypto.PublicKey,
    plaintext []byte,
    info []byte,
    exportCtx []byte,
    exportLen int,
) (packet []byte, exporterSecret []byte, err error) {

	pubKey, ok := peer.(*ecdh.PublicKey)
	if !ok {
		return nil, nil, fmt.Errorf("hpke: invalid key type, expected ECDH but got %T", peer)
	}
    suite := hpke.NewSuite(
        hpke.KEM_X25519_HKDF_SHA256,
        hpke.KDF_HKDF_SHA256,
        hpke.AEAD_ChaCha20Poly1305,
    )

    kem := hpke.KEM_X25519_HKDF_SHA256.Scheme()
    rp, err := kem.UnmarshalBinaryPublicKey(pubKey.Bytes())
    if err != nil {
        return nil, nil, fmt.Errorf("hpke unmarshal pub: %w", err)
    }

    sender, err := suite.NewSender(rp, info)
    if err != nil {
        return nil, nil, fmt.Errorf("hpke new sender: %w", err)
    }

    enc, sealer, err := sender.Setup(rand.Reader)
    if err != nil {
        return nil, nil, fmt.Errorf("hpke setup: %w", err)
    }

    ct, err := sealer.Seal(plaintext, info) // aad = info
    if err != nil {
        return nil, nil, fmt.Errorf("hpke seal: %w", err)
    }

    secret := sealer.Export(exportCtx, uint(exportLen))
    return append(append([]byte{}, enc...), ct...), secret, nil
}

func HPKEOpenAndExportWithX25519Priv(
    priv crypto.PrivateKey,
    packet []byte,
    info []byte,
    exportCtx []byte,
    exportLen int,
) (plaintext []byte, exporterSecret []byte, err error) {
	privKey, ok := priv.(*ecdh.PrivateKey)
	if !ok {
		return nil, nil, fmt.Errorf("hpke: invalid key type, expected ECDH but got %T", priv)
	}

    const encLen = 32 // X25519 KEM enc length
    if len(packet) < encLen {
        return nil, nil, fmt.Errorf("packet too short: %d", len(packet))
    }
    enc := packet[:encLen]
    ct := packet[encLen:]

    suite := hpke.NewSuite(
        hpke.KEM_X25519_HKDF_SHA256,
        hpke.KDF_HKDF_SHA256,
        hpke.AEAD_ChaCha20Poly1305,
    )

    kem := hpke.KEM_X25519_HKDF_SHA256.Scheme()
    skR, err := kem.UnmarshalBinaryPrivateKey(privKey.Bytes())
    if err != nil {
        return nil, nil, fmt.Errorf("hpke unmarshal priv: %w", err)
    }

    receiver, err := suite.NewReceiver(skR, info)
    if err != nil {
        return nil, nil, fmt.Errorf("hpke new receiver: %w", err)
    }

    opener, err := receiver.Setup(enc)
    if err != nil {
        return nil, nil, fmt.Errorf("hpke receiver setup: %w", err)
    }

    pt, err := opener.Open(ct, info) // aad = info
    if err != nil {
        return nil, nil, fmt.Errorf("hpke open: %w", err)
    }

    secret := opener.Export(exportCtx, uint(exportLen))
    return pt, secret, nil
}
