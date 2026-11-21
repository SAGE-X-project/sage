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

package crypto

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/big"
	"testing"

	"github.com/sage-x-project/sage/core/obs"
)

const (
	popDID = "did:sage:ethereum:0x1111111111111111111111111111111111111111"
	// The published proofs of vec:did/pop-ed25519 and vec:did/pop-secp256k1.
	vecEd25519PoP = "04dbd9601516dfa4d930e14e286170dc13f580e72ce3f1c8d6d2dd47e15d54eb97cfb7a03ad0d05c7359050d598bbebe7194275e276c92e5d3930d2b136aa800"
	vecK1PoP      = "2ef383803e18e1fe74d390cc650d82e87ad2fd0b4815f75ec3f6cba76603e51e6ea3715557e7fe7a772ce92cb13d41a79be5104025c9f338f864c86a5a9bcf4200"
)

func popSigners(t *testing.T) (ed, k1 Signer) {
	t.Helper()
	ed, err := NewEd25519Signer(labelSecret(vecEd25519Label))
	if err != nil {
		t.Fatal(err)
	}
	k1, err = NewSecp256k1Signer(labelSecret(vecK1Label))
	if err != nil {
		t.Fatal(err)
	}
	return ed, k1
}

func TestPoPChallenge(t *testing.T) {
	got := string(PoPChallenge(popDID, []byte{0xAB, 0x01}))
	if got != "SAGE-PoP:"+popDID+":ab01" {
		t.Fatalf("challenge %q", got)
	}
}

// TestPublishedPoP verifies and reproduces the published proofs.
func TestPublishedPoP(t *testing.T) {
	ed, k1 := popSigners(t)
	for _, c := range []struct {
		s     Signer
		proof string
	}{{ed, vecEd25519PoP}, {k1, vecK1PoP}} {
		data := c.s.Public().Bytes()
		if err := VerifyPoP(c.s.Public(), data, popDID, unhex(t, c.proof)); err != nil {
			t.Fatalf("%s: published proof: %v", c.s.Public().Alg(), err)
		}
		sig, err := SignPoP(context.Background(), c.s, popDID, data)
		if err != nil {
			t.Fatal(err)
		}
		if hex.EncodeToString(sig.Bytes()) != c.proof {
			t.Fatalf("%s: SignPoP does not reproduce the published proof", c.s.Public().Alg())
		}
	}
}

// TestPoPKeyDataAsStored: the challenge uses key_data exactly as stored,
// so a proof over the 33-byte form is valid for that form only (E01-03).
func TestPoPKeyDataAsStored(t *testing.T) {
	_, k1 := popSigners(t)
	pub := k1.Public()
	compressed, xy := unhex(t, vecK1PubC), pub.Bytes()[1:]
	for _, data := range [][]byte{compressed, xy} {
		sig, err := SignPoP(context.Background(), k1, popDID, data)
		if err != nil {
			t.Fatalf("%d-byte key_data: %v", len(data), err)
		}
		if err := VerifyPoP(pub, data, popDID, sig.Bytes()); err != nil {
			t.Fatalf("%d-byte key_data: %v", len(data), err)
		}
		wantReason(t, VerifyPoP(pub, pub.Bytes(), popDID, sig.Bytes()), obs.ReasonSignature)
	}
}

