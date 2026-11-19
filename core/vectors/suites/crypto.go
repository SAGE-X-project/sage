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

package suites

import (
	"context"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"filippo.io/edwards25519"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"

	"github.com/sage-x-project/sage/core/crypto"
	"github.com/sage-x-project/sage/core/obs"
	"github.com/sage-x-project/sage/core/secret"
	"github.com/sage-x-project/sage/core/vectors"
)

// CryptoDescription is the description of the crypto suite file.
const CryptoDescription = "Signature primitives as used on every SAGE wire path: Ed25519 (pure, no pre-hash), " +
	"secp256k1 over Keccak-256 with RFC 6979 and the Ethereum r||s||v encoding, " +
	"P-256 over SHA-256 with raw r||s low-S encoding, and the key identifier helper. " +
	"Reject vectors hold alg, public_key, message and signature; the input is rejected when alg, the key " +
	"or the signature is not accepted or the signature does not verify (01-crypto.md §1 to §3)."

// Labels and message of vectors/README.md: every private key is derived as
// SHA-256(label).
const (
	labelEd25519A   = "sage-spec/vectors/ed25519/agent-a"
	labelEd25519B   = "sage-spec/vectors/ed25519/agent-b"
	labelSecp256k1A = "sage-spec/vectors/secp256k1/agent-a"
	labelP256A      = "sage-spec/vectors/p256/agent-a"
	messageA        = "SAGE test vector message: The quick brown fox jumps over the lazy dog."
)

// p256SignatureA is the published p256-sha256-sign signature. P-256 signing
// is randomised, so the reject vectors mutate this fixed value.
const p256SignatureA = "de826013c62f41c0b65a2a7313dfa95954ed9564e6ef8700fef5c56407d72394630efeef7fb6276651ace68880c0235d99ea200e6f1c6459efbde0a53d134902"

// Crypto returns the crypto suite: the four published vectors and the
// vectors of errata §11 for 01 (E01-02 to E01-06, E01-12, E01-14).
func Crypto() (vectors.Suite, error) {
	m, err := newMaterial()
	if err != nil {
		return vectors.Suite{}, err
	}
	return vectors.Suite{
		Name:        "crypto",
		Description: CryptoDescription,
		Cases:       append(cryptoPublished(), cryptoNew(m)...),
	}, nil
}

func labelSeed(label string) []byte {
	sum := sha256.Sum256([]byte(label))
	return sum[:]
}

func cryptoPublished() []vectors.Case {
	msg := hex.EncodeToString([]byte(messageA))
	edB := ed25519.NewKeyFromSeed(labelSeed(labelEd25519B)).Public().(ed25519.PublicKey)
	return []vectors.Case{
		{
			Name: "ed25519-sign", Mode: vectors.ModeDeterministic,
			Description: "Ed25519 signature over the raw message (RFC 8032, no pre-hash).",
			Input:       map[string]any{"seed_label": labelEd25519A, "message": msg},
			Produce:     produceEd25519, Verify: verifyStored(crypto.AlgEd25519),
		},
		{
			Name: "secp256k1-keccak-sign", Mode: vectors.ModeDeterministic,
			Description: "secp256k1: Keccak-256(message), RFC 6979 deterministic ECDSA, 65-byte r||s||v, low-S. " +
				"Identical to an Ethereum signature without the EIP-191 prefix.",
			Input:   map[string]any{"scalar_label": labelSecp256k1A, "message": msg},
			Produce: produceSecp256k1, Verify: verifyStored(crypto.AlgES256K),
		},
		{
			Name: "p256-sha256-sign", Mode: vectors.ModeVerify,
			Description: "P-256: SHA-256(message), ECDSA, raw r||s (64 bytes) with low-S normalisation. " +
				"Go's ecdsa.Sign is randomised, so this vector is verify-only until deterministic signing lands (B-11b).",
			Input:   map[string]any{"scalar_label": labelP256A, "message": msg},
			Produce: produceP256, Verify: verifyStored(crypto.AlgP256),
		},
		{
			Name: "key-id", Mode: vectors.ModeDeterministic,
			Description: "keys.KeyID: the key identifier derived from raw public key bytes.",
			Input:       map[string]any{"public_key": hex.EncodeToString(edB)},
			Produce: func(in map[string]any) (map[string]any, error) {
				k, err := parseKey(crypto.AlgEd25519, in, "public_key")
				if err != nil {
					return nil, err
				}
				return map[string]any{"key_id": crypto.KeyIDOf(k).String()}, nil
			},
		},
	}
}

