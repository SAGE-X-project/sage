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
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"filippo.io/edwards25519"

	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/core/obs"
	"github.com/sage-x-project/sage/core/secret"
)

// Values of sage-spec v1/vectors/crypto.json.
const (
	vecMessage      = "SAGE test vector message: The quick brown fox jumps over the lazy dog."
	vecEd25519Label = "sage-spec/vectors/ed25519/agent-a"
	vecEd25519Pub   = "123e4e4034bebe7db6edcbb65d0f95bf10dab35729cd218da51a72abe7bc7610"
	vecEd25519Sig   = "0fe02454adb2f05b096a148b3b1972c658a6efc397b39f1eb542b0e6c42b69656406778ceee62ee607620698fa4215936cc7902702660b102398f8e07a50e00b"
	vecEd25519KeyID = "0ea249d59a0394ed"
	vecK1Label      = "sage-spec/vectors/secp256k1/agent-a"
	vecK1Pub        = "04e4ca58183538cf9b3ae81e0a2b9f77c1f32f2ad256025f856c2182bb548675e37390e2e0c047f94686a454de3937009dff7af4cb5a1e57409b31e0be35e73f78"
	vecK1PubC       = "02e4ca58183538cf9b3ae81e0a2b9f77c1f32f2ad256025f856c2182bb548675e3"
	vecK1Sig        = "ae2ab47818f0ca3c1367e629709133bc10e3e9fb32bcc5adc15c8fd22c2619be0084cd8f085896fc080f1ecb716c8a8c15ae77d1f98ead7b520053c5b58f533d01"
	vecK1KeyID      = "4809980dad831b92"
	vecK1Address    = "0x7016158acbaa8d06c5dc661498bb6ff76381c82f"
	vecP256Pub      = "04a8b98f6ec0083554828dffc9876d3b00e39f79c4fc3f559d0de2fde4718a774639f25cf19fb0935ee621da503a4f65128415d90e078ec01267726538b0e94d55"
	vecP256Sig      = "de826013c62f41c0b65a2a7313dfa95954ed9564e6ef8700fef5c56407d72394630efeef7fb6276651ace68880c0235d99ea200e6f1c6459efbde0a53d134902"
	vecKeyIDInput   = "37af5b79b1d08725d007175621b267e6892e1ce6cab3cb5a39cb75cee280db08"
	vecKeyIDOutput  = "31df14d7d9fcb731"
)

