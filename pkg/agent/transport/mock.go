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

package transport

import (
	"context"
	"sync"
)

// MockTransport is a mock implementation of MessageTransport for testing.
//
// This allows tests to inject custom behavior without requiring a real
// transport implementation (gRPC server, HTTP server, etc.).
//
// Example usage:
//
//	mock := &transport.MockTransport{
//	    SendFunc: func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
//	        // Custom test logic
//	        return &transport.Response{Success: true, MessageID: msg.ID}, nil
//	    },
//	}
//	client := handshake.NewClient(mock, keyPair)
//	resp, err := client.Invitation(ctx, invMsg, did)
type MockTransport struct {
	// SendFunc is the function to call when Send is invoked.
	// If nil, a default successful response is returned.
	SendFunc func(ctx context.Context, msg *SecureMessage) (*Response, error)

	// Captured messages (for verification in tests)
	SentMessages []*SecureMessage

	// mu protects SentMessages for concurrent access
	mu sync.Mutex
}

// Send implements the MessageTransport interface.
func (m *MockTransport) Send(ctx context.Context, msg *SecureMessage) (*Response, error) {
	// Capture the message for test verification (thread-safe)
	m.mu.Lock()
	m.SentMessages = append(m.SentMessages, msg)
	m.mu.Unlock()

	// Call custom function if provided
	if m.SendFunc != nil {
		return m.SendFunc(ctx, msg)
	}

	// Default: return success
	return &Response{
		Success:   true,
		MessageID: msg.ID,
		TaskID:    msg.TaskID,
		Data:      []byte("mock response"),
	}, nil
}

// Reset clears the captured messages (useful between test cases).
func (m *MockTransport) Reset() {
	m.mu.Lock()
	m.SentMessages = nil
	m.mu.Unlock()
}

// LastMessage returns the most recently sent message (or nil if none).
func (m *MockTransport) LastMessage() *SecureMessage {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.SentMessages) == 0 {
		return nil
	}
	return m.SentMessages[len(m.SentMessages)-1]
}
