package benchmark

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/handshake"
	"github.com/sage-x-project/sage/session"
)

// BenchmarkSessionCreation benchmarks session creation
func BenchmarkSessionCreation(b *testing.B) {
	clientKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)
	serverKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := session.CreateSession(
			clientKey.PublicKey(),
			serverKey.PublicKey(),
			clientKey,
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSessionEncryption benchmarks message encryption in sessions
func BenchmarkSessionEncryption(b *testing.B) {
	clientKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)
	serverKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)

	sess, _ := session.CreateSession(
		clientKey.PublicKey(),
		serverKey.PublicKey(),
		clientKey,
	)

	sizes := []int{64, 256, 1024, 4096, 16384}

	for _, size := range sizes {
		message := make([]byte, size)
		rand.Read(message)

		b.Run(formatBytes(size), func(b *testing.B) {
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
	clientKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)
	serverKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)

	sess, _ := session.CreateSession(
		clientKey.PublicKey(),
		serverKey.PublicKey(),
		clientKey,
	)

	sizes := []int{64, 256, 1024, 4096, 16384}

	for _, size := range sizes {
		message := make([]byte, size)
		rand.Read(message)
		encrypted, _ := sess.Encrypt(message)

		b.Run(formatBytes(size), func(b *testing.B) {
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

// BenchmarkHandshakeProtocol benchmarks full handshake process
func BenchmarkHandshakeProtocol(b *testing.B) {
	b.Run("Complete", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Client setup
			clientDID, _ := crypto.GenerateKeyPair(crypto.KeyTypeEd25519)
			clientEphemeral, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)

			// Server setup
			serverDID, _ := crypto.GenerateKeyPair(crypto.KeyTypeEd25519)
			serverEphemeral, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)

			// Client initiates handshake
			clientHandshake := handshake.NewClient(clientDID, clientEphemeral)
			initMsg, err := clientHandshake.InitiateHandshake(serverDID.PublicKey())
			if err != nil {
				b.Fatal(err)
			}

			// Server processes and responds
			serverHandshake := handshake.NewServer(serverDID, serverEphemeral)
			responseMsg, err := serverHandshake.ProcessInitiation(initMsg)
			if err != nil {
				b.Fatal(err)
			}

			// Client finalizes
			_, err = clientHandshake.FinalizeHandshake(responseMsg)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkSessionManager benchmarks session manager operations
func BenchmarkSessionManager(b *testing.B) {
	manager := session.NewManager(session.ManagerConfig{
		MaxSessions:       1000,
		SessionMaxAge:     1 * time.Hour,
		SessionIdleTimeout: 10 * time.Minute,
		CleanupInterval:   30 * time.Second,
	})

	b.Run("CreateSession", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			clientKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)
			serverKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)

			_, err := manager.Create(
				clientKey.PublicKey(),
				serverKey.PublicKey(),
				clientKey,
			)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("GetSession", func(b *testing.B) {
		clientKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)
		serverKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)
		sess, _ := manager.Create(clientKey.PublicKey(), serverKey.PublicKey(), clientKey)
		sessionID := sess.ID()

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := manager.Get(sessionID)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("DeleteSession", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			clientKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)
			serverKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)
			sess, _ := manager.Create(clientKey.PublicKey(), serverKey.PublicKey(), clientKey)

			b.StartTimer()
			err := manager.Delete(sess.ID())
			b.StopTimer()

			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkConcurrentSessions benchmarks concurrent session operations
func BenchmarkConcurrentSessions(b *testing.B) {
	manager := session.NewManager(session.ManagerConfig{
		MaxSessions:       10000,
		SessionMaxAge:     1 * time.Hour,
		SessionIdleTimeout: 10 * time.Minute,
		CleanupInterval:   30 * time.Second,
	})

	b.Run("Parallel_Create", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				clientKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)
				serverKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)

				_, err := manager.Create(
					clientKey.PublicKey(),
					serverKey.PublicKey(),
					clientKey,
				)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})

	// Create sessions for read test
	sessions := make([]string, 100)
	for i := 0; i < 100; i++ {
		clientKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)
		serverKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)
		sess, _ := manager.Create(clientKey.PublicKey(), serverKey.PublicKey(), clientKey)
		sessions[i] = sess.ID()
	}

	b.Run("Parallel_Get", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				sessionID := sessions[i%len(sessions)]
				_, err := manager.Get(sessionID)
				if err != nil {
					b.Fatal(err)
				}
				i++
			}
		})
	})
}

// BenchmarkNonceValidation benchmarks nonce validation
func BenchmarkNonceValidation(b *testing.B) {
	manager := session.NewManager(session.ManagerConfig{
		MaxSessions:       1000,
		SessionMaxAge:     1 * time.Hour,
		SessionIdleTimeout: 10 * time.Minute,
		CleanupInterval:   30 * time.Second,
	})

	clientKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)
	serverKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)
	sess, _ := manager.Create(clientKey.PublicKey(), serverKey.PublicKey(), clientKey)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		nonce := make([]byte, 32)
		rand.Read(nonce)

		err := sess.ValidateNonce(nonce, time.Now())
		if err != nil {
			b.Fatal(err)
		}
	}
}
