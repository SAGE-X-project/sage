package registry010

import (
	"bytes"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"strings"

	"filippo.io/edwards25519"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

type webProofKey010 struct {
	Name  string `json:"name"`
	Alg   string `json:"alg"`
	Key   string `json:"key"`
	Proof struct {
		Signer string `json:"signer"`
		Value  string `json:"value"`
	} `json:"proof"`
}

// CheckWebRegistryProofs010 verifies every REG-04 signing proof and X25519
// endorsement in a structurally valid web record. Historical KEM signers may
// be revoked or expired on a read; their earlier authority still requires an
// authenticated source history. This function does not authenticate the HTTPS
// origin, controller, mutation history, or record freshness beyond the wrapper.
// Success must never authorize a protected operation by itself.
func CheckWebRegistryProofs010(raw []byte, expectedDID string, now int64) error {
	return checkWebRegistryProofs010(raw, expectedDID, now, true)
}

func checkWebRegistryProofs010(raw []byte, expectedDID string, now int64, requireUsableSigning bool) error {
	if err := checkWebRegistryRecordShape010(raw, expectedDID, now, requireUsableSigning); err != nil {
		return err
	}
	var response struct {
		Record struct {
			Keys []webProofKey010 `json:"keys"`
		} `json:"record"`
	}
	if json.Unmarshal(raw, &response) != nil {
		return ErrInvalidRecord010
	}
	registryID, agentID, ok := webProofIDs010(expectedDID)
	if !ok {
		return ErrInvalidRecord010
	}
	signing := make(map[string]webProofKey010, len(response.Record.Keys))
	for _, key := range response.Record.Keys {
		if key.Alg == "x25519" {
			continue
		}
		material, err := base64.RawURLEncoding.DecodeString(key.Key)
		if err != nil || !webValidSigningPoint010(key.Alg, material) {
			return ErrInvalidRecord010
		}
		challenge, err := PoPChallenge010(registryID, agentID, key.Name, key.Alg, material)
		if err != nil {
			return ErrInvalidRecord010
		}
		proof, err := base64.RawURLEncoding.DecodeString(key.Proof.Value)
		if err != nil || !webVerifySignature010(key.Alg, material, challenge, proof) {
			return ErrInvalidRecord010
		}
		signing[key.Name] = key
	}
	for _, key := range response.Record.Keys {
		if key.Alg != "x25519" {
			continue
		}
		material, err := base64.RawURLEncoding.DecodeString(key.Key)
		if err != nil || !webValidX25519Point010(material) {
			return ErrInvalidRecord010
		}
		signerName := strings.TrimPrefix(key.Proof.Signer, expectedDID+"#")
		signer, found := signing[signerName]
		if !found {
			return ErrInvalidRecord010
		}
		signerMaterial, err := base64.RawURLEncoding.DecodeString(signer.Key)
		if err != nil {
			return ErrInvalidRecord010
		}
		challenge, err := PoPChallenge010(registryID, agentID, key.Name, key.Alg, material)
		if err != nil {
			return ErrInvalidRecord010
		}
		proof, err := base64.RawURLEncoding.DecodeString(key.Proof.Value)
		if err != nil || !webVerifySignature010(signer.Alg, signerMaterial, challenge, proof) {
			return ErrInvalidRecord010
		}
	}
	return nil
}

func webProofIDs010(did string) (string, string, bool) {
	rest := strings.TrimPrefix(did, "did:sage:")
	index := strings.LastIndexByte(rest, ':')
	if index <= 0 || index == len(rest)-1 {
		return "", "", false
	}
	return rest[:index], rest[index+1:], true
}

func webValidSigningPoint010(alg string, material []byte) bool {
	switch alg {
	case "ed25519":
		return webPrimeEdPoint010(material)
	case "ecdsa-p256-sha256":
		_, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), material)
		return err == nil
	case "sage-secp256k1-keccak256":
		_, err := ethcrypto.UnmarshalPubkey(material)
		return err == nil
	default:
		return false
	}
}

func webPrimeEdPoint010(encoded []byte) bool {
	if len(encoded) != ed25519.PublicKeySize {
		return false
	}
	point, err := new(edwards25519.Point).SetBytes(encoded)
	if err != nil || !bytes.Equal(point.Bytes(), encoded) || point.Equal(edwards25519.NewIdentityPoint()) == 1 {
		return false
	}
	// [L]P must be identity. This also rejects mixed-torsion points that a
	// small-order-only check would leave in the input domain.
	order, _ := new(big.Int).SetString("7237005577332262213973186563042994240857116359379907606001950938285454250989", 10)
	accumulator := edwards25519.NewIdentityPoint()
	addend := point
	for i := 0; i < order.BitLen(); i++ {
		if order.Bit(i) == 1 {
			accumulator = new(edwards25519.Point).Add(accumulator, addend)
		}
		addend = new(edwards25519.Point).Add(addend, addend)
	}
	return accumulator.Equal(edwards25519.NewIdentityPoint()) == 1
}

func webValidX25519Point010(material []byte) bool {
	curve := ecdh.X25519()
	public, err := curve.NewPublicKey(material)
	if err != nil {
		return false
	}
	privateBytes := make([]byte, 32)
	privateBytes[0] = 1
	private, err := curve.NewPrivateKey(privateBytes)
	if err != nil {
		return false
	}
	_, err = private.ECDH(public)
	return err == nil
}

func webVerifySignature010(alg string, material, challenge, signature []byte) bool {
	switch alg {
	case "ed25519":
		if len(signature) != ed25519.SignatureSize || !webPrimeEdPoint010(signature[:32]) {
			return false
		}
		if _, err := new(edwards25519.Scalar).SetCanonicalBytes(signature[32:]); err != nil {
			return false
		}
		return ed25519.Verify(ed25519.PublicKey(material), challenge, signature)
	case "ecdsa-p256-sha256":
		if len(signature) != 64 {
			return false
		}
		curve := elliptic.P256()
		public, err := ecdsa.ParseUncompressedPublicKey(curve, material)
		if err != nil || !webLowS010(signature, curve.Params().N) {
			return false
		}
		digest := sha256.Sum256(challenge)
		return ecdsa.Verify(public, digest[:], new(big.Int).SetBytes(signature[:32]), new(big.Int).SetBytes(signature[32:]))
	case "sage-secp256k1-keccak256":
		if len(signature) != 65 || signature[64] > 1 || !webLowS010(signature[:64], ethcrypto.S256().Params().N) {
			return false
		}
		digest := ethcrypto.Keccak256(challenge)
		if !ethcrypto.VerifySignature(material, digest, signature[:64]) {
			return false
		}
		recovered, err := ethcrypto.SigToPub(digest, signature)
		return err == nil && bytes.Equal(ethcrypto.FromECDSAPub(recovered), material)
	default:
		return false
	}
}

func webLowS010(signature []byte, order *big.Int) bool {
	if len(signature) != 64 || order == nil {
		return false
	}
	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])
	return r.Sign() > 0 && r.Cmp(order) < 0 && s.Sign() > 0 && s.Cmp(new(big.Int).Rsh(new(big.Int).Set(order), 1)) <= 0
}
