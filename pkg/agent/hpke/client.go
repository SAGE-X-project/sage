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
	"crypto/ecdh"
	"crypto/hmac"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	a2a "github.com/a2aproject/a2a/grpc"
	"github.com/google/uuid"
	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/sage-x-project/sage/pkg/agent/session"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// Client performs the HPKE-based initialization and session creation.
type Client struct {
	a2a      a2a.A2AServiceClient
	resolver did.Resolver
	key      sagecrypto.KeyPair // Ed25519 used to sign A2A Message (metadata)
	DID      string
	info     InfoBuilder
	sessMgr  *session.Manager
}

func NewClient(conn grpc.ClientConnInterface, resolver did.Resolver, key sagecrypto.KeyPair, didStr string, ib InfoBuilder, sessMgr *session.Manager) *Client {
	if ib == nil {
		ib = DefaultInfoBuilder{}
	}
	return &Client{
		a2a:      a2a.NewA2AServiceClient(conn),
		resolver: resolver,
		key:      key,
		DID:      didStr,
		info:     ib,
		sessMgr:  sessMgr,
	}
}

// Initialize performs HPKE Base sender-side derivation, mixes an E2E ephemeral
// X25519 DH with the HPKE exporter, verifies the server-signed response,
// and creates/binds a session keyed by kid.
func (c *Client) Initialize(ctx context.Context, ctxID, initDID, peerDID string) (kid string, err error) {
	// 1) Resolve peer's KEM (X25519) public key.
	peerKEM, err := c.resolvePeerKEM(ctx, peerDID)
	if err != nil {
		return "", err
	}

	// 2) Build HPKE info/export contexts (stable transcript inputs).
	info := c.info.BuildInfo(ctxID, initDID, peerDID)
	exportCtx := c.info.BuildExportContext(ctxID)

	// 3) Derive HPKE sender secrets: enc (ephemeral HPKE pub) and exporter.
	enc, exporterHPKE, err := c.deriveHPKESenderSecrets(peerKEM, info, exportCtx)
	if err != nil {
		return "", err
	}

	// 4) Generate client ephemeral X25519 key for additional E2E DH.
	ephCpriv, ephCPubBytes, err := genEphX25519()
	if err != nil {
		return "", err
	}

	// 5) Build HPKE-init message and sign its metadata (DID-bound).
	nonce := uuid.NewString()
	msg, meta, err := c.buildAndSignInitMsg(ctxID, initDID, peerDID, info, exportCtx, nonce, enc, ephCPubBytes)
	if err != nil {
		return "", err
	}

	// 6) Send the request and receive a server-signed A2A Message response.
	m, err := c.sendAndGetSignedMsg(ctx, msg, meta)
	if err != nil {
		return "", err
	}

	// 7) Verify the server's signature (metadata.signature) over (message without metadata).
	if err := c.verifyServerMessageSig(ctx, m); err != nil {
		return "", err
	}

	// 8) Extract response fields: kid, ackTagB64, ephS.
	st := m.Content[0].GetData().GetData()
	kid, ackTag, ephSbytes, err := parseServerFields(st)
	if err != nil {
		return "", err
	}

	// 9) Compute the E2E DH: ssE2E = X25519(ephCpriv, ephSPub).
	ssE2E, err := computeE2ESecret(ephCpriv, ephSbytes)
	if err != nil {
		return "", err
	}

	// 10) Combine HPKE exporter and E2E secret using HKDF with exportCtx as salt.
	combined, err := CombineSecrets(exporterHPKE, ssE2E, exportCtx)
	if err != nil {
		return "", fmt.Errorf("combine: %w", err)
	}

	// 11) Verify server's key confirmation tag (ackTag).
	if err := verifyAckTag(combined, ctxID, nonce, kid, info, exportCtx, enc, ephCPubBytes, ephSbytes, initDID, peerDID, ackTag); err != nil {
		return "", err
	}

	// 12) Create session as the initiator and bind the key ID.
	if err := c.createAndBindSession(combined, kid); err != nil {
		return "", err
	}
	return kid, nil
}

