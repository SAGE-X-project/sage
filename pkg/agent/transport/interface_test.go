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

package transport_test

import (
	"context"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockTransport_DefaultBehavior(t *testing.T) {
	mock := &transport.MockTransport{}

	msg := &transport.SecureMessage{
		ID:        "test-id",
		ContextID: "ctx-123",
		TaskID:    "task-456",
		Payload:   []byte("encrypted payload"),
		DID:       "did:sage:ethereum:alice",
		Signature: []byte("signature"),
		Role:      "user",
	}

	resp, err := mock.Send(context.Background(), msg)

	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "test-id", resp.MessageID)
	assert.Equal(t, "task-456", resp.TaskID)
}

func TestMockTransport_CustomFunction(t *testing.T) {
	called := false
	mock := &transport.MockTransport{
		SendFunc: func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
			called = true
			assert.Equal(t, "custom-id", msg.ID)
			return &transport.Response{
				Success:   true,
				MessageID: msg.ID,
				Data:      []byte("custom response"),
			}, nil
		},
	}

	msg := &transport.SecureMessage{
		ID:      "custom-id",
		Payload: []byte("test"),
	}

	resp, err := mock.Send(context.Background(), msg)

	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, "custom response", string(resp.Data))
}

func TestMockTransport_CapturesMessages(t *testing.T) {
	mock := &transport.MockTransport{}

	msg1 := &transport.SecureMessage{ID: "msg-1"}
	msg2 := &transport.SecureMessage{ID: "msg-2"}

	_, _ = mock.Send(context.Background(), msg1)
	_, _ = mock.Send(context.Background(), msg2)

	require.Len(t, mock.SentMessages, 2)
	assert.Equal(t, "msg-1", mock.SentMessages[0].ID)
	assert.Equal(t, "msg-2", mock.SentMessages[1].ID)
}

func TestMockTransport_LastMessage(t *testing.T) {
	mock := &transport.MockTransport{}

	// No messages yet
	assert.Nil(t, mock.LastMessage())

	// Send a message
	msg := &transport.SecureMessage{ID: "last-msg"}
	_, _ = mock.Send(context.Background(), msg)

	// Should return the last message
	last := mock.LastMessage()
	require.NotNil(t, last)
	assert.Equal(t, "last-msg", last.ID)
}

func TestMockTransport_Reset(t *testing.T) {
	mock := &transport.MockTransport{}

	msg := &transport.SecureMessage{ID: "test"}
	_, _ = mock.Send(context.Background(), msg)

	require.Len(t, mock.SentMessages, 1)

	mock.Reset()

	assert.Len(t, mock.SentMessages, 0)
	assert.Nil(t, mock.LastMessage())
}
