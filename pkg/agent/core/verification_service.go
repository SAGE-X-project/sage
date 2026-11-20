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

package core

import (
	"context"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/core/rfc9421"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// DIDResolver interface for resolving agent information
type DIDResolver interface {
	ResolveAgent(ctx context.Context, agentDID did.AgentDID) (*did.AgentMetadata, error)
	ResolvePublicKey(ctx context.Context, agentDID did.AgentDID) (interface{}, error)
}

// VerificationService provides signature verification with DID integration.
//
// Deprecated: every method refuses (W-010); the type is removed with the
// legacy draft.1 code (W-079).
type VerificationService struct {
	didResolver DIDResolver
	verifier    *rfc9421.Verifier
}

// NewVerificationService creates a new verification service
func NewVerificationService(didResolver DIDResolver) *VerificationService {
	return &VerificationService{
		didResolver: didResolver,
		verifier:    rfc9421.NewVerifier(),
	}
}

// VerifyAgentMessage always reports an invalid result and returns
// rfc9421.ErrMessageFormatRemoved, without resolving anything.
//
// Deprecated: the SAGE Message format is not part of SAGE 0.9.0-draft.1.1
// (errata E03-19) and its verifier was forgeable (security review C1);
// authenticate agents with RFC 9421 HTTP message signatures.
func (s *VerificationService) VerifyAgentMessage(
	ctx context.Context,
	message *rfc9421.Message,
	opts *rfc9421.VerificationOptions,
) (*VerificationResult, error) {
	return refused(), rfc9421.ErrMessageFormatRemoved
}

// VerifyMessageFromHeaders always reports an invalid result and returns
// rfc9421.ErrMessageFormatRemoved.
//
// Deprecated: see VerifyAgentMessage.
func (s *VerificationService) VerifyMessageFromHeaders(
	ctx context.Context,
	headers map[string]string,
	body []byte,
	signature []byte,
) (*VerificationResult, error) {
	return refused(), rfc9421.ErrMessageFormatRemoved
}

// QuickVerify always returns rfc9421.ErrMessageFormatRemoved. It verified
// arbitrary bytes, with no freshness check, under an algorithm chosen from
// the DID's chain, which E01-11 forbids.
//
// Deprecated: see VerifyAgentMessage.
func (s *VerificationService) QuickVerify(
	ctx context.Context,
	agentDID string,
	message []byte,
	signature []byte,
) error {
	return rfc9421.ErrMessageFormatRemoved
}

func refused() *VerificationResult {
	return &VerificationResult{Valid: false, Error: rfc9421.ErrMessageFormatRemoved.Error(), VerifiedAt: time.Now()}
}

// VerificationResult contains the result of agent message verification
type VerificationResult struct {
	Valid        bool                   `json:"valid"`
	Error        string                 `json:"error,omitempty"`
	AgentID      string                 `json:"agent_id"`
	AgentName    string                 `json:"agent_name,omitempty"`
	AgentOwner   string                 `json:"agent_owner,omitempty"`
	Capabilities map[string]interface{} `json:"capabilities,omitempty"`
	VerifiedAt   time.Time              `json:"verified_at"`
}
