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

	sagecrypto "github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/chain"
)

// ClientFactory creates DID clients for different blockchain types
type ClientFactory interface {
	// CreateClient creates a DID client for the specified chain
	CreateClient(config *RegistryConfig) (Client, error)

	// GetRecommendedKeyType returns the recommended key type for a chain
	GetRecommendedKeyType(chainType Chain) (sagecrypto.KeyType, error)

	// ValidateKeyTypeForChain validates if a key type is compatible with a chain
	ValidateKeyTypeForChain(keyType sagecrypto.KeyType, chainType Chain) error

	// GetRFC9421Algorithm returns the RFC 9421 algorithm for a key type
	GetRFC9421Algorithm(keyType sagecrypto.KeyType) (string, error)
}

// defaultClientFactory is the default implementation of ClientFactory
type defaultClientFactory struct {
	keyMapper chain.ChainKeyTypeMapper
}

// NewClientFactory creates a new DID client factory
func NewClientFactory() ClientFactory {
	return &defaultClientFactory{
		keyMapper: chain.NewChainKeyTypeMapper(),
	}
}

// CreateClient creates a DID client for the specified chain
func (f *defaultClientFactory) CreateClient(config *RegistryConfig) (Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	switch config.Chain {
	case ChainEthereum:
		// Import here to avoid circular dependencies
		// In actual use, this would be in a separate package
		return createEthereumClient(config)

	case ChainSolana:
		// Import here to avoid circular dependencies
		return createSolanaClient(config)

	default:
		return nil, fmt.Errorf("%w: %s", ErrChainNotSupported, config.Chain)
	}
}

// GetRecommendedKeyType returns the recommended key type for a chain
func (f *defaultClientFactory) GetRecommendedKeyType(chainType Chain) (sagecrypto.KeyType, error) {
	// Convert did.Chain to chain.ChainType
	var cryptoChainType chain.ChainType
	switch chainType {
	case ChainEthereum:
		cryptoChainType = chain.ChainTypeEthereum
	case ChainSolana:
		cryptoChainType = chain.ChainTypeSolana
	default:
		return "", fmt.Errorf("%w: %s", ErrChainNotSupported, chainType)
	}

	return f.keyMapper.GetRecommendedKeyType(cryptoChainType)
}

// ValidateKeyTypeForChain validates if a key type is compatible with a chain
func (f *defaultClientFactory) ValidateKeyTypeForChain(keyType sagecrypto.KeyType, chainType Chain) error {
	// Convert did.Chain to chain.ChainType
	var cryptoChainType chain.ChainType
	switch chainType {
	case ChainEthereum:
		cryptoChainType = chain.ChainTypeEthereum
	case ChainSolana:
		cryptoChainType = chain.ChainTypeSolana
	default:
		return fmt.Errorf("%w: %s", ErrChainNotSupported, chainType)
	}

	return f.keyMapper.ValidateKeyTypeForChain(keyType, cryptoChainType)
}

// GetRFC9421Algorithm returns the RFC 9421 algorithm for a key type
func (f *defaultClientFactory) GetRFC9421Algorithm(keyType sagecrypto.KeyType) (string, error) {
	return f.keyMapper.GetRFC9421Algorithm(keyType)
}

// Global default factory instance for convenience
var defaultFactory = NewClientFactory()

// CreateClient is a convenience function using the default factory
func CreateClient(config *RegistryConfig) (Client, error) {
	return defaultFactory.CreateClient(config)
}

// GetRecommendedKeyType is a convenience function using the default factory
func GetRecommendedKeyType(chainType Chain) (sagecrypto.KeyType, error) {
	return defaultFactory.GetRecommendedKeyType(chainType)
}

// ValidateKeyTypeForChain is a convenience function using the default factory
func ValidateKeyTypeForChain(keyType sagecrypto.KeyType, chainType Chain) error {
	return defaultFactory.ValidateKeyTypeForChain(keyType, chainType)
}

// GetRFC9421Algorithm is a convenience function using the default factory
func GetRFC9421Algorithm(keyType sagecrypto.KeyType) (string, error) {
	return defaultFactory.GetRFC9421Algorithm(keyType)
}

// Client creation functions - these will be implemented by importing the specific packages
// We use function variables to avoid import cycles

var (
	createEthereumClient func(*RegistryConfig) (Client, error)
	createSolanaClient   func(*RegistryConfig) (Client, error)
)

// RegisterEthereumClientCreator registers the Ethereum client creator
// This should be called during initialization by the ethereum package
func RegisterEthereumClientCreator(creator func(*RegistryConfig) (Client, error)) {
	createEthereumClient = creator
}

// RegisterSolanaClientCreator registers the Solana client creator
// This should be called during initialization by the solana package
func RegisterSolanaClientCreator(creator func(*RegistryConfig) (Client, error)) {
	createSolanaClient = creator
}
