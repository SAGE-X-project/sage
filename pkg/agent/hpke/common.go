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

package hpke

import (
	"crypto"
	"crypto/ed25519"

	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/crypto/hkdf"
)

// info is the RFC9180 "info" input and explicitly encodes suite/combiner/context data.
// The order and delimiters are fixed so both sides produce identical byte sequences.
func (DefaultInfoBuilder) BuildInfo(ctxID, initDID, respDID string) []byte {
	return []byte(
		infoLabel +
			"|suite=" + hpkeSuiteID +
			"|combiner=" + combinerID +
			"|ctx=" + ctxID +
			"|init=" + initDID +
			"|resp=" + respDID,
	)
}

// exportCtx is used as the HKDF salt / "export context" value.
// It repeats the domain label, suite, combiner, and context in a fixed order.
func (DefaultInfoBuilder) BuildExportContext(ctxID string) []byte {
	return []byte(
		exportCtxLabel +
			"|suite=" + hpkeSuiteID +
			"|combiner=" + combinerID +
			"|ctx=" + ctxID,
	)
}

// DefaultInfo returns the canonical HPKE transcript info bytes used by SAGE.
func DefaultInfo(ctxID, initDID, respDID string) []byte {
	return DefaultInfoBuilder{}.BuildInfo(ctxID, initDID, respDID)
}

// DefaultExportContext returns the canonical export context bytes used by SAGE.
func DefaultExportContext(ctxID string) []byte {
	return DefaultInfoBuilder{}.BuildExportContext(ctxID)
}

// Combine exporterHPKE || ssE2E using HKDF-Extract(salt=exportCtx) then
// HKDF-Expand("SAGE-HPKE+E2E-Combiner") to 32 bytes.
func combineSecrets(exporterHPKE, ssE2E, exportCtx []byte) ([]byte, error) {
	ikm := make([]byte, 0, len(exporterHPKE)+len(ssE2E))
	ikm = append(ikm, exporterHPKE...)
	ikm = append(ikm, ssE2E...)
	prk := hkdf.Extract(sha256.New, ikm, exportCtx)
	r := hkdf.Expand(sha256.New, prk, []byte("SAGE-HPKE+E2E-Combiner"))
	out := make([]byte, 32)
	_, err := io.ReadFull(r, out)
	return out, err
}

// zeroBytes best-effort clears a byte slice.
func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func isAllZero32(b []byte) bool {
    if len(b) != 32 { return false }
    var z [32]byte
    return subtle.ConstantTimeCompare(b, z[:]) == 1
}

type TrafficKeys struct {
	C2SKey []byte // 32 bytes
	C2SIV  []byte // 12 bytes
	S2CKey []byte // 32 bytes
	S2CIV  []byte // 12 bytes
	CB     []byte // 32 bytes channel-binding value
}

// DeriveTrafficKeys splits a 32-byte seed into directional keys, IVs and CB.
func DeriveTrafficKeys(seed []byte) TrafficKeys {
	return TrafficKeys{
		C2SKey: hkdfExpand(seed, c2sKeyLabel, 32),
		C2SIV:  hkdfExpand(seed, c2sIVLabel, 12),
		S2CKey: hkdfExpand(seed, s2cKeyLabel, 32),
		S2CIV:  hkdfExpand(seed, s2cIVLabel, 12),
		CB:     hkdfExpand(seed, cbLabel, 32),
	}
}


// verifySignature verifies a detached signature in a constant-time friendly way.
func verifySignature(payload, signature []byte, senderPub crypto.PublicKey) error {
	if len(signature) == 0 {
		return errors.New("missing signature")
	}

	// Support either a custom Verify interface or raw ed25519.PublicKey
	type verifyKey interface {
		Verify(msg, sig []byte) error
	}

	switch pk := senderPub.(type) {
	case verifyKey:
		if err := pk.Verify(payload, signature); err != nil {
			return fmt.Errorf("signature verify failed: %w", err)
		}
		return nil
	case ed25519.PublicKey:
		if !ed25519.Verify(pk, payload, signature) {
			return errors.New("signature verify failed: invalid ed25519 signature")
		}
		return nil
	default:
		return fmt.Errorf("unsupported public key type: %T", senderPub)
	}
}