func unhex(t testing.TB, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func labelSecret(label string) secret.Bytes {
	sum := sha256.Sum256([]byte(label))
	return secret.From(sum[:])
}

func wantReason(t *testing.T, err error, want obs.Reason) {
	t.Helper()
	var r *core.Rejection
	if !errors.As(err, &r) {
		t.Fatalf("error %v is not a core.Rejection", err)
	}
	if r.Reason() != want {
		t.Fatalf("reason %s, want %s", r.Reason(), want)
	}
	if err.Error() != core.AuthenticationFailedText {
		t.Fatalf("error text %q", err)
	}
}

func mustKey(t *testing.T, alg Alg, h string) PublicKey {
	t.Helper()
	k, err := ParsePublicKey(alg, unhex(t, h))
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func mustSig(t *testing.T, alg Alg, b []byte) Signature {
	t.Helper()
	s, err := ParseSignature(alg, b)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestAlg(t *testing.T) {
	for s, want := range map[string]Alg{"ed25519": AlgEd25519, "es256k": AlgES256K, "ecdsa-p256-sha256": AlgP256} {
		got, err := ParseAlg(s)
		if err != nil || got != want || got.String() != s {
			t.Errorf("%s: %v %v", s, got, err)
		}
	}
	for _, s := range []string{"rsa-pss-sha256", "rsa-v1_5-sha256", "ES256K", "Ed25519", "ed25519 ", "", "hmac-sha256", "ecdsa-p384-sha384"} {
		_, err := ParseAlg(s)
		wantReason(t, err, obs.ReasonAlgorithm)
	}
	if AlgUnknown.String() != "unknown" {
		t.Error("unknown alg name")
	}
}

func TestPublishedEd25519(t *testing.T) {
	signer, err := NewEd25519Signer(labelSecret(vecEd25519Label))
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(signer.Public().Bytes()) != vecEd25519Pub {
		t.Fatal("public key differs from the vector")
	}
	sig, err := signer.Sign(context.Background(), PurposeHTTPRequest, []byte(vecMessage))
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(sig.Bytes()) != vecEd25519Sig || sig.Alg() != AlgEd25519 {
		t.Fatal("signature differs from the vector")
	}
	pub := mustKey(t, AlgEd25519, vecEd25519Pub)
	if KeyIDOf(pub).String() != vecEd25519KeyID {
		t.Fatal("key id")
	}
	if err := Verify(pub, sig, []byte(vecMessage)); err != nil {
		t.Fatal(err)
	}
	wantReason(t, Verify(pub, sig, []byte(vecMessage+".")), obs.ReasonSignature)
}

func TestEd25519Strict(t *testing.T) {
	sig := unhex(t, vecEd25519Sig)
	// S + L, still below 2^256.
	l, _ := new(big.Int).SetString("7237005577332262213973186563042994240857116359379907606001950938285454250989", 10)
	s := new(big.Int).SetBytes(reverse(sig[32:]))
	plusL := append(append([]byte{}, sig[:32]...), reverse(new(big.Int).Add(s, l).FillBytes(make([]byte, 32)))...)
	_, err := ParseSignature(AlgEd25519, plusL)
	wantReason(t, err, obs.ReasonNonCanonical)
	for _, n := range []int{0, 63, 65} {
		_, err := ParseSignature(AlgEd25519, make([]byte, n))
		wantReason(t, err, obs.ReasonMalformed)
	}
	// R with y >= p: y = p + 2 is not a canonical encoding.
	nonCanonR := append(nonCanonicalY(t), sig[32:]...)
	_, err = ParseSignature(AlgEd25519, nonCanonR)
	wantReason(t, err, obs.ReasonNonCanonical)

	_, err = ParsePublicKey(AlgEd25519, nonCanonicalY(t))
	wantReason(t, err, obs.ReasonNonCanonical)
	for name, h := range map[string]string{
		"identity":     "0100000000000000000000000000000000000000000000000000000000000000",
		"order 2":      "ecffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f",
		"order 4":      "0000000000000000000000000000000000000000000000000000000000000000",
		"order 8":      "c7176a703d4dd84fba3c0b760d10670f2a2053fa2c39ccc64ec7fd7792ac037a",
		"order 8 neg":  "26e8958fc2b227b045c3f489f2ef98f0d5dfac05d3c63339b13802886d53fc05",
		"order 8 sign": "c7176a703d4dd84fba3c0b760d10670f2a2053fa2c39ccc64ec7fd7792ac03fa",
	} {
		_, err := ParsePublicKey(AlgEd25519, unhex(t, h))
		if err == nil {
			t.Errorf("%s: small-order key accepted", name)
			continue
		}
		var r *core.Rejection
		if !errors.As(err, &r) || (r.Reason() != obs.ReasonMalformed && r.Reason() != obs.ReasonNonCanonical) {
			t.Errorf("%s: %v", name, err)
		}
	}
	for _, n := range []int{31, 33} {
		_, err := ParsePublicKey(AlgEd25519, make([]byte, n))
		wantReason(t, err, obs.ReasonMalformed)
	}
	// y = 2 has no x on the curve.
	notOnCurve := make([]byte, 32)
	notOnCurve[0] = 2
	_, err = ParsePublicKey(AlgEd25519, notOnCurve)
	wantReason(t, err, obs.ReasonMalformed)
}

// nonCanonicalY returns p + y' for the smallest y' whose point exists and is
// not of small order, the encoding E01-04 requires verifiers to reject.
func nonCanonicalY(t *testing.T) []byte {
	t.Helper()
	identity := edwards25519.NewIdentityPoint()
	for y := byte(2); y <= 18; y++ {
		enc := make([]byte, 32)
		for i := range enc {
			enc[i] = 0xff
		}
		enc[0], enc[31] = 0xed+y, 0x7f // little-endian p + y
		p, err := new(edwards25519.Point).SetBytes(enc)
		if err != nil {
			continue
		}
		if new(edwards25519.Point).MultByCofactor(p).Equal(identity) == 0 {
			return enc
		}
	}
	t.Fatal("no non-canonical encoding found")
	return nil
}

func reverse(b []byte) []byte {
	out := make([]byte, len(b))
	for i := range b {
		out[i] = b[len(b)-1-i]
	}
	return out
}

func TestPublishedSecp256k1(t *testing.T) {
	signer, err := NewSecp256k1Signer(labelSecret(vecK1Label))
	if err != nil {
		t.Fatal(err)
	}
	pub := signer.Public()
	if hex.EncodeToString(pub.Bytes()) != vecK1Pub {
		t.Fatal("public key")
	}
	sig, err := signer.Sign(context.Background(), PurposeHTTPRequest, []byte(vecMessage))
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(sig.Bytes()) != vecK1Sig {
		t.Fatalf("signature %x differs from the vector", sig.Bytes())
	}
	if err := Verify(pub, sig, []byte(vecMessage)); err != nil {
		t.Fatal(err)
	}
	if KeyIDOf(pub).String() != vecK1KeyID {
		t.Fatal("key id")
	}
	addr, err := EthereumAddress(pub)
	if err != nil || addr != vecK1Address {
		t.Fatalf("address %s %v", addr, err)
	}
	if _, err := EthereumAddress(mustKey(t, AlgEd25519, vecEd25519Pub)); err == nil {
		t.Fatal("address of an Ed25519 key")
	}
	// Compressed and X||Y forms decode to the same key and key id (E01-03, E01-12).
	full := unhex(t, vecK1Pub)
	for _, form := range [][]byte{unhex(t, vecK1PubC), full[1:], full} {
		k, err := ParsePublicKey(AlgES256K, form)
		if err != nil {
			t.Fatal(err)
		}
		if !k.Equal(pub) || KeyIDOf(k).String() != vecK1KeyID || len(k.Bytes()) != 65 {
			t.Fatalf("form %d bytes decodes differently", len(form))
		}
	}
}

func TestSecp256k1Strict(t *testing.T) {
	pub := mustKey(t, AlgES256K, vecK1Pub)
	msg := []byte(vecMessage)
	sig := unhex(t, vecK1Sig)
	n := secp256k1N()
	r, s := new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:64])
	mut := func(f func(b []byte)) []byte {
		b := append([]byte{}, sig...)
		f(b)
		return b
	}
	highS := mut(func(b []byte) { new(big.Int).Sub(n, s).FillBytes(b[32:64]); b[64] ^= 1 })
	cases := map[string]struct {
		sig    []byte
		reason obs.Reason
	}{
		"high s":      {highS, obs.ReasonNonCanonical},
		"64 bytes":    {sig[:64], obs.ReasonMalformed},
		"der":         {derEncode(r, s), obs.ReasonMalformed},
		"der 2^300":   {derEncode(new(big.Int).Lsh(big.NewInt(1), 300), s), obs.ReasonMalformed},
		"v 27":        {mut(func(b []byte) { b[64] += 27 }), obs.ReasonNonCanonical},
		"v 2":         {mut(func(b []byte) { b[64] = 2 }), obs.ReasonNonCanonical},
		"r zero":      {mut(func(b []byte) { clear(b[:32]) }), obs.ReasonMalformed},
		"s zero":      {mut(func(b []byte) { clear(b[32:64]) }), obs.ReasonMalformed},
		"r equals n":  {mut(func(b []byte) { n.FillBytes(b[:32]) }), obs.ReasonNonCanonical},
		"r all ones":  {mut(func(b []byte) { copy(b[:32], strings.Repeat("\xff", 32)) }), obs.ReasonNonCanonical},
		"s equals n":  {mut(func(b []byte) { n.FillBytes(b[32:64]) }), obs.ReasonNonCanonical},
		"66 bytes":    {append(append([]byte{}, sig...), 0), obs.ReasonMalformed},
		"half n + 1s": {mut(func(b []byte) { new(big.Int).Add(new(big.Int).Rsh(n, 1), big.NewInt(1)).FillBytes(b[32:64]) }), obs.ReasonNonCanonical},
	}
	for name, c := range cases {
		_, err := ParseSignature(AlgES256K, c.sig)
		if err == nil {
			t.Errorf("%s: accepted", name)
			continue
		}
		t.Run(name, func(t *testing.T) { wantReason(t, err, c.reason) })
	}
	// s = n/2 is low-S and parses.
	if _, err := ParseSignature(AlgES256K, mut(func(b []byte) { new(big.Int).Rsh(n, 1).FillBytes(b[32:64]) })); err != nil {
		t.Errorf("s = n/2 rejected: %v", err)
	}
	wrongV := mustSig(t, AlgES256K, mut(func(b []byte) { b[64] ^= 1 }))
	wantReason(t, Verify(pub, wrongV, msg), obs.ReasonSignature)
	wantReason(t, Verify(pub, mustSig(t, AlgES256K, sig), append([]byte{}, msg[1:]...)), obs.ReasonSignature)
}

func TestSecp256k1KeyEncodings(t *testing.T) {
	full := unhex(t, vecK1Pub)
	offCurve := append([]byte{}, full...)
	offCurve[64] ^= 1
	hybrid := append([]byte{}, full...)
	hybrid[0] = 0x06
	compressedBadPrefix := unhex(t, vecK1PubC)
	compressedBadPrefix[0] = 0x04
	xTooBig := append([]byte{0x02}, []byte(strings.Repeat("\xff", 32))...)
	for name, b := range map[string][]byte{
		"off curve": offCurve, "hybrid": hybrid, "compressed prefix 04": compressedBadPrefix,
		"x >= p": xTooBig, "xy off curve": offCurve[1:], "zero point": make([]byte, 65),
		"32 bytes": full[1:33], "empty": nil,
	} {
		_, err := ParsePublicKey(AlgES256K, b)
		t.Run(name, func(t *testing.T) { wantReason(t, err, obs.ReasonMalformed) })
	}
}

func TestP256(t *testing.T) {
	pub := mustKey(t, AlgP256, vecP256Pub)
	msg := []byte(vecMessage)
	sig := unhex(t, vecP256Sig)
	if err := Verify(pub, mustSig(t, AlgP256, sig), msg); err != nil {
		t.Fatalf("published P-256 signature: %v", err)
	}
	n := elliptic.P256().Params().N
	r, s := new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:])
	high := append(append([]byte{}, sig[:32]...), new(big.Int).Sub(n, s).FillBytes(make([]byte, 32))...)
	_, err := ParseSignature(AlgP256, high)
	wantReason(t, err, obs.ReasonNonCanonical)
	_, err = ParseSignature(AlgP256, derEncode(r, s))
	wantReason(t, err, obs.ReasonMalformed)
	_, err = ParseSignature(AlgP256, append(append([]byte{}, sig...), 0))
	wantReason(t, err, obs.ReasonMalformed)
	half := new(big.Int).Rsh(n, 1)
	withS := func(v *big.Int) []byte {
		return append(append([]byte{}, sig[:32]...), v.FillBytes(make([]byte, 32))...)
	}
	withR := func(v *big.Int) []byte { return append(v.FillBytes(make([]byte, 32)), sig[32:]...) }
	if _, err := ParseSignature(AlgP256, withS(half)); err != nil {
		t.Errorf("s = n/2 rejected: %v", err)
	}
	for name, c := range map[string]struct {
		b      []byte
		reason obs.Reason
	}{
		"s = n/2 + 1": {withS(new(big.Int).Add(half, big.NewInt(1))), obs.ReasonNonCanonical},
		"s = 0":       {withS(big.NewInt(0)), obs.ReasonMalformed},
		"r = 0":       {withR(big.NewInt(0)), obs.ReasonMalformed},
		"r = n":       {withR(n), obs.ReasonNonCanonical},
	} {
		_, err := ParseSignature(AlgP256, c.b)
		t.Run(name, func(t *testing.T) { wantReason(t, err, c.reason) })
	}
	wantReason(t, Verify(pub, mustSig(t, AlgP256, sig), msg[1:]), obs.ReasonSignature)
	full := unhex(t, vecP256Pub)
	off := append([]byte{}, full...)
	off[64] ^= 1
	for _, b := range [][]byte{off, full[1:], make([]byte, 65), append([]byte{0x02}, full[1:33]...)} {
		_, err := ParsePublicKey(AlgP256, b)
		wantReason(t, err, obs.ReasonMalformed)
	}

	signer, err := NewP256Signer(labelSecret("sage-spec/vectors/p256/agent-a"))
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(signer.Public().Bytes()) != vecP256Pub {
		t.Fatal("P-256 key from the label differs from the vector")
	}
	for i := 0; i < 32; i++ {
		sig, err := signer.Sign(context.Background(), PurposeHTTPRequest, msg)
		if err != nil {
			t.Fatal(err)
		}
		if err := Verify(pub, sig, msg); err != nil {
			t.Fatalf("own signature: %v", err)
		}
	}
}

