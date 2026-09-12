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

package chain

import (
	"fmt"

	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	_ "github.com/sage-x-project/sage/pkg/agent/crypto/keys" // Import to register algorithms
)

// chainKeyMap maps chain types to the recommended key type.
var chainKeyMap = map[ChainType]sagecrypto.KeyType{
	ChainTypeEthereum: sagecrypto.KeyTypeSecp256k1, // Ethereum uses ECDSA (Secp256k1)
	ChainTypeSolana:   sagecrypto.KeyTypeEd25519,   // Solana uses Ed25519
	ChainTypeBitcoin:  sagecrypto.KeyTypeSecp256k1, // Bitcoin uses ECDSA (Secp256k1)
	ChainTypeCosmos:   sagecrypto.KeyTypeSecp256k1, // Cosmos typically uses Secp256k1
}

// chainSupportedKeys maps chain types to every supported key type.
var chainSupportedKeys = map[ChainType][]sagecrypto.KeyType{
	ChainTypeEthereum: {sagecrypto.KeyTypeSecp256k1},
	ChainTypeSolana:   {sagecrypto.KeyTypeEd25519},
	ChainTypeBitcoin:  {sagecrypto.KeyTypeSecp256k1},
	ChainTypeCosmos:   {sagecrypto.KeyTypeSecp256k1, sagecrypto.KeyTypeEd25519}, // Some Cosmos chains support Ed25519
}

// GetRecommendedKeyType returns the recommended key type for a given chain
func GetRecommendedKeyType(chainType ChainType) (sagecrypto.KeyType, error) {
	keyType, exists := chainKeyMap[chainType]
	if !exists {
		return "", fmt.Errorf("%w: %s", ErrChainNotSupported, chainType)
	}
	return keyType, nil
}

// GetSupportedKeyTypes returns all supported key types for a given chain
func GetSupportedKeyTypes(chainType ChainType) ([]sagecrypto.KeyType, error) {
	keyTypes, exists := chainSupportedKeys[chainType]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrChainNotSupported, chainType)
	}
	// Return a copy to prevent external modification
	result := make([]sagecrypto.KeyType, len(keyTypes))
	copy(result, keyTypes)
	return result, nil
}

// ValidateKeyTypeForChain validates if a key type is supported for a chain
func ValidateKeyTypeForChain(keyType sagecrypto.KeyType, chainType ChainType) error {
	supportedKeyTypes, err := GetSupportedKeyTypes(chainType)
	if err != nil {
		return err
	}
	for _, supported := range supportedKeyTypes {
		if keyType == supported {
			return nil
		}
	}
	return fmt.Errorf("key type %s is not supported for chain %s (supported: %v)",
		keyType, chainType, supportedKeyTypes)
}

// GetRFC9421Algorithm returns the RFC 9421 signature algorithm for a key type,
// read from the algorithm table in pkg/agent/crypto.
func GetRFC9421Algorithm(keyType sagecrypto.KeyType) (string, error) {
	return sagecrypto.GetRFC9421AlgorithmName(keyType)
}