func TestVerifyPoPRejects(t *testing.T) {
	ctx := context.Background()
	ed, k1 := popSigners(t)
	edPub, k1Pub := ed.Public(), k1.Public()
	edData, k1Data := edPub.Bytes(), k1Pub.Bytes()
	edProof, k1Proof := unhex(t, vecEd25519PoP), unhex(t, vecK1PoP)
	other, err := NewSecp256k1Signer(labelSecret("sage-core/crypto-test/other"))
	if err != nil {
		t.Fatal(err)
	}
	p256, err := NewP256Signer(labelSecret("sage-core/crypto-test/p256"))
	if err != nil {
		t.Fatal(err)
	}
	byOther, err := other.Sign(ctx, PurposePoP, PoPChallenge(popDID, k1Data))
	if err != nil {
		t.Fatal(err)
	}
	// An es256k signature over the challenge as a message (Keccak-256, the
	// path of every other signature) is not a proof (E01-07).
	keccakSigned, err := k1.Sign(ctx, PurposeHTTPRequest, PoPChallenge(popDID, k1Data))
	if err != nil {
		t.Fatal(err)
	}
	edRaw, err := ed.Sign(ctx, PurposeHTTPRequest, PoPChallenge(popDID, edData))
	if err != nil {
		t.Fatal(err)
	}
	n := secp256k1N()
	high := append([]byte(nil), k1Proof...)
	new(big.Int).Sub(n, new(big.Int).SetBytes(k1Proof[32:64])).FillBytes(high[32:64])
	high[64] ^= 1
	v27 := append([]byte(nil), k1Proof...)
	v27[64] += 27
	r, s := new(big.Int).SetBytes(k1Proof[:32]), new(big.Int).SetBytes(k1Proof[32:64])
	for _, c := range []struct {
		name  string
		pub   PublicKey
		data  []byte
		did   string
		proof []byte
		want  obs.Reason
	}{
		{"wrong did", k1Pub, k1Data, "did:sage:ethereum:0x2222222222222222222222222222222222222222", k1Proof, obs.ReasonSignature},
		{"did case changed", edPub, edData, "did:sage:Ethereum:0x1111111111111111111111111111111111111111", edProof, obs.ReasonSignature},
		{"another key's signature", k1Pub, k1Data, popDID, byOther.Bytes(), obs.ReasonSignature},
		{"keccak signed", k1Pub, k1Data, popDID, keccakSigned.Bytes(), obs.ReasonSignature},
		{"raw challenge", edPub, edData, popDID, edRaw.Bytes(), obs.ReasonSignature},
		{"high s", k1Pub, k1Data, popDID, high, obs.ReasonNonCanonical},
		{"v 27", k1Pub, k1Data, popDID, v27, obs.ReasonNonCanonical},
		{"64 bytes", k1Pub, k1Data, popDID, k1Proof[:64], obs.ReasonMalformed},
		{"der", k1Pub, k1Data, popDID, derEncode(r, s), obs.ReasonMalformed},
		{"ed25519 63 bytes", edPub, edData, popDID, edProof[:63], obs.ReasonMalformed},
		{"key data of another key", k1Pub, other.Public().Bytes(), popDID, k1Proof, obs.ReasonKeyNotEligible},
		{"key data malformed", k1Pub, k1Data[:40], popDID, k1Proof, obs.ReasonKeyNotEligible},
		{"p-256 key", p256.Public(), p256.Public().Bytes(), popDID, k1Proof[:64], obs.ReasonAlgorithm},
		{"nil key", nil, k1Data, popDID, k1Proof, obs.ReasonKeyNotEligible},
		{"foreign key", foreignKey{k1Pub}, k1Data, popDID, k1Proof, obs.ReasonKeyNotEligible},
	} {
		t.Run(c.name, func(t *testing.T) {
			wantReason(t, VerifyPoP(c.pub, c.data, c.did, c.proof), c.want)
		})
	}
}

// TestPoPIsNotAMessageSignature: Verify, the path of every other signature,
// does not accept a proof over the challenge (E01-07).
func TestPoPIsNotAMessageSignature(t *testing.T) {
	ed, k1 := popSigners(t)
	for _, c := range []struct {
		s     Signer
		proof string
	}{{ed, vecEd25519PoP}, {k1, vecK1PoP}} {
		sig := mustSig(t, c.s.Public().Alg(), unhex(t, c.proof))
		challenge := PoPChallenge(popDID, c.s.Public().Bytes())
		wantReason(t, Verify(c.s.Public(), sig, challenge), obs.ReasonSignature)
	}
}

// keccakSigner signs a PoP digest like a message, as a host signer that
// ignores the purpose would.
type keccakSigner struct{ Signer }

