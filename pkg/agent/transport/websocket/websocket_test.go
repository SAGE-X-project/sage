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
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/transport"
)

func TestWSTransport_Send(t *testing.T) {
	t.Run("Successful message send", func(t *testing.T) {
		// Create test server
		handler := func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
			// Verify received message
			if msg.ID != "test-msg-123" {
				t.Errorf("Expected message ID 'test-msg-123', got '%s'", msg.ID)
			}
			if msg.DID != "did:sage:ethereum:0x123" {
				t.Errorf("Expected DID 'did:sage:ethereum:0x123', got '%s'", msg.DID)
			}
			if string(msg.Payload) != "test payload" {
				t.Errorf("Expected payload 'test payload', got '%s'", string(msg.Payload))
			}

			// Return success response
			return &transport.Response{
				Success:   true,
				MessageID: msg.ID,
				TaskID:    msg.TaskID,
				Data:      []byte("response data"),
			}, nil
		}

		server := NewWSServer(handler)
		testServer := httptest.NewServer(server.Handler())
		defer testServer.Close()

		// Convert http:// to ws://
		wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

		// Create WebSocket transport client
		client := NewWSTransport(wsURL)
		defer client.Close()

		// Send message
		msg := &transport.SecureMessage{
			ID:        "test-msg-123",
			ContextID: "ctx-456",
			TaskID:    "task-789",
			Payload:   []byte("test payload"),
			DID:       "did:sage:ethereum:0x123",
			Signature: []byte("signature"),
			Role:      "user",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		resp, err := client.Send(ctx, msg)
		if err != nil {
			t.Fatalf("Send failed: %v", err)
		}

		// Verify response
		if !resp.Success {
			t.Errorf("Expected success=true, got false")
		}
		if resp.MessageID != "test-msg-123" {
			t.Errorf("Expected MessageID 'test-msg-123', got '%s'", resp.MessageID)
		}
		if string(resp.Data) != "response data" {
			t.Errorf("Expected data 'response data', got '%s'", string(resp.Data))
		}
	})

	t.Run("Server error handling", func(t *testing.T) {
		// Create test server that returns error
		handler := func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
			return nil, fmt.Errorf("server processing error")
		}

		server := NewWSServer(handler)
		testServer := httptest.NewServer(server.Handler())
		defer testServer.Close()

		wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

		client := NewWSTransport(wsURL)
		defer client.Close()

		msg := &transport.SecureMessage{
			ID:      "test-msg-123",
			Payload: []byte("test payload"),
			DID:     "did:sage:ethereum:0x123",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		resp, err := client.Send(ctx, msg)
		if err != nil {
			t.Fatalf("Send failed: %v", err)
		}

		// Verify error response
		if resp.Success {
			t.Errorf("Expected success=false, got true")
		}
		if resp.Error == nil {
			t.Errorf("Expected error to be set")
		} else if resp.Error.Error() != "server processing error" {
			t.Errorf("Expected error 'server processing error', got '%s'", resp.Error.Error())
		}
	})

	t.Run("Multiple messages on same connection", func(t *testing.T) {
		messageCount := 0
		handler := func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
			messageCount++
			return &transport.Response{
				Success:   true,
				MessageID: msg.ID,
				Data:      []byte(fmt.Sprintf("response %d", messageCount)),
			}, nil
		}

		server := NewWSServer(handler)
		testServer := httptest.NewServer(server.Handler())
		defer testServer.Close()

		wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

		client := NewWSTransport(wsURL)
		defer client.Close()

		// Send multiple messages
		for i := 1; i <= 3; i++ {
			msg := &transport.SecureMessage{
				ID:      fmt.Sprintf("msg-%d", i),
				Payload: []byte(fmt.Sprintf("payload %d", i)),
				DID:     "did:sage:ethereum:0x123",
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			resp, err := client.Send(ctx, msg)
			cancel()

			if err != nil {
				t.Fatalf("Send %d failed: %v", i, err)
			}
			if !resp.Success {
				t.Errorf("Message %d: expected success", i)
			}
		}

		if messageCount != 3 {
			t.Errorf("Expected 3 messages, got %d", messageCount)
		}
	})

	t.Run("Invalid message handling", func(t *testing.T) {
		handler := func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
			return &transport.Response{Success: true}, nil
		}

		server := NewWSServer(handler)
		testServer := httptest.NewServer(server.Handler())
		defer testServer.Close()

		wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

		client := NewWSTransport(wsURL)
		defer client.Close()

		// Test nil message
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := client.Send(ctx, nil)
		if err == nil {
			t.Errorf("Expected error for nil message")
		}
	})

	t.Run("Connection timeout", func(t *testing.T) {
		// Create client with short timeout
		client := NewWSTransportWithTimeouts("ws://localhost:19999", 100*time.Millisecond, 1*time.Second, 1*time.Second)
		defer client.Close()

		msg := &transport.SecureMessage{
			ID:      "test-msg",
			Payload: []byte("test"),
			DID:     "did:sage:ethereum:0x123",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		_, err := client.Send(ctx, msg)
		if err == nil {
			t.Errorf("Expected connection error")
		}
	})
}

func TestWSServer_Validation(t *testing.T) {
	t.Run("Missing required fields", func(t *testing.T) {
		receivedError := false
		handler := func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
			t.Errorf("Handler should not be called for invalid message")
			return nil, fmt.Errorf("should not reach here")
		}

		server := NewWSServer(handler)
		testServer := httptest.NewServer(server.Handler())
		defer testServer.Close()

		wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

		tests := []struct {
			name    string
			message *transport.SecureMessage
		}{
			{
				name: "Missing ID",
				message: &transport.SecureMessage{
					Payload: []byte("payload"),
					DID:     "did:sage:ethereum:0x123",
				},
			},
			{
				name: "Missing DID",
				message: &transport.SecureMessage{
					ID:      "msg-123",
					Payload: []byte("payload"),
				},
			},
			{
				name: "Missing Payload",
				message: &transport.SecureMessage{
					ID:  "msg-123",
					DID: "did:sage:ethereum:0x123",
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				client := NewWSTransport(wsURL)
				defer client.Close()

				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()

				resp, err := client.Send(ctx, tt.message)
				if err != nil {
					// Error during send is acceptable
					receivedError = true
					return
				}

				// Response should indicate failure
				if resp.Success {
					t.Errorf("Expected failure for invalid message")
				} else {
					receivedError = true
				}
			})
		}

		if !receivedError {
			t.Errorf("Expected at least one validation error")
		}
	})
}

func TestWSServer_ConnectionCount(t *testing.T) {
	handler := func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
		time.Sleep(100 * time.Millisecond) // Keep connection alive briefly
		return &transport.Response{Success: true, MessageID: msg.ID}, nil
	}

	server := NewWSServer(handler)
	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	// Initial count should be 0
	if count := server.GetConnectionCount(); count != 0 {
		t.Errorf("Expected 0 connections, got %d", count)
	}

	// Create client and connect
	client := NewWSTransport(wsURL)
	defer client.Close()

	msg := &transport.SecureMessage{
		ID:      "test-msg",
		Payload: []byte("test"),
		DID:     "did:sage:ethereum:0x123",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Send message to establish connection
	go client.Send(ctx, msg)

	// Wait for connection to be established
	time.Sleep(50 * time.Millisecond)

	// Check connection count
	if count := server.GetConnectionCount(); count != 1 {
		t.Errorf("Expected 1 connection, got %d", count)
	}
}
