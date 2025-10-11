package benchmark

import (
	"crypto/rand"
	"net/http"
	"testing"

	"github.com/sage-x-project/sage/core/rfc9421"
	"github.com/sage-x-project/sage/crypto"
)

// BenchmarkHTTPSignature benchmarks RFC 9421 HTTP message signature operations
func BenchmarkHTTPSignature(b *testing.B) {
	keyPair, _ := crypto.GenerateKeyPair(crypto.KeyTypeEd25519)

	// Create a sample HTTP request
	req, _ := http.NewRequest("POST", "https://example.com/api/messages", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Date", "Tue, 07 Jun 2022 20:51:35 GMT")

	b.Run("Sign", func(b *testing.B) {
		signer := rfc9421.NewSigner(keyPair)
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			err := signer.SignRequest(req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Verify", func(b *testing.B) {
		signer := rfc9421.NewSigner(keyPair)
		signer.SignRequest(req)

		verifier := rfc9421.NewVerifier()
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			err := verifier.VerifyRequest(req, keyPair.PublicKey())
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkSignatureComponents benchmarks different signature components
func BenchmarkSignatureComponents(b *testing.B) {
	keyPair, _ := crypto.GenerateKeyPair(crypto.KeyTypeEd25519)

	req, _ := http.NewRequest("POST", "https://example.com/api/messages", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Date", "Tue, 07 Jun 2022 20:51:35 GMT")
	req.Header.Set("Authorization", "Bearer token123")

	components := [][]string{
		{"@method", "@target-uri", "@authority"},
		{"@method", "@target-uri", "@authority", "content-type", "date"},
		{"@method", "@target-uri", "@authority", "content-type", "date", "authorization"},
	}

	for i, comps := range components {
		b.Run(formatComponents(i+1, len(comps)), func(b *testing.B) {
			signer := rfc9421.NewSignerWithComponents(keyPair, comps)
			b.ReportAllocs()
			b.ResetTimer()

			for j := 0; j < b.N; j++ {
				err := signer.SignRequest(req)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkHMACSignature benchmarks HMAC-based signatures
func BenchmarkHMACSignature(b *testing.B) {
	secret := make([]byte, 32)
	rand.Read(secret)

	req, _ := http.NewRequest("POST", "https://example.com/api/messages", nil)
	req.Header.Set("Content-Type", "application/json")

	b.Run("Sign_HMAC", func(b *testing.B) {
		signer := rfc9421.NewHMACSigner(secret)
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			err := signer.SignRequest(req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Verify_HMAC", func(b *testing.B) {
		signer := rfc9421.NewHMACSigner(secret)
		signer.SignRequest(req)

		verifier := rfc9421.NewHMACVerifier(secret)
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			err := verifier.VerifyRequest(req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkPayloadSizes benchmarks signing with different payload sizes
func BenchmarkPayloadSizes(b *testing.B) {
	keyPair, _ := crypto.GenerateKeyPair(crypto.KeyTypeEd25519)
	signer := rfc9421.NewSigner(keyPair)

	sizes := []int{0, 1024, 10240, 102400, 1048576}

	for _, size := range sizes {
		payload := make([]byte, size)
		if size > 0 {
			rand.Read(payload)
		}

		b.Run(formatBytes(size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				req, _ := http.NewRequest("POST", "https://example.com/api", nil)
				req.ContentLength = int64(size)
				req.Header.Set("Content-Type", "application/json")

				err := signer.SignRequest(req)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkConcurrentSignature benchmarks concurrent signature operations
func BenchmarkConcurrentSignature(b *testing.B) {
	keyPair, _ := crypto.GenerateKeyPair(crypto.KeyTypeEd25519)

	b.Run("Parallel_Sign", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			signer := rfc9421.NewSigner(keyPair)
			for pb.Next() {
				req, _ := http.NewRequest("POST", "https://example.com/api", nil)
				req.Header.Set("Content-Type", "application/json")

				err := signer.SignRequest(req)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})

	b.Run("Parallel_Verify", func(b *testing.B) {
		signer := rfc9421.NewSigner(keyPair)
		req, _ := http.NewRequest("POST", "https://example.com/api", nil)
		req.Header.Set("Content-Type", "application/json")
		signer.SignRequest(req)

		b.ResetTimer()
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			verifier := rfc9421.NewVerifier()
			for pb.Next() {
				err := verifier.VerifyRequest(req, keyPair.PublicKey())
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})
}

func formatComponents(index, count int) string {
	return string(rune('0'+index)) + "_components_" + string(rune('0'+count))
}
