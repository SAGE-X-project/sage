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

package handshake

import (
	"context"
	"crypto"
	"crypto/ecdh"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sage-x-project/sage/internal/metrics"
	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/formats"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/sage-x-project/sage/pkg/agent/session"
	"github.com/sage-x-project/sage/pkg/agent/transport"
	"golang.org/x/sync/singleflight"
)

// pendingState holds only public transcript material with an expiry.
// No shared secret is stored here (Core derives it later at OnComplete).
type pendingState struct {
	peerEph   []byte    // raw client ephemeral public (32B)
	serverEph []byte    // raw server ephemeral public (32B) provided by Core
	expires   time.Time // TTL for cleanup if Complete never arrives
}

// Server handles secure handshake message processing
//   - Does not create/store sessions.
//   - Emits Events so the application layer can manage sessions separately.
//   - Can send Response to the peer via transport if configured.
type Server struct {
	key       sagecrypto.KeyPair
	events    Events
	transport transport.MessageTransport // Optional: for sending responses

	// Resolve the sender pubkey from ctx/message/metadata for signature check or decrypt.
	// Parse JWT and get DID field
	resolver did.Resolver
	mu       sync.Mutex
	sf       singleflight.Group
	// pending holds per-context ephemeral handshake state created at Request phase.
	pending map[string]pendingState

	// sessionCfg defines default session policies for SecureSession instances.
	sessionCfg session.Config

	peers map[string]cachedPeer
	// TTL and cleaner
	pendingTTL    time.Duration
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
	cleanupDone   chan struct{}

	exporter sagecrypto.KeyExporter
	importer sagecrypto.KeyImporter
}

type cachedPeer struct {
	pub     crypto.PublicKey
	did     string
	expires time.Time
}

// NewServer creates a server with required dependencies.
// - events: application-level hooks (can be NoopEvents{})
// - transport: optional transport for sending responses (can be nil)
func NewServer(
	key sagecrypto.KeyPair,
	events Events,
	resolver did.Resolver,
	sessionCfg *session.Config,
	cleanupInterval time.Duration,
	t transport.MessageTransport,
) *Server {
	if events == nil {
		events = NoopEvents{}
	}

	// Apply defaults if sessionCfg is nil
	var cfg session.Config
	if sessionCfg == nil {
		cfg = session.Config{
			MaxAge:      time.Hour,        // default absolute expiration
			IdleTimeout: 10 * time.Minute, // default idle timeout
			MaxMessages: 10000,            // default max message count
		}
	} else {
		cfg = *sessionCfg
	}
	s := &Server{
		key:         key,
		events:      events,
		resolver:    resolver,
		transport:   t,
		pending:     make(map[string]pendingState),
		peers:       make(map[string]cachedPeer),
		sessionCfg:  cfg,
		exporter:    formats.NewJWKExporter(),
		importer:    formats.NewJWKImporter(),
		stopCleanup: make(chan struct{}),
		cleanupDone: make(chan struct{}),
	}

	if s.pendingTTL == 0 {
		s.pendingTTL = 15 * time.Minute
	}
	interval := cleanupInterval
	if interval <= 0 {
		interval = 10 * time.Minute
	}
	s.cleanupTicker = time.NewTicker(interval)
	go s.cleanupLoop()
	return s
}

