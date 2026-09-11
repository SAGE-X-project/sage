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

import "fmt"

// WireMessage is the JSON form of SecureMessage on every transport (HTTP,
// WebSocket). It is the single definition of the field names on the wire.
type WireMessage struct {
	ID        string            `json:"id"`
	ContextID string            `json:"context_id,omitempty"`
	TaskID    string            `json:"task_id,omitempty"`
	Payload   []byte            `json:"payload"`
	DID       string            `json:"did"`
	Signature []byte            `json:"signature"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Role      string            `json:"role,omitempty"`
}

// WireResponse is the JSON form of Response on every transport.
type WireResponse struct {
	Success   bool   `json:"success"`
	MessageID string `json:"message_id"`
	TaskID    string `json:"task_id,omitempty"`
	Data      []byte `json:"data,omitempty"`
	Error     string `json:"error,omitempty"`
}

// ToWireMessage converts a SecureMessage to its wire form.
func ToWireMessage(msg *SecureMessage) *WireMessage {
	return &WireMessage{
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

// FromWireMessage converts a received wire message to a SecureMessage.
// Metadata is never nil on the result.
func FromWireMessage(wire *WireMessage) *SecureMessage {
	msg := &SecureMessage{
		ID:        wire.ID,
		ContextID: wire.ContextID,
		TaskID:    wire.TaskID,
		Payload:   wire.Payload,
		DID:       wire.DID,
		Signature: wire.Signature,
		Metadata:  wire.Metadata,
		Role:      wire.Role,
	}
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]string)
	}
	return msg
}

// ToWireResponse converts a Response to its wire form. A non-nil Error forces
// Success to false.
func ToWireResponse(resp *Response) *WireResponse {
	wire := &WireResponse{
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

// FromWireResponse converts a received wire response to a Response, filling
// in the request's message and task ids when the response omits them.
func FromWireResponse(wire *WireResponse, msgID, taskID string) *Response {
	result := &Response{
		Success:   wire.Success,
		MessageID: wire.MessageID,
		TaskID:    wire.TaskID,
		Data:      wire.Data,
	}
	if result.MessageID == "" {
		result.MessageID = msgID
	}
	if result.TaskID == "" {
		result.TaskID = taskID
	}
	if wire.Error != "" {
		result.Error = fmt.Errorf("%s", wire.Error)
		result.Success = false
	}
	return result
}
