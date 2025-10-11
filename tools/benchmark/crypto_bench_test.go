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

package benchmark

import (
	"crypto/rand"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/formats"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
)

// BenchmarkKeyGeneration benchmarks key pair generation
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

	b.Run("Secp256k1", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := keys.GenerateSecp256k1KeyPair()
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

// BenchmarkSigning benchmarks message signing
func BenchmarkSigning(b *testing.B) {
	message := make([]byte, 1024)
	_, _ = rand.Read(message)

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

// BenchmarkVerification benchmarks signature verification
func BenchmarkVerification(b *testing.B) {
	message := make([]byte, 1024)
	_, _ = rand.Read(message)

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

// BenchmarkKeyExport benchmarks key export to different formats
func BenchmarkKeyExport(b *testing.B) {
	keyPair, _ := keys.GenerateEd25519KeyPair()

	b.Run("JWK", func(b *testing.B) {
		exporter := formats.NewJWKExporter()
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := exporter.Export(keyPair, crypto.KeyFormatJWK)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("PEM", func(b *testing.B) {
		exporter := formats.NewPEMExporter()
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := exporter.Export(keyPair, crypto.KeyFormatPEM)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkKeyImport benchmarks key import from different formats
func BenchmarkKeyImport(b *testing.B) {
	keyPair, _ := keys.GenerateEd25519KeyPair()

	b.Run("JWK", func(b *testing.B) {
		exporter := formats.NewJWKExporter()
		jwkData, _ := exporter.Export(keyPair, crypto.KeyFormatJWK)

		importer := formats.NewJWKImporter()
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := importer.Import(jwkData, crypto.KeyFormatJWK)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("PEM", func(b *testing.B) {
		exporter := formats.NewPEMExporter()
		pemData, _ := exporter.Export(keyPair, crypto.KeyFormatPEM)

		importer := formats.NewPEMImporter()
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := importer.Import(pemData, crypto.KeyFormatPEM)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkMessageSizes benchmarks signing with different message sizes
func BenchmarkMessageSizes(b *testing.B) {
	keyPair, _ := keys.GenerateEd25519KeyPair()

	sizes := []int{64, 256, 1024, 4096, 16384, 65536}

	for _, size := range sizes {
		message := make([]byte, size)
		_, _ = rand.Read(message)

		b.Run(formatBytes(size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := keyPair.Sign(message)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// formatBytes moved to comparison_bench_test.go to avoid duplication
