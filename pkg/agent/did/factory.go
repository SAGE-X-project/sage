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

package did

import (
	"fmt"
	"sync"

	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/chain"
)

// ChainClient is what a chain package installs into a Manager: one object
// that both writes to and reads from the chain's registry.
type ChainClient interface {
	Registry
	Resolver
}

// ClientCreator builds a ChainClient from a registry configuration.
type ClientCreator func(config *RegistryConfig) (ChainClient, error)

var (
	clientCreators   = map[Chain]ClientCreator{}
	clientCreatorsMu sync.RWMutex
)

// RegisterClientCreator installs the creator Manager.Configure uses for
// chain. Chain packages call it from their Register function; the
// composition root (internal/app.RegisterDefaults) calls those. A nil
// creator removes the registration.
func RegisterClientCreator(chain Chain, creator ClientCreator) {
	clientCreatorsMu.Lock()
	defer clientCreatorsMu.Unlock()
	if creator == nil {
		delete(clientCreators, chain)
		return
	}
	clientCreators[chain] = creator
}

// ClientCreatorFor returns the creator registered for chain, or nil.
func ClientCreatorFor(chain Chain) ClientCreator {
	clientCreatorsMu.RLock()
	defer clientCreatorsMu.RUnlock()
	return clientCreators[chain]
}

// chainType maps a DID chain to the crypto chain table.
func chainType(chainName Chain) (chain.ChainType, error) {
	switch chainName {
	case ChainEthereum:
		return chain.ChainTypeEthereum, nil
	case ChainSolana:
		return chain.ChainTypeSolana, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrChainNotSupported, chainName)
	}
}

// GetRecommendedKeyType returns the recommended key type for a chain.
//
// Deprecated: use chain.GetRecommendedKeyType from pkg/agent/crypto/chain.
func GetRecommendedKeyType(chainName Chain) (sagecrypto.KeyType, error) {
	ct, err := chainType(chainName)
	if err != nil {
		return "", err
	}
	return chain.GetRecommendedKeyType(ct)
}

// ValidateKeyTypeForChain validates if a key type is compatible with a chain.
//
// Deprecated: use chain.ValidateKeyTypeForChain from pkg/agent/crypto/chain.
func ValidateKeyTypeForChain(keyType sagecrypto.KeyType, chainName Chain) error {
	ct, err := chainType(chainName)
	if err != nil {
		return err
	}
	return chain.ValidateKeyTypeForChain(keyType, ct)
}

// GetRFC9421Algorithm returns the RFC 9421 algorithm for a key type.
//
// Deprecated: use chain.GetRFC9421Algorithm from pkg/agent/crypto/chain.
func GetRFC9421Algorithm(keyType sagecrypto.KeyType) (string, error) {
	return chain.GetRFC9421Algorithm(keyType)
}