// material holds the valid signatures the reject vectors mutate.
type material struct {
	msg                    []byte
	edPub, edSig           []byte
	k1Pub, k1PubC, k1Sig   []byte
	p256Pub, p256Sig       []byte
	ed25519NonCanonicalKey []byte
	ed25519SmallOrderKey   []byte
	ed25519SmallOrderSig   []byte
	secp256k1N, p256N, edL *big.Int
}

func newMaterial() (*material, error) {
	m := &material{msg: []byte(messageA)}
	ctx := context.Background()
	ed, err := crypto.NewEd25519Signer(secret.From(labelSeed(labelEd25519A)))
	if err != nil {
		return nil, err
	}
	edSig, err := ed.Sign(ctx, crypto.PurposeHTTPRequest, m.msg)
	if err != nil {
		return nil, err
	}
	m.edPub, m.edSig = ed.Public().Bytes(), edSig.Bytes()
	k1, err := crypto.NewSecp256k1Signer(secret.From(labelSeed(labelSecp256k1A)))
	if err != nil {
		return nil, err
	}
	k1Sig, err := k1.Sign(ctx, crypto.PurposeHTTPRequest, m.msg)
	if err != nil {
		return nil, err
	}
	m.k1Pub, m.k1Sig = k1.Public().Bytes(), k1Sig.Bytes()
	pk, err := secp256k1.ParsePubKey(m.k1Pub)
	if err != nil {
		return nil, err
	}
	m.k1PubC = pk.SerializeCompressed()
	p, err := crypto.NewP256Signer(secret.From(p256Scalar(labelP256A)))
	if err != nil {
		return nil, err
	}
	m.p256Pub = p.Public().Bytes()
	if m.p256Sig, err = hex.DecodeString(p256SignatureA); err != nil {
		return nil, err
	}
	m.secp256k1N, _ = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
	m.p256N = elliptic.P256().Params().N
	m.edL, _ = new(big.Int).SetString("7237005577332262213973186563042994240857116359379907606001950938285454250989", 10)
	if m.ed25519NonCanonicalKey, err = nonCanonicalEd25519(); err != nil {
		return nil, err
	}
	// An order-8 key A, R = the order-4 point with y = 0 (32 zero bytes) and
	// S = 0. For messageA both [S]B = R + [k]A and its cofactored form
	// hold, so a verifier that skips the small-order key check accepts it.
	m.ed25519SmallOrderKey, _ = hex.DecodeString("c7176a703d4dd84fba3c0b760d10670f2a2053fa2c39ccc64ec7fd7792ac037a")
	m.ed25519SmallOrderSig = make([]byte, 64)
	return m, nil
}

// p256Scalar reduces SHA-256(label) modulo the group order (vectors/README.md).
func p256Scalar(label string) []byte {
	d := new(big.Int).SetBytes(labelSeed(label))
	d.Mod(d, elliptic.P256().Params().N)
	return d.FillBytes(make([]byte, 32))
}

