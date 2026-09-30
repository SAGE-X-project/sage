package registry010

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"testing"

	"filippo.io/edwards25519"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

const proofDID010 = "did:sage:web:agents.example.com:billing-bot"

func proofFixture010(t *testing.T) map[string]any {
	t.Helper()
	encode := base64.RawURLEncoding.EncodeToString
	edPrivate := ed25519.NewKeyFromSeed(make([]byte, 32))
	edPublic := edPrivate.Public().(ed25519.PublicKey)
	makeEntry := func(name, alg string, pub, sig []byte, signer string) map[string]any {
		return map[string]any{"name": name, "alg": alg, "key": encode(pub),
			"proof": map[string]any{"signer": signer, "value": encode(sig)}, "state": "accepted"}
	}
	challenge := func(name, alg string, pub []byte) []byte {
		message, err := PoPChallenge010("web:agents.example.com", "billing-bot", name, alg, pub)
		if err != nil {
			t.Fatal(err)
		}
		return message
	}
	ed := makeEntry("a-ed", "ed25519", edPublic,
		ed25519.Sign(edPrivate, challenge("a-ed", "ed25519", edPublic)), proofDID010+"#a-ed")
	kemPrivateBytes := make([]byte, 32)
	kemPrivateBytes[0] = 7
	kemPrivate, err := ecdh.X25519().NewPrivateKey(kemPrivateBytes)
	if err != nil {
		t.Fatal(err)
	}
	kemPublic := kemPrivate.PublicKey().Bytes()
	kem := makeEntry("b-kem", "x25519", kemPublic,
		ed25519.Sign(edPrivate, challenge("b-kem", "x25519", kemPublic)), proofDID010+"#a-ed")
	p256Private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	p256Public, err := p256Private.PublicKey.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	p256Digest := sha256.Sum256(challenge("c-p256", "ecdsa-p256-sha256", p256Public))
	r, s, err := ecdsa.Sign(rand.Reader, p256Private, p256Digest[:])
	if err != nil {
		t.Fatal(err)
	}
	if s.Cmp(new(big.Int).Rsh(new(big.Int).Set(elliptic.P256().Params().N), 1)) > 0 {
		s.Sub(elliptic.P256().Params().N, s)
	}
	p256Sig := append(r.FillBytes(make([]byte, 32)), s.FillBytes(make([]byte, 32))...)
	p256 := makeEntry("c-p256", "ecdsa-p256-sha256", p256Public, p256Sig, proofDID010+"#c-p256")
	secpPrivate, err := ethcrypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	secpPublic := ethcrypto.FromECDSAPub(&secpPrivate.PublicKey)
	secpSig, err := ethcrypto.Sign(ethcrypto.Keccak256(challenge("d-secp", "sage-secp256k1-keccak256", secpPublic)), secpPrivate)
	if err != nil {
		t.Fatal(err)
	}
	secp := makeEntry("d-secp", "sage-secp256k1-keccak256", secpPublic, secpSig, proofDID010+"#d-secp")
	return map[string]any{"id": proofDID010, "controller": "operator", "state": "active", "version": "1",
		"keys": []any{ed, kem, p256, secp}, "services": []any{}}
}

func proofBody010(t *testing.T, record map[string]any) []byte {
	t.Helper()
	encoded, err := json.Marshal(map[string]any{"record": record, "issued": 100, "expires": 105})
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestWebRegistryProofs010(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(map[string]any)
		want bool
	}{
		{"valid all suites", func(map[string]any) {}, true},
		{"changed proof", func(r map[string]any) {
			r["keys"].([]any)[0].(map[string]any)["proof"].(map[string]any)["value"] = base64.RawURLEncoding.EncodeToString(make([]byte, 64))
		}, false},
		{"wrong KEM signer", func(r map[string]any) {
			r["keys"].([]any)[1].(map[string]any)["proof"].(map[string]any)["signer"] = proofDID010 + "#missing"
		}, false},
		{"wrong KEM signature", func(r map[string]any) {
			r["keys"].([]any)[1].(map[string]any)["proof"].(map[string]any)["value"] = base64.RawURLEncoding.EncodeToString(make([]byte, 64))
		}, false},
		{"low order KEM", func(r map[string]any) {
			r["keys"].([]any)[1].(map[string]any)["key"] = base64.RawURLEncoding.EncodeToString(make([]byte, 32))
		}, false},
		{"historical KEM signer", func(r map[string]any) {
			r["keys"].([]any)[0].(map[string]any)["state"] = "revoked"
		}, true},
		{"high S P256", func(r map[string]any) {
			proof := r["keys"].([]any)[2].(map[string]any)["proof"].(map[string]any)
			sig, _ := base64.RawURLEncoding.DecodeString(proof["value"].(string))
			s := new(big.Int).Sub(elliptic.P256().Params().N, new(big.Int).SetBytes(sig[32:]))
			copy(sig[32:], s.FillBytes(make([]byte, 32)))
			proof["value"] = base64.RawURLEncoding.EncodeToString(sig)
		}, false},
		{"wrong secp recovery", func(r map[string]any) {
			proof := r["keys"].([]any)[3].(map[string]any)["proof"].(map[string]any)
			sig, _ := base64.RawURLEncoding.DecodeString(proof["value"].(string))
			sig[64] ^= 1
			proof["value"] = base64.RawURLEncoding.EncodeToString(sig)
		}, false},
		{"high S secp", func(r map[string]any) {
			proof := r["keys"].([]any)[3].(map[string]any)["proof"].(map[string]any)
			sig, _ := base64.RawURLEncoding.DecodeString(proof["value"].(string))
			s := new(big.Int).Sub(ethcrypto.S256().Params().N, new(big.Int).SetBytes(sig[32:64]))
			copy(sig[32:64], s.FillBytes(make([]byte, 32)))
			sig[64] ^= 1
			proof["value"] = base64.RawURLEncoding.EncodeToString(sig)
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := proofFixture010(t)
			tc.edit(record)
			err := CheckWebRegistryProofs010(proofBody010(t, record), proofDID010, 100)
			if (err == nil) != tc.want {
				t.Fatalf("got %v, want success %v", err, tc.want)
			}
		})
	}
}

func TestWebRegistryProofsRejectsMixedTorsion010(t *testing.T) {
	base := edwards25519.NewGeneratorPoint()
	orderTwo := make([]byte, 32)
	orderTwo[0] = 0xec
	for i := 1; i < 31; i++ {
		orderTwo[i] = 0xff
	}
	orderTwo[31] = 0x7f
	torsion, err := new(edwards25519.Point).SetBytes(orderTwo)
	if err != nil {
		t.Fatal(err)
	}
	mixed := new(edwards25519.Point).Add(base, torsion)
	if webPrimeEdPoint010(mixed.Bytes()) || !webPrimeEdPoint010(base.Bytes()) {
		t.Fatal("Ed25519 prime-order boundary")
	}
}
