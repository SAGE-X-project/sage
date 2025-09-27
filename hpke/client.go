package hpke

import (
	"context"
	"crypto/hmac"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	sagecrypto "github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/keys"
	"github.com/sage-x-project/sage/did"
	"github.com/sage-x-project/sage/session"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	a2a "github.com/a2aproject/a2a/grpc"
)

type Client struct {
	a2a      a2a.A2AServiceClient
	resolver did.Resolver
	key 	sagecrypto.KeyPair
	DID  string
	info     InfoBuilder

	sessMgr *session.Manager // EnsureSessionWithExporter/BindKeyID 사용
}

func NewClient(conn grpc.ClientConnInterface, resolver did.Resolver, key sagecrypto.KeyPair, did string, ib InfoBuilder, sessMgr *session.Manager) *Client {
	if ib == nil { ib = DefaultInfoBuilder{} }
	return &Client{ 
		a2a:  a2a.NewA2AServiceClient(conn),
		key: key,
		resolver: resolver, 
		DID: did, 
		info: ib, 
		sessMgr: sessMgr,
	}
}

// Complete: exporter 합의 → 로컬 SID 계산(미노출) → 서버 호출 → ackTag 검증 → kid 바인딩
func (c *Client) Initialize(ctx context.Context, ctxID, initDID, peerDID string) (kid string, err error) {
	if c.resolver == nil { 
		return "", fmt.Errorf("nil Resolver") 
	}
	
	peerPub, _ := c.resolver.ResolvePublicKey(ctx, did.AgentDID(peerDID))
	if peerPub == nil {
		return "", fmt.Errorf("cannot resolve sender pubkey")
	}
	if err != nil { 
		return "", fmt.Errorf("resolve KEM pub: %w", err) 
	}

	// 2) HPKE 파라미터
	info := c.info.BuildInfo(ctxID, initDID, peerDID)
	exportCtx := c.info.BuildExportContext(ctxID)

	enc, s, err := keys.HPKEDeriveSharedSecretToPeer(peerPub, info, exportCtx, 32)
	if err != nil {
		return "", fmt.Errorf("HPKE sender derive: %v", err)
	}
	if len(enc) != 32 || len(s) != 32 {
		return "", fmt.Errorf("unexpected sizes: enc=%d, exporter=%d", len(enc), len(s))
	}

	_, sid, _, err := c.sessMgr.EnsureSessionFromExporterWithRole(
		s,
		"sage/hpke v1",
		true,
		nil,
	)
	if err != nil { 
		return "", fmt.Errorf("fail session create: %w", err) 
	}

	nonce := uuid.NewString()
	pl := map[string]any{
		"initDid":   initDID,
		"respDid":   peerDID,
		"info":      string(info),
		"exportCtx": string(exportCtx),
		"nonce":     nonce,
		"ts":        time.Now().UTC().Format(time.RFC3339Nano),
	}
	putBase64(pl, "enc", enc)
	msg := &a2a.Message{
		MessageId: uuid.NewString(),
		ContextId: ctxID,
		TaskId:    TaskHPKEComplete,
		Role:      a2a.Role_ROLE_USER,
		Content:   []*a2a.Part{{ Part: &a2a.Part_Data{ Data: &a2a.DataPart{ Data: toStruct(pl) }}}},
	}

	bytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("marshal for signing: %w", err)
	}

	meta, err := signStruct(c.key, bytes, c.DID)
	if err != nil { 
		return "", fmt.Errorf("sign meta: %w", err) 
	}
	resp, err := c.a2a.SendMessage(ctx, &a2a.SendMessageRequest{ Request: msg, Metadata: meta })
	
	
	if err != nil { 
		return "", fmt.Errorf("a2a send: %w", err) 
	}
	task := resp.GetTask()
	if task == nil || task.Metadata == nil {
		return "", fmt.Errorf("empty task/metadata")
	}
	md := task.Metadata
	kid, err = getString(md, "kid")
	
	if err != nil { return "", err }
	ackTagB64, err := getString(md, "ackTagB64"); if err != nil { 
		return "", err 
	}
	ackTag, err := base64.RawURLEncoding.DecodeString(ackTagB64)
	if err != nil { 
		return "", fmt.Errorf("ack tag b64: %w", err) 
	}

	// 8) **키확인(HMAC 검증)** — 서버가 exporter를 가졌는지 확인 + kid 바인딩
	expect := makeAckTag(s, ctxID, nonce, kid)
	if !hmac.Equal(expect, ackTag) {
		return "", fmt.Errorf("ack tag mismatch")
	}

	if c.sessMgr != nil {
		c.sessMgr.BindKeyID(kid, sid)
	}
	return kid, nil
}
