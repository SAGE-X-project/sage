package vectors

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/sage-x-project/sage/pkg/agent/session"
)

const (
	sessionLabel   = "sage/hpke+e2e v1"
	sessionPlainA  = "hello from agent A"
	sessionPlainB  = "hello from agent B"
	sessionAADTest = "aad:message-1"
)

func sessionParams() session.Params {
	return session.Params{
		ContextID: vecCtxID,
		SelfEph:   seed32("sage-spec/vectors/session/ephA"),
		PeerEph:   seed32("sage-spec/vectors/session/ephB"),
		Label:     sessionLabel,
	}
}

func sessionSuite() Suite {
	sharedSecret := seed32("sage-spec/vectors/session/shared-secret")
	baseInput := map[string]any{
		"shared_secret": hx(sharedSecret),
		"context_id":    vecCtxID,
		"eph_a":         hx(seed32("sage-spec/vectors/session/ephA")),
		"eph_b":         hx(seed32("sage-spec/vectors/session/ephB")),
		"label":         sessionLabel,
	}
	return Suite{
		Name: "session",
		Description: "Session layer: seed and id derivation, ChaCha20-Poly1305 with the wire format " +
			"seq(8, big-endian) || nonce(12) || ciphertext, AAD = seq || caller AAD, 1024-entry replay window, " +
			"key rotation every RekeyInterval messages (HKDF info \"sage-session-rekey-v1\" || direction || be64(generation)). " +
			"Nonces are random, so ciphertext vectors are verify-only: the check decrypts them with a fresh session built from the same seed.",
		Cases: []Case{
			{
				Name: "seed-and-id", Mode: ModeDeterministic,
				Description: "seed = HKDF-Extract(SHA-256, sharedSecret, salt = SHA256(label || ctx || lo || hi)) with (lo, hi) the byte-sorted ephemeral keys; " +
					"id = base64url-raw(SHA256(label || seed)[:16]).",
				Input: baseInput,
				Produce: func(in map[string]any) (map[string]any, error) {
					ss, err := unhex(in, "shared_secret")
					if err != nil {
						return nil, err
					}
					p := sessionParams()
					seed, err := session.DeriveSessionSeed(ss, p)
					if err != nil {
						return nil, err
					}
					sid, err := session.ComputeSessionIDFromSeed(seed, p.Label)
					if err != nil {
						return nil, err
					}
					return map[string]any{"seed": hx(seed), "session_id": sid}, nil
				},
			},
			{
				Name: "encrypt-decrypt", Mode: ModeVerify,
				Description: "Encrypt/Decrypt with the shared (non-directional) key at seq 0 and, after 256 messages, at seq 256 (generation 1, rotated key). " +
					"The check builds a receiver from the same seed and id and decrypts both records in order.",
				Input: withRekey(baseInput, 256, sessionPlainA),
				Produce: func(in map[string]any) (map[string]any, error) {
					s, err := sessionFromInput(in)
					if err != nil {
						return nil, err
					}
					defer func() { _ = s.Close() }()
					pt := []byte(in["plaintext"].(string))
					first, err := s.Encrypt(pt)
					if err != nil {
						return nil, err
					}
					var last []byte
					for i := 0; i < 256; i++ {
						last, err = s.Encrypt(pt)
						if err != nil {
							return nil, err
						}
					}
					return map[string]any{
						"seq0":        hx(first),
						"seq0_header": hx(first[:session.HeaderSize]),
						"seq256":      hx(last),
						"seq256_seq":  binary.BigEndian.Uint64(last[:session.SeqSize]),
					}, nil
				},
				Verify: func(in, out map[string]any) error {
					s, err := sessionFromInput(in)
					if err != nil {
						return err
					}
					defer func() { _ = s.Close() }()
					want := []byte(in["plaintext"].(string))
					for _, key := range []string{"seq0", "seq256"} {
						ct, err := unhex(out, key)
						if err != nil {
							return err
						}
						got, err := s.Decrypt(ct)
						if err != nil {
							return fmt.Errorf("%s: %w", key, err)
						}
						if !bytes.Equal(got, want) {
							return fmt.Errorf("%s: plaintext mismatch", key)
						}
					}
					ct, _ := unhex(out, "seq0")
					if _, err := s.Decrypt(ct); !errors.Is(err, session.ErrReplayedMessage) {
						return fmt.Errorf("replay of seq0 not rejected: %v", err)
					}
					return nil
				},
			},
			{
				Name: "directional-with-aad", Mode: ModeVerify,
				Description: "EncryptOutbound/DecryptInbound with directional keys (initiator sends on c2s) and EncryptWithAAD/DecryptWithAAD binding caller AAD after the seq header.",
				Input:       withRekey(baseInput, 0, sessionPlainB),
				Produce: func(in map[string]any) (map[string]any, error) {
					init, err := sessionWithRole(in, true)
					if err != nil {
						return nil, err
					}
					defer func() { _ = init.Close() }()
					pt := []byte(in["plaintext"].(string))
					out, err := init.EncryptOutbound(pt)
					if err != nil {
						return nil, err
					}
					aadCT, err := init.EncryptWithAAD(pt, []byte(sessionAADTest))
					if err != nil {
						return nil, err
					}
					return map[string]any{"c2s": hx(out), "aad": sessionAADTest, "with_aad": hx(aadCT)}, nil
				},
				Verify: func(in, out map[string]any) error {
					resp, err := sessionWithRole(in, false)
					if err != nil {
						return err
					}
					defer func() { _ = resp.Close() }()
					want := []byte(in["plaintext"].(string))
					ct, err := unhex(out, "c2s")
					if err != nil {
						return err
					}
					got, err := resp.DecryptInbound(ct)
					if err != nil {
						return fmt.Errorf("c2s: %w", err)
					}
					if !bytes.Equal(got, want) {
						return errors.New("c2s: plaintext mismatch")
					}
					aadCT, err := unhex(out, "with_aad")
					if err != nil {
						return err
					}
					// The AAD record was sealed with the outbound directional key at seq 1;
					// the receiver opens it with its inbound key.
					plain, err := resp.DecryptWithAAD(aadCT, []byte(out["aad"].(string)))
					if err != nil {
						return fmt.Errorf("with_aad: %w", err)
					}
					if !bytes.Equal(plain, want) {
						return errors.New("with_aad: plaintext mismatch")
					}
					return nil
				},
			},
		},
	}
}

