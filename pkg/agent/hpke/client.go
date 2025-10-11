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
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/sage-x-project/sage/pkg/agent/session"
	"github.com/sage-x-project/sage/pkg/agent/transport"
)

// Client performs the HPKE-based initialization and session creation.
type Client struct {
	transport transport.MessageTransport
	resolver  did.Resolver
	key       sagecrypto.KeyPair // Ed25519 used to sign messages
	DID       string
	info      InfoBuilder
	sessMgr   *session.Manager

	cookies CookieSource 	// optional
	pins map[string][]byte  // DID -> ed25519 pub (TOFU pin)
}

func NewClient(t transport.MessageTransport, resolver did.Resolver, key sagecrypto.KeyPair, didStr string, ib InfoBuilder, sessMgr *session.Manager) *Client {
	if ib == nil {
		ib = DefaultInfoBuilder{}
	}
	return &Client{
		transport: t,
		resolver:  resolver,
		key:       key,
		DID:       didStr,
		info:      ib,
		sessMgr:   sessMgr,
		pins:      make(map[string][]byte),
	}
}

// WithCookieSource enables optional cookie attachment.
func (c *Client) WithCookieSource(src CookieSource) *Client {
	c.cookies = src
	return c
}

// Initialize performs HPKE Base sender-side derivation, mixes E2E DH, verifies ackTag & server signature,
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
		zeroBytes(exporterHPKE)
		return "", err
	}

	// 5) Build HPKE-init message and sign it (DID-bound).
	nonce := uuid.NewString()
	msg, err := c.buildAndSignInitMsg(ctxID, initDID, peerDID, info, exportCtx, nonce, enc, ephCPubBytes)
	if err != nil {
		zeroBytes(exporterHPKE)
		return "", err
	}

	// Optional cookie
	if c.cookies != nil {
		if cookie, ok := c.cookies.GetCookie(ctxID, initDID, peerDID); ok && cookie != "" {
			if msg.Metadata == nil {
				msg.Metadata = map[string]string{}
			}
			msg.Metadata["cookie"] = cookie
		}
	}

	// 6) Send the request and receive a server-signed response.
	resp, err := c.sendAndGetSignedMsg(ctx, msg)
	if err != nil {
		zeroBytes(exporterHPKE)
		return "", err
	}

	// 7) Parse response (incl. server sig)
	r, err := parseServerSignedResponse(resp.Data)
	if err != nil {
		zeroBytes(exporterHPKE)
		return "", err
	}

	// 8) Compute ssE2E
	ssE2E, err := computeE2ESecret(ephCpriv, r.EphSBytes)
	if err != nil {
		zeroBytes(exporterHPKE)
		return "", err
	}

	// 9) Combine exporter + ssE2E
	combined, err := combineSecrets(exporterHPKE, ssE2E, exportCtx)
	zeroBytes(exporterHPKE)
	zeroBytes(ssE2E)
	if err != nil {
		return "", fmt.Errorf("combine: %w", err)
	}

	// 10) Verify ackTag first (HMAC with combined secret)
	binds := [][]byte{
		info, exportCtx, enc, ephCPubBytes, r.EphSBytes, []byte(initDID), []byte(peerDID),
	}
	if err := verifyAckTag(combined, ctxID, nonce, r.Kid, binds, r.AckTag); err != nil {
		zeroBytes(combined)
		return "", err
	}

	// Cross-check that response echoed our enc/ephC (if present)
	if len(r.Enc) > 0 {
		if base64.RawURLEncoding.EncodeToString(enc) != r.EncB64 {
			zeroBytes(combined)
			return "", fmt.Errorf("enc mismatch")
		}
	}
	if len(r.EphC) > 0 {
		if base64.RawURLEncoding.EncodeToString(ephCPubBytes) != r.EphCB64 {
			zeroBytes(combined)
			return "", fmt.Errorf("ephC mismatch")
		}
	}

	// 11) Verify server Ed25519 signature over canonical envelope
	if err := c.verifySignature(ctx, peerDID, r, ctxID, info, exportCtx, enc, ephCPubBytes); err != nil {
		zeroBytes(combined)
		return "", err
	}

	// 12) Create session and bind kid
	if err := c.createAndBindSession(combined, r.Kid); err != nil {
		zeroBytes(combined)
		return "", err
	}
	zeroBytes(combined)
	return r.Kid, nil
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

// Build a transport message for HPKE init and sign it using DID key.
func (c *Client) buildAndSignInitMsg(ctxID, initDID, peerDID string, info, exportCtx []byte, nonce string, enc, ephCPubBytes []byte) (*transport.SecureMessage, error) {
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

	payload, err := json.Marshal(pl)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	signature, err := c.key.Sign(payload)
	if err != nil {
		return nil, fmt.Errorf("sign: %w", err)
	}

	msg := &transport.SecureMessage{
		ID:        uuid.NewString(),
		ContextID: ctxID,
		TaskID:    TaskHPKEComplete,
		Payload:   payload,
		DID:       c.DID,
		Signature: signature,
		Role:      "user",
		Metadata:  make(map[string]string),
	}

	return msg, nil
}

// Send and receive a server-signed message response.
func (c *Client) sendAndGetSignedMsg(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
	resp, err := c.transport.Send(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("transport send: %w", err)
	}
	if resp == nil || len(resp.Data) == 0 {
		return nil, fmt.Errorf("empty response data")
	}
	return resp, nil
}

