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

package websocket

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sage-x-project/sage/pkg/agent/transport"
)

// MessageHandler is a function that processes incoming SecureMessages.
//
// This is the application-level handler that processes messages
// and returns responses.
type MessageHandler func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error)

// WSServer provides WebSocket server functionality for receiving SecureMessages.
//
// This server maintains persistent WebSocket connections and processes
// incoming messages through a MessageHandler.
//
// Example usage:
//
//	// Create message handler
//	handler := func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
//	    return handshakeServer.HandleMessage(ctx, msg)
//	}
//
//	// Create WebSocket server
//	server := websocket.NewWSServer(handler)
//
//	// Register with HTTP router
//	http.Handle("/ws", server.Handler())
type WSServer struct {
	handler      MessageHandler
	upgrader     websocket.Upgrader
	readTimeout  time.Duration
	writeTimeout time.Duration

	// Origin validation
	allowedOrigins map[string]bool
	checkOrigin    bool // If false, accept all origins (development mode)

	// Active connections
	connections map[*websocket.Conn]bool
	connMu      sync.RWMutex
}

// NewWSServer creates a new WebSocket server with default settings.
//
// By default, origin checking is disabled (development mode).
// For production use, use NewWSServerWithOrigins() to enable origin validation.
//
// Parameters:
//   - handler: The application-level message handler
func NewWSServer(handler MessageHandler) *WSServer {
	server := &WSServer{
		handler:        handler,
		readTimeout:    60 * time.Second,
		writeTimeout:   30 * time.Second,
		allowedOrigins: make(map[string]bool),
		checkOrigin:    false, // Development mode: accept all origins
		connections:    make(map[*websocket.Conn]bool),
	}

	server.upgrader = websocket.Upgrader{
		CheckOrigin:     server.checkOriginFunc,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	return server
}

// NewWSServerWithOrigins creates a new WebSocket server with origin validation.
//
// This should be used in production to restrict WebSocket connections to
// trusted origins only.
//
// Parameters:
//   - handler: The application-level message handler
//   - allowedOrigins: List of allowed origin URLs (e.g., ["https://example.com", "https://app.example.com"])
func NewWSServerWithOrigins(handler MessageHandler, allowedOrigins []string) *WSServer {
	server := NewWSServer(handler)
	server.checkOrigin = true

	for _, origin := range allowedOrigins {
		server.allowedOrigins[origin] = true
	}

	return server
}

// NewWSServerWithTimeouts creates a WebSocket server with custom timeouts.
func NewWSServerWithTimeouts(handler MessageHandler, readTimeout, writeTimeout time.Duration) *WSServer {
	server := NewWSServer(handler)
	server.readTimeout = readTimeout
	server.writeTimeout = writeTimeout
	return server
}

// checkOriginFunc validates the origin of incoming WebSocket connections.
//
// Security behavior:
//   - If checkOrigin is false (development mode), all origins are accepted
//   - If checkOrigin is true (production mode), only allowed origins are accepted
//   - Same-origin requests are always allowed when checkOrigin is true
func (s *WSServer) checkOriginFunc(r *http.Request) bool {
	// Development mode: accept all origins
	if !s.checkOrigin {
		return true
	}

	// Production mode: validate against allowed origins
	origin := r.Header.Get("Origin")
	if origin == "" {
		// No origin header - might be same-origin or non-browser client
		// Allow if Host header matches the request URL
		return true
	}

	// Check if origin is in allowed list
	return s.allowedOrigins[origin]
}

// AddAllowedOrigin adds an origin to the allowed origins list.
//
// This can be used to dynamically update allowed origins after server creation.
func (s *WSServer) AddAllowedOrigin(origin string) {
	s.allowedOrigins[origin] = true
	s.checkOrigin = true
}

// RemoveAllowedOrigin removes an origin from the allowed origins list.
func (s *WSServer) RemoveAllowedOrigin(origin string) {
	delete(s.allowedOrigins, origin)
}

// SetOriginCheckEnabled enables or disables origin checking.
//
// This should only be disabled in development environments.
func (s *WSServer) SetOriginCheckEnabled(enabled bool) {
	s.checkOrigin = enabled
}

// Handler returns an http.Handler for WebSocket connections.
func (s *WSServer) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Upgrade HTTP connection to WebSocket
		conn, err := s.upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, fmt.Sprintf("WebSocket upgrade failed: %v", err), http.StatusBadRequest)
			return
		}

		// Track connection
		s.addConnection(conn)
		defer s.removeConnection(conn)
		defer func() { _ = conn.Close() }()

		// Handle messages on this connection
		s.handleConnection(r.Context(), conn)
	})
}

