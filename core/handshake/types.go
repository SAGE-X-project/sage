package handshake

import (
	"encoding/json"
	"time"

	"github.com/sage-x-project/sage/core/message"
)

// HandshakeStep represents a phase in the A2A handshake.
const (
    Invitation string = "invitation"
    Request    string = "request"
    Response   string = "response"
    Complete   string = "complete"
)

const contextIDPrefix = "ctx-handshake"
const sessionIDPrefix = "session-handshake"

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
	Session Session 			    `json:"session"`
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
	Session Session 		`json:"session"`
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

// Session contains the parameters required for message exchange before the communication channel is established.
type Session struct {
    ID                string 			`json:"id"`
    EphemeralPubKey   json.RawMessage 	`json:"ephemeralPublicKey"` // JWK format
    KeyInfo           *KeyInfo 	        `json:"signaturePolicy,omitempty"`
    Status            string 			`json:"status,omitempty"`
    CreatedAt         string 			`json:"createdAt,omitempty"`
    ExpiresAt         string 			`json:"expiresAt,omitempty"`
}

// KeyInfo contains the information required for RFC‑9421 signature verification.
type KeyInfo struct {
    KeyID               string   `json:"keyid"`             // (RFC‑9421) identifier of the public key to be used for signing
    Salt                string   `json:"salt"`              // a random value (32 bytes) to uniquely identify the signer, encoded in Base64URL
    SignatureSpec       string   `json:"signatureSpec"`     // the signature format understood by the implementation (e.g., “RFC-9421”)
    FieldsToSign        []string `json:"fieldsToSign"`      // an array of JSONPath expressions indicating which fields to include (e.g., @header, @payload.id)
    TimestampTolerance  string   `json:"timestampTolerance"`// an RFC 3339 duration string specifying the allowed clock skew (e.g., “5s”, “2m”, “1h”)
}