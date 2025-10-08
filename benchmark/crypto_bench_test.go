package benchmark

import (
	"crypto/rand"
	"testing"

	"github.com/sage-x-project/sage/crypto"
)

// BenchmarkKeyGeneration benchmarks key pair generation
func BenchmarkKeyGeneration(b *testing.B) {
	b.Run("Ed25519", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := crypto.GenerateKeyPair(crypto.KeyTypeEd25519)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Secp256k1", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := crypto.GenerateKeyPair(crypto.KeyTypeSecp256k1)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("X25519", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := crypto.GenerateKeyPair(crypto.KeyTypeX25519)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkSigning benchmarks message signing
func BenchmarkSigning(b *testing.B) {
	message := make([]byte, 1024)
	rand.Read(message)

	b.Run("Ed25519", func(b *testing.B) {
		keyPair, _ := crypto.GenerateKeyPair(crypto.KeyTypeEd25519)
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
		keyPair, _ := crypto.GenerateKeyPair(crypto.KeyTypeSecp256k1)
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
	rand.Read(message)

	b.Run("Ed25519", func(b *testing.B) {
		keyPair, _ := crypto.GenerateKeyPair(crypto.KeyTypeEd25519)
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
		keyPair, _ := crypto.GenerateKeyPair(crypto.KeyTypeSecp256k1)
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
	keyPair, _ := crypto.GenerateKeyPair(crypto.KeyTypeEd25519)

	b.Run("JWK", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := keyPair.ExportJWK()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("PEM", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := keyPair.ExportPEM()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkKeyImport benchmarks key import from different formats
func BenchmarkKeyImport(b *testing.B) {
	keyPair, _ := crypto.GenerateKeyPair(crypto.KeyTypeEd25519)

	b.Run("JWK", func(b *testing.B) {
		jwk, _ := keyPair.ExportJWK()
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := crypto.ImportJWK(jwk)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("PEM", func(b *testing.B) {
		pem, _ := keyPair.ExportPEM()
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := crypto.ImportPEM(pem)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkMessageSizes benchmarks signing with different message sizes
func BenchmarkMessageSizes(b *testing.B) {
	keyPair, _ := crypto.GenerateKeyPair(crypto.KeyTypeEd25519)

	sizes := []int{64, 256, 1024, 4096, 16384, 65536}

	for _, size := range sizes {
		message := make([]byte, size)
		rand.Read(message)

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

func formatBytes(size int) string {
	if size < 1024 {
		return string(rune(size)) + "B"
	} else if size < 1024*1024 {
		return string(rune(size/1024)) + "KB"
	}
	return string(rune(size/(1024*1024))) + "MB"
}
