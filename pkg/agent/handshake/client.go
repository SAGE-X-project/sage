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


package handshake

import (
	"context"
	"crypto"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	a2a "github.com/a2aproject/a2a/grpc"
	"github.com/google/uuid"
	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/internal/metrics"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type Client struct {
	a2a.A2AServiceClient
	key sagecrypto.KeyPair
}

func NewClient(conn grpc.ClientConnInterface, key sagecrypto.KeyPair) *Client {
	return &Client{
		A2AServiceClient: a2a.NewA2AServiceClient(conn),
		key:              key,
	}
}

// Invitation sends the initial invitation (clear JSON payload).
func (c *Client) Invitation(ctx context.Context, invMsg InvitationMessage, did string) (*a2a.SendMessageResponse, error) {
	start := time.Now()
	metrics.HandshakesInitiated.WithLabelValues("client").Inc()
	defer func() {
		metrics.HandshakeDuration.WithLabelValues(Invitation.String()).Observe(
			time.Since(start).Seconds(),
		)
	}()

	payload, err := toStructPB(invMsg)
	if err != nil {
		metrics.HandshakesFailed.WithLabelValues("marshal_error").Inc()
		return nil, fmt.Errorf("marshal invitation: %w", err)
	}
	msg := &a2a.Message{
		MessageId: uuid.NewString(),
		ContextId: invMsg.ContextID,
		TaskId:    GenerateTaskID(Invitation),
		Role:      a2a.Role_ROLE_USER,
		Content:   []*a2a.Part{{Part: &a2a.Part_Data{Data: &a2a.DataPart{Data: payload}}}},
	}
	bytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(msg)
	if err != nil {
		metrics.HandshakesFailed.WithLabelValues("marshal_error").Inc()
		return nil, fmt.Errorf("marshal for signing: %w", err)
	}
	meta, err := signStruct(c.key, bytes, did)
	if err != nil {
		metrics.HandshakesFailed.WithLabelValues("sign_error").Inc()
		return nil, fmt.Errorf("sign: %w", err)
	}
	resp, err := c.SendMessage(ctx, &a2a.SendMessageRequest{Request: msg, Metadata: meta})
	if err != nil {
		metrics.HandshakesFailed.WithLabelValues("send_error").Inc()
		return nil, err
	}
	metrics.HandshakesCompleted.WithLabelValues("success").Inc()
	return resp, nil
}