func TestVerifyChecksAlgorithm(t *testing.T) {
	ed := mustKey(t, AlgEd25519, vecEd25519Pub)
	k1sig := mustSig(t, AlgES256K, unhex(t, vecK1Sig))
	wantReason(t, Verify(ed, k1sig, []byte(vecMessage)), obs.ReasonAlgorithm)
	wantReason(t, Verify(ed, Signature{}, []byte(vecMessage)), obs.ReasonAlgorithm)
	// A missing key is a rejection like any other (DP-5: no distinct error
	// for an unknown agent).
	wantReason(t, Verify(nil, k1sig, nil), obs.ReasonKeyNotEligible)
	wantReason(t, Verify(foreignKey{ed}, mustSig(t, AlgEd25519, unhex(t, vecEd25519Sig)), []byte(vecMessage)), obs.ReasonKeyNotEligible)
	for _, a := range []Alg{AlgUnknown, Alg(99)} {
		_, err := ParsePublicKey(a, unhex(t, vecEd25519Pub))
		wantReason(t, err, obs.ReasonAlgorithm)
		_, err = ParseSignature(a, unhex(t, vecEd25519Sig))
		wantReason(t, err, obs.ReasonAlgorithm)
	}
}

func TestKeyID(t *testing.T) {
	if got := KeyIDOf(mustKey(t, AlgEd25519, vecKeyIDInput)).String(); got != vecKeyIDOutput {
		t.Fatalf("key id %s", got)
	}
	if KeyIDOf(nil).String() != "" {
		t.Fatal("key id of nil")
	}
	// A type that embeds a real key to satisfy the sealed interface cannot
	// choose its key id.
	if KeyIDOf(foreignKey{mustKey(t, AlgEd25519, vecKeyIDInput)}).String() != "" {
		t.Fatal("key id of a foreign key type")
	}
	if _, err := EthereumAddress(foreignKey{mustKey(t, AlgES256K, vecK1Pub)}); err == nil {
		t.Fatal("address of a foreign key type")
	}
}

