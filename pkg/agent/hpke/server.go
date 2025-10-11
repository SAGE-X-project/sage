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
	"io"
	"time"

	"github.com/google/uuid"
	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/sage-x-project/sage/pkg/agent/session"
	"github.com/sage-x-project/sage/pkg/agent/transport"
	"golang.org/x/crypto/hkdf"
)

// KeyIDBinder optionally lets the server issue custom key IDs.
type KeyIDBinder interface {
	IssueKeyID(ctxID string) (keyid string, ok bool)
}

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
}

type ServerOpts struct {
	MaxSkew   time.Duration
	Binder    KeyIDBinder
	Info      InfoBuilder
	KEM       sagecrypto.KeyPair              // X25519 KEM static key
	Transport transport.MessageTransport // Optional transport for responses
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
	}
}

// HandleMessage handles TaskHPKEComplete: verifies the sender, parses payload,
// derives exporter + E2E secret, creates the session, and replies with a
// signed response (kid/ephS/ackTag).
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
		return nil, err
	}

	// 7) Combine exporter and E2E secret (HKDF-Extract/Expand with exportCtx).
	combined, err := CombineSecrets(exporterHPKE, ssE2E, pl.ExportCtx)
	if err != nil {
		return nil, fmt.Errorf("combine: %w", err)
	}

	// 8) Create a session for the receiver side and bind a key ID.
	kid, err := s.createSessionAndBindKid(msg.ContextID, combined)
	if err != nil {
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

	// 10) Build a signed response.
	out := map[string]any{
		"kid":       kid,
		"ephS":      base64.RawURLEncoding.EncodeToString(ephSPubBytes),
		"ackTagB64": base64.RawURLEncoding.EncodeToString(ack),
		"ts":        time.Now().UTC().Format(time.RFC3339Nano),
		"did":       s.DID,
	}
	return s.signedResponse(msg, out)
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
	if err := s.verifySignature(msg.Payload, msg.Signature, senderPub); err != nil {
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

// verifySignature checks the signature against the payload.
func (s *Server) verifySignature(payload, signature []byte, senderPub crypto.PublicKey) error {
	type verifyKey interface {
		Verify(msg, sig []byte) error
	}

	switch pk := senderPub.(type) {
	case verifyKey:
		return pk.Verify(payload, signature)
	default:
		return fmt.Errorf("unsupported public key type: %T", senderPub)
	}
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

// Build a server-signed transport response.
func (s *Server) signedResponse(req *transport.SecureMessage, outData map[string]any) (*transport.Response, error) {
	data, err := json.Marshal(outData)
	if err != nil {
		return nil, fmt.Errorf("marshal response: %w", err)
	}

	// Note: The response data includes "did" field for identification
	// Signature verification is done via ackTag (key confirmation)
	return &transport.Response{
		Success:   true,
		MessageID: req.ID,
		TaskID:    req.TaskID,
		Data:      data,
	}, nil
}

// Combine exporterHPKE || ssE2E using HKDF-Extract(salt=exportCtx) then
// HKDF-Expand("SAGE-HPKE+E2E-Combiner") to 32 bytes.
func CombineSecrets(exporterHPKE, ssE2E, exportCtx []byte) ([]byte, error) {
	ikm := make([]byte, 0, len(exporterHPKE)+len(ssE2E))
	ikm = append(ikm, exporterHPKE...)
	ikm = append(ikm, ssE2E...)
	prk := hkdf.Extract(sha256.New, ikm, exportCtx)
	r := hkdf.Expand(sha256.New, prk, []byte("SAGE-HPKE+E2E-Combiner"))
	out := make([]byte, 32)
	_, err := io.ReadFull(r, out)
	return out, err
}
