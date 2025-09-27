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


type KeyIDBinder interface { IssueKeyID(ctxID string) (keyid string, ok bool) }

type Server struct {
	a2a.UnimplementedA2AServiceServer

	key  	 sagecrypto.KeyPair
	DID   string
	resolver did.Resolver

	sessMgr   *session.Manager 
	info      InfoBuilder

	maxSkew   time.Duration
	nonces    *nonceStore

	binder    KeyIDBinder            // (선택) kid 발급 위임
}

type ServerOpts struct {
	MaxSkew   time.Duration
	Binder    KeyIDBinder
	Info      InfoBuilder
}

func NewServer(key sagecrypto.KeyPair, sessMgr *session.Manager, did string, resolver did.Resolver, opts *ServerOpts) *Server {
	if opts == nil { opts = &ServerOpts{} }
	ib := opts.Info; if ib == nil { ib = DefaultInfoBuilder{} }
	ms := opts.MaxSkew; if ms == 0 { ms = 2 * time.Minute }
	return &Server{
		key:  	 key,
		resolver: resolver,
		sessMgr:  sessMgr,
		info:     ib,
		DID:  	  did,
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
		return nil, fmt.Errorf("initDID mismatch (meta=%s payload=%s)", senderDID, pl.InitDID)
	}

	now := time.Now()
	if pl.Timestamp.Before(now.Add(-s.maxSkew)) || pl.Timestamp.After(now.Add(s.maxSkew)) { 
		return nil, fmt.Errorf("ts out of window") 
	}

	if !s.nonces.checkAndMark(msg.GetContextId() + "|" + pl.Nonce) { 
		return nil, fmt.Errorf("replay detected") 
	}

	// 3) Canonical info/exportCtx 일치 검사
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
		return nil, fmt.Errorf("HPKE open derive: %v", err)
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

	// 6) kid 발급/바인딩
	kid := "kid-" + uuid.NewString()
	if s.binder != nil {
		if v, ok := s.binder.IssueKeyID(msg.GetContextId()); ok && v != "" { kid = v }
	}
	s.sessMgr.BindKeyID(kid, sid)

	// 7) **키 확인(HMAC 태그)** — SID/seed 미노출
	// ackTag = HMAC(HKDF(exporter,"ack-key"), "hpke-ack|ctxID|nonce|kid")
	ackTag := makeAckTag(se, msg.GetContextId(), pl.Nonce, kid)

	// 8) 동기 응답(Task.Metadata): kid + ackTagB64 만
	meta := map[string]any{
		"note":     "hpke_session_ready",
		"context":  msg.GetContextId(),
		"initDid":  pl.RespDID,
		"respDid":  pl.RespDID,
		"kid":      kid,
		"ackTagB64": base64.RawURLEncoding.EncodeToString(ackTag),
		"ts":       time.Now().UTC().Format(time.RFC3339Nano),
	}
	task := &a2a.Task{
		Id:        msg.TaskId,
		ContextId: msg.GetContextId(),
		Metadata:  toStruct(meta),
		Status:    &a2a.TaskStatus{ State: a2a.TaskState_TASK_STATE_SUBMITTED, Timestamp: timestamppb.Now() },
	}
	return &a2a.SendMessageResponse{ Payload: &a2a.SendMessageResponse_Task{ Task: task } }, nil
}
