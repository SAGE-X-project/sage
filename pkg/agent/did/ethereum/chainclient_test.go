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

package ethereum

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sage-x-project/sage/pkg/agent/did"
)

// Neither client needs a node for these checks: they fail before any call.
func TestAgentCardClient_ChainClientContract(t *testing.T) {
	c := &AgentCardClient{config: &did.RegistryConfig{}}
	ctx := context.Background()

	_, err := c.Register(ctx, &did.RegistrationRequest{DID: "did:sage:ethereum:0x1"})
	require.ErrorIs(t, err, ErrCommitRevealRequired)

	_, err = c.Search(ctx, did.SearchCriteria{Name: "x"})
	require.ErrorIs(t, err, ErrSearchUnsupported)

	_, err = c.ListAgentsByOwner(ctx, "not-an-address")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid Ethereum address")

	err = c.Update(ctx, "did:sage:ethereum:0x1", map[string]interface{}{"name": "new"}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not allow changing the name")
	err = c.Update(ctx, "did:sage:ethereum:0x1", map[string]interface{}{"description": "d"}, nil)
	require.Error(t, err)
	err = c.Update(ctx, "did:sage:ethereum:0x1", map[string]interface{}{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no updatable field")
}

// The wrapper promotes every AgentCardClient method, so it is a ChainClient
// through embedding and a nil inner client is the only way to get one
// without a node.
func TestEthereumClient_WrapsAgentCardClient(t *testing.T) {
	var _ did.ChainClient = (*EthereumClient)(nil)
	w := &EthereumClient{AgentCardClient: &AgentCardClient{config: &did.RegistryConfig{ContractAddress: "0x1"}}}
	assert.Equal(t, "0x1", w.config.ContractAddress)
	_, err := w.Register(context.Background(), &did.RegistrationRequest{})
	assert.True(t, errors.Is(err, ErrCommitRevealRequired))
}
