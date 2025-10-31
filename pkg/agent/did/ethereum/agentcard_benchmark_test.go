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
	"crypto/rand"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// BenchmarkCommitmentHashCalculation measures the performance of commitment hash computation
// This is critical as it must match Solidity's abi.encode() exactly
func BenchmarkCommitmentHashCalculation(b *testing.B) {
	// Setup test data
	key, err := crypto.GenerateKey()
	if err != nil {
		b.Fatal(err)
	}

	config := &did.RegistryConfig{
		RPCEndpoint:     "http://localhost:8545",
		ContractAddress: "0x1234567890123456789012345678901234567890",
		PrivateKey:      common.Bytes2Hex(crypto.FromECDSA(key)),
	}

	client, err := NewAgentCardClient(config)
	if err != nil {
		b.Fatal(err)
	}

	params := &did.RegistrationParams{
		DID:          "did:sage:ethereum:benchmark-test",
		Name:         "Benchmark Agent",
		Description:  "Performance testing agent",
		Endpoint:     "https://benchmark.example.com",
		Capabilities: `{"features":["benchmark"]}`,
		Keys:         [][]byte{make([]byte, 65)},
		KeyTypes:     []did.KeyType{did.KeyTypeECDSA},
		Signatures:   [][]byte{make([]byte, 65)},
	}

	if _, err := rand.Read(params.Salt[:]); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := client.computeCommitmentHash(params)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCommitmentHashWithMultipleKeys tests performance with varying key counts
func BenchmarkCommitmentHashWithMultipleKeys(b *testing.B) {
	key, err := crypto.GenerateKey()
	if err != nil {
		b.Fatal(err)
	}

	config := &did.RegistryConfig{
		RPCEndpoint:     "http://localhost:8545",
		ContractAddress: "0x1234567890123456789012345678901234567890",
		PrivateKey:      common.Bytes2Hex(crypto.FromECDSA(key)),
	}

	client, err := NewAgentCardClient(config)
	if err != nil {
		b.Fatal(err)
	}

	// Test with 1, 3, 5, and 10 keys
	keyCounts := []int{1, 3, 5, 10}

	for _, keyCount := range keyCounts {
		b.Run(fmt.Sprintf("%d_keys", keyCount), func(b *testing.B) {
			params := &did.RegistrationParams{
				DID:        "did:sage:ethereum:multi-key-test",
				Name:       "Multi-Key Agent",
				Endpoint:   "https://multi.example.com",
				Keys:       make([][]byte, keyCount),
				KeyTypes:   make([]did.KeyType, keyCount),
				Signatures: make([][]byte, keyCount),
			}

			for i := 0; i < keyCount; i++ {
				params.Keys[i] = make([]byte, 65)
				params.KeyTypes[i] = did.KeyTypeECDSA
				params.Signatures[i] = make([]byte, 65)
			}

			if _, err := rand.Read(params.Salt[:]); err != nil {
				b.Fatal(err)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := client.computeCommitmentHash(params)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkAgentIDComputation measures agent ID derivation from DID
func BenchmarkAgentIDComputation(b *testing.B) {
	key, err := crypto.GenerateKey()
	if err != nil {
		b.Fatal(err)
	}

	config := &did.RegistryConfig{
		RPCEndpoint:     "http://localhost:8545",
		ContractAddress: "0x1234567890123456789012345678901234567890",
		PrivateKey:      common.Bytes2Hex(crypto.FromECDSA(key)),
	}

	client, err := NewAgentCardClient(config)
	if err != nil {
		b.Fatal(err)
	}

	testDID := "did:sage:ethereum:benchmark-agent-id"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = client.computeAgentID(testDID)
	}
}

// BenchmarkParameterConversion measures the cost of converting Go params to Solidity format
func BenchmarkParameterConversion(b *testing.B) {
	key, err := crypto.GenerateKey()
	if err != nil {
		b.Fatal(err)
	}

	config := &did.RegistryConfig{
		RPCEndpoint:     "http://localhost:8545",
		ContractAddress: "0x1234567890123456789012345678901234567890",
		PrivateKey:      common.Bytes2Hex(crypto.FromECDSA(key)),
	}

	client, err := NewAgentCardClient(config)
	if err != nil {
		b.Fatal(err)
	}

	params := &did.RegistrationParams{
		DID:          "did:sage:ethereum:param-conversion",
		Name:         "Conversion Test",
		Description:  "Testing parameter conversion",
		Endpoint:     "https://conversion.example.com",
		Capabilities: `{"test":true}`,
		Keys:         [][]byte{make([]byte, 65), make([]byte, 32), make([]byte, 32)},
		KeyTypes:     []did.KeyType{did.KeyTypeECDSA, did.KeyTypeEd25519, did.KeyTypeX25519},
		Signatures:   [][]byte{make([]byte, 65), make([]byte, 64), make([]byte, 64)},
	}

	if _, err := rand.Read(params.Salt[:]); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := client.toContractParams(params)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkKeyTypeConversion measures enum conversion performance
func BenchmarkKeyTypeConversion(b *testing.B) {
	keyTypes := []did.KeyType{
		did.KeyTypeECDSA,
		did.KeyTypeEd25519,
		did.KeyTypeX25519,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, kt := range keyTypes {
			// Convert to uint8 (simulating Solidity conversion)
			_ = uint8(kt)
		}
	}
}

// BenchmarkCommitmentStatusSerialization measures JSON serialization performance
func BenchmarkCommitmentStatusSerialization(b *testing.B) {
	status := &did.CommitmentStatus{
		Phase:      did.PhaseCommitted,
		CommitHash: [32]byte{1, 2, 3, 4, 5},
		Params: &did.RegistrationParams{
			DID:        "did:sage:ethereum:serialize-test",
			Name:       "Serialization Test",
			Endpoint:   "https://test.com",
			Keys:       [][]byte{make([]byte, 65)},
			KeyTypes:   []did.KeyType{did.KeyTypeECDSA},
			Signatures: [][]byte{make([]byte, 65)},
		},
	}

	b.Run("Marshal", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(status)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	data, err := json.Marshal(status)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("Unmarshal", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var s did.CommitmentStatus
			if err := json.Unmarshal(data, &s); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkMemoryAllocation measures memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("RegistrationParams", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			params := &did.RegistrationParams{
				DID:        "did:sage:ethereum:alloc-test",
				Name:       "Allocation Test",
				Endpoint:   "https://alloc.example.com",
				Keys:       make([][]byte, 3),
				KeyTypes:   make([]did.KeyType, 3),
				Signatures: make([][]byte, 3),
			}
			_ = params
		}
	})

	b.Run("CommitmentStatus", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			status := &did.CommitmentStatus{
				Phase: did.PhaseCommitted,
				Params: &did.RegistrationParams{
					Keys:       make([][]byte, 3),
					KeyTypes:   make([]did.KeyType, 3),
					Signatures: make([][]byte, 3),
				},
			}
			_ = status
		}
	})
}

// BenchmarkConcurrentHashCalculation tests concurrent performance
func BenchmarkConcurrentHashCalculation(b *testing.B) {
	key, err := crypto.GenerateKey()
	if err != nil {
		b.Fatal(err)
	}

	config := &did.RegistryConfig{
		RPCEndpoint:     "http://localhost:8545",
		ContractAddress: "0x1234567890123456789012345678901234567890",
		PrivateKey:      common.Bytes2Hex(crypto.FromECDSA(key)),
	}

	client, err := NewAgentCardClient(config)
	if err != nil {
		b.Fatal(err)
	}

	params := &did.RegistrationParams{
		DID:        "did:sage:ethereum:concurrent-test",
		Name:       "Concurrent Test",
		Endpoint:   "https://concurrent.example.com",
		Keys:       [][]byte{make([]byte, 65)},
		KeyTypes:   []did.KeyType{did.KeyTypeECDSA},
		Signatures: [][]byte{make([]byte, 65)},
	}

	if _, err := rand.Read(params.Salt[:]); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := client.computeCommitmentHash(params)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