// HandleMessage is the single entry point for all phases.
// It validates input, decodes payload, and triggers event callbacks.
func (s *Server) HandleMessage(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
	start := time.Now()

	if msg == nil {
		return nil, errors.New("empty message")
	}

	phase, err := parsePhase(msg.TaskID)
	if err != nil {
		return nil, err
	}

	// Track handshake initiation and duration
	metrics.HandshakesInitiated.WithLabelValues("server").Inc()
	defer func() {
		metrics.HandshakeDuration.WithLabelValues(phase.String()).Observe(
			time.Since(start).Seconds(),
		)
	}()

	switch phase {

	case Invitation:
		if msg.DID == "" {
			return nil, errors.New("missing did in invitation")
		}
		senderDID := msg.DID

		if s.resolver == nil {
			return nil, errors.New("cannot resolve sender pubkey: resolver not set")
		}

		// Use singleflight for both cache check and resolve to prevent race conditions
		v, err, _ := s.sf.Do("resolve:"+senderDID+":"+msg.ContextID, func() (any, error) {
			// Check cache inside singleflight to ensure only one goroutine resolves
			if cache, ok := s.getPeer(msg.ContextID); ok && cache.did == senderDID && time.Now().Before(cache.expires) {
				return cache.pub, nil
			}

			// Resolve public key from DID
			pub, err := s.resolver.ResolvePublicKey(ctx, did.AgentDID(senderDID))
			if err != nil {
				return nil, err
			}

			// Cache the resolved public key for future requests
			if pubKey, ok := pub.(crypto.PublicKey); ok {
				s.savePeer(msg.ContextID, pubKey, senderDID)
			}

			return pub, nil
		})

		if err != nil || v == nil {
			return nil, errors.New("cannot resolve sender pubkey")
		}

		var senderPub crypto.PublicKey
		var okType bool
		senderPub, okType = v.(crypto.PublicKey)
		if !okType {
			return nil, fmt.Errorf("resolver returned unexpected key type: %T", v)
		}

		// Verify sender signature
		if err := s.verifySignature(msg.Payload, msg.Signature, senderPub); err != nil {
			metrics.HandshakesFailed.WithLabelValues("signature_error").Inc()
			return nil, fmt.Errorf("signature verification failed: %w", err)
		}

		var inv InvitationMessage
		if err := json.Unmarshal(msg.Payload, &inv); err != nil {
			metrics.HandshakesFailed.WithLabelValues("decode_error").Inc()
			return nil, fmt.Errorf("invitation decode: %w", err)
		}
		_ = s.events.OnInvitation(ctx, msg.ContextID, inv)
		metrics.HandshakesCompleted.WithLabelValues("success").Inc()
		return s.ackResponse(msg, "invitation_received")

	case Request:
		cache, ok := s.getPeer(msg.ContextID)
		if !ok {
			metrics.HandshakesFailed.WithLabelValues("missing_context").Inc()
			return nil, errors.New("no cached peer for context; invitation required first")
		}

		if err := s.verifySignature(msg.Payload, msg.Signature, cache.pub); err != nil {
			metrics.HandshakesFailed.WithLabelValues("signature_error").Inc()
			return nil, fmt.Errorf("request signature verification failed: %w", err)
		}

		plain, err := keys.DecryptWithEd25519Peer(s.key.PrivateKey(), msg.Payload)
		if err != nil {
			metrics.HandshakesFailed.WithLabelValues("decrypt_error").Inc()
			return nil, fmt.Errorf("request decrypt: %w", err)
		}

		var req RequestMessage
		if err := json.Unmarshal(plain, &req); err != nil {
			metrics.HandshakesFailed.WithLabelValues("decode_error").Inc()
			return nil, fmt.Errorf("request json: %w", err)
		}

		if len(req.EphemeralPubKey) == 0 {
			metrics.HandshakesFailed.WithLabelValues("invalid_key").Inc()
			return nil, fmt.Errorf("empty peer ephemeral public key")
		}

		exported, err := s.importer.ImportPublic([]byte(req.EphemeralPubKey), sagecrypto.KeyFormatJWK)
		if err != nil {
			metrics.HandshakesFailed.WithLabelValues("invalid_key").Inc()
			return nil, fmt.Errorf("import peer ephemeral key: %w", err)
		}
		peerPub, ok := exported.(*ecdh.PublicKey)
		if !ok {
			metrics.HandshakesFailed.WithLabelValues("invalid_key").Inc()
			return nil, fmt.Errorf("unexpected peer eph key type: %T", peerPub)
		}
		peerEphRaw := peerPub.Bytes()
		if len(peerEphRaw) != 32 {
			metrics.HandshakesFailed.WithLabelValues("invalid_key").Inc()
			return nil, fmt.Errorf("invalid peer eph length: %d", len(peerEphRaw))
		}

		serverEphRaw, serverEphJWK, err := s.events.AskEphemeral(ctx, msg.ContextID)
		if err != nil {
			metrics.HandshakesFailed.WithLabelValues("ephemeral_error").Inc()
			return nil, fmt.Errorf("ask ephemeral: %w", err)
		}
		if len(serverEphRaw) != 32 {
			metrics.HandshakesFailed.WithLabelValues("ephemeral_error").Inc()
			return nil, fmt.Errorf("invalid server eph length: %d", len(serverEphRaw))
		}

		s.savePending(msg.ContextID, pendingState{
			peerEph:   append([]byte(nil), peerEphRaw...),
			serverEph: append([]byte(nil), serverEphRaw...),
		})
		_ = s.events.OnRequest(ctx, msg.ContextID, req, cache.pub)

		// Optionally respond immediately to the peer.
		res := ResponseMessage{
			EphemeralPubKey: json.RawMessage(serverEphJWK),
			Ack:             true,
		}

		metrics.HandshakesCompleted.WithLabelValues("success").Inc()
		return s.sendResponseToPeer(ctx, res, msg.ContextID, cache.pub, cache.did)

	case Complete:
		cache, ok := s.getPeer(msg.ContextID)
		if !ok {
			metrics.HandshakesFailed.WithLabelValues("missing_context").Inc()
			return nil, errors.New("no cached peer for context; invitation required first")
		}

		if err := s.verifySignature(msg.Payload, msg.Signature, cache.pub); err != nil {
			metrics.HandshakesFailed.WithLabelValues("signature_error").Inc()
			return nil, fmt.Errorf("complete signature verification failed: %w", err)
		}

		var comp CompleteMessage
		_ = json.Unmarshal(msg.Payload, &comp) // best-effort

		st, ok := s.takePending(msg.ContextID)
		if !ok {
			_ = s.events.OnComplete(ctx, msg.ContextID, comp, session.Params{})
			metrics.HandshakesCompleted.WithLabelValues("success").Inc()
			return s.ackResponse(msg, "complete_received_no_pending")
		}

		sessParams := session.Params{
			ContextID: msg.ContextID,
			SelfEph:   st.serverEph,
			PeerEph:   st.peerEph,
			Label:     "a2a/handshake v1",
		}

		_ = s.events.OnComplete(ctx, msg.ContextID, comp, sessParams)

		if binder, ok := any(s.events).(KeyIDBinder); ok && cache.pub != nil {
			if kid, ok2 := binder.IssueKeyID(msg.ContextID); ok2 && kid != "" {
				res := ResponseMessage{
					Ack:   true,
					KeyID: kid,
				}
				metrics.HandshakesCompleted.WithLabelValues("success").Inc()
				return s.sendResponseToPeer(ctx, res, msg.ContextID, cache.pub, cache.did)
			}
		}
		metrics.HandshakesCompleted.WithLabelValues("success").Inc()
		return s.ackResponse(msg, "complete_received_session_ready")

	default:
		return nil, errors.New("unknown phase")
	}
}

