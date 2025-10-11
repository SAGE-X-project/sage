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

package handshake_test

import (
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
)

// BenchmarkKeyGeneration benchmarks key pair generation for handshake
func BenchmarkKeyGeneration(b *testing.B) {
	b.Run("Ed25519", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := keys.GenerateEd25519KeyPair()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("X25519", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := keys.GenerateX25519KeyPair()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkSignatureGeneration benchmarks signature generation
func BenchmarkSignatureGeneration(b *testing.B) {
	message := []byte("test message for signing benchmark")

	b.Run("Ed25519", func(b *testing.B) {
		keyPair, _ := keys.GenerateEd25519KeyPair()
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := keyPair.Sign(message)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Secp256k1", func(b *testing.B) {
		keyPair, _ := keys.GenerateSecp256k1KeyPair()
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := keyPair.Sign(message)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkSignatureVerification benchmarks signature verification
func BenchmarkSignatureVerification(b *testing.B) {
	message := []byte("test message for verification benchmark")

	b.Run("Ed25519", func(b *testing.B) {
		keyPair, _ := keys.GenerateEd25519KeyPair()
		signature, _ := keyPair.Sign(message)
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			err := keyPair.Verify(message, signature)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Secp256k1", func(b *testing.B) {
		keyPair, _ := keys.GenerateSecp256k1KeyPair()
		signature, _ := keyPair.Sign(message)
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			err := keyPair.Verify(message, signature)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// NOTE: Session encryption/decryption benchmarks are available in tools/benchmark/session_bench_test.go
// These have been removed from here to avoid duplication and session expiration issues.
