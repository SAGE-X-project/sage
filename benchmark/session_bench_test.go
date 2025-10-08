package benchmark

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/sage-x-project/sage/session"
)

// BenchmarkSessionCreation benchmarks session creation
func BenchmarkSessionCreation(b *testing.B) {
	manager := session.NewManager()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sharedSecret := make([]byte, 32)
		rand.Read(sharedSecret)
		sessionID := fmt.Sprintf("bench-session-%d", i)

		_, err := manager.CreateSession(sessionID, sharedSecret)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSessionEncryption benchmarks message encryption in sessions
func BenchmarkSessionEncryption(b *testing.B) {
	manager := session.NewManager()
	sharedSecret := make([]byte, 32)
	rand.Read(sharedSecret)

	sess, _ := manager.CreateSession("encrypt-bench", sharedSecret)

	sizes := []int{64, 256, 1024, 4096, 16384}

	for _, size := range sizes {
		message := make([]byte, size)
		rand.Read(message)

		b.Run(fmt.Sprintf("%dB", size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := sess.Encrypt(message)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkSessionDecryption benchmarks message decryption in sessions
func BenchmarkSessionDecryption(b *testing.B) {
	manager := session.NewManager()
	sharedSecret := make([]byte, 32)
	rand.Read(sharedSecret)

	sess, _ := manager.CreateSession("decrypt-bench", sharedSecret)

	sizes := []int{64, 256, 1024, 4096, 16384}

	for _, size := range sizes {
		message := make([]byte, size)
		rand.Read(message)
		encrypted, _ := sess.Encrypt(message)

		b.Run(fmt.Sprintf("%dB", size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := sess.Decrypt(encrypted)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Note: Advanced session benchmarks removed due to API changes
// - BenchmarkHandshakeProtocol: Handshake API now requires gRPC connections
// - BenchmarkSessionManager: Manager API changed significantly
// - BenchmarkConcurrentSessions: Old manager API no longer available
// - BenchmarkNonceValidation: Session interface changed
//
// These benchmarks will be re-added once the new API is stable
