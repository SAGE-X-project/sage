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
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/sage-x-project/sage/pkg/agent/transport"
)

// MessageHandler is the interface that server implementations (like hpke.Server, handshake.Server)
// must implement to handle messages
type MessageHandler interface {
	HandleMessage(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error)
}

// A2AServerAdapter adapts a MessageHandler to implement a2a.A2AServiceServer
type A2AServerAdapter struct {
	a2apb.UnimplementedA2AServiceServer
	handler MessageHandler
}

// NewA2AServerAdapter creates a new server adapter
func NewA2AServerAdapter(handler MessageHandler) *A2AServerAdapter {
	return &A2AServerAdapter{
		handler: handler,
	}
}

// SendMessage implements a2apb.A2AServiceServer.SendMessage
func (s *A2AServerAdapter) SendMessage(ctx context.Context, req *a2apb.SendMessageRequest) (*a2apb.SendMessageResponse, error) {
	if req == nil || req.GetRequest() == nil {
		return nil, fmt.Errorf("empty request")
	}

	// Convert A2A message to transport.SecureMessage
	msg, err := a2aMessageToSecure(req.GetRequest(), req.GetMetadata())
	if err != nil {
		return nil, fmt.Errorf("convert from a2a: %w", err)
	}

	// Call the handler
	resp, err := s.handler.HandleMessage(ctx, msg)
	if err != nil {
		return nil, err
	}

	// Convert transport.Response to A2A response
	return transportResponseToA2A(resp)
}

// a2aMessageToSecure converts a2a.Message to transport.SecureMessage
func a2aMessageToSecure(msg *a2apb.Message, reqMetadata *structpb.Struct) (*transport.SecureMessage, error) {
	if msg == nil {
		return nil, fmt.Errorf("nil a2a message")
	}

	// Extract payload from first content part
	var payload []byte
	if len(msg.Content) > 0 {
		if dataPart := msg.Content[0].GetData(); dataPart != nil && dataPart.Data != nil {
			// Convert structpb to JSON
			jsonBytes, err := json.Marshal(dataPart.Data.AsMap())
			if err != nil {
				return nil, fmt.Errorf("marshal data part: %w", err)
			}
			payload = jsonBytes
		}
	}

	// Extract metadata
	metadata := make(map[string]string)
	var did string
	var signature []byte

	if reqMetadata != nil {
		for k, v := range reqMetadata.Fields {
			strVal := v.GetStringValue()
			if strVal != "" {
				if k == "did" {
					did = strVal
				} else if k == "signature" {
					// Decode base64 signature
					sig, err := base64.StdEncoding.DecodeString(strVal)
					if err == nil {
						signature = sig
					}
				} else {
					metadata[k] = strVal
				}
			}
		}
	}

	// Determine role string
	role := "user"
	if msg.Role == a2apb.Role_ROLE_AGENT {
		role = "agent"
	}

	return &transport.SecureMessage{
		ID:        msg.MessageId,
		ContextID: msg.ContextId,
		TaskID:    msg.TaskId,
		Payload:   payload,
		DID:       did,
		Signature: signature,
		Metadata:  metadata,
		Role:      role,
	}, nil
}

// transportResponseToA2A converts transport.Response to a2a.SendMessageResponse
func transportResponseToA2A(resp *transport.Response) (*a2apb.SendMessageResponse, error) {
	if resp == nil {
		return nil, fmt.Errorf("nil transport response")
	}

	// Parse response data as JSON and convert to structpb
	var dataStruct *structpb.Struct
	if len(resp.Data) > 0 {
		var dataMap map[string]interface{}
		if err := json.Unmarshal(resp.Data, &dataMap); err == nil {
			var err2 error
			dataStruct, err2 = structpb.NewStruct(dataMap)
			if err2 != nil {
				return nil, fmt.Errorf("convert response data: %w", err2)
			}
		}
	}

	// Create response message
	responseMsg := &a2apb.Message{
		MessageId: resp.MessageID,
		TaskId:    resp.TaskID,
		Role:      a2apb.Role_ROLE_AGENT,
	}

	// Add data as content if available
	if dataStruct != nil {
		responseMsg.Content = []*a2apb.Part{
			{
				Part: &a2apb.Part_Data{
					Data: &a2apb.DataPart{
						Data: dataStruct,
					},
				},
			},
		}
	}

	return &a2apb.SendMessageResponse{
		Payload: &a2apb.SendMessageResponse_Msg{
			Msg: responseMsg,
		},
	}, nil
}
