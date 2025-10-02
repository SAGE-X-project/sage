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

// SPDX-License-Identifier: LGPL-3.0-or-later


package hpke

import (
	"context"
	"crypto"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	sagecrypto "github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/keys"
	"github.com/sage-x-project/sage/did"
	"github.com/sage-x-project/sage/session"
	"google.golang.org/protobuf/types/known/timestamppb"

	a2a "github.com/a2aproject/a2a/grpc"
)

type KeyIDBinder interface {
	IssueKeyID(ctxID string) (keyid string, ok bool)
}

type Server struct {
	a2a.UnimplementedA2AServiceServer

	key      sagecrypto.KeyPair
	DID      string
	resolver did.Resolver

	sessMgr *session.Manager
	info    InfoBuilder

	maxSkew time.Duration
	nonces  *nonceStore

	binder KeyIDBinder // optional hook to issue a custom kid
}

type ServerOpts struct {
	MaxSkew time.Duration
	Binder  KeyIDBinder
	Info    InfoBuilder
}

func NewServer(key sagecrypto.KeyPair, sessMgr *session.Manager, did string, resolver did.Resolver, opts *ServerOpts) *Server {
	if opts == nil {
		opts = &ServerOpts{}
	}
	ib := opts.Info
	if ib == nil {
		ib = DefaultInfoBuilder{}
	}
	ms := opts.MaxSkew
	if ms == 0 {
		ms = 2 * time.Minute
	}
	return &Server{
		key:      key,
		resolver: resolver,
		sessMgr:  sessMgr,
		info:     ib,
		DID:      did,
		maxSkew:  ms,
		nonces:   newNonceStore(10 * time.Minute),
		binder:   opts.Binder,
	}
}

func (s *Server) SendMessage(ctx context.Context, in *a2a.SendMessageRequest) (*a2a.SendMessageResponse, error) {
	if in == nil || in.Request == nil {
		return nil, errors.New("empty request")
	}
	msg := in.Request

	if msg.TaskId != TaskHPKEComplete {
		return nil, fmt.Errorf("unsupported task: %s", msg.TaskId)
	}
	if len(msg.Content) == 0 || msg.Content[0].GetData() == nil || msg.Content[0].GetData().Data == nil {
		return nil, errors.New("missing data part")
	}
	st := msg.Content[0].GetData().Data

	var senderPub crypto.PublicKey
	var senderDID string
	if s.resolver != nil {
		v := in.Metadata.GetFields()["did"]
		if v == nil || v.GetStringValue() == "" {
			return nil, errors.New("missing did")
		}
		senderDID = v.GetStringValue()
		senderPub, _ = s.resolver.ResolvePublicKey(ctx, did.AgentDID(senderDID))
		if senderPub == nil {
			return nil, errors.New("cannot resolve sender pubkey")
		}
	}

	// Verify sender signature if metadata is present.
	if in.Metadata != nil {
		if err := verifySenderSignature(msg, in.Metadata, senderPub); err != nil {
			return nil, fmt.Errorf("signature verification failed: %w", err)
		}
	}

	pl, err := ParseHPKEInitPayload(st)
	if err != nil {
		return nil, fmt.Errorf("parse payload fail: %w", err)
	}

	if senderDID != "" && senderDID != pl.InitDID {
		// Log detailed error for debugging (consider using proper logging framework)
		// log.Debugf("initDID mismatch: meta=%s payload=%s", senderDID, pl.InitDID)
		return nil, fmt.Errorf("authentication failed")
	}

	now := time.Now()
	if pl.Timestamp.Before(now.Add(-s.maxSkew)) || pl.Timestamp.After(now.Add(s.maxSkew)) {
		return nil, fmt.Errorf("ts out of window")
	}

	if !s.nonces.checkAndMark(msg.GetContextId() + "|" + pl.Nonce) {
		return nil, fmt.Errorf("replay detected")
	}

	// Step 3: ensure canonical info and exportCtx match what we expect.
	cInfo := s.info.BuildInfo(msg.GetContextId(), pl.InitDID, pl.RespDID)
	if string(cInfo) != string(pl.Info) {
		return nil, fmt.Errorf("info mismatch")
	}
	cExport := s.info.BuildExportContext(msg.GetContextId())
	if string(cExport) != string(pl.ExportCtx) {
		return nil, fmt.Errorf("exportCtx mismatch")
	}

	se, err := keys.HPKEOpenSharedSecretWithPriv(s.key.PrivateKey(), pl.Enc, pl.Info, pl.ExportCtx, 32)
	if err != nil {
		// Log detailed error for debugging (consider using proper logging framework)
		// log.Debugf("HPKE open derive failed: %v", err)
		return nil, fmt.Errorf("key exchange failed")
	}

	_, sid, _, err := s.sessMgr.EnsureSessionFromExporterWithRole(
		se,
		"sage/hpke v1",
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("fail session create: %w", err)
	}

	// Step 6: issue and bind the key ID.
	kid := "kid-" + uuid.NewString()
	if s.binder != nil {
		if v, ok := s.binder.IssueKeyID(msg.GetContextId()); ok && v != "" {
			kid = v
		}
	}
	s.sessMgr.BindKeyID(kid, sid)

	// Step 7: key confirmation (HMAC tag) keeps SID and seed implicit.
	// ackTag = HMAC(HKDF(exporter,"ack-key"), "hpke-ack|ctxID|nonce|kid")
	ackTag := makeAckTag(se, msg.GetContextId(), pl.Nonce, kid)

	// Step 8: synchronous Task metadata only carries kid and ackTagB64.
	meta := map[string]any{
		"note":      "hpke_session_ready",
		"context":   msg.GetContextId(),
		"initDid":   pl.RespDID,
		"respDid":   pl.RespDID,
		"kid":       kid,
		"ackTagB64": base64.RawURLEncoding.EncodeToString(ackTag),
		"ts":        time.Now().UTC().Format(time.RFC3339Nano),
	}
	task := &a2a.Task{
		Id:        msg.TaskId,
		ContextId: msg.GetContextId(),
		Metadata:  toStruct(meta),
		Status:    &a2a.TaskStatus{State: a2a.TaskState_TASK_STATE_SUBMITTED, Timestamp: timestamppb.Now()},
	}
	return &a2a.SendMessageResponse{Payload: &a2a.SendMessageResponse_Task{Task: task}}, nil
}