// sendResponseToPeer builds and sends a Response to the peer using transport.
// It encrypts the Response with the peer's public key (bootstrap envelope).
func (s *Server) sendResponseToPeer(ctx context.Context, res ResponseMessage, ctxID string, peerPub crypto.PublicKey, senderDID string) (*transport.Response, error) {
	if s.transport == nil {
		// No transport configured, return success without sending
		return &transport.Response{
			Success:   true,
			MessageID: uuid.NewString(),
			TaskID:    GenerateTaskID(Response),
		}, nil
	}

	plain, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("marshal response: %w", err)
	}

	packet, err := keys.EncryptWithEd25519Peer(peerPub, plain)
	if err != nil {
		return nil, fmt.Errorf("encrypt response: %w", err)
	}

	signature, err := s.key.Sign(packet)
	if err != nil {
		return nil, fmt.Errorf("sign response: %w", err)
	}

	msg := &transport.SecureMessage{
		ID:        uuid.NewString(),
		ContextID: ctxID,
		TaskID:    GenerateTaskID(Response),
		Payload:   packet,
		DID:       senderDID,
		Signature: signature,
		Role:      "agent",
		Metadata:  make(map[string]string),
	}

	return s.transport.Send(ctx, msg)
}

// verifySignature checks the signature against the payload.
func (s *Server) verifySignature(payload, signature []byte, senderPub crypto.PublicKey) error {
	if len(signature) == 0 {
		return errors.New("missing signature")
	}

	// Support either a custom Verify interface or raw ed25519.PublicKey
	type verifyKey interface {
		Verify(msg, sig []byte) error
	}

	switch pk := senderPub.(type) {
	case verifyKey:
		// Custom key type implements Verify([]byte, []byte) error
		if err := pk.Verify(payload, signature); err != nil {
			return fmt.Errorf("signature verify failed: %w", err)
		}
		return nil
	case ed25519.PublicKey:
		// Standard ed25519
		if !ed25519.Verify(pk, payload, signature) {
			return errors.New("signature verify failed: invalid ed25519 signature")
		}
		return nil
	default:
		return fmt.Errorf("unsupported public key type: %T", senderPub)
	}
}