type serverSignedResponse struct {
	V             string
	Task          string
	Ctx           string
	Kid           string
	EphSBytes     []byte
	AckTag        []byte
	Ts            string
	Did           string
	InfoHash      []byte
	ExportCtxHash []byte
	Enc           []byte
	EphC          []byte
	EncB64        string
	EphCB64       string
	Sig           []byte
}

func parseServerSignedResponse(data []byte) (*serverSignedResponse, error) {
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("unmarshal resp: %w", err)
	}
	get := func(k string) (string, error) {
		v, ok := m[k]
		if !ok || v == "" {
			return "", fmt.Errorf("missing %s", k)
		}
		return v, nil
	}

	v, _ := get("v")            // version
	task, _ := get("task")
	ctx, _ := get("ctx")
	kid, _ := get("kid")
	ephSB64, _ := get("ephS")
	ackB64, _ := get("ackTagB64")
	ts, _ := get("ts")
	did, _ := get("did")
	infoHB64, _ := get("infoHash")
	exportHB64, _ := get("exportCtxHash")
	sigB64, _ := get("sigB64")

	ephS, err := base64.RawURLEncoding.DecodeString(ephSB64)
	if err != nil || len(ephS) != 32 {
		return nil, fmt.Errorf("bad ephS")
	}
	ack, err := base64.RawURLEncoding.DecodeString(ackB64)
	if err != nil {
		return nil, fmt.Errorf("bad ackTagB64")
	}
	ih, err := base64.RawURLEncoding.DecodeString(infoHB64)
	if err != nil || len(ih) != 32 {
		return nil, fmt.Errorf("bad infoHash")
	}
	eh, err := base64.RawURLEncoding.DecodeString(exportHB64)
	if err != nil || len(eh) != 32 {
		return nil, fmt.Errorf("bad exportCtxHash")
	}
	sig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return nil, fmt.Errorf("bad sigB64")
	}

	var enc, ephC []byte
	encB64, ok := m["enc"]
	if ok && encB64 != "" {
		enc, err = base64.RawURLEncoding.DecodeString(encB64)
		if err != nil {
			return nil, fmt.Errorf("bad enc")
		}
	}
	ephCB64, ok := m["ephC"]
	if ok && ephCB64 != "" {
		ephC, err = base64.RawURLEncoding.DecodeString(ephCB64)
		if err != nil {
			return nil, fmt.Errorf("bad ephC")
		}
	}

	return &serverSignedResponse{
		V:             v,
		Task:          task,
		Ctx:           ctx,
		Kid:           kid,
		EphSBytes:     ephS,
		AckTag:        ack,
		Ts:            ts,
		Did:           did,
		InfoHash:      ih,
		ExportCtxHash: eh,
		Enc:           enc,
		EphC:          ephC,
		EncB64:        encB64,
		EphCB64:       ephCB64,
		Sig:           sig,
	}, nil
}

func (c *Client) verifySignature(ctx context.Context, serverDID string, r *serverSignedResponse, ctxID string, info, exportCtx, enc, ephC []byte) error {
	if r.V != "v1" || r.Task != TaskHPKEComplete {
		return fmt.Errorf("unsupported version/task: %s/%s", r.V, r.Task)
	}
	// Resolve server signing key (Ed25519)
	if c.resolver == nil {
		return fmt.Errorf("nil resolver")
	}
	pub, err := c.resolver.ResolvePublicKey(ctx, did.AgentDID(serverDID))
	if err != nil || pub == nil {
		return fmt.Errorf("cannot resolve server pubkey")
	}

	if pinned, ok := c.pins[serverDID]; ok {
        pk, ok := pub.(ed25519.PublicKey)
        if !ok || subtle.ConstantTimeCompare(pinned, pk) != 1 {
            return fmt.Errorf("pin mismatch: server signing key changed")
        }
    }

	// Recompute info/export hashes and check equality
	ih := sha256.Sum256(info)
	eh := sha256.Sum256(exportCtx)
	if !hmac.Equal(ih[:], r.InfoHash) || !hmac.Equal(eh[:], r.ExportCtxHash) {
		return fmt.Errorf("info/exportCtx hash mismatch")
	}

	// Rebuild the exact envelope bytes (must match server side)
	env := serverSigEnvelope{
		V:             r.V,
		Task:          r.Task,
		Ctx:           ctxID, // prefer local ctxID
		Kid:           r.Kid,
		EphS:          base64.RawURLEncoding.EncodeToString(r.EphSBytes),
		AckTagB64:     base64.RawURLEncoding.EncodeToString(r.AckTag),
		Ts:            r.Ts,
		Did:           serverDID,
		InfoHash:      base64.RawURLEncoding.EncodeToString(ih[:]),
		ExportCtxHash: base64.RawURLEncoding.EncodeToString(eh[:]),
		Enc:           base64.RawURLEncoding.EncodeToString(enc),
		EphC:          base64.RawURLEncoding.EncodeToString(ephC),
	}
	envBytes, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal env (client): %w", err)
	}

	// Verify detached signature
	if err := verifySignature(envBytes, r.Sig, pub); err != nil {
		return fmt.Errorf("server signature verify failed: %w", err)
	}
	return nil
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
	// RFC 7748
	if isAllZero32(ssE2E) { return nil, fmt.Errorf("invalid ECDH (all-zero)") }
	return ssE2E, nil
}

// Constant-time verification of the server's ack tag.
func verifyAckTag(seed []byte, ctxID, nonce, kid string, binds [][]byte, tag []byte) error {
	expect := MakeAckTag(seed, ctxID, nonce, kid, binds...)
	if !hmac.Equal(expect, tag) {
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