// nonCanonicalEd25519 returns the encoding p + y for the smallest y whose
// point exists and is not of small order, so only the y >= p rule rejects it.
func nonCanonicalEd25519() ([]byte, error) {
	identity := edwards25519.NewIdentityPoint()
	for y := byte(2); y <= 18; y++ {
		enc := make([]byte, 32)
		for i := range enc {
			enc[i] = 0xff
		}
		enc[0], enc[31] = 0xed+y, 0x7f // little-endian p + y, p = 2^255 - 19
		p, err := new(edwards25519.Point).SetBytes(enc)
		if err != nil {
			continue
		}
		if new(edwards25519.Point).MultByCofactor(p).Equal(identity) == 0 {
			return enc, nil
		}
	}
	return nil, errors.New("no non-canonical Ed25519 encoding found")
}

func cryptoNew(m *material) []vectors.Case {
	in := func(alg crypto.Alg, pub, sig []byte) map[string]any {
		return map[string]any{
			"alg": alg.String(), "public_key": hex.EncodeToString(pub),
			"message": hex.EncodeToString(m.msg), "signature": hex.EncodeToString(sig),
		}
	}
	r := func(name, desc string, input map[string]any, reason obs.Reason) vectors.Case {
		return vectors.Case{Name: name, Mode: vectors.ModeReject, Description: desc, Input: input, Reject: rejectSignature, Reason: reason}
	}
	edS := new(big.Int).SetBytes(reversed(m.edSig[32:]))
	edPlusL := append(append([]byte{}, m.edSig[:32]...), reversed(new(big.Int).Add(edS, m.edL).FillBytes(make([]byte, 32)))...)
	k1R, k1S := new(big.Int).SetBytes(m.k1Sig[:32]), new(big.Int).SetBytes(m.k1Sig[32:64])
	k1High := append(append(append([]byte{}, m.k1Sig[:32]...), new(big.Int).Sub(m.secp256k1N, k1S).FillBytes(make([]byte, 32))...), m.k1Sig[64]^1)
	k1WrongV := append(append([]byte{}, m.k1Sig[:64]...), m.k1Sig[64]^1)
	k1V27 := append(append([]byte{}, m.k1Sig[:64]...), m.k1Sig[64]+27)
	pR, pS := new(big.Int).SetBytes(m.p256Sig[:32]), new(big.Int).SetBytes(m.p256Sig[32:])
	pHigh := append(append([]byte{}, m.p256Sig[:32]...), new(big.Int).Sub(m.p256N, pS).FillBytes(make([]byte, 32))...)
	offCurve := func(pub []byte) []byte {
		b := append([]byte{}, pub...)
		b[64] ^= 1
		return b
	}
	return []vectors.Case{
		r("ed25519-s-not-reduced", "E01-04: ed25519-sign with S replaced by S + L.",
			in(crypto.AlgEd25519, m.edPub, edPlusL), obs.ReasonNonCanonical),
		r("ed25519-small-order-key", "E01-04: an order-8 public key with R = 32 zero bytes and S = 0; both the cofactorless and the cofactored equation accept it, so only the small-order key check rejects it.",
			in(crypto.AlgEd25519, m.ed25519SmallOrderKey, m.ed25519SmallOrderSig), obs.ReasonMalformed),
		r("ed25519-noncanonical-key", "E01-04: ed25519-sign with a public key encoded with y >= p (a point that is not of small order). Nobody knows a key for that point, so a lax verifier fails it as signature; the code tells the two apart.",
			in(crypto.AlgEd25519, m.ed25519NonCanonicalKey, m.edSig), obs.ReasonNonCanonical),
		r("secp256k1-high-s", "E01-05: secp256k1-keccak-sign with s replaced by n - s and v flipped.",
			in(crypto.AlgES256K, m.k1Pub, k1High), obs.ReasonNonCanonical),
		r("secp256k1-64-byte", "E01-05: the published signature_rs (64 bytes) presented alone.",
			in(crypto.AlgES256K, m.k1Pub, m.k1Sig[:64]), obs.ReasonMalformed),
		r("secp256k1-der", "E01-05: DER of the (r, s) of secp256k1-keccak-sign.",
			in(crypto.AlgES256K, m.k1Pub, derEncode(k1R, k1S)), obs.ReasonMalformed),
		r("secp256k1-wrong-v", "E01-05: secp256k1-keccak-sign with v flipped, which recovers a different key.",
			in(crypto.AlgES256K, m.k1Pub, k1WrongV), obs.ReasonSignature),
		r("secp256k1-v-27", "E01-05: secp256k1-keccak-sign with v + 27.",
			in(crypto.AlgES256K, m.k1Pub, k1V27), obs.ReasonNonCanonical),
		r("p256-high-s", "E01-06: p256-sha256-sign with s replaced by n - s.",
			in(crypto.AlgP256, m.p256Pub, pHigh), obs.ReasonNonCanonical),
		r("p256-der", "E01-06: DER of the (r, s) of p256-sha256-sign.",
			in(crypto.AlgP256, m.p256Pub, derEncode(pR, pS)), obs.ReasonMalformed),
		r("pubkey-off-curve-secp256k1", "E01-14: secp256k1-keccak-sign with the last byte of Y flipped (not on the curve).",
			in(crypto.AlgES256K, offCurve(m.k1Pub), m.k1Sig), obs.ReasonMalformed),
		r("pubkey-off-curve-p256", "E01-14: p256-sha256-sign with the last byte of Y flipped (not on the curve).",
			in(crypto.AlgP256, offCurve(m.p256Pub), m.p256Sig), obs.ReasonMalformed),
		r("rsa-rejected", "E01-02: ed25519-sign presented with alg = rsa-pss-sha256.",
			map[string]any{"alg": "rsa-pss-sha256", "public_key": hex.EncodeToString(m.edPub),
				"message": hex.EncodeToString(m.msg), "signature": hex.EncodeToString(m.edSig)}, obs.ReasonAlgorithm),
		{
			Name: "key-id-secp256k1-from-compressed", Mode: vectors.ModeDeterministic,
			Description: "E01-03, E01-12: the 33-byte compressed key of secp256k1-keccak-sign decodes to the 65-byte key, and its KeyID is computed over the 65 bytes.",
			Input:       map[string]any{"public_key_compressed": hex.EncodeToString(m.k1PubC)},
			Produce: func(in map[string]any) (map[string]any, error) {
				k, err := parseKey(crypto.AlgES256K, in, "public_key_compressed")
				if err != nil {
					return nil, err
				}
				return map[string]any{"public_key": hex.EncodeToString(k.Bytes()), "key_id": crypto.KeyIDOf(k).String()}, nil
			},
		},
	}
}

