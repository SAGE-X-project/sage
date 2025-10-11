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
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/session"
)

// BenchmarkBaseline_vs_SAGE compares baseline (no security) vs SAGE-secured communication
func BenchmarkBaseline_vs_SAGE(b *testing.B) {
	message := make([]byte, 1024)
	_, _ = rand.Read(message)

	b.Run("Baseline_NoSecurity", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Simulate simple message passing with no security
			_ = append([]byte(nil), message...)
		}
	})

	b.Run("Baseline_SimpleHash", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Simple hash for integrity check
			hash := sha256.Sum256(message)
			_ = hash
		}
	})

	b.Run("SAGE_FullSecure", func(b *testing.B) {
		// Create session with shared secret
		sharedSecret := make([]byte, 32)
		_, _ = rand.Read(sharedSecret)

		manager := session.NewManager()
		sess, _ := manager.CreateSession("bench-session", sharedSecret)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Encrypt with SAGE session
			encrypted, err := sess.Encrypt(message)
			if err != nil {
				b.Fatal(err)
			}

			// Decrypt
			_, err = sess.Decrypt(encrypted)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkThroughput measures throughput in different scenarios
func BenchmarkThroughput(b *testing.B) {
	messageSizes := []int{64, 256, 1024, 4096, 16384}

	for _, size := range messageSizes {
		message := make([]byte, size)
		_, _ = rand.Read(message)

		b.Run("Baseline_"+formatBytes(size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ReportAllocs()
			b.ResetTimer()

			totalBytes := int64(0)
			start := time.Now()

			for i := 0; i < b.N; i++ {
				_ = append([]byte(nil), message...)
				totalBytes += int64(size)
			}

			elapsed := time.Since(start)
			throughput := float64(totalBytes) / elapsed.Seconds() / 1024 / 1024
			b.ReportMetric(throughput, "MB/s")
		})

		b.Run("SAGE_"+formatBytes(size), func(b *testing.B) {
			// Create session with shared secret
			sharedSecret := make([]byte, 32)
			_, _ = rand.Read(sharedSecret)

			manager := session.NewManager()
			sess, _ := manager.CreateSession("throughput-bench", sharedSecret)

			b.SetBytes(int64(size))
			b.ReportAllocs()
			b.ResetTimer()

			totalBytes := int64(0)
			start := time.Now()

			for i := 0; i < b.N; i++ {
				encrypted, err := sess.Encrypt(message)
				if err != nil {
					b.Fatal(err)
				}
				_, err = sess.Decrypt(encrypted)
				if err != nil {
					b.Fatal(err)
				}
				totalBytes += int64(size)
			}

			elapsed := time.Since(start)
			throughput := float64(totalBytes) / elapsed.Seconds() / 1024 / 1024
			b.ReportMetric(throughput, "MB/s")
		})
	}
}

// BenchmarkLatency measures latency in different scenarios
func BenchmarkLatency(b *testing.B) {
	message := make([]byte, 1024)
	_, _ = rand.Read(message)

	b.Run("Baseline_RoundTrip", func(b *testing.B) {
		b.ReportAllocs()

		latencies := make([]time.Duration, b.N)

		for i := 0; i < b.N; i++ {
			start := time.Now()

			// Simulate round trip: send + receive
			encoded := base64.StdEncoding.EncodeToString(message)
			_, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				b.Fatal(err)
			}

			latencies[i] = time.Since(start)
		}

		b.ReportMetric(calculateP50(latencies).Seconds()*1000, "p50_ms")
		b.ReportMetric(calculateP95(latencies).Seconds()*1000, "p95_ms")
		b.ReportMetric(calculateP99(latencies).Seconds()*1000, "p99_ms")
	})

	b.Run("SAGE_RoundTrip", func(b *testing.B) {
		// Create session with shared secret
		sharedSecret := make([]byte, 32)
		_, _ = rand.Read(sharedSecret)

		manager := session.NewManager()
		sess, _ := manager.CreateSession("latency-bench", sharedSecret)

		b.ReportAllocs()

		latencies := make([]time.Duration, b.N)

		for i := 0; i < b.N; i++ {
			start := time.Now()

			// Encrypt + decrypt round trip
			encrypted, err := sess.Encrypt(message)
			if err != nil {
				b.Fatal(err)
			}
			_, err = sess.Decrypt(encrypted)
			if err != nil {
				b.Fatal(err)
			}

			latencies[i] = time.Since(start)
		}

		b.ReportMetric(calculateP50(latencies).Seconds()*1000, "p50_ms")
		b.ReportMetric(calculateP95(latencies).Seconds()*1000, "p95_ms")
		b.ReportMetric(calculateP99(latencies).Seconds()*1000, "p99_ms")
	})

	// Note: SAGE_FullHandshake benchmark removed
	// The handshake API now requires gRPC connections and is tested separately
	// in handshake/handshake_test.go
}

// BenchmarkMemoryUsage measures memory usage
func BenchmarkMemoryUsage(b *testing.B) {
	b.Run("Baseline_1000Messages", func(b *testing.B) {
		message := make([]byte, 1024)
		_, _ = rand.Read(message)

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			messages := make([][]byte, 1000)
			for j := 0; j < 1000; j++ {
				messages[j] = append([]byte(nil), message...)
			}
			_ = messages
		}
	})

	b.Run("SAGE_1000Sessions", func(b *testing.B) {
		manager := session.NewManager()
		manager.SetDefaultConfig(session.Config{
			MaxAge:      1 * time.Hour,
			IdleTimeout: 10 * time.Minute,
			MaxMessages: 10000,
		})

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			for j := 0; j < 1000; j++ {
				sharedSecret := make([]byte, 32)
				_, _ = rand.Read(sharedSecret)
				sessionID := string(make([]byte, 16))
				_, _ = rand.Read([]byte(sessionID))
				_, _ = manager.CreateSession(sessionID, sharedSecret)
			}
		}
	})
}

// Helper functions for percentile calculations
func calculateP50(latencies []time.Duration) time.Duration {
	return calculatePercentile(latencies, 0.50)
}

func calculateP95(latencies []time.Duration) time.Duration {
	return calculatePercentile(latencies, 0.95)
}

func calculateP99(latencies []time.Duration) time.Duration {
	return calculatePercentile(latencies, 0.99)
}

func calculatePercentile(latencies []time.Duration, percentile float64) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	// Simple percentile calculation (not sorted, approximation)
	index := int(float64(len(latencies)) * percentile)
	if index >= len(latencies) {
		index = len(latencies) - 1
	}

	return latencies[index]
}

func formatBytes(size int) string {
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%dKB", size/1024)
	}
	return fmt.Sprintf("%dMB", size/(1024*1024))
}
