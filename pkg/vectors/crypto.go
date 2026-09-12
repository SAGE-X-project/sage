package vectors

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/sage-x-project/sage/pkg/agent/crypto/jcs"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
)

const messageA = "SAGE test vector message: The quick brown fox jumps over the lazy dog."

func cryptoSuite() Suite {
	return Suite{
		Name: "crypto",
		Description: "Signature primitives as used on every SAGE wire path: Ed25519 (pure, no pre-hash), " +
			"secp256k1 over Keccak-256 with RFC 6979 and the Ethereum r||s||v encoding, " +
			"P-256 over SHA-256 with raw r||s low-S encoding, and the key identifier helper.",
		Cases: []Case{
			{
				Name: "ed25519-sign", Mode: ModeDeterministic,
				Description: "Ed25519 signature over the raw message (RFC 8032, no pre-hash).",
				Input:       map[string]any{"seed_label": labelEd25519A, "message": hx([]byte(messageA))},
				Produce: func(in map[string]any) (map[string]any, error) {
					label, _ := str(in, "seed_label")
					msg, err := unhex(in, "message")
					if err != nil {
						return nil, err
					}
					priv := ed25519FromLabel(label)
					pub := priv.Public().(ed25519.PublicKey)
					return map[string]any{
						"seed":       hx(priv.Seed()),
						"public_key": hx(pub),
						"key_id":     keys.KeyID(pub),
						"signature":  hx(ed25519.Sign(priv, msg)),
					}, nil
				},
				Verify: func(in, out map[string]any) error {
					msg, err := unhex(in, "message")
					if err != nil {
						return err
					}
					pub, err := unhex(out, "public_key")
					if err != nil {
						return err
					}
					sig, err := unhex(out, "signature")
					if err != nil {
						return err
					}
					return keys.VerifySignature(ed25519.PublicKey(pub), msg, sig)
				},
			},
			{
				Name: "secp256k1-keccak-sign", Mode: ModeDeterministic,
				Description: "secp256k1: Keccak-256(message), RFC 6979 deterministic ECDSA, 65-byte r||s||v, low-S. " +
					"Identical to an Ethereum signature without the EIP-191 prefix.",
				Input: map[string]any{"scalar_label": labelSecp256k1A, "message": hx([]byte(messageA))},
				Produce: func(in map[string]any) (map[string]any, error) {
					label, _ := str(in, "scalar_label")
					msg, err := unhex(in, "message")
					if err != nil {
						return nil, err
					}
					priv, err := secp256k1FromLabel(label)
					if err != nil {
						return nil, err
					}
					sig, err := keys.SignSecp256k1Keccak(priv, msg)
					if err != nil {
						return nil, err
					}
					addr, err := ethAddress(&priv.PublicKey)
					if err != nil {
						return nil, err
					}
					pubBytes := secp256k1PublicBytes(&priv.PublicKey)
					scalar, err := scalarBytes(priv)
					if err != nil {
						return nil, err
					}
					return map[string]any{
						"private_scalar":        scalar,
						"public_key":            hx(pubBytes),
						"public_key_compressed": hx(compressedSecp256k1(&priv.PublicKey)),
						"ethereum_address":      addr,
						"key_id":                keys.KeyID(pubBytes),
						"signature":             hx(sig),
						"signature_rs":          hx(sig[:64]),
						"recovery_id":           int(sig[64]),
					}, nil
				},
				Verify: func(in, out map[string]any) error {
					msg, err := unhex(in, "message")
					if err != nil {
						return err
					}
					pubBytes, err := unhex(out, "public_key")
					if err != nil {
						return err
					}
					sig, err := unhex(out, "signature")
					if err != nil {
						return err
					}
					pub, err := secp256k1PublicFromBytes(pubBytes)
					if err != nil {
						return err
					}
					if err := keys.VerifySecp256k1Keccak(pub, msg, sig); err != nil {
						return fmt.Errorf("65-byte: %w", err)
					}
					return keys.VerifySecp256k1Keccak(pub, msg, sig[:64])
				},
			},
			{
				Name: "p256-sha256-sign", Mode: ModeVerify,
				Description: "P-256: SHA-256(message), ECDSA, raw r||s (64 bytes) with low-S normalisation. " +
					"Go's ecdsa.Sign is randomised, so this vector is verify-only until deterministic signing lands (B-11b).",
				Input: map[string]any{"scalar_label": labelP256A, "message": hx([]byte(messageA))},
				Produce: func(in map[string]any) (map[string]any, error) {
					label, _ := str(in, "scalar_label")
					msg, err := unhex(in, "message")
					if err != nil {
						return nil, err
					}
					priv, err := p256FromLabel(label)
					if err != nil {
						return nil, err
					}
					digest := sha256.Sum256(msg)
					r, s, err := ecdsa.Sign(rand.Reader, priv, digest[:])
					if err != nil {
						return nil, err
					}
					scalar, err := scalarBytes(priv)
					if err != nil {
						return nil, err
					}
					return map[string]any{
						"private_scalar": scalar,
						"public_key":     hx(p256PublicBytes(&priv.PublicKey)),
						"signature":      hx(keys.EncodeRawECDSASignature(priv.Curve, r, s)),
					}, nil
				},
				Verify: func(in, out map[string]any) error {
					msg, err := unhex(in, "message")
					if err != nil {
						return err
					}
					pubBytes, err := unhex(out, "public_key")
					if err != nil {
						return err
					}
					sig, err := unhex(out, "signature")
					if err != nil {
						return err
					}
					pub, err := p256PublicFromBytes(pubBytes)
					if err != nil {
						return err
					}
					if len(sig) != 64 {
						return errors.New("signature must be 64 bytes")
					}
					if !isLowS(pub.Curve, sig[32:]) {
						return errors.New("signature s is not low-S")
					}
					return keys.VerifySignature(pub, msg, sig)
				},
			},
			{
				Name: "key-id", Mode: ModeDeterministic,
				Description: "keys.KeyID: the key identifier derived from raw public key bytes.",
				Input:       map[string]any{"public_key": hx(ed25519FromLabel(labelEd25519B).Public().(ed25519.PublicKey))},
				Produce: func(in map[string]any) (map[string]any, error) {
					pub, err := unhex(in, "public_key")
					if err != nil {
						return nil, err
					}
					return map[string]any{"key_id": keys.KeyID(pub)}, nil
				},
			},
		},
	}
}