// rejectSignature runs the verifier path of a reject vector: alg, key,
// signature, then verification.
func rejectSignature(in map[string]any) error {
	if len(in) != 4 {
		return errors.New("input must hold alg, public_key, message and signature")
	}
	name, ok := in["alg"].(string)
	if !ok {
		return errors.New("alg must be a string")
	}
	pubHex, err := hexField(in, "public_key")
	if err != nil {
		return err
	}
	msg, err := hexField(in, "message")
	if err != nil {
		return err
	}
	sigHex, err := hexField(in, "signature")
	if err != nil {
		return err
	}
	alg, err := crypto.ParseAlg(name)
	if err != nil {
		return err
	}
	pub, err := crypto.ParsePublicKey(alg, pubHex)
	if err != nil {
		return err
	}
	sig, err := crypto.ParseSignature(alg, sigHex)
	if err != nil {
		return err
	}
	return crypto.Verify(pub, sig, msg)
}

func produceEd25519(in map[string]any) (map[string]any, error) {
	label, msg, err := labelAndMessage(in, "seed_label")
	if err != nil {
		return nil, err
	}
	seed := labelSeed(label)
	s, err := crypto.NewEd25519Signer(secret.From(seed))
	if err != nil {
		return nil, err
	}
	sig, err := s.Sign(context.Background(), crypto.PurposeHTTPRequest, msg)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"seed":       hex.EncodeToString(seed),
		"public_key": hex.EncodeToString(s.Public().Bytes()),
		"key_id":     crypto.KeyIDOf(s.Public()).String(),
		"signature":  hex.EncodeToString(sig.Bytes()),
	}, nil
}

