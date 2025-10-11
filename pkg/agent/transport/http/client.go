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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/transport"
)

// HTTPTransport implements MessageTransport using HTTP/REST protocol.
//
// This transport sends SecureMessage payloads over HTTP POST requests
// and receives Response data in the HTTP response body.
//
// Example usage:
//
//	// Create HTTP transport
//	transport := http.NewHTTPTransport("https://agent.example.com/messages")
//
//	// Use with handshake client
//	client := handshake.NewClient(transport, keyPair)
type HTTPTransport struct {
	baseURL    string       // Base URL of the remote agent (e.g., https://agent.example.com)
	httpClient *http.Client // HTTP client for making requests
}

// NewHTTPTransport creates a new HTTP transport client.
//
// Parameters:
//   - baseURL: The base URL of the remote agent endpoint (e.g., "https://agent.example.com")
//
// The transport will POST messages to {baseURL}/messages
func NewHTTPTransport(baseURL string) *HTTPTransport {
	return &HTTPTransport{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewHTTPTransportWithClient creates an HTTP transport with a custom HTTP client.
//
// This allows customization of timeout, retry logic, TLS config, etc.
func NewHTTPTransportWithClient(baseURL string, httpClient *http.Client) *HTTPTransport {
	return &HTTPTransport{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// Send implements the MessageTransport interface.
//
// Sends the SecureMessage via HTTP POST to {baseURL}/messages and
// returns the Response from the server.
func (t *HTTPTransport) Send(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
	if msg == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}

	// Convert SecureMessage to HTTP wire format
	wireMsg := toWireMessage(msg)

	// Marshal to JSON
	jsonData, err := json.Marshal(wireMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	// Create HTTP request
	url := t.baseURL + "/messages"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-SAGE-DID", msg.DID)
	req.Header.Set("X-SAGE-Message-ID", msg.ID)
	if msg.ContextID != "" {
		req.Header.Set("X-SAGE-Context-ID", msg.ContextID)
	}
	if msg.TaskID != "" {
		req.Header.Set("X-SAGE-Task-ID", msg.TaskID)
	}

	// Add custom metadata as headers
	for key, value := range msg.Metadata {
		req.Header.Set("X-SAGE-Meta-"+key, value)
	}

	// Send request
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return &transport.Response{
			Success:   false,
			MessageID: msg.ID,
			TaskID:    msg.TaskID,
			Error:     fmt.Errorf("HTTP request failed: %w", err),
		}, err
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &transport.Response{
			Success:   false,
			MessageID: msg.ID,
			TaskID:    msg.TaskID,
			Error:     fmt.Errorf("failed to read response: %w", err),
		}, err
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return &transport.Response{
			Success:   false,
			MessageID: msg.ID,
			TaskID:    msg.TaskID,
			Data:      respBody,
			Error:     fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody)),
		}, nil
	}

	// Parse response
	var wireResp wireResponse
	if err := json.Unmarshal(respBody, &wireResp); err != nil {
		return &transport.Response{
			Success:   false,
			MessageID: msg.ID,
			TaskID:    msg.TaskID,
			Data:      respBody,
			Error:     fmt.Errorf("failed to parse response: %w", err),
		}, err
	}

	// Convert to transport.Response
	return fromWireResponse(&wireResp, msg.ID, msg.TaskID), nil
}

// wireMessage is the JSON representation of SecureMessage for HTTP transport
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

// wireResponse is the JSON representation of Response for HTTP transport
type wireResponse struct {
	Success   bool   `json:"success"`
	MessageID string `json:"message_id"`
	TaskID    string `json:"task_id,omitempty"`
	Data      []byte `json:"data,omitempty"`
	Error     string `json:"error,omitempty"`
}

// toWireMessage converts transport.SecureMessage to HTTP wire format
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

// fromWireResponse converts HTTP wire response to transport.Response
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
