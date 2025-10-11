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
	"encoding/base64"
	"encoding/json"
	"fmt"

	a2apb "github.com/a2aproject/a2a/grpc"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/sage-x-project/sage/pkg/agent/transport"
)

// A2ATransport implements transport.MessageTransport using A2A protocol over gRPC
type A2ATransport struct {
	client a2apb.A2AServiceClient
}

// NewA2ATransport creates a new A2A transport from a gRPC connection
func NewA2ATransport(conn grpc.ClientConnInterface) *A2ATransport {
	return &A2ATransport{
		client: a2apb.NewA2AServiceClient(conn),
	}
}

// Send implements transport.MessageTransport.Send
func (t *A2ATransport) Send(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
	if msg == nil {
		return nil, fmt.Errorf("nil message")
	}

	// Convert SecureMessage to A2A protobuf message
	a2aMsg, reqMetadata, err := secureMessageToA2A(msg)
	if err != nil {
		return nil, fmt.Errorf("convert to a2a: %w", err)
	}

	// Send via gRPC
	req := &a2apb.SendMessageRequest{
		Request:  a2aMsg,
		Metadata: reqMetadata,
	}

	resp, err := t.client.SendMessage(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("grpc send: %w", err)
	}

	// Convert A2A response to transport.Response
	return a2aResponseToTransport(resp)
}

// secureMessageToA2A converts transport.SecureMessage to a2a.Message
func secureMessageToA2A(msg *transport.SecureMessage) (*a2apb.Message, *structpb.Struct, error) {
	// Parse payload as JSON and convert to structpb
	var payloadMap map[string]interface{}
	if err := json.Unmarshal(msg.Payload, &payloadMap); err != nil {
		return nil, nil, fmt.Errorf("unmarshal payload: %w", err)
	}

	payloadStruct, err := structpb.NewStruct(payloadMap)
	if err != nil {
		return nil, nil, fmt.Errorf("convert payload to struct: %w", err)
	}

	// Convert metadata map to structpb
	metadataMap := make(map[string]interface{})
	for k, v := range msg.Metadata {
		metadataMap[k] = v
	}
	// Add DID and signature to metadata
	metadataMap["did"] = msg.DID
	metadataMap["signature"] = base64.StdEncoding.EncodeToString(msg.Signature)

	requestMetadata, err := structpb.NewStruct(metadataMap)
	if err != nil {
		return nil, nil, fmt.Errorf("convert metadata: %w", err)
	}

	// Determine role
	role := a2apb.Role_ROLE_USER
	if msg.Role == "agent" {
		role = a2apb.Role_ROLE_AGENT
	}

	// Create A2A message
	a2aMsg := &a2apb.Message{
		MessageId: msg.ID,
		ContextId: msg.ContextID,
		TaskId:    msg.TaskID,
		Role:      role,
		Content: []*a2apb.Part{
			{
				Part: &a2apb.Part_Data{
					Data: &a2apb.DataPart{
						Data: payloadStruct,
					},
				},
			},
		},
	}

	return a2aMsg, requestMetadata, nil
}

// a2aResponseToTransport converts a2a.SendMessageResponse to transport.Response
func a2aResponseToTransport(resp *a2apb.SendMessageResponse) (*transport.Response, error) {
	if resp == nil {
		return nil, fmt.Errorf("nil a2a response")
	}

	// Extract response message
	var responseData []byte
	var messageID, taskID string

	if msg := resp.GetMsg(); msg != nil {
		messageID = msg.MessageId
		taskID = msg.TaskId

		// Extract data from first content part if available
		if len(msg.Content) > 0 {
			if dataPart := msg.Content[0].GetData(); dataPart != nil && dataPart.Data != nil {
				// Convert structpb back to JSON
				jsonBytes, err := json.Marshal(dataPart.Data.AsMap())
				if err == nil {
					responseData = jsonBytes
				}
			}
		}
	}

	return &transport.Response{
		Success:   true, // If we got here, the gRPC call succeeded
		MessageID: messageID,
		TaskID:    taskID,
		Data:      responseData,
		Error:     nil,
	}, nil
}
