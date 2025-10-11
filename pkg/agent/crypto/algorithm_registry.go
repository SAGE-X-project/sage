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

package crypto

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"errors"
	"sync"
)

// AlgorithmInfo contains metadata about a cryptographic algorithm
type AlgorithmInfo struct {
	// KeyType is the internal key type identifier
	KeyType KeyType

	// Name is the human-readable name of the algorithm
	Name string

	// Description provides details about the algorithm
	Description string

	// RFC9421Algorithm is the algorithm name used in RFC 9421 HTTP Message Signatures
	// Empty string if the algorithm doesn't support RFC 9421
	RFC9421Algorithm string

	// SupportsRFC9421 indicates if this algorithm can be used for RFC 9421 signatures
	SupportsRFC9421 bool

	// SupportsKeyGeneration indicates if the system can generate keys for this algorithm
	SupportsKeyGeneration bool

	// SupportsSignature indicates if this algorithm supports digital signatures
	SupportsSignature bool

	// SupportsEncryption indicates if this algorithm supports encryption
	SupportsEncryption bool
}

// algorithmRegistry stores all registered algorithms
var (
	registry                 = make(map[KeyType]*AlgorithmInfo)
	rfc9421ToKeyType         = make(map[string]KeyType)
	registryMutex            sync.RWMutex
	ErrAlgorithmNotSupported = errors.New("algorithm not supported")
	ErrAlgorithmExists       = errors.New("algorithm already registered")
)

// RegisterAlgorithm registers a new algorithm in the registry
// This should be called during package initialization
func RegisterAlgorithm(info AlgorithmInfo) error {
	registryMutex.Lock()
	defer registryMutex.Unlock()

	if info.KeyType == "" {
		return errors.New("key type cannot be empty")
	}

	if _, exists := registry[info.KeyType]; exists {
		return ErrAlgorithmExists
	}

	// Validate RFC 9421 algorithm if claimed to be supported
	if info.SupportsRFC9421 && info.RFC9421Algorithm == "" {
		return errors.New("RFC9421Algorithm must be set if SupportsRFC9421 is true")
	}

	// Store in registry
	registry[info.KeyType] = &info

	// Also index by RFC 9421 algorithm name
	if info.SupportsRFC9421 && info.RFC9421Algorithm != "" {
		rfc9421ToKeyType[info.RFC9421Algorithm] = info.KeyType
	}

	return nil
}

// GetAlgorithmInfo returns information about a registered algorithm
func GetAlgorithmInfo(keyType KeyType) (*AlgorithmInfo, error) {
	registryMutex.RLock()
	defer registryMutex.RUnlock()

	info, exists := registry[keyType]
	if !exists {
		return nil, ErrAlgorithmNotSupported
	}

	// Return a copy to prevent external modification
	infoCopy := *info
	return &infoCopy, nil
}

// ListSupportedAlgorithms returns a list of all supported algorithms
func ListSupportedAlgorithms() []AlgorithmInfo {
	registryMutex.RLock()
	defer registryMutex.RUnlock()

	result := make([]AlgorithmInfo, 0, len(registry))
	for _, info := range registry {
		result = append(result, *info)
	}

	return result
}

// ListRFC9421SupportedAlgorithms returns a list of RFC 9421 algorithm names
func ListRFC9421SupportedAlgorithms() []string {
	registryMutex.RLock()
	defer registryMutex.RUnlock()

	result := make([]string, 0, len(rfc9421ToKeyType))
	for algName := range rfc9421ToKeyType {
		result = append(result, algName)
	}

	return result
}

// GetRFC9421AlgorithmName returns the RFC 9421 algorithm name for a key type
func GetRFC9421AlgorithmName(keyType KeyType) (string, error) {
	info, err := GetAlgorithmInfo(keyType)
	if err != nil {
		return "", err
	}

	if !info.SupportsRFC9421 {
		return "", errors.New("algorithm does not support RFC 9421")
	}

	return info.RFC9421Algorithm, nil
}

// GetKeyTypeFromRFC9421Algorithm returns the key type for an RFC 9421 algorithm name
func GetKeyTypeFromRFC9421Algorithm(rfc9421Algorithm string) (KeyType, error) {
	registryMutex.RLock()
	defer registryMutex.RUnlock()

	keyType, exists := rfc9421ToKeyType[rfc9421Algorithm]
	if !exists {
		return "", ErrAlgorithmNotSupported
	}

	return keyType, nil
}

// SupportsRFC9421 checks if an algorithm supports RFC 9421
func SupportsRFC9421(keyType KeyType) bool {
	info, err := GetAlgorithmInfo(keyType)
	if err != nil {
		return false
	}

	return info.SupportsRFC9421
}

// SupportsKeyGeneration checks if an algorithm supports key generation
func SupportsKeyGeneration(keyType KeyType) bool {
	info, err := GetAlgorithmInfo(keyType)
	if err != nil {
		return false
	}

	return info.SupportsKeyGeneration
}

// SupportsSignature checks if an algorithm supports digital signatures
func SupportsSignature(keyType KeyType) bool {
	info, err := GetAlgorithmInfo(keyType)
	if err != nil {
		return false
	}

	return info.SupportsSignature
}

// IsAlgorithmSupported checks if an algorithm is registered
func IsAlgorithmSupported(keyType KeyType) bool {
	_, err := GetAlgorithmInfo(keyType)
	return err == nil
}

// GetKeyTypeFromPublicKey maps a Go crypto.PublicKey to our KeyType
// This is used for algorithm validation in signature verification
func GetKeyTypeFromPublicKey(publicKey interface{}) (KeyType, error) {
	switch key := publicKey.(type) {
	case ed25519.PublicKey:
		return KeyTypeEd25519, nil
	case *ecdsa.PublicKey:
		return KeyTypeSecp256k1, nil
	case *rsa.PublicKey:
		return KeyTypeRSA, nil
	default:
		_ = key // Avoid unused variable error
		return "", errors.New("unsupported public key type")
	}
}

// ValidateAlgorithmForPublicKey validates that an RFC 9421 algorithm is compatible with a public key
// Returns nil if valid, error otherwise
func ValidateAlgorithmForPublicKey(publicKey interface{}, algorithm string) error {
	// Empty algorithm is allowed - will be inferred from key type
	if algorithm == "" {
		return nil
	}

	// Check if algorithm is supported in registry
	keyType, err := GetKeyTypeFromRFC9421Algorithm(algorithm)
	if err != nil {
		return err
	}

	// Get the expected key type from the public key
	expectedKeyType, err := GetKeyTypeFromPublicKey(publicKey)
	if err != nil {
		return err
	}

	// Validate they match
	if keyType != expectedKeyType {
		expectedAlg, _ := GetRFC9421AlgorithmName(expectedKeyType)
		return errors.New("algorithm mismatch: key type is " + string(expectedKeyType) +
			" (expects " + expectedAlg + ") but algorithm is " + algorithm)
	}

	return nil
}
