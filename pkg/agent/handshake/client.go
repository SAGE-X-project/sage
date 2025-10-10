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
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sage-x-project/sage/internal/metrics"
	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/pkg/agent/transport"
)

type Client struct {
	transport transport.MessageTransport
	key       sagecrypto.KeyPair
}

func NewClient(t transport.MessageTransport, key sagecrypto.KeyPair) *Client {
	return &Client{
		transport: t,
		key:       key,
	}
}

// Invitation sends the initial invitation (clear JSON payload).
func (c *Client) Invitation(ctx context.Context, invMsg InvitationMessage, did string) (*transport.Response, error) {
	start := time.Now()
	metrics.HandshakesInitiated.WithLabelValues("client").Inc()
	defer func() {
		metrics.HandshakeDuration.WithLabelValues(Invitation.String()).Observe(
			time.Since(start).Seconds(),
		)
	}()

	payload, err := json.Marshal(invMsg)
	if err != nil {
		metrics.HandshakesFailed.WithLabelValues("marshal_error").Inc()
		return nil, fmt.Errorf("marshal invitation: %w", err)
	}

	signature, err := c.key.Sign(payload)
	if err != nil {
		metrics.HandshakesFailed.WithLabelValues("sign_error").Inc()
		return nil, fmt.Errorf("sign: %w", err)
	}

	msg := &transport.SecureMessage{
		ID:        uuid.NewString(),
		ContextID: invMsg.ContextID,
		TaskID:    GenerateTaskID(Invitation),
		Payload:   payload,
		DID:       did,
		Signature: signature,
		Role:      "user",
		Metadata:  make(map[string]string),
	}

	resp, err := c.transport.Send(ctx, msg)
	if err != nil {
		metrics.HandshakesFailed.WithLabelValues("send_error").Inc()
		return nil, err
	}
	metrics.HandshakesCompleted.WithLabelValues("success").Inc()
	return resp, nil
}

// Request encrypts RequestMessage for the peer using bootstrap envelope.
func (c *Client) Request(ctx context.Context, reqMsg RequestMessage, edPeerPub crypto.PublicKey, did string) (*transport.Response, error) {
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

	signature, err := c.key.Sign(packet)
	if err != nil {
		metrics.HandshakesFailed.WithLabelValues("sign_error").Inc()
		return nil, fmt.Errorf("sign: %w", err)
	}

	msg := &transport.SecureMessage{
		ID:        uuid.NewString(),
		ContextID: reqMsg.ContextID,
		TaskID:    GenerateTaskID(Request),
		Payload:   packet,
		DID:       did,
		Signature: signature,
		Role:      "user",
		Metadata:  make(map[string]string),
	}

	resp, err := c.transport.Send(ctx, msg)
	if err != nil {
		metrics.HandshakesFailed.WithLabelValues("send_error").Inc()
		return nil, err
	}
	metrics.HandshakesCompleted.WithLabelValues("success").Inc()
	return resp, nil
}

// Response is sent by the agent back to the initiator (bootstrap envelope).
func (c *Client) Response(ctx context.Context, resMsg ResponseMessage, edPeerPub crypto.PublicKey, did string) (*transport.Response, error) {
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

	signature, err := c.key.Sign(packet)
	if err != nil {
		metrics.HandshakesFailed.WithLabelValues("sign_error").Inc()
		return nil, fmt.Errorf("sign: %w", err)
	}

	msg := &transport.SecureMessage{
		ID:        uuid.NewString(),
		ContextID: resMsg.ContextID,
		TaskID:    GenerateTaskID(Response),
		Payload:   packet,
		DID:       did,
		Signature: signature,
		Role:      "agent",
		Metadata:  make(map[string]string),
	}

	resp, err := c.transport.Send(ctx, msg)
	if err != nil {
		metrics.HandshakesFailed.WithLabelValues("send_error").Inc()
		return nil, err
	}
	metrics.HandshakesCompleted.WithLabelValues("success").Inc()
	return resp, nil
}

// Complete notifies completion (clear JSON payload).
func (c *Client) Complete(ctx context.Context, compMsg CompleteMessage, did string) (*transport.Response, error) {
	start := time.Now()
	metrics.HandshakesInitiated.WithLabelValues("client").Inc()
	defer func() {
		metrics.HandshakeDuration.WithLabelValues(Complete.String()).Observe(
			time.Since(start).Seconds(),
		)
	}()

	payload, err := json.Marshal(compMsg)
	if err != nil {
		metrics.HandshakesFailed.WithLabelValues("marshal_error").Inc()
		return nil, fmt.Errorf("marshal complete: %w", err)
	}

	signature, err := c.key.Sign(payload)
	if err != nil {
		metrics.HandshakesFailed.WithLabelValues("sign_error").Inc()
		return nil, fmt.Errorf("sign: %w", err)
	}

	msg := &transport.SecureMessage{
		ID:        uuid.NewString(),
		ContextID: compMsg.ContextID,
		TaskID:    GenerateTaskID(Complete),
		Payload:   payload,
		DID:       did,
		Signature: signature,
		Role:      "user",
		Metadata:  make(map[string]string),
	}

	resp, err := c.transport.Send(ctx, msg)
	if err != nil {
		metrics.HandshakesFailed.WithLabelValues("send_error").Inc()
		return nil, err
	}
	metrics.HandshakesCompleted.WithLabelValues("success").Inc()
	return resp, nil
}