func (s keccakSigner) Sign(ctx context.Context, _ Purpose, msg []byte) (Signature, error) {
	return s.Signer.Sign(ctx, PurposeHTTPRequest, msg)
}

func TestSignPoPRefuses(t *testing.T) {
	ctx := context.Background()
	ed, k1 := popSigners(t)
	p256, err := NewP256Signer(labelSecret("sage-core/crypto-test/p256"))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name string
		s    Signer
		data []byte
	}{
		{"nil signer", nil, ed.Public().Bytes()},
		{"p-256 signer", p256, p256.Public().Bytes()},
		{"key data of another key", k1, ed.Public().Bytes()},
		{"key data malformed", k1, k1.Public().Bytes()[:40]},
		{"signer hashes the digest again", keccakSigner{k1}, k1.Public().Bytes()},
	} {
		t.Run(c.name, func(t *testing.T) {
			if _, err := SignPoP(ctx, c.s, popDID, c.data); !errors.Is(err, ErrPoP) {
				t.Fatalf("err = %v, want ErrPoP", err)
			}
		})
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := SignPoP(cancelled, ed, popDID, ed.Public().Bytes()); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled context: %v", err)
	}
}

// TestPurposePoPNeedsAChallenge: the signers take PurposePoP only for a
// challenge and hash it themselves, so a caller cannot get a signature over
// a digest it chose (for example an es256k message digest); the P-256
// signer refuses the purpose (06 §4).
func TestPurposePoPNeedsAChallenge(t *testing.T) {
	ctx := context.Background()
	ed, k1 := popSigners(t)
	p256, err := NewP256Signer(labelSecret("sage-core/crypto-test/p256"))
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte("transfer everything")
	digest := keccak256(msg)
	for _, s := range []Signer{ed, k1} {
		for _, m := range [][]byte{nil, digest, []byte("SAGE-PoP"), []byte("sage-pop:x")} {
			if _, err := s.Sign(ctx, PurposePoP, m); !errors.Is(err, ErrPurpose) {
				t.Fatalf("%s, %q: %v", s.Public().Alg(), m, err)
			}
		}
		challenge := PoPChallenge(popDID, s.Public().Bytes())
		sig, err := s.Sign(ctx, PurposePoP, challenge)
		if err != nil {
			t.Fatalf("%s: %v", s.Public().Alg(), err)
		}
		if err := VerifyPoP(s.Public(), s.Public().Bytes(), popDID, sig.Bytes()); err != nil {
			t.Fatalf("%s: %v", s.Public().Alg(), err)
		}
	}
	if _, err := p256.Sign(ctx, PurposePoP, PoPChallenge(popDID, p256.Public().Bytes())); !errors.Is(err, ErrPurpose) {
		t.Fatalf("p-256: %v", err)
	}
}

// TestEd25519PoPOverTheDigest records the limit stated at VerifyPoP: an
// Ed25519 proof verifies with Verify over the 32-byte digest, never over
// the challenge.
func TestEd25519PoPOverTheDigest(t *testing.T) {
	ed, _ := popSigners(t)
	sig := mustSig(t, AlgEd25519, unhex(t, vecEd25519PoP))
	digest := sha256.Sum256(PoPChallenge(popDID, ed.Public().Bytes()))
	if err := Verify(ed.Public(), sig, digest[:]); err != nil {
		t.Fatalf("the construction changed: %v", err)
	}
}

func FuzzVerifyPoP(f *testing.F) {
	f.Add(unhex(f, vecK1Pub), "did:sage:ethereum:0x11", unhex(f, vecK1PoP))
	f.Add(unhex(f, vecEd25519Pub), popDID, unhex(f, vecEd25519PoP))
	ed, err := NewEd25519Signer(labelSecret(vecEd25519Label))
	if err != nil {
		f.Fatal(err)
	}
	k1, err := NewSecp256k1Signer(labelSecret(vecK1Label))
	if err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, data []byte, did string, proof []byte) {
		for _, pub := range []PublicKey{ed.Public(), k1.Public()} {
			_ = VerifyPoP(pub, data, did, proof)
		}
	})
}