type nonceStore struct {
	ttl     time.Duration
	mu      sync.Mutex
	entries map[string]time.Time
}

func newNonceStore(ttl time.Duration) *nonceStore {
	return &nonceStore{ttl: ttl, entries: make(map[string]time.Time)}
}
func (s *nonceStore) checkAndMark(key string) bool {
	now := time.Now()
	exp := now.Add(s.ttl)
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range s.entries {
		if now.After(v) {
			delete(s.entries, k)
		}
	}
	if _, ok := s.entries[key]; ok {
		return false
	}
	s.entries[key] = exp
	return true
}

func getString(m map[string]string, key string) (string, error) {
	v, ok := m[key]
	if !ok || v == "" {
		return "", fmt.Errorf("missing %s", key)
	}
	return v, nil
}

func getBase64(m map[string]string, key string) ([]byte, error) {
	s, err := getString(m, key)
	if err != nil {
		return nil, err
	}
	return base64.RawURLEncoding.DecodeString(s)
}

func strContains(arr []string, v string) bool {
    for _, x := range arr {
        if x == v { return true }
    }
    return false
}

// ACK (HMAC) - key confirmation without ciphertext
func hkdfExpand(key []byte, info string, outLen int) []byte {
	h := hmac.New(sha256.New, key)
	var out []byte
	var counter uint32 = 1
	for len(out) < outLen {
		h.Reset()
		h.Write([]byte(info))
		var c [4]byte
		binary.BigEndian.PutUint32(c[:], counter)
		h.Write(c[:])
		out = append(out, h.Sum(nil)...)
		counter++
	}
	return out[:outLen]
}

// ackKey = HKDF-Expand(seed, "SAGE-ack-key-v1", 32)
// ackMsg = "SAGE-ack-msg|v1|" || len(ctxID)||ctxID || len(nonce)||nonce || len(kid)||kid || SHA256(transcript...)
// ackTag = HMAC(ackKey, ackMsg)
func MakeAckTag(seed []byte, ctxID, nonce, kid string, binds ...[]byte) []byte {
	ackKey := hkdfExpand(seed, "SAGE-ack-key-v1", 32)

	// length-prefix helper (big-endian 2 bytes)
	writeStr := func(h io.Writer, s string) {
		l := len(s)
		_, _ = h.Write([]byte{byte(l >> 8), byte(l)})
		_, _ = h.Write([]byte(s))
	}

	// transcript hash
	th := sha256.New()
	for _, b := range binds {
		// Add a single-byte delimiter (or length prefix) to avoid ambiguity between segments.
		th.Write([]byte{0})
		th.Write(b)
	}
	transcriptHash := th.Sum(nil)

	mac := hmac.New(sha256.New, ackKey)
	mac.Write([]byte("SAGE-ack-msg|v1|"))
	writeStr(mac, ctxID)
	writeStr(mac, nonce)
	writeStr(mac, kid)
	mac.Write(transcriptHash)
	return mac.Sum(nil)
}

type HPKEInitPayload struct {
	InitDID   string
	RespDID   string
	Info      []byte
	ExportCtx []byte
	Enc       []byte // HPKE enc (sender eph KEM pub) - raw
	EphC      []byte // Client ephemeral X25519 pub - raw (32B)
	Nonce     string
	Timestamp time.Time
}