func TestValuesAreCopies(t *testing.T) {
	k := mustKey(t, AlgES256K, vecK1Pub)
	b := k.Bytes()
	b[1] ^= 0xff
	if hex.EncodeToString(k.Bytes()) != vecK1Pub {
		t.Fatal("Bytes exposes the key")
	}
	in := unhex(t, vecK1Sig)
	s := mustSig(t, AlgES256K, in)
	in[0] ^= 0xff
	out := s.Bytes()
	out[1] ^= 0xff
	if hex.EncodeToString(s.Bytes()) != vecK1Sig {
		t.Fatal("Signature shares memory with its input or output")
	}
	if k.Equal(nil) || k.Equal(mustKey(t, AlgEd25519, vecEd25519Pub)) {
		t.Fatal("Equal")
	}
}

func TestSigners(t *testing.T) {
	ctx := context.Background()
	makers := map[string]func(secret.Bytes) (Signer, error){
		"ed25519": NewEd25519Signer, "secp256k1": NewSecp256k1Signer, "p256": NewP256Signer,
	}
	for name, mk := range makers {
		t.Run(name, func(t *testing.T) {
			label := "sage-core/crypto-test/" + name
			s, err := mk(labelSecret(label))
			if err != nil {
				t.Fatal(err)
			}
			for _, p := range []Purpose{PurposeUnknown, PurposePoP, PurposePoP + 1} {
				if _, err := s.Sign(ctx, p, []byte("m")); !errors.Is(err, ErrPurpose) {
					t.Fatalf("purpose %d: %v", p, err)
				}
			}
			cancelled, cancel := context.WithCancel(ctx)
			cancel()
			if _, err := s.Sign(cancelled, PurposeCardProof, []byte("m")); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelled context: %v", err)
			}
			sum := sha256.Sum256([]byte(label))
			seedHex := hex.EncodeToString(sum[:])
			// A signer held in an unexported field is printed by fmt without
			// its methods; the fields must still hide the key.
			type holder struct {
				name string
				s    Signer
			}
			verbs := []string{"%v", "%+v", "%#v", "%s", "%x", "%X", "%d", "%q", "%t", "%c", "%e", "%U", "%b", "%o"}
			for _, verb := range verbs {
				out := fmt.Sprintf(verb, holder{"h", s}) + fmt.Sprintf(verb, &holder{"h", s})
				if leaksKey(out, sum[:]) {
					t.Fatalf("%s of a holder prints the key: %s", verb, out)
				}
			}
			for _, verb := range verbs[:8] {
				out := fmt.Sprintf(verb, s)
				if strings.Contains(strings.ToLower(out), seedHex[:16]) || leaksKey(out, sum[:]) {
					t.Fatalf("%s prints the key: %s", verb, out)
				}
				if !strings.Contains(out, secret.Redacted) {
					t.Fatalf("%s: %q lacks the redaction", verb, out)
				}
			}
			for _, bad := range [][]byte{nil, make([]byte, 31), make([]byte, 33)} {
				if _, err := mk(secret.From(bad)); err == nil {
					t.Fatalf("%d-byte key accepted", len(bad))
				}
			}
			destroyed := labelSecret(label)
			destroyed.Destroy()
			if _, err := mk(destroyed); err == nil {
				t.Fatal("destroyed secret accepted")
			}
		})
	}
	for name, c := range map[string]struct {
		mk func(secret.Bytes) (Signer, error)
		n  *big.Int
	}{"secp256k1": {NewSecp256k1Signer, secp256k1N()}, "p256": {NewP256Signer, elliptic.P256().Params().N}} {
		nMinus1 := new(big.Int).Sub(c.n, big.NewInt(1))
		for _, scalar := range [][]byte{make([]byte, 32), []byte(strings.Repeat("\xff", 32)), c.n.FillBytes(make([]byte, 32))} {
			if _, err := c.mk(secret.From(scalar)); err == nil {
				t.Errorf("%s: scalar %x... accepted", name, scalar[:2])
			}
		}
		if _, err := c.mk(secret.From(nMinus1.FillBytes(make([]byte, 32)))); err != nil {
			t.Errorf("%s: scalar n-1 rejected: %v", name, err)
		}
	}
}