func produceSecp256k1(in map[string]any) (map[string]any, error) {
	label, msg, err := labelAndMessage(in, "scalar_label")
	if err != nil {
		return nil, err
	}
	scalar := labelSeed(label)
	s, err := crypto.NewSecp256k1Signer(secret.From(scalar))
	if err != nil {
		return nil, err
	}
	sig, err := s.Sign(context.Background(), crypto.PurposeHTTPRequest, msg)
	if err != nil {
		return nil, err
	}
	pub := s.Public().Bytes()
	pk, err := secp256k1.ParsePubKey(pub)
	if err != nil {
		return nil, err
	}
	addr, err := crypto.EthereumAddress(s.Public())
	if err != nil {
		return nil, err
	}
	b := sig.Bytes()
	return map[string]any{
		"private_scalar":        hex.EncodeToString(scalar),
		"public_key":            hex.EncodeToString(pub),
		"public_key_compressed": hex.EncodeToString(pk.SerializeCompressed()),
		"ethereum_address":      addr,
		"key_id":                crypto.KeyIDOf(s.Public()).String(),
		"signature":             hex.EncodeToString(b),
		"signature_rs":          hex.EncodeToString(b[:64]), // informative; not a valid encoding (E01-05)
		"recovery_id":           int(b[64]),
	}, nil
}

func produceP256(in map[string]any) (map[string]any, error) {
	label, msg, err := labelAndMessage(in, "scalar_label")
	if err != nil {
		return nil, err
	}
	scalar := p256Scalar(label)
	s, err := crypto.NewP256Signer(secret.From(scalar))
	if err != nil {
		return nil, err
	}
	sig, err := s.Sign(context.Background(), crypto.PurposeHTTPRequest, msg)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"private_scalar": hex.EncodeToString(scalar),
		"public_key":     hex.EncodeToString(s.Public().Bytes()),
		"signature":      hex.EncodeToString(sig.Bytes()),
	}, nil
}

// verifyStored verifies output.signature over input.message with
// output.public_key. Only the 65-byte signature counts for es256k;
// signature_rs is informative.
func verifyStored(alg crypto.Alg) func(in, out map[string]any) error {
	return func(in, out map[string]any) error {
		msg, err := hexField(in, "message")
		if err != nil {
			return err
		}
		pub, err := parseKey(alg, out, "public_key")
		if err != nil {
			return err
		}
		b, err := hexField(out, "signature")
		if err != nil {
			return err
		}
		sig, err := crypto.ParseSignature(alg, b)
		if err != nil {
			return err
		}
		return crypto.Verify(pub, sig, msg)
	}
}

func parseKey(alg crypto.Alg, m map[string]any, name string) (crypto.PublicKey, error) {
	b, err := hexField(m, name)
	if err != nil {
		return nil, err
	}
	return crypto.ParsePublicKey(alg, b)
}

func labelAndMessage(in map[string]any, labelName string) (string, []byte, error) {
	label, ok := in[labelName].(string)
	if !ok || label == "" {
		return "", nil, fmt.Errorf("%s must be a non-empty string", labelName)
	}
	msg, err := hexField(in, "message")
	return label, msg, err
}

// hexField returns the lower-case hex string member name of m, decoded.
func hexField(m map[string]any, name string) ([]byte, error) {
	s, ok := m[name].(string)
	if !ok {
		return nil, fmt.Errorf("%s must be a hex string", name)
	}
	b, err := hex.DecodeString(s)
	if err != nil || hex.EncodeToString(b) != s {
		return nil, fmt.Errorf("%s must be lower-case hex", name)
	}
	return b, nil
}

func reversed(b []byte) []byte {
	out := make([]byte, len(b))
	for i := range b {
		out[i] = b[len(b)-1-i]
	}
	return out
}

// derEncode returns the ASN.1 DER SEQUENCE of two INTEGERs.
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
