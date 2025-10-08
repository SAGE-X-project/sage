// Copyright (C) 2025 sage-x-project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// Copyright (C) 2025 sage-x-project
// SPDX-License-Identifier: LGPL-3.0-or-later

package hpke

import (
	"context"
	"crypto"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"

	a2a "github.com/a2aproject/a2a/grpc"
	"github.com/google/uuid"
	sagecrypto "github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/keys"
	"github.com/sage-x-project/sage/did"
	"github.com/sage-x-project/sage/session"
	"golang.org/x/crypto/hkdf"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// KeyIDBinder optionally lets the server issue custom key IDs.
type KeyIDBinder interface {
	IssueKeyID(ctxID string) (keyid string, ok bool)
}

// Server accepts HPKE init, verifies DID-signature, derives secrets,
// creates a session, and returns a signed A2A Message with kid/ephS/ackTag.
type Server struct {
	a2a.UnimplementedA2AServiceServer

	key      sagecrypto.KeyPair // Ed25519 for signing A2A Message (metadata)
	kem      sagecrypto.KeyPair // X25519 KEM static key (HPKE Base recipient)
	DID      string
	resolver did.Resolver

	sessMgr *session.Manager
	info    InfoBuilder

	maxSkew time.Duration
	nonces  *nonceStore

	binder KeyIDBinder
}

type ServerOpts struct {
	MaxSkew time.Duration
	Binder  KeyIDBinder
	Info    InfoBuilder
	KEM     sagecrypto.KeyPair // X25519 KEM static key
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
		key:      key,
		kem:      opts.KEM,
		resolver: resolver,
		sessMgr:  sessMgr,
		info:     opts.Info,
		DID:      didStr,
		maxSkew:  opts.MaxSkew,
		nonces:   newNonceStore(10 * time.Minute),
		binder:   opts.Binder,
	}
}

// SendMessage handles TaskHPKEComplete: verifies the sender, parses payload,
// derives exporter + E2E secret, creates the session, and replies with a
// server-signed A2A Message (kid/ephS/ackTag).
func (s *Server) SendMessage(ctx context.Context, in *a2a.SendMessageRequest) (*a2a.SendMessageResponse, error) {
	// 1) Basic validation and extract the data struct.
	msg, st, err := validateAndExtract(in)
	if err != nil {
		return nil, err
	}

	// 2) Verify sender DID and signature (metadata.signature over message-without-metadata).
	senderDID, _, err := s.verifySender(ctx, in)
	if err != nil {
		return nil, err
	}

	// 3) Parse HPKE init payload (HPKE Base + client ephC).
	pl, err := ParseHPKEInitPayloadWithEphC(st)
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
	kid, err := s.createSessionAndBindKid(msg.GetContextId(), combined)
	if err != nil {
		return nil, err
	}

	// 9) Compute the key confirmation tag (ackTag).
	ack := MakeAckTag(
		combined,
		msg.GetContextId(),
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

	// 10) Build a signed A2A Message response (SendMessageResponse_Msg).
	out := map[string]any{
		"kid":       kid,
		"ephS":      base64.RawURLEncoding.EncodeToString(ephSPubBytes),
		"ackTagB64": base64.RawURLEncoding.EncodeToString(ack),
		"ts":        time.Now().UTC().Format(time.RFC3339Nano),
	}
	return s.signedMsgResponse(msg, out)
}


// Validate request and extract the first data part as a struct.
func validateAndExtract(in *a2a.SendMessageRequest) (*a2a.Message, *structpb.Struct, error) {
	if in == nil || in.Request == nil {
		return nil, nil, errors.New("empty request")
	}
	msg := in.Request
	if msg.TaskId != TaskHPKEComplete {
		return nil, nil, fmt.Errorf("unsupported task: %s", msg.TaskId)
	}
	if len(msg.Content) == 0 || msg.Content[0].GetData() == nil || msg.Content[0].GetData().Data == nil {
		return nil, nil, errors.New("missing data part")
	}
	return msg, msg.Content[0].GetData().Data, nil
}

// Verify sender DID and metadata signature (over message-without-metadata).
func (s *Server) verifySender(ctx context.Context, in *a2a.SendMessageRequest) (senderDID string, senderPub crypto.PublicKey, err error) {
	if s.resolver == nil {
		return "", nil, errors.New("resolver not configured")
	}
	v := in.Metadata.GetFields()["did"]
	if v == nil || v.GetStringValue() == "" {
		return "", nil, errors.New("missing did")
	}
	senderDID = v.GetStringValue()
	senderPub, err = s.resolver.ResolvePublicKey(ctx, did.AgentDID(senderDID))
	if err != nil || senderPub == nil {
		return "", nil, errors.New("cannot resolve sender pubkey")
	}
	if in.Metadata != nil {
		unsigned := proto.Clone(in.Request).(*a2a.Message)
		unsigned.Metadata = nil
		if err := verifySenderSignature(unsigned, in.Metadata, senderPub); err != nil {
			return "", nil, fmt.Errorf("signature verification failed: %w", err)
		}
	}
	return senderDID, senderPub, nil
}

// Validate DID binding, timestamp window, replay protection, and info/exportCtx.
func (s *Server) validateInitEnvelope(msg *a2a.Message, pl HPKEInitPayload, senderDID string) error {
	if senderDID != "" && senderDID != pl.InitDID {
		return fmt.Errorf("authentication failed")
	}
	now := time.Now()
	if pl.Timestamp.Before(now.Add(-s.maxSkew)) || pl.Timestamp.After(now.Add(s.maxSkew)) {
		return fmt.Errorf("ts out of window")
	}
	if !s.nonces.checkAndMark(msg.GetContextId() + "|" + pl.Nonce) {
		return fmt.Errorf("replay detected")
	}
	cInfo := s.info.BuildInfo(msg.GetContextId(), pl.InitDID, pl.RespDID)
	if string(cInfo) != string(pl.Info) {
		return fmt.Errorf("info mismatch")
	}
	cExport := s.info.BuildExportContext(msg.GetContextId())
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

// Build a server-signed A2A Message response (SendMessageResponse_Msg).
func (s *Server) signedMsgResponse(req *a2a.Message, outData map[string]any) (*a2a.SendMessageResponse, error) {
	msgOut := &a2a.Message{
		MessageId: uuid.NewString(),
		ContextId: req.GetContextId(),
		TaskId:    req.GetTaskId(),
		Role:      a2a.Role_ROLE_AGENT,
		Content: []*a2a.Part{{
			Part: &a2a.Part_Data{Data: &a2a.DataPart{Data: toStruct(outData)}},
		}},
		Metadata: nil,
	}
	// Sign the message bytes WITHOUT metadata.
	bin, err := marshalForSig(msgOut)
	if err != nil {
		return nil, fmt.Errorf("marshalForSig: %w", err)
	}
	meta, err := signStruct(s.key, bin, s.DID)
	if err != nil {
		return nil, fmt.Errorf("sign meta: %w", err)
	}
	msgOut.Metadata = meta
	return &a2a.SendMessageResponse{
		Payload: &a2a.SendMessageResponse_Msg{Msg: msgOut},
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