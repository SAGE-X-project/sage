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
	"context"
	"crypto"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/sage-x-project/sage/pkg/agent/session"
	"github.com/sage-x-project/sage/pkg/agent/transport"
)

// Server accepts HPKE init, verifies DID-signature, derives secrets,
// creates a session, and returns a signed response with kid/ephS/ackTag.
type Server struct {
	key       sagecrypto.KeyPair // Ed25519 for signing messages
	kem       sagecrypto.KeyPair // X25519 KEM static key (HPKE Base recipient)
	DID       string
	resolver  did.Resolver
	transport transport.MessageTransport // Optional: for sending responses

	sessMgr *session.Manager
	info    InfoBuilder

	maxSkew time.Duration
	nonces  *nonceStore

	binder KeyIDBinder
	cookies CookieVerifier // optional anti-DoS
}

type ServerOpts struct {
	MaxSkew   time.Duration
	Binder    KeyIDBinder
	Info      InfoBuilder
	KEM       sagecrypto.KeyPair            // X25519 KEM static key
	Transport transport.MessageTransport	// Optional transport for responses
	Cookies   CookieVerifier
}

// serverSigEnvelope is the canonical structure signed by the server.
// Field order is fixed to guarantee deterministic JSON encoding.
type serverSigEnvelope struct {
	V             string `json:"v"`
	Task          string `json:"task"`
	Ctx           string `json:"ctx"`
	Kid           string `json:"kid"`
	EphS          string `json:"ephS"`
	AckTagB64     string `json:"ackTagB64"`
	Ts            string `json:"ts"`
	Did           string `json:"did"`
	InfoHash      string `json:"infoHash"`      // b64url(SHA256(info))
	ExportCtxHash string `json:"exportCtxHash"` // b64url(SHA256(exportCtx))
	Enc           string `json:"enc"`           // b64url(sender enc)
	EphC          string `json:"ephC"`          // b64url(client ephC)
}

func NewServer(key sagecrypto.KeyPair, sessMgr *session.Manager, didStr string, resolver did.Resolver, opts *ServerOpts) *Server {
	if opts == nil {
		opts = &ServerOpts{}
	}
	if opts.Info == nil {
		opts.Info = DefaultInfoBuilder{}
	}
	if opts.MaxSkew == 0 {
		opts.MaxSkew = 2 * time.Minute
	}
	return &Server{
		key:       key,
		kem:       opts.KEM,
		resolver:  resolver,
		transport: opts.Transport,
		sessMgr:   sessMgr,
		info:      opts.Info,
		DID:       didStr,
		maxSkew:   opts.MaxSkew,
		nonces:    newNonceStore(10 * time.Minute),
		binder:    opts.Binder,
		cookies:   opts.Cookies,
	}
}