// derEncode returns the ASN.1 DER SEQUENCE of two INTEGERs, for tests.
func derEncode(r, s *big.Int) []byte {
	integer := func(v *big.Int) []byte {
		b := v.Bytes()
		if len(b) == 0 || b[0]&0x80 != 0 {
			b = append([]byte{0}, b...)
		}
		return append([]byte{0x02, byte(len(b))}, b...)
	}
	body := append(integer(r), integer(s)...)
	return append([]byte{0x30, byte(len(body))}, body...)
}

func secp256k1N() *big.Int {
	n, _ := new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
	return n
}

func FuzzParseSignature(f *testing.F) {
	f.Add(uint8(AlgES256K), []byte{})
	f.Add(uint8(AlgEd25519), make([]byte, 64))
	f.Add(uint8(AlgP256), derEncode(new(big.Int).Lsh(big.NewInt(1), 300), big.NewInt(1)))
	f.Fuzz(func(t *testing.T, alg uint8, b []byte) {
		s, err := ParseSignature(Alg(alg), b)
		if err != nil {
			var r *core.Rejection
			if !errors.As(err, &r) {
				t.Fatalf("non-rejection error %v", err)
			}
			return
		}
		if string(s.Bytes()) != string(b) {
			t.Fatal("accepted signature changed its bytes")
		}
	})
}