func withRekey(base map[string]any, interval uint64, plaintext string) map[string]any {
	m := map[string]any{}
	for k, v := range base {
		m[k] = v
	}
	m["rekey_interval"] = interval
	m["plaintext"] = plaintext
	return m
}

func seedAndID(in map[string]any) ([]byte, string, error) {
	ss, err := unhex(in, "shared_secret")
	if err != nil {
		return nil, "", err
	}
	p := sessionParams()
	seed, err := session.DeriveSessionSeed(ss, p)
	if err != nil {
		return nil, "", err
	}
	sid, err := session.ComputeSessionIDFromSeed(seed, p.Label)
	return seed, sid, err
}

func sessionConfig(in map[string]any) session.Config {
	var interval uint64
	switch v := in["rekey_interval"].(type) {
	case uint64:
		interval = v
	case float64:
		interval = uint64(v)
	}
	return session.Config{RekeyInterval: interval}
}

func sessionFromInput(in map[string]any) (*session.SecureSession, error) {
	seed, sid, err := seedAndID(in)
	if err != nil {
		return nil, err
	}
	return session.NewSecureSession(sid, seed, sessionConfig(in))
}

func sessionWithRole(in map[string]any, initiator bool) (*session.SecureSession, error) {
	seed, sid, err := seedAndID(in)
	if err != nil {
		return nil, err
	}
	return session.NewSecureSessionFromExporterWithRole(sid, seed, initiator, sessionConfig(in))
}
