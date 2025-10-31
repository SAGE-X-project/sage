// SAGE - Secure Agent Guarantee Engine
// Copyright (C) 2025 SAGE-X-project
//
// Edge case and error handling tests for HPKE Client

package hpke

import (
	"context"
	"errors"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/transport"
	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/assert"
)

// mockTransport simulates various transport behaviors for testing
type mockTransport struct {
	response      *transport.Response
	err           error
	sendCallCount int
}

func (m *mockTransport) Send(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
	m.sendCallCount++
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

// Test sendAndGetSignedMsg error handling paths (PR #118)
func TestClient_SendAndGetSignedMsg_ErrorHandling(t *testing.T) {
	helpers.LogTestSection(t, "6.3.1", "HPKE Client Error Handling - Enhanced Error Messages")

	ctx := context.Background()
	msg := &transport.SecureMessage{
		TaskID:  "hpke/init",
		Payload: []byte("test"),
	}

	t.Run("Transport send error", func(t *testing.T) {
		mockTransport := &mockTransport{
			err: errors.New("network timeout"),
		}

		client := &Client{transport: mockTransport}
		resp, err := client.sendAndGetSignedMsg(ctx, msg)

		assert.Nil(t, resp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "transport send")
		assert.Contains(t, err.Error(), "network timeout")
		helpers.LogSuccess(t, "Transport error handled with context")
	})

	t.Run("Nil response from server", func(t *testing.T) {
		mockTransport := &mockTransport{
			response: nil,
			err:      nil,
		}

		client := &Client{transport: mockTransport}
		resp, err := client.sendAndGetSignedMsg(ctx, msg)

		assert.Nil(t, resp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil response")
		helpers.LogSuccess(t, "Nil response detected")
	})

	t.Run("Response with Success=false and Error field", func(t *testing.T) {
		mockTransport := &mockTransport{
			response: &transport.Response{
				Success: false,
				Error:   errors.New("authentication failed"),
				Data:    []byte{},
			},
		}

		client := &Client{transport: mockTransport}
		resp, err := client.sendAndGetSignedMsg(ctx, msg)

		assert.Nil(t, resp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "handshake failed")
		assert.Contains(t, err.Error(), "authentication failed")
		helpers.LogSuccess(t, "Error field extracted and reported")
	})

	t.Run("Response with Success=false and error in Data", func(t *testing.T) {
		mockTransport := &mockTransport{
			response: &transport.Response{
				Success: false,
				Error:   nil,
				Data:    []byte("  Invalid signature  "),
			},
		}

		client := &Client{transport: mockTransport}
		resp, err := client.sendAndGetSignedMsg(ctx, msg)

		assert.Nil(t, resp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "handshake failed")
		assert.Contains(t, err.Error(), "Invalid signature")
		assert.NotContains(t, err.Error(), "  ", "Should trim whitespace")
		helpers.LogSuccess(t, "Error message from Data field extracted")
	})

	t.Run("Response with Success=false but no error details", func(t *testing.T) {
		mockTransport := &mockTransport{
			response: &transport.Response{
				Success: false,
				Error:   nil,
				Data:    []byte{},
			},
		}

		client := &Client{transport: mockTransport}
		resp, err := client.sendAndGetSignedMsg(ctx, msg)

		assert.Nil(t, resp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no error details provided")
		helpers.LogSuccess(t, "Generic error for missing details")
	})

	t.Run("Response with Success=true but empty Data", func(t *testing.T) {
		mockTransport := &mockTransport{
			response: &transport.Response{
				Success: true,
				Error:   nil,
				Data:    []byte{},
			},
		}

		client := &Client{transport: mockTransport}
		resp, err := client.sendAndGetSignedMsg(ctx, msg)

		assert.Nil(t, resp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty response data")
		helpers.LogSuccess(t, "Empty data detected despite Success=true")
	})

	t.Run("Valid successful response", func(t *testing.T) {
		mockTransport := &mockTransport{
			response: &transport.Response{
				Success: true,
				Error:   nil,
				Data:    []byte("valid response data"),
			},
		}

		client := &Client{transport: mockTransport}
		resp, err := client.sendAndGetSignedMsg(ctx, msg)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.Equal(t, "valid response data", string(resp.Data))
		helpers.LogSuccess(t, "Valid response accepted")
	})

	t.Run("Multiple consecutive calls", func(t *testing.T) {
		mockTransport := &mockTransport{
			response: &transport.Response{
				Success: true,
				Data:    []byte("test"),
			},
		}

		client := &Client{transport: mockTransport}

		// Call multiple times
		for i := 0; i < 3; i++ {
			resp, err := client.sendAndGetSignedMsg(ctx, msg)
			assert.NoError(t, err)
			assert.NotNil(t, resp)
		}

		assert.Equal(t, 3, mockTransport.sendCallCount, "Should call transport 3 times")
		helpers.LogSuccess(t, "Multiple calls handled correctly")
	})
}

// Test error handling with context cancellation
func TestClient_SendAndGetSignedMsg_ContextCancellation(t *testing.T) {
	helpers.LogTestSection(t, "6.3.2", "HPKE Client - Context Handling")

	t.Run("Context already cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		mockTransport := &mockTransport{
			err: context.Canceled,
		}

		client := &Client{transport: mockTransport}
		msg := &transport.SecureMessage{TaskID: "test"}

		resp, err := client.sendAndGetSignedMsg(ctx, msg)

		assert.Nil(t, resp)
		assert.Error(t, err)
		helpers.LogSuccess(t, "Cancelled context handled")
	})

	t.Run("Context deadline exceeded", func(t *testing.T) {
		mockTransport := &mockTransport{
			err: context.DeadlineExceeded,
		}

		client := &Client{transport: mockTransport}
		msg := &transport.SecureMessage{TaskID: "test"}

		resp, err := client.sendAndGetSignedMsg(context.Background(), msg)

		assert.Nil(t, resp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "deadline")
		helpers.LogSuccess(t, "Deadline exceeded handled")
	})
}

// Test edge cases with response data
func TestClient_SendAndGetSignedMsg_ResponseDataEdgeCases(t *testing.T) {
	helpers.LogTestSection(t, "6.3.3", "HPKE Client - Response Data Edge Cases")

	ctx := context.Background()
	msg := &transport.SecureMessage{TaskID: "test"}

	t.Run("Response with only whitespace in Data", func(t *testing.T) {
		mockTransport := &mockTransport{
			response: &transport.Response{
				Success: false,
				Data:    []byte("   \n\t\r   "),
			},
		}

		client := &Client{transport: mockTransport}
		resp, err := client.sendAndGetSignedMsg(ctx, msg)

		assert.Nil(t, resp)
		assert.Error(t, err)
		// After trimming whitespace, should be treated as empty
		helpers.LogSuccess(t, "Whitespace-only data handled")
	})

	t.Run("Response with very large Data", func(t *testing.T) {
		largeData := make([]byte, 1024*1024) // 1MB
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}

		mockTransport := &mockTransport{
			response: &transport.Response{
				Success: true,
				Data:    largeData,
			},
		}

		client := &Client{transport: mockTransport}
		resp, err := client.sendAndGetSignedMsg(ctx, msg)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, len(largeData), len(resp.Data))
		helpers.LogSuccess(t, "Large response data handled")
	})

	t.Run("Response with binary data in error message", func(t *testing.T) {
		binaryError := []byte{0x00, 0x01, 0xFF, 0xFE, 'E', 'r', 'r', 'o', 'r'}

		mockTransport := &mockTransport{
			response: &transport.Response{
				Success: false,
				Data:    binaryError,
			},
		}

		client := &Client{transport: mockTransport}
		resp, err := client.sendAndGetSignedMsg(ctx, msg)

		assert.Nil(t, resp)
		assert.Error(t, err)
		// Should not panic with binary data
		helpers.LogSuccess(t, "Binary data in error handled")
	})
}

// Benchmark error path performance
func BenchmarkClient_SendAndGetSignedMsg_ErrorPath(b *testing.B) {
	mockTransport := &mockTransport{
		response: &transport.Response{
			Success: false,
			Error:   errors.New("test error"),
		},
	}

	client := &Client{transport: mockTransport}
	ctx := context.Background()
	msg := &transport.SecureMessage{TaskID: "test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.sendAndGetSignedMsg(ctx, msg)
	}
}

func BenchmarkClient_SendAndGetSignedMsg_SuccessPath(b *testing.B) {
	mockTransport := &mockTransport{
		response: &transport.Response{
			Success: true,
			Data:    []byte("test data"),
		},
	}

	client := &Client{transport: mockTransport}
	ctx := context.Background()
	msg := &transport.SecureMessage{TaskID: "test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.sendAndGetSignedMsg(ctx, msg)
	}
}
