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
	handler MessageHandler
}

// NewHTTPServer creates a new HTTP server that processes SecureMessages.
//
// Parameters:
//   - handler: The application-level message handler
func NewHTTPServer(handler MessageHandler) *HTTPServer {
	return &HTTPServer{
		handler: handler,
	}
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

		// Read request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			s.sendErrorResponse(w, "", "", fmt.Errorf("failed to read request body: %w", err))
			return
		}
		defer r.Body.Close()

		// Parse wire message
		var wireMsg wireMessage
		if err := json.Unmarshal(body, &wireMsg); err != nil {
			s.sendErrorResponse(w, "", "", fmt.Errorf("invalid JSON: %w", err))
			return
		}

		// Convert to SecureMessage
		secureMsg := fromWireMessage(&wireMsg, r.Header)

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

// fromWireMessage converts HTTP wire format to transport.SecureMessage
func fromWireMessage(wire *wireMessage, headers http.Header) *transport.SecureMessage {
	msg := &transport.SecureMessage{
		ID:        wire.ID,
		ContextID: wire.ContextID,
		TaskID:    wire.TaskID,
		Payload:   wire.Payload,
		DID:       wire.DID,
		Signature: wire.Signature,
		Metadata:  wire.Metadata,
		Role:      wire.Role,
	}

	// Extract metadata from headers if not in body
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]string)
	}

	// Override with headers if present
	if did := headers.Get("X-SAGE-DID"); did != "" {
		msg.DID = did
	}
	if id := headers.Get("X-SAGE-Message-ID"); id != "" {
		msg.ID = id
	}
	if ctxID := headers.Get("X-SAGE-Context-ID"); ctxID != "" {
		msg.ContextID = ctxID
	}
	if taskID := headers.Get("X-SAGE-Task-ID"); taskID != "" {
		msg.TaskID = taskID
	}

	// Extract custom metadata from X-SAGE-Meta- headers
	for key := range headers {
		if len(key) > 12 && key[:12] == "X-Sage-Meta-" {
			metaKey := key[12:]
			msg.Metadata[metaKey] = headers.Get(key)
		}
	}

	return msg
}

// toWireResponse converts transport.Response to HTTP wire format
func toWireResponse(resp *transport.Response) *wireResponse {
	wire := &wireResponse{
		Success:   resp.Success,
		MessageID: resp.MessageID,
		TaskID:    resp.TaskID,
		Data:      resp.Data,
	}

	if resp.Error != nil {
		wire.Error = resp.Error.Error()
		wire.Success = false
	}

	return wire
}

// sendSuccessResponse sends a successful response
func (s *HTTPServer) sendSuccessResponse(w http.ResponseWriter, resp *transport.Response) {
	wire := toWireResponse(resp)
	s.sendJSONResponse(w, http.StatusOK, wire)
}

// sendErrorResponse sends an error response
func (s *HTTPServer) sendErrorResponse(w http.ResponseWriter, msgID, taskID string, err error) {
	wire := &wireResponse{
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