// Parse: enc and ephC arrive as base64url strings and are converted to raw bytes.
func ParseHPKEInitPayloadWithEphCFromJSON(data []byte) (HPKEInitPayload, error) {
	var out HPKEInitPayload
	var m map[string]string

	if err := json.Unmarshal(data, &m); err != nil {
		return out, fmt.Errorf("unmarshal: %w", err)
	}

	var err error
	if out.InitDID, err = getString(m, "initDid"); err != nil {
		return out, err
	}
	if out.RespDID, err = getString(m, "respDid"); err != nil {
		return out, err
	}
	var infoStr string
	if infoStr, err = getString(m, "info"); err != nil {
		return out, err
	}
	out.Info = []byte(infoStr)

	var exportCtxStr string
	if exportCtxStr, err = getString(m, "exportCtx"); err != nil {
		return out, err
	}
	out.ExportCtx = []byte(exportCtxStr)

	if out.Enc, err = getBase64(m, "enc"); err != nil {
		return out, err
	}
	if out.Nonce, err = getString(m, "nonce"); err != nil {
		return out, err
	}
	tsStr, err := getString(m, "ts")
	if err != nil {
		return out, err
	}
	out.Timestamp, err = time.Parse(time.RFC3339Nano, tsStr)
	if err != nil {
		return out, fmt.Errorf("bad ts: %w", err)
	}

	// ephC is required to provide PFS.
	if out.EphC, err = getBase64(m, "ephC"); err != nil {
		return out, fmt.Errorf("missing ephC: %w", err)
	}
	if l := len(out.EphC); l != 32 {
		return out, fmt.Errorf("bad ephC length: %d", l)
	}
	if l := len(out.Enc); l != 32 {
		return out, fmt.Errorf("bad enc length: %d", l)
	}
	return out, nil
}

type HPKEBaseInitPayload struct {
	InitDID   string
	RespDID   string
	Info      []byte
	ExportCtx []byte
	Enc       []byte // HPKE enc (X25519 KEM encapsulated pub) - raw 32B
	Nonce     string
	Timestamp time.Time
}

func ParseHPKEBaseInitPayloadFromJSON(data []byte) (HPKEBaseInitPayload, error) {
	var out HPKEBaseInitPayload
	var m map[string]string

	if err := json.Unmarshal(data, &m); err != nil {
		return HPKEBaseInitPayload{}, fmt.Errorf("unmarshal: %w", err)
	}

	var err error
	if out.InitDID, err = getString(m, "initDid"); err != nil || out.InitDID == "" {
		return HPKEBaseInitPayload{}, fmt.Errorf("initDid missing: %w", err)
	}
	if out.RespDID, err = getString(m, "respDid"); err != nil || out.RespDID == "" {
		return HPKEBaseInitPayload{}, fmt.Errorf("respDid missing: %w", err)
	}
	var infoStr string
	if infoStr, err = getString(m, "info"); err != nil || infoStr == "" {
		return HPKEBaseInitPayload{}, fmt.Errorf("info missing: %w", err)
	}
	out.Info = []byte(infoStr)

	var exportCtxStr string
	if exportCtxStr, err = getString(m, "exportCtx"); err != nil || exportCtxStr == "" {
		return HPKEBaseInitPayload{}, fmt.Errorf("exportCtx missing: %w", err)
	}
	out.ExportCtx = []byte(exportCtxStr)

	if out.Enc, err = getBase64(m, "enc"); err != nil {
		return HPKEBaseInitPayload{}, err
	}
	if len(out.Enc) != 32 {
		return HPKEBaseInitPayload{}, fmt.Errorf("enc length must be 32, got %d", len(out.Enc))
	}

	if out.Nonce, err = getString(m, "nonce"); err != nil || out.Nonce == "" {
		return HPKEBaseInitPayload{}, fmt.Errorf("nonce missing: %w", err)
	}
	tsStr, err := getString(m, "ts")
	if err != nil {
		return HPKEBaseInitPayload{}, err
	}
	if out.Timestamp, err = time.Parse(time.RFC3339Nano, tsStr); err != nil {
		return HPKEBaseInitPayload{}, fmt.Errorf("bad ts: %w", err)
	}
	return out, nil
}
