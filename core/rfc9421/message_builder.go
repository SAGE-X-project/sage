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


package rfc9421

import (
	"strings"
	"time"
)

// MessageBuilder helps construct RFC-9421 compliant messages
type MessageBuilder struct {
	message *Message
}

// NewMessageBuilder creates a new message builder
func NewMessageBuilder() *MessageBuilder {
	return &MessageBuilder{
		message: &Message{
			Headers:      make(map[string]string),
			Metadata:     make(map[string]interface{}),
			SignedFields: []string{},
			Timestamp:    time.Now(),
		},
	}
}

// WithAgentDID sets the agent DID
func (b *MessageBuilder) WithAgentDID(did string) *MessageBuilder {
	b.message.AgentDID = did
	return b
}

// WithMessageID sets the message ID
func (b *MessageBuilder) WithMessageID(id string) *MessageBuilder {
	b.message.MessageID = id
	return b
}

// WithTimestamp sets the timestamp
func (b *MessageBuilder) WithTimestamp(t time.Time) *MessageBuilder {
	b.message.Timestamp = t
	return b
}

// WithNonce sets the nonce
func (b *MessageBuilder) WithNonce(nonce string) *MessageBuilder {
	b.message.Nonce = nonce
	return b
}

// WithBody sets the message body
func (b *MessageBuilder) WithBody(body []byte) *MessageBuilder {
	b.message.Body = body
	return b
}

// AddHeader adds a header
func (b *MessageBuilder) AddHeader(key, value string) *MessageBuilder {
	b.message.Headers[key] = value
	return b
}

// AddMetadata adds metadata
func (b *MessageBuilder) AddMetadata(key string, value interface{}) *MessageBuilder {
	b.message.Metadata[key] = value
	return b
}

// WithAlgorithm sets the signature algorithm
func (b *MessageBuilder) WithAlgorithm(alg SignatureAlgorithm) *MessageBuilder {
	b.message.Algorithm = string(alg)
	return b
}

// WithKeyID sets the key ID
func (b *MessageBuilder) WithKeyID(keyID string) *MessageBuilder {
	b.message.KeyID = keyID
	return b
}

// WithSignedFields sets which fields should be signed
func (b *MessageBuilder) WithSignedFields(fields ...string) *MessageBuilder {
	b.message.SignedFields = fields
	return b
}

// WithSignature sets the signature
func (b *MessageBuilder) WithSignature(signature []byte) *MessageBuilder {
	b.message.Signature = signature
	return b
}

// Build returns the constructed message
func (b *MessageBuilder) Build() *Message {
	// If no signed fields specified, use default set
	if len(b.message.SignedFields) == 0 {
		b.message.SignedFields = []string{"agent_did", "message_id", "timestamp", "nonce", "body"}
	}
	
	return b.message
}

// ParseMessageFromHeaders creates a Message from HTTP-style headers
func ParseMessageFromHeaders(headers map[string]string, body []byte) (*Message, error) {
	builder := NewMessageBuilder()
	
	// Extract standard headers
	if did, ok := headers["X-Agent-DID"]; ok {
		builder.WithAgentDID(did)
	}
	
	if messageID, ok := headers["X-Message-ID"]; ok {
		builder.WithMessageID(messageID)
	}
	
	if timestamp, ok := headers["X-Timestamp"]; ok {
		if ts, err := time.Parse(time.RFC3339, timestamp); err == nil {
			builder.WithTimestamp(ts)
		}
	}
	
	if nonce, ok := headers["X-Nonce"]; ok {
		builder.WithNonce(nonce)
	}
	
	if algorithm, ok := headers["X-Signature-Algorithm"]; ok {
		builder.WithAlgorithm(SignatureAlgorithm(algorithm))
	}
	
	if keyID, ok := headers["X-Key-ID"]; ok {
		builder.WithKeyID(keyID)
	}
	
	if signedFields, ok := headers["X-Signed-Fields"]; ok {
		fields := strings.Split(signedFields, ",")
		for i := range fields {
			fields[i] = strings.TrimSpace(fields[i])
		}
		builder.WithSignedFields(fields...)
	}
	
	// Add all headers
	for k, v := range headers {
		builder.AddHeader(k, v)
	}
	
	// Set body
	builder.WithBody(body)
	
	return builder.Build(), nil
}