// Resolve KEM public key of the peer by DID.
func (c *Client) resolvePeerKEM(ctx context.Context, peerDID string) (*ecdh.PublicKey, error) {
	if c.resolver == nil {
		return nil, fmt.Errorf("nil Resolver")
	}
	peerPub, err := c.resolver.ResolveKEMKey(ctx, did.AgentDID(peerDID))
	if err != nil || peerPub == nil {
		return nil, fmt.Errorf("cannot resolve receiver KEM pubkey: %w", err)
	}

	// Type assert to *ecdh.PublicKey
	kemPub, ok := peerPub.(*ecdh.PublicKey)
	if !ok {
		return nil, fmt.Errorf("expected *ecdh.PublicKey, got %T", peerPub)
	}

	return kemPub, nil
}

// HPKE sender-side derivation: returns enc and exporter.
func (c *Client) deriveHPKESenderSecrets(peerKEM *ecdh.PublicKey, info, exportCtx []byte) (enc, exporter []byte, err error) {
	enc, exporter, err = keys.HPKEDeriveSharedSecretToPeer(peerKEM, info, exportCtx, 32)
	if err != nil {
		return nil, nil, fmt.Errorf("HPKE sender derive: %v", err)
	}
	if len(enc) != 32 || len(exporter) != 32 {
		return nil, nil, fmt.Errorf("unexpected sizes: enc=%d exporter=%d", len(enc), len(exporter))
	}
	return enc, exporter, nil
}

// Create ephemeral X25519 key pair and return (priv, pubBytes).
func genEphX25519() (priv *ecdh.PrivateKey, pubBytes []byte, err error) {
	x := ecdh.X25519()
	priv, err = x.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("ephC gen: %w", err)
	}
	return priv, priv.PublicKey().Bytes(), nil
}

// Build an A2A Message for HPKE init and sign its metadata using DID key.
func (c *Client) buildAndSignInitMsg(ctxID, initDID, peerDID string, info, exportCtx []byte, nonce string, enc, ephCPubBytes []byte) (*a2a.Message, *structpb.Struct, error) {
	pl := map[string]any{
		"initDid":   initDID,
		"respDid":   peerDID,
		"info":      string(info),
		"exportCtx": string(exportCtx),
		"nonce":     nonce,
		"ts":        time.Now().UTC().Format(time.RFC3339Nano),
		"enc":       base64.RawURLEncoding.EncodeToString(enc),
		"ephC":      base64.RawURLEncoding.EncodeToString(ephCPubBytes),
	}
	msg := &a2a.Message{
		MessageId: uuid.NewString(),
		ContextId: ctxID,
		TaskId:    TaskHPKEComplete,
		Role:      a2a.Role_ROLE_USER,
		Content:   []*a2a.Part{{Part: &a2a.Part_Data{Data: &a2a.DataPart{Data: toStruct(pl)}}}},
	}
	// Sign the message bytes WITHOUT metadata.
	bin, err := marshalForSig(msg)
	if err != nil {
		return nil, nil, fmt.Errorf("marshalForSig: %w", err)
	}
	meta, err := signStruct(c.key, bin, c.DID)
	if err != nil {
		return nil, nil, fmt.Errorf("sign meta: %w", err)
	}
	return msg, meta, nil
}

// Send and receive a server-signed Message (SendMessageResponse_Msg).
func (c *Client) sendAndGetSignedMsg(ctx context.Context, msg *a2a.Message, meta *structpb.Struct) (*a2a.Message, error) {
	resp, err := c.a2a.SendMessage(ctx, &a2a.SendMessageRequest{Request: msg, Metadata: meta})
	if err != nil {
		return nil, fmt.Errorf("a2a send: %w", err)
	}
	m := resp.GetMsg()
	if m == nil || m.Metadata == nil || len(m.Content) == 0 || m.Content[0].GetData() == nil {
		return nil, fmt.Errorf("empty msg or metadata")
	}
	return m, nil
}

