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

package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/sage-x-project/sage/pkg/agent/transport"
)

// MessageHandler is a function that processes incoming SecureMessages.
//
// This is the application-level handler that processes decrypted messages
// and returns responses. The HTTP server adapter calls this handler for
// each received message.
type MessageHandler func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error)

// HTTPServer provides HTTP server functionality for receiving SecureMessages.
//
// This server exposes a REST endpoint that accepts SecureMessage payloads
// and delegates processing to a MessageHandler.
//
// Example usage:
//
//	// Create message handler
//	handler := func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
//	    // Process message with handshake server
//	    return handshakeServer.HandleMessage(ctx, msg)
//	}
//
//	// Create HTTP server
//	server := http.NewHTTPServer(handler)
//
//	// Register with HTTP router
//	router.Handle("/messages", server.MessagesHandler())
type HTTPServer struct {
	handler      MessageHandler
	maxBodyBytes int64
}

// DefaultMaxBodyBytes is the request body limit applied by NewHTTPServer.
const DefaultMaxBodyBytes int64 = 1 << 20 // 1 MiB

// NewHTTPServer creates a new HTTP server that processes SecureMessages.
// Request bodies are limited to DefaultMaxBodyBytes; see SetMaxBodyBytes.
//
// Parameters:
//   - handler: The application-level message handler
func NewHTTPServer(handler MessageHandler) *HTTPServer {
	return &HTTPServer{
		handler:      handler,
		maxBodyBytes: DefaultMaxBodyBytes,
	}
}

// SetMaxBodyBytes changes the request body limit. Requests with a larger body
// are rejected with 413 before being parsed. Values <= 0 restore the default.
func (s *HTTPServer) SetMaxBodyBytes(n int64) {
	if n <= 0 {
		n = DefaultMaxBodyBytes
	}
	s.maxBodyBytes = n
}

// MessagesHandler returns an http.Handler for the /messages endpoint.
//
// This handler:
//  1. Receives HTTP POST requests with SecureMessage JSON payloads
//  2. Validates and parses the message
//  3. Calls the MessageHandler to process the message
//  4. Returns the Response as JSON
func (s *HTTPServer) MessagesHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only accept POST
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Read request body, bounded so an oversized request cannot exhaust memory
		r.Body = http.MaxBytesReader(w, r.Body, s.maxBodyBytes)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			s.sendErrorResponse(w, "", "", fmt.Errorf("failed to read request body"))
			return
		}
		defer func() {
			if err := r.Body.Close(); err != nil {
				// Log error but don't fail the request since body was already read
				fmt.Printf("Warning: failed to close request body: %v\n", err)
			}
		}()

		// Parse wire message
		var wireMsg transport.WireMessage
		if err := json.Unmarshal(body, &wireMsg); err != nil {
			s.sendErrorResponse(w, "", "", fmt.Errorf("invalid JSON: %w", err))
			return
		}

		// Convert to SecureMessage
		secureMsg, err := fromWireMessage(&wireMsg, r.Header)
		if err != nil {
			s.sendErrorResponse(w, wireMsg.ID, wireMsg.TaskID, err)
			return
		}

		// Validate required fields
		if secureMsg.ID == "" {
			s.sendErrorResponse(w, "", "", fmt.Errorf("message ID is required"))
			return
		}
		if secureMsg.DID == "" {
			s.sendErrorResponse(w, "", "", fmt.Errorf("DID is required"))
			return
		}
		if len(secureMsg.Payload) == 0 {
			s.sendErrorResponse(w, secureMsg.ID, secureMsg.TaskID, fmt.Errorf("payload is required"))
			return
		}

		// Call application handler
		resp, err := s.handler(r.Context(), secureMsg)
		if err != nil {
			s.sendErrorResponse(w, secureMsg.ID, secureMsg.TaskID, err)
			return
		}

		// Send response
		s.sendSuccessResponse(w, resp)
	})
}

// fromWireMessage converts HTTP wire format to transport.SecureMessage.
//
// The JSON body is authoritative for the message identity (DID, message id,
// context id, task id). The X-SAGE-* identity headers the client also sends
// are accepted only when they agree with the body; a header that contradicts
// the body is rejected rather than allowed to override the signed payload.
func fromWireMessage(wire *transport.WireMessage, headers http.Header) (*transport.SecureMessage, error) {
	msg := transport.FromWireMessage(wire)

	// Identity headers must match the body; they never override it.
	for header, bodyValue := range map[string]string{
		"X-SAGE-DID":        msg.DID,
		"X-SAGE-Message-ID": msg.ID,
		"X-SAGE-Context-ID": msg.ContextID,
		"X-SAGE-Task-ID":    msg.TaskID,
	} {
		if v := headers.Get(header); v != "" && v != bodyValue {
			return nil, fmt.Errorf("%s header does not match the message body", header)
		}
	}

	// Extract custom metadata from X-SAGE-Meta- headers
	for key := range headers {
		if len(key) > 12 && key[:12] == "X-Sage-Meta-" {
			metaKey := key[12:]
			msg.Metadata[metaKey] = headers.Get(key)
		}
	}

	return msg, nil
}

// sendSuccessResponse sends a successful response
func (s *HTTPServer) sendSuccessResponse(w http.ResponseWriter, resp *transport.Response) {
	wire := transport.ToWireResponse(resp)
	s.sendJSONResponse(w, http.StatusOK, wire)
}

// sendErrorResponse sends an error response
func (s *HTTPServer) sendErrorResponse(w http.ResponseWriter, msgID, taskID string, err error) {
	wire := &transport.WireResponse{
		Success:   false,
		MessageID: msgID,
		TaskID:    taskID,
		Error:     err.Error(),
	}
	s.sendJSONResponse(w, http.StatusOK, wire) // Still 200 OK, error in response body
}

// sendJSONResponse sends a JSON response
func (s *HTTPServer) sendJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Log error but can't send response anymore
		fmt.Printf("Failed to encode JSON response: %v\n", err)
	}
}

// ServeHTTP implements http.Handler interface for the server
func (s *HTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.MessagesHandler().ServeHTTP(w, r)
}