// Request encrypts RequestMessage for the peer using bootstrap envelope.
func (c *Client) Request(ctx context.Context, reqMsg RequestMessage, edPeerPub crypto.PublicKey, did string) (*a2a.SendMessageResponse, error) {
	start := time.Now()
	metrics.HandshakesInitiated.WithLabelValues("client").Inc()
	defer func() {
		metrics.HandshakeDuration.WithLabelValues(Request.String()).Observe(
			time.Since(start).Seconds(),
		)
	}()

	reqBytes, err := json.Marshal(reqMsg)
	if err != nil {
		metrics.HandshakesFailed.WithLabelValues("marshal_error").Inc()
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	packet, err := keys.EncryptWithEd25519Peer(edPeerPub, reqBytes)
	if err != nil {
		metrics.HandshakesFailed.WithLabelValues("encrypt_error").Inc()
		return nil, fmt.Errorf("encrypt request: %w", err)
	}
	payload, _ := b64ToStructPB(base64.RawURLEncoding.EncodeToString(packet))
	msg := &a2a.Message{
		MessageId: uuid.NewString(),
		ContextId: reqMsg.ContextID,
		TaskId:    GenerateTaskID(Request),
		Role:      a2a.Role_ROLE_USER,
		Content:   []*a2a.Part{{Part: &a2a.Part_Data{Data: &a2a.DataPart{Data: payload}}}},
	}
	bytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(msg)
	if err != nil {
		metrics.HandshakesFailed.WithLabelValues("marshal_error").Inc()
		return nil, fmt.Errorf("marshal for signing: %w", err)
	}
	meta, err := signStruct(c.key, bytes, did)
	if err != nil {
		metrics.HandshakesFailed.WithLabelValues("sign_error").Inc()
		return nil, fmt.Errorf("sign: %w", err)
	}
	resp, err := c.SendMessage(ctx, &a2a.SendMessageRequest{Request: msg, Metadata: meta})
	if err != nil {
		metrics.HandshakesFailed.WithLabelValues("send_error").Inc()
		return nil, err
	}
	metrics.HandshakesCompleted.WithLabelValues("success").Inc()
	return resp, nil
}

// Response is sent by the agent back to the initiator (bootstrap envelope).
func (c *Client) Response(ctx context.Context, resMsg ResponseMessage, edPeerPub crypto.PublicKey, did string) (*a2a.SendMessageResponse, error) {
	start := time.Now()
	metrics.HandshakesInitiated.WithLabelValues("client").Inc()
	defer func() {
		metrics.HandshakeDuration.WithLabelValues(Response.String()).Observe(
			time.Since(start).Seconds(),
		)
	}()

	resBytes, err := json.Marshal(resMsg)
	if err != nil {
		metrics.HandshakesFailed.WithLabelValues("marshal_error").Inc()
		return nil, fmt.Errorf("marshal response: %w", err)
	}
	packet, err := keys.EncryptWithEd25519Peer(edPeerPub, resBytes)
	if err != nil {
		metrics.HandshakesFailed.WithLabelValues("encrypt_error").Inc()
		return nil, fmt.Errorf("encrypt response: %w", err)
	}
	payload, _ := b64ToStructPB(base64.RawURLEncoding.EncodeToString(packet))
	msg := &a2a.Message{
		MessageId: uuid.NewString(),
		ContextId: resMsg.ContextID,
		TaskId:    GenerateTaskID(Response),
		Role:      a2a.Role_ROLE_AGENT,
		Content:   []*a2a.Part{{Part: &a2a.Part_Data{Data: &a2a.DataPart{Data: payload}}}},
	}
	bytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(msg)
	if err != nil {
		metrics.HandshakesFailed.WithLabelValues("marshal_error").Inc()
		return nil, fmt.Errorf("marshal for signing: %w", err)
	}
	meta, err := signStruct(c.key, bytes, did)
	if err != nil {
		metrics.HandshakesFailed.WithLabelValues("sign_error").Inc()
		return nil, fmt.Errorf("sign: %w", err)
	}
	resp, err := c.SendMessage(ctx, &a2a.SendMessageRequest{Request: msg, Metadata: meta})
	if err != nil {
		metrics.HandshakesFailed.WithLabelValues("send_error").Inc()
		return nil, err
	}
	metrics.HandshakesCompleted.WithLabelValues("success").Inc()
	return resp, nil
}

// Complete notifies completion (clear JSON payload).
func (c *Client) Complete(ctx context.Context, compMsg CompleteMessage, did string) (*a2a.SendMessageResponse, error) {
	start := time.Now()
	metrics.HandshakesInitiated.WithLabelValues("client").Inc()
	defer func() {
		metrics.HandshakeDuration.WithLabelValues(Complete.String()).Observe(
			time.Since(start).Seconds(),
		)
	}()

	payload, err := toStructPB(compMsg)
	if err != nil {
		metrics.HandshakesFailed.WithLabelValues("marshal_error").Inc()
		return nil, fmt.Errorf("marshal complete: %w", err)
	}
	msg := &a2a.Message{
		MessageId: uuid.NewString(),
		ContextId: compMsg.ContextID,
		TaskId:    GenerateTaskID(Complete),
		Role:      a2a.Role_ROLE_USER,
		Content:   []*a2a.Part{{Part: &a2a.Part_Data{Data: &a2a.DataPart{Data: payload}}}},
	}
	bytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(msg)
	if err != nil {
		metrics.HandshakesFailed.WithLabelValues("marshal_error").Inc()
		return nil, fmt.Errorf("marshal for signing: %w", err)
	}
	meta, err := signStruct(c.key, bytes, did)
	if err != nil {
		metrics.HandshakesFailed.WithLabelValues("sign_error").Inc()
		return nil, fmt.Errorf("sign: %w", err)
	}
	resp, err := c.SendMessage(ctx, &a2a.SendMessageRequest{Request: msg, Metadata: meta})
	if err != nil {
		metrics.HandshakesFailed.WithLabelValues("send_error").Inc()
		return nil, err
	}
	metrics.HandshakesCompleted.WithLabelValues("success").Inc()
	return resp, nil
}