func FuzzParsePublicKey(f *testing.F) {
	f.Add(uint8(AlgES256K), []byte{0x02})
	f.Add(uint8(AlgEd25519), make([]byte, 32))
	f.Fuzz(func(t *testing.T, alg uint8, b []byte) {
		k, err := ParsePublicKey(Alg(alg), b)
		if err != nil {
			var r *core.Rejection
			if !errors.As(err, &r) {
				t.Fatalf("non-rejection error %v", err)
			}
			return
		}
		again, err := ParsePublicKey(k.Alg(), k.Bytes())
		if err != nil || !again.Equal(k) {
			t.Fatalf("canonical bytes do not parse to the same key: %v", err)
		}
	})
}

// leaksKey reports whether out shows the first bytes of key in hex or as the
// decimal list fmt prints for a byte slice.
func leaksKey(out string, key []byte) bool {
	dec := fmt.Sprint(key[0], " ", key[1], " ", key[2], " ", key[3])
	return strings.Contains(strings.ToLower(out), hex.EncodeToString(key[:8])) || strings.Contains(out, dec)
}

// foreignKey embeds a real key to satisfy the sealed interface.
type foreignKey struct{ PublicKey }

func (foreignKey) Bytes() []byte { return []byte("chosen by the caller") }

func TestX25519KeyHolderRedacted(t *testing.T) {
	sum := sha256.Sum256([]byte("sage-core/crypto-test/x25519"))
	k, err := NewX25519KEMKey(secret.From(sum[:]))
	if err != nil {
		t.Fatal(err)
	}
	type holder struct{ k KEMKey }
	for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%x", "%d", "%q", "%t", "%c", "%e", "%U"} {
		if out := fmt.Sprintf(verb, holder{k}); leaksKey(out, sum[:]) {
			t.Fatalf("%s prints the KEM key: %s", verb, out)
		}
	}
}

