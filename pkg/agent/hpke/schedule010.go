package hpke

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"io"

	"golang.org/x/crypto/hkdf"
)

var errSchedule010 = errors.New("invalid 0.10.0 HPKE schedule input")

// CombineSecrets010 derives the 0.10.0 seed from a verified HPKE exporter,
// a nonzero E2E X25519 shared result, and SHA256(JCS(T)). All inputs are 32 bytes.
// It does not perform DH, authenticate T, verify current keys, or create a session.
// The caller must reject invalid HPKE KEM results before obtaining the exporter.
// The caller owns and must erase the returned seed when no longer needed.
func CombineSecrets010(exporter, shared, th []byte) ([]byte, error) {
	return combineSecretsFor(profile010, exporter, shared, th)
}

// CombineSecrets011 is CombineSecrets010 with the SAGE 0.11.0 combiner label.
func CombineSecrets011(exporter, shared, th []byte) ([]byte, error) {
	return combineSecretsFor(profile011, exporter, shared, th)
}

func combineSecretsFor(p *profile, exporter, shared, th []byte) ([]byte, error) {
	if len(exporter) != 32 || len(shared) != 32 || len(th) != 32 || isAllZero32(shared) {
		return nil, errSchedule010
	}
	var ikm [64]byte
	copy(ikm[:32], exporter)
	copy(ikm[32:], shared)
	defer zeroBytes(ikm[:])
	prk := hkdf.Extract(sha256.New, ikm[:], th)
	defer zeroBytes(prk)
	return expand010(prk, p.combiner, th)
}

func expand010(key []byte, label string, th []byte) ([]byte, error) {
	info := append([]byte(label), th...)
	out := make([]byte, 32)
	if _, err := io.ReadFull(hkdf.Expand(sha256.New, key, info), out); err != nil {
		zeroBytes(out)
		return nil, errSchedule010
	}
	return out, nil
}

// MakeAckTag010 derives the transcript-bound ACK without exposing the ACK key.
// The seed must come from CombineSecrets010 for the same authenticated transcript.
func MakeAckTag010(seed, th []byte) ([]byte, error) { return makeAckTag(profile010, seed, th) }

// MakeAckTag011 is MakeAckTag010 with the SAGE 0.11.0 ACK label.
func MakeAckTag011(seed, th []byte) ([]byte, error) { return makeAckTag(profile011, seed, th) }

func makeAckTag(p *profile, seed, th []byte) ([]byte, error) {
	if len(seed) != 32 || len(th) != 32 {
		return nil, errSchedule010
	}
	key, err := expand010(seed, p.ack, th)
	if err != nil {
		return nil, err
	}
	defer zeroBytes(key)
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(th)
	return mac.Sum(nil), nil
}

// VerifyAckTag010 compares a 32-byte ACK in constant time. Success proves only
// agreement on this seed/transcript, not signatures, key status, or pending state.
func VerifyAckTag010(seed, th, tag []byte) bool { return verifyAckTagFor(profile010, seed, th, tag) }

// VerifyAckTag011 is VerifyAckTag010 with the SAGE 0.11.0 ACK label.
func VerifyAckTag011(seed, th, tag []byte) bool { return verifyAckTagFor(profile011, seed, th, tag) }

func verifyAckTagFor(p *profile, seed, th, tag []byte) bool {
	if len(tag) != 32 {
		return false
	}
	expected, err := makeAckTag(p, seed, th)
	if err != nil {
		return false
	}
	defer zeroBytes(expected)
	return hmac.Equal(expected, tag)
}
