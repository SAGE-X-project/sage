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
	"encoding/json"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/core/message"
	"github.com/sage-x-project/sage/pkg/agent/session"
)

type Phase int

const (
	Invitation Phase = iota + 1
	Request
	Response
	Complete
)

// String implements the Stringer interface for Phase
func (p Phase) String() string {
	switch p {
	case Invitation:
		return "invitation"
	case Request:
		return "request"
	case Response:
		return "response"
	case Complete:
		return "complete"
	default:
		return "unknown"
	}
}

// Events for agent (session) layer
// Events defines callbacks for the agent/application layer.
// The handshake package does not create or store sessions; it only emits events.
type Events interface {
	// OnInvitation is called when an Invitation is received.
	OnInvitation(ctx context.Context, ctxID string, inv InvitationMessage) error
	// OnRequest is called after the Request is decrypted and parsed.
	// The agent can derive a shared secret and create/store a session outside.
	OnRequest(ctx context.Context, ctxID string, req RequestMessage, senderPub crypto.PublicKey) error
	// OnResponse is called after a Response is received and parsed (if clear/unencrypted).
	// If your protocol encrypts Response, handle decryption at the agent layer.
	OnResponse(ctx context.Context, ctxID string, res ResponseMessage, senderPub crypto.PublicKey) error
	// OnComplete is called when Complete is received.
	// If a SecureSession has been established during the handshake, it will be provided.
	OnComplete(ctx context.Context, ctxID string, comp CompleteMessage, sessParams session.Params) error

	// AskEphemeral asks the app-layer to mint an X25519 ephemeral keypair for this ctxID.
	// The implementation MUST keep the private key internally and return:
	//  - rawPub: raw 32-byte X25519 public key (for transcript and HKDF binding),
	//  - jwkPub: a JWK-encoded public key to send back to the peer.
	AskEphemeral(ctx context.Context, ctxID string) (rawPub []byte, jwkPub json.RawMessage, err error)
}

// NoopEvents is a default no-op implementation.
type NoopEvents struct{}

func (NoopEvents) OnInvitation(context.Context, string, InvitationMessage) error { return nil }
func (NoopEvents) OnRequest(context.Context, string, RequestMessage, crypto.PublicKey) error {
	return nil
}
func (NoopEvents) OnResponse(context.Context, string, ResponseMessage, crypto.PublicKey) error {
	return nil
}
func (NoopEvents) OnComplete(context.Context, string, CompleteMessage, session.Params) error {
	return nil
}
func (NoopEvents) AskEphemeral(context.Context, string) ([]byte, json.RawMessage, error) {
	return nil, nil, nil
}

// 1) Optional extension: let the application issue and bind a keyid after session is ensured.
// If implemented, the server will embed the issued keyid into the Complete ACK response.
type KeyIDBinder interface {
	// IssueKeyID returns an opaque keyid bound to the negotiated session for the given context.
	// ok=false means no keyid is available (server will omit it).
	IssueKeyID(ctxID string) (keyid string, ok bool)
}

// InvitationMessage represents an invitation packet containing only the Session ID.
// It is delivered alongside a JWT carrying the agent's DID information.
type InvitationMessage struct {
	message.BaseMessage
	message.MessageControlHeader
}

func (m *InvitationMessage) GetSequence() uint64 {
	return m.Sequence
}

func (m *InvitationMessage) GetNonce() string {
	return m.Nonce
}

func (m *InvitationMessage) GetTimestamp() time.Time {
	return m.Timestamp
}

// RequestMessage represents a request packet in the A2A handshake,
// including on-chain signature authentication fields.
type RequestMessage struct {
	message.BaseMessage
	message.MessageControlHeader
	EphemeralPubKey json.RawMessage `json:"ephemeralPublicKey"` // JWK format
}

func (m *RequestMessage) GetSequence() uint64 {
	return m.Sequence
}

func (m *RequestMessage) GetNonce() string {
	return m.Nonce
}

func (m *RequestMessage) GetTimestamp() time.Time {
	return m.Timestamp
}

// ResponseMessage defines the payload sent by the server in reply to a client's Request.
// It confirms the agreed session parameters and attaches the server's signature.
type ResponseMessage struct {
	message.BaseMessage
	message.MessageControlHeader
	EphemeralPubKey json.RawMessage `json:"ephemeralPublicKey"` // JWK format
	KeyID           string          `json:"keyid,omitempty"`
	Ack             bool            `json:"ack"`
}

func (m *ResponseMessage) GetSequence() uint64 {
	return m.Sequence
}

func (m *ResponseMessage) GetNonce() string {
	return m.Nonce
}

func (m *ResponseMessage) GetTimestamp() time.Time {
	return m.Timestamp
}

// CompleteMessage signals the end of the handshake with only the session identifier.
type CompleteMessage struct {
	message.BaseMessage
	message.MessageControlHeader
}

func (m *CompleteMessage) GetSequence() uint64 {
	return m.Sequence
}

func (m *CompleteMessage) GetNonce() string {
	return m.Nonce
}

func (m *CompleteMessage) GetTimestamp() time.Time {
	return m.Timestamp
}

// KeyInfo contains the information required for RFC‑9421 signature verification.
type KeyInfo struct {
	KeyID              string   `json:"keyid"`              // (RFC‑9421) identifier of the public key to be used for signing
	Salt               string   `json:"salt"`               // a random value (32 bytes) to uniquely identify the signer, encoded in Base64URL
	SignatureSpec      string   `json:"signatureSpec"`      // the signature format understood by the implementation (e.g., “RFC-9421”)
	FieldsToSign       []string `json:"fieldsToSign"`       // an array of JSONPath expressions indicating which fields to include (e.g., @header, @payload.id)
	TimestampTolerance string   `json:"timestampTolerance"` // an RFC 3339 duration string specifying the allowed clock skew (e.g., “5s”, “2m”, “1h”)
}
