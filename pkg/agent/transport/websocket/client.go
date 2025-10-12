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
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sage-x-project/sage/pkg/agent/transport"
)

// WSTransport implements MessageTransport using WebSocket protocol.
//
// This transport maintains a persistent WebSocket connection for
// bidirectional communication with the remote agent.
//
// Example usage:
//
//	// Create WebSocket transport
//	transport := websocket.NewWSTransport("wss://agent.example.com/ws")
//
//	// Use with handshake client
//	client := handshake.NewClient(transport, keyPair)
type WSTransport struct {
	url          string
	conn         *websocket.Conn
	mu           sync.Mutex
	dialTimeout  time.Duration
	readTimeout  time.Duration
	writeTimeout time.Duration

	// Response handling
	pendingResponses map[string]chan *wireResponse
	pendingMu        sync.RWMutex

	// Connection state
	connected bool
	connMu    sync.RWMutex
}

// NewWSTransport creates a new WebSocket transport client.
//
// Parameters:
//   - url: The WebSocket URL (e.g., "wss://agent.example.com/ws")
func NewWSTransport(url string) *WSTransport {
	return &WSTransport{
		url:              url,
		dialTimeout:      30 * time.Second,
		readTimeout:      60 * time.Second,
		writeTimeout:     30 * time.Second,
		pendingResponses: make(map[string]chan *wireResponse),
	}
}

// NewWSTransportWithTimeouts creates a WebSocket transport with custom timeouts.
func NewWSTransportWithTimeouts(url string, dialTimeout, readTimeout, writeTimeout time.Duration) *WSTransport {
	return &WSTransport{
		url:              url,
		dialTimeout:      dialTimeout,
		readTimeout:      readTimeout,
		writeTimeout:     writeTimeout,
		pendingResponses: make(map[string]chan *wireResponse),
	}
}

// Connect establishes the WebSocket connection.
func (t *WSTransport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Check if already connected
	if t.conn != nil {
		return nil
	}

	// Create dialer with timeout
	dialer := &websocket.Dialer{
		HandshakeTimeout: t.dialTimeout,
	}

	// Connect to WebSocket
	conn, resp, err := dialer.DialContext(ctx, t.url, nil)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("WebSocket dial failed (HTTP %d): %w", resp.StatusCode, err)
		}
		return fmt.Errorf("WebSocket dial failed: %w", err)
	}

	t.conn = conn
	t.setConnected(true)

	// Start response reader
	go t.readResponses()

	return nil
}

// Send implements the MessageTransport interface.
//
// Sends the SecureMessage via WebSocket and waits for the response.
func (t *WSTransport) Send(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
	if msg == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}

	// Ensure connection
	if err := t.ensureConnected(ctx); err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}

	// Convert to wire format
	wireMsg := toWireMessage(msg)

	// Create response channel
	respChan := make(chan *wireResponse, 1)
	t.pendingMu.Lock()
	t.pendingResponses[msg.ID] = respChan
	t.pendingMu.Unlock()

	// Clean up response channel on exit
	defer func() {
		t.pendingMu.Lock()
		delete(t.pendingResponses, msg.ID)
		t.pendingMu.Unlock()
		close(respChan)
	}()

	// Send message
	if err := t.writeMessage(wireMsg); err != nil {
		return &transport.Response{
			Success:   false,
			MessageID: msg.ID,
			TaskID:    msg.TaskID,
			Error:     fmt.Errorf("send failed: %w", err),
		}, err
	}

	// Wait for response with timeout
	select {
	case <-ctx.Done():
		return &transport.Response{
			Success:   false,
			MessageID: msg.ID,
			TaskID:    msg.TaskID,
			Error:     ctx.Err(),
		}, ctx.Err()
	case wireResp := <-respChan:
		return fromWireResponse(wireResp, msg.ID, msg.TaskID), nil
	case <-time.After(t.readTimeout):
		return &transport.Response{
			Success:   false,
			MessageID: msg.ID,
			TaskID:    msg.TaskID,
			Error:     fmt.Errorf("response timeout"),
		}, fmt.Errorf("response timeout")
	}
}