// Verify server's metadata signature over the message (excluding metadata).
func (c *Client) verifyServerMessageSig(ctx context.Context, m *a2a.Message) error {
	v := m.Metadata.GetFields()["did"]
	if v == nil || v.GetStringValue() == "" {
		return errors.New("missing did")
	}
	senderDID := v.GetStringValue()
	senderPub, err := c.resolver.ResolvePublicKey(ctx, did.AgentDID(senderDID))
	if err != nil || senderPub == nil {
		return errors.New("cannot resolve sender pubkey")
	}
	unsigned := proto.Clone(m).(*a2a.Message)
	unsigned.Metadata = nil
	return verifySenderSignature(unsigned, m.Metadata, senderPub)
}

// Parse server fields (kid, ackTag, ephS) from the data struct.
func parseServerFields(st *structpb.Struct) (kid string, ackTag []byte, ephSbytes []byte, err error) {
	kid, err = getString(st, "kid")
	if err != nil || kid == "" {
		return "", nil, nil, fmt.Errorf("missing kid: %w", err)
	}
	ackTagB64, err := getString(st, "ackTagB64")
	if err != nil || ackTagB64 == "" {
		return "", nil, nil, fmt.Errorf("missing ackTagB64: %w", err)
	}
	ackTag, err = base64.RawURLEncoding.DecodeString(ackTagB64)
	if err != nil {
		return "", nil, nil, fmt.Errorf("ackTag b64: %w", err)
	}
	ephSB64, err := getString(st, "ephS")
	if err != nil || ephSB64 == "" {
		return "", nil, nil, fmt.Errorf("missing ephS: %w", err)
	}
	ephSbytes, err = base64.RawURLEncoding.DecodeString(ephSB64)
	if err != nil {
		return "", nil, nil, fmt.Errorf("ephS b64: %w", err)
	}
	if len(ephSbytes) != 32 {
		return "", nil, nil, fmt.Errorf("bad ephS length: %d", len(ephSbytes))
	}
	return kid, ackTag, ephSbytes, nil
}

// Compute ssE2E = X25519(ephCpriv, ephSPub).
func computeE2ESecret(ephCpriv *ecdh.PrivateKey, ephSbytes []byte) ([]byte, error) {
	x := ecdh.X25519()
	ephSPub, err := x.NewPublicKey(ephSbytes)
	if err != nil {
		return nil, fmt.Errorf("ephS parse: %w", err)
	}
	ssE2E, err := ephCpriv.ECDH(ephSPub)
	if err != nil {
		return nil, fmt.Errorf("e2e ecdh: %w", err)
	}
	return ssE2E, nil
}

// Constant-time verification of the server's ack tag.
func verifyAckTag(combined []byte, ctxID, nonce, kid string, info, exportCtx, enc, ephCPubBytes, ephSbytes []byte, initDID, peerDID string, ack []byte) error {
	expect := MakeAckTag(
		combined,
		ctxID,
		nonce,
		kid,
		info,
		exportCtx,
		enc,
		ephCPubBytes,
		ephSbytes,
		[]byte(initDID),
		[]byte(peerDID),
	)
	if !hmac.Equal(expect, ack) {
		return fmt.Errorf("ack tag mismatch")
	}
	return nil
}

// Create a session as initiator and bind the provided key ID.
func (c *Client) createAndBindSession(combined []byte, kid string) error {
	_, sid, _, err := c.sessMgr.EnsureSessionFromExporterWithRole(
		combined,
		"sage/hpke+e2e v1",
		true, // initiator
		nil,
	)
	if err != nil {
		return fmt.Errorf("session: %w", err)
	}
	if c.sessMgr != nil {
		c.sessMgr.BindKeyID(kid, sid)
	}
	return nil
}
