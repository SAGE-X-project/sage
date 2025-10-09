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

package sessioninit

import (
	"context"
	"crypto"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"

	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/formats"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/pkg/agent/handshake"
	"github.com/sage-x-project/sage/pkg/agent/session"
)

// Creator implements handshake.Events and forwards to Manager.
type Creator struct {
	sessionMgr *session.Manager

	mu           sync.RWMutex
	ephPrivByCtx map[string]*keys.X25519KeyPair
	sidByCtx     map[string]string
	exporter     sagecrypto.KeyExporter
}

// New creates a handshake integration Creator.
func NewCreator(sm *session.Manager) *Creator {
	return &Creator{
		sessionMgr:   sm,
		ephPrivByCtx: make(map[string]*keys.X25519KeyPair),
		sidByCtx:     make(map[string]string),
		exporter:     formats.NewJWKExporter(),
	}
}

// --- handshake.Events implementation ---

func (a *Creator) OnInvitation(ctx context.Context, ctxID string, inv handshake.InvitationMessage) error {
	// no-op or logging/metrics
	return nil
}

func (a *Creator) OnRequest(ctx context.Context, ctxID string, req handshake.RequestMessage, senderPub crypto.PublicKey) error {
	// no-op or audit
	return nil
}

func (a *Creator) OnResponse(ctx context.Context, ctxID string, res handshake.ResponseMessage, senderPub crypto.PublicKey) error {
	// no-op or metrics
	return nil
}

func (a *Creator) OnComplete(ctx context.Context, ctxID string, comp handshake.CompleteMessage, p session.Params) error {
	a.mu.RLock()
	my := a.ephPrivByCtx[ctxID]
	a.mu.RUnlock()
	if my == nil {
		return fmt.Errorf("no ephemeral private for ctx=%s", ctxID)
	}

	shared, err := my.DeriveSharedSecret(p.PeerEph)
	if err != nil {
		return fmt.Errorf("derive shared: %w", err)
	}
	p.SharedSecret = shared
	// Deterministically create or get session (same ID/keys on both peers).
	_, sid, _, err := a.sessionMgr.EnsureSessionWithParams(p, nil)
	if err != nil {
		return fmt.Errorf("ensure session: %w", err)
	}

	// Cleanup ephemeral.
	a.mu.Lock()
	delete(a.ephPrivByCtx, ctxID)
	a.sidByCtx[ctxID] = sid
	a.mu.Unlock()

	_ = sid
	return nil
}

func (a *Creator) AskEphemeral(ctx context.Context, ctxID string) ([]byte, json.RawMessage, error) {
	kp, err := keys.GenerateX25519KeyPair()
	if err != nil {
		return nil, nil, fmt.Errorf("gen x25519: %w", err)
	}
	x := kp.(*keys.X25519KeyPair)

	a.mu.Lock()
	a.ephPrivByCtx[ctxID] = x
	a.mu.Unlock()

	raw := x.PublicBytesKey()

	jwkBytes, err := a.exporter.ExportPublic(kp, sagecrypto.KeyFormatJWK)
	if err != nil {
		return nil, nil, fmt.Errorf("export jwk: %w", err)
	}
	return raw, json.RawMessage(jwkBytes), nil
}

// IssueKeyID generates a new opaque key ID for the given context ID,
// binds it to the existing session, and returns it to be sent to the peer.
func (a *Creator) IssueKeyID(ctxID string) (string, bool) {
	a.mu.Lock()
	sid, ok := a.sidByCtx[ctxID]
	if ok {
		delete(a.sidByCtx, ctxID)
	}
	a.mu.Unlock()
	if !ok {
		return "", false
	}

	keyid := "session:" + randBase64URL(12)
	a.sessionMgr.BindKeyID(keyid, sid)
	return keyid, true
}

func randBase64URL(length int) string {
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		// If the system's CSPRNG fails, it's a critical error
		panic(fmt.Errorf("crypto/rand read failed: %w", err))
	}
	// Encode to base64 URL without padding
	return base64.RawURLEncoding.EncodeToString(buf)
}