// ackResponse builds a transport.Response with acknowledgment information.
func (s *Server) ackResponse(msg *transport.SecureMessage, note string) (*transport.Response, error) {
	ackData := map[string]any{
		"note": note,
		"ts":   time.Now().UTC().Format(time.RFC3339Nano),
		"ctx":  msg.ContextID,
		"task": msg.TaskID,
	}
	data, _ := json.Marshal(ackData)

	return &transport.Response{
		Success:   true,
		MessageID: msg.ID,
		TaskID:    msg.TaskID,
		Data:      data,
	}, nil
}

// peer cache helpers
func (s *Server) savePeer(ctxID string, pub crypto.PublicKey, did string) {
	s.mu.Lock()
	s.peers[ctxID] = cachedPeer{
		pub:     pub,
		did:     did,
		expires: time.Now().Add(s.pendingTTL),
	}
	s.mu.Unlock()
}

func (s *Server) getPeer(ctxID string) (cachedPeer, bool) {
	s.mu.Lock()
	cp, ok := s.peers[ctxID]
	s.mu.Unlock()
	return cp, ok
}

func (s *Server) cleanupLoop() {
	ticker := s.cleanupTicker
	for {
		select {
		case <-ticker.C:
			s.cleanupExpired(time.Now())
		case <-s.stopCleanup:
			ticker.Stop()
			s.mu.Lock()
			if s.cleanupDone != nil {
				close(s.cleanupDone)
			}
			s.mu.Unlock()
			return
		}
	}
}

func (s *Server) cleanupExpired(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for ctxID, st := range s.pending {
		if now.After(st.expires) {
			delete(s.pending, ctxID)
		}
	}
	for ctxID, cp := range s.peers {
		if now.After(cp.expires) {
			delete(s.peers, ctxID)
		}
	}
}

// save/take helpers
func (s *Server) savePending(id string, st pendingState) {
	s.mu.Lock()
	s.pending[id] = st
	s.mu.Unlock()
}
func (s *Server) takePending(id string) (pendingState, bool) {
	s.mu.Lock()
	st, ok := s.pending[id]
	if ok {
		delete(s.pending, id) // delete on normal Complete path
	}
	s.mu.Unlock()
	return st, ok
}
