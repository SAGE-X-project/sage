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
	"crypto/ed25519"
	"testing"
	"time"

	_ "github.com/sage-x-project/sage/internal/cryptoinit" // Initialize crypto wrappers
	"github.com/sage-x-project/sage/pkg/agent/crypto"
)

// BenchmarkKeyGeneration measures key generation performance for different algorithms
func BenchmarkKeyGeneration(b *testing.B) {
	benchmarks := []struct {
		name string
		fn   func() error
	}{
		{
			name: "Ed25519KeyGeneration",
			fn: func() error {
				_, err := crypto.GenerateEd25519KeyPair()
				return err
			},
		},
		{
			name: "ECDSAKeyGeneration",
			fn: func() error {
				_, err := crypto.GenerateSecp256k1KeyPair()
				return err
			},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if err := bm.fn(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkProofOfPossession measures PoP signature generation and verification
func BenchmarkProofOfPossession(b *testing.B) {
	// Pre-generate keys for testing
	ed25519KeyPair, _ := crypto.GenerateEd25519KeyPair()
	ed25519PubKey := ed25519KeyPair.PublicKey().(ed25519.PublicKey)
	ed25519PrivKey := ed25519KeyPair.PrivateKey().(ed25519.PrivateKey)

	ecdsaKeyPair, _ := crypto.GenerateSecp256k1KeyPair()
	ecdsaPubKeyBytes, _ := MarshalPublicKey(ecdsaKeyPair.PublicKey())

	did := AgentDID("did:sage:ethereum:0x1234567890abcdef")

	b.Run("Ed25519_GeneratePoP", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := GenerateKeyProofOfPossession(did, ed25519PubKey, ed25519PrivKey, KeyTypeEd25519)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ECDSA_GeneratePoP", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := GenerateKeyProofOfPossession(did, ecdsaPubKeyBytes, ecdsaKeyPair.PrivateKey(), KeyTypeECDSA)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// Pre-generate signatures for verification benchmarks
	ed25519Sig, _ := GenerateKeyProofOfPossession(did, ed25519PubKey, ed25519PrivKey, KeyTypeEd25519)
	ed25519Key := &AgentKey{
		Type:      KeyTypeEd25519,
		KeyData:   ed25519PubKey,
		Signature: ed25519Sig,
		Verified:  true,
		CreatedAt: time.Now(),
	}

	ecdsaSig, _ := GenerateKeyProofOfPossession(did, ecdsaPubKeyBytes, ecdsaKeyPair.PrivateKey(), KeyTypeECDSA)
	ecdsaKey := &AgentKey{
		Type:      KeyTypeECDSA,
		KeyData:   ecdsaPubKeyBytes,
		Signature: ecdsaSig,
		Verified:  true,
		CreatedAt: time.Now(),
	}

	b.Run("Ed25519_VerifyPoP", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if err := VerifyKeyProofOfPossession(did, ed25519Key); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ECDSA_VerifyPoP", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if err := VerifyKeyProofOfPossession(did, ecdsaKey); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkA2ACardProof measures A2A card proof generation and verification
func BenchmarkA2ACardProof(b *testing.B) {
	// Pre-generate test metadata with Ed25519 key
	ed25519KeyPair, _ := crypto.GenerateEd25519KeyPair()
	ed25519PubKey := ed25519KeyPair.PublicKey().(ed25519.PublicKey)
	ed25519PrivKey := ed25519KeyPair.PrivateKey().(ed25519.PrivateKey)

	metadata := &AgentMetadataV4{
		DID:         "did:sage:ethereum:0x1234567890abcdef",
		Name:        "Benchmark Agent",
		Description: "Performance testing agent",
		Endpoint:    "https://benchmark.agent.com",
		Keys: []AgentKey{
			{
				Type:      KeyTypeEd25519,
				KeyData:   ed25519PubKey,
				Verified:  true,
				CreatedAt: time.Now(),
			},
		},
		Capabilities: map[string]interface{}{
			"benchmark": true,
		},
		Owner:     "0x1234567890abcdef",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	b.Run("GenerateA2ACardWithProof", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := GenerateA2ACardWithProof(metadata, ed25519PrivKey, KeyTypeEd25519)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// Pre-generate card for verification benchmark
	cardWithProof, _ := GenerateA2ACardWithProof(metadata, ed25519PrivKey, KeyTypeEd25519)

	b.Run("VerifyA2ACardProof", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			valid, err := VerifyA2ACardProof(cardWithProof)
			if err != nil || !valid {
				b.Fatal(err)
			}
		}
	})

	b.Run("ValidateA2ACardWithProof", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if err := ValidateA2ACardWithProof(cardWithProof); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkMultiKeyOperations measures performance with different key counts
func BenchmarkMultiKeyOperations(b *testing.B) {
	keyCounts := []int{1, 2, 5, 10}
	did := AgentDID("did:sage:ethereum:0x1234567890abcdef")

	for _, count := range keyCounts {
		// Generate keys
		keys := make([]AgentKey, count)
		for i := 0; i < count; i++ {
			keyPair, _ := crypto.GenerateEd25519KeyPair()
			pubKey := keyPair.PublicKey().(ed25519.PublicKey)
			privKey := keyPair.PrivateKey().(ed25519.PrivateKey)

			sig, _ := GenerateKeyProofOfPossession(did, pubKey, privKey, KeyTypeEd25519)
			keys[i] = AgentKey{
				Type:      KeyTypeEd25519,
				KeyData:   pubKey,
				Signature: sig,
				Verified:  true,
				CreatedAt: time.Now(),
			}
		}

		metadata := &AgentMetadataV4{
			DID:         did,
			Name:        "Multi-Key Agent",
			Endpoint:    "https://multikey.agent.com",
			Keys:        keys,
			Owner:       "0x1234567890abcdef",
			IsActive:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		b.Run("VerifyAllKeyProofs_"+string(rune(count))+"keys", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if err := VerifyAllKeyProofs(metadata); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkMetadataConversion measures conversion between formats
func BenchmarkMetadataConversion(b *testing.B) {
	// Create AgentMetadata
	metadata := &AgentMetadata{
		DID:         "did:sage:ethereum:0x1234567890abcdef",
		Name:        "Test Agent",
		Description: "Conversion benchmark",
		Endpoint:    "https://test.agent.com",
		PublicKey:   make([]byte, 33),
		Owner:       "0x1234567890abcdef",
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	b.Run("FromAgentMetadata", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = FromAgentMetadata(metadata)
		}
	})

	// Create AgentMetadataV4
	metadataV4 := &AgentMetadataV4{
		DID:      AgentDID("did:sage:ethereum:0x1234567890abcdef"),
		Name:     "Test Agent",
		Endpoint: "https://test.agent.com",
		Keys: []AgentKey{
			{
				Type:      KeyTypeECDSA,
				KeyData:   make([]byte, 33),
				Verified:  true,
				CreatedAt: time.Now(),
			},
		},
		Owner:     "0x1234567890abcdef",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	b.Run("ToAgentMetadata", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = metadataV4.ToAgentMetadata()
		}
	})
}

// BenchmarkDIDParsing measures DID string parsing performance
func BenchmarkDIDParsing(b *testing.B) {
	testCases := []struct {
		name string
		did  AgentDID
	}{
		{
			name: "SimpleEthereumDID",
			did:  "did:sage:ethereum:0x1234567890abcdef",
		},
		{
			name: "EthereumDIDWithNonce",
			did:  "did:sage:ethereum:0x1234567890abcdef:1",
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _, err := ParseDID(tc.did)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkA2ACardValidation measures A2A card validation without proof
func BenchmarkA2ACardValidation(b *testing.B) {
	// Create valid A2A card
	card := &A2AAgentCard{
		ID:          "did:sage:ethereum:0x1234567890abcdef",
		Name:        "Benchmark Agent",
		Description: "Performance testing",
		PublicKeys: []A2APublicKey{
			{
				ID:              "did:sage:ethereum:0x1234567890abcdef#key-1",
				Type:            "Ed25519VerificationKey2020",
				Controller:      "did:sage:ethereum:0x1234567890abcdef",
				PublicKeyBase58: "ATestBase58EncodedPublicKey1234567890",
			},
		},
		Endpoints: []A2AEndpoint{
			{
				Type: "grpc",
				URI:  "https://agent.example.com:443",
			},
		},
		Capabilities: []string{"chat", "search"},
	}

	b.Run("ValidateA2ACard", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if err := ValidateA2ACard(card); err != nil {
				b.Fatal(err)
			}
		}
	})

	// Benchmark with multiple keys
	cardMultiKey := &A2AAgentCard{
		ID:          "did:sage:ethereum:0x1234567890abcdef",
		Name:        "Multi-Key Agent",
		Description: "Multiple keys for benchmarking",
		PublicKeys: []A2APublicKey{
			{
				ID:              "did:sage:ethereum:0x1234567890abcdef#key-1",
				Type:            "Ed25519VerificationKey2020",
				Controller:      "did:sage:ethereum:0x1234567890abcdef",
				PublicKeyBase58: "Key1Base58Encoded1234567890",
			},
			{
				ID:              "did:sage:ethereum:0x1234567890abcdef#key-2",
				Type:            "EcdsaSecp256k1VerificationKey2019",
				Controller:      "did:sage:ethereum:0x1234567890abcdef",
				PublicKeyBase58: "Key2Base58Encoded1234567890",
			},
			{
				ID:              "did:sage:ethereum:0x1234567890abcdef#key-3",
				Type:            "X25519KeyAgreementKey2020",
				Controller:      "did:sage:ethereum:0x1234567890abcdef",
				PublicKeyBase58: "Key3Base58Encoded1234567890",
			},
		},
		Endpoints: []A2AEndpoint{
			{
				Type: "grpc",
				URI:  "https://agent.example.com:443",
			},
		},
		Capabilities: []string{"chat", "search", "encrypt"},
	}

	b.Run("ValidateA2ACard_MultiKey", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if err := ValidateA2ACard(cardMultiKey); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkKeyMarshalUnmarshal measures key serialization performance
func BenchmarkKeyMarshalUnmarshal(b *testing.B) {
	// Generate test keys
	ecdsaKeyPair, _ := crypto.GenerateSecp256k1KeyPair()
	ed25519KeyPair, _ := crypto.GenerateEd25519KeyPair()

	b.Run("MarshalPublicKey_ECDSA", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := MarshalPublicKey(ecdsaKeyPair.PublicKey())
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("MarshalPublicKey_Ed25519", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := MarshalPublicKey(ed25519KeyPair.PublicKey())
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
