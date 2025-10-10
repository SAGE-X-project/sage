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
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/transport"
)

func TestHTTPTransport_Send(t *testing.T) {
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

		server := NewHTTPServer(handler)
		testServer := httptest.NewServer(server.MessagesHandler())
		defer testServer.Close()

		// Create HTTP transport client
		client := NewHTTPTransport(testServer.URL)

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

		resp, err := client.Send(context.Background(), msg)
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

		server := NewHTTPServer(handler)
		testServer := httptest.NewServer(server.MessagesHandler())
		defer testServer.Close()

		// Create HTTP transport client
		client := NewHTTPTransport(testServer.URL)

		// Send message
		msg := &transport.SecureMessage{
			ID:      "test-msg-123",
			Payload: []byte("test payload"),
			DID:     "did:sage:ethereum:0x123",
		}

		resp, err := client.Send(context.Background(), msg)
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

	t.Run("Invalid message handling", func(t *testing.T) {
		// Create test server
		handler := func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
			return &transport.Response{Success: true}, nil
		}

		server := NewHTTPServer(handler)
		testServer := httptest.NewServer(server.MessagesHandler())
		defer testServer.Close()

		// Create HTTP transport client
		client := NewHTTPTransport(testServer.URL)

		// Test nil message
		_, err := client.Send(context.Background(), nil)
		if err == nil {
			t.Errorf("Expected error for nil message")
		}
	})

	t.Run("Metadata and headers", func(t *testing.T) {
		// Create test server
		handler := func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
			// Verify metadata
			if msg.Metadata["custom-key"] != "custom-value" {
				t.Errorf("Expected metadata 'custom-key'='custom-value', got '%s'", msg.Metadata["custom-key"])
			}

			return &transport.Response{
				Success:   true,
				MessageID: msg.ID,
			}, nil
		}

		server := NewHTTPServer(handler)
		testServer := httptest.NewServer(server.MessagesHandler())
		defer testServer.Close()

		// Create HTTP transport client
		client := NewHTTPTransport(testServer.URL)

		// Send message with metadata
		msg := &transport.SecureMessage{
			ID:      "test-msg-123",
			Payload: []byte("test payload"),
			DID:     "did:sage:ethereum:0x123",
			Metadata: map[string]string{
				"custom-key": "custom-value",
			},
		}

		resp, err := client.Send(context.Background(), msg)
		if err != nil {
			t.Fatalf("Send failed: %v", err)
		}

		if !resp.Success {
			t.Errorf("Expected success=true, got false")
		}
	})
}

func TestHTTPServer_Validation(t *testing.T) {
	t.Run("Missing required fields", func(t *testing.T) {
		handler := func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
			t.Errorf("Handler should not be called for invalid message")
			return nil, fmt.Errorf("should not reach here")
		}

		server := NewHTTPServer(handler)

		tests := []struct {
			name    string
			message *wireMessage
		}{
			{
				name: "Missing ID",
				message: &wireMessage{
					Payload: []byte("payload"),
					DID:     "did:sage:ethereum:0x123",
				},
			},
			{
				name: "Missing DID",
				message: &wireMessage{
					ID:      "msg-123",
					Payload: []byte("payload"),
				},
			},
			{
				name: "Missing Payload",
				message: &wireMessage{
					ID:  "msg-123",
					DID: "did:sage:ethereum:0x123",
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				testServer := httptest.NewServer(server.MessagesHandler())
				defer testServer.Close()

				client := NewHTTPTransport(testServer.URL)

				// Convert wireMessage to SecureMessage
				msg := &transport.SecureMessage{
					ID:      tt.message.ID,
					Payload: tt.message.Payload,
					DID:     tt.message.DID,
				}

				resp, err := client.Send(context.Background(), msg)
				if err != nil {
					// Error during send is acceptable
					return
				}

				// Response should indicate failure
				if resp.Success {
					t.Errorf("Expected failure for invalid message")
				}
			})
		}
	})

	t.Run("Method not allowed", func(t *testing.T) {
		handler := func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
			return &transport.Response{Success: true}, nil
		}

		server := NewHTTPServer(handler)
		testServer := httptest.NewServer(server.MessagesHandler())
		defer testServer.Close()

		// Try GET request
		resp, err := http.Get(testServer.URL)
		if err != nil {
			t.Fatalf("GET request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, resp.StatusCode)
		}
	})
}
