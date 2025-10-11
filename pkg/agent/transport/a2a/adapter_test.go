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

//go:build a2a
// +build a2a

package a2a

import (
	"context"
	"testing"

	a2apb "github.com/a2aproject/a2a/grpc"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/sage-x-project/sage/pkg/agent/transport"
)

// mockA2AClient implements a2apb.A2AServiceClient for testing
type mockA2AClient struct {
	a2apb.A2AServiceClient // Embed to get default implementations
	sendFunc               func(ctx context.Context, in *a2apb.SendMessageRequest, opts ...grpc.CallOption) (*a2apb.SendMessageResponse, error)
}

func (m *mockA2AClient) SendMessage(ctx context.Context, in *a2apb.SendMessageRequest, opts ...grpc.CallOption) (*a2apb.SendMessageResponse, error) {
	if m.sendFunc != nil {
		return m.sendFunc(ctx, in, opts...)
	}

	// Create a simple mock response with message
	dataMap := map[string]interface{}{"result": "mock response"}
	dataStruct, _ := structpb.NewStruct(dataMap)

	return &a2apb.SendMessageResponse{
		Payload: &a2apb.SendMessageResponse_Msg{
			Msg: &a2apb.Message{
				MessageId: in.GetRequest().MessageId,
				TaskId:    in.GetRequest().TaskId,
				Role:      a2apb.Role_ROLE_AGENT,
				Content: []*a2apb.Part{
					{
						Part: &a2apb.Part_Data{
							Data: &a2apb.DataPart{
								Data: dataStruct,
							},
						},
					},
				},
			},
		},
	}, nil
}

// mockMessageHandler implements MessageHandler for testing
type mockMessageHandler struct {
	handleFunc func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error)
}

func (m *mockMessageHandler) HandleMessage(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
	if m.handleFunc != nil {
		return m.handleFunc(ctx, msg)
	}
	return &transport.Response{
		Success:   true,
		MessageID: msg.ID,
		TaskID:    msg.TaskID,
		Data:      []byte("mock handler response"),
	}, nil
}

func TestA2ATransport_Send(t *testing.T) {
	mockClient := &mockA2AClient{}
	tr := &A2ATransport{client: mockClient}

	msg := &transport.SecureMessage{
		ID:        "test-id",
		ContextID: "ctx-1",
		TaskID:    "task-1",
		Payload:   []byte(`{"test":"payload"}`),
		DID:       "did:sage:test",
		Signature: []byte("sig"),
		Role:      "user",
		Metadata:  map[string]string{"key": "value"},
	}

	resp, err := tr.Send(context.Background(), msg)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if !resp.Success {
		t.Error("expected success=true")
	}
	if resp.MessageID != msg.ID {
		t.Errorf("expected messageID=%s, got %s", msg.ID, resp.MessageID)
	}
}

func TestA2AServerAdapter_SendMessage(t *testing.T) {
	handler := &mockMessageHandler{}
	adapter := NewA2AServerAdapter(handler)

	payloadData, _ := structpb.NewStruct(map[string]interface{}{"test": "payload"})
	reqMetadata, _ := structpb.NewStruct(map[string]interface{}{
		"did":       "did:sage:test",
		"signature": "c2ln", // base64 of "sig"
		"key":       "value",
	})

	req := &a2apb.SendMessageRequest{
		Request: &a2apb.Message{
			MessageId: "test-id",
			ContextId: "ctx-1",
			TaskId:    "task-1",
			Role:      a2apb.Role_ROLE_USER,
			Content: []*a2apb.Part{
				{
					Part: &a2apb.Part_Data{
						Data: &a2apb.DataPart{
							Data: payloadData,
						},
					},
				},
			},
		},
		Metadata: reqMetadata,
	}

	resp, err := adapter.SendMessage(context.Background(), req)
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	if resp.GetMsg() == nil {
		t.Error("expected response message")
	}
	if resp.GetMsg().MessageId != "test-id" {
		t.Errorf("expected messageId=test-id, got %s", resp.GetMsg().MessageId)
	}
}

func TestSecureMessageToA2A(t *testing.T) {
	msg := &transport.SecureMessage{
		ID:        "test-id",
		ContextID: "ctx-1",
		TaskID:    "task-1",
		Payload:   []byte(`{"test":"payload"}`),
		DID:       "did:sage:test",
		Signature: []byte("sig"),
		Role:      "user",
		Metadata:  map[string]string{"k1": "v1"},
	}

	a2aMsg, reqMeta, err := secureMessageToA2A(msg)
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}

	if a2aMsg.MessageId != msg.ID {
		t.Errorf("ID mismatch: got %s", a2aMsg.MessageId)
	}
	if reqMeta.Fields["did"].GetStringValue() != msg.DID {
		t.Errorf("DID mismatch in metadata")
	}
	if len(a2aMsg.Content) == 0 {
		t.Error("expected content parts")
	}
}

func TestA2AMessageToSecure(t *testing.T) {
	payloadData, _ := structpb.NewStruct(map[string]interface{}{"test": "payload"})
	reqMetadata, _ := structpb.NewStruct(map[string]interface{}{
		"did":       "did:sage:test",
		"signature": "c2ln", // base64 of "sig"
		"k1":        "v1",
	})

	a2aMsg := &a2apb.Message{
		MessageId: "test-id",
		ContextId: "ctx-1",
		TaskId:    "task-1",
		Role:      a2apb.Role_ROLE_USER,
		Content: []*a2apb.Part{
			{
				Part: &a2apb.Part_Data{
					Data: &a2apb.DataPart{
						Data: payloadData,
					},
				},
			},
		},
	}

	msg, err := a2aMessageToSecure(a2aMsg, reqMetadata)
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}

	if msg.ID != a2aMsg.MessageId {
		t.Errorf("ID mismatch: got %s", msg.ID)
	}
	if msg.DID != "did:sage:test" {
		t.Errorf("DID mismatch: got %s", msg.DID)
	}
	if msg.Metadata["k1"] != "v1" {
		t.Errorf("metadata mismatch")
	}
}