// ensureConnected checks connection and reconnects if needed
func (t *WSTransport) ensureConnected(ctx context.Context) error {
	if t.isConnected() {
		return nil
	}
	return t.Connect(ctx)
}

// writeMessage writes a message to the WebSocket
func (t *WSTransport) writeMessage(msg *wireMessage) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.conn == nil {
		return fmt.Errorf("not connected")
	}

	// Set write deadline
	if err := t.conn.SetWriteDeadline(time.Now().Add(t.writeTimeout)); err != nil {
		return fmt.Errorf("set write deadline: %w", err)
	}

	// Write JSON message
	if err := t.conn.WriteJSON(msg); err != nil {
		t.setConnected(false)
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}

// readResponses continuously reads responses from the WebSocket
func (t *WSTransport) readResponses() {
	defer t.setConnected(false)

	for {
		// Check if still connected
		if !t.isConnected() {
			return
		}

		// Set read deadline
		t.mu.Lock()
		conn := t.conn
		t.mu.Unlock()

		if conn == nil {
			return
		}

		if err := conn.SetReadDeadline(time.Now().Add(t.readTimeout)); err != nil {
			return
		}

		// Read message
		var wireResp wireResponse
		if err := conn.ReadJSON(&wireResp); err != nil {
			// Check if it's a normal close
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				fmt.Printf("WebSocket read error: %v\n", err)
			}
			return
		}

		// Deliver response to waiting sender
		t.pendingMu.RLock()
		if respChan, ok := t.pendingResponses[wireResp.MessageID]; ok {
			select {
			case respChan <- &wireResp:
			default:
				// Channel full or closed, skip
			}
		}
		t.pendingMu.RUnlock()
	}
}

// Close closes the WebSocket connection
func (t *WSTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.conn == nil {
		return nil
	}

	// Send close message
	err := t.conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
	)

	// Close connection
	closeErr := t.conn.Close()
	t.conn = nil
	t.setConnected(false)

	if err != nil {
		return err
	}
	return closeErr
}

// isConnected checks connection state
func (t *WSTransport) isConnected() bool {
	t.connMu.RLock()
	defer t.connMu.RUnlock()
	return t.connected
}

// setConnected sets connection state
func (t *WSTransport) setConnected(connected bool) {
	t.connMu.Lock()
	defer t.connMu.Unlock()
	t.connected = connected
}

// wireMessage is the WebSocket wire format for SecureMessage
type wireMessage struct {
	ID        string            `json:"id"`
	ContextID string            `json:"context_id,omitempty"`
	TaskID    string            `json:"task_id,omitempty"`
	Payload   []byte            `json:"payload"`
	DID       string            `json:"did"`
	Signature []byte            `json:"signature"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Role      string            `json:"role,omitempty"`
}

// wireResponse is the WebSocket wire format for Response
type wireResponse struct {
	Success   bool   `json:"success"`
	MessageID string `json:"message_id"`
	TaskID    string `json:"task_id,omitempty"`
	Data      []byte `json:"data,omitempty"`
	Error     string `json:"error,omitempty"`
}

// toWireMessage converts transport.SecureMessage to WebSocket wire format
func toWireMessage(msg *transport.SecureMessage) *wireMessage {
	return &wireMessage{
		ID:        msg.ID,
		ContextID: msg.ContextID,
		TaskID:    msg.TaskID,
		Payload:   msg.Payload,
		DID:       msg.DID,
		Signature: msg.Signature,
		Metadata:  msg.Metadata,
		Role:      msg.Role,
	}
}

// fromWireResponse converts WebSocket wire response to transport.Response
func fromWireResponse(resp *wireResponse, msgID, taskID string) *transport.Response {
	result := &transport.Response{
		Success:   resp.Success,
		MessageID: resp.MessageID,
		TaskID:    resp.TaskID,
		Data:      resp.Data,
	}

	// Use provided IDs if response doesn't include them
	if result.MessageID == "" {
		result.MessageID = msgID
	}
	if result.TaskID == "" {
		result.TaskID = taskID
	}

	// Convert error string to error type
	if resp.Error != "" {
		result.Error = fmt.Errorf("%s", resp.Error)
		result.Success = false
	}

	return result
}
