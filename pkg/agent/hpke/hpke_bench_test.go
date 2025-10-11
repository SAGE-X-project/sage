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

package hpke

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
)

// BenchmarkHPKEDeriveSharedSecret benchmarks HPKE sender-side derivation
func BenchmarkHPKEDeriveSharedSecret(b *testing.B) {
	recipientKP, _ := keys.GenerateX25519KeyPair()
	recipientPubKey := recipientKP.PublicKey()

	info := []byte("sage/hpke-handshake v1|ctx:test|init:alice|resp:bob")
	exportCtx := []byte("sage/session exporter v1")

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		enc, exporter, err := keys.HPKEDeriveSharedSecretToPeer(recipientPubKey, info, exportCtx, 32)
		if err != nil {
			b.Fatal(err)
		}
		_ = enc
		_ = exporter
	}
}

// BenchmarkHPKEOpenSharedSecret benchmarks HPKE receiver-side derivation
func BenchmarkHPKEOpenSharedSecret(b *testing.B) {
	recipientKP, _ := keys.GenerateX25519KeyPair()
	recipientPubKey := recipientKP.PublicKey()
	recipientPrivKey := recipientKP.PrivateKey()

	info := []byte("sage/hpke-handshake v1|ctx:test|init:alice|resp:bob")
	exportCtx := []byte("sage/session exporter v1")

	// Generate sender enc
	enc, _, _ := keys.HPKEDeriveSharedSecretToPeer(recipientPubKey, info, exportCtx, 32)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		exporter, err := keys.HPKEOpenSharedSecretWithPriv(recipientPrivKey, enc, info, exportCtx, 32)
		if err != nil {
			b.Fatal(err)
		}
		_ = exporter
	}
}

// BenchmarkHPKEFullRoundtrip benchmarks a complete HPKE roundtrip
func BenchmarkHPKEFullRoundtrip(b *testing.B) {
	recipientKP, _ := keys.GenerateX25519KeyPair()
	recipientPubKey := recipientKP.PublicKey()
	recipientPrivKey := recipientKP.PrivateKey()

	info := []byte("sage/hpke-handshake v1|ctx:test|init:alice|resp:bob")
	exportCtx := []byte("sage/session exporter v1")

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Sender side
		enc, exporterA, err := keys.HPKEDeriveSharedSecretToPeer(recipientPubKey, info, exportCtx, 32)
		if err != nil {
			b.Fatal(err)
		}

		// Receiver side
		exporterB, err := keys.HPKEOpenSharedSecretWithPriv(recipientPrivKey, enc, info, exportCtx, 32)
		if err != nil {
			b.Fatal(err)
		}

		// Verify they match
		if len(exporterA) != len(exporterB) {
			b.Fatal("exporter mismatch")
		}
	}
}

// BenchmarkHPKEExportLengths benchmarks different export lengths
func BenchmarkHPKEExportLengths(b *testing.B) {
	recipientKP, _ := keys.GenerateX25519KeyPair()
	recipientPubKey := recipientKP.PublicKey()

	info := []byte("sage/hpke-handshake v1|ctx:test|init:alice|resp:bob")
	exportCtx := []byte("sage/session exporter v1")

	exportLengths := []int{16, 32, 64, 128, 256}

	for _, length := range exportLengths {
		b.Run(fmt.Sprintf("Export_%dB", length), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, exporter, err := keys.HPKEDeriveSharedSecretToPeer(recipientPubKey, info, exportCtx, length)
				if err != nil {
					b.Fatal(err)
				}
				_ = exporter
			}
		})
	}
}

// BenchmarkHPKEInfoSizes benchmarks different info sizes
func BenchmarkHPKEInfoSizes(b *testing.B) {
	recipientKP, _ := keys.GenerateX25519KeyPair()
	recipientPubKey := recipientKP.PublicKey()

	exportCtx := []byte("sage/session exporter v1")

	infoSizes := []int{32, 64, 128, 256, 512}

	for _, size := range infoSizes {
		info := make([]byte, size)
		rand.Read(info)

		b.Run(fmt.Sprintf("Info_%dB", size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				enc, exporter, err := keys.HPKEDeriveSharedSecretToPeer(recipientPubKey, info, exportCtx, 32)
				if err != nil {
					b.Fatal(err)
				}
				_ = enc
				_ = exporter
			}
		})
	}
}

// BenchmarkX25519KeyGeneration benchmarks X25519 key pair generation
func BenchmarkX25519KeyGeneration(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := keys.GenerateX25519KeyPair()
		if err != nil {
			b.Fatal(err)
		}
	}
}