// JCS inputs are written with JSON escape sequences (backslash-r, backslash-n,
// backslash-u escapes) rather than raw control characters; RFC 8785 requires
// the canonical output to be identical either way.
func jcsSuite() Suite {
	inputs := []struct {
		name, desc, raw string
	}{
		{"rfc8785-appendix-a", "RFC 8785 Appendix A sample: numbers, escaped string and literals.",
			`{"numbers": [333333333.33333329, 1E30, 4.50, 2e-3, 0.000000000000000000000000001], "string": "\u20ac$\u000F\u000aA'\u0042\u0022\u005c\\\"\/", "literals": [null, true, false]}`},
		{"rfc8785-unicode-sorting", "RFC 8785 section 3.2.3: keys sorted by UTF-16 code units.",
			`{"€": "Euro Sign", "\r": "Carriage Return", "דּ": "Hebrew Letter Dalet With Dagesh", "1": "One", "😂": "Smiley", "\u0080": "Control", "ö": "Latin Small Letter O With Diaeresis", "\n": "Newline"}`},
		{"agent-card-like", "Nested object with the shape of an A2A agent card (proof removed before signing).",
			`{"type":["Agent","AIAgent"],"name":"vector-agent","id":"did:sage:ethereum:0x000000000000000000000000000000000000abcd","publicKey":[{"type":"Ed25519VerificationKey2020","publicKeyHex":"00","publicKeyBase58":"1","id":"did:sage:ethereum:0x000000000000000000000000000000000000abcd#key-1","controller":"did:sage:ethereum:0x000000000000000000000000000000000000abcd"}],"@context":["https://www.w3.org/ns/did/v1"],"service":[],"created":"2026-01-01T00:00:00Z","updated":"2026-01-01T00:00:00Z","description":""}`},
		{"numbers", "Integers, negative zero, exponents and fractions per ECMAScript Number::toString.",
			`{"i":10,"z":-0,"e":1e21,"f":1.0,"g":0.1,"h":100000000000000000000,"j":123456789012345680000,"k":-1.5e-7}`},
	}
	s := Suite{Name: "jcs", Description: "RFC 8785 JSON Canonicalization Scheme as applied to HPKE response envelopes and A2A agent-card proofs."}
	for _, c := range inputs {
		s.Cases = append(s.Cases, Case{
			Name: c.name, Mode: ModeDeterministic, Description: c.desc,
			Input: map[string]any{"json": c.raw},
			Produce: func(in map[string]any) (map[string]any, error) {
				j, err := str(in, "json")
				if err != nil {
					return nil, err
				}
				out, err := jcs.Canonicalize([]byte(j))
				if err != nil {
					return nil, err
				}
				sum := sha256.Sum256(out)
				return map[string]any{"canonical": string(out), "canonical_hex": hx(out), "sha256": hx(sum[:])}, nil
			},
		})
	}
	return s
}
