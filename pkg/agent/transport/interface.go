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

// Package transport provides transport layer abstraction for SAGE.
//
// This package defines interfaces for message transport protocols,
// allowing SAGE security layer to remain independent of specific
// transport implementations (gRPC, HTTP, WebSocket, etc.).
package transport

import "context"

// MessageTransport is the transport layer abstraction interface.
//
// This interface allows SAGE to send secure messages without depending
// on specific transport protocols. Implementations can be provided for
// gRPC (A2A), HTTP, WebSocket, or any custom transport.
//
// Example usage:
//
//	// Create a transport (gRPC, HTTP, etc.)
//	transport := grpc.NewA2ATransport(conn)
//
//	// Use in handshake client
//	client := handshake.NewClient(transport, keyPair)
//	resp, err := client.Invitation(ctx, invMsg, did)
type MessageTransport interface {
	// Send transmits a secure message and returns the response.
	//
	// The msg parameter contains the encrypted payload, signature,
	// and metadata prepared by the SAGE security layer. The transport
	// implementation is responsible for converting this to the
	// appropriate wire format and handling network transmission.
	//
	// Returns:
	//   - Response: The transport response containing status and data
	//   - error: Network or protocol errors
	Send(ctx context.Context, msg *SecureMessage) (*Response, error)
}

// SecureMessage represents a secure message prepared by SAGE.
//
// This message contains all security-related information (encryption,
// signature, DID) but is independent of the transport protocol.
// Transport implementations convert this to their specific format.
type SecureMessage struct {
	// Message identifiers
	ID        string // Unique message ID (UUID)
	ContextID string // Conversation context ID
	TaskID    string // Task identifier for the message

	// Security payload (prepared by SAGE)
	Payload []byte // Encrypted message content

	// DID and authentication
	DID       string // Sender DID (did:sage:ethereum:...)
	Signature []byte // RFC 9421 signature or DID signature

	// Additional metadata
	Metadata map[string]string // Custom headers/metadata

	// Message role
	Role string // "user" or "agent"
}

// Response represents the transport layer response.
//
// This contains the result of sending a message through the transport,
// independent of the specific protocol used.
type Response struct {
	// Status
	Success bool // Whether the message was successfully delivered

	// Message tracking
	MessageID string // Echo of the sent message ID
	TaskID    string // Echo of the task ID

	// Response data
	Data []byte // Response payload (if any)

	// Error information
	Error error // Transport or protocol error
}