// HandleMessage handles TaskHPKEComplete, does verification/derivations, and replies with a signed envelope
func (s *Server) HandleMessage(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
	// 1) Basic validation
	if msg == nil {
		return nil, errors.New("empty message")
	}
	if msg.TaskID != TaskHPKEComplete {
		return nil, fmt.Errorf("unsupported task: %s", msg.TaskID)
	}

	// 2) Verify sender DID and signature
	senderDID, _, err := s.verifySender(ctx, msg)
	if err != nil {
		return nil, err
	}

	// 3) Parse HPKE init payload (HPKE Base + client ephC).
	pl, err := ParseHPKEInitPayloadWithEphCFromJSON(msg.Payload)
	if err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}

	// 3.5) Optional DoS cookie check (before expensive HPKE)
	if s.cookies != nil {
		cookie := msg.Metadata["cookie"]
		if !s.cookies.Verify(cookie, msg.ContextID, pl.InitDID, pl.RespDID) {
			return nil, fmt.Errorf("cookie required or invalid")
		}
	}

	// 4) Validate DID binding, timestamp window, replay, info/exportCtx.
	if err := s.validateInitEnvelope(msg, pl, senderDID); err != nil {
		return nil, err
	}

	// 5) Reproduce HPKE exporter from server skR and sender enc.
	exporterHPKE, err := s.reproduceExporter(pl)
	if err != nil {
		return nil, err
	}

	// 6) Generate server ephS and compute ssE2E with client ephC.
	ephSPubBytes, ssE2E, err := generateSrvE2E(pl.EphC)
	if err != nil {
		zeroBytes(exporterHPKE)
		return nil, err
	}

	// 7) Combine exporter and E2E secret (HKDF-Extract/Expand with exportCtx).
	combined, err := combineSecrets(exporterHPKE, ssE2E, pl.ExportCtx)
	zeroBytes(exporterHPKE)
	zeroBytes(ssE2E)
	if err != nil {
		return nil, fmt.Errorf("combine: %w", err)
	}

	// 8) Create a session for the receiver side and bind a key ID.
	kid, err := s.createSessionAndBindKid(msg.ContextID, combined)
	if err != nil {
		zeroBytes(combined)
		return nil, err
	}

	// 9) Compute the key confirmation tag (ackTag).
	ack := MakeAckTag(
		combined,
		msg.ContextID,
		pl.Nonce,
		kid,
		pl.Info,
		pl.ExportCtx,
		pl.Enc,
		pl.EphC,
		ephSPubBytes,
		[]byte(pl.InitDID),
		[]byte(pl.RespDID),
	)

	// Best-effort wipe local seed copy
	zeroBytes(combined)

	// 10) Build signed response (envelope + sigB64)
	return s.buildAndSignResponse(msg, pl, kid, ack, ephSPubBytes)
}

// Verify sender DID and signature.
func (s *Server) verifySender(ctx context.Context, msg *transport.SecureMessage) (senderDID string, senderPub crypto.PublicKey, err error) {
	if s.resolver == nil {
		return "", nil, errors.New("resolver not configured")
	}
	if msg.DID == "" {
		return "", nil, errors.New("missing did")
	}
	senderDID = msg.DID
	senderPub, err = s.resolver.ResolvePublicKey(ctx, did.AgentDID(senderDID))
	if err != nil || senderPub == nil {
		return "", nil, errors.New("cannot resolve sender pubkey")
	}

	// Verify over the exact bytes that were signed by the client.
	// Do NOT re-marshal or re-encode here; use msg.Payload as-is.
	if err := verifySignature(msg.Payload, msg.Signature, senderPub); err != nil {
		return "", nil, fmt.Errorf("signature verification failed: %w", err)
	}
	return senderDID, senderPub, nil
}

// Validate DID binding, timestamp window, replay protection, and info/exportCtx.
func (s *Server) validateInitEnvelope(msg *transport.SecureMessage, pl HPKEInitPayload, senderDID string) error {
	if senderDID != "" && senderDID != pl.InitDID {
		return fmt.Errorf("authentication failed")
	}
	now := time.Now()
	if pl.Timestamp.Before(now.Add(-s.maxSkew)) || pl.Timestamp.After(now.Add(s.maxSkew)) {
		return fmt.Errorf("ts out of window")
	}
	if !s.nonces.checkAndMark(msg.ContextID + "|" + pl.Nonce) {
		return fmt.Errorf("replay detected")
	}
	cInfo := s.info.BuildInfo(msg.ContextID, pl.InitDID, pl.RespDID)
	if string(cInfo) != string(pl.Info) {
		return fmt.Errorf("info mismatch")
	}
	cExport := s.info.BuildExportContext(msg.ContextID)
	if string(cExport) != string(pl.ExportCtx) {
		return fmt.Errorf("exportCtx mismatch")
	}
	return nil
}

// Recompute HPKE exporter from server KEM private key and sender enc.
func (s *Server) reproduceExporter(pl HPKEInitPayload) ([]byte, error) {
	if s.kem == nil {
		return nil, fmt.Errorf("server KEM private key not configured")
	}
	exporter, err := keys.HPKEOpenSharedSecretWithPriv(
		s.kem.PrivateKey(), // X25519 KEM skR
		pl.Enc,             // sender enc (32B)
		pl.Info,
		pl.ExportCtx,
		32,
	)
	if err != nil {
		return nil, fmt.Errorf("hpke open: %w", err)
	}
	return exporter, nil
}