// TestEd25519IsCofactorless signs with a mixed-order key A' = [a]B + T, T of
// order 8. A cofactored verifier accepts every such signature; the
// cofactorless equation of 01-crypto.md §2 accepts it only when k is a
// multiple of 8.
func TestEd25519IsCofactorless(t *testing.T) {
	h := sha512.Sum512(labelSeedBytes(vecEd25519Label))
	a, err := new(edwards25519.Scalar).SetBytesWithClamping(h[:32])
	if err != nil {
		t.Fatal(err)
	}
	torsion, err := new(edwards25519.Point).SetBytes(unhex(t, "c7176a703d4dd84fba3c0b760d10670f2a2053fa2c39ccc64ec7fd7792ac037a"))
	if err != nil {
		t.Fatal(err)
	}
	mixed := new(edwards25519.Point).Add(new(edwards25519.Point).ScalarBaseMult(a), torsion)
	pub, err := ParsePublicKey(AlgEd25519, mixed.Bytes())
	if err != nil {
		t.Fatalf("a mixed-order key is not of small order and must parse: %v", err)
	}
	rejected, accepted := 0, 0
	for i := 0; i < 32; i++ {
		msg := []byte(fmt.Sprintf("mixed-order message %d", i))
		rh := sha512.Sum512(append(append([]byte{}, h[32:]...), msg...))
		r, _ := new(edwards25519.Scalar).SetUniformBytes(rh[:])
		R := new(edwards25519.Point).ScalarBaseMult(r)
		kh := sha512.New()
		kh.Write(R.Bytes())
		kh.Write(mixed.Bytes())
		kh.Write(msg)
		k, _ := new(edwards25519.Scalar).SetUniformBytes(kh.Sum(nil))
		S := new(edwards25519.Scalar).MultiplyAdd(k, a, r)
		sig := mustSig(t, AlgEd25519, append(R.Bytes(), S.Bytes()...))
		err := Verify(pub, sig, msg)
		if k.Bytes()[0]%8 == 0 {
			if err != nil {
				t.Fatalf("k = 0 mod 8 must verify under both equations: %v", err)
			}
			accepted++
			continue
		}
		wantReason(t, err, obs.ReasonSignature)
		rejected++
	}
	if rejected == 0 {
		t.Fatal("no message exercised the cofactorless rejection")
	}
	t.Logf("rejected %d, accepted %d (k = 0 mod 8)", rejected, accepted)
}

func labelSeedBytes(label string) []byte {
	sum := sha256.Sum256([]byte(label))
	return sum[:]
}

// TestX25519PublicKey covers E01-14 (an X25519 key that is not 32 bytes is
// rejected) and E01-12 (its KeyID is over the 32 bytes). It is not a
// signing key: it does not satisfy PublicKey, so Verify cannot take it.
func TestX25519PublicKey(t *testing.T) {
	for _, n := range []int{0, 31, 33, 65} {
		_, err := ParseX25519PublicKey(make([]byte, n))
		wantReason(t, err, obs.ReasonMalformed)
	}
	in := unhex(t, vecKeyIDInput)
	k, err := ParseX25519PublicKey(in)
	if err != nil {
		t.Fatal(err)
	}
	if k.KeyID().String() != vecKeyIDOutput || hex.EncodeToString(k.Bytes()) != vecKeyIDInput {
		t.Fatalf("key id %s", k.KeyID())
	}
	in[0] ^= 0xff
	out := k.Bytes()
	out[1] ^= 0xff
	if hex.EncodeToString(k.Bytes()) != vecKeyIDInput {
		t.Fatal("X25519PublicKey shares memory with its input or output")
	}
	other, err := ParseX25519PublicKey(in)
	if err != nil {
		t.Fatal(err)
	}
	if !k.Equal(k) || k.Equal(other) || k.Equal(X25519PublicKey{}) {
		t.Fatal("Equal")
	}
	if (X25519PublicKey{}).KeyID().String() != "" || (X25519PublicKey{}).Bytes() != nil {
		t.Fatal("the zero X25519PublicKey must be empty")
	}
	if _, ok := any(k).(PublicKey); ok {
		t.Fatal("an X25519 key must not be a signing PublicKey (DP-3.2)")
	}
}