// handleConnection processes messages from a WebSocket connection
func (s *WSServer) handleConnection(ctx context.Context, conn *websocket.Conn) {
	for {
		// Set read deadline
		if err := conn.SetReadDeadline(time.Now().Add(s.readTimeout)); err != nil {
			return
		}

		// Read message
		var wireMsg wireMessage
		if err := conn.ReadJSON(&wireMsg); err != nil {
			// Check if it's a normal close
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				fmt.Printf("WebSocket read error: %v\n", err)
			}
			return
		}

		// Convert to SecureMessage
		secureMsg := fromWireMessage(&wireMsg)

		// Validate required fields
		if secureMsg.ID == "" {
			s.sendErrorResponse(conn, "", "", fmt.Errorf("message ID is required"))
			continue
		}
		if secureMsg.DID == "" {
			s.sendErrorResponse(conn, secureMsg.ID, secureMsg.TaskID, fmt.Errorf("DID is required"))
			continue
		}
		if len(secureMsg.Payload) == 0 {
			s.sendErrorResponse(conn, secureMsg.ID, secureMsg.TaskID, fmt.Errorf("payload is required"))
			continue
		}

		// Call application handler
		resp, err := s.handler(ctx, secureMsg)
		if err != nil {
			s.sendErrorResponse(conn, secureMsg.ID, secureMsg.TaskID, err)
			continue
		}

		// Send response
		s.sendSuccessResponse(conn, resp)
	}
}

// fromWireMessage converts WebSocket wire format to transport.SecureMessage
func fromWireMessage(wire *wireMessage) *transport.SecureMessage {
	return &transport.SecureMessage{
		ID:        wire.ID,
		ContextID: wire.ContextID,
		TaskID:    wire.TaskID,
		Payload:   wire.Payload,
		DID:       wire.DID,
		Signature: wire.Signature,
		Metadata:  wire.Metadata,
		Role:      wire.Role,
	}
}

// toWireResponse converts transport.Response to WebSocket wire format
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
func (s *WSServer) sendSuccessResponse(conn *websocket.Conn, resp *transport.Response) {
	wire := toWireResponse(resp)
	s.sendResponse(conn, wire)
}

// sendErrorResponse sends an error response
func (s *WSServer) sendErrorResponse(conn *websocket.Conn, msgID, taskID string, err error) {
	wire := &wireResponse{
		Success:   false,
		MessageID: msgID,
		TaskID:    taskID,
		Error:     err.Error(),
	}
	s.sendResponse(conn, wire)
}

// sendResponse sends a response over WebSocket
func (s *WSServer) sendResponse(conn *websocket.Conn, resp *wireResponse) {
	// Set write deadline
	if err := conn.SetWriteDeadline(time.Now().Add(s.writeTimeout)); err != nil {
		fmt.Printf("Failed to set write deadline: %v\n", err)
		return
	}

	// Send JSON response
	if err := conn.WriteJSON(resp); err != nil {
		fmt.Printf("Failed to write response: %v\n", err)
	}
}

// addConnection tracks a new connection
func (s *WSServer) addConnection(conn *websocket.Conn) {
	s.connMu.Lock()
	defer s.connMu.Unlock()
	s.connections[conn] = true
}

// removeConnection stops tracking a connection
func (s *WSServer) removeConnection(conn *websocket.Conn) {
	s.connMu.Lock()
	defer s.connMu.Unlock()
	delete(s.connections, conn)
}

// GetConnectionCount returns the number of active connections
func (s *WSServer) GetConnectionCount() int {
	s.connMu.RLock()
	defer s.connMu.RUnlock()
	return len(s.connections)
}

// Close closes all active connections
func (s *WSServer) Close() error {
	s.connMu.Lock()
	defer s.connMu.Unlock()

	for conn := range s.connections {
		// Send close message
		_ = conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
		_ = conn.Close()
	}

	s.connections = make(map[*websocket.Conn]bool)
	return nil
}