// Generate server ephemeral X25519 and compute ssE2E with client ephC.
func generateSrvE2E(ephC []byte) (ephSPubBytes, ssE2E []byte, err error) {
	x := ecdh.X25519()
	ephCPub, err := x.NewPublicKey(ephC)
	if err != nil {
		return nil, nil, fmt.Errorf("bad ephC: %w", err)
	}
	srvPriv, err := x.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("srv eph gen: %w", err)
	}
	sec, err := srvPriv.ECDH(ephCPub)
	if err != nil {
		return nil, nil, fmt.Errorf("e2e ecdh: %w", err)
	}
	return srvPriv.PublicKey().Bytes(), sec, nil
}

// Create a session as receiver and bind a generated (or issued) key ID.
func (s *Server) createSessionAndBindKid(ctxID string, combined []byte) (string, error) {
	_, sid, _, err := s.sessMgr.EnsureSessionFromExporterWithRole(
		combined,
		"sage/hpke+e2e v1",
		false, // receiver
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("session: %w", err)
	}
	kid := "kid-" + uuid.NewString()
	if s.binder != nil {
		if v, ok := s.binder.IssueKeyID(ctxID); ok && v != "" {
			kid = v
		}
	}
	s.sessMgr.BindKeyID(kid, sid)
	return kid, nil
}

// buildAndSignResponse creates the canonical envelope and attaches an Ed25519 signature.
func (s *Server) buildAndSignResponse(req *transport.SecureMessage, pl HPKEInitPayload, kid string, ack []byte, ephSPubBytes []byte) (*transport.Response, error) {
	if s.key == nil {
		return nil, fmt.Errorf("server signing key not configured")
	}

	ts := time.Now().UTC().Format(time.RFC3339Nano)

	ih := sha256.Sum256(pl.Info)
	eh := sha256.Sum256(pl.ExportCtx)
	env := serverSigEnvelope{
		V:             "v1",
		Task:          req.TaskID,
		Ctx:           req.ContextID,
		Kid:           kid,
		EphS:          base64.RawURLEncoding.EncodeToString(ephSPubBytes),
		AckTagB64:     base64.RawURLEncoding.EncodeToString(ack),
		Ts:            ts,
		Did:           s.DID,
		InfoHash:      base64.RawURLEncoding.EncodeToString(ih[:]),
		ExportCtxHash: base64.RawURLEncoding.EncodeToString(eh[:]),
		Enc:           base64.RawURLEncoding.EncodeToString(pl.Enc),
		EphC:          base64.RawURLEncoding.EncodeToString(pl.EphC),
	}

	envBytes, err := json.Marshal(env) // deterministic order by struct field order
	if err != nil {
		return nil, fmt.Errorf("marshal env: %w", err)
	}
	sig, err := s.key.Sign(envBytes) // Ed25519 sign
	if err != nil {
		return nil, fmt.Errorf("sign env: %w", err)
	}

	out := map[string]any{
		// Envelope fields
		"v":             env.V,
		"task":          env.Task,
		"ctx":           env.Ctx,
		"kid":           env.Kid,
		"ephS":          env.EphS,
		"ackTagB64":     env.AckTagB64,
		"ts":            env.Ts,
		"did":           env.Did,
		"infoHash":      env.InfoHash,
		"exportCtxHash": env.ExportCtxHash,
		"enc":           env.Enc,
		"ephC":          env.EphC,
		// Detached signature
		"sigB64": base64.RawURLEncoding.EncodeToString(sig),
	}

	data, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("marshal response: %w", err)
	}
	return &transport.Response{
		Success:   true,
		MessageID: req.ID,
		TaskID:    req.TaskID,
		Data:      data,
	}, nil
